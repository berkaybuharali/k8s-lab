// cli/pkg/prerequisites/prerequisite.go
// Package prerequisites provides a system for checking required tools and conditions
// before executing commands. This prevents partial failures and provides clear error
// messages listing all missing prerequisites at once.
//
// The package follows a declarative pattern where:
// - Commands declare what tools they need (CommandPrereqs map)
// - Cloud providers declare what tools they need (CloudPrereqs map)
// - PersistentPreRunE checks all prerequisites before command execution
//
// This matches bash scripts behavior (check_prerequisites function) but with
// better modularity and extensibility.
package prerequisites

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Prerequisite represents a tool or condition that must be met before
// a command can execute. This is typically a binary that must exist in PATH,
// but could be extended to check other conditions (network connectivity,
// file existence, etc.).
type Prerequisite interface {
	// Name returns the human-readable name of the prerequisite
	Name() string

	// Check verifies if the prerequisite is satisfied
	// Returns nil if OK, descriptive error with install instructions if not
	Check(ctx context.Context) error

	// Required indicates if this prerequisite is mandatory
	// Currently all prerequisites are required, but this allows for
	// future optional prerequisites with warnings instead of errors
	Required() bool
}

// BinaryPrerequisite checks if a binary exists in PATH.
// This is the most common type of prerequisite check.
//
// Example:
//
//	terraform := &BinaryPrerequisite{
//	    name:        "terraform",
//	    binaryName:  "terraform",
//	    installHint: "brew install terraform",
//	    required:    true,
//	}
type BinaryPrerequisite struct {
	name        string // Human-readable name (e.g., "Terraform")
	binaryName  string // Binary to look for in PATH (e.g., "terraform")
	installHint string // Installation instructions
	required    bool   // Whether this is mandatory
}

// Name returns the prerequisite name.
func (b *BinaryPrerequisite) Name() string {
	return b.name
}

// Required returns whether this prerequisite is mandatory.
func (b *BinaryPrerequisite) Required() bool {
	return b.required
}

// Check verifies the binary exists in PATH.
// Uses exec.LookPath which searches PATH environment variable.
// This is equivalent to bash: command -v <binary>
func (b *BinaryPrerequisite) Check(ctx context.Context) error {
	_, err := exec.LookPath(b.binaryName)
	if err != nil {
		return fmt.Errorf(
			"%s not found in PATH\n    Install: %s",
			b.name, b.installHint,
		)
	}
	return nil
}

// Pre-defined prerequisites for common tools.
// These are package-level variables that can be referenced by commands.
//
// Command-specific tools:
var (
	// Terraform is required for infrastructure provisioning (infra commands)
	Terraform = &BinaryPrerequisite{
		name:        "Terraform",
		binaryName:  "terraform",
		installHint: "brew install terraform",
		required:    true,
	}

	// Kubectl is required for Kubernetes operations (platform, workloads, backup)
	Kubectl = &BinaryPrerequisite{
		name:        "kubectl",
		binaryName:  "kubectl",
		installHint: "brew install kubectl",
		required:    true,
	}

	// NOTE: Talosctl is NOT required!
	// We use the Talos Go SDK (github.com/siderolabs/talos/pkg/machinery)
	// which is compiled into the binary. No external talosctl binary needed.

	// Velero is required for backup operations (backup commands)
	Velero = &BinaryPrerequisite{
		name:        "Velero",
		binaryName:  "velero",
		installHint: "brew install velero",
		required:    true,
	}
)

// Cloud-specific tools:
var (
	// Gcloud is required for GCP operations, specifically for:
	// - Setting up Application Default Credentials: gcloud auth application-default login
	// - IAP tunneling for cluster access
	Gcloud = &BinaryPrerequisite{
		name:        "gcloud",
		binaryName:  "gcloud",
		installHint: "https://cloud.google.com/sdk/docs/install",
		required:    true,
	}

	// Future cloud-specific tools will be added here:
	// StackitCLI for STACKIT cloud
	// AzureCLI for Azure
	// etc.
)

// CommandPrereqs maps top-level command names to their required tools.
// This is a declarative way for commands to specify their dependencies.
//
// Example: "deploy-infra" commands need terraform only (Talos uses Go SDK, no binary)
//
// Note: This only includes command-specific tools, not cloud-specific tools
// (those are in CloudPrereqs and checked separately based on --cloud flag)
var CommandPrereqs = map[string][]Prerequisite{
	"deploy-infra": {Terraform}, // Talos uses Go SDK, no talosctl binary needed
	"platform":     {Kubectl},
	"workloads":    {Kubectl},
	"backup":       {Kubectl, Velero},
	"restore":      {Kubectl, Velero},
}

// CloudPrereqs maps cloud provider names to their required tools.
// These are checked in addition to command prerequisites when --cloud flag is set.
//
// Example: If --cloud gcp is set, gcloud must be installed
var CloudPrereqs = map[string][]Prerequisite{
	"gcp": {Gcloud},
	// Future:
	// "stackit": {StackitCLI},
	// "azure":   {AzureCLI},
}

// GetCommandPrereqs returns prerequisites for the given command name.
// Returns nil if command has no specific prerequisites.
//
// Example: GetCommandPrereqs("infra") returns {Terraform}
//
// Parameters:
//   - cmdName: Top-level command name (e.g., "infra", "platform")
//
// Returns: Slice of prerequisites, or nil if command has no prerequisites
func GetCommandPrereqs(cmdName string) []Prerequisite {
	return CommandPrereqs[cmdName]
}

// GetCloudPrereqs returns prerequisites for a cloud provider.
// Returns nil if cloud is empty string or has no specific prerequisites.
//
// Example: GetCloudPrereqs("gcp") returns {Gcloud}
//
// Parameters:
//   - cloudName: Cloud provider name from --cloud flag
//
// Returns: Slice of prerequisites, or nil if no cloud-specific prerequisites
func GetCloudPrereqs(cloudName string) []Prerequisite {
	if cloudName == "" {
		return nil
	}
	return CloudPrereqs[cloudName]
}

// CheckAll verifies all given prerequisites are satisfied.
// If any prerequisite fails, returns an error listing ALL missing prerequisites.
// This allows the user to see everything they need to install at once,
// rather than fixing one issue at a time.
//
// Example output:
//
//	missing prerequisites:
//	  ✗ Terraform not found in PATH
//	    Install: brew install terraform
//	  ✗ gcloud not found in PATH
//	    Install: https://cloud.google.com/sdk/docs/install
//
// Parameters:
//   - ctx: Context for cancellation
//   - prereqs: Variadic list of prerequisites to check
//
// Returns: nil if all prerequisites are satisfied, error with details if any are missing
func CheckAll(ctx context.Context, prereqs ...Prerequisite) error {
	if len(prereqs) == 0 {
		return nil
	}

	var missing []string

	for _, prereq := range prereqs {
		if err := prereq.Check(ctx); err != nil {
			// Format: "  ✗ <error message>"
			missing = append(missing, fmt.Sprintf("  ✗ %s", err.Error()))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"missing prerequisites:\n%s",
			strings.Join(missing, "\n"),
		)
	}

	return nil
}
