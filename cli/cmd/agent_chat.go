package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
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
	Long: `Send a single message or start an interactive conversation with Magic Cake agents.

Without a message argument, enters interactive mode (Ctrl+C or empty line to exit).

Examples:
  k8s-lab agent-chat --system commerce --cloud gcp
  k8s-lab agent-chat --system commerce --cloud gcp "I want to order a cake"
  k8s-lab agent-chat --system commerce --session abc123 --cloud gcp "Make it chocolate"
  k8s-lab agent-chat --system supply-chain --cloud gcp "What is current stock?"`,
	Args: cobra.MaximumNArgs(1),
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

	if agentSystem != "commerce" && agentSystem != "supply-chain" {
		return fmt.Errorf("invalid system: %s (must be 'commerce' or 'supply-chain')", agentSystem)
	}

	log.Info("==============================================")
	log.Info("  Magic Cake - Agent Chat")
	log.Info("  System: %s", agentSystem)
	if sessionID != "" {
		log.Info("  Session: %s", sessionID)
	}
	log.Info("==============================================")

	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	// Create tunnel to K8s API
	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	log.Info("Creating tunnel...")
	_, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()

	// sessionID is the initial contextId hint; the agent returns the real one on first response

	// Determine service and local port
	var serviceName string
	var port int
	if agentSystem == "commerce" {
		serviceName = "commerce"
		port = 8001
	} else {
		serviceName = "supply-chain"
		port = 8002
	}

	// Start kubectl port-forward in background
	log.Info("Port-forwarding %s:%d...", serviceName, port)
	pfCtx, pfCancel := context.WithCancel(ctx)

	pfCmd := exec.CommandContext(pfCtx, "kubectl", "port-forward",
		"-n", "agents",
		"svc/"+serviceName,
		fmt.Sprintf("%d:%d", port, port),
	)
	if err := pfCmd.Start(); err != nil {
		pfCancel()
		return fmt.Errorf("failed to start port-forward: %w", err)
	}
	defer func() {
		pfCancel()
		pfCmd.Wait() //nolint:errcheck
	}()

	// Poll until port-forward is ready (up to 10s)
	if err := waitForPort(port, 10*time.Second); err != nil {
		return fmt.Errorf("port-forward not ready: %w", err)
	}

	if len(args) == 1 {
		// Single-turn mode
		log.Info("Sending message...")
		response, contextID, err := sendA2AMessage(ctx, port, sessionID, args[0])
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		if response != "" {
			fmt.Println()
			fmt.Println(response)
			fmt.Println()
		}
		log.Info("Continue conversation: --session %s", contextID)
	} else {
		// Interactive mode
		fmt.Println()
		fmt.Println("Interactive mode. Empty line or Ctrl+C to exit.")
		fmt.Println()

		contextID := sessionID

		// For new sessions, trigger the agent to greet first
		if contextID == "" {
			greeting, newCtxID, err := sendA2AMessage(ctx, port, contextID, "hello")
			if newCtxID != "" {
				contextID = newCtxID
			}
			if err == nil && greeting != "" {
				fmt.Println("Agent:", greeting)
				fmt.Println()
			}
		}

		printSession := func() {
			if contextID != "" {
				fmt.Printf("\nResume with: --session %s\n", contextID)
			}
		}

		// Print session ID on Ctrl+C
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			printSession()
			os.Exit(0)
		}()

		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("You: ")
			if !scanner.Scan() {
				break
			}
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				break
			}
			response, newContextID, err := sendA2AMessage(ctx, port, contextID, text)
			if newContextID != "" {
				contextID = newContextID
			}
			if err != nil {
				fmt.Printf("Error: %v\n\n", err)
				continue
			}
			if response != "" {
				fmt.Println()
				fmt.Println("Agent:", response)
				fmt.Println()
			}
		}
		printSession()
	}

	return nil
}

// waitForPort polls until the TCP port accepts connections or the timeout expires.
func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("localhost:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("port %d not ready after %s", port, timeout)
}

// a2aResponse is the JSON-RPC response from the A2A endpoint.
type a2aResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  struct {
		ContextID string `json:"contextId"`
		Artifacts []struct {
			Parts []struct {
				Kind string `json:"kind"`
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"artifacts"`
		History []struct {
			Role  string `json:"role"`
			Parts []struct {
				Kind string `json:"kind"`
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"history"`
		// Status carries task outcome; State=="failed" means the agent errored.
		// Text is in Status.Message.Parts (not in artifacts/history on failure).
		Status struct {
			State   string `json:"state"`
			Message struct {
				Parts []struct {
					Kind string `json:"kind"`
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"message"`
		} `json:"status"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// sendA2AMessage sends a message using the A2A JSON-RPC v0.3 protocol.
// Returns (responseText, contextID, error). contextID is used for multi-turn sessions.
func sendA2AMessage(ctx context.Context, port int, contextID, message string) (string, string, error) {
	msg := map[string]interface{}{
		"role":      "user",
		"parts":     []map[string]string{{"kind": "text", "text": message}},
		"messageId": fmt.Sprintf("msg-%d", time.Now().UnixNano()),
	}
	if contextID != "" {
		msg["contextId"] = contextID
	}
	params := map[string]interface{}{
		"message": msg,
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "message/send",
		"id":      "1",
		"params":  params,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d/", port)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 2 * time.Minute,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("agent returned %d: %s", resp.StatusCode, string(body))
	}

	var result a2aResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return "", "", fmt.Errorf("agent error %d: %s", result.Error.Code, result.Error.Message)
	}

	text, failed := extractA2AText(result.Result)
	if failed {
		return "", "", fmt.Errorf("agent error: %s", text)
	}
	return text, result.Result.ContextID, nil
}

// extractA2AText pulls the agent's response text from an A2A result.
// Returns (text, failed). failed=true means the task status was "failed" and
// text contains the error description rather than a normal agent reply.
//
// Priority: artifacts > history (after last user msg) > status.message (errors).
func extractA2AText(result struct {
	ContextID string `json:"contextId"`
	Artifacts []struct {
		Parts []struct {
			Kind string `json:"kind"`
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"artifacts"`
	History []struct {
		Role  string `json:"role"`
		Parts []struct {
			Kind string `json:"kind"`
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"history"`
	Status struct {
		State   string `json:"state"`
		Message struct {
			Parts []struct {
				Kind string `json:"kind"`
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"message"`
	} `json:"status"`
}) (string, bool) {
	// 1. Artifacts: direct response for this turn
	for _, artifact := range result.Artifacts {
		for _, part := range artifact.Parts {
			if part.Kind == "text" && part.Text != "" {
				return part.Text, false
			}
		}
	}

	// 2. History fallback: find last user message, return first agent text after it
	lastUserIdx := -1
	for i := len(result.History) - 1; i >= 0; i-- {
		if result.History[i].Role == "user" {
			lastUserIdx = i
			break
		}
	}
	for i := lastUserIdx + 1; i < len(result.History); i++ {
		if result.History[i].Role == "agent" {
			for _, part := range result.History[i].Parts {
				if part.Kind == "text" && part.Text != "" {
					return part.Text, false
				}
			}
		}
	}

	// 3. Status.message fallback: task failed or agent returned error
	if result.Status.State == "failed" || result.Status.State == "error" {
		for _, part := range result.Status.Message.Parts {
			if part.Kind == "text" && part.Text != "" {
				return part.Text, true
			}
		}
		return "agent task failed with no error details", true
	}

	return "", false
}
