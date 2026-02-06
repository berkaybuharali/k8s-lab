// cli/pkg/config/config.go
// Package config manages CLI configuration including paths, cluster settings,
// and cloud provider options. It replicates the constants and path logic
// from scripts/lib/common.sh.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the CLI configuration.
// This replaces environment variables and constants from common.sh.
type Config struct {
	// RepoRoot is the absolute path to the repository root directory.
	// Equivalent to bash: REPO_ROOT
	RepoRoot string

	// ClusterName is the name of the Kubernetes cluster.
	// Equivalent to bash: CLUSTER_NAME="k8s-lab"
	ClusterName string

	// SupportedClouds lists valid cloud provider names.
	// Equivalent to bash: SUPPORTED_CLOUDS=("gcp")
	SupportedClouds []string

	// Cloud is the currently selected cloud provider (from --cloud flag).
	Cloud string

	// Verbose enables detailed debug logging.
	Verbose bool
}

// New creates a new Config with default values.
// It auto-detects the repository root by walking up from the current directory
// until it finds a directory containing "go.mod" or ".git".
func New() (*Config, error) {
	// Find repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find repository root: %w", err)
	}

	return &Config{
		RepoRoot:        repoRoot,
		ClusterName:     "k8s-lab",
		SupportedClouds: []string{"gcp"}, // Will expand to include STACKIT
		Cloud:           "",
		Verbose:         false,
	}, nil
}

// ValidateCloud checks if the specified cloud provider is supported.
// Equivalent to bash: validate_cloud() function
func (c *Config) ValidateCloud() error {
	if c.Cloud == "" {
		return fmt.Errorf("cloud provider required (use --cloud flag)")
	}

	for _, supported := range c.SupportedClouds {
		if c.Cloud == supported {
			return nil
		}
	}

	return fmt.Errorf("unsupported cloud provider: %s (supported: %v)", c.Cloud, c.SupportedClouds)
}

// GetTerraformDir returns the Terraform directory for the selected cloud.
// Equivalent to bash: TF_DIR="${REPO_ROOT}/infra/gcp/terraform"
func (c *Config) GetTerraformDir() string {
	return filepath.Join(c.RepoRoot, "infra", c.Cloud, "terraform")
}

// GetConfigsDir returns the configs directory.
// Equivalent to bash: CONFIGS_DIR="${REPO_ROOT}/configs"
func (c *Config) GetConfigsDir() string {
	return filepath.Join(c.RepoRoot, "configs")
}

// GetTalosConfigsDir returns the Talos configs directory.
// Equivalent to bash: TALOS_CONFIGS_DIR="${CONFIGS_DIR}/talos"
func (c *Config) GetTalosConfigsDir() string {
	return filepath.Join(c.GetConfigsDir(), "talos")
}

// GetAppsDir returns the applications manifest directory.
// For cloud-specific apps: apps/<cloud>/
// For cloud-agnostic apps: apps/
func (c *Config) GetAppsDir() string {
	return filepath.Join(c.RepoRoot, "apps")
}

// GetCloudAppsDir returns the cloud-specific applications directory.
func (c *Config) GetCloudAppsDir() string {
	return filepath.Join(c.GetAppsDir(), c.Cloud)
}

// GetKubeconfigPath returns the path to the kubeconfig file.
// This is created by deploy-infra command and used by other commands.
func (c *Config) GetKubeconfigPath() string {
	return filepath.Join(c.GetTalosConfigsDir(), "kubeconfig")
}

// GetRepoRoot returns the repository root directory.
func (c *Config) GetRepoRoot() string {
	return c.RepoRoot
}

// findRepoRoot checks if the current directory is the repository root.
// The k8s-lab CLI must be run from the repository root directory,
// matching the behavior of Makefile commands.
//
// This is simpler than walking up the directory tree and matches user
// expectations - both "make deploy gcp" and "k8s-lab infra deploy --cloud gcp"
// should be run from the same location (repo root).
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if current directory has repo markers
	hasGit := fileExists(filepath.Join(dir, ".git"))
	hasMakefile := fileExists(filepath.Join(dir, "Makefile"))
	hasInfra := fileExists(filepath.Join(dir, "infra"))

	// Verify we're in the k8s-lab repository root
	if hasGit && hasMakefile && hasInfra {
		return dir, nil
	}

	// Provide helpful error message
	return "", fmt.Errorf(
		"must run k8s-lab from repository root directory\n"+
			"Current directory: %s\n"+
			"Expected: directory containing .git, Makefile, and infra/\n"+
			"Hint: cd to your k8s-lab repository root",
		dir,
	)
}

// fileExists checks if a file or directory exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
