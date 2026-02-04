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
	"encoding/json"
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

// Apply runs terraform apply with auto-approve.
// Equivalent to bash: terraform apply -auto-approve
//
// This operation:
// - Generates an execution plan (shows what will change)
// - Applies changes without user confirmation (-auto-approve)
// - Creates infrastructure: VMs, networks, firewalls, load balancers, etc.
// - Updates existing resources if configuration changed
//
// Idempotent behavior:
// - If resources already exist and match config: no changes made
// - If resources exist but config changed: resources updated
// - If resources don't exist: resources created
//
// This can take several minutes for complex infrastructure.
// Terraform will output progress to stdout/stderr (we preserve this).
//
// Parameters:
// - ctx: Context for cancellation and timeout
//
// Returns:
// - nil if apply succeeds
// - error if apply fails (with full terraform error details)
//
// Common error scenarios:
// - Quota exceeded (not enough CPU/IP addresses in region)
// - Permission denied (service account lacks required roles)
// - Resource already exists (created outside terraform)
// - Invalid configuration (syntax errors, invalid values)
// - API rate limiting
func (c *Client) Apply(ctx context.Context) error {
	c.log.Step("Applying Terraform configuration in %s", c.workingDir)

	// Run terraform apply with auto-approve
	// This is equivalent to: terraform apply -auto-approve
	// The Apply() method from terraform-exec automatically adds -auto-approve
	err := c.tf.Apply(ctx)
	if err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	c.log.Info("Terraform apply completed successfully")
	return nil
}

// Destroy runs terraform destroy with auto-approve.
// Equivalent to bash: terraform destroy -auto-approve
//
// This operation:
// - Generates a destruction plan (shows what will be deleted)
// - Destroys ALL resources managed by terraform without user confirmation
// - Removes: VMs, networks, firewalls, load balancers, disks, etc.
// - Preserves: State file (for future apply operations)
//
// WARNING: This is a destructive operation!
// - All infrastructure in the terraform state will be DELETED
// - This cannot be undone
// - Use with caution in production environments
//
// Idempotent behavior:
// - If resources already destroyed: returns success (no-op)
// - If state file doesn't exist: returns success (nothing to destroy)
// - If resources exist: destroys them
//
// This can take several minutes for complex infrastructure.
// Resources are destroyed in reverse dependency order.
//
// Parameters:
// - ctx: Context for cancellation and timeout
//
// Returns:
// - nil if destroy succeeds or nothing to destroy
// - error if destroy fails
//
// Common error scenarios:
// - Resource locked (in use by another process)
// - Permission denied (service account lacks delete permissions)
// - Resource already deleted outside terraform (state mismatch)
// - Dependency issues (resource depended on by external resources)
func (c *Client) Destroy(ctx context.Context) error {
	c.log.Step("Destroying Terraform-managed infrastructure in %s", c.workingDir)

	// Run terraform destroy with auto-approve
	// This is equivalent to: terraform destroy -auto-approve
	// The Destroy() method from terraform-exec automatically adds -auto-approve
	err := c.tf.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	c.log.Info("Infrastructure destroyed successfully")
	return nil
}

// Output retrieves a single terraform output value by name.
// Equivalent to bash: terraform output -json <name> | jq -r '.value'
//
// This is a convenience wrapper around Outputs() for getting a single value.
// It retrieves all outputs and returns the requested one.
//
// Parameters:
// - ctx: Context for cancellation
// - name: Output variable name (e.g., "control_plane_ip", "worker_ips")
//
// Returns:
// - The output value as a string
// - Error if output doesn't exist or is not a string type
//
// Note: This only works for string outputs. For complex types (lists, maps),
// use Outputs() and handle the interface{} type yourself.
//
// Example:
//
//	ip, err := client.Output(ctx, "control_plane_ip")
//	if err != nil {
//	    return fmt.Errorf("failed to get control plane IP: %w", err)
//	}
//	log.Info("Control plane IP: %s", ip)
func (c *Client) Output(ctx context.Context, name string) (string, error) {
	// Get all outputs
	outputs, err := c.Outputs(ctx)
	if err != nil {
		return "", err
	}

	// Check if output exists
	value, ok := outputs[name]
	if !ok {
		return "", fmt.Errorf("output '%s' not found in terraform state", name)
	}

	// Terraform outputs are stored as interface{}, need type assertion
	// Most outputs are strings, but could be numbers, bools, lists, maps
	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf(
			"output '%s' is not a string (type: %T)\n"+
				"Use Outputs() for non-string output types",
			name, value,
		)
	}

	return strValue, nil
}

// Outputs retrieves all terraform outputs as a map.
// Equivalent to bash: terraform output -json
//
// Returns a map where:
// - Keys are output variable names (e.g., "control_plane_ip", "worker_ips")
// - Values are the output values as interface{} (can be string, number, bool, list, map)
//
// The values are returned as interface{} to support all terraform output types:
// - Strings: "10.0.0.1"
// - Numbers: 3
// - Booleans: true
// - Lists: []interface{}{"10.0.0.1", "10.0.0.2"}
// - Maps: map[string]interface{}{"key": "value"}
//
// You'll need to type assert the values based on your terraform outputs:
//
//	outputs, err := client.Outputs(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to get outputs: %w", err)
//	}
//
//	// String output
//	cpIP := outputs["control_plane_ip"].(string)
//
//	// List output
//	workerIPs := outputs["worker_ips"].([]interface{})
//	for _, ip := range workerIPs {
//	    log.Info("Worker IP: %s", ip.(string))
//	}
//
// Parameters:
// - ctx: Context for cancellation
//
// Returns:
// - Map of output names to values
// - Error if terraform output fails or state doesn't exist
//
// Common error scenarios:
// - No terraform state file (run apply first)
// - Invalid state file (corrupted or wrong version)
// - Terraform working directory doesn't exist
func (c *Client) Outputs(ctx context.Context) (map[string]interface{}, error) {
	c.log.Debug("Reading Terraform outputs from %s", c.workingDir)

	// Get outputs using terraform-exec Output() method
	// This returns map[string]*tfjson.StateOutput
	tfOutputs, err := c.tf.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform outputs: %w", err)
	}

	// Convert terraform-exec's StateOutput format to simple map[string]interface{}
	// StateOutput has fields: Value, Type, Sensitive
	// We only need the Value field for our purposes
	//
	// IMPORTANT: output.Value is json.RawMessage (raw JSON bytes)
	// We need to unmarshal it to get actual Go values
	result := make(map[string]interface{})
	for name, output := range tfOutputs {
		var value interface{}
		if err := json.Unmarshal(output.Value, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output '%s': %w", name, err)
		}
		result[name] = value
	}

	c.log.Debug("Read %d terraform outputs", len(result))

	return result, nil
}
