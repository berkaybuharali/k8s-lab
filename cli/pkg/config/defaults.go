// Package-level constants for well-known string literals that appear
// in multiple packages. Import this package instead of repeating literals.
package config

const (
	// VeleroNamespace is the Kubernetes namespace where Velero is installed.
	// Keep in sync with cli/pkg/velero/client.go:VeleroNamespace.
	VeleroNamespace = "velero"

	// AgentsNamespace is the namespace for all agent workloads (Redis, Commerce, Supply Chain).
	AgentsNamespace = "agents"

	// ClusterName is the default Kubernetes cluster name.
	ClusterName = "k8s-lab"
)
