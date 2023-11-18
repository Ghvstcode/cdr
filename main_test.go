package main

import (
	"context"
	pb "github.com/Ghvstcode/cdr/pkg/pubsub"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"
)

// createTestWebSocketConnection establishes a WebSocket connection for testing.
func createTestWebSocketConnection(t *testing.T, server *httptest.Server) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial WebSocket: %v", err)
	}
	return ws
}

// TestWebSocketConnection verifies basic WebSocket connection functionality.
func TestWebSocketConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketUpgrade(t, w, r)
	}))
	defer server.Close()

	ws := createTestWebSocketConnection(t, server)
	defer ws.Close()
}

// handleWebSocketUpgrade upgrades an HTTP request to a WebSocket connection.
func handleWebSocketUpgrade(t *testing.T, w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.Fatalf("Failed to upgrade to WebSocket: %v", err)
	}
	defer ws.Close()
	pubsub := pb.NewPubsub()
	handleWebSocketConnection(ws, pubsub)
}

// TestMessagePublishSubscribe checks the publish-subscribe functionality.
func TestMessagePublishSubscribe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketUpgrade(t, w, r)
	}))
	defer server.Close()

	ws := createTestWebSocketConnection(t, server)
	defer ws.Close()

	testMessagePublishSubscribe(t, ws)
}

// testMessagePublishSubscribe encapsulates message publish-subscribe logic.
func testMessagePublishSubscribe(t *testing.T, ws *websocket.Conn) {
	msg := Message{
		Topic:   "testTopic",
		Content: "testMessage",
	}

	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}

	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}
	if string(p) != "testMessage" {
		t.Errorf("Expected 'testMessage', got '%s'", string(p))
	}
}

// TestServerShutdown verifies graceful server shutdown.
func TestServerShutdown(t *testing.T) {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM)

	s := &http.Server{Addr: ":8000", Handler: nil}

	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			t.Errorf("ListenAndServe: %v", err)
		}
	}()

	triggerShutdown(t, stopChan, s)
}

// triggerShutdown initiates and verifies server shutdown.
func triggerShutdown(t *testing.T, stopChan chan os.Signal, s *http.Server) {
	go func() {
		stopChan <- syscall.SIGTERM
	}()

	<-stopChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Errorf("Server shutdown failed: %+v", err)
	}
}

// TestMultipleWebSocketConnections ensures handling of multiple connections.
func TestMultipleWebSocketConnections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketUpgrade(t, w, r)
	}))
	defer server.Close()

	createMultipleConnections(t, server, 5)
}

// createMultipleConnections creates and manages multiple WebSocket connections.
func createMultipleConnections(t *testing.T, server *httptest.Server, numConnections int) {
	connections := make([]*websocket.Conn, numConnections)

	for i := range connections {
		connections[i] = createTestWebSocketConnection(t, server)
		defer connections[i].Close()
	}
}

// TestPingPong verifies the ping/pong mechanism.
func TestPingPong(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketUpgrade(t, w, r)
	}))
	defer server.Close()

	ws := createTestWebSocketConnection(t, server)
	defer ws.Close()

	testPingPongMechanism(t, ws)
}

// testPingPongMechanism tests the WebSocket ping/pong mechanism.
func testPingPongMechanism(t *testing.T, ws *websocket.Conn) {
	err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(WriteWait))
	if err != nil {
		t.Errorf("Failed to write ping message: %v", err)
	}
}
