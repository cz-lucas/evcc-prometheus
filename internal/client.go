package evccprometheus

import (
	"log/slog"

	"github.com/gorilla/websocket"
)

func Connect(logger *slog.Logger, websocketUrl string) {
	logger.Info("Connecting to websocket")

	// Dial opens a WebSocket connection.
	// The first return value is the connection.
	// The second return value is the HTTP response.
	// The third return value is an error, if something went wrong.
	conn, _, err := websocket.DefaultDialer.Dial(websocketUrl, nil)
	if err != nil {
		logger.Error("Could not connect:", "message", err)
		return
	}

	logger.Info("Connected")

	// Keep reading messages forever.
	for {
		// ReadMessage waits until the server sends us a message.
		//
		// messageType tells us what kind of WebSocket message it is
		// (text, binary, etc.).
		// message contains the actual data.
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			logger.Warn("Read error:", "message", err)
			return
		}

		// Log everything we receive.
		logger.Info("Received message",
			"type", messageType,
			"payload", message,
		)
	}

}
