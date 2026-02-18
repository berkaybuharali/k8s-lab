// cli/cmd/deployment_context.go
// DeploymentContext is a carrier for initialized clients and shared state
// across the infrastructure deployment pipeline.
//
// Instead of passing ctx, cfg, provider, and log as individual parameters
// to every sub-function in deploy_infra.go, callers build a DeploymentContext
// once and pass it through.
//
// This is a demonstration of the pattern (Item 5.13). Full threading through
// all sub-functions is left as a follow-up; bootstrapKubernetes and
// verifyClusterReady show the before/after comparison.
package cmd

import (
	"context"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/cloud"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/config"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/talos"
)

// DeploymentContext holds initialized clients and shared configuration for
// a single deployment run. Build it once in the top-level command and pass
// it to sub-functions to avoid repeated initialization and parameter lists.
type DeploymentContext struct {
	// Ctx is the root context for the deployment operation.
	Ctx context.Context

	// Config is the CLI configuration (paths, cluster name, cloud).
	Config *config.Config

	// Provider is the resolved cloud provider implementation.
	Provider cloud.Provider

	// TalosClient is a connected Talos SDK client.
	// May be nil if the Talos phase has not yet started.
	TalosClient *talos.Client

	// Infra holds the Terraform output values populated after provisioning.
	// May be nil before the infra phase completes.
	Infra *InfrastructureInfo

	// Log is the shared logger for the deployment run.
	Log *logger.Logger
}
