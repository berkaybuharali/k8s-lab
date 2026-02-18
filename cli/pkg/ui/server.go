package ui

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/config"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/terraform"
)

//go:embed dist
var staticFiles embed.FS

// Server represents the UI HTTP server.
type Server struct {
	port       int
	cloud      string
	logger     *logger.Logger
	provider   cloud.Provider
	config     *config.Config
	tunnel     *TunnelManager
	wsHub      *WebSocketHub
	opMu       sync.Mutex
	infraReady bool // true when terraform outputs show infra exists

	// k8sClient is lazily initialised on first successful tunnel connection.
	// Used by refreshStatus to replace kubectl subprocess calls.
	k8sClient *k8s.Client

	// Cached status for instant responses
	cachedStatus   *GlobalStatus
	cachedStatusMu sync.RWMutex
}

// NewServer creates a new UI server instance.
func NewServer(port int, cloudName string, cfg *config.Config, log *logger.Logger, provider cloud.Provider) (*Server, error) {
	s := &Server{
		port:     port,
		cloud:    cloudName,
		logger:   log,
		provider: provider,
		config:   cfg,
		wsHub:    NewWebSocketHub(),
	}

	return s, nil
}

// Start starts the HTTP server and blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	// Check if infra exists and setup tunnel
	if projectID, err := s.provider.GetProjectID(s.config.GetTerraformDir()); err == nil {
		if cpName, zone, err := s.getControlPlaneInfo(); err == nil && cpName != "" {
			s.infraReady = true
			s.tunnel = NewTunnelManager(projectID, cpName, zone, s.logger)
			s.tunnel.Start(ctx)
		} else {
			s.logger.Info("Infrastructure not found or incomplete, tunnel will remain Idle")
		}
	} else {
		s.logger.Debug("Could not get project ID (normal if terraform.tfvars not set): %v", err)
	}

	// Start background status poller (refreshes every 10s, handler returns cached result instantly)
	go s.statusPoller(ctx)

	// Setup router
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/auth", s.handleAuth)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/ws/logs", s.wsHub.HandleWebSocket)

	// Operation Routes
	ops := []string{
		"deploy-infra", "deploy-tools", "deploy-applications",
		"deploy", "destroy", "seed-redis", "backup", "restore",
		"deploy-agents", "seed-inventory", "seed-data", "cleanup-cakes",
	}
	for _, op := range ops {
		mux.HandleFunc("/api/"+op, s.handleOperation)
	}

	// Data Routes
	mux.HandleFunc("/api/nodes", s.handleNodes)
	mux.HandleFunc("/api/pods", s.handlePods)
	// Match specific pod routes first? No, ServeMux matches longest pattern.
	// /api/pods/ matches /api/pods/foo and /api/pods/foo/logs
	mux.HandleFunc("/api/pods/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/logs") {
			s.handlePodLogs(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/deployment") {
			s.handlePodDeployment(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/service") {
			s.handlePodService(w, r)
		} else {
			s.handlePodDetail(w, r)
		}
	})
	mux.HandleFunc("/api/pvcs", s.handlePVCs)
	mux.HandleFunc("/api/snapshots", s.handleSnapshots)
	mux.HandleFunc("/api/backups", s.handleBackups)
	mux.HandleFunc("/api/backups/", s.handleDeleteBackup)
	mux.HandleFunc("/api/terraform/resources", s.handleTerraformResources)
	mux.HandleFunc("/api/namespaces", s.handleNamespaces)

	// Redis Routes
	mux.HandleFunc("/api/redis/keys", s.handleRedisKeys)
	mux.HandleFunc("/api/redis/get/", s.handleRedisGet)
	mux.HandleFunc("/api/redis/set", s.handleRedisSet)
	mux.HandleFunc("/api/redis/del/", s.handleRedisDel)
	mux.HandleFunc("/api/redis/flush", s.handleRedisFlush)
	mux.HandleFunc("/api/redis/dbsize", s.handleRedisDBSize)

	// Agent API Routes
	mux.HandleFunc("/api/agent/chat", s.handleAgentChat)
	mux.HandleFunc("/api/agent/status", s.handleAgentStatus)
	mux.HandleFunc("/api/inventory", s.handleInventory)
	mux.HandleFunc("/api/fulfillment/route", s.handleFulfillmentRoute)
	mux.HandleFunc("/api/orders", s.handleOrders)
	mux.HandleFunc("/api/orders/stats", s.handleOrderStats)
	mux.HandleFunc("/api/agent/activity", s.handleAgentActivity)

	// Static files
	// dist folder is embedded as "dist", but we want to serve the content of "dist" at root
	distFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		return fmt.Errorf("failed to load static files: %w", err)
	}
	fileServer := http.FileServer(http.FS(distFS))

	// Serve index.html for unknown routes (SPA fallback)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API routes are handled separately (if we add them to mux later)
		// But if we use the same mux, we need to be careful.
		// For now, API routes will be added before this catch-all if we use patterns.
		// But "/" matches everything in ServeMux unless a longer pattern matches.

		// Check if it's an API request (should have been handled, but just in case)
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			http.NotFound(w, r)
			return
		}

		// Check if file exists in FS
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:] // strip leading /
		}

		f, err := distFS.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for client-side routing
		index, err := distFS.Open("index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer index.Close()
		stat, _ := index.Stat()

		if seeker, ok := index.(io.ReadSeeker); ok {
			http.ServeContent(w, r, "index.html", stat.ModTime(), seeker)
		} else {
			// This should not happen with embed.FS
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		s.logger.Info("Starting UI server on http://localhost:%d", s.port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server failed: %v", err)
		}
	}()

	// Wait for context cancellation or signal
	<-ctx.Done()
	s.logger.Info("Shutting down UI server...")

	if s.tunnel != nil {
		s.tunnel.Stop()
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// getControlPlaneInfo retrieves CP name and zone from Terraform state.
// Works with both local and remote (GCS) state backends.
func (s *Server) getControlPlaneInfo() (string, string, error) {
	tfDir := s.config.GetTerraformDir()

	// Check if terraform is initialized (has .terraform directory)
	dotTF := filepath.Join(tfDir, ".terraform")
	if _, err := os.Stat(dotTF); os.IsNotExist(err) {
		return "", "", fmt.Errorf("terraform not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tfClient, err := terraform.NewClient(ctx, tfDir, s.logger)
	if err != nil {
		return "", "", err
	}

	outputs, err := tfClient.Outputs(ctx)
	if err != nil {
		return "", "", err
	}

	cpName, ok := outputs["control_plane_name"].(string)
	if !ok {
		return "", "", fmt.Errorf("control_plane_name not found in outputs")
	}

	cpZone, ok := outputs["control_plane_zone"].(string)
	if !ok {
		return "", "", fmt.Errorf("control_plane_zone not found in outputs")
	}

	return cpName, cpZone, nil
}

// statusPoller refreshes the cached status every 10 seconds in the background.
// This way /api/status returns instantly with the latest cached result.
func (s *Server) statusPoller(ctx context.Context) {
	// Initial poll immediately
	s.refreshStatus()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshStatus()
		}
	}
}

// refreshStatus computes the current status and caches it.
func (s *Server) refreshStatus() {
	status := GlobalStatus{
		Infra:   "Not Created",
		K8s:     "Not Ready",
		Tools:   "Not Installed",
		Apps:    "Not Deployed",
		Tunnel:  string(TunnelStatusIdle),
		Version: "0.1.0",
	}

	if s.infraReady {
		status.Infra = "Running"
	}

	if s.tunnel != nil {
		status.Tunnel = string(s.tunnel.GetStatus())
	}

	if status.Tunnel == string(TunnelStatusConnected) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Lazily initialise the k8s client on first successful tunnel connection.
		// This avoids spawning kubectl subprocesses every 10 seconds for status polling.
		if s.k8sClient == nil {
			if kc, err := k8s.NewClient(s.config.GetKubeconfigPath(), s.logger); err == nil {
				s.k8sClient = kc
			}
		}

		if s.k8sClient != nil {
			if _, err := s.k8sClient.GetNodes(ctx); err == nil {
				status.K8s = "Ready"
				if s.k8sClient.HasNamespace(ctx, "velero") {
					status.Tools = "Installed"
				}
				if s.k8sClient.HasNamespace(ctx, "application") {
					status.Apps = "Deployed"
				}
			}
		}
	}

	s.cachedStatusMu.Lock()
	s.cachedStatus = &status
	s.cachedStatusMu.Unlock()
}
