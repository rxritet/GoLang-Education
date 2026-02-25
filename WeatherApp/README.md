# weather-cli

Production-ready CLI utility for fetching current weather data from [OpenWeatherMap API](https://openweathermap.org/api).

## Project Structure

```
weather-cli/
├── cmd/
│   └── weather/
│       └── main.go           # Entry point, flag parsing, output formatting
├── internal/
│   └── weather/
│       ├── client.go         # HTTP client with context & timeout
│       ├── client_test.go    # Unit tests (httptest, no network)
│       └── models.go         # JSON response/error structs
├── go.mod
├── Makefile
└── README.md
```

## Getting an API Key

1. Register at [openweathermap.org](https://home.openweathermap.org/users/sign_up).
2. Go to **API Keys** tab in your account.
3. Copy the default key or generate a new one.
4. The free tier allows **60 calls/minute** — more than enough for a CLI tool.

## Usage

### Set API key (one of two ways)

```bash
# Option A: environment variable
export OWM_API_KEY="your_api_key_here"

# Option B: pass directly via flag (overrides env)
./weather -key="your_api_key_here"
```

### Run

```bash
# Default city (Almaty)
go run ./cmd/weather

# Specify city and timeout
go run ./cmd/weather -city="London" -timeout=10s

# With all flags
go run ./cmd/weather -key="abc123" -city="Tokyo" -timeout=3s
```

### Example Output

```
☁️  Weather in Almaty, KZ
─────────────────────────────────
🌡️  Temperature:  -5.2 °C
🤔  Feels like:   -9.8 °C
💧  Humidity:      72%
💨  Wind:          3.5 m/s
📋  Condition:     Clouds (overcast clouds)
```

### Build & Test

```bash
# Build binary
make build

# Run tests (no internet required)
make test

# Run directly
make run
```

## Flags

| Flag       | Default   | Description                        |
|------------|-----------|------------------------------------|
| `-key`     | —         | OpenWeatherMap API key             |
| `-city`    | `Almaty`  | City name                          |
| `-timeout` | `5s`      | HTTP request timeout (Go duration) |

## Design Decisions

- **No `http.DefaultClient`** — a dedicated `http.Client` with explicit timeout prevents hanging requests.
- **`context.Context`** — enables cancellation propagation from the caller (e.g., OS signals).
- **`url.URL` + `Query().Set()`** — safe URL construction, no string concatenation vulnerabilities.
- **`httptest.NewServer`** in tests — fully offline, deterministic unit tests.
- **Standard library only** — zero external dependencies.
