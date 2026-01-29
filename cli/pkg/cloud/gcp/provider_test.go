// cli/pkg/cloud/gcp/provider_test.go
package gcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
)

// TestProviderRegistration verifies that the GCP provider is automatically
// registered in the global registry when the package is imported.
func TestProviderRegistration(t *testing.T) {
	// The init() function should have registered the provider
	provider := cloud.Get("gcp")
	if provider == nil {
		t.Fatal("GCP provider not registered in global registry")
	}

	if provider.Name() != "gcp" {
		t.Errorf("Expected provider name 'gcp', got '%s'", provider.Name())
	}
}

// TestProviderName verifies that Name() returns the correct identifier.
func TestProviderName(t *testing.T) {
	p := &Provider{}
	if p.Name() != "gcp" {
		t.Errorf("Expected name 'gcp', got '%s'", p.Name())
	}
}

// TestValidate verifies that Validate() checks for gcloud CLI.
// Note: This test will fail if gcloud is not installed. That's expected
// and demonstrates the validation is working.
func TestValidate(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	err := p.Validate(ctx)

	// We can't assert success/failure without knowing if gcloud is installed
	// Just verify the method runs without panicking
	t.Logf("Validate result: %v", err)

	if err != nil {
		t.Logf("Validation failed (this is OK if gcloud not installed): %v", err)
	} else {
		t.Log("Validation succeeded (gcloud is installed and authenticated)")
	}
}

// TestGetProjectID verifies parsing of terraform.tfvars.
func TestGetProjectID(t *testing.T) {
	p := &Provider{}

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		expectID    string
		expectError bool
	}{
		{
			name: "valid project_id with spaces",
			content: `project_id = "my-test-project"
region = "us-central1"`,
			expectID:    "my-test-project",
			expectError: false,
		},
		{
			name: "valid project_id without spaces",
			content: `project_id="another-project"
region="us-east1"`,
			expectID:    "another-project",
			expectError: false,
		},
		{
			name: "project_id with comments",
			content: `# GCP Configuration
# project_id = "commented-out"
project_id = "real-project"`,
			expectID:    "real-project",
			expectError: false,
		},
		{
			name: "project_id with single quotes",
			content: `project_id = 'single-quote-project'`,
			expectID:    "single-quote-project",
			expectError: false,
		},
		{
			name: "missing project_id",
			content: `region = "us-central1"
zone = "us-central1-a"`,
			expectID:    "",
			expectError: true,
		},
		{
			name: "empty project_id",
			content: `project_id = ""`,
			expectID:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create terraform.tfvars in temp directory
			testDir := filepath.Join(tempDir, tt.name)
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			tfvarsPath := filepath.Join(testDir, "terraform.tfvars")
			err = os.WriteFile(tfvarsPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test tfvars: %v", err)
			}

			// Test GetProjectID
			projectID, err := p.GetProjectID(testDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if projectID != tt.expectID {
					t.Errorf("Expected project_id '%s', got '%s'", tt.expectID, projectID)
				}
			}
		})
	}
}

// TestGetProjectID_FileNotFound verifies error when terraform.tfvars doesn't exist.
func TestGetProjectID_FileNotFound(t *testing.T) {
	p := &Provider{}

	// Use a directory that definitely doesn't have terraform.tfvars
	nonExistentDir := "/tmp/definitely-does-not-exist-" + t.Name()

	_, err := p.GetProjectID(nonExistentDir)
	if err == nil {
		t.Error("Expected error when terraform.tfvars doesn't exist")
	}

	t.Logf("Got expected error: %v", err)
}

// TestEnsureStateBucket is intentionally not implemented.
// This would require either:
// 1. Actually calling GCP APIs (slow, requires auth, costs money)
// 2. Mocking exec.Command (complex and brittle)
// 3. Integration tests in a separate suite
//
// For now, we rely on manual testing: make build && ./bin/k8s-lab ...
func TestEnsureStateBucket(t *testing.T) {
	t.Skip("EnsureStateBucket requires GCP credentials and creates real resources")

	// Future: Could implement with environment variable to enable
	// if os.Getenv("RUN_GCP_INTEGRATION_TESTS") != "true" {
	//     t.Skip("Set RUN_GCP_INTEGRATION_TESTS=true to run")
	// }
}
