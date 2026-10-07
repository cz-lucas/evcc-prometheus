package evccprometheus

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

const (
	initialRetryDelay      = time.Second
	maximumRetryDelay      = 30 * time.Second
	stableConnectionPeriod = time.Minute
)

// Run connects to EVCC and reconnects until ctx is canceled, then closes messageChan.
func Run(ctx context.Context, logger *slog.Logger, messageChan chan<- string, websocketURL string) {
	runWithBackoff(ctx, logger, messageChan, websocketURL, initialRetryDelay, maximumRetryDelay)
}

// runWithBackoff maintains the WebSocket connection and increases retry delays after failures.
func runWithBackoff(ctx context.Context, logger *slog.Logger, messageChan chan<- string, websocketURL string, initialDelay, maximumDelay time.Duration) {
	defer close(messageChan)

	retryDelay := initialDelay
	for ctx.Err() == nil {
		logger.Info("Connecting to websocket")
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, websocketURL, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			appReady = false
			logger.Warn("Could not connect to EVCC", "error", err, "retry_in", retryDelay)
		} else {
			appReady = true
			logger.Info("Connected")
			connectedAt := time.Now()
			connectionDone := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					_ = conn.Close()
				case <-connectionDone:
				}
			}()

			err = receiveMessages(ctx, logger, conn, messageChan)
			close(connectionDone)
			appReady = false
			_ = conn.Close()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				logger.Warn("WebSocket connection to EVCC lost", "error", err, "retry_in", retryDelay)
				appReady = false
			}
			if time.Since(connectedAt) >= stableConnectionPeriod {
				retryDelay = initialDelay
			}
		}

		if !waitForRetry(ctx, retryDelay) {
			return
		}
		retryDelay = nextRetryDelay(retryDelay, maximumDelay)
	}
}

// waitForRetry waits for a randomized retry delay or returns early when the context is canceled.
func waitForRetry(ctx context.Context, delay time.Duration) bool {
	actualDelay := delay / 2
	actualDelay += time.Duration(rand.Float64() * float64(delay-actualDelay))
	timer := time.NewTimer(actualDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// nextRetryDelay doubles the retry delay without exceeding the configured maximum.
func nextRetryDelay(delay, maximum time.Duration) time.Duration {
	if delay >= maximum/2 {
		return maximum
	}
	return delay * 2
}

// receiveMessages forwards text frames until the connection closes or the context is canceled.
func receiveMessages(ctx context.Context, logger *slog.Logger, conn *websocket.Conn, messageChan chan<- string) error {
	for {
		// ReadMessage waits until the server sends us a message.
		//
		// messageType tells us what kind of WebSocket message it is
		// (text, binary, etc.).
		// message contains the actual data.
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				return nil
			}
			appReady = false
			return err
		}

		if messageType == websocket.TextMessage {
			select {
			case messageChan <- string(message):
			case <-ctx.Done():
				return ctx.Err()
			}
			logger.Debug("Received message from EVCC",
				"payload", message,
			)
		} else {
			logger.Info("Received message of unknown type",
				"type", messageType,
				"payload", message,
			)
		}
		appReady = true
	}
}
