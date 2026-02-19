package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type AuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	Account       string `json:"account,omitempty"`
	Project       string `json:"project,omitempty"`
	Region        string `json:"region,omitempty"`
	Provider      string `json:"provider"`
	StateBucket   string `json:"stateBucket,omitempty"`
	Error         string `json:"error,omitempty"`
}

type GlobalStatus struct {
	Infra   string `json:"infra"`   // "Running", "Not Created", "Error"
	K8s     string `json:"k8s"`     // "Ready", "Not Ready", "Error"
	Tools   string `json:"tools"`   // "Installed", "Not Installed"
	Apps    string `json:"apps"`    // "Deployed", "Not Deployed"
	Tunnel  string `json:"tunnel"`  // Connected, Reconnecting, etc.
	Version string `json:"version"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.cachedStatusMu.RLock()
	status := s.cachedStatus
	s.cachedStatusMu.RUnlock()

	if status == nil {
		// Not yet polled, return defaults
		status = &GlobalStatus{
			Infra:   "Not Created",
			K8s:     "Not Ready",
			Tools:   "Not Installed",
			Apps:    "Not Deployed",
			Tunnel:  string(TunnelStatusIdle),
			Version: "0.1.0",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := AuthStatus{
		Provider: s.cloud,
	}

	// Get Project ID from provider (reads terraform.tfvars)
	projectID, err := s.provider.GetProjectID(s.config.GetTerraformDir())
	if err == nil {
		status.Project = projectID
	}

	// Get Region from terraform.tfvars
	region, err := readTerraformVariable(s.config.GetTerraformDir(), "region")
	if err == nil {
		status.Region = region
	}

	// Get State Bucket from terraform.tfvars
	bucket, err := readTerraformVariable(s.config.GetTerraformDir(), "state_bucket")
	if err == nil && bucket != "" {
		status.StateBucket = bucket
	}

	// Check Authentication
	if s.cloud == "gcp" {
		// gcloud config get-value account
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "gcloud", "config", "get-value", "account")
		out, err := cmd.Output()
		if err == nil {
			account := strings.TrimSpace(string(out))
			if account != "" {
				status.Authenticated = true
				status.Account = account
			} else {
				status.Error = "No active account found in gcloud"
			}
		} else {
			status.Error = "Failed to check gcloud auth"
		}
	} else {
		// Assume authenticated for other providers (since Validate passed)
		status.Authenticated = true
		status.Account = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// readTerraformVariable reads a variable value from terraform.tfvars.
// Copied from deploy_infra.go to avoid dependency cycle or exporting it there.
func readTerraformVariable(tfDir, varName string) (string, error) {
	tfvarsPath := filepath.Join(tfDir, "terraform.tfvars")
	file, err := os.Open(tfvarsPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == varName {
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, `"'`)
			return value, nil
		}
	}
	return "", nil
}

// handleOperation executes a CLI command and streams output to WebSocket.
func (s *Server) handleOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	operation := strings.TrimPrefix(r.URL.Path, "/api/")
	
	// Validate operation
	allowed := map[string]bool{
		"deploy-infra": true, "deploy-tools": true,
		"deploy": true, "destroy": true, "backup": true, "restore": true,
		"deploy-agents": true, "seed-data": true, "cleanup-cakes": true,
	}
	if !allowed[operation] {
		http.Error(w, "Invalid operation", http.StatusBadRequest)
		return
	}

	// Lock mutex to prevent concurrent operations
	if !s.opMu.TryLock() {
		http.Error(w, "Another operation is already in progress", http.StatusConflict)
		return
	}

	// Prepare arguments
	args := []string{operation, "--cloud", s.cloud, "--verbose"}
	if operation == "destroy" {
		args = []string{"destroy", "--cloud", s.cloud, "--verbose"}
	}

	// Handle operation-specific flags from query params
	if operation == "restore" {
		if backup := r.URL.Query().Get("backup"); backup != "" {
			args = append(args, "--backup", backup)
		}
		if r.URL.Query().Get("clean") == "true" {
			args = append(args, "--clean")
		}
	}

	exe, err := os.Executable()
	if err != nil {
		s.opMu.Unlock()
		http.Error(w, "Failed to get executable path", http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(exe, args...)
	
	// Only set managed tunnel for non-infra/lifecycle operations
	// deploy-infra, deploy, and destroy must manage their own connectivity/lifecycle
	if operation != "deploy-infra" && operation != "deploy" && operation != "destroy" {
		cmd.Env = append(os.Environ(), "K8SLAB_TUNNEL_MANAGED=true")
	} else {
		cmd.Env = os.Environ()
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		s.opMu.Unlock()
		http.Error(w, fmt.Sprintf("Failed to start command: %v", err), http.StatusInternalServerError)
		return
	}

	// Send immediate response
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started", "operation": operation})

	// Background streaming
	go func() {
		defer s.opMu.Unlock()

		s.wsHub.Broadcast("start", fmt.Sprintf(">>> Starting operation: %s\n", operation))

		// Stream stdout and stderr concurrently to avoid pipe deadlock.
		// io.MultiReader would read stdout until EOF first, blocking if stderr buffer fills.
		var wg sync.WaitGroup
		scan := func(r io.Reader) {
			defer wg.Done()
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				s.wsHub.Broadcast("log", sc.Text())
			}
		}
		wg.Add(2)
		go scan(stdout)
		go scan(stderr)
		wg.Wait()

		err := cmd.Wait()
		if err != nil {
			s.wsHub.Broadcast("error", fmt.Sprintf("\n>>> Operation failed: %v", err))
		} else {
			s.wsHub.Broadcast("done", "\n>>> Operation completed successfully.")
			
			// If deploy-infra (or deploy) succeeded, we should start the persistent tunnel
			if operation == "deploy-infra" || operation == "deploy" {
				s.infraReady = true
				// Refresh infra info and start tunnel
				if projectID, err := s.provider.GetProjectID(s.config.GetTerraformDir()); err == nil {
					if cpName, zone, err := s.getControlPlaneInfo(); err == nil {
						// Stop existing if any (re-deploy case)
						if s.tunnel != nil {
							s.tunnel.Stop()
						}
						s.tunnel = NewTunnelManager(projectID, cpName, zone, s.logger)
						s.tunnel.Start(context.Background())
					}
				}
			}
			
			// If destroy succeeded, stop the tunnel and reset infra state
			if operation == "destroy" {
				s.infraReady = false
				if s.tunnel != nil {
					s.tunnel.Stop()
					s.tunnel = nil
				}
			}
		}
	}()
}
