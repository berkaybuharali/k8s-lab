// cli/pkg/terraform/terraform.go
// Package terraform provides a Go wrapper around Terraform operations.
// Instead of calling "terraform" as a shell command, we use the official
// terraform-exec library to interact with Terraform programmatically.
//
// This is safer and more reliable than parsing shell command output.
//
// The terraform-exec library is the official way to interact with Terraform
// from Go code. It wraps the terraform binary and provides a type-safe API.
//
// IMPORTANT: This library requires the terraform binary to be installed.
// Unlike cloud SDKs (GCP, AWS) which are pure Go, Terraform has no Go SDK
// for execution - terraform-exec is the official wrapper around the binary.
package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/terraform-exec/tfexec"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

// Client wraps terraform-exec for our specific needs.
// It handles init, apply, destroy, and output operations.
//
// The Client maintains:
// - tf: The terraform-exec instance that wraps the terraform binary
// - log: Logger for user-facing messages (matches bash script output)
// - workingDir: The Terraform directory (e.g., infra/gcp/terraform)
type Client struct {
	// tf is the terraform-exec instance
	tf *tfexec.Terraform

	// log is the logger for user-facing messages
	log *logger.Logger

	// workingDir is the Terraform directory (e.g., infra/gcp/terraform)
	workingDir string
}

// NewClient creates a new Terraform client.
//
// Parameters:
// - ctx: Context for cancellation and timeout
// - workingDir: Absolute path to the Terraform directory
// - log: Logger instance for output
//
// Returns error if:
// - terraform binary is not found in PATH
// - terraform-exec initialization fails
//
// Example usage:
//
//	tfDir := cfg.GetTerraformDir()
//	client, err := terraform.NewClient(ctx, tfDir, log)
//	if err != nil {
//	    return fmt.Errorf("failed to create terraform client: %w", err)
//	}
func NewClient(ctx context.Context, workingDir string, log *logger.Logger) (*Client, error) {
	// Find terraform binary in PATH
	// This will be implemented in step 3b
	terraformBin, err := findTerraformBinary()
	if err != nil {
		return nil, fmt.Errorf("terraform not found: %w", err)
	}

	// Create terraform-exec instance
	// This doesn't execute anything yet, just sets up the wrapper
	tf, err := tfexec.NewTerraform(workingDir, terraformBin)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize terraform: %w", err)
	}

	// Configure terraform output streams
	// By default, terraform output goes to our stdout/stderr
	// This preserves terraform's colored output and progress indicators
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	return &Client{
		tf:         tf,
		log:        log,
		workingDir: workingDir,
	}, nil
}

// findTerraformBinary locates the terraform binary in PATH.
// It uses exec.LookPath which searches the directories listed in the PATH
// environment variable for an executable file named "terraform".
//
// This is equivalent to bash: command -v terraform
//
// Note: The prerequisites checking system already verified terraform exists,
// so this function should always succeed during normal operation. However,
// we still check and return a clear error in case something changes between
// the prerequisites check and this call.
//
// Returns:
// - path: Full path to terraform binary (e.g., "/opt/homebrew/bin/terraform")
// - error: If terraform is not found in PATH
func findTerraformBinary() (string, error) {
	path, err := exec.LookPath("terraform")
	if err != nil {
		return "", fmt.Errorf(
			"terraform binary not found in PATH\n"+
				"Install: brew install terraform\n"+
				"Verify: which terraform",
		)
	}
	return path, nil
}

// Init runs terraform init with upgrade flag.
// Equivalent to bash: terraform init -upgrade
//
// This operation:
// - Downloads provider plugins (hashicorp/google, etc.)
// - Initializes the backend (GCS bucket for state storage)
// - Upgrades providers to latest allowed versions (respects version constraints)
// - Creates .terraform directory and .terraform.lock.hcl
//
// This is idempotent - safe to run multiple times.
// Running init again will:
// - Re-download plugins if missing
// - Re-configure backend if changed
// - Upgrade plugins if new versions available
//
// Parameters:
// - ctx: Context for cancellation and timeout
//
// Returns:
// - nil if initialization succeeds
// - error if terraform init fails (with full error details from terraform)
//
// Example error scenarios:
// - Backend bucket doesn't exist
// - Invalid terraform configuration syntax
// - Network issues downloading providers
// - Insufficient permissions for GCS backend
func (c *Client) Init(ctx context.Context) error {
	c.log.Step("Initializing Terraform in %s", c.workingDir)

	// Run terraform init with upgrade flag
	// tfexec.Upgrade(true) adds -upgrade to the command
	// This is equivalent to: terraform init -upgrade
	err := c.tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	c.log.Info("Terraform initialized successfully")
	return nil
}

// Apply runs terraform apply.
// Equivalent to bash: terraform apply -auto-approve
//
// This operation:
// - Plans infrastructure changes
// - Applies changes without user confirmation (-auto-approve)
// - Creates VMs, networks, firewalls, etc.
//
// This is idempotent for most resources (creates if not exists).
func (c *Client) Apply(ctx context.Context) error {
	// TODO: Implement in step 3c
	return fmt.Errorf("not implemented yet")
}

// Destroy runs terraform destroy.
// Equivalent to bash: terraform destroy -auto-approve
//
// This operation:
// - Plans destruction of all managed resources
// - Destroys resources without user confirmation (-auto-approve)
// - Removes VMs, networks, firewalls, etc.
//
// WARNING: This deletes infrastructure. State file remains for next apply.
func (c *Client) Destroy(ctx context.Context) error {
	// TODO: Implement in step 3d
	return fmt.Errorf("not implemented yet")
}

// Output retrieves a single terraform output value.
// Equivalent to bash: terraform output -json <name> | jq -r
//
// Parameters:
// - ctx: Context for cancellation
// - name: Output variable name (e.g., "control_plane_ip")
//
// Returns the output value as a string.
// Returns error if output doesn't exist.
//
// Example:
//
//	ip, err := client.Output(ctx, "control_plane_ip")
//	if err != nil {
//	    return fmt.Errorf("failed to get control plane IP: %w", err)
//	}
func (c *Client) Output(ctx context.Context, name string) (string, error) {
	// TODO: Implement in step 3e
	return "", fmt.Errorf("not implemented yet")
}

// Outputs retrieves all terraform outputs as a map.
// Equivalent to bash: terraform output -json
//
// Returns a map where:
// - Keys are output variable names
// - Values are the output values (any type)
//
// This is useful for getting multiple outputs at once.
//
// Example:
//
//	outputs, err := client.Outputs(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to get outputs: %w", err)
//	}
//	cpIP := outputs["control_plane_ip"]
func (c *Client) Outputs(ctx context.Context) (map[string]interface{}, error) {
	// TODO: Implement in step 3e
	return nil, fmt.Errorf("not implemented yet")
}
