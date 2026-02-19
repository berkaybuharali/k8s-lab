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
)

// kubectl runs a kubectl subcommand with the managed kubeconfig appended.
// Not suitable for kubectl exec (kubeconfig must precede "--"); use execRedis for that.
func (s *Server) kubectl(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "kubectl",
		append(args, "--kubeconfig", s.config.GetKubeconfigPath())...,
	).Output()
}

// nsFromQuery returns the "ns" query param, defaulting to "agents".
func nsFromQuery(r *http.Request) string {
	if ns := r.URL.Query().Get("ns"); ns != "" {
		return ns
	}
	return "agents"
}

// resourceName derives a K8s resource name from a pod name by stripping
// the two trailing hash segments added by ReplicaSet controllers.
func resourceName(podName string) string {
	parts := strings.Split(podName, "-")
	if len(parts) > 2 {
		return strings.Join(parts[:len(parts)-2], "-")
	}
	return podName
}

// ensureGet rejects non-GET requests.
func ensureGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// serveCommandOutput runs an external command and writes its output as application/json.
// Automatically appends --kubeconfig for kubectl/velero commands.
func (s *Server) serveCommandOutput(w http.ResponseWriter, r *http.Request, command string, args ...string) {
	if !ensureGet(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	fullArgs := args
	if command == "kubectl" || command == "velero" {
		fullArgs = append(args, "--kubeconfig", s.config.GetKubeconfigPath())
	}
	output, err := exec.CommandContext(ctx, command, fullArgs...).Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Command failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// --- Infra / K8s data handlers ---

func (s *Server) handleMapsKey(w http.ResponseWriter, r *http.Request) {
	key := os.Getenv("GOOGLE_API_KEY")
	if key == "" {
		key = os.Getenv("GOOGLE_MAPS_API_KEY")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	s.serveCommandOutput(w, r, "kubectl", "get", "nodes", "-o", "json")
}

func (s *Server) handlePods(w http.ResponseWriter, r *http.Request) {
	s.serveCommandOutput(w, r, "kubectl", "get", "pods", "-n", nsFromQuery(r), "-o", "json")
}

func (s *Server) handlePVCs(w http.ResponseWriter, r *http.Request) {
	s.serveCommandOutput(w, r, "kubectl", "get", "pvc", "-n", nsFromQuery(r), "-o", "json")
}

func (s *Server) handleBackups(w http.ResponseWriter, r *http.Request) {
	s.serveCommandOutput(w, r, "velero", "backup", "get", "-o", "json")
}

func (s *Server) handleNamespaces(w http.ResponseWriter, r *http.Request) {
	s.serveCommandOutput(w, r, "kubectl", "get", "ns", "-o", "json")
}

func (s *Server) handleTerraformResources(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

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

	output, err := exec.CommandContext(ctx, "gcloud", "compute", "snapshots", "list",
		"--project", projectID, "--format=json").Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get snapshots: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

// --- Pod detail handlers ---

func (s *Server) handlePodDetail(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := s.kubectl(ctx, "get", "pod", parts[3], "-n", nsFromQuery(r), "-o", "json")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pod: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (s *Server) handlePodLogs(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := s.kubectl(ctx, "logs", parts[3], "-n", nsFromQuery(r), "--tail=100")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(out)
}

// handlePodResource is shared by handlePodDeployment and handlePodService.
// It strips the pod hash suffix to derive the owner resource name.
func (s *Server) handlePodResource(w http.ResponseWriter, r *http.Request, resource string) {
	if !ensureGet(w, r) {
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := s.kubectl(ctx, "get", resource, resourceName(parts[3]), "-n", nsFromQuery(r), "-o", "yaml")
	if err != nil {
		http.Error(w, fmt.Sprintf("%s not found: %v", resource, err), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(out)
}

func (s *Server) handlePodDeployment(w http.ResponseWriter, r *http.Request) {
	s.handlePodResource(w, r, "deployment")
}

func (s *Server) handlePodService(w http.ResponseWriter, r *http.Request) {
	s.handlePodResource(w, r, "service")
}

// --- Redis handlers ---

// execRedis runs a redis-cli command inside the redis pod.
// kubeconfig is placed before "--" so it is not passed to redis-cli.
func (s *Server) execRedis(ctx context.Context, args ...string) ([]byte, error) {
	base := []string{"exec", "-n", "agents", "deploy/redis", "--kubeconfig", s.config.GetKubeconfigPath(), "--", "redis-cli"}
	return exec.CommandContext(ctx, "kubectl", append(base, args...)...).Output()
}

func (s *Server) handleRedisDBSize(w http.ResponseWriter, r *http.Request) {
	if !ensureGet(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	output, err := s.execRedis(ctx, "DBSIZE")
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
		return
	}
	raw := strings.TrimSpace(string(output))
	count := 0
	if fields := strings.Fields(raw); len(fields) >= 2 {
		fmt.Sscanf(fields[1], "%d", &count)
	} else {
		fmt.Sscanf(raw, "%d", &count)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
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

	output, err := s.execRedis(ctx, "KEYS", pattern)
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
		return
	}
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
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	output, err := s.execRedis(ctx, "GET", parts[4])
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
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	output, err := s.execRedis(ctx, "DEL", parts[4])
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

// --- Backup handler ---

func (s *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "velero", "backup", "delete", parts[3],
		"--confirm", "--kubeconfig", s.config.GetKubeconfigPath()).Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete backup: %v\n%s", err, output), http.StatusInternalServerError)
		return
	}
	w.Write(output)
}
