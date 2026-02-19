package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// gcsImageRegex matches GCS object paths for image files in agent responses.
var gcsImageRegex = regexp.MustCompile(`gs://[^\s\]"']+\.(?:png|jpg|jpeg|webp)`)

// rewriteGCSPaths replaces gs:// image URLs in agent text with /api/image proxy URLs
// so the browser can load private GCS objects through the authenticated backend.
func rewriteGCSPaths(text string) string {
	return gcsImageRegex.ReplaceAllStringFunc(text, func(match string) string {
		return "/api/image?path=" + url.QueryEscape(match)
	})
}

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
		"id":      uuid.New().String(),
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

	text := rewriteGCSPaths(extractAgentResponseText(resp))

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

	cmd := exec.CommandContext(r.Context(), "kubectl", "exec", "-n", "agents", redisPod, "--", "sh", "-c", script)
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

// handleOrders returns list of orders from Redis in a single kubectl exec.
// Scans for order:CAKE-* keys and HGETALLs each in one shell pipeline to avoid
// the N+1 kubectl exec overhead of the previous implementation.
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	redisPod, err := s.getRedisPodName(r.Context())
	if err != nil {
		http.Error(w, "Redis not found", http.StatusServiceUnavailable)
		return
	}

	// Single exec: scan + HGETALL for each key.
	// Output format: lines starting with "BOUNDARY:<key>" delimit blocks.
	script := `redis-cli --scan --pattern "order:CAKE-*" | while IFS= read -r key; do printf 'BOUNDARY:%s\n' "$key"; redis-cli HGETALL "$key"; done`
	cmd := exec.CommandContext(r.Context(), "kubectl", "exec", "-n", "agents", redisPod, "--", "sh", "-c", script)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+s.config.GetKubeconfigPath())

	out, err := cmd.Output()
	if err != nil {
		s.logger.Error("Failed to fetch orders: %v", err)
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	raw := strings.TrimSpace(string(out))
	orders := make([]map[string]string, 0)

	if raw != "" {
		for _, block := range strings.Split(raw, "BOUNDARY:") {
			block = strings.TrimSpace(block)
			if block == "" {
				continue
			}
			// First line is the key name; rest is HGETALL alternating field/value lines.
			nl := strings.IndexByte(block, '\n')
			if nl < 0 {
				continue
			}
			order := parseHGETALL(block[nl+1:])
			if len(order) > 0 {
				orders = append(orders, order)
			}
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

// handleOrderStats returns aggregate order statistics computed from Redis in a single exec.
func (s *Server) handleOrderStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	zeros := map[string]interface{}{"count": 0, "revenue": 0.0, "average": 0.0}

	redisPod, err := s.getRedisPodName(r.Context())
	if err != nil {
		json.NewEncoder(w).Encode(zeros)
		return
	}

	// Single exec: scan + HGET total_price for each key.
	script := `redis-cli --scan --pattern "order:CAKE-*" | while IFS= read -r key; do redis-cli HGET "$key" total_price; done`
	cmd := exec.CommandContext(r.Context(), "kubectl", "exec", "-n", "agents", redisPod, "--", "sh", "-c", script)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+s.config.GetKubeconfigPath())

	out, err := cmd.Output()
	if err != nil {
		s.logger.Error("Failed to fetch order stats: %v", err)
		json.NewEncoder(w).Encode(zeros)
		return
	}

	count := 0
	var totalRevenue float64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "(nil)") {
			continue
		}
		var price float64
		if _, err := fmt.Sscanf(line, "%f", &price); err != nil {
			continue
		}
		count++
		totalRevenue += price
	}

	var average float64
	if count > 0 {
		average = totalRevenue / float64(count)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   count,
		"revenue": totalRevenue,
		"average": average,
	})
}

// handleImageProxy streams a GCS object through the backend using authenticated gcloud credentials.
// GCS objects are private; the browser cannot load gs:// URLs directly.
func (s *Server) handleImageProxy(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !strings.HasPrefix(path, "gs://") {
		http.Error(w, "Invalid path: must start with gs://", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcloud", "storage", "cat", path)
	out, err := cmd.Output()
	if err != nil {
		s.logger.Error("Failed to fetch image from GCS %s: %v", path, err)
		http.Error(w, "Failed to fetch image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(out)
}

// handleAgentActivity returns recent agent interaction logs.
func (s *Server) handleAgentActivity(w http.ResponseWriter, r *http.Request) {
	logs := []map[string]string{
		{"timestamp": time.Now().Format(time.RFC3339), "system": "commerce", "message": "Agent initialized"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// getRedisPodName resolves the name of the first redis pod in the agents namespace.
func (s *Server) getRedisPodName(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", "agents", "-l", "app=redis", "-o", "jsonpath={.items[0].metadata.name}", "--kubeconfig", s.config.GetKubeconfigPath())
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
