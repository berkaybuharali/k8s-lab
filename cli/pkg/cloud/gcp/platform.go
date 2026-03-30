// cli/pkg/cloud/gcp/platform.go
// Package gcp implements platform tools installation for GCP.
//
// This file handles installation of cloud-specific platform components:
// - GCE Persistent Disk CSI driver
// - Velero backup configuration
package gcp

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/velero"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// GCPCSIDriverOverlay is the kustomize overlay for GCE PD CSI driver.
	// Using 'noauth' overlay - relies on VM's service account (no workload identity).
	GCPCSIDriverOverlay = "https://github.com/kubernetes-sigs/gcp-compute-persistent-disk-csi-driver/deploy/kubernetes/overlays/noauth?ref=v1.16.1"

	// GCPCSINamespace is the namespace where CSI driver is installed.
	GCPCSINamespace = "gce-pd-csi-driver"

	// GCPVeleroPlugin is the Velero plugin image for GCP.
	GCPVeleroPlugin = "velero/velero-plugin-for-gcp:v1.11.0"
)

// InstallCSIDriver installs the GCE Persistent Disk CSI driver.
//
// This enables dynamic provisioning of GCE persistent disks for Kubernetes PVCs.
// The driver is installed via kubectl kustomize and patched for Talos compatibility.
//
// Talos compatibility issue:
// - Talos has read-only /etc filesystem
// - /etc/udev does not exist on Talos
// - Upstream CSI driver mounts /etc/udev as hostPath (fails on Talos)
// - We patch it to use emptyDir instead (driver only needs /lib/udev)
//
func (p *Provider) InstallCSIDriver(ctx context.Context, kubeconfigPath string) error {
	p.log.Step("Installing GCE PD CSI driver")

	// Create Kubernetes clientset
	clientset, err := p.createClientset(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Check if already installed
	if p.isCSIInstalled(ctx, clientset) {
		p.log.Info("GCE PD CSI driver already running")
		return nil
	}

	// Create namespace with privileged label (CSI drivers need host access)
	p.log.Info("Creating %s namespace with privileged policy", GCPCSINamespace)
	if err := p.createCSINamespace(ctx, clientset); err != nil {
		return fmt.Errorf("failed to create CSI namespace: %w", err)
	}

	// Apply CSI driver using kustomize (kubectl apply -k)
	// This is one place where we shell out because kubectl's built-in kustomize
	// is simpler than using the kustomize Go library
	p.log.Info("Applying CSI driver manifests")
	if err := p.applyCSIDriver(ctx, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to apply CSI driver: %w", err)
	}

	// Patch for Talos compatibility
	p.log.Info("Patching CSI node DaemonSet for Talos (read-only /etc/udev)")
	if err := p.patchCSIForTalos(ctx, clientset); err != nil {
		return fmt.Errorf("failed to patch CSI driver: %w", err)
	}

	// Wait for controller readiness
	p.log.Info("Waiting for CSI driver controller to be ready")
	if err := p.waitForDeployment(ctx, clientset, GCPCSINamespace, "csi-gce-pd-controller", 180*time.Second); err != nil {
		return fmt.Errorf("CSI controller not ready: %w", err)
	}

	// Wait for node driver readiness
	p.log.Info("Waiting for CSI driver nodes to be ready")
	if err := p.waitForDaemonSet(ctx, clientset, GCPCSINamespace, "csi-gce-pd-node", 180*time.Second); err != nil {
		return fmt.Errorf("CSI node driver not ready: %w", err)
	}

	p.log.Info("GCE PD CSI driver installed")
	return nil
}

// createClientset creates a Kubernetes clientset from kubeconfig.
func (p *Provider) createClientset(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, nil
}

// isCSIInstalled checks if CSI driver is already running.
func (p *Provider) isCSIInstalled(ctx context.Context, clientset *kubernetes.Clientset) bool {
	deployment, err := clientset.AppsV1().Deployments(GCPCSINamespace).Get(ctx, "csi-gce-pd-controller", metav1.GetOptions{})
	if err != nil {
		return false // Deployment doesn't exist
	}

	return deployment.Status.ReadyReplicas > 0
}

// createCSINamespace creates the CSI driver namespace with privileged label.
func (p *Provider) createCSINamespace(ctx context.Context, clientset *kubernetes.Clientset) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: GCPCSINamespace,
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "privileged",
			},
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		// Ignore already exists error
		if !isAlreadyExistsError(err) {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	return nil
}

// applyCSIDriver applies the CSI driver manifests via kustomize.
// This shells out to kubectl because kubectl's built-in kustomize is simpler
// than using the kustomize Go library for remote overlays.
func (p *Provider) applyCSIDriver(ctx context.Context, kubeconfigPath string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "apply",
		"-k", GCPCSIDriverOverlay,
		"--kubeconfig", kubeconfigPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		p.log.Debug("kubectl output:\n%s", string(output))
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	return nil
}

// patchCSIForTalos patches the CSI node DaemonSet for Talos compatibility.
//
// Talos issue: /etc/udev doesn't exist (read-only /etc filesystem).
// Upstream CSI driver tries to mount /etc/udev as hostPath type:Directory.
// This fails because containerd requires the host path to exist.
//
// Solution: Replace /etc/udev mount with emptyDir.
// The driver only needs /lib/udev for udev rules anyway.
//
//	kubectl patch daemonset csi-gce-pd-node -n gce-pd-csi-driver --type=json -p='[
//	  {"op": "test", "path": "/spec/template/spec/volumes/4/name", "value": "udev-rules-etc"},
//	  {"op": "replace", "path": "/spec/template/spec/volumes/4", "value": {"name": "udev-rules-etc", "emptyDir": {}}}
//	]'
func (p *Provider) patchCSIForTalos(ctx context.Context, clientset *kubernetes.Clientset) error {
	// JSON patch to replace /etc/udev hostPath with emptyDir
	patch := []byte(`[
  {"op": "test", "path": "/spec/template/spec/volumes/4/name", "value": "udev-rules-etc"},
  {"op": "replace", "path": "/spec/template/spec/volumes/4", "value": {"name": "udev-rules-etc", "emptyDir": {}}}
]`)

	_, err := clientset.AppsV1().DaemonSets(GCPCSINamespace).Patch(
		ctx,
		"csi-gce-pd-node",
		types.JSONPatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch daemonset: %w", err)
	}

	return nil
}

// waitForDeployment waits for a deployment to have ready replicas.
func (p *Provider) waitForDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	watcher, err := clientset.AppsV1().Deployments(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("failed to watch deployment: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment %s/%s", namespace, name)

		case event := <-watcher.ResultChan():
			if event.Type == watch.Error {
				return fmt.Errorf("watch error for deployment %s/%s", namespace, name)
			}

			if event.Type == watch.Added || event.Type == watch.Modified {
				deployment, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					continue
				}

				if deployment.Status.ReadyReplicas > 0 && deployment.Status.ReadyReplicas >= *deployment.Spec.Replicas {
					return nil
				}
			}
		}
	}
}

// waitForDaemonSet waits for a DaemonSet to have ready replicas on all nodes.
func (p *Provider) waitForDaemonSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	watcher, err := clientset.AppsV1().DaemonSets(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("failed to watch daemonset: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for daemonset %s/%s", namespace, name)

		case event := <-watcher.ResultChan():
			if event.Type == watch.Error {
				return fmt.Errorf("watch error for daemonset %s/%s", namespace, name)
			}

			if event.Type == watch.Added || event.Type == watch.Modified {
				ds, ok := event.Object.(*appsv1.DaemonSet)
				if !ok {
					continue
				}

				// DaemonSet is ready when number ready equals desired number
				if ds.Status.NumberReady > 0 && ds.Status.NumberReady >= ds.Status.DesiredNumberScheduled {
					return nil
				}
			}
		}
	}
}

// isAlreadyExistsError checks if error is "already exists".
func isAlreadyExistsError(err error) bool {
	return err != nil && (
	// K8s errors contain "already exists" in the message
	contains(err.Error(), "already exists") ||
		contains(err.Error(), "AlreadyExists"))
}

// contains checks if string contains substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}

// GetVeleroInstallConfig returns GCP-specific Velero configuration.
//
// This reads Terraform outputs to build the Velero install config:
// - state_bucket: GCS bucket for backups (same as Terraform state bucket)
// - project_id: GCP project ID
// - node_service_account_email: Service account for GCS access
//
//	bucket=$(terraform output -raw state_bucket)
//	project=$(terraform output -raw project_id)
//	sa=$(terraform output -raw node_service_account_email)
//	velero install --provider gcp --bucket "$bucket" ...
func (p *Provider) GetVeleroInstallConfig(terraformDir string) (interface{}, error) {
	// Read bucket name (same bucket as Terraform state)
	bucketName, err := p.getTerraformOutput(terraformDir, "state_bucket")
	if err != nil {
		return nil, fmt.Errorf("failed to get state_bucket: %w", err)
	}

	// Read project ID
	projectID, err := p.GetProjectID(terraformDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get project_id: %w", err)
	}

	// Read service account email (optional - may not exist in outputs)
	saEmail, _ := p.getTerraformOutput(terraformDir, "node_service_account_email")

	return &velero.InstallConfig{
		Provider:            "gcp",
		BucketName:          bucketName,
		ProjectID:           projectID,
		ServiceAccountEmail: saEmail,
		PluginImage:         GCPVeleroPlugin,
		UseWorkloadIdentity: true, // GCP always uses workload identity (--no-secret)
	}, nil
}

// getTerraformOutput reads a Terraform output value.
// Helper method to avoid duplication.
func (p *Provider) getTerraformOutput(terraformDir, outputName string) (string, error) {
	cmd := exec.Command("terraform", "output", "-raw", outputName)
	cmd.Dir = terraformDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("terraform output failed: %w", err)
	}

	return string(output), nil
}
