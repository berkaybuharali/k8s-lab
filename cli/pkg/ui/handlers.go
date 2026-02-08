package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type AuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	Account       string `json:"account,omitempty"`
	Project       string `json:"project,omitempty"`
	Region        string `json:"region,omitempty"`
	Provider      string `json:"provider"`
	Error         string `json:"error,omitempty"`
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
