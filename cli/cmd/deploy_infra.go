// cli/cmd/deploy_infra.go
// Package cmd implements the deploy-infra command which creates infrastructure
// and bootstraps a Kubernetes cluster.
//
// This command orchestrates:
// - Terraform (infrastructure provisioning)
// - Talos (OS configuration and cluster bootstrap)
// - Kubernetes (cluster verification)
//
// Equivalent to bash: make deploy-infra gcp
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/config"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/talos"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/terraform"
)

func init() {
	rootCmd.AddCommand(deployInfraCmd)
}

// deployInfraCmd represents the deploy-infra command.
var deployInfraCmd = &cobra.Command{
	Use:   "deploy-infra",
	Short: "Deploy infrastructure and bootstrap Kubernetes cluster",
	Long: `Deploy infrastructure and bootstrap a Kubernetes cluster.

This command performs a complete deployment:
1. Creates cloud infrastructure (VPC, firewall, VMs) via Terraform
2. Generates and applies Talos machine configurations
3. Bootstraps Kubernetes cluster
4. Verifies cluster is ready

The cluster will be ready for platform tools and workload deployment.

Example:
  k8s-lab deploy-infra --cloud gcp

After deployment, use:
  k8s-lab deploy-tools --cloud gcp     # Install CSI driver, Velero
  k8s-lab deploy-apps --cloud gcp      # Deploy applications`,
	RunE: runDeployInfra,
}

// InfrastructureInfo holds information about deployed infrastructure.
// This is extracted from Terraform outputs and used throughout deployment.
type InfrastructureInfo struct {
	ProjectID   string   // Cloud project/account ID
	CPName      string   // Control plane instance name
	CPZone      string   // Control plane zone
	CPIP        string   // Control plane internal IP (for cluster endpoint)
	WorkerNames []string // Worker instance names
	WorkerZones []string // Worker zones
}

// runDeployInfra orchestrates the full infrastructure deployment.
// This is the main entry point that coordinates all deployment phases.
// getConfigPatches returns cloud-specific Talos config patches.
// These patches are applied during config generation to customize
// the Talos configuration for specific cloud providers.
func getConfigPatches(cfg *config.Config, provider cloud.Provider) []string {
	repoRoot := cfg.GetRepoRoot()

	switch provider.Name() {
	case "gcp":
		return []string{
			filepath.Join(repoRoot, "infra/gcp/talos-patches/csi.yaml"),
			// Artifact Registry auth handled by gcr-credential-sync DaemonSet
		}
	default:
		return nil
	}
}

func runDeployInfra(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	printDeploymentHeader(provider.Name(), log)

	// Phase 1: Infrastructure provisioning
	infra, err := provisionInfrastructure(ctx, cfg, provider, log)
	if err != nil {
		return err
	}

	// Phase 2: Talos cluster configuration
	if err := configureTalosCluster(ctx, cfg, provider, infra, log); err != nil {
		return err
	}

	// Phase 3: Kubernetes bootstrap
	kubeconfigPath, err := bootstrapKubernetes(ctx, cfg, provider, infra, log)
	if err != nil {
		return err
	}

	// Phase 4: Cluster verification
	if err := verifyClusterReady(ctx, cfg, provider, kubeconfigPath, infra, log); err != nil {
		return err
	}

	printDeploymentSuccess(provider.Name(), log)
	return nil
}

// provisionInfrastructure creates cloud infrastructure via Terraform.
// Returns infrastructure details needed for subsequent steps.
func provisionInfrastructure(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	log *logger.Logger,
) (*InfrastructureInfo, error) {
	tfDir := cfg.GetTerraformDir()

	// Read project ID
	projectID, err := provider.GetProjectID(tfDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}
	log.Info("Project ID: %s", projectID)

	// Ensure state bucket exists
	if err := ensureStateBucket(ctx, tfDir, projectID, provider); err != nil {
		return nil, err
	}

	// Run Terraform
	tfClient, err := terraform.NewClient(ctx, tfDir, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform client: %w", err)
	}

	if err := tfClient.Init(ctx); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w", err)
	}

	if err := tfClient.Apply(ctx); err != nil {
		return nil, fmt.Errorf("terraform apply failed: %w", err)
	}

	// Extract infrastructure info
	infra, err := extractInfrastructureInfo(ctx, tfClient, projectID, log)
	if err != nil {
		return nil, err
	}

	return infra, nil
}

// ensureStateBucket creates Terraform state bucket if it doesn't exist.
func ensureStateBucket(
	ctx context.Context,
	tfDir string,
	projectID string,
	provider cloud.Provider,
) error {
	bucketName, err := readTerraformVariable(tfDir, "state_bucket")
	if err != nil {
		return fmt.Errorf("failed to read state_bucket from terraform.tfvars: %w", err)
	}

	if err := provider.EnsureStateBucket(ctx, bucketName, projectID); err != nil {
		return fmt.Errorf("failed to ensure state bucket: %w", err)
	}

	return nil
}

// extractInfrastructureInfo reads Terraform outputs and builds InfrastructureInfo.
func extractInfrastructureInfo(
	ctx context.Context,
	tfClient *terraform.Client,
	projectID string,
	log *logger.Logger,
) (*InfrastructureInfo, error) {
	outputs, err := tfClient.Outputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get terraform outputs: %w", err)
	}

	// Extract control plane info
	cpName := outputs["control_plane_name"].(string)
	cpZone := outputs["control_plane_zone"].(string)
	cpIP := outputs["control_plane_ip"].(string)

	// Extract worker info
	workerNamesRaw := outputs["worker_names"].([]interface{})
	workerZonesRaw := outputs["worker_zones"].([]interface{})

	workerNames := make([]string, len(workerNamesRaw))
	workerZones := make([]string, len(workerZonesRaw))
	for i := range workerNamesRaw {
		workerNames[i] = workerNamesRaw[i].(string)
		workerZones[i] = workerZonesRaw[i].(string)
	}

	infra := &InfrastructureInfo{
		ProjectID:   projectID,
		CPName:      cpName,
		CPZone:      cpZone,
		CPIP:        cpIP,
		WorkerNames: workerNames,
		WorkerZones: workerZones,
	}

	logInfrastructureInfo(infra, log)
	return infra, nil
}

// logInfrastructureInfo prints deployed infrastructure details.
func logInfrastructureInfo(infra *InfrastructureInfo, log *logger.Logger) {
	log.Info("Infrastructure created:")
	log.Info("  Control plane: %s (%s) in %s", infra.CPName, infra.CPIP, infra.CPZone)
	for i, name := range infra.WorkerNames {
		log.Info("  Worker %d: %s in %s", i, name, infra.WorkerZones[i])
	}
}

// configureTalosCluster generates and applies Talos configurations to all nodes.
func configureTalosCluster(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	infra *InfrastructureInfo,
	log *logger.Logger,
) error {
	// Wait for VMs to boot into Talos maintenance mode
	waitForVMBoot(log)

	talosConfigsDir := cfg.GetTalosConfigsDir()
	talosClient, err := talos.NewClient(ctx, talosConfigsDir, log)
	if err != nil {
		return fmt.Errorf("failed to create talos client: %w", err)
	}

	// Generate configs with cloud-specific patches
	// IMPORTANT: Cluster endpoint uses INTERNAL IP (cpIP) - this is how nodes find each other
	// Our access to Talos/K8s APIs uses tunnels from CreateTalosEndpoint/CreateK8sEndpoint
	clusterEndpoint := fmt.Sprintf("https://%s:6443", infra.CPIP)

	// Get cloud-specific patches
	patches := getConfigPatches(cfg, provider)

	if err := talosClient.GenerateConfigs(ctx, cfg.ClusterName, clusterEndpoint,
		talos.WithConfigPatches(patches),
	); err != nil {
		return fmt.Errorf("failed to generate talos configs: %w", err)
	}

	// Apply configs to all nodes
	if err := applyConfigToControlPlane(ctx, cfg, provider, talosClient, infra, log); err != nil {
		return err
	}

	if err := applyConfigsToWorkers(ctx, cfg, provider, talosClient, infra, log); err != nil {
		return err
	}

	log.Info("All nodes configured")

	// Wait for nodes to reboot after config apply
	// Patches are already applied during config generation
	waitForNodeReboot(log)

	return nil
}

// waitForVMBoot waits for VMs to boot into Talos maintenance mode.
func waitForVMBoot(log *logger.Logger) {
	log.Info("")
	log.Info("Waiting for VMs to boot into Talos maintenance mode (3 minutes)...")
	time.Sleep(3 * time.Minute)
}

// waitForNodeReboot waits for nodes to reboot after config application.
func waitForNodeReboot(log *logger.Logger) {
	log.Info("")
	log.Info("Waiting for nodes to come back after config apply (1 minute)...")
	time.Sleep(1 * time.Minute)
}

// applyConfigToControlPlane applies Talos configuration to control plane.
// Creates tunnel via CreateTalosEndpoint, applies config, closes tunnel.
func applyConfigToControlPlane(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	talosClient *talos.Client,
	infra *InfrastructureInfo,
	log *logger.Logger,
) error {
	log.Info("Applying config to control plane (%s)...", infra.CPName)

	configPath := filepath.Join(cfg.GetTalosConfigsDir(), "controlplane.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read controlplane config: %w", err)
	}

	// Create Talos API tunnel
	endpoint, cleanup, err := provider.CreateTalosEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create Talos endpoint for CP: %w", err)
	}
	defer cleanup()

	// Apply config (insecure - node doesn't trust us yet)
	if err := talosClient.ApplyConfig(ctx, endpoint, configData, true); err != nil {
		return fmt.Errorf("failed to apply config to CP: %w", err)
	}

	log.Info("Control plane configured successfully")
	return nil
}

// applyConfigsToWorkers applies Talos configuration to all worker nodes.
// Creates tunnel for each worker, applies config, closes tunnel.
func applyConfigsToWorkers(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	talosClient *talos.Client,
	infra *InfrastructureInfo,
	log *logger.Logger,
) error {
	configPath := filepath.Join(cfg.GetTalosConfigsDir(), "worker.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read worker config: %w", err)
	}

	for i, name := range infra.WorkerNames {
		if err := applyConfigToWorker(ctx, provider, talosClient, name, infra.WorkerZones[i], infra.ProjectID, configData, i, log); err != nil {
			return err
		}
	}

	return nil
}

// applyConfigToWorker applies Talos configuration to a single worker node.
func applyConfigToWorker(
	ctx context.Context,
	provider cloud.Provider,
	talosClient *talos.Client,
	nodeName string,
	zone string,
	projectID string,
	configData []byte,
	workerIndex int,
	log *logger.Logger,
) error {
	log.Info("Applying config to worker-%d (%s)...", workerIndex, nodeName)

	// Create Talos API tunnel
	endpoint, cleanup, err := provider.CreateTalosEndpoint(ctx, nodeName, zone, projectID)
	if err != nil {
		return fmt.Errorf("failed to create Talos endpoint for worker-%d: %w", workerIndex, err)
	}
	defer cleanup()

	// Apply config (insecure - node doesn't trust us yet)
	if err := talosClient.ApplyConfig(ctx, endpoint, configData, true); err != nil {
		return fmt.Errorf("failed to apply config to worker-%d: %w", workerIndex, err)
	}

	log.Info("Worker-%d configured successfully", workerIndex)
	return nil
}

// bootstrapKubernetes bootstraps the Kubernetes cluster and fetches kubeconfig.
// Returns path to kubeconfig file.
func bootstrapKubernetes(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	infra *InfrastructureInfo,
	log *logger.Logger,
) (string, error) {
	talosConfigsDir := cfg.GetTalosConfigsDir()
	talosconfigPath := filepath.Join(talosConfigsDir, "talosconfig")
	kubeconfigPath := filepath.Join(talosConfigsDir, "kubeconfig")

	talosClient, err := talos.NewClient(ctx, talosConfigsDir, log)
	if err != nil {
		return "", fmt.Errorf("failed to create talos client: %w", err)
	}

	// Create Talos API tunnel for bootstrap
	talosEndpoint, cleanupTalos, err := provider.CreateTalosEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to create Talos endpoint: %w", err)
	}
	defer cleanupTalos()

	// Bootstrap cluster
	if err := talosClient.Bootstrap(ctx, talosEndpoint, talosconfigPath); err != nil {
		return "", fmt.Errorf("failed to bootstrap cluster: %w", err)
	}

	// Create K8s API tunnel for kubeconfig fetch
	k8sEndpoint, cleanupK8s, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to create K8s endpoint: %w", err)
	}
	defer cleanupK8s()

	// Fetch kubeconfig and modify server URL to point to tunnel endpoint
	k8sServerURL := fmt.Sprintf("https://%s", k8sEndpoint)
	if err := talosClient.FetchKubeconfig(ctx, talosEndpoint, talosconfigPath, kubeconfigPath, k8sServerURL); err != nil {
		return "", fmt.Errorf("failed to fetch kubeconfig: %w", err)
	}

	return kubeconfigPath, nil
}

// verifyClusterReady verifies all nodes are Ready in the cluster.
// Creates a K8s API tunnel, then verifies nodes are ready.
func verifyClusterReady(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	kubeconfigPath string,
	infra *InfrastructureInfo,
	log *logger.Logger,
) error {
	// Create K8s API tunnel for verification
	log.Info("")
	log.Info("Creating tunnel to verify cluster...")
	k8sEndpoint, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s endpoint: %w", err)
	}
	defer cleanup()

	log.Debug("K8s API accessible at: %s", k8sEndpoint)

	k8sClient, err := k8s.NewClient(kubeconfigPath, log)
	if err != nil {
		return fmt.Errorf("failed to create K8s client: %w", err)
	}

	expectedNodes := 1 + len(infra.WorkerNames)
	if err := k8sClient.WaitForNodesReady(ctx, expectedNodes, 10*time.Minute); err != nil {
		return fmt.Errorf("cluster not ready: %w", err)
	}

	return nil
}

// readTerraformVariable reads a variable value from terraform.tfvars.
// Simple parser that extracts quoted values: var_name = "value"
func readTerraformVariable(tfDir, varName string) (string, error) {
	tfvarsPath := filepath.Join(tfDir, "terraform.tfvars")
	data, err := os.ReadFile(tfvarsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read terraform.tfvars: %w", err)
	}

	lines := splitLines(string(data))
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			// Look for: varName = "value" or varName="value"
			if hasPrefix(line, varName+" = \"") || hasPrefix(line, varName+"=\"") {
				start := indexByte(line, '"')
				if start >= 0 {
					end := indexByte(line[start+1:], '"')
					if end >= 0 {
						return line[start+1 : start+1+end], nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("variable '%s' not found in terraform.tfvars", varName)
}

// String helper functions

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// UI helper functions

func printDeploymentHeader(cloud string, log *logger.Logger) {
	log.Info("============================================")
	log.Info("  Kubernetes Lab - Cluster Deployment")
	log.Info("  Cloud: %s", cloud)
	log.Info("============================================")
	log.Info("")
}

func printDeploymentSuccess(cloud string, log *logger.Logger) {
	log.Info("")
	log.Info("==============================================")
	log.Info("  Cluster is ready!")
	log.Info("==============================================")
	log.Info("")
	log.Info("Next steps:")
	log.Info("")
	log.Info("1. Deploy platform tools (CSI, Velero):")
	log.Info("   k8s-lab deploy-tools --cloud %s", cloud)
	log.Info("")
	log.Info("2. Deploy applications:")
	log.Info("   k8s-lab deploy-applications --cloud %s", cloud)
	log.Info("")
	log.Info("Or use all-in-one:")
	log.Info("   k8s-lab deploy --cloud %s", cloud)
	log.Info("")
	log.Info("To destroy the cluster:")
	log.Info("   k8s-lab destroy --cloud %s", cloud)
	log.Info("")
}
