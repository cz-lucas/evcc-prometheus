package evccprometheus

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var appReady = false

func StartPrometheusServer(logger *slog.Logger, addr string, reg *prometheus.Registry, includeCollectors bool) (*http.Server, <-chan error) {
	if includeCollectors {
		reg.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)
	}

	server := &http.Server{Addr: addr, Handler: StartHttpServer(reg)}
	serverDone := make(chan error, 1)
	logger.Info("Starting Prometheus server", "addr", addr)
	go func() {
		serverDone <- server.ListenAndServe()
	}()
	return server, serverDone
}

func HandlerHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func HandlerAppReady(w http.ResponseWriter, r *http.Request) {
	if appReady {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Not ready"))
	}
}

func StartHttpServer(reg *prometheus.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", HandlerHealthCheck)
	mux.HandleFunc("/ready", HandlerAppReady)
	return mux
}
