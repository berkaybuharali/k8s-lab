package ui

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local tool
	},
}

// LogMessage represents a message sent to the frontend.
type LogMessage struct {
	Type      string `json:"type"`      // "log", "error", "done"
	Data      string `json:"data"`      // The log line
	Timestamp string `json:"timestamp"` // RFC3339
}

// WebSocketHub manages active WebSocket connections.
type WebSocketHub struct {
	clients   map[*websocket.Conn]bool
	clientsMu sync.Mutex
}

// NewWebSocketHub creates a new hub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[*websocket.Conn]bool),
	}
}

// HandleWebSocket upgrades HTTP requests to WebSocket connections.
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.clientsMu.Lock()
	h.clients[conn] = true
	h.clientsMu.Unlock()

	// Keep connection alive with ping/pong
	go h.keepAlive(conn)
}

// keepAlive handles the connection lifecycle.
func (h *WebSocketHub) keepAlive(conn *websocket.Conn) {
	defer func() {
		h.clientsMu.Lock()
		delete(h.clients, conn)
		h.clientsMu.Unlock()
		conn.Close()
	}()

	for {
		// Read message to handle close frames and ping/pong
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// Broadcast sends a log message to all connected clients.
func (h *WebSocketHub) Broadcast(msgType, data string) {
	msg := LogMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	jsonMsg, _ := json.Marshal(msg)

	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	for conn := range h.clients {
		err := conn.WriteMessage(websocket.TextMessage, jsonMsg)
		if err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
