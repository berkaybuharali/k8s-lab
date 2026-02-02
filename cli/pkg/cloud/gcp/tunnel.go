// cli/pkg/cloud/gcp/tunnel.go
// Package gcp implements IAP tunnel support for accessing VMs without external IPs.
//
// Why gcloud CLI instead of Go SDK:
// - Google does not provide a Go SDK for IAP tunneling
// - cloud.google.com/go/iap only handles IAP configuration, NOT tunneling
// - Google's official recommendation is to use gcloud CLI
// This is a standard pattern in Go for cloud operations where SDK coverage is incomplete.
package gcp

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Tunnel represents an active IAP tunnel (gcloud process).
// Forwards localhost:LocalPort to VM's RemotePort.
type Tunnel struct {
	Instance   string
	Zone       string
	ProjectID  string
	LocalPort  int
	RemotePort int
	cmd        *exec.Cmd
}

// StartTunnel creates IAP tunnel via gcloud subprocess.
// Waits 10 seconds for tunnel to be ready.
// Caller must call tunnel.Stop() when done.
func (p *Provider) StartTunnel(ctx context.Context, instance, zone, projectID string,
	remotePort, localPort int) (*Tunnel, error) {

	p.log.Info("Starting IAP tunnel to %s (port %d -> localhost:%d)...", instance, remotePort, localPort)

	// Build gcloud command
	// Equivalent to bash:
	//   gcloud compute start-iap-tunnel instance port \
	//     --local-host-port=localhost:localPort \
	//     --zone=zone --project=projectID
	cmd := exec.CommandContext(ctx,
		"gcloud", "compute", "start-iap-tunnel",
		instance,
		fmt.Sprintf("%d", remotePort),
		"--local-host-port", fmt.Sprintf("localhost:%d", localPort),
		"--zone", zone,
		"--project", projectID,
	)

	// Redirect stdout/stderr to prevent blocking
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start tunnel process in background
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf(
			"failed to start IAP tunnel to %s: %w\n"+
				"Check:\n"+
				"  - gcloud auth status: gcloud auth list\n"+
				"  - Instance exists: gcloud compute instances list --project=%s\n"+
				"  - IAP permissions: roles/iap.tunnelResourceAccessor",
			instance, err, projectID,
		)
	}

	tunnel := &Tunnel{
		Instance:   instance,
		Zone:       zone,
		ProjectID:  projectID,
		LocalPort:  localPort,
		RemotePort: remotePort,
		cmd:        cmd,
	}

	p.log.Info("Waiting for IAP tunnel connection...")

	// Wait for tunnel to be ready
	if err := tunnel.waitReady(ctx); err != nil {
		tunnel.Stop()
		return nil, fmt.Errorf("tunnel to %s failed to become ready: %w", instance, err)
	}

	p.log.Info("IAP tunnel ready: %s", tunnel.Endpoint())

	return tunnel, nil
}

// waitReady waits 10 seconds for tunnel to establish.
func (t *Tunnel) waitReady(ctx context.Context) error {
	select {
	case <-time.After(10 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Endpoint returns "localhost:port".
func (t *Tunnel) Endpoint() string {
	return fmt.Sprintf("localhost:%d", t.LocalPort)
}

// Stop kills the gcloud process. Idempotent.
func (t *Tunnel) Stop() error {
	if t.cmd == nil || t.cmd.Process == nil {
		return nil
	}

	if err := t.cmd.Process.Kill(); err != nil {
		return nil
	}

	_ = t.cmd.Wait()
	return nil
}

// CreateTalosEndpoint creates IAP tunnel to Talos API (port 50000).
// Returns "localhost:50000", cleanup function, error.
func (p *Provider) CreateTalosEndpoint(ctx context.Context, instance, zone, projectID string) (string, func(), error) {
	p.log.Info("Creating Talos API endpoint for %s...", instance)

	tunnel, err := p.StartTunnel(ctx, instance, zone, projectID, 50000, 50000)
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		p.log.Info("Closing tunnel to %s", instance)
		_ = tunnel.Stop()
	}

	return tunnel.Endpoint(), cleanup, nil
}

// CreateK8sEndpoint creates IAP tunnel to Kubernetes API (port 6443).
// Returns "localhost:6443", cleanup function, error.
func (p *Provider) CreateK8sEndpoint(ctx context.Context, instance, zone, projectID string) (string, func(), error) {
	p.log.Info("Creating Kubernetes API endpoint for %s...", instance)

	tunnel, err := p.StartTunnel(ctx, instance, zone, projectID, 6443, 6443)
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		p.log.Info("Closing tunnel to %s", instance)
		_ = tunnel.Stop()
	}

	return tunnel.Endpoint(), cleanup, nil
}
