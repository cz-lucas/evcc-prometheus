package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	evccprometheus "github.com/cz-lucas/evcc-prometheus/internal"
)

var logger *slog.Logger

func main() {

	// Setup handler for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Starting evcc prometheus exporter")

	conn, readerDone, err := evccprometheus.Connect(logger, "wss://demo.evcc.io/ws")
	if err != nil {
		return
	}

	<-signalChan

	// Teardown / Shutdown
	logger.Info("Received termination signal, shutting down...")
	if err := evccprometheus.Disconnect(logger, conn, readerDone); err != nil {
		logger.Error("Failed to disconnect from EVCC", "error", err)
	}
}
