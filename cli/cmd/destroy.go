// cli/cmd/destroy.go
// Package cmd implements the destroy command which tears down all infrastructure.
//
// This command orchestrates:
// - Terraform (infrastructure destruction)
// - Config cleanup (remove generated files)
//
// Equivalent to bash: make destroy gcp
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/config"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/terraform"
)

func init() {
	rootCmd.AddCommand(destroyCmd)
}

// destroyCmd represents the destroy command.
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy infrastructure and clean up configs",
	Long: `Destroy all cloud infrastructure and remove generated configs.

This command performs a complete teardown:
1. Destroys cloud infrastructure (VMs, networks, firewalls) via Terraform
2. Removes generated configuration files (talosconfig, kubeconfig)

WARNING: This is a destructive operation that cannot be undone!
All infrastructure will be deleted and cannot be recovered.

Example:
  k8s-lab destroy --cloud gcp

After destruction:
  k8s-lab deploy-infra --cloud gcp  # Deploy fresh cluster`,
	RunE: runDestroy,
}

// runDestroy orchestrates the full infrastructure destruction.
func runDestroy(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	printDestructionHeader(provider.Name(), log)

	// Phase 1: Terraform destruction
	if err := destroyInfrastructure(ctx, cfg, provider, log); err != nil {
		return err
	}

	// Phase 2: Config cleanup
	if err := cleanupConfigs(cfg, log); err != nil {
		return err
	}

	printDestructionSuccess(log)
	return nil
}

// destroyInfrastructure destroys cloud infrastructure via Terraform.
func destroyInfrastructure(
	ctx context.Context,
	cfg *config.Config,
	provider cloud.Provider,
	log *logger.Logger,
) error {
	tfDir := cfg.GetTerraformDir()

	// Read project ID
	projectID, err := provider.GetProjectID(tfDir)
	if err != nil {
		return fmt.Errorf("failed to get project ID: %w", err)
	}
	log.Info("Project ID: %s", projectID)

	// Check if state exists
	tfClient, err := terraform.NewClient(ctx, tfDir, log)
	if err != nil {
		return fmt.Errorf("failed to create terraform client: %w", err)
	}

	// Initialize terraform (in case backend changed)
	if err := tfClient.Init(ctx); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// Destroy all resources
	log.Info("")
	log.Step("Destroying infrastructure...")
	if err := tfClient.Destroy(ctx); err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	return nil
}

// cleanupConfigs removes generated configuration files.
func cleanupConfigs(cfg *config.Config, log *logger.Logger) error {
	log.Info("")
	log.Step("Cleaning up generated configs...")

	talosConfigsDir := cfg.GetTalosConfigsDir()

	// Check if configs directory exists
	if _, err := os.Stat(talosConfigsDir); os.IsNotExist(err) {
		log.Info("No configs directory found - nothing to clean up")
		return nil
	}

	// Remove configs directory
	if err := os.RemoveAll(talosConfigsDir); err != nil {
		return fmt.Errorf("failed to remove configs directory: %w", err)
	}

	log.Info("Removed: %s", talosConfigsDir)
	return nil
}

// printDestructionHeader prints the destruction banner.
func printDestructionHeader(cloud string, log *logger.Logger) {
	log.Info("============================================")
	log.Info("  Kubernetes Lab - Cluster Destruction")
	log.Info("  Cloud: %s", cloud)
	log.Info("============================================")
	log.Info("")
	log.Info("WARNING: This will destroy all infrastructure!")
	log.Info("")
}

// printDestructionSuccess prints success message after destruction.
func printDestructionSuccess(log *logger.Logger) {
	log.Info("")
	log.Info("==============================================")
	log.Info("  Cluster destroyed")
	log.Info("==============================================")
	log.Info("")
	log.Info("All infrastructure has been removed.")
	log.Info("")
	log.Info("To deploy a new cluster:")
	log.Info("  k8s-lab deploy-infra --cloud gcp")
	log.Info("")
}
