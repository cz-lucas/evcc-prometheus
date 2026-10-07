package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	evccprometheus "github.com/cz-lucas/evcc-prometheus/internal"
	"github.com/prometheus/client_golang/prometheus"
)

var logger *slog.Logger

// main wires EVCC message ingestion to the state store and starts the metrics endpoint.
func main() {

	// Read configuration from environment variables
	websocketURL := evccprometheus.EnvOrDefault("EVCC_WS_URL", "wss://demo.evcc.io/ws")
	address := evccprometheus.EnvOrDefault("PROMETHEUS_ADDR", ":9070")
	logLevel := evccprometheus.EnvOrDefault("LOG_LEVEL", "info")
	goMetricsEnabled := evccprometheus.EnvOrDefault("GO_METRICS_ENABLED", "true")

	// Setup handler for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChan)
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: evccprometheus.ParseLogLevel(logLevel)}))
	logger.Info("Starting evcc prometheus exporter")

	// Validate WebSocket URL
	if !evccprometheus.UrlValidator(websocketURL) {
		logger.Error("Invalid WebSocket URL", "url", websocketURL)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := evccprometheus.NewStateStore()
	decoder := evccprometheus.NewDecoder()
	registry := prometheus.NewRegistry()
	registry.MustRegister(evccprometheus.NewMetricsCollector(store))

	includeGoMetrics, err := strconv.ParseBool(goMetricsEnabled)
	if err != nil {
		logger.Warn("Invalid GO_METRICS_ENABLED value; using true", "value", goMetricsEnabled)
		includeGoMetrics = true
	}
	server, serverDone := evccprometheus.StartPrometheusServer(logger, address, registry, includeGoMetrics)

	messageChan := make(chan string, 16)
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		evccprometheus.Run(ctx, logger, messageChan, websocketURL)
	}()
	logger.Info("Prometheus endpoint starting", "addr", address)

	// This loop is the sole state writer; the collector reads snapshots during scrapes.
	for {
		select {
		case <-signalChan:
			logger.Info("Received termination signal, shutting down...")
			goto shutdown
		case err := <-serverDone:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("Prometheus server stopped", "error", err)
			}
			goto shutdown
		case message, ok := <-messageChan:
			if !ok {
				messageChan = nil
				continue
			}
			update, err := decoder.Decode(message)
			if err != nil {
				logger.Warn("Failed to decode message from EVCC", "error", err)
				continue
			}
			store.Apply(update)
		}
	}

shutdown:
	cancel()
	<-clientDone
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Warn("Failed to shut down Prometheus server cleanly", "error", err)
	}
}
