package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const availableProductsEndpoint = "/api/v0/available_products"

type UploadProductInput struct {
	ContentLength   int64
	Product         io.Reader
	ContentType     string
	PollingInterval int
}

type ProductInfo struct {
	Name    string `json:"name"`
	Version string `json:"product_version"`
}

type UploadProductOutput struct{}

type AvailableProductsOutput struct {
	ProductsList []ProductInfo
}

type AvailableProductsInput struct {
	ProductName    string
	ProductVersion string
}

type AvailableProductsService struct {
	client         httpClient
	progressClient httpClient
}

func NewAvailableProductsService(client httpClient, progressClient httpClient) AvailableProductsService {
	return AvailableProductsService{
		client:         client,
		progressClient: progressClient,
	}
}

func (ap AvailableProductsService) Upload(input UploadProductInput) (UploadProductOutput, error) {
	req, err := http.NewRequest("POST", availableProductsEndpoint, input.Product)
	if err != nil {
		return UploadProductOutput{}, err
	}

	req.Header.Set("Content-Type", input.ContentType)
	req.ContentLength = input.ContentLength

	req = req.WithContext(context.WithValue(req.Context(), "polling-interval", time.Duration(input.PollingInterval)*time.Second))

	resp, err := ap.progressClient.Do(req)
	if err != nil {
		return UploadProductOutput{}, fmt.Errorf("could not make api request to available_products endpoint: %s", err)
	}

	defer resp.Body.Close()

	if err = ValidateStatusOK(resp); err != nil {
		return UploadProductOutput{}, err
	}

	return UploadProductOutput{}, nil
}

func (ap AvailableProductsService) List() (AvailableProductsOutput, error) {
	avReq, err := http.NewRequest("GET", availableProductsEndpoint, nil)
	if err != nil {
		return AvailableProductsOutput{}, err
	}

	resp, err := ap.client.Do(avReq)
	if err != nil {
		return AvailableProductsOutput{}, fmt.Errorf("could not make api request to available_products endpoint: %s", err)
	}
	defer resp.Body.Close()

	if err = ValidateStatusOK(resp); err != nil {
		return AvailableProductsOutput{}, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return AvailableProductsOutput{}, err
	}

	var availableProducts []ProductInfo
	err = json.Unmarshal(respBody, &availableProducts)
	if err != nil {
		return AvailableProductsOutput{}, fmt.Errorf("could not unmarshal available_products response: %s", err)
	}

	return AvailableProductsOutput{ProductsList: availableProducts}, nil
}

func (ap AvailableProductsService) CheckProductAvailability(productName string, productVersion string) (bool, error) {
	availableProducts, err := ap.List()
	if err != nil {
		return false, err
	}

	for _, product := range availableProducts.ProductsList {
		if product.Name == productName && product.Version == productVersion {
			return true, nil
		}
	}

	return false, nil
}

func (ap AvailableProductsService) Delete(input AvailableProductsInput, all bool) error {
	req, err := http.NewRequest("DELETE", availableProductsEndpoint, nil)

	if !all {
		query := url.Values{}
		query.Add("product_name", input.ProductName)
		query.Add("version", input.ProductVersion)

		req.URL.RawQuery = query.Encode()
	}

	resp, err := ap.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not make api request to available_products endpoint: %s", err)
	}

	defer resp.Body.Close()

	if err = ValidateStatusOK(resp); err != nil {
		return err
	}

	return nil
}
