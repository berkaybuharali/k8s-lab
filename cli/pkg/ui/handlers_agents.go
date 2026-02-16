package ui

// Phase 5: Agent-specific HTTP API handlers
// TODO: Implement handlers for Magic Cake Shop and Backoffice

// Required endpoints:
//
// POST /api/agent/chat
//   Body: {system: "commerce"|"supply-chain", message: string, session_id?: string}
//   Returns: {response: string, session_id: string}
//   Action: Tunnel to agent pod, send message, return response
//
// GET /api/agent/status
//   Returns: {commerce: {ready: bool, replicas: int}, supply_chain: {...}}
//   Action: kubectl get deployments -n agents
//
// GET /api/inventory
//   Returns: {chocolate: 4, ananas: 1, banana: 3, walnut: 2, almond: 4}
//   Action: Redis HGETALL for all 5 ingredients
//
// GET /api/orders?date=YYYY-MM-DD
//   Returns: [{order_id, customer_name, cakes, address, delivery_date, price, image_urls}]
//   Action: Redis SCAN order:CAKE-*, optionally filter by delivery_date
//
// DELETE /api/orders/:id
//   Returns: {success: bool}
//   Action: Redis DEL + GCS image cleanup
//
// GET /api/orders/:id/image
//   Returns: Redirect to GCS signed URL or proxy image bytes
//   Action: Get signed URL from gcs_images.py tool
//
// GET /api/fulfillment/route?date=YYYY-MM-DD
//   Returns: {waypoints: [{lat, lng, address, order_id}], polyline: string}
//   Action: Call Fulfillment agent via A2A: "Plan route for {date}"
//
// GET /api/orders/stats
//   Returns: {count: int, total_revenue: float, average_order_value: float}
//   Action: Redis order stats aggregation
//
// GET /api/agent/activity
//   Returns: [{timestamp, system, query, action}]
//   Action: Read from agent activity log (Redis stream or in-memory)

// Implementation notes:
// - Reuse existing k8s client, redis client from other handlers
// - Follow pattern from handlers.go (getLogs, getPods, etc.)
// - Add to allowed operations in server.go
// - Error handling with proper HTTP status codes
// - CORS headers already handled by middleware

/*
Example handler structure:

import (
    "encoding/json"
    "net/http"
)

type AgentChatRequest struct {
    System    string `json:"system"`
    Message   string `json:"message"`
    SessionID string `json:"session_id,omitempty"`
}

type AgentChatResponse struct {
    Response  string `json:"response"`
    SessionID string `json:"session_id"`
}

func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
    var req AgentChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // TODO: Validate system (commerce or supply-chain)
    // TODO: Create tunnel to agent pod
    // TODO: Send message via A2A (RemoteA2aAgent or HTTP POST)
    // TODO: Return response

    http.Error(w, "Not implemented (Phase 5)", http.StatusNotImplemented)
}

func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
    // TODO: Connect to Redis
    // TODO: HGETALL for each ingredient
    // TODO: Return JSON

    http.Error(w, "Not implemented (Phase 5)", http.StatusNotImplemented)
}

// Register in server.go:
// mux.HandleFunc("/api/agent/chat", s.handleAgentChat)
// mux.HandleFunc("/api/inventory", s.handleInventory)
// ... etc
*/
