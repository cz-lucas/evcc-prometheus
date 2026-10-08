package evccprometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusHandlerServesMetricsEndpoint(t *testing.T) {
	store := NewStateStore()

	registry := prometheus.NewRegistry()
	registry.MustRegister(NewMetricsCollector(store))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	StartHttpServer(registry).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestHealthEndpoint(t *testing.T) {
	store := NewStateStore()

	registry := prometheus.NewRegistry()
	registry.MustRegister(NewMetricsCollector(store))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	StartHttpServer(registry).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if recorder.Body.String() != "OK" {
		t.Fatalf("GET /health text = %q, want %q", recorder.Body.String(), "OK")
	}
}

func TestRootEndpoint(t *testing.T) {
	store := NewStateStore()

	registry := prometheus.NewRegistry()
	registry.MustRegister(NewMetricsCollector(store))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	StartHttpServer(registry).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if strings.Contains(recorder.Body.String(), "EVCC Prometheus Exporter") == false {
		t.Fatalf("GET / text = %q, want %q", recorder.Body.String(), "OK")
	}
}

func TestReadyEndpoint(t *testing.T) {
	tests := []struct {
		name               string
		appReady           bool
		expectedValue      string
		expectedStatusCode int
	}{
		{name: "Not ready", appReady: false, expectedValue: `Not ready`, expectedStatusCode: http.StatusServiceUnavailable},
		{name: "Ready", appReady: true, expectedValue: `Ready`, expectedStatusCode: http.StatusOK},
	}

	previousReady := appReady.Load()
	t.Cleanup(func() { appReady.Store(previousReady) })

	store := NewStateStore()
	registry := prometheus.NewRegistry()
	registry.MustRegister(NewMetricsCollector(store))
	handler := StartHttpServer(registry)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			appReady.Store(test.appReady)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/ready", nil)
			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.expectedStatusCode {
				t.Fatalf("GET /ready status = %d, want %d", recorder.Code, test.expectedStatusCode)
			}

			if recorder.Body.String() != test.expectedValue {
				t.Fatalf("GET /ready text = %q, want %q", recorder.Body.String(), test.expectedValue)
			}
		})
	}
}
