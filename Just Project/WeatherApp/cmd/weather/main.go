package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/weather-cli/internal/weather"
)

func main() {
	var (
		apiKey  = flag.String("key", "", "OpenWeatherMap API key (overrides OWM_API_KEY env)")
		city    = flag.String("city", "Almaty", "City name to check weather for")
		timeout = flag.Duration("timeout", 5*time.Second, "HTTP request timeout")
	)
	flag.Parse()

	key := resolveAPIKey(*apiKey)
	if key == "" {
		fmt.Fprintln(os.Stderr, "error: API key is required. Use -key flag or set OWM_API_KEY environment variable.")
		os.Exit(1)
	}

	client := weather.NewClient(key, *timeout)

	// Context with timeout gives us a hard deadline independent of the HTTP client timeout.
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	w, err := client.FetchWeather(ctx, *city)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	printWeather(w)
}

// resolveAPIKey returns the API key following the priority chain:
// flag > environment variable > empty string.
func resolveAPIKey(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("OWM_API_KEY")
}

func weatherEmoji(condition string) string {
	switch condition {
	case "Clear":
		return "☀️"
	case "Clouds":
		return "☁️"
	case "Rain", "Drizzle":
		return "🌧️"
	case "Thunderstorm":
		return "⛈️"
	case "Snow":
		return "❄️"
	case "Mist", "Fog", "Haze":
		return "🌫️"
	default:
		return "🌡️"
	}
}

func printWeather(w *weather.WeatherResponse) {
	condition := ""
	description := ""
	if len(w.Weather) > 0 {
		condition = w.Weather[0].Main
		description = w.Weather[0].Description
	}

	emoji := weatherEmoji(condition)

	fmt.Printf("\n%s  Weather in %s, %s\n", emoji, w.Name, w.Sys.Country)
	fmt.Println("─────────────────────────────────")

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "🌡️  Temperature:\t%.1f °C\n", w.Main.Temp)
	fmt.Fprintf(tw, "🤔  Feels like:\t%.1f °C\n", w.Main.FeelsLike)
	fmt.Fprintf(tw, "💧  Humidity:\t%d%%\n", w.Main.Humidity)
	fmt.Fprintf(tw, "💨  Wind:\t%.1f m/s\n", w.Wind.Speed)
	fmt.Fprintf(tw, "📋  Condition:\t%s (%s)\n", condition, description)
	tw.Flush()

	fmt.Println()
}
