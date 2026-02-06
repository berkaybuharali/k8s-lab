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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
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

	// dynamicClient is the dynamic Kubernetes client for generic operations
	dynamicClient dynamic.Interface

	// discoveryClient is used to discover server resources
	discoveryClient discovery.DiscoveryInterface

	// restMapper maps GVK to GVR, cached for performance
	restMapper meta.RESTMapper

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

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create REST mapper
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	log.Debug("Kubernetes client created successfully")

	return &Client{
		clientset:       clientset,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
		restMapper:      mapper,
		log:             log,
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

// ApplyManifest applies a Kubernetes manifest file to the cluster.
//
// This method is generic and can handle any Kubernetes resource type.
// It uses Server-Side Apply (SSA) logic.
//
// It parses the file (which can contain multiple documents separated by "---"),
// discovers the resource type using the cached RESTMapper, and applies it
// using the dynamic client.
//
// Parameters:
//   - ctx: Context for cancellation
//   - manifestPath: Absolute path to the YAML manifest file
//
// Returns:
//   - error: If reading, parsing, discovery, or application fails
func (c *Client) ApplyManifest(ctx context.Context, manifestPath string) error {
	c.log.Debug("Applying manifest: %s", manifestPath)

	// Check if file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest file not found: %s", manifestPath)
	}

	// Read file
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Decode YAML (handle multiple documents)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)

	for {
		var rawObj unstructured.Unstructured
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode YAML: %w", err)
		}

		if len(rawObj.Object) == 0 {
			continue // Skip empty documents
		}

		// Get GroupVersionKind
		gvk := rawObj.GroupVersionKind()
		
		// Find GVR (GroupVersionResource) mapping using cached mapper
		mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("failed to find mapping for %s: %w", gvk.String(), err)
		}

		// Prepare resource interface
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if rawObj.GetNamespace() == "" {
				return fmt.Errorf("resource %s/%s is namespaced but has no namespace defined", gvk.Kind, rawObj.GetName())
			}
			dr = c.dynamicClient.Resource(mapping.Resource).Namespace(rawObj.GetNamespace())
		} else {
			dr = c.dynamicClient.Resource(mapping.Resource)
		}

		// Apply using Server-Side Apply
		// We use "k8s-lab" as the field manager
		data, err := rawObj.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal object: %w", err)
		}

		c.log.Debug("Applying %s/%s (%s)...", rawObj.GetNamespace(), rawObj.GetName(), gvk.Kind)

		// REMOVED: Force: ptr(true) - preventing dangerous overrides
		_, err = dr.Patch(ctx, rawObj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "k8s-lab",
		})
		
		if err != nil {
			return fmt.Errorf("failed to apply %s/%s: %w", gvk.Kind, rawObj.GetName(), err)
		}
	}

	return nil
}

// WaitForDeploymentReady waits for a Deployment to be fully available.
//
// It matches kubectl rollout status logic by checking:
// - Status.Conditions["Available"] == True
// - Status.Conditions["Progressing"] != False (failed)
// - Status.UpdatedReplicas == Spec.Replicas
//
// Parameters:
//   - ctx: Context for cancellation
//   - namespace: Kubernetes namespace
//   - name: Deployment name
//   - timeout: Maximum time to wait
//
// Returns:
//   - error: If timeout reached, deployment fails, or progress deadline exceeded
func (c *Client) WaitForDeploymentReady(ctx context.Context, namespace, name string, timeout time.Duration) error {
	c.log.Info("Waiting for Deployment %s/%s to be ready (timeout: %v)...", namespace, name, timeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment %s/%s: %w", namespace, name, ctx.Err())

		case <-ticker.C:
			deploy, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				c.log.Debug("Failed to get deployment: %v", err)
				continue
			}

			if deploy.Spec.Replicas == nil {
				continue // Should not happen for standard deployments
			}
			desiredReplicas := *deploy.Spec.Replicas

			// Check conditions
			isAvailable := false
			isFailed := false

			for _, cond := range deploy.Status.Conditions {
				if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
					isAvailable = true
				}
				if cond.Type == appsv1.DeploymentProgressing && cond.Status == corev1.ConditionFalse {
					isFailed = true
					c.log.Warn("Deployment %s paused/failed: %s", name, cond.Reason)
				}
			}

			c.log.Debug("Deployment %s: Available=%v, Failed=%v, Replicas=%d/%d, Updated=%d",
				name, isAvailable, isFailed, deploy.Status.AvailableReplicas, desiredReplicas, deploy.Status.UpdatedReplicas)

			// Success criteria:
			// 1. Available condition is True
			// 2. Updated replicas match desired
			// 3. Available replicas match desired (fully rolled out)
			// 4. No explicit failure
			if isAvailable && !isFailed &&
				deploy.Status.UpdatedReplicas == desiredReplicas &&
				deploy.Status.AvailableReplicas == desiredReplicas {
				c.log.Info("Deployment %s is ready", name)
				return nil
			}

			if isFailed {
				return fmt.Errorf("deployment %s failed to progress", name)
			}
		}
	}
}

// CheckStorageClassExists checks if a StorageClass exists in the cluster.
// This is used as a prerequisite check for stateful applications.
func (c *Client) CheckStorageClassExists(ctx context.Context) (bool, error) {
	list, err := c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list storage classes: %w", err)
	}
	return len(list.Items) > 0, nil
}

