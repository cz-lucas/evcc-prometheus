package evccprometheus

import (
	"log/slog"
	"os"
	"testing"
)

func TestRoundFloatValidPrecision(t *testing.T) {
	tests := []struct {
		val       float64
		precision uint
		want      float64
	}{
		{val: 1.2345, precision: 0, want: 1},
		{val: 1.5678, precision: 1, want: 1.6},
		{val: 0.1234, precision: 2, want: 0.12},
		{val: 0.1267, precision: 2, want: 0.13},
		{val: 1.2345, precision: 3, want: 1.235},
		{val: 1.23, precision: 3, want: 1.23},
	}

	for _, tt := range tests {
		got := roundFloat(tt.val, tt.precision)
		if got != tt.want {
			t.Errorf("roundFloat(%v, %d) = %v, want %v", tt.val, tt.precision, got, tt.want)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{
			name:     "debug",
			input:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "info",
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn",
			input:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "error",
			input:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "unknown level",
			input:    "unknown",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty level",
			input:    "",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseLogLevel(tt.input); got != tt.expected {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEnvOrDefault(t *testing.T) {
	const key = "TEST_ENV_OR_DEFAULT"

	t.Run("returns environment variable when set", func(t *testing.T) {
		t.Setenv(key, "configured-value")

		if got := EnvOrDefault(key, "fallback"); got != "configured-value" {
			t.Errorf("EnvOrDefault(%q, %q) = %q, want %q",
				key, "fallback", got, "configured-value")
		}
	})

	t.Run("returns fallback when environment variable is unset", func(t *testing.T) {
		t.Setenv(key, "")
		os.Unsetenv(key)

		if got := EnvOrDefault(key, "fallback"); got != "fallback" {
			t.Errorf("EnvOrDefault(%q, %q) = %q, want %q",
				key, "fallback", got, "fallback")
		}
	})

	t.Run("returns fallback when environment variable is empty", func(t *testing.T) {
		t.Setenv(key, "")

		if got := EnvOrDefault(key, "fallback"); got != "fallback" {
			t.Errorf("EnvOrDefault(%q, %q) = %q, want %q",
				key, "fallback", got, "fallback")
		}
	})
}

func TestUrlValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid ws URL",
			input:    "ws://example.com/ws",
			expected: true,
		},
		{
			name:     "valid wss URL",
			input:    "wss://example.com/ws",
			expected: true,
		},
		{
			name:     "invalid URL missing ws",
			input:    "ws://example.com",
			expected: false,
		},
		{
			name:     "invalid URL wrong scheme",
			input:    "http://example.com/ws",
			expected: false,
		},
		{
			name:     "invalid URL empty",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid URL missing scheme",
			input:    "example.com/ws",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UrlValidator(tt.input); got != tt.expected {
				t.Errorf("UrlValidator(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
