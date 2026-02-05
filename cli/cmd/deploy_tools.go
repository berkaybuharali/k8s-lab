// cli/cmd/deploy_tools.go
// Package cmd implements the deploy-tools command which installs platform tools.
//
// This command orchestrates:
// - CSI driver (cloud-specific persistent storage)
// - StorageClass (cloud-specific storage configuration)
// - Velero (backup/restore)
//
// Equivalent to bash: make deploy-tools gcp
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/config"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/terraform"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/velero"
)

func init() {
	rootCmd.AddCommand(deployToolsCmd)
}

// deployToolsCmd represents the deploy-tools command.
var deployToolsCmd = &cobra.Command{
	Use:   "deploy-tools",
	Short: "Deploy platform tools (CSI driver, Velero)",
	Long: `Deploy platform tools required for applications.

This command installs:
1. Cloud-specific CSI driver (for persistent storage)
2. StorageClass (for dynamic volume provisioning)
3. Velero (for backup/restore operations)

These tools must be deployed before applications that use persistent storage
or before performing backup/restore operations.

Example:
  k8s-lab deploy-tools --cloud gcp

After deployment, use:
  k8s-lab deploy-applications --cloud gcp   # Deploy apps
  k8s-lab backup --cloud gcp                # Create backup`,
	RunE: runDeployTools,
}

// runDeployTools orchestrates the platform tools deployment.
func runDeployTools(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	printToolsHeader(provider.Name(), log)

	// Check prerequisites
	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	// Get infrastructure info from Terraform
	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	// Create K8s API tunnel (kubeconfig points to localhost:6443)
	// This tunnel must stay alive for all K8s operations
	log.Info("")
	log.Info("Creating tunnel to Kubernetes API...")
	k8sEndpoint, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()

	log.Debug("K8s API accessible at: %s", k8sEndpoint)

	// Get kubeconfig path (created by deploy-infra)
	kubeconfigPath := cfg.GetKubeconfigPath()

	// Phase 1: Install CSI driver
	if err := installCSIDriver(ctx, provider, kubeconfigPath, log); err != nil {
		return err
	}

	// Phase 2: Apply StorageClass
	if err := applyStorageClass(ctx, cfg, provider, kubeconfigPath, log); err != nil {
		return err
	}

	// Phase 3: Install Velero
	if err := installVelero(ctx, cfg, provider, kubeconfigPath, log); err != nil {
		return err
	}

	printToolsSuccess(provider.Name(), log)
	return nil
}

// checkToolsPrerequisites verifies that prerequisites are met.
func checkToolsPrerequisites(cfg *config.Config, log *logger.Logger) error {
	log.Step("Checking prerequisites")

	kubeconfigPath := cfg.GetKubeconfigPath()
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return fmt.Errorf(
			"kubeconfig not found: %s\n\n"+
				"Run 'k8s-lab deploy-infra --cloud <cloud>' first",
			kubeconfigPath,
		)
	}

	log.Info("Prerequisites satisfied")
	return nil
}

// toolsInfraInfo holds minimal infrastructure info needed for tools deployment.
// Only control plane info is needed to create K8s API tunnel.
type toolsInfraInfo struct {
	ProjectID string // Cloud project/account ID
	CPName    string // Control plane instance name
	CPZone    string // Control plane zone
}

// getInfrastructureInfo reads infrastructure info from Terraform outputs.
// This is needed to create the K8s API tunnel.
func getInfrastructureInfo(
	cfg *config.Config,
	provider cloud.Provider,
	log *logger.Logger,
) (*toolsInfraInfo, error) {
	terraformDir := cfg.GetTerraformDir()

	// Get project ID
	projectID, err := provider.GetProjectID(terraformDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}

	// Get Terraform outputs
	tfClient, err := terraform.NewClient(context.Background(), terraformDir, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform client: %w", err)
	}

	outputs, err := tfClient.Outputs(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get terraform outputs: %w", err)
	}

	// Extract control plane info
	cpName, ok := outputs["control_plane_name"].(string)
	if !ok {
		return nil, fmt.Errorf("control_plane_name output not found")
	}

	cpZone, ok := outputs["control_plane_zone"].(string)
	if !ok {
		return nil, fmt.Errorf("control_plane_zone output not found")
	}

	return &toolsInfraInfo{
		ProjectID: projectID,
		CPName:    cpName,
		CPZone:    cpZone,
	}, nil
}

// installCSIDriver installs the cloud-specific CSI driver.
func installCSIDriver(
	ctx context.Context,
	provider cloud.Provider,
	kubeconfigPath string,
	log *logger.Logger,
) error {
	log.Info("")
	return provider.InstallCSIDriver(ctx, kubeconfigPath)
}

// applyStorageClass applies the cloud-specific StorageClass.
func applyStorageClass(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	kubeconfigPath string,
	log *logger.Logger,
) error {
	log.Info("")
	log.Step("Applying StorageClass")

	// StorageClass YAML location: apps/<cloud>/storageclass.yaml
	repoRoot := cfg.GetRepoRoot()
	storageClassPath := filepath.Join(repoRoot, "apps", provider.Name(), "storageclass.yaml")

	// Check if StorageClass exists for this cloud
	if _, err := os.Stat(storageClassPath); os.IsNotExist(err) {
		log.Warn("No StorageClass found for %s", provider.Name())
		return nil
	}

	log.Info("Applying %s StorageClass", provider.Name())

	// Apply using client-go
	if err := applyStorageClassYAML(ctx, kubeconfigPath, storageClassPath); err != nil {
		return fmt.Errorf("failed to apply StorageClass: %w", err)
	}

	log.Info("StorageClass applied")
	return nil
}

// installVelero installs Velero with cloud-specific configuration.
func installVelero(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	kubeconfigPath string,
	log *logger.Logger,
) error {
	log.Info("")

	// Get Velero configuration from provider
	terraformDir := cfg.GetTerraformDir()
	veleroConfigInterface, err := provider.GetVeleroInstallConfig(terraformDir)
	if err != nil {
		return fmt.Errorf("failed to get Velero config: %w", err)
	}

	// Type assert to velero.InstallConfig
	veleroConfig, ok := veleroConfigInterface.(*velero.InstallConfig)
	if !ok {
		return fmt.Errorf("invalid Velero config type from provider")
	}

	// Create Velero client
	veleroClient, err := velero.NewClient(kubeconfigPath, log)
	if err != nil {
		return fmt.Errorf("failed to create Velero client: %w", err)
	}

	// Install Velero
	if err := veleroClient.Install(ctx, veleroConfig); err != nil {
		return fmt.Errorf("Velero installation failed: %w", err)
	}

	return nil
}

// printToolsHeader prints the deployment banner.
func printToolsHeader(cloud string, log *logger.Logger) {
	log.Info("==============================================")
	log.Info("  Kubernetes Lab - Tools Deployment")
	log.Info("  Cloud: %s", cloud)
	log.Info("==============================================")
	log.Info("")
}

// printToolsSuccess prints success message after deployment.
func printToolsSuccess(cloud string, log *logger.Logger) {
	log.Info("")
	log.Info("==============================================")
	log.Info("  Tools deployed successfully!")
	log.Info("==============================================")
	log.Info("")
	log.Info("Next steps:")
	log.Info("  k8s-lab deploy-applications --cloud %s", cloud)
	log.Info("")
	log.Info("Or restore from backup:")
	log.Info("  k8s-lab restore --cloud %s", cloud)
	log.Info("")
}

// applyStorageClassYAML applies a StorageClass YAML file using client-go.
func applyStorageClassYAML(ctx context.Context, kubeconfigPath, yamlPath string) error {
	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Read YAML file
	file, err := os.Open(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Decode YAML
	decoder := yaml.NewYAMLOrJSONDecoder(file, 4096)
	var storageClass storagev1.StorageClass
	if err := decoder.Decode(&storageClass); err != nil {
		return fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Apply StorageClass (create or update)
	_, err = clientset.StorageV1().StorageClasses().Get(ctx, storageClass.Name, metav1.GetOptions{})
	if err != nil {
		// StorageClass doesn't exist, create it
		_, err = clientset.StorageV1().StorageClasses().Create(ctx, &storageClass, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create StorageClass: %w", err)
		}
	} else {
		// StorageClass exists, update it
		_, err = clientset.StorageV1().StorageClasses().Update(ctx, &storageClass, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update StorageClass: %w", err)
		}
	}

	return nil
}
