package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"fx-gateway/internal/domain"
	"net/http"
	"time"
)


type APIClient struct{
	baseURL    string 
	httpClient *http.Client
	volatilityMatrix map[string]int64
}

func NewAPIClient(baseURL string) *APIClient{
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		volatilityMatrix: map[string]int64{
			"EUR": 600,  // 6.00% 
			"GBP": 750,  // 7.50%
			"JPY": 900,  // 9.00%
			"TRY": 4500, // 45.00% 
		},
	}
}

type exchangeResponse struct{
	ConversionRates map[string]float64 `json:"conversion_rates"`
	Rates           map[string]float64 `json:"rates"`
}

func (c *APIClient) FetchRate(ctx context.Context, from, to string) (domain.Quote, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, from)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.Quote{}, fmt.Errorf("api_client: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.Quote{}, fmt.Errorf("api_client: external network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.Quote{}, fmt.Errorf("api_client: provider returned status %d", resp.StatusCode)
	}

	var data exchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return domain.Quote{}, fmt.Errorf("api_client: failed to decode response: %w", err)
	}

	rates := data.ConversionRates
	if len(rates) == 0 {
		rates = data.Rates
	}

	rawRate, exists := rates[to]
	if !exists {
		return domain.Quote{}, fmt.Errorf("api_client: currency rate for %s not found", to)
	}

	vol, hasVol := c.volatilityMatrix[to]
	if !hasVol {
		vol = 1500 
	}

	return domain.Quote{
		From:       from,
		To:         to,
		Rate:       int64(rawRate * 10000),
		Volatility: vol,
		ValidUntil: time.Now().Add(5 * time.Minute), // Quote TTL
	}, nil
}