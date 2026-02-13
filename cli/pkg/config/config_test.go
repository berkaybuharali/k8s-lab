// cli/pkg/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDir changes to repo root for testing.
// Tests need to run from repo root since that's where the CLI expects to run.
func setupTestDir(t *testing.T) func() {
	// Save original directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Walk up to find repo root (for tests only)
	dir := originalDir
	for {
		if fileExists(filepath.Join(dir, ".git")) &&
			fileExists(filepath.Join(dir, "cli")) {
			// Found repo root, change to it
			if err := os.Chdir(dir); err != nil {
				t.Fatalf("Failed to change to repo root: %v", err)
			}
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("Could not find repo root from: %s", originalDir)
		}
		dir = parent
	}

	// Return cleanup function
	return func() {
		os.Chdir(originalDir)
	}
}

// TestNew verifies that New() creates a config with proper defaults.
func TestNew(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg, err := New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Check default values
	if cfg.ClusterName != "k8s-lab" {
		t.Errorf("Expected cluster name 'k8s-lab', got %s", cfg.ClusterName)
	}

	if len(cfg.SupportedClouds) == 0 {
		t.Error("Expected at least one supported cloud")
	}

	// Verify GCP is in supported clouds
	gcpFound := false
	for _, cloud := range cfg.SupportedClouds {
		if cloud == "gcp" {
			gcpFound = true
			break
		}
	}
	if !gcpFound {
		t.Error("Expected 'gcp' in supported clouds")
	}

	// RepoRoot should be detected
	if cfg.RepoRoot == "" {
		t.Error("Expected RepoRoot to be detected, got empty string")
	}

	// RepoRoot should end with k8s-lab (since we're in that repo)
	if !strings.HasSuffix(cfg.RepoRoot, "k8s-lab") {
		t.Logf("Warning: RepoRoot '%s' doesn't end with 'k8s-lab'", cfg.RepoRoot)
	}

	t.Logf("Repo root: %s", cfg.RepoRoot)
	t.Logf("Supported clouds: %v", cfg.SupportedClouds)
}

// TestValidateCloud verifies cloud validation logic.
func TestValidateCloud(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg, _ := New()

	// Test empty cloud
	cfg.Cloud = ""
	if err := cfg.ValidateCloud(); err == nil {
		t.Error("Expected error for empty cloud provider")
	}

	// Test invalid cloud
	cfg.Cloud = "invalid"
	if err := cfg.ValidateCloud(); err == nil {
		t.Error("Expected error for invalid cloud")
	}

	// Test valid cloud
	cfg.Cloud = "gcp"
	if err := cfg.ValidateCloud(); err != nil {
		t.Errorf("Expected no error for valid cloud 'gcp': %v", err)
	}
}

// TestGetTerraformDir verifies Terraform directory path construction.
func TestGetTerraformDir(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg, _ := New()
	cfg.Cloud = "gcp"

	tfDir := cfg.GetTerraformDir()

	// Should end with infra/gcp/terraform
	if !strings.HasSuffix(tfDir, "infra/gcp/terraform") {
		t.Errorf("Expected path to end with 'infra/gcp/terraform', got: %s", tfDir)
	}

	t.Logf("Terraform dir: %s", tfDir)
}

// TestGetConfigsDir verifies configs directory path construction.
func TestGetConfigsDir(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg, _ := New()

	configsDir := cfg.GetConfigsDir()

	// Should end with configs
	if !strings.HasSuffix(configsDir, "configs") {
		t.Errorf("Expected path to end with 'configs', got: %s", configsDir)
	}

	t.Logf("Configs dir: %s", configsDir)
}

// TestGetTalosConfigsDir verifies Talos configs directory path construction.
func TestGetTalosConfigsDir(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg, _ := New()

	talosDir := cfg.GetTalosConfigsDir()

	// Should end with configs/talos
	if !strings.HasSuffix(talosDir, "configs/talos") {
		t.Errorf("Expected path to end with 'configs/talos', got: %s", talosDir)
	}

	t.Logf("Talos configs dir: %s", talosDir)
}

// TestGetAppsDir verifies apps directory path construction.
func TestGetAppsDir(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg, _ := New()

	appsDir := cfg.GetAppsDir()

	// Should end with apps
	if !strings.HasSuffix(appsDir, "apps") {
		t.Errorf("Expected path to end with 'apps', got: %s", appsDir)
	}

	t.Logf("Apps dir: %s", appsDir)
}

// TestGetCloudAppsDir verifies cloud-specific apps directory path construction.
func TestGetCloudAppsDir(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg, _ := New()
	cfg.Cloud = "gcp"

	cloudAppsDir := cfg.GetCloudAppsDir()

	// Should end with apps/gcp
	if !strings.HasSuffix(cloudAppsDir, "apps/gcp") {
		t.Errorf("Expected path to end with 'apps/gcp', got: %s", cloudAppsDir)
	}

	t.Logf("Cloud apps dir: %s", cloudAppsDir)
}
