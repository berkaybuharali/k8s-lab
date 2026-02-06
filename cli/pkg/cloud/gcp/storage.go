// cli/pkg/cloud/gcp/storage.go
// GCS (Google Cloud Storage) bucket operations for Terraform state management.
package gcp

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
)

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

	p.log.Info("Ensuring GCS bucket exists: %s", bucketName)

	// Create storage client if not cached
	if p.storageClient == nil {
		p.log.Debug("Creating GCS storage client...")
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
	p.log.Debug("Checking if bucket %s exists...", bucketName)
	_, err := bucket.Attrs(ctx)
	if err == nil {
		// Bucket exists, we're done
		p.log.Info("GCS bucket %s already exists", bucketName)
		return nil
	}

	// If error is not "bucket doesn't exist", return it
	if err != storage.ErrBucketNotExist {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	// Bucket doesn't exist, create it
	p.log.Info("Creating GCS bucket %s in US-CENTRAL1...", bucketName)
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

	p.log.Info("GCS bucket %s created successfully", bucketName)

	return nil
}
