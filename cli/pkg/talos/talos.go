// cli/pkg/talos/talos.go
// Package talos provides a Go wrapper around Talos operations.
//
// Talos is a modern OS for Kubernetes that's API-driven and immutable.
// This package uses the official Talos Go SDK (github.com/siderolabs/talos/pkg/machinery)
// to perform all operations programmatically without shelling to talosctl binary.
//
// Key SDK packages used:
// - client: Talos API client for bootstrap, apply-config, health checks
// - config/generate: Machine configuration generation
// - config/types: Configuration data structures
//
// Unlike talosctl binary, the Talos Go SDK provides:
// - Type-safe operations with compile-time checks
// - Direct gRPC calls (no subprocess overhead)
// - Structured errors instead of parsing stderr
// - Connection reuse across operations
//
// Architecture:
// - talos.go: Main Client struct and high-level orchestration
// - config.go: Configuration generation using generate package
// - client.go: Talos API client helpers and connection management
package talos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

// Client wraps Talos SDK operations for cluster configuration and bootstrapping.
// Unlike terraform which requires the binary, Talos provides a complete Go SDK.
//
// The Client maintains:
// - log: Logger for user-facing messages
// - configsDir: Directory where generated configs are stored
//
// All methods use the Talos SDK to perform operations that would normally
// require the talosctl binary. This provides better performance, error handling,
// and eliminates the need for users to install talosctl separately.
type Client struct {
	// log is the logger for user-facing messages
	log *logger.Logger

	// configsDir is where generated configs are stored
	// Example: configs/gcp/talos/
	configsDir string
}

// NewClient creates a new Talos client.
//
// Parameters:
//   - ctx: Context for cancellation
//   - configsDir: Absolute path to directory for storing generated configs
//   - log: Logger instance for output
//
// Returns:
//   - *Client: Talos client instance
//   - error: If initialization fails
//
// Example usage:
//
//	talosDir := cfg.GetTalosConfigsDir()
//	client, err := talos.NewClient(ctx, talosDir, log)
//	if err != nil {
//	    return fmt.Errorf("failed to create talos client: %w", err)
//	}
func NewClient(ctx context.Context, configsDir string, log *logger.Logger) (*Client, error) {
	return &Client{
		log:        log,
		configsDir: configsDir,
	}, nil
}

// buildGenerateOptions converts configOptions to generate.Option slice.
func (c *Client) buildGenerateOptions(options *configOptions) []generate.Option {
	genOpts := []generate.Option{}

	// Add additional SANs
	if len(options.additionalSANs) > 0 {
		genOpts = append(genOpts, generate.WithAdditionalSubjectAltNames(options.additionalSANs))
	}

	// Add install disk
	if options.installDisk != "" {
		genOpts = append(genOpts, generate.WithInstallDisk(options.installDisk))
	}

	return genOpts
}

// writeConfigFile writes config bytes to file with logging.
// Creates parent directory if needed.
func (c *Client) writeConfigFile(path string, data []byte, description string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", description, err)
	}

	c.log.Info("%s written to: %s", description, path)
	return nil
}

// GenerateConfigs generates Talos machine configurations using SDK.
//
// This creates three files:
//   - controlplane.yaml: Configuration for control plane nodes
//     (runs etcd, kube-apiserver, controller-manager, scheduler)
//   - worker.yaml: Configuration for worker nodes
//     (runs kubelet only)
//   - talosconfig: Client authentication config
//     (contains endpoints and credentials for Talos API access)
//
// Uses: github.com/siderolabs/talos/pkg/machinery/config/generate
// Replaces: talosctl gen config <cluster> <endpoint> [options]
//
// Parameters:
//   - ctx: Context for cancellation
//   - clusterName: Name of the Kubernetes cluster
//   - endpoint: Kubernetes API endpoint (e.g., "https://10.0.0.1:6443")
//   - opts: Optional configuration options (patches, SANs, etc.)
//
// Returns:
//   - error: If config generation fails
//
// Example:
//
//	err := client.GenerateConfigs(ctx,
//	    "my-cluster",
//	    "https://10.0.0.1:6443",
//	    talos.WithAdditionalSANs([]string{"localhost"}),
//	)
//
// Implementation: Step 4c
func (c *Client) GenerateConfigs(ctx context.Context, clusterName, endpoint string, opts ...ConfigOption) error {
	c.log.Info("Generating Talos configurations for cluster: %s", clusterName)

	// Apply functional options
	options := applyOptions(opts...)

	// Build generate options (SANs, install disk)
	genOpts := c.buildGenerateOptions(options)

	// Create generate.Input (contains all configs and secrets)
	input, err := generate.NewInput(clusterName, endpoint, options.kubernetesVersion, genOpts...)
	if err != nil {
		return fmt.Errorf("failed to create config input: %w", err)
	}

	// Generate control plane config
	c.log.Info("Generating control plane configuration...")
	controlPlaneConfig, err := input.Config(machine.TypeControlPlane)
	if err != nil {
		return fmt.Errorf("failed to generate control plane config: %w", err)
	}

	// Generate worker config
	c.log.Info("Generating worker configuration...")
	workerConfig, err := input.Config(machine.TypeWorker)
	if err != nil {
		return fmt.Errorf("failed to generate worker config: %w", err)
	}

	// Get talosconfig (client auth config)
	talosconfig, err := input.Talosconfig()
	if err != nil {
		return fmt.Errorf("failed to generate talosconfig: %w", err)
	}

	// Write all configs to disk
	if err := c.writeGeneratedConfigs(controlPlaneConfig, workerConfig, talosconfig); err != nil {
		return err
	}

	c.log.Info("Configuration generation complete")
	return nil
}

// writeGeneratedConfigs writes all three config files to disk.
func (c *Client) writeGeneratedConfigs(cpConfig, workerConfig config.Provider, talosconfig *clientconfig.Config) error {
	// Marshal configs to bytes
	cpBytes, err := cpConfig.Bytes()
	if err != nil {
		return fmt.Errorf("failed to marshal control plane config: %w", err)
	}

	workerBytes, err := workerConfig.Bytes()
	if err != nil {
		return fmt.Errorf("failed to marshal worker config: %w", err)
	}

	talosconfigBytes, err := talosconfig.Bytes()
	if err != nil {
		return fmt.Errorf("failed to marshal talosconfig: %w", err)
	}

	// Write files
	cpPath := filepath.Join(c.configsDir, "controlplane.yaml")
	if err := c.writeConfigFile(cpPath, cpBytes, "Control plane config"); err != nil {
		return err
	}

	workerPath := filepath.Join(c.configsDir, "worker.yaml")
	if err := c.writeConfigFile(workerPath, workerBytes, "Worker config"); err != nil {
		return err
	}

	talosconfigPath := filepath.Join(c.configsDir, "talosconfig")
	if err := c.writeConfigFile(talosconfigPath, talosconfigBytes, "Talosconfig"); err != nil {
		return err
	}

	return nil
}

// ApplyConfig applies configuration to a Talos node using SDK client.
//
// After receiving config, Talos will:
// 1. Configure networking
// 2. Start containerd and kubelet
// 3. Wait for bootstrap (control plane) or join cluster (workers)
//
// Uses: client.ApplyConfiguration()
// Replaces: talosctl apply-config --insecure --nodes <endpoint> --file <config>
//
// Parameters:
//   - ctx: Context for cancellation
//   - endpoint: Talos API endpoint (e.g., "localhost:50000" via IAP tunnel)
//   - configData: Configuration YAML bytes
//   - insecure: Use insecure connection (true for initial apply, false after)
//
// Returns:
//   - error: If config application fails
//
// Note: For GCP, endpoint is typically a tunnel endpoint (localhost:50000)
// provided by the cloud provider's StartTunnel() method.
//
// Example:
//
//	configData, _ := os.ReadFile("controlplane.yaml")
//	err := client.ApplyConfig(ctx, "localhost:50000", configData, true)
//
// Implementation: Step 4d
func (c *Client) ApplyConfig(ctx context.Context, endpoint string, configData []byte, insecure bool) error {
	c.log.Info("Applying Talos configuration to %s...", endpoint)

	// Create insecure client (node has no certs yet)
	talosClient, err := c.createInsecureClient(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to create Talos client: %w", err)
	}
	defer talosClient.Close()

	c.log.Info("Sending configuration to node...")

	// Apply configuration with reboot
	// In maintenance mode (initial config), the node MUST reboot to apply config
	// After reboot, the node will start with the applied configuration
	req := &machineapi.ApplyConfigurationRequest{
		Data: configData,
		Mode: machineapi.ApplyConfigurationRequest_REBOOT,
	}

	_, err = talosClient.ApplyConfiguration(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to apply configuration: %w", err)
	}

	c.log.Info("Configuration applied successfully to %s", endpoint)
	return nil
}

// Bootstrap initializes etcd and starts Kubernetes control plane using SDK.
//
// What bootstrap does:
// 1. Initializes etcd (distributed key-value store for Kubernetes)
// 2. Generates cluster PKI (certificates for secure communication)
// 3. Starts kube-apiserver, controller-manager, scheduler
// 4. Workers automatically join via kubelet once API is available
//
// IMPORTANT: Run only ONCE on ONE control plane node.
// Running again will corrupt the cluster state.
//
// Uses: client.Bootstrap()
// Replaces: talosctl bootstrap --nodes <endpoint> --endpoints <endpoint>
//
// Parameters:
//   - ctx: Context for cancellation
//   - endpoint: Talos API endpoint (e.g., "localhost:50000" via tunnel)
//   - talosconfig: Path to talosconfig file (for authenticated access)
//
// Returns:
//   - error: If bootstrap fails
//
// Example:
//
//	err := client.Bootstrap(ctx, "localhost:50000", "configs/gcp/talos/talosconfig")
//
// Implementation: Step 4e
func (c *Client) Bootstrap(ctx context.Context, endpoint, talosconfig string) error {
	c.log.Step("Bootstrapping Kubernetes cluster...")
	c.log.Info("Endpoint: %s", endpoint)

	// Create authenticated client (uses talosconfig credentials)
	// The endpoint parameter is passed explicitly to override talosconfig endpoints
	// This is needed for cloud environments where we connect via tunnel (localhost)
	talosClient, err := c.createAuthenticatedClient(ctx, endpoint, talosconfig)
	if err != nil {
		return fmt.Errorf("failed to create Talos client: %w", err)
	}
	defer talosClient.Close()

	// Step 1: Wait for Talos API to be ready
	// After config was applied, node reboots from maintenance -> running mode
	// This can take 1-3 minutes
	c.log.Info("Waiting for node to finish rebooting...")
	if err := c.waitForAPIReady(ctx, talosClient); err != nil {
		return fmt.Errorf("Talos API not ready: %w", err)
	}

	// Step 2: Initiate bootstrap
	// This tells Talos to:
	// - Initialize etcd cluster (single-node etcd for now)
	// - Generate cluster certificates
	// - Start Kubernetes control plane components
	c.log.Info("Initiating cluster bootstrap (this starts etcd and Kubernetes)...")

	err = talosClient.Bootstrap(ctx, &machineapi.BootstrapRequest{})
	if err != nil {
		return fmt.Errorf("bootstrap request failed: %w", err)
	}

	c.log.Info("Bootstrap initiated")

	// Step 3: Wait for cluster to become healthy
	// Bootstrap happens asynchronously - we need to poll until healthy
	// This checks: etcd, kubelet, apiserver, controller-manager, scheduler
	c.log.Info("Waiting for cluster components to become healthy (this may take several minutes)...")

	if err := c.waitForClusterHealth(ctx, talosClient); err != nil {
		return fmt.Errorf("cluster health check failed: %w", err)
	}

	c.log.Info("Cluster bootstrapped successfully!")
	return nil
}

// FetchKubeconfig retrieves kubeconfig from cluster using SDK.
//
// The kubeconfig contains:
// - Cluster CA certificate
// - Admin client certificate
// - API server endpoint (modified to use k8sServerURL)
//
// Uses: client.Kubeconfig()
// Replaces: talosctl kubeconfig <output> --nodes <endpoint> --endpoints <endpoint>
//
// Parameters:
//   - ctx: Context for cancellation
//   - endpoint: Talos API endpoint (e.g., "localhost:50000" from CreateTalosEndpoint)
//   - talosconfig: Path to talosconfig file (for authentication)
//   - outputPath: Where to save the kubeconfig file
//   - k8sServerURL: Kubernetes API server URL (from CreateK8sEndpoint, e.g., "https://localhost:6443")
//
// Returns:
//   - error: If kubeconfig fetch fails
//
// Example (in command code):
//
//	k8sEndpoint, cleanup, _ := provider.CreateK8sEndpoint(ctx, cpInstance, zone, projectID)
//	defer cleanup()
//	k8sServerURL := "https://" + k8sEndpoint  // https://localhost:6443
//	err := talosClient.FetchKubeconfig(ctx, talosEndpoint, talosconfig, kubeconfigPath, k8sServerURL)
func (c *Client) FetchKubeconfig(ctx context.Context, endpoint, talosconfig, outputPath, k8sServerURL string) error {
	c.log.Step("Fetching kubeconfig from cluster...")
	c.log.Info("Endpoint: %s", endpoint)

	// Create authenticated Talos client
	talosClient, err := c.createAuthenticatedClient(ctx, endpoint, talosconfig)
	if err != nil {
		return fmt.Errorf("failed to create Talos client: %w", err)
	}
	defer talosClient.Close()

	// Fetch kubeconfig from cluster
	// This retrieves admin credentials and cluster configuration
	c.log.Info("Requesting kubeconfig from Talos API...")

	kubeconfigBytes, err := talosClient.Kubeconfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch kubeconfig: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write kubeconfig to file with restrictive permissions (0600)
	// This is critical for security - kubeconfig contains admin credentials
	if err := os.WriteFile(outputPath, kubeconfigBytes, 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	c.log.Info("Kubeconfig saved: %s", outputPath)

	// Modify kubeconfig server URL to use the endpoint from CreateK8sEndpoint
	// Talos generates kubeconfig with internal cluster IP, but we access via cloud provider's endpoint
	// (tunnel, VPN, direct access, etc. - cloud provider handles the details)
	c.log.Info("Updating kubeconfig server URL to: %s", k8sServerURL)

	if err := c.modifyKubeconfigServer(outputPath, k8sServerURL); err != nil {
		return fmt.Errorf("failed to modify kubeconfig: %w", err)
	}

	c.log.Info("Kubeconfig ready for use")
	return nil
}

// modifyKubeconfigServer updates the server URL in kubeconfig.
//
// Why this is needed:
// - Talos generates kubeconfig with server pointing to internal cluster IP (e.g., https://10.0.0.1:6443)
// - Cloud provider gives us the correct endpoint to use (via CreateK8sEndpoint)
// - Different clouds use different approaches:
//   - GCP: IAP tunnel → https://localhost:6443
//   - AWS: SSM tunnel → https://localhost:6443
//   - Direct access clouds: https://external-ip:6443
// - This method updates kubeconfig to use the cloud provider's endpoint
//
// Example transformation:
//
//	Before: server: https://10.0.0.1:6443
//	After:  server: https://localhost:6443 (or whatever cloud provider returns)
//
// Equivalent to bash: sed -i.bak "s|server: https://${CP_IP}:6443|server: https://localhost:6443|g"
func (c *Client) modifyKubeconfigServer(kubeconfigPath, newServerURL string) error {
	// Read kubeconfig
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	// Simple string replacement in YAML
	// Look for "server: https://<anything>:6443" and replace with newServerURL
	// This is simpler than parsing YAML and handles all edge cases bash sed handles
	content := string(data)

	// Find the server line
	lines := strings.Split(content, "\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "server:") {
			oldURL := strings.TrimSpace(strings.TrimPrefix(trimmed, "server:"))
			c.log.Debug("Replacing server URL: %s -> %s", oldURL, newServerURL)

			// Replace the entire line with correct indentation
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			lines[i] = indent + "server: " + newServerURL
			modified = true
			break // Typically only one cluster in kubeconfig
		}
	}

	if !modified {
		return fmt.Errorf("could not find server URL in kubeconfig")
	}

	// Write back
	modifiedContent := strings.Join(lines, "\n")
	if err := os.WriteFile(kubeconfigPath, []byte(modifiedContent), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	c.log.Debug("Kubeconfig server URL updated successfully")
	return nil
}
