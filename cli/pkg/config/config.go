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

// findRepoRoot walks up the directory tree to find the repository root.
// It looks for a directory containing ".git" or "Makefile".
// We prioritize .git and Makefile over go.mod because the Go module
// is in the cli/ subdirectory, not the actual repo root.
func findRepoRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find repo markers or reach root
	for {
		// Check for .git first (most reliable indicator of repo root)
		if fileExists(filepath.Join(dir, ".git")) {
			return dir, nil
		}

		// Check for Makefile (our repo has one at root)
		if fileExists(filepath.Join(dir, "Makefile")) {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding repo
			return "", fmt.Errorf("repository root not found")
		}
		dir = parent
	}
}

// fileExists checks if a file or directory exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
