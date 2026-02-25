package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testAPIKey = "test-key"

// successResponse returns a realistic OpenWeatherMap JSON payload.
func successResponse() WeatherResponse {
	return WeatherResponse{
		Name: "Almaty",
		Sys: struct {
			Country string `json:"country"`
		}{Country: "KZ"},
		Main: struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Humidity  int     `json:"humidity"`
			TempMin   float64 `json:"temp_min"`
			TempMax   float64 `json:"temp_max"`
		}{
			Temp:      -5.2,
			FeelsLike: -9.8,
			Humidity:  72,
			TempMin:   -7.0,
			TempMax:   -3.0,
		},
		Wind: struct {
			Speed float64 `json:"speed"`
		}{Speed: 3.5},
		Weather: []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		}{
			{Main: "Clouds", Description: "overcast clouds"},
		},
	}
}

func newTestClient(baseURL string) *Client {
	client := NewClient(testAPIKey, 5*time.Second)
	client.baseURL = baseURL
	return client
}

func TestFetchWeatherSuccess(t *testing.T) {
	resp := successResponse()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("q"); got != "Almaty" {
			t.Errorf("expected city=Almaty, got %s", got)
		}
		if got := q.Get("appid"); got != testAPIKey {
			t.Errorf("expected appid=%s, got %s", testAPIKey, got)
		}
		if got := q.Get("units"); got != "metric" {
			t.Errorf("expected units=metric, got %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	got, err := newTestClient(srv.URL).FetchWeather(context.Background(), "Almaty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Name != "Almaty" {
		t.Errorf("expected name Almaty, got %s", got.Name)
	}
	if got.Sys.Country != "KZ" {
		t.Errorf("expected country KZ, got %s", got.Sys.Country)
	}
	if got.Main.Temp != -5.2 {
		t.Errorf("expected temp -5.2, got %f", got.Main.Temp)
	}
	if got.Main.Humidity != 72 {
		t.Errorf("expected humidity 72, got %d", got.Main.Humidity)
	}
	if got.Wind.Speed != 3.5 {
		t.Errorf("expected wind 3.5, got %f", got.Wind.Speed)
	}
	if len(got.Weather) == 0 || got.Weather[0].Main != "Clouds" {
		t.Errorf("expected weather condition Clouds, got %+v", got.Weather)
	}
}

func TestFetchWeatherNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Cod:     "404",
			Message: "city not found",
		})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).FetchWeather(context.Background(), "NonExistentCity")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}

	expected := "API error (HTTP 404): city not found"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestFetchWeatherUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIError{
			Cod:     401,
			Message: "Invalid API key",
		})
	}))
	defer srv.Close()

	client := NewClient("bad-key", 5*time.Second)
	client.baseURL = srv.URL

	_, err := client.FetchWeather(context.Background(), "Almaty")
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}

	expected := "API error (HTTP 401): Invalid API key"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestFetchWeatherServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).FetchWeather(context.Background(), "Almaty")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestFetchWeatherContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := newTestClient(srv.URL).FetchWeather(ctx, "Almaty")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
