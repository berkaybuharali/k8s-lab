// cli/cmd/restore.go
// Package cmd implements the restore command.
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/velero"
)

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().String("backup", "", "Name of backup to restore from (default: latest)")
	restoreCmd.Flags().Bool("clean", true, "Delete namespace before restore (default: true)")
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from a Velero backup",
	Long: `Restore the application namespace from a Velero backup.

If no backup name is provided, the latest successful backup will be used.

By default (--clean=true), this command DELETES the existing application
namespace before restoring to ensure a clean disaster recovery test.
Use --clean=false to merge with existing resources (Velero default behavior).

Usage:
  k8s-lab restore --cloud gcp
  k8s-lab restore --cloud gcp --backup k8s-lab-backup-04022026-1430
  k8s-lab restore --cloud gcp --clean=false  # Merge restore`,
	RunE: runRestore,
}

func runRestore(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	backupName, _ := cmd.Flags().GetString("backup")
	clean, _ := cmd.Flags().GetBool("clean")

	log.Info("==============================================")
	log.Info("  Kubernetes Lab - Restore")
	log.Info("  Cloud: %s", provider.Name())
	log.Info("==============================================")
	log.Info("")

	// Check prerequisites
	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	// Get infrastructure info
	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	// Create tunnel
	log.Info("")
	log.Info("Creating tunnel to Kubernetes API...")
	k8sEndpoint, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}
	defer cleanup()
	log.Debug("K8s API accessible at: %s", k8sEndpoint)

	// Create Clients
	veleroClient, err := velero.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return fmt.Errorf("failed to create velero client: %w", err)
	}

	k8sClient, err := k8s.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// 1. Verify Velero is ready
	log.Step("Verifying Velero installation")
	if err := veleroClient.WaitForReady(ctx, 10*time.Second); err != nil {
		return fmt.Errorf("velero not ready: %w", err)
	}

	// 2. Clean up existing namespace (if --clean flag set)
	if clean {
		if err := deleteApplicationNamespace(ctx, k8sClient, log); err != nil {
			return err
		}
	} else {
		log.Info("Skipping namespace deletion (--clean=false)")
		log.Warn("Velero will merge with existing resources")
	}

	// 3. Perform Restore
	if err := veleroClient.Restore(ctx, backupName); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	// 4. Verify Applications
	if err := verifyRestoredApps(ctx, k8sClient, log); err != nil {
		return err
	}

	log.Info("")
	log.Info("==============================================")
	log.Info("  Restore completed successfully!")
	log.Info("==============================================")
	log.Info("")

	return nil
}

// deleteApplicationNamespace removes the application namespace to ensure
// a clean restore. Velero by default merges with existing resources, so
// this guarantees we're testing true restore capability from backup.
func deleteApplicationNamespace(ctx context.Context, client *k8s.Client, log *logger.Logger) error {
	log.Step("Cleaning up existing namespace: %s", ApplicationNamespace)

	if err := client.DeleteNamespace(ctx, ApplicationNamespace); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)

	}

	return nil
}

// verifyRestoredApps waits for NGINX and Redis deployments to become
// ready after Velero restore completes. This confirms the restore
// operation successfully recovered the application state.
func verifyRestoredApps(ctx context.Context, client *k8s.Client, log *logger.Logger) error {
	log.Step("Verifying restored applications")

	// Wait for NGINX
	if err := client.WaitForDeploymentReady(ctx, ApplicationNamespace, "nginx", 5*time.Minute); err != nil {
		return fmt.Errorf("nginx failed to recover: %w", err)
	}

	// Wait for Redis
	if err := client.WaitForDeploymentReady(ctx, ApplicationNamespace, "redis", 5*time.Minute); err != nil {

		return fmt.Errorf("redis failed to recover: %w", err)

	}
	return nil
}
