package biteship

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.biteship.com"

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(
	baseURL string,
	apiKey string,
	timeout time.Duration,
) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) SearchAreas(
	ctx context.Context,
	query string,
) ([]areaAPIResult, error) {
	endpoint := c.baseURL + "/v1/maps/areas"

	params := url.Values{}
	params.Set("countries", "ID")
	params.Set("input", query)
	params.Set("type", "single")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint+"?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create Biteship request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send Biteship request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"Biteship returned HTTP status %d",
			resp.StatusCode,
		)
	}

	var result searchAreasAPIResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Biteship response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Biteship area search was unsuccessful")
	}

	return result.Areas, nil
}
