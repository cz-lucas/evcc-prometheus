package evccprometheus

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartPrometheusServer(logger *slog.Logger, addr string, reg *prometheus.Registry, includeCollectors bool) {
	if includeCollectors {
		reg.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	logger.Info("Starting Prometheus server", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("Failed to start Prometheus server", "err", err)
	}
}
