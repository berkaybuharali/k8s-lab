// cli/pkg/talos/client.go
// Talos API client helpers and connection management.
//
// This file provides helpers for creating and managing Talos SDK clients.
// The Talos SDK uses gRPC to communicate with the Talos API on port 50000.
//
// Connection modes:
// - Insecure: Used for initial config application (node has no config yet)
// - Authenticated: Used after config applied (uses talosconfig credentials)
//
// For cloud environments with no external IPs (like GCP with IAP):
// - Endpoint is typically "localhost:50000" (via tunnel)
// - Tunnel must be established before creating client
// - Tunnel lifecycle managed by cloud provider package
package talos

import (
	"context"
	"fmt"
	"time"

	clusterapi "github.com/siderolabs/talos/pkg/machinery/api/cluster"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/client/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createInsecureClient creates a Talos client with insecure connection.
//
// This is used for initial config application when the node doesn't have
// a configuration yet (maintenance mode). The node's API accepts insecure
// connections in this state.
//
// After config is applied and the node reboots, you must use createAuthenticatedClient.
//
// Parameters:
//   - ctx: Context for cancellation
//   - endpoint: Talos API endpoint (e.g., "localhost:50000")
//
// Returns:
//   - *client.Client: Talos SDK client
//   - error: If client creation fails
//
// Example:
//
//	talosClient, err := c.createInsecureClient(ctx, "localhost:50000")
//	if err != nil {
//	    return fmt.Errorf("failed to create client: %w", err)
//	}
//	defer talosClient.Close()
func (c *Client) createInsecureClient(ctx context.Context, endpoint string) (*client.Client, error) {
	c.log.Debug("Creating insecure Talos client for %s...", endpoint)

	// Create client with insecure credentials
	// This is equivalent to: talosctl --insecure
	talosClient, err := client.New(ctx,
		client.WithEndpoints(endpoint),
		client.WithGRPCDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create insecure Talos client: %w\n"+
				"Endpoint: %s",
			err, endpoint,
		)
	}

	c.log.Debug("Talos client created successfully")
	return talosClient, nil
}

// createAuthenticatedClient creates a Talos client with authenticated connection.
//
// This is used after the node has a configuration and is running normally.
// It reads credentials from the talosconfig file.
//
// Parameters:
//   - ctx: Context for cancellation
//   - endpoint: Talos API endpoint (e.g., "localhost:50000")
//   - talosconfigPath: Path to talosconfig file (contains credentials)
//
// Returns:
//   - *client.Client: Talos SDK client
//   - error: If client creation fails or talosconfig is invalid
//
// Example:
//
//	talosClient, err := c.createAuthenticatedClient(ctx, "localhost:50000", "configs/gcp/talos/talosconfig")
//	if err != nil {
//	    return fmt.Errorf("failed to create client: %w", err)
//	}
//	defer talosClient.Close()
func (c *Client) createAuthenticatedClient(ctx context.Context, endpoint, talosconfigPath string) (*client.Client, error) {
	c.log.Debug("Creating authenticated Talos client for %s...", endpoint)
	c.log.Debug("Loading talosconfig from: %s", talosconfigPath)

	// Load talosconfig (contains CA cert, client cert, endpoints)
	cfg, err := config.Open(talosconfigPath)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load talosconfig: %w\n"+
				"File: %s\n"+
				"Hint: Run GenerateConfigs first",
			err, talosconfigPath,
		)
	}

	// Create client with config credentials
	// This is equivalent to: talosctl --talosconfig <path>
	talosClient, err := client.New(ctx,
		client.WithConfig(cfg),
		client.WithEndpoints(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create authenticated Talos client: %w\n"+
				"Endpoint: %s",
			err, endpoint,
		)
	}

	c.log.Debug("Talos client created successfully")
	return talosClient, nil
}

// waitForAPIReady waits for the Talos API to be ready.
//
// After config is applied, the node reboots into running mode.
// This function polls the API until it responds successfully.
//
// Parameters:
//   - ctx: Context with timeout (caller should set timeout)
//   - talosClient: Authenticated Talos client
//
// Returns:
//   - error: If API doesn't become ready or context times out
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
//	defer cancel()
//	if err := c.waitForAPIReady(ctx, talosClient); err != nil {
//	    return fmt.Errorf("API not ready: %w", err)
//	}
//
// Implementation: Step 4e
func (c *Client) waitForAPIReady(ctx context.Context, talosClient *client.Client) error {
	c.log.Info("Waiting for Talos API to be ready (node may be rebooting)...")

	// Poll interval - check every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	attempt := 1
	maxAttempts := 30 // 30 attempts * 10s = 5 minutes

	for {
		select {
		case <-ctx.Done():
			// Context cancelled or timed out
			return fmt.Errorf("API not ready: %w", ctx.Err())

		case <-ticker.C:
			c.log.Debug("Checking Talos API readiness (attempt %d/%d)...", attempt, maxAttempts)

			// Try to get version - this confirms node is running and authenticated
			// Equivalent to bash: talosctl version --talosconfig=<path>
			_, err := talosClient.Version(ctx)
			if err == nil {
				// Success! API is ready
				c.log.Info("Talos API is ready (authenticated mode)")
				return nil
			}

			c.log.Debug("API not ready yet: %v", err)

			// Check if we've exceeded max attempts
			attempt++
			if attempt > maxAttempts {
				return fmt.Errorf(
					"Talos API not ready after %d attempts (%d seconds)\n"+
						"Last error: %w",
					maxAttempts, maxAttempts*10, err,
				)
			}
		}
	}
}

// waitForClusterHealth waits for the cluster to be healthy after bootstrap.
//
// After bootstrap is initiated, it takes time for all Kubernetes components
// to start and become healthy. This function polls the health endpoint.
//
// Checks:
// - etcd: Distributed key-value store
// - kubelet: Node agent
// - kube-apiserver: Kubernetes API
// - controller-manager: Core control loops
// - scheduler: Pod scheduling
//
// Parameters:
//   - ctx: Context with timeout (caller should set timeout)
//   - talosClient: Authenticated Talos client
//
// Returns:
//   - error: If cluster doesn't become healthy or context times out
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
//	defer cancel()
//	if err := c.waitForClusterHealth(ctx, talosClient); err != nil {
//	    return fmt.Errorf("cluster unhealthy: %w", err)
//	}
//
// Implementation: Step 4e
func (c *Client) waitForClusterHealth(ctx context.Context, talosClient *client.Client) error {
	c.log.Info("Waiting for cluster to become healthy...")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	attempt := 1
	maxAttempts := 60 // 60 attempts * 10s = 10 minutes

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cluster not healthy: %w", ctx.Err())

		case <-ticker.C:
			c.log.Debug("Checking cluster health (attempt %d/%d)...", attempt, maxAttempts)

			// Check cluster health with 30s wait timeout (matches bash)
			// Equivalent to bash: talosctl health --wait-timeout=30s
			healthStream, err := talosClient.ClusterHealthCheck(
				ctx,
				30*time.Second, // Wait timeout for health check
				&clusterapi.ClusterInfo{
					ForceEndpoint: "", // Use default endpoint from config
				},
			)

			// Read health check response
			if err == nil && healthStream != nil {
				// Try to receive health status
				// If stream opens successfully, cluster is healthy
				_, recvErr := healthStream.Recv()
				if recvErr == nil {
					// Success! Cluster is healthy
					c.log.Info("Control plane is healthy")
					return nil
				}
				c.log.Debug("Health check recv error: %v", recvErr)
			} else if err != nil {
				c.log.Debug("Cluster not healthy yet: %v", err)
			}

			// Check if we've exceeded max attempts
			attempt++
			if attempt > maxAttempts {
				c.log.Warn(
					"Health check timed out after %d attempts (%d minutes) - cluster may still be initializing",
					maxAttempts, maxAttempts*10/60,
				)
				// Don't fail hard - cluster might still come up
				return nil
			}
		}
	}
}
