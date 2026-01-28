// cli/cmd/root.go
// Package cmd implements the CLI commands using the Cobra framework.
// Cobra provides command structure, flag parsing, and help generation.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is the CLI version, set at build time or defaulted here.
var Version = "0.1.0"

// rootCmd represents the base command when called without any subcommands.
// It serves as the parent for all other commands (deploy, backup, etc.).
var rootCmd = &cobra.Command{
	Use:   "k8s-lab",
	Short: "Kubernetes lab environment manager",
	Long: `k8s-lab is a CLI tool for managing ephemeral Kubernetes lab environments.

It handles:
- Infrastructure provisioning (Terraform)
- Cluster bootstrapping (Talos Linux)
- Platform tools (Velero, CSI drivers)
- Application deployments (NGINX, Redis, PostgreSQL)
- Backup and restore operations

Supported clouds: GCP (more coming soon)`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// If command execution fails, Cobra will print the error and help text.
	// We exit with code 1 to indicate failure.
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags available to all subcommands
	// --cloud: Specify cloud provider (required for most commands)
	rootCmd.PersistentFlags().StringP("cloud", "c", "", "Cloud provider (gcp, stackit)")

	// --verbose: Enable detailed logging
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
}
