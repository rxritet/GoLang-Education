package weather

// WeatherResponse represents the successful JSON response from OpenWeatherMap API.
type WeatherResponse struct {
	Name string `json:"name"`
	Sys  struct {
		Country string `json:"country"`
	} `json:"sys"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  int     `json:"humidity"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
}

// APIError represents an error response from OpenWeatherMap API.
type APIError struct {
	Cod     any    `json:"cod"` // API returns cod as int or string depending on context
	Message string `json:"message"`
}
