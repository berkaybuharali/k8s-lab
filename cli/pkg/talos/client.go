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
//	c, err := createInsecureClient(ctx, "localhost:50000")
//	if err != nil {
//	    return fmt.Errorf("failed to create client: %w", err)
//	}
//	defer c.Close()
func createInsecureClient(ctx context.Context, endpoint string) (*client.Client, error) {
	// Create client with insecure credentials
	// This is equivalent to: talosctl --insecure
	c, err := client.New(ctx,
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

	return c, nil
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
//	c, err := createAuthenticatedClient(ctx, "localhost:50000", "configs/gcp/talos/talosconfig")
//	if err != nil {
//	    return fmt.Errorf("failed to create client: %w", err)
//	}
//	defer c.Close()
func createAuthenticatedClient(ctx context.Context, endpoint, talosconfigPath string) (*client.Client, error) {
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
	c, err := client.New(ctx,
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

	return c, nil
}

// waitForAPIReady waits for the Talos API to be ready.
//
// After config is applied, the node reboots into running mode.
// This function polls the API until it responds successfully.
//
// Parameters:
//   - ctx: Context with timeout (caller should set timeout)
//   - c: Authenticated Talos client
//
// Returns:
//   - error: If API doesn't become ready or context times out
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
//	defer cancel()
//	if err := waitForAPIReady(ctx, client); err != nil {
//	    return fmt.Errorf("API not ready: %w", err)
//	}
//
// Implementation: Step 4e (needed for bootstrap)
func waitForAPIReady(ctx context.Context, c *client.Client) error {
	// TODO: Implement in 4e
	// Poll client.Version() until it succeeds
	// Equivalent to bash: talosctl version (polls until success)
	return fmt.Errorf("not implemented yet")
}
