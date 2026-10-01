package evccprometheus

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRunReconnectsAfterReadFailure(t *testing.T) {
	var connections atomic.Int32
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		if connections.Add(1) == 1 {
			_ = conn.Close()
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("reconnected")); err != nil {
			return
		}
		_, _, _ = conn.ReadMessage()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	messages := make(chan string, 1)
	done := make(chan struct{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	go func() {
		defer close(done)
		runWithBackoff(ctx, logger, messages, "ws"+strings.TrimPrefix(server.URL, "http"), 10*time.Millisecond, 20*time.Millisecond)
	}()

	select {
	case message := <-messages:
		if message != "reconnected" {
			t.Fatalf("message = %q, want %q", message, "reconnected")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message after reconnect")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
	if connections.Load() < 2 {
		t.Fatalf("websocket connections = %d, want at least 2", connections.Load())
	}
}

func TestRunRetriesAfterDialFailure(t *testing.T) {
	var requests atomic.Int32
	var acceptingConnections atomic.Bool
	dialFailed := make(chan struct{}, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if !acceptingConnections.Load() {
			http.Error(writer, "temporarily unavailable", http.StatusServiceUnavailable)
			select {
			case dialFailed <- struct{}{}:
			default:
			}
			return
		}

		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("connected")); err != nil {
			return
		}
		_, _, _ = conn.ReadMessage()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	messages := make(chan string, 1)
	done := make(chan struct{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http")
	go func() {
		defer close(done)
		runWithBackoff(ctx, logger, messages, websocketURL, 10*time.Millisecond, 20*time.Millisecond)
	}()

	select {
	case <-dialFailed:
		acceptingConnections.Store(true)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial dial failure")
	}

	select {
	case message := <-messages:
		if message != "connected" {
			t.Fatalf("message = %q, want %q", message, "connected")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message after dial retry")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
	if requests.Load() < 2 {
		t.Fatalf("websocket requests = %d, want at least 2", requests.Load())
	}
}

func TestNextRetryDelay(t *testing.T) {
	tests := []struct {
		name    string
		delay   time.Duration
		maximum time.Duration
		want    time.Duration
	}{
		{name: "doubles initial delay", delay: time.Second, maximum: 30 * time.Second, want: 2 * time.Second},
		{name: "doubles below cap", delay: 8 * time.Second, maximum: 30 * time.Second, want: 16 * time.Second},
		{name: "caps doubled delay", delay: 16 * time.Second, maximum: 30 * time.Second, want: 30 * time.Second},
		{name: "keeps maximum delay capped", delay: 30 * time.Second, maximum: 30 * time.Second, want: 30 * time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := nextRetryDelay(test.delay, test.maximum); got != test.want {
				t.Errorf("nextRetryDelay(%v, %v) = %v, want %v", test.delay, test.maximum, got, test.want)
			}
		})
	}
}

func TestWaitForRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool, 1)
	go func() {
		done <- waitForRetry(ctx, time.Hour)
	}()

	cancel()
	select {
	case waited := <-done:
		if waited {
			t.Fatal("waitForRetry() = true after context cancellation, want false")
		}
	case <-time.After(time.Second):
		t.Fatal("waitForRetry() did not stop after context cancellation")
	}
}
