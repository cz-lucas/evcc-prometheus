package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	evccprometheus "github.com/cz-lucas/evcc-prometheus/internal"
)

var logger *slog.Logger
var evccState evccprometheus.EVCCData

func main() {

	// Setup handler for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	// Setup
	logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	logger.Info("Starting evcc prometheus exporter")

	messageChan := make(chan string, 8)
	conn, readerDone, err := evccprometheus.Connect(logger, messageChan, "wss://demo.evcc.io/ws")
	if err != nil {
		return
	}

	go evccprometheus.MessageToState(logger, messageChan, &evccState)

	<-signalChan

	// Teardown / Shutdown
	logger.Info("Received termination signal, shutting down...")
	if err := evccprometheus.Disconnect(logger, conn, readerDone); err != nil {
		logger.Error("Failed to disconnect from EVCC", "error", err)
	}
}
