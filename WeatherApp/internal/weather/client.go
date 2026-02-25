package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.openweathermap.org/data/2.5/weather"

// Client wraps an HTTP client configured for OpenWeatherMap API.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string // overridable for testing
}

// NewClient creates a Client with an explicit timeout instead of http.DefaultClient.
func NewClient(apiKey string, timeout time.Duration) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// FetchWeather requests current weather for the given city.
// The context allows the caller (e.g. main) to enforce cancellation or deadline.
func (c *Client) FetchWeather(ctx context.Context, city string) (*WeatherResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	q := u.Query()
	q.Set("q", city)
	q.Set("appid", c.apiKey)
	q.Set("units", "metric")
	q.Set("lang", "en")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("API error (HTTP %d): unable to decode body", resp.StatusCode)
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, apiErr.Message)
	}

	var weather WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &weather, nil
}
