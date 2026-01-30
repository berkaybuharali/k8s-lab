// cli/pkg/talos/config.go
// Configuration generation using Talos SDK.
//
// This file handles machine configuration generation using the
// github.com/siderolabs/talos/pkg/machinery/config/generate package.
//
// The generate package provides a builder-pattern API for creating
// Talos machine configurations programmatically. This is more powerful
// than shelling to talosctl, as we can:
// - Apply configuration patches programmatically
// - Customize per-node settings easily
// - Generate configs for different machine types (control plane, worker)
// - Reuse secrets bundle across multiple nodes
package talos

// ConfigOption is a function that modifies config generation options.
// This follows the functional options pattern, allowing callers to
// customize config generation without a huge parameter list.
//
// Example usage:
//
//	client.GenerateConfigs(ctx, "my-cluster", "https://10.0.0.1:6443",
//	    talos.WithAdditionalSANs([]string{"localhost"}),
//	    talos.WithInstallDisk("/dev/sda"),
//	)
type ConfigOption func(*configOptions)

// configOptions holds all options for config generation.
// These map to talosctl gen config flags and SDK generate.Input options.
//
// Fields are set via ConfigOption functions (functional options pattern).
type configOptions struct {
	// additionalSANs are extra Subject Alternative Names for API server cert
	// Required for IAP tunnel access: must include "localhost"
	additionalSANs []string

	// installDisk specifies which disk to install Talos on
	// Default from SDK: auto-detected
	// Example: "/dev/sda"
	installDisk string

	// kubernetesVersion is the K8s version to install
	// Default from SDK: latest stable
	// Example: "v1.29.0"
	kubernetesVersion string

	// TODO: Add more options as needed in step 4c:
	// - patchFiles []string (cloud-specific config patches)
	// - installImage string (Talos installer image)
	// - dnsDomain string (cluster DNS domain)
}

// WithAdditionalSANs adds Subject Alternative Names to the API server certificate.
//
// This is required for IAP tunnel access - we must include "localhost"
// so the certificate is valid when accessing via tunnel.
//
// Equivalent to: talosctl gen config --additional-sans localhost
//
// Example:
//
//	WithAdditionalSANs([]string{"localhost", "my-cluster.example.com"})
func WithAdditionalSANs(sans []string) ConfigOption {
	return func(opts *configOptions) {
		opts.additionalSANs = sans
	}
}

// WithInstallDisk specifies which disk to install Talos on.
//
// Equivalent to: talosctl gen config with install.disk patch
//
// Example:
//
//	WithInstallDisk("/dev/sda")
func WithInstallDisk(disk string) ConfigOption {
	return func(opts *configOptions) {
		opts.installDisk = disk
	}
}

// WithKubernetesVersion specifies the Kubernetes version to install.
//
// Equivalent to: talosctl gen config --kubernetes-version <version>
//
// Example:
//
//	WithKubernetesVersion("v1.29.0")
func WithKubernetesVersion(version string) ConfigOption {
	return func(opts *configOptions) {
		opts.kubernetesVersion = version
	}
}

// applyOptions applies ConfigOption functions to create final options.
// This is called internally by GenerateConfigs to build the configOptions.
func applyOptions(opts ...ConfigOption) *configOptions {
	result := &configOptions{
		// Defaults
		additionalSANs: []string{"localhost"}, // Required for IAP tunnel
	}

	for _, opt := range opts {
		opt(result)
	}

	return result
}
