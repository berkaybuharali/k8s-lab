package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
)

var (
	agentSystem string
	sessionID   string
)

func init() {
	rootCmd.AddCommand(agentChatCmd)
	agentChatCmd.Flags().StringVar(&agentSystem, "system", "", "Agent system: commerce or supply-chain (required)")
	agentChatCmd.Flags().StringVar(&sessionID, "session", "", "Session ID for multi-turn conversation (optional)")
	agentChatCmd.MarkFlagRequired("system")
}

var agentChatCmd = &cobra.Command{
	Use:   "agent-chat [message]",
	Short: "Chat with Magic Cake agents via A2A protocol",
	Long: `Send messages to Magic Cake agents and get responses.

Examples:
  # Chat with Commerce system (customer flow)
  k8s-lab agent-chat --system commerce --cloud gcp "I want to order a cake"

  # Continue conversation with session
  k8s-lab agent-chat --system commerce --session abc123 --cloud gcp "Make it chocolate"

  # Chat with Supply Chain system (backoffice queries)
  k8s-lab agent-chat --system supply-chain --cloud gcp "What is current stock?"
  k8s-lab agent-chat --system supply-chain --cloud gcp "Show me all orders for tomorrow"

The agent will respond in natural language. For multi-turn conversations, use --session flag.`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentChat,
}

func runAgentChat(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	// Validate system
	if agentSystem != "commerce" && agentSystem != "supply-chain" {
		return fmt.Errorf("invalid system: %s (must be 'commerce' or 'supply-chain')", agentSystem)
	}

	message := args[0]

	log.Info("==============================================")
	log.Info("  Magic Cake - Agent Chat")
	log.Info("  System: %s", agentSystem)
	if sessionID != "" {
		log.Info("  Session: %s", sessionID)
	}
	log.Info("==============================================")

	// Check prerequisites
	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	// Generate session ID if not provided
	if sessionID == "" {
		sessionID = fmt.Sprintf("cli-%d", time.Now().Unix())
		log.Info("Generated session ID: %s", sessionID)
	}

	// Get K8s client
	client, err := k8s.NewClient(cfg.GetKubeConfigPath())
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Determine service and port
	var serviceName string
	var port int
	if agentSystem == "commerce" {
		serviceName = "commerce"
		port = 8001
	} else {
		serviceName = "supply-chain"
		port = 8002
	}

	log.Info("Setting up port-forward to %s service...", serviceName)

	// Create port-forward
	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	defer close(stopChan)

	localPort := port // Use same port locally

	go func() {
		err := client.PortForward(
			ctx,
			"agents",
			serviceName,
			[]string{fmt.Sprintf("%d:%d", localPort, port)},
			stopChan,
			readyChan,
		)
		if err != nil {
			log.Error("Port-forward error: %v", err)
		}
	}()

	// Wait for port-forward to be ready
	select {
	case <-readyChan:
		log.Info("Port-forward established")
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for port-forward")
	}

	// Give it a moment to stabilize
	time.Sleep(1 * time.Second)

	// Send message to agent via A2A protocol
	log.Info("Sending message to %s agent...", agentSystem)
	response, err := sendA2AMessage(ctx, localPort, agentSystem, sessionID, message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Print response
	log.Info("")
	log.Info("=== Agent Response ===")
	fmt.Println(response)
	log.Info("")
	log.Info("Session ID: %s (use --session %s for follow-up)", sessionID, sessionID)

	return nil
}

// sendA2AMessage sends a message to an agent via A2A protocol
func sendA2AMessage(ctx context.Context, port int, appName, sessionID, message string) (string, error) {
	// Construct A2A request payload
	// Based on google-adk A2A protocol format
	payload := map[string]interface{}{
		"app_name":   appName,
		"user_id":    "cli-user",
		"session_id": sessionID,
		"new_message": map[string]interface{}{
			"role": "user",
			"parts": []map[string]string{
				{"text": message},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send POST request to /run endpoint
	url := fmt.Sprintf("http://localhost:%d/run", port)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("agent returned error (status %d): %s", resp.StatusCode, string(body))
	}

	// Read response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text from response
	// A2A response format: {response_message: {parts: [{text: "..."}]}}
	responseMsg, ok := result["response_message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: missing response_message")
	}

	parts, ok := responseMsg["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", fmt.Errorf("invalid response format: missing parts")
	}

	var textParts []string
	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		if text, ok := partMap["text"].(string); ok {
			textParts = append(textParts, text)
		}
	}

	if len(textParts) == 0 {
		return "", fmt.Errorf("no text in agent response")
	}

	return strings.Join(textParts, "\n"), nil
}
