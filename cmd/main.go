package main

import (
	"log/slog"
	"os"

	evccprometheus "github.com/cz-lucas/evcc-prometheus/internal"
)

var logger *slog.Logger

func main() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Starting evcc prometheus exporter")

	evccprometheus.Connect(logger, "wss://demo.evcc.io/ws")
}
