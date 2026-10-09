package evccprometheus

import (
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var appReady atomic.Bool

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
	if appReady.Load() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Not ready"))
	}
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("EVCC Prometheus Exporter (https://github.com/cz-lucas/evcc-prometheus) \n" +
		"Routes:\n/metrics\n/health\n/ready\n"))
}

func StartHttpServer(reg *prometheus.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", HandlerHealthCheck)
	mux.HandleFunc("/ready", HandlerAppReady)
	mux.HandleFunc("/", HandleRoot)
	return mux
}
