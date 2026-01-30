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

// findTerraformBinary locates the terraform binary.
// It uses the same logic as bash: command -v terraform
//
// This is a placeholder for step 3b.
// For now, we just return "terraform" and let exec.LookPath find it.
func findTerraformBinary() (string, error) {
	// TODO: Implement in step 3b
	// Will use exec.LookPath to find terraform in PATH
	return "terraform", nil
}

// Init runs terraform init.
// Equivalent to bash: terraform init -upgrade
//
// This operation:
// - Downloads provider plugins
// - Initializes the backend (GCS for state storage)
// - Upgrades providers to latest allowed versions
//
// This is idempotent - safe to run multiple times.
func (c *Client) Init(ctx context.Context) error {
	// TODO: Implement in step 3b
	return fmt.Errorf("not implemented yet")
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
