// cli/cmd/deploy_applications.go
// Package cmd implements the deploy-applications command.
//
// This command orchestrates the deployment of standard workloads:
// - NGINX (Stateless web server)
// - Redis (Stateful data store with PVC)
//
// Equivalent to bash: make deploy-applications gcp
package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

func init() {
	rootCmd.AddCommand(deployApplicationsCmd)
}

// deployApplicationsCmd represents the deploy-applications command.
var deployApplicationsCmd = &cobra.Command{
	Use:   "deploy-applications",
	Short: "Deploy applications (NGINX, Redis)",
	Long: `Deploy standard applications to the cluster.

This command deploys:
1. Application Namespace (apps/namespace.yaml)
2. NGINX - Stateless web server (apps/nginx.yaml)
3. Redis - Stateful store with persistence (apps/redis.yaml)

It waits for all deployments to be ready before completing.

Example:
  k8s-lab deploy-applications --cloud gcp`,
	RunE: runDeployApplications,
}

// runDeployApplications orchestrates the application deployment.
func runDeployApplications(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	printAppsHeader(provider.Name(), log)

	// Check tools prerequisites (kubeconfig)
	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	// Get infrastructure info for tunnel
	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	// Create K8s API tunnel
	log.Info("")
	log.Info("Creating tunnel to Kubernetes API...")
	k8sEndpoint, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()

	log.Debug("K8s API accessible at: %s", k8sEndpoint)

	// Create K8s client
	kubeconfigPath := cfg.GetKubeconfigPath()
	k8sClient, err := k8s.NewClient(kubeconfigPath, log)
	if err != nil {
		return fmt.Errorf("failed to create K8s client: %w", err)
	}

	// Verify StorageClass exists (Prerequisite for Redis)
	log.Step("Verifying prerequisites")
	hasSC, err := k8sClient.CheckStorageClassExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check StorageClass: %w", err)
	}
	if !hasSC {
		return fmt.Errorf("no StorageClass found. Did you run 'k8s-lab deploy-tools'?")
	}
	log.Info("StorageClass found")

	// Deploy Applications
	repoRoot := cfg.GetRepoRoot()
	appsDir := filepath.Join(repoRoot, "apps")

	manifests := []struct {
		File string
		Desc string
	}{
		{filepath.Join(appsDir, "namespace.yaml"), "Application Namespace"},
		{filepath.Join(appsDir, "nginx.yaml"), "NGINX"},
		{filepath.Join(appsDir, "redis.yaml"), "Redis"},
	}

	for _, m := range manifests {
		if err := applyManifest(ctx, k8sClient, m.File, m.Desc, log); err != nil {
			return err
		}
	}

	// Wait for Readiness
	log.Info("")
	log.Step("Waiting for applications to be ready")

	// Wait for NGINX
	if err := k8sClient.WaitForDeploymentReady(ctx, ApplicationNamespace, "nginx", 5*time.Minute); err != nil {
		return fmt.Errorf("nginx deployment failed: %w", err)
	}

	// Wait for Redis
	if err := k8sClient.WaitForDeploymentReady(ctx, ApplicationNamespace, "redis", 5*time.Minute); err != nil {
		return fmt.Errorf("redis deployment failed: %w", err)
	}

	printAppsSuccess(log)
	return nil
}

// applyManifest is a helper to apply a manifest with logging.
func applyManifest(ctx context.Context, client *k8s.Client, path, description string, log *logger.Logger) error {
	log.Info("Applying %s...", description)
	if err := client.ApplyManifest(ctx, path); err != nil {
		return fmt.Errorf("failed to apply %s: %w", description, err)
	}
	return nil
}

func printAppsHeader(cloud string, log *logger.Logger) {
	log.Info("==============================================")
	log.Info("  Kubernetes Lab - Application Deployment")
	log.Info("  Cloud: %s", cloud)
	log.Info("==============================================")
	log.Info("")
}

func printAppsSuccess(log *logger.Logger) {
	log.Info("")
	log.Info("==============================================")
	log.Info("  Applications deployed successfully!")
	log.Info("==============================================")
	log.Info("")
	log.Info("Next steps:")
	log.Info("  k8s-lab seed-redis --cloud gcp     # Seed test data")
	log.Info("  k8s-lab backup --cloud gcp         # Create backup")
	log.Info("")
}
