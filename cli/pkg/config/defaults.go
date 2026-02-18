// Package-level constants for well-known string literals that appear
// in multiple packages. Import this package instead of repeating literals.
package config

const (
	// VeleroNamespace is the Kubernetes namespace where Velero is installed.
	// Keep in sync with cli/pkg/velero/client.go:VeleroNamespace.
	VeleroNamespace = "velero"

	// ApplicationNamespace is the namespace for user workloads (NGINX, Redis, etc.)
	ApplicationNamespace = "application"

	// AgentsNamespace is the namespace for AI agent workloads.
	AgentsNamespace = "agents"

	// ClusterName is the default Kubernetes cluster name.
	ClusterName = "k8s-lab"
)
