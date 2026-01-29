// cli/cmd/root.go
// Package cmd implements the CLI commands using the Cobra framework.
// Cobra provides command structure, flag parsing, and help generation.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
	_ "github.com/berkaybuharali/k8s-lab/cli/pkg/cloud/gcp" // Register GCP provider
	"github.com/berkaybuharali/k8s-lab/cli/pkg/config"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

// Version is the CLI version, set at build time or defaulted here.
var Version = "0.1.0"

// Context keys for storing values in command context.
// These are used to pass config, logger, and provider between commands.
type contextKey string

const (
	configKey   contextKey = "config"
	loggerKey   contextKey = "logger"
	providerKey contextKey = "provider"
)

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

	// PersistentPreRunE runs before every command and is inherited by all subcommands.
	// We use it to initialize shared resources that all commands need:
	// 1. Configuration (paths, cluster name, etc.)
	// 2. Logger (colored output)
	// 3. Cloud provider (if --cloud flag is set)
	//
	// These are stored in the command context so subcommands can access them
	// without needing to pass them as parameters.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		cloudFlag, _ := cmd.Flags().GetString("cloud")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Initialize configuration
		// This auto-detects the repository root and sets up paths
		cfg, err := config.New()
		if err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
		cfg.Cloud = cloudFlag
		cfg.Verbose = verbose

		// Initialize logger
		// All output goes to stderr (stdout is for data/json)
		log := logger.New(verbose)

		// Store in command context for subcommands to access
		ctx := context.WithValue(cmd.Context(), configKey, cfg)
		ctx = context.WithValue(ctx, loggerKey, log)

		// If cloud provider is specified, validate and store it
		if cloudFlag != "" {
			// Check if cloud provider name is valid
			if err := cfg.ValidateCloud(); err != nil {
				return err
			}

			// Get provider from registry
			// Providers auto-register themselves via init() functions
			provider := cloud.Get(cloudFlag)
			if provider == nil {
				return fmt.Errorf(
					"cloud provider '%s' not available\n"+
						"Available providers: %v",
					cloudFlag, cloud.List(),
				)
			}

			// Validate cloud provider prerequisites
			// For GCP: checks Application Default Credentials exist
			log.Debug("Validating cloud provider: %s", cloudFlag)
			if err := provider.Validate(ctx); err != nil {
				return fmt.Errorf("cloud provider validation failed: %w", err)
			}
			log.Debug("Cloud provider validated successfully")

			// Store provider in context
			ctx = context.WithValue(ctx, providerKey, provider)
		}

		// Update command context with all values
		cmd.SetContext(ctx)

		return nil
	}
}

// GetConfig retrieves the Config from the command context.
// This is a helper for subcommands to access the shared configuration.
//
// The Config contains paths, cluster name, cloud provider selection, etc.
//
// Example usage in a subcommand:
//
//	func deployCmd() *cobra.Command {
//	    return &cobra.Command{
//	        RunE: func(cmd *cobra.Command, args []string) error {
//	            cfg := GetConfig(cmd)
//	            log := GetLogger(cmd)
//	            log.Info("Deploying to %s", cfg.Cloud)
//	            // ...
//	        },
//	    }
//	}
func GetConfig(cmd *cobra.Command) *config.Config {
	val := cmd.Context().Value(configKey)
	if val == nil {
		// This should never happen if PersistentPreRunE ran successfully
		panic("config not found in context - this is a bug")
	}
	return val.(*config.Config)
}

// GetLogger retrieves the Logger from the command context.
// This is a helper for subcommands to access the shared logger.
//
// The Logger provides colored output with Info, Warn, Error, Step, Debug levels.
//
// Example usage:
//
//	log := GetLogger(cmd)
//	log.Step("Creating infrastructure...")
//	log.Info("Resources created successfully")
func GetLogger(cmd *cobra.Command) *logger.Logger {
	val := cmd.Context().Value(loggerKey)
	if val == nil {
		panic("logger not found in context - this is a bug")
	}
	return val.(*logger.Logger)
}

// GetProvider retrieves the cloud Provider from the command context.
// Returns nil if no cloud provider was specified (--cloud flag not set)
// or if context hasn't been initialized.
//
// The Provider interface provides cloud-specific operations:
// - Validate() checks authentication
// - EnsureStateBucket() creates Terraform state bucket
// - GetProjectID() reads project ID from terraform.tfvars
//
// Example usage:
//
//	provider := GetProvider(cmd)
//	if provider == nil {
//	    return fmt.Errorf("--cloud flag is required")
//	}
//	projectID, err := provider.GetProjectID(cfg.GetTerraformDir())
func GetProvider(cmd *cobra.Command) cloud.Provider {
	ctx := cmd.Context()
	if ctx == nil {
		return nil
	}
	val := ctx.Value(providerKey)
	if val == nil {
		return nil
	}
	return val.(cloud.Provider)
}

// RequireCloud is a helper that ensures the --cloud flag was provided.
// Returns a clear error message if not.
//
// Use this in commands that require a cloud provider:
//
//	func deployCmd() *cobra.Command {
//	    return &cobra.Command{
//	        RunE: func(cmd *cobra.Command, args []string) error {
//	            provider := GetProvider(cmd)
//	            if err := RequireCloud(provider); err != nil {
//	                return err
//	            }
//	            // ... use provider ...
//	        },
//	    }
//	}
func RequireCloud(provider cloud.Provider) error {
	if provider == nil {
		return fmt.Errorf(
			"--cloud flag is required\nAvailable providers: %v",
			cloud.List(),
		)
	}
	return nil
}
