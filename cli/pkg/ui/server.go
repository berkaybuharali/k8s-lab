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
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/terraform"
)

//go:embed dist
var staticFiles embed.FS

// Server represents the UI HTTP server.
type Server struct {
	port     int
	cloud    string
	logger   *logger.Logger
	provider cloud.Provider
	config   *config.Config
	tunnel   *TunnelManager
	wsHub    *WebSocketHub
	opMu     sync.Mutex
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
		// Try to find control plane name from state
		// We'll use a simplified check: if we can get CP name, infra exists
		if cpName, zone, err := s.getControlPlaneInfo(); err == nil && cpName != "" {
			s.tunnel = NewTunnelManager(projectID, cpName, zone, s.logger)
			s.tunnel.Start(ctx)
		} else {
			s.logger.Info("Infrastructure not found or incomplete, tunnel will remain Idle")
		}
	} else {
		s.logger.Debug("Could not get project ID (normal if terraform.tfvars not set): %v", err)
	}

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
		} else {
			s.handlePodDetail(w, r)
		}
	})
	mux.HandleFunc("/api/pvcs", s.handlePVCs)
	mux.HandleFunc("/api/backups", s.handleBackups)
	mux.HandleFunc("/api/terraform/resources", s.handleTerraformResources)
	mux.HandleFunc("/api/namespaces", s.handleNamespaces)

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
func (s *Server) getControlPlaneInfo() (string, string, error) {
	// This uses the same logic as deploy-infra to extract info
	tfDir := s.config.GetTerraformDir()
	
	// Check if state file exists first to avoid unnecessary terraform calls
	statePath := filepath.Join(tfDir, "terraform.tfstate")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("state file not found")
	}

	// We use context.Background because this is a one-off check on startup
	// and we don't want to block server start indefinitely if terraform hangs
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
