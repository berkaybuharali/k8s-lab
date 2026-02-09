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
	Type      string `json:"type"`      // "log", "error", "done", "start"
	Data      string `json:"data"`      // The log line
	Timestamp string `json:"timestamp"` // RFC3339
}

const maxHistorySize = 1000

// WebSocketHub manages active WebSocket connections and log history.
type WebSocketHub struct {
	clients   map[*websocket.Conn]bool
	clientsMu sync.Mutex

	history   []LogMessage
	historyMu sync.RWMutex
	running   bool // true between "start" and "done"/"error"
}

// NewWebSocketHub creates a new hub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[*websocket.Conn]bool),
		history: make([]LogMessage, 0, 256),
	}
}

// HandleWebSocket upgrades HTTP requests to WebSocket connections.
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Replay history to the new client before adding to broadcast pool
	h.historyMu.RLock()
	for _, msg := range h.history {
		jsonMsg, _ := json.Marshal(msg)
		if err := conn.WriteMessage(websocket.TextMessage, jsonMsg); err != nil {
			h.historyMu.RUnlock()
			conn.Close()
			return
		}
	}
	h.historyMu.RUnlock()

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

// Broadcast sends a log message to all connected clients and stores in history.
func (h *WebSocketHub) Broadcast(msgType, data string) {
	msg := LogMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Track running state
	if msgType == "start" {
		h.historyMu.Lock()
		h.running = true
		h.historyMu.Unlock()
	} else if msgType == "done" || msgType == "error" {
		h.historyMu.Lock()
		h.running = false
		h.historyMu.Unlock()
	}

	// Append to history (ring buffer)
	h.historyMu.Lock()
	if len(h.history) >= maxHistorySize {
		// Drop oldest 25% to avoid frequent shifts
		drop := maxHistorySize / 4
		h.history = h.history[drop:]
	}
	h.history = append(h.history, msg)
	h.historyMu.Unlock()

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
