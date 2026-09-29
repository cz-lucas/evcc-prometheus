package evccprometheus

import (
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

func Connect(logger *slog.Logger, messageChan chan string, websocketUrl string) (*websocket.Conn, <-chan struct{}, error) {
	logger.Info("Connecting to websocket")

	// Dial opens a WebSocket connection.
	// The first return value is the connection.
	// The second return value is the HTTP response.
	// The third return value is an error, if something went wrong.
	conn, _, err := websocket.DefaultDialer.Dial(websocketUrl, nil)
	if err != nil {
		logger.Error("Could not connect:", "message", err)
		return nil, nil, err
	}

	logger.Info("Connected")

	// Keep reading messages forever.
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		receiveMessages(logger, conn, messageChan)
	}()

	return conn, readerDone, nil
}

func Disconnect(logger *slog.Logger, conn *websocket.Conn, readerDone <-chan struct{}) error {
	logger.Info("Closing connection to EVCC")
	deadline := time.Now().Add(time.Second)
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	if err := conn.WriteControl(websocket.CloseMessage, closeMessage, deadline); err != nil && !errors.Is(err, net.ErrClosed) {
		logger.Warn("Failed to send close message to EVCC", "error", err)
	}

	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case <-readerDone:
	case <-timer.C:
		logger.Warn("Timed out waiting for EVCC close response")
	}

	err := conn.Close()

	if err == nil {
		logger.Info("Connection to EVCC closed")
	} else {
		logger.Warn("Failed to close connection to EVCC", "error", err)
	}

	return err
}

func receiveMessages(logger *slog.Logger, conn *websocket.Conn, messageChan chan string) {
	for {
		// ReadMessage waits until the server sends us a message.
		//
		// messageType tells us what kind of WebSocket message it is
		// (text, binary, etc.).
		// message contains the actual data.
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) && !websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				logger.Warn("Read error:", "message", err)
			}
			return
		}

		if messageType == 1 {
			messageChan <- string(message)
			logger.Debug("Received message from EVCC",
				"payload", message,
			)
		} else {
			logger.Info("Received message of unknown type",
				"type", messageType,
				"payload", message,
			)
		}
	}
}
