package ui

import (
	"context"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

// TunnelStatus represents the current state of the IAP tunnel.
type TunnelStatus string

const (
	TunnelStatusIdle         TunnelStatus = "Idle"
	TunnelStatusStarting     TunnelStatus = "Starting"
	TunnelStatusConnected    TunnelStatus = "Connected"
	TunnelStatusReconnecting TunnelStatus = "Reconnecting"
	TunnelStatusDisconnected TunnelStatus = "Disconnected"
)

// TunnelManager handles the persistent IAP tunnel to the Kubernetes API.
type TunnelManager struct {
	projectID string
	instance  string
	zone      string
	log       *logger.Logger

	status     TunnelStatus
	statusMu   sync.RWMutex
	cmd        *exec.Cmd
	cmdMu      sync.Mutex
	cancelFunc context.CancelFunc

	reconnectAttempts int
	attemptsMu        sync.Mutex
	maxAttempts       int
}

// NewTunnelManager creates a new manager instance.
func NewTunnelManager(projectID, instance, zone string, log *logger.Logger) *TunnelManager {
	return &TunnelManager{
		projectID:   projectID,
		instance:    instance,
		zone:        zone,
		log:         log,
		status:      TunnelStatusIdle,
		maxAttempts: 5,
	}
}

// GetStatus returns the current tunnel status.
func (tm *TunnelManager) GetStatus() TunnelStatus {
	tm.statusMu.RLock()
	defer tm.statusMu.RUnlock()
	return tm.status
}

// setStatus updates the tunnel status.
func (tm *TunnelManager) setStatus(status TunnelStatus) {
	tm.statusMu.Lock()
	defer tm.statusMu.Unlock()
	tm.status = status
}

// Start spawns the gcloud tunnel process and starts the health check loop.
func (tm *TunnelManager) Start(ctx context.Context) {
	tm.statusMu.Lock()
	if tm.status == TunnelStatusConnected || tm.status == TunnelStatusStarting {
		tm.statusMu.Unlock()
		return
	}
	tm.status = TunnelStatusStarting
	tm.statusMu.Unlock()

	tm.attemptsMu.Lock()
	tm.reconnectAttempts = 0
	tm.attemptsMu.Unlock()

	tm.log.Info("Starting persistent K8s tunnel manager...")

	runCtx, cancel := context.WithCancel(ctx)
	tm.cancelFunc = cancel

	go tm.runTunnel(runCtx)
	go tm.healthCheckLoop(runCtx)
}

// Stop kills the tunnel process and stops loops.
func (tm *TunnelManager) Stop() {
	tm.log.Info("Stopping K8s tunnel manager...")
	if tm.cancelFunc != nil {
		tm.cancelFunc()
	}
	tm.killProcess()
	tm.setStatus(TunnelStatusIdle)
}

// runTunnel executes the gcloud command.
func (tm *TunnelManager) runTunnel(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			tm.log.Info("Spawning IAP tunnel: gcloud compute start-iap-tunnel %s 6443...", tm.instance)

			cmd := exec.CommandContext(ctx,
				"gcloud", "compute", "start-iap-tunnel",
				tm.instance, "6443",
				"--local-host-port", "localhost:6443",
				"--zone", tm.zone,
				"--project", tm.projectID,
			)

			tm.cmdMu.Lock()
			tm.cmd = cmd
			tm.cmdMu.Unlock()

			if err := cmd.Start(); err != nil {
				tm.log.Error("Failed to start tunnel process: %v", err)
				tm.setStatus(TunnelStatusDisconnected)
				return
			}

			// Wait for process to exit
			err := cmd.Wait()
			if err != nil && ctx.Err() == nil {
				tm.log.Warn("Tunnel process exited with error: %v", err)
			}

			if ctx.Err() != nil {
				return
			}

			// If it exited unexpectedly, wait before retry
			time.Sleep(2 * time.Second)
		}
	}
}

// healthCheckLoop pings localhost:6443 every 30 seconds.
func (tm *TunnelManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial check after 10s wait for startup
	time.Sleep(10 * time.Second)
	tm.checkHealth()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tm.checkHealth()
		}
	}
}

// checkHealth attempts a TCP connection to localhost:6443.
func (tm *TunnelManager) checkHealth() {
	conn, err := net.DialTimeout("tcp", "localhost:6443", 2*time.Second)
	if err != nil {
		status := tm.GetStatus()
		if status == TunnelStatusConnected {
			tm.log.Warn("K8s tunnel health check failed: %v", err)
			tm.setStatus(TunnelStatusReconnecting)

			tm.attemptsMu.Lock()
			tm.reconnectAttempts = 1
			tm.attemptsMu.Unlock()
		} else if status == TunnelStatusReconnecting {
			tm.attemptsMu.Lock()
			tm.reconnectAttempts++
			attempts := tm.reconnectAttempts
			tm.attemptsMu.Unlock()

			if attempts >= tm.maxAttempts {
				tm.log.Error("K8s tunnel failed after %d reconnect attempts", tm.maxAttempts)
				tm.setStatus(TunnelStatusDisconnected)
				tm.killProcess()
			}
		} else if status == TunnelStatusStarting {
			tm.log.Debug("K8s tunnel not ready yet...")
		}
		return
	}
	conn.Close()

	if tm.GetStatus() != TunnelStatusConnected {
		tm.log.Info("K8s tunnel connected and healthy")
		tm.setStatus(TunnelStatusConnected)

		tm.attemptsMu.Lock()
		tm.reconnectAttempts = 0
		tm.attemptsMu.Unlock()
	}
}

// killProcess kills the current gcloud process.
func (tm *TunnelManager) killProcess() {
	tm.cmdMu.Lock()
	defer tm.cmdMu.Unlock()
	if tm.cmd != nil && tm.cmd.Process != nil {
		_ = tm.cmd.Process.Kill()
	}
}
