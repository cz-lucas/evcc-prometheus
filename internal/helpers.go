package evccprometheus

import (
	"log/slog"
	"math"
	"os"
	"strings"
)

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func ParseLogLevel(logLevel string) slog.Level {
	switch logLevel {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// EnvOrDefault returns an environment variable's value or its fallback when unset.
func EnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

/*
* Validated that the URL starts with ws:// or wss:// and ends with /ws
* I've implemented it without regex because I don't like using more dependencies for such a simple task.
 */
func UrlValidator(url string) bool {

	// Do not accecpt urls that contain spaces
	if strings.Contains(url, " ") {
		return false
	}

	// Validate scheme
	if !(strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://")) {
		return false
	}

	url = strings.TrimPrefix(url, "wss://")
	url = strings.TrimPrefix(url, "ws://")

	// Validate path
	if !(strings.HasSuffix(url, "/ws")) {
		return false
	}

	url = strings.TrimSuffix(url, "/ws")

	return len(url) > 0
}
