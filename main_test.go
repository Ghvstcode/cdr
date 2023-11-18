package main

import (
	"context"
	pb "github.com/Ghvstcode/cdr/pkg/pubsub"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof"
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
		pubsub := pb.NewPubsub()
		handleWebSocketUpgrade(t, w, r, pubsub)
	}))
	defer server.Close()

	ws := createTestWebSocketConnection(t, server)
	defer ws.Close()
}

// handleWebSocketUpgrade upgrades an HTTP request to a WebSocket connection.
func handleWebSocketUpgrade(t *testing.T, w http.ResponseWriter, r *http.Request, pubsub *pb.Pubsub) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.Fatalf("Failed to upgrade to WebSocket: %v", err)
	}
	defer ws.Close()
	//pubsub := pb.NewPubsub()
	handleWebSocketConnection(ws, pubsub)
}

// TestMessagePublishSubscribe checks the publish-subscribe functionality.
func TestMessagePublishSubscribe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketUpgrade(t, w, r, pb.NewPubsub())
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
		handleWebSocketUpgrade(t, w, r, pb.NewPubsub())
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
		handleWebSocketUpgrade(t, w, r, pb.NewPubsub())
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

// TestMessageDeliveryToMultipleSubscribers verifies message delivery to multiple subscribers.
func TestMessageDeliveryToMultipleSubscribers(t *testing.T) {
	ps := pb.NewPubsub()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketUpgrade(t, w, r, ps)
	}))
	defer server.Close()

	ws1, ws2 := createTestWebSocketConnection(t, server), createTestWebSocketConnection(t, server)

	testMessageDelivery(t, ws1, ws2, "testTopic", "testMessage", ps)
}

func testMessageDelivery(t *testing.T, ws1, ws2 *websocket.Conn, topic, message string, pubsubService *pb.Pubsub) {
	msgs1, msgs2 := make(chan string), make(chan string)
	go readMessages(ws1, msgs1)
	go readMessages(ws2, msgs2)
	subscribeAndPublish(t, ws1, ws2, topic, message, pubsubService)
	waitForExpectedMessage(t, msgs2, message)
}

func subscribeAndPublish(t *testing.T, ws1, ws2 *websocket.Conn, topic, message string, pubsubService *pb.Pubsub) {
	// Subscribe both connections to a topic
	subscribe(t, ws1, topic)
	subscribe(t, ws2, topic)

	time.Sleep(5 * time.Second)

	publish(t, ws1, topic, message)
}

// subscribe subscribes a WebSocket connection to a topic.
func subscribe(t *testing.T, ws *websocket.Conn, topic string) {
	msg := Message{Topic: topic, Content: "establishSubscription"}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Failed to write subscription JSON: %v", err)
	}
}

// publish sends a message on a topic from a WebSocket connection.
func publish(t *testing.T, ws *websocket.Conn, topic, content string) {
	msg := Message{Topic: topic, Content: content}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}
}

// waitForExpectedMessage waits for a specific message on a channel.
func waitForExpectedMessage(t *testing.T, msgs chan string, expectedMessage string) {
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	for {
		select {
		case message := <-msgs:
			if message == expectedMessage {
				return
			}
		case <-timer.C:
			t.Fatalf("Timeout waiting for expected message: '%s'", expectedMessage)
		}
	}
}

// readMessages continuously reads messages from a WebSocket connection and sends them to a channel.
// This function is used for testing message delivery in publish-subscribe scenarios.
func readMessages(ws *websocket.Conn, messages chan<- string) {
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			// Log and exit the loop if there's an error reading messages.
			// This could happen if the connection is closed.
			log.Printf("Error reading message: %v", err)
			break
		}
		// Send the received message to the channel.
		messages <- string(msg)
	}
	// Close the channel to signal that no more messages will be sent.
	close(messages)
}
