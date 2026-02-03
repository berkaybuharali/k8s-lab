// cli/pkg/k8s/client.go
// Package k8s provides a wrapper around Kubernetes client-go library.
//
// This package provides high-level operations for interacting with Kubernetes
// clusters, using the official client-go library. It wraps common operations
// like node verification and waiting for cluster readiness.
//
// The package is cloud-agnostic - it only requires a kubeconfig file.
// Cloud-specific tunnel/VPN/access management is handled by the cloud provider
// package (via CreateK8sEndpoint), which ensures the kubeconfig points to
// the correct endpoint.
package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

// Client wraps the Kubernetes client-go clientset.
// It provides high-level operations for cluster management.
//
// The client is initialized from a kubeconfig file, which contains:
// - Cluster endpoint (e.g., https://localhost:6443 via tunnel)
// - Authentication credentials (admin cert from Talos)
// - Cluster CA certificate
type Client struct {
	// clientset is the Kubernetes API client
	clientset *kubernetes.Clientset

	// log is the logger for user-facing messages
	log *logger.Logger
}

// NewClient creates a new Kubernetes client from kubeconfig file.
//
// The kubeconfig must be valid and the cluster must be accessible.
// For cloud environments with tunnels, ensure the tunnel is active
// before creating the client and keep it alive while using the client.
//
// Parameters:
//   - kubeconfigPath: Absolute path to kubeconfig file (from FetchKubeconfig)
//   - log: Logger instance for output
//
// Returns:
//   - *Client: Kubernetes client instance
//   - error: If kubeconfig is invalid or cluster is unreachable
//
// Example usage (in command code):
//
//	// Create K8s tunnel
//	k8sEndpoint, cleanup, _ := provider.CreateK8sEndpoint(ctx, cpInstance, zone, projectID)
//	defer cleanup()  // Keep tunnel alive
//
//	// Create K8s client (kubeconfig already points to tunnel endpoint)
//	k8sClient, err := k8s.NewClient(kubeconfigPath, log)
//	if err != nil {
//	    return fmt.Errorf("failed to create k8s client: %w", err)
//	}
//
//	// Use client (tunnel is active)
//	nodes, _ := k8sClient.GetNodes(ctx)
//
//	// cleanup() called via defer - tunnel closed
//
// Equivalent to bash: kubectl --kubeconfig=<path>
func NewClient(kubeconfigPath string, log *logger.Logger) (*Client, error) {
	log.Debug("Creating Kubernetes client from kubeconfig: %s", kubeconfigPath)

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	log.Debug("Kubernetes client created successfully")

	return &Client{
		clientset: clientset,
		log:       log,
	}, nil
}

// GetNodes returns a list of node names in the cluster.
//
// This is useful for verifying cluster accessibility and checking
// which nodes are present.
//
// Parameters:
//   - ctx: Context for cancellation/timeout
//
// Returns:
//   - []string: List of node names
//   - error: If API is unreachable or request fails
//
// Example:
//
//	nodes, err := k8sClient.GetNodes(ctx)
//	if err != nil {
//	    return fmt.Errorf("cluster not accessible: %w", err)
//	}
//	log.Info("Found %d nodes: %v", len(nodes), nodes)
//
// Equivalent to bash: kubectl get nodes --no-headers | awk '{print $1}'
func (c *Client) GetNodes(ctx context.Context) ([]string, error) {
	c.log.Debug("Fetching cluster nodes...")

	nodeList, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	names := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		names[i] = node.Name
	}

	c.log.Debug("Found %d nodes", len(names))
	return names, nil
}

// WaitForNodesReady waits for the expected number of nodes to be in Ready state.
//
// This polls the cluster until:
// - The expected number of nodes are Ready, OR
// - The timeout is reached
//
// A node is considered Ready when it has the Ready condition with status True.
//
// Parameters:
//   - ctx: Context for cancellation (independent of timeout parameter)
//   - expectedCount: Number of nodes to wait for (control plane + workers)
//   - timeout: Maximum time to wait (recommended: 5-10 minutes)
//
// Returns:
//   - error: If timeout reached or context cancelled
//
// Example:
//
//	// Wait for 3 nodes (1 CP + 2 workers) for up to 10 minutes
//	err := k8sClient.WaitForNodesReady(ctx, 3, 10*time.Minute)
//	if err != nil {
//	    return fmt.Errorf("nodes not ready: %w", err)
//	}
//
// Equivalent to bash: talos_wait_for_all_nodes() in scripts/lib/talos.sh:310-336
func (c *Client) WaitForNodesReady(ctx context.Context, expectedCount int, timeout time.Duration) error {
	c.log.Info("Waiting for %d nodes to be Ready (timeout: %v)...", expectedCount, timeout)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	attempt := 1

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for nodes: %w", ctx.Err())

		case <-ticker.C:
			c.log.Debug("Checking node readiness (attempt %d)...", attempt)

			// Get all nodes
			nodeList, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				c.log.Debug("Failed to list nodes: %v", err)
				attempt++
				continue
			}

			// Count Ready nodes
			readyCount := 0
			for _, node := range nodeList.Items {
				if isNodeReady(&node) {
					readyCount++
				}
			}

			c.log.Debug("Nodes ready: %d/%d", readyCount, expectedCount)

			if readyCount >= expectedCount {
				c.log.Info("All %d nodes are Ready", expectedCount)
				return nil
			}

			attempt++
		}
	}
}

// isNodeReady checks if a node has the Ready condition set to True.
//
// A node is Ready when:
// - It has a condition with Type=Ready
// - That condition has Status=True
//
// This matches kubectl's definition of node readiness.
func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
