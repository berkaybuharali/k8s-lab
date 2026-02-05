// cli/pkg/cloud/provider.go
// Package cloud defines the interface for cloud provider implementations.
// Each cloud (GCP, STACKIT, etc.) implements this interface to provide
// cloud-specific operations like authentication validation, bucket management,
// and configuration reading.
//
// The Provider interface enables the CLI to work with different clouds without
// changing command logic. This is similar to how bash scripts source different
// cloud-specific modules (scripts/lib/gcp/*.sh, scripts/lib/stackit/*.sh).
package cloud

import (
	"context"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

// Provider is the interface that all cloud providers must implement.
// This abstraction allows the CLI to work with any cloud provider by
// implementing a consistent set of operations.
//
// Design principle: Methods should be cloud-agnostic operations that
// make sense for any cloud. Cloud-specific details are hidden in the
// implementation, not exposed in the interface.
type Provider interface {
	// Name returns the cloud provider identifier (e.g., "gcp", "stackit").
	// This matches the value passed to the --cloud flag.
	//
	// Example: "gcp" for Google Cloud Platform
	Name() string

	// SetLogger sets the logger for this provider.
	// This is called by root command after provider creation.
	// Logger is MANDATORY for all operations.
	SetLogger(log *logger.Logger)

	// Validate checks if cloud-specific prerequisites are met.
	// This includes:
	// - Cloud CLI tools are installed (gcloud, openstack, etc.)
	// - Authentication is configured
	// - Required permissions are available (if checkable)
	//
	// Returns an error with actionable message if validation fails.
	// Equivalent to bash: check_prerequisites() for cloud-specific tools
	//
	// Context can be used for timeout or cancellation.
	Validate(ctx context.Context) error

	// EnsureStateBucket creates the Terraform state storage bucket if it doesn't exist.
	// Terraform needs remote state storage for team collaboration and state locking.
	//
	// Parameters:
	//   - bucketName: Name of the storage bucket (from terraform.tfvars: state_bucket)
	//   - projectID: Cloud project/account identifier (from terraform.tfvars: project_id)
	//
	// Implementation should:
	// 1. Check if bucket exists
	// 2. Create if it doesn't exist
	// 3. Apply appropriate security settings (private access, encryption)
	// 4. Return nil if bucket already exists (idempotent)
	//
	// Equivalent to bash: tf_ensure_state_bucket()
	EnsureStateBucket(ctx context.Context, bucketName, projectID string) error

	// GetProjectID reads the cloud project/account ID from terraform.tfvars.
	// Different clouds call this different things:
	// - GCP: project_id
	// - AWS: account_id
	// - STACKIT: project_id
	//
	// Parameters:
	//   - terraformDir: Absolute path to Terraform directory containing terraform.tfvars
	//
	// Returns the project/account ID string, or error if not found or unreadable.
	// Equivalent to bash: tf_get_project_id()
	GetProjectID(terraformDir string) (string, error)

	// CreateTalosEndpoint creates access to Talos API (port 50000).
	// Returns endpoint string, cleanup function, and error.
	// Caller must defer cleanup().
	//
	// GCP: Creates IAP tunnel, returns "localhost:50000"
	// AWS: Creates SSM session
	// Direct-access: Returns "ip:50000"
	CreateTalosEndpoint(ctx context.Context, instance, zone, projectID string) (string, func(), error)

	// CreateK8sEndpoint creates access to Kubernetes API (port 6443).
	// Same pattern as CreateTalosEndpoint but for K8s API.
	CreateK8sEndpoint(ctx context.Context, instance, zone, projectID string) (string, func(), error)

	// InstallCSIDriver installs the cloud-specific CSI driver for persistent storage.
	// The CSI driver enables dynamic provisioning of persistent disks.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - kubeconfigPath: Path to kubeconfig file for cluster access
	//
	// Returns error if installation fails.
	// Should be idempotent (safe to call if already installed).
	//
	// Equivalent to bash: gcp_csi_install() in scripts/lib/gcp/csi.sh
	InstallCSIDriver(ctx context.Context, kubeconfigPath string) error

	// GetVeleroInstallConfig returns cloud-specific Velero configuration.
	// This reads Terraform outputs to build the Velero install config.
	//
	// Parameters:
	//   - terraformDir: Path to Terraform directory
	//
	// Returns InstallConfig with cloud-specific settings (bucket, plugin, etc.)
	//
	// Equivalent to bash: Building args for gcp_velero_install()
	GetVeleroInstallConfig(terraformDir string) (interface{}, error)
}

// Registry holds all registered cloud providers.
// Providers register themselves by calling Register() in their init() function.
// This allows automatic discovery of available providers without hardcoding
// a list in the main package.
//
// Example:
//   gcp.Provider registers as "gcp"
//   stackit.Provider registers as "stackit"
//
// Usage:
//   provider := cloud.Get("gcp")
var Registry = make(map[string]Provider)

// Register adds a provider to the global registry.
// This should be called by each provider's init() function.
//
// Example from gcp/provider.go:
//   func init() {
//       cloud.Register("gcp", &Provider{})
//   }
//
// Panics if a provider with the same name is already registered
// (indicates programming error - two providers claiming same name).
func Register(name string, provider Provider) {
	if _, exists := Registry[name]; exists {
		panic("cloud provider already registered: " + name)
	}
	Registry[name] = provider
}

// Get retrieves a provider by name from the registry.
// Returns nil if provider not found (caller should check).
//
// Example:
//   provider := cloud.Get("gcp")
//   if provider == nil {
//       return fmt.Errorf("cloud provider 'gcp' not available")
//   }
func Get(name string) Provider {
	return Registry[name]
}

// List returns all registered provider names.
// Useful for validation and help messages.
//
// Example output: ["gcp", "stackit"]
func List() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}
