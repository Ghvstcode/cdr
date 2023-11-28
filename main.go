package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "github.com/Ghvstcode/cdr/pkg/pubsub"
	"github.com/gorilla/websocket"
)

// Constants for server and WebSocket configuration.
const (
	ReadTimeout    = 120 * time.Second
	WriteTimeout   = 120 * time.Second
	IdleTimeout    = 120 * time.Second
	WriteWait      = 10 * time.Second
	PongWait       = 60 * time.Second
	PingPeriod     = (PongWait * 9) / 10 // Send pings to peer with this period. Must be less than PongWait.
	MaxMessageSize = 512                 // Maximum message size allowed from peer.
)

// upgrader is used to upgrade an HTTP connection to a WebSocket connection.
// CheckOrigin function should be updated to allow connections only from trusted origins.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement a secure origin check.
		return true // Currently allowing all origins, not recommended for production.
	},
}

// Message represents a message received over WebSocket.
type Message struct {
	Topic   string `json:"topic"`
	Content string `json:"message"`
}

type SafeWebSocketConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex // Mutex for synchronizing write operations
}

func main() {
	if err := setupServer(); err != nil {
		log.Fatalf("Failed to set up server: %v", err)
	}
}

func NewSafeWebSocketConn(conn *websocket.Conn) *SafeWebSocketConn {
	return &SafeWebSocketConn{conn: conn}
}

func (swsc *SafeWebSocketConn) writeToWebSocket(msgType int, data []byte) error {
	swsc.writeMu.Lock()
	defer swsc.writeMu.Unlock()
	return swsc.conn.WriteMessage(msgType, data)
}

// setupServer initializes and starts the HTTP server with WebSocket support.
func setupServer() error {
	pubsubService := pb.NewPubsub()
	defer pubsubService.Close()

	// Set up channel for handling system signals for graceful shutdown.
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Serve static files from the 'public' directory.
	http.Handle("/", http.FileServer(http.Dir("./public")))

	// Define WebSocket endpoint.
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketRequest(w, r, pubsubService)
	})

	// Configure and start the HTTP server.
	srv := &http.Server{
		Addr:         ":8000",
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	go func() {
		log.Println("HTTP server started on :8000")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-stopChan // Wait for interrupt signal.

	// Initiate graceful shutdown.
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
	return nil
}

// handleWebSocketRequest handles incoming WebSocket requests by upgrading
// the HTTP connection and processing the WebSocket connection.
func handleWebSocketRequest(w http.ResponseWriter, r *http.Request, pubsubService *pb.Pubsub) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		http.Error(w, "Could not open WebSocket connection", http.StatusBadRequest)
		return
	}
	defer ws.Close()

	handleWebSocketConnection(ws, pubsubService)
}

// handleWebSocketConnection manages a WebSocket connection.
func handleWebSocketConnection(ws *websocket.Conn, pubsub *pb.Pubsub) {

	setupConnection(ws)

	safeWs := NewSafeWebSocketConn(ws)
	subscribedTopics := make(map[string]bool)
	wsChan := make(chan string)
	messageChan := make(chan Message)
	var wg sync.WaitGroup

	defer func() {
		// Close the WebSocket connection gracefully.
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		safeWs.writeToWebSocket(websocket.CloseMessage, closeMsg)
		log.Println("Closing WebSocket connection")
	}()

	// Setup goroutines for sending and receiving messages.
	go sendMessages(safeWs, wsChan)
	go messageReader(safeWs, messageChan)

	// Process incoming messages and manage subscriptions.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range messageChan {
			if _, ok := subscribedTopics[msg.Topic]; !ok {
				subscribedTopics[msg.Topic] = true
				pubsub.Subscribe(msg.Topic, wsChan)
			}

			pubsub.Publish(msg.Topic, msg.Content)
		}
	}()

	wg.Wait()
	//close(wsChan) // Close the channel once all processing is done.
}

// setupConnection configures WebSocket connection settings.
func setupConnection(ws *websocket.Conn) {
	ws.SetReadLimit(MaxMessageSize)
	ws.SetReadDeadline(time.Now().Add(PongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})
}

// messageReader reads messages from the WebSocket and sends them to messageChan.
func messageReader(ws *SafeWebSocketConn, messageChan chan<- Message) {
	defer close(messageChan)
	for {
		var msg Message
		if err := ws.conn.ReadJSON(&msg); err != nil {
			log.Printf("ReadJSON error: %v", err)
			break
		}

		messageChan <- msg
	}
}

// sendMessages sends messages to the WebSocket client.
func sendMessages(ws *SafeWebSocketConn, wsChan <-chan string) {
	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-wsChan:
			if !ok {
				ws.writeToWebSocket(websocket.CloseMessage, []byte(""))
				return
			}
			ws.writeToWebSocket(websocket.TextMessage, []byte(msg))
		case <-ticker.C:
			ws.writeToWebSocket(websocket.PingMessage, []byte(""))
		}
	}
}
