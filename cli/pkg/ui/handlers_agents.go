package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AgentChatRequest represents an incoming chat message
type AgentChatRequest struct {
	System    string `json:"system"`     // "commerce" or "supply-chain"
	Message   string `json:"message"`    // User message
	SessionID string `json:"session_id"` // Optional session ID for continuity
}

// AgentChatResponse represents the agent's reply
type AgentChatResponse struct {
	System    string `json:"system"`
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
	Error     string `json:"error,omitempty"`
}

// handleAgentChat processes chat messages to the agents
func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.System != "commerce" && req.System != "supply-chain" {
		http.Error(w, "Invalid system (must be commerce or supply-chain)", http.StatusBadRequest)
		return
	}

	port := 8001
	if req.System == "supply-chain" {
		port = 8002
	}

	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	// Reach the in-cluster service via the K8s API server proxy.
	// contextId must be inside params.message — see MEMORY.md for A2A protocol details.
	proxyPath := fmt.Sprintf("/api/v1/namespaces/agents/services/%s:%d/proxy/", req.System, port)

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "message/send",
		"id":      "1",
		"params": map[string]interface{}{
			"message": map[string]interface{}{
				"role":      "user",
				"parts":     []map[string]string{{"kind": "text", "text": req.Message}},
				"messageId": uuid.New().String(),
				"contextId": req.SessionID,
			},
		},
	}

	resp, err := s.proxyRequestToAgent(r.Context(), proxyPath, payload)
	if err != nil {
		s.logger.Error("Agent chat failed: %v", err)
		json.NewEncoder(w).Encode(AgentChatResponse{
			System:    req.System,
			Error:     fmt.Sprintf("Failed to communicate with agent: %v", err),
			SessionID: req.SessionID,
		})
		return
	}

	text := extractAgentResponseText(resp)

	json.NewEncoder(w).Encode(AgentChatResponse{
		System:    req.System,
		Response:  text,
		SessionID: req.SessionID,
	})
}

// proxyRequestToAgent sends a JSON request to a K8s service via the API server proxy.
func (s *Server) proxyRequestToAgent(ctx context.Context, path string, payload interface{}) (map[string]interface{}, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "kubectl", "create", "--raw", path, "-f", "-", "--kubeconfig", s.config.GetKubeconfigPath())
	cmd.Stdin = strings.NewReader(string(payloadBytes))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("kubectl failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from agent: %w", err)
	}

	return result, nil
}

// extractAgentResponseText extracts the text reply from an A2A result envelope.
// It tries artifacts first, then falls back to the last message in history.
func extractAgentResponseText(result map[string]interface{}) string {
	res, ok := result["result"].(map[string]interface{})
	if !ok {
		return "No result in response"
	}

	if artifacts, ok := res["artifacts"].([]interface{}); ok {
		for _, art := range artifacts {
			if aMap, ok := art.(map[string]interface{}); ok {
				if parts, ok := aMap["parts"].([]interface{}); ok {
					for _, p := range parts {
						if pMap, ok := p.(map[string]interface{}); ok {
							if txt, ok := pMap["text"].(string); ok && txt != "" {
								return txt
							}
						}
					}
				}
			}
		}
	}

	if history, ok := res["history"].([]interface{}); ok && len(history) > 0 {
		if last, ok := history[len(history)-1].(map[string]interface{}); ok {
			if parts, ok := last["parts"].([]interface{}); ok {
				for _, p := range parts {
					if pMap, ok := p.(map[string]interface{}); ok {
						if txt, ok := pMap["text"].(string); ok && txt != "" {
							return txt
						}
					}
				}
			}
		}
	}

	return "No text response found"
}

// handleAgentStatus checks pod status for commerce and supply-chain.
func (s *Server) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	statuses := map[string]string{
		"commerce":     "Not Found",
		"supply-chain": "Not Found",
	}

	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", "agents", "-l", "app=commerce", "-o", "jsonpath={.items[0].status.phase}", "--kubeconfig", s.config.GetKubeconfigPath())
	if out, err := cmd.Output(); err == nil {
		statuses["commerce"] = string(out)
	}

	cmd = exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", "agents", "-l", "app=supply-chain", "-o", "jsonpath={.items[0].status.phase}", "--kubeconfig", s.config.GetKubeconfigPath())
	if out, err := cmd.Output(); err == nil {
		statuses["supply-chain"] = string(out)
	}

	json.NewEncoder(w).Encode(statuses)
}

// handleInventory returns current stock levels from Redis.
// FIX 4.3: handles "(nil)" responses from redis-cli when a key or field is missing.
func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
	ingredients := []string{"chocolate", "ananas", "banana", "walnut", "almond"}
	inventory := make(map[string]int)

	redisPod, err := s.getRedisPodName(r.Context())
	if err != nil {
		http.Error(w, "Redis not found", http.StatusServiceUnavailable)
		return
	}

	script := ""
	for _, ing := range ingredients {
		script += fmt.Sprintf("redis-cli HGET inventory:%s quantity; ", ing)
	}

	cmd := exec.CommandContext(r.Context(), "kubectl", "exec", "-n", "application", redisPod, "--", "sh", "-c", script)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+s.config.GetKubeconfigPath())

	out, err := cmd.Output()
	if err != nil {
		s.logger.Error("Failed to fetch inventory: %v", err)
		http.Error(w, "Failed to fetch inventory", http.StatusInternalServerError)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, line := range lines {
		if i >= len(ingredients) {
			break
		}
		line = strings.TrimSpace(line)
		// redis-cli returns "(nil)" when the key or field does not exist.
		if line == "" || strings.Contains(line, "(nil)") {
			inventory[ingredients[i]] = 0
			continue
		}
		var qty int
		if _, err := fmt.Sscanf(line, "%d", &qty); err != nil {
			inventory[ingredients[i]] = 0
			continue
		}
		inventory[ingredients[i]] = qty
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inventory)
}

// handleFulfillmentRoute returns route data.
// Returns mock data until the Fulfillment agent A2A integration is complete.
func (s *Server) handleFulfillmentRoute(w http.ResponseWriter, r *http.Request) {
	route := map[string]interface{}{
		"date": "2026-02-18",
		"stops": []string{
			"Danzigerkade 4, 1013 AP",
			"Herengracht 12, 1013 AP",
			"Prinsengracht 281, 1016 GW",
		},
		"total_distance": "5.2 km",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

// handleOrders returns list of orders parsed reliably in Go.
// FIX 4.1 + 4.2: replaces the awk NR%2 shell script with Go-side HGETALL parsing.
// The awk approach desynchronised when Redis values contained newlines (e.g., the
// cakes JSON field) and produced invalid JSON via manual gsub escaping.
// Now we fetch raw HGETALL output and build the map in Go using encoding/json.
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	redisPod, err := s.getRedisPodName(r.Context())
	if err != nil {
		http.Error(w, "Redis not found", http.StatusServiceUnavailable)
		return
	}

	// Step 1: scan for all order keys
	scanCmd := exec.CommandContext(r.Context(), "kubectl", "exec", "-n", "application", redisPod, "--", "sh", "-c", `redis-cli --scan --pattern "order:CAKE-*"`)
	scanCmd.Env = append(os.Environ(), "KUBECONFIG="+s.config.GetKubeconfigPath())

	scanOut, err := scanCmd.Output()
	if err != nil {
		s.logger.Error("Failed to scan order keys: %v", err)
		http.Error(w, "Failed to scan orders", http.StatusInternalServerError)
		return
	}

	rawKeys := strings.TrimSpace(string(scanOut))
	if rawKeys == "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	keys := strings.Split(rawKeys, "\n")
	orders := make([]map[string]string, 0, len(keys))

	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		// Step 2: HGETALL per key — returns alternating field/value lines
		hgetCmd := exec.CommandContext(r.Context(), "kubectl", "exec", "-n", "application", redisPod, "--", "sh", "-c", fmt.Sprintf("redis-cli HGETALL %s", key))
		hgetCmd.Env = append(os.Environ(), "KUBECONFIG="+s.config.GetKubeconfigPath())

		hgetOut, err := hgetCmd.Output()
		if err != nil {
			s.logger.Error("Failed to HGETALL %s: %v", key, err)
			continue
		}

		// Step 3: parse in Go — encoding/json handles all escaping correctly
		order := parseHGETALL(string(hgetOut))
		if len(order) > 0 {
			orders = append(orders, order)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		s.logger.Error("Failed to encode orders: %v", err)
	}
}

// parseHGETALL parses the newline-delimited output of redis-cli HGETALL into a
// map[string]string. HGETALL returns alternating field/value lines:
//
//	field1\nvalue1\nfield2\nvalue2\n...
//
// If the output has an odd number of lines (corrupt data), the trailing field
// is dropped rather than paired with an empty value.
func parseHGETALL(raw string) map[string]string {
	lines := strings.Split(raw, "\n")
	result := make(map[string]string)

	// Remove the trailing empty element produced by a trailing newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for i := 0; i+1 < len(lines); i += 2 {
		field := strings.TrimSpace(lines[i])
		value := lines[i+1] // preserve value exactly; json.Encoder handles escaping
		if field != "" {
			result[field] = value
		}
	}

	return result
}

// handleOrderStats returns aggregate order statistics.
func (s *Server) handleOrderStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"count":   0,
		"revenue": 0.0,
		"average": 0.0,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleAgentActivity returns recent agent interaction logs.
func (s *Server) handleAgentActivity(w http.ResponseWriter, r *http.Request) {
	logs := []map[string]string{
		{"timestamp": time.Now().Format(time.RFC3339), "system": "commerce", "message": "Agent initialized"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// getRedisPodName resolves the name of the first redis pod in the application namespace.
func (s *Server) getRedisPodName(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", "application", "-l", "app=redis", "-o", "jsonpath={.items[0].metadata.name}", "--kubeconfig", s.config.GetKubeconfigPath())
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
