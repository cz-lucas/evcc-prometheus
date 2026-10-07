package evccprometheus

import (
	"log/slog"
	"math"
	"os"
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
	if (len(url) >= 5 && (url[:5] == "ws://" || url[:6] == "wss://")) && len(url) >= 3 && url[len(url)-3:] == "/ws" {
		return true
	}
	return false
}
