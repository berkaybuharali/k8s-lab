// cli/pkg/cloud/gcp/provider.go
// Package gcp implements the cloud.Provider interface for Google Cloud Platform.
// It uses Google Cloud Go SDK for all GCP operations:
// - cloud.google.com/go/storage for GCS bucket operations
// - golang.org/x/oauth2/google for authentication
//
// Authentication is via Application Default Credentials (ADC).
// User sets up once with: gcloud auth application-default login
//
// This matches bash scripts behavior but uses Go libraries instead of CLI commands.
package gcp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
)

// Provider implements cloud.Provider for Google Cloud Platform.
type Provider struct {
	// storageClient is cached to avoid creating multiple clients.
	storageClient *storage.Client
}

// Ensure Provider implements cloud.Provider interface at compile time.
var _ cloud.Provider = (*Provider)(nil)

func init() {
	// Auto-register this provider in the global registry.
	cloud.Register("gcp", &Provider{})
}

// Name returns the cloud provider identifier "gcp".
func (p *Provider) Name() string {
	return "gcp"
}

// Validate checks if Google Cloud credentials are configured.
// It verifies Application Default Credentials (ADC) exist.
//
// User setup (one time):
//   gcloud auth application-default login
//
// This creates: ~/.config/gcloud/application_default_credentials.json
// All Google Cloud SDKs automatically use these credentials.
//
// Alternative: Set GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
func (p *Provider) Validate(ctx context.Context) error {
	// Check if Application Default Credentials exist
	// This looks for credentials in this order:
	// 1. GOOGLE_APPLICATION_CREDENTIALS env var (service account JSON)
	// 2. ~/.config/gcloud/application_default_credentials.json
	// 3. GCE/GKE metadata server (if running on GCP)
	_, err := google.FindDefaultCredentials(ctx, storage.ScopeFullControl)
	if err != nil {
		return fmt.Errorf(
			"no Google Cloud credentials found: %w\n\n"+
				"Setup:\n"+
				"  gcloud auth application-default login\n\n"+
				"Or set environment variable:\n"+
				"  export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json\n\n"+
				"Install gcloud: https://cloud.google.com/sdk/docs/install",
			err,
		)
	}

	// Credentials exist - actual access will be verified when we use them
	return nil
}

// EnsureStateBucket creates a GCS bucket for Terraform state if it doesn't exist.
// The bucket stores terraform.tfstate with versioning enabled.
//
// This operation is idempotent - safe to call multiple times.
//
// GCS bucket names must be globally unique across ALL Google Cloud users.
// If creation fails, try a more unique name (add project ID or random suffix).
func (p *Provider) EnsureStateBucket(ctx context.Context, bucketName, projectID string) error {
	if bucketName == "" {
		return fmt.Errorf("bucket name is required")
	}
	if projectID == "" {
		return fmt.Errorf("project ID is required")
	}

	// Create storage client if not cached
	if p.storageClient == nil {
		client, err := storage.NewClient(ctx)
		if err != nil {
			return fmt.Errorf(
				"failed to create storage client: %w\n"+
					"Run: gcloud auth application-default login",
				err,
			)
		}
		p.storageClient = client
	}

	bucket := p.storageClient.Bucket(bucketName)

	// Check if bucket already exists
	_, err := bucket.Attrs(ctx)
	if err == nil {
		// Bucket exists, we're done
		return nil
	}

	// If error is not "bucket doesn't exist", return it
	if err != storage.ErrBucketNotExist {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	// Bucket doesn't exist, create it
	err = bucket.Create(ctx, projectID, &storage.BucketAttrs{
		Location: "US-CENTRAL1", // TODO: Make configurable
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true, // Use IAM only (no ACLs)
		},
		VersioningEnabled: true, // Enable versioning for state file rollback
	})

	if err != nil {
		// Check if error is due to bucket name already taken globally
		errMsg := err.Error()
		if strings.Contains(errMsg, "already exists") ||
			strings.Contains(errMsg, "You already own this bucket") {
			return fmt.Errorf(
				"bucket name '%s' is already taken (names are globally unique)\n"+
					"Fix: Use a different name in terraform.tfvars\n"+
					"Example: state_bucket = \"my-project-tfstate-abc123\"",
				bucketName,
			)
		}

		// Generic error
		return fmt.Errorf(
			"failed to create bucket '%s': %w\n"+
				"Check: project ID, IAM permissions, billing enabled",
			bucketName, err,
		)
	}

	return nil
}

// GetProjectID reads project_id from terraform.tfvars file.
// Expected format: project_id = "my-gcp-project"
//
// This is a simple file parser - just looks for the project_id line.
func (p *Provider) GetProjectID(terraformDir string) (string, error) {
	tfvarsPath := filepath.Join(terraformDir, "terraform.tfvars")

	file, err := os.Open(tfvarsPath)
	if err != nil {
		return "", fmt.Errorf(
			"cannot open terraform.tfvars: %w\n"+
				"Expected: %s\n"+
				"Hint: Copy from terraform.tfvars.example",
			err, tfvarsPath,
		)
	}
	defer file.Close()

	// Scan file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Look for: project_id = "value"
		if strings.HasPrefix(line, "project_id") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			// Extract value and remove quotes
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, `"'`)

			if value == "" {
				return "", fmt.Errorf("project_id is empty in terraform.tfvars")
			}

			return value, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading terraform.tfvars: %w", err)
	}

	return "", fmt.Errorf(
		"project_id not found in terraform.tfvars\n"+
			"Expected: project_id = \"your-gcp-project\"\n"+
			"File: %s",
		tfvarsPath,
	)
}
