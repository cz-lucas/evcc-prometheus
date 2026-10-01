package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	evccprometheus "github.com/cz-lucas/evcc-prometheus/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var logger *slog.Logger

// main wires EVCC message ingestion to the state store and starts the metrics endpoint.
func main() {

	// Setup handler for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChan)
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("Starting evcc prometheus exporter")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read configuration from environment variables
	sourceID := envOrDefault("EVCC_SOURCE_ID", "default")
	websocketURL := envOrDefault("EVCC_WS_URL", "wss://demo.evcc.io/ws")
	address := envOrDefault("PROMETHEUS_ADDR", ":9070")
	store := evccprometheus.NewStateStore()
	decoder := evccprometheus.NewDecoder()
	registry := prometheus.NewRegistry()
	registry.MustRegister(evccprometheus.NewMetricsCollector(store, sourceID))
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	server := &http.Server{Addr: address, Handler: evccprometheus.PrometheusHandler(registry)}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.ListenAndServe() }()

	messageChan := make(chan string, 16)
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		evccprometheus.Run(ctx, logger, messageChan, websocketURL)
	}()
	logger.Info("Prometheus endpoint ready", "addr", address, "source_id", sourceID)

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

// envOrDefault returns an environment variable's value or its fallback when unset.
func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
