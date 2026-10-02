package evccprometheus

import (
	"net/http"
	"net/http/httptest"
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
