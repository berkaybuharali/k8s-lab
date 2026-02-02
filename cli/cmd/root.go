// cli/cmd/root.go
// Package cmd implements the CLI commands using the Cobra framework.
// Cobra provides command structure, flag parsing, and help generation.
//
// The root command sets up the CLI infrastructure:
// - Global flags (--cloud, --verbose)
// - Shared resources (config, logger, cloud provider)
// - Prerequisites checking (ensures required tools are installed)
// - Context-based dependency injection (passes resources to subcommands)
//
// All subcommands inherit PersistentPreRunE which runs before every command
// to initialize these shared resources and validate prerequisites.
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
	"github.com/berkaybuharali/k8s-lab/cli/pkg/prerequisites"
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
	// It initializes shared resources that all commands need:
	// 1. Configuration and logger
	// 2. Prerequisites checking (fail fast if tools missing, skipped for help/version)
	// 3. Cloud provider validation (if --cloud flag is set)
	//
	// These are stored in the command context for subcommands to access.
	rootCmd.PersistentPreRunE = persistentPreRun
}

// persistentPreRun is the main setup function that runs before every command.
// It orchestrates initialization, prerequisites checking, and cloud provider setup.
//
// This function is extracted from init() for better testability and readability.
func persistentPreRun(cmd *cobra.Command, args []string) error {
	// Step 1: Parse flags and initialize config/logger
	cloudFlag, _ := cmd.Flags().GetString("cloud")
	verbose, _ := cmd.Flags().GetBool("verbose")

	cfg, log, ctx, err := initializeConfigAndLogger(cmd.Context(), cloudFlag, verbose)
	if err != nil {
		return err
	}

	// Step 2: Check prerequisites (skip for help/version commands)
	if !shouldSkipPrerequisites(cmd) {
		log.Debug("Checking prerequisites...")
		if err := checkPrerequisites(ctx, cmd, cloudFlag, log); err != nil {
			return err
		}
		log.Debug("All prerequisites satisfied")
	}

	// Step 3: Initialize cloud provider if --cloud flag is set
	if cloudFlag != "" {
		provider, err := initializeCloudProvider(ctx, cloudFlag, cfg, log)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, providerKey, provider)
	}

	// Update command context with all initialized values
	cmd.SetContext(ctx)

	return nil
}

// initializeConfigAndLogger creates the config and logger instances
// and stores them in the context.
//
// Returns:
// - cfg: Configuration with paths and settings
// - log: Logger instance for output
// - ctx: Context with config and logger stored
// - err: Any initialization error
func initializeConfigAndLogger(parentCtx context.Context, cloudFlag string, verbose bool) (*config.Config, *logger.Logger, context.Context, error) {
	// Initialize configuration
	// This auto-detects the repository root and sets up paths
	cfg, err := config.New()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize config: %w", err)
	}
	cfg.Cloud = cloudFlag
	cfg.Verbose = verbose

	// Initialize logger
	// All output goes to stderr (stdout is reserved for data/json)
	log := logger.New(verbose)

	// Store in context for subcommands to access
	ctx := context.WithValue(parentCtx, configKey, cfg)
	ctx = context.WithValue(ctx, loggerKey, log)

	return cfg, log, ctx, nil
}

// shouldSkipPrerequisites determines if prerequisites checking should be skipped
// for the given command.
//
// We skip prerequisites for:
// - help command (users should be able to see help without tools installed)
// - version command (just prints version)
//
// Returns: true if prerequisites should be skipped, false otherwise
func shouldSkipPrerequisites(cmd *cobra.Command) bool {
	skipCommands := map[string]bool{
		"help":    true,
		"version": true,
	}
	return skipCommands[cmd.Name()]
}

// checkPrerequisites verifies all required tools are installed before proceeding.
// This implements fail-fast behavior: if any tool is missing, the command fails
// immediately with a clear error listing ALL missing tools.
//
// Prerequisites are checked in two categories:
// 1. Command-specific tools (e.g., terraform for infra, kubectl for platform)
// 2. Cloud-specific tools (e.g., gcloud for GCP, if --cloud gcp is set)
//
// Parameters:
// - ctx: Context for the check
// - cmd: Cobra command being executed
// - cloudFlag: Value of --cloud flag (empty string if not set)
// - log: Logger for debug output
//
// Returns: Error listing all missing prerequisites, or nil if all are satisfied
func checkPrerequisites(ctx context.Context, cmd *cobra.Command, cloudFlag string, log *logger.Logger) error {
	var allPrereqs []prerequisites.Prerequisite

	// Collect command-specific prerequisites
	// Walk up command tree to find top-level command name
	topLevelCmd := cmd
	for topLevelCmd.Parent() != nil && topLevelCmd.Parent().Name() != "k8s-lab" {
		topLevelCmd = topLevelCmd.Parent()
	}
	cmdPrereqs := prerequisites.GetCommandPrereqs(topLevelCmd.Name())
	allPrereqs = append(allPrereqs, cmdPrereqs...)

	// Collect cloud-specific prerequisites
	if cloudFlag != "" {
		cloudPrereqs := prerequisites.GetCloudPrereqs(cloudFlag)
		allPrereqs = append(allPrereqs, cloudPrereqs...)
	}

	// Check all prerequisites at once
	// This shows ALL missing tools in a single error, not one at a time
	if len(allPrereqs) > 0 {
		if err := prerequisites.CheckAll(ctx, allPrereqs...); err != nil {
			return err
		}
	}

	return nil
}

// initializeCloudProvider validates and initializes the cloud provider.
// This includes:
// 1. Validating the cloud provider name
// 2. Getting the provider from the registry
// 3. Validating cloud-specific authentication (e.g., GCP Application Default Credentials)
//
// Parameters:
// - ctx: Context for cloud provider operations
// - cloudFlag: Cloud provider name (e.g., "gcp")
// - cfg: Configuration instance
// - log: Logger for output
//
// Returns: Initialized cloud provider, or error if validation fails
func initializeCloudProvider(ctx context.Context, cloudFlag string, cfg *config.Config, log *logger.Logger) (cloud.Provider, error) {
	// Check if cloud provider name is valid
	if err := cfg.ValidateCloud(); err != nil {
		return nil, err
	}

	// Get provider from registry
	// Providers auto-register themselves via init() functions
	provider := cloud.Get(cloudFlag)
	if provider == nil {
		return nil, fmt.Errorf(
			"cloud provider '%s' not available\n"+
				"Available providers: %v",
			cloudFlag, cloud.List(),
		)
	}

	// Inject logger into provider (cloud-agnostic requirement)
	// All providers MUST have logger for operations logging
	provider.SetLogger(log)

	// Validate cloud provider authentication and configuration
	// For GCP: checks Application Default Credentials exist
	log.Debug("Validating cloud provider: %s", cloudFlag)
	if err := provider.Validate(ctx); err != nil {
		return nil, fmt.Errorf("cloud provider validation failed: %w", err)
	}
	log.Debug("Cloud provider validated successfully")

	return provider, nil
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
