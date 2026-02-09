package ui

import (
	"context"
	"encoding/json"
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

// handleSnapshots returns GCE disk snapshots.
func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	if s.cloud != "gcp" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	projectID, err := s.provider.GetProjectID(s.config.GetTerraformDir())
	if err != nil {
		http.Error(w, "Failed to get project ID", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcloud", "compute", "snapshots", "list",
		"--project", projectID, "--format=json")
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// handlePodDeployment returns the deployment YAML for a pod's owner.
func (s *Server) handlePodDeployment(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
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

	// Get pod to find owner deployment name (strip pod hash suffix)
	// Convention: deployment name is pod name without the replicaset hash
	deployName := podName
	parts := strings.Split(podName, "-")
	if len(parts) > 2 {
		deployName = strings.Join(parts[:len(parts)-2], "-")
	}

	cmd := exec.CommandContext(ctx, "kubectl", "get", "deployment", deployName, "-n", namespace,
		"-o", "yaml", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Deployment not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}

// handlePodService returns the service YAML for a pod's related service.
func (s *Server) handlePodService(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
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

	// Derive service name from pod name (strip replicaset hash)
	svcName := podName
	parts := strings.Split(podName, "-")
	if len(parts) > 2 {
		svcName = strings.Join(parts[:len(parts)-2], "-")
	}

	cmd := exec.CommandContext(ctx, "kubectl", "get", "service", svcName, "-n", namespace,
		"-o", "yaml", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Service not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}

// --- Redis Handlers ---

// execRedis runs a redis-cli command in the redis pod.
func (s *Server) execRedis(ctx context.Context, args ...string) ([]byte, error) {
	// kubectl exec -n application deploy/redis --kubeconfig ... -- redis-cli <args>
	fullArgs := []string{"exec", "-n", "application", "deploy/redis", "--kubeconfig", s.config.GetKubeconfigPath(), "--", "redis-cli"}
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(ctx, "kubectl", fullArgs...)
	return cmd.Output()
}

func (s *Server) handleRedisKeys(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "*"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Use KEYS for simplicity (in prod SCAN is better, but this is a lab)
	output, err := s.execRedis(ctx, "KEYS", pattern)
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse newlines to array
	keys := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(keys) == 1 && keys[0] == "" {
		keys = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (s *Server) handleRedisGet(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	// /api/redis/get/{key}
	if len(pathParts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	key := pathParts[4]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	output, err := s.execRedis(ctx, "GET", key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}

func (s *Server) handleRedisSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	output, err := s.execRedis(ctx, "SET", req.Key, req.Value)
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(output)
}

func (s *Server) handleRedisDel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	key := pathParts[4]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	output, err := s.execRedis(ctx, "DEL", key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(output)
}

func (s *Server) handleRedisFlush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	output, err := s.execRedis(ctx, "FLUSHDB")
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(output)
}

// --- Backup Handler ---

func (s *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	// /api/backups/{name}
	if len(pathParts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	name := pathParts[3]

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// velero backup delete <name> --confirm
	cmd := exec.CommandContext(ctx, "velero", "backup", "delete", name, "--confirm", "--kubeconfig", s.config.GetKubeconfigPath())
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete backup: %v\n%s", err, output), http.StatusInternalServerError)
		return
	}

	w.Write(output)
}
