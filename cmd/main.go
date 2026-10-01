package main

import (
	"context"
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
	defer signal.Stop(signalChan)

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	// Setup
	logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	logger.Info("Starting evcc prometheus exporter")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messageChan := make(chan string, 16)
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		evccprometheus.Run(ctx, logger, messageChan, "wss://demo.evcc.io/ws")
	}()
	messageProcessorDone := make(chan struct{})
	go func() {
		defer close(messageProcessorDone)
		evccprometheus.MessageToState(logger, messageChan, &evccState)
	}()

	<-signalChan

	// Teardown / Shutdown
	logger.Info("Received termination signal, shutting down...")
	cancel()
	<-clientDone
	<-messageProcessorDone
}
