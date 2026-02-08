package ui

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ensureGet checks if request method is GET.
func ensureGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// handleNodes returns list of nodes.
func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes", "-o", "json", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get nodes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handlePods returns list of pods for a namespace.
func (s *Server) handlePods(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	namespace := r.URL.Query().Get("ns")
	if namespace == "" {
		namespace = "application" // Default
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", namespace, "-o", "json", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pods: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handlePVCs returns list of PVCs.
func (s *Server) handlePVCs(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	namespace := r.URL.Query().Get("ns")
	if namespace == "" {
		namespace = "application"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "get", "pvc", "-n", namespace, "-o", "json", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get PVCs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handleBackups returns list of Velero backups.
func (s *Server) handleBackups(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "velero", "backup", "get", "-o", "json", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get backups: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handleTerraformResources returns terraform state.
func (s *Server) handleTerraformResources(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second) // Terraform might be slower
	defer cancel()

	// terraform show -json
	cmd := exec.CommandContext(ctx, "terraform", "show", "-json")
	cmd.Dir = s.config.GetTerraformDir()
	
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get terraform state: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handleNamespaces returns list of namespaces.
func (s *Server) handleNamespaces(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "get", "ns", "-o", "json", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get namespaces: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handlePodDetail returns single pod details.
func (s *Server) handlePodDetail(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	// /api/pods/{name} -> ["", "api", "pods", "name"]
	if len(pathParts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	podName := pathParts[3]
	namespace := r.URL.Query().Get("ns")
	if namespace == "" {
		namespace = "application"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Fetch Pod JSON
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pod", podName, "-n", namespace, "-o", "json", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pod: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handlePodLogs returns pod logs.
func (s *Server) handlePodLogs(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	// /api/pods/{name}/logs -> ["", "api", "pods", "name", "logs"]
	if len(pathParts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	podName := pathParts[3]
	namespace := r.URL.Query().Get("ns")
	if namespace == "" {
		namespace = "application"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Simple log fetch (last 100 lines)
	cmd := exec.CommandContext(ctx, "kubectl", "logs", podName, "-n", namespace, "--tail=100", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Return as plain text
	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}