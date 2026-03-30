// cli/cmd/backup.go
// Package cmd implements the backup command.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/velero"
)

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().String("name", "k8s-lab-backup", "Base name for the backup")
	backupCmd.Flags().String("namespaces", AgentsNamespace, "Comma-separated list of namespaces to backup")
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a Velero backup",
	Long: `Create a Velero backup of specified namespaces.

The backup name will be suffixed with a timestamp (ddmmyyyy-hhmm).
Example: k8s-lab-backup-04022026-1430

Usage:
  k8s-lab backup --cloud gcp --name my-backup`,
	RunE: runBackup,
}

func runBackup(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	namespaces, _ := cmd.Flags().GetString("namespaces")

	log.Info("==============================================")
	log.Info("  Kubernetes Lab - Backup")
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

	// Create Velero client
	veleroClient, err := velero.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return fmt.Errorf("failed to create velero client: %w", err)
	}

	// Create Backup
	finalName, err := veleroClient.CreateBackup(ctx, name, namespaces)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	log.Info("")
	log.Info("==============================================")
	log.Info("  Backup completed successfully!")
	log.Info("  Name: %s", finalName)
	log.Info("==============================================")
	log.Info("")
	log.Info("Next steps:")
	log.Info("  k8s-lab restore --cloud gcp --backup %s", finalName)
	log.Info("")

	return nil
}
