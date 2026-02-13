// cli/pkg/velero/client.go
// Package velero provides operations for Velero backup/restore management.
//
// This package handles:
// - Velero installation using the velero CLI binary
// - Backup creation using Kubernetes dynamic client with Velero CRDs
// - Restore operations using Kubernetes dynamic client with Velero CRDs
// - Backup/restore status monitoring
//
// Design Decision:
//   - Installation uses velero CLI binary (like terraform/gcloud pattern)
//     because Velero's Go SDK does not support installation
//   - Backup/restore use client-go dynamic client for pure Go operations
package velero

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

const (
	// VeleroNamespace is the namespace where Velero is installed
	VeleroNamespace = "velero"

	// DefaultBackupTTL is the default backup retention period (30 days)
	DefaultBackupTTL = 720 * time.Hour // 30 days
)

var (
	// BackupGVR is the GroupVersionResource for Velero Backups
	BackupGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backups",
	}

	// RestoreGVR is the GroupVersionResource for Velero Restores
	RestoreGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "restores",
	}
)

// Client wraps the Kubernetes and dynamic clients for Velero operations.
type Client struct {
	// dynamicClient is used for Velero CRD operations (Backup, Restore)
	dynamicClient dynamic.Interface

	// clientset is used for standard Kubernetes operations (deployments)
	clientset *kubernetes.Clientset

	// kubeconfigPath is the path to kubeconfig (for velero CLI commands)
	kubeconfigPath string

	// log is the logger for user-facing messages
	log *logger.Logger
}

// NewClient creates a new Velero client from kubeconfig file.
//
// Parameters:
//   - kubeconfigPath: Absolute path to kubeconfig file
//   - log: Logger instance for output
//
// Returns:
//   - *Client: Velero client instance
//   - error: If kubeconfig is invalid or cluster is unreachable
//
// Example usage:
//
//	veleroClient, err := velero.NewClient(kubeconfigPath, log)
//	if err != nil {
//	    return fmt.Errorf("failed to create velero client: %w", err)
//	}
func NewClient(kubeconfigPath string, log *logger.Logger) (*Client, error) {
	log.Debug("Creating Velero client from kubeconfig: %s", kubeconfigPath)

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create dynamic client for Velero CRDs
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create standard clientset for deployments
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	log.Debug("Velero client created successfully")

	return &Client{
		dynamicClient:  dynamicClient,
		clientset:      clientset,
		kubeconfigPath: kubeconfigPath,
		log:            log,
	}, nil
}

// InstallConfig contains configuration for Velero installation.
type InstallConfig struct {
	// Provider is the cloud provider (e.g., "gcp", "aws", "azure")
	Provider string

	// BucketName is the storage bucket for backups
	// From Terraform: terraform output -raw bucket_name
	BucketName string

	// ProjectID is the cloud project ID (for GCP)
	// From Terraform: terraform output -raw project_id
	ProjectID string

	// ServiceAccountEmail is the service account for cloud access
	// From Terraform: terraform output -raw node_service_account_email
	// Used for --backup-location-config serviceAccount=<email>
	ServiceAccountEmail string

	// PluginImage is the cloud-specific Velero plugin image
	// Example: "velero/velero-plugin-for-gcp:v1.11.0"
	PluginImage string

	// UseWorkloadIdentity indicates if using VM service account (no secret)
	// For GCP, this is always true (--no-secret flag)
	UseWorkloadIdentity bool
}

// Install installs Velero using the velero CLI binary.
//
// This shells out to the velero binary because Velero's Go SDK does not
// support installation operations. This follows the same pattern as
// terraform (terraform-exec wrapper) and gcloud (exec.Command).
//
// Parameters:
//   - ctx: Context for cancellation
//   - config: Installation configuration
//
// Returns:
//   - error: If installation fails
//
// Example:
//
//	config := &velero.InstallConfig{
//	    Provider: "gcp",
//	    BucketName: "my-backup-bucket",
//	    ProjectID: "my-project",
//	    ServiceAccountEmail: "velero@my-project.iam.gserviceaccount.com",
//	    PluginImage: "velero/velero-plugin-for-gcp:v1.11.0",
//	    UseWorkloadIdentity: true,
//	}
//	err := veleroClient.Install(ctx, config)
//
func (c *Client) Install(ctx context.Context, config *InstallConfig) error {
	c.log.Step("Installing Velero with %s plugin", config.Provider)

	// Check if already installed
	isInstalled, err := c.isInstalled(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if Velero is installed: %w", err)
	}
	if isInstalled {
		c.log.Info("Velero already running")
		return nil
	}

	c.log.Info("Using GCS bucket: %s (prefix: velero)", config.BucketName)
	if config.ServiceAccountEmail != "" {
		c.log.Info("Using service account: %s", config.ServiceAccountEmail)
	}

	// Build velero install command
	args := []string{
		"install",
		"--provider", config.Provider,
		"--plugins", config.PluginImage,
		"--bucket", config.BucketName,
		"--prefix", "velero",
		"--snapshot-location-config", fmt.Sprintf("project=%s", config.ProjectID),
		"--use-volume-snapshots=true",
		"--default-volumes-to-fs-backup=false",
		"--kubeconfig", c.kubeconfigPath,
	}

	// Add backup location config if service account provided
	if config.ServiceAccountEmail != "" {
		args = append(args, "--backup-location-config", fmt.Sprintf("serviceAccount=%s", config.ServiceAccountEmail))
	}

	// Add --no-secret flag for workload identity
	if config.UseWorkloadIdentity {
		args = append(args, "--no-secret")
	}

	c.log.Debug("Running: velero %s", strings.Join(args, " "))

	// Execute velero install
	cmd := exec.CommandContext(ctx, "velero", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.log.Debug("velero install output:\n%s", string(output))
		return fmt.Errorf("velero install failed: %w\nOutput: %s", err, string(output))
	}

	c.log.Debug("velero install output:\n%s", string(output))

	// Patch Velero deployment to increase client QPS/burst for IAP tunnel latency
	// This helps with high-latency connections like IAP tunnels
	c.log.Info("Configuring Velero server QPS/burst limits")
	if err := c.patchVeleroQPS(ctx); err != nil {
		// Non-fatal - log warning and continue
		c.log.Warn("Failed to patch Velero QPS limits: %v", err)
	}

	// Wait for Velero to be ready
	if err := c.WaitForReady(ctx, 2*time.Minute); err != nil {
		return fmt.Errorf("Velero did not become ready: %w", err)
	}

	c.log.Info("Velero installed with %s plugin", config.Provider)
	return nil
}

// patchVeleroQPS patches the Velero deployment to increase client QPS/burst.
// This helps with high-latency connections like IAP tunnels.
// Uses kubectl patch to match bash implementation exactly.
//
//	kubectl patch deployment velero -n velero --type='json' -p='[
//	  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--client-qps=100"},
//	  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--client-burst=200"}
//	]'
func (c *Client) patchVeleroQPS(ctx context.Context) error {
	// JSON patch to append QPS and burst args
	patch := `[
		{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--client-qps=100"},
		{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--client-burst=200"}
	]`

	cmd := exec.CommandContext(ctx, "kubectl", "patch", "deployment", "velero",
		"-n", VeleroNamespace,
		"--type=json",
		"-p", patch,
		"--kubeconfig", c.kubeconfigPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.log.Debug("kubectl patch output:\n%s", string(output))
		return fmt.Errorf("failed to patch velero deployment: %w", err)
	}

	return nil
}

// isInstalled checks if Velero is already installed and running.
func (c *Client) isInstalled(ctx context.Context) (bool, error) {
	deployments, err := c.clientset.AppsV1().Deployments(VeleroNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=velero",
	})
	if err != nil {
		return false, nil // Namespace might not exist yet
	}

	for _, deploy := range deployments.Items {
		if deploy.Status.ReadyReplicas > 0 {
			return true, nil
		}
	}

	return false, nil
}

// WaitForReady waits for Velero deployment to be ready.
//
// Uses kubectl rollout status to avoid client-go rate limiter issues with
// high-latency IAP tunnels. This matches the bash implementation exactly.
//
// Parameters:
//   - ctx: Context for cancellation
//   - timeout: Maximum time to wait
//
// Returns:
//   - error: If timeout reached or context cancelled
//
//	kubectl rollout status deployment/velero -n velero --timeout=120s
func (c *Client) WaitForReady(ctx context.Context, timeout time.Duration) error {
	c.log.Step("Waiting for Velero deployment to be ready")

	// Use kubectl rollout status (simpler than client-go watch, avoids rate limiter)
	timeoutStr := fmt.Sprintf("%ds", int(timeout.Seconds()))
	cmd := exec.CommandContext(ctx, "kubectl", "rollout", "status",
		"deployment/velero",
		"-n", VeleroNamespace,
		"--timeout="+timeoutStr,
		"--kubeconfig", c.kubeconfigPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.log.Debug("kubectl rollout status output:\n%s", string(output))
		return fmt.Errorf("Velero deployment not ready: %w", err)
	}

	c.log.Info("Velero is ready")
	return nil
}

// CreateBackup creates a new Velero backup.
//
// The backup name will have a timestamp suffix added automatically.
// Format: <name>-<ddmmyyyy-hhmm>
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Base backup name (timestamp will be appended)
//   - namespaces: Comma-separated list of namespaces to backup
//
// Returns:
//   - string: Full backup name with timestamp
//   - error: If backup creation or wait fails
//
// Example:
//
//	backupName, err := veleroClient.CreateBackup(ctx, "k8s-lab-backup", "application")
//	// Creates: k8s-lab-backup-04022026-1430
//
func (c *Client) CreateBackup(ctx context.Context, name string, namespaces string) (string, error) {
	// Generate timestamp: ddmmyyyy-hhmm (UTC)
	timestamp := time.Now().UTC().Format("02012006-1504")
	backupName := fmt.Sprintf("%s-%s", name, timestamp)

	c.log.Step("Creating Velero backup: %s", backupName)
	c.log.Info("Namespaces: %s", namespaces)

	// Parse namespaces (comma-separated to slice)
	nsSlice := strings.Split(namespaces, ",")

	// Create Backup CR
	// Only set essential fields - let Velero use defaults for snapshot behavior
	// Key: Do NOT set IncludeClusterResources=false or SnapshotVolumes explicitly
	// Velero will automatically snapshot PVs when VolumeSnapshotLocations is set
	backup := &velerov1.Backup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "velero.io/v1",
			Kind:       "Backup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: VeleroNamespace,
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces:      nsSlice,
			StorageLocation:         "default",
			VolumeSnapshotLocations: []string{"default"},
			TTL:                     metav1.Duration{Duration: DefaultBackupTTL},
		},
	}

	// Convert to unstructured for dynamic client
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(backup)
	if err != nil {
		return "", fmt.Errorf("failed to convert backup to unstructured: %w", err)
	}

	unstructuredBackup := &unstructured.Unstructured{Object: unstructuredObj}

	_, err = c.dynamicClient.Resource(BackupGVR).Namespace(VeleroNamespace).Create(ctx, unstructuredBackup, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	c.log.Info("Waiting for backup to complete (timeout: 10m)")

	// Wait for backup to reach terminal phase
	if err := c.waitForBackupCompletion(ctx, backupName, 10*time.Minute); err != nil {
		return "", err
	}

	// Verify backup succeeded
	if err := c.verifyBackup(ctx, backupName); err != nil {
		return "", err
	}

	return backupName, nil
}

// waitForBackupCompletion waits for backup to reach a terminal phase.
func (c *Client) waitForBackupCompletion(ctx context.Context, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for backup: %w", ctx.Err())

		case <-ticker.C:
			unstructuredBackup, err := c.dynamicClient.Resource(BackupGVR).Namespace(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				c.log.Debug("Failed to get backup status: %v", err)
				continue
			}

			// Extract phase from status
			phase, found, err := unstructured.NestedString(unstructuredBackup.Object, "status", "phase")
			if err != nil || !found {
				c.log.Debug("Backup phase not found yet")
				continue
			}

			if phase == string(velerov1.BackupPhaseCompleted) ||
				phase == string(velerov1.BackupPhaseFailed) ||
				phase == string(velerov1.BackupPhasePartiallyFailed) {
				return nil
			}

			c.log.Debug("Backup phase: %s", phase)
		}
	}
}

// verifyBackup checks that the backup completed successfully.
func (c *Client) verifyBackup(ctx context.Context, name string) error {
	c.log.Step("Verifying backup: %s", name)

	unstructuredBackup, err := c.dynamicClient.Resource(BackupGVR).Namespace(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}

	phase, found, err := unstructured.NestedString(unstructuredBackup.Object, "status", "phase")
	if err != nil || !found {
		return fmt.Errorf("backup status not available")
	}

	if phase != string(velerov1.BackupPhaseCompleted) {
		return fmt.Errorf("backup failed with status: %s", phase)
	}

	c.log.Info("Backup verified: %s", phase)
	return nil
}

// GetBackupStatus returns the current status of a backup.
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Backup name
//
// Returns:
//   - string: Backup phase (Completed, Failed, InProgress, etc.)
//   - error: If backup not found
//
// Example:
//
//	status, err := veleroClient.GetBackupStatus(ctx, "k8s-lab-backup-04022026-1430")
//	if status == "Completed" {
//	    // Backup ready for restore
//	}
//
// Equivalent to bash: kubectl get backup <name> -n velero -o jsonpath='{.status.phase}'
func (c *Client) GetBackupStatus(ctx context.Context, name string) (string, error) {
	unstructuredBackup, err := c.dynamicClient.Resource(BackupGVR).Namespace(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get backup: %w", err)
	}

	phase, found, err := unstructured.NestedString(unstructuredBackup.Object, "status", "phase")
	if err != nil || !found {
		return "", fmt.Errorf("backup status not available")
	}

	return phase, nil
}

// Restore restores from a Velero backup.
//
// If backupName is empty, restores from the most recent successful backup.
//
// Parameters:
//   - ctx: Context for cancellation
//   - backupName: Name of backup to restore from (empty = latest)
//
// Returns:
//   - error: If restore fails
//
// Example:
//
//	err := veleroClient.Restore(ctx, "k8s-lab-backup-04022026-1430")
//
func (c *Client) Restore(ctx context.Context, backupName string) error {
	// If no backup name provided, find latest
	if backupName == "" {
		latest, err := c.findLatestBackup(ctx)
		if err != nil {
			return err
		}
		backupName = latest
		c.log.Info("Using latest backup: %s", backupName)
	}

	c.log.Step("Restoring from backup: %s", backupName)

	// Generate restore name with timestamp
	timestamp := time.Now().UTC().Format("150405") // hhmmss
	restoreName := fmt.Sprintf("%s-%s", backupName, timestamp)

	// Create Restore CR
	restore := &velerov1.Restore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "velero.io/v1",
			Kind:       "Restore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreName,
			Namespace: VeleroNamespace,
		},
		Spec: velerov1.RestoreSpec{
			BackupName: backupName,
		},
	}

	// Convert to unstructured
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(restore)
	if err != nil {
		return fmt.Errorf("failed to convert restore to unstructured: %w", err)
	}

	unstructuredRestore := &unstructured.Unstructured{Object: unstructuredObj}

	_, err = c.dynamicClient.Resource(RestoreGVR).Namespace(VeleroNamespace).Create(ctx, unstructuredRestore, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create restore: %w", err)
	}

	c.log.Info("Waiting for restore to complete (timeout: 10m)")

	// Wait for restore to reach terminal phase
	if err := c.waitForRestoreCompletion(ctx, restoreName, 10*time.Minute); err != nil {
		return err
	}

	// Verify restore
	if err := c.verifyRestore(ctx, restoreName); err != nil {
		return err
	}

	return nil
}

// findLatestBackup finds the most recent successful backup.
func (c *Client) findLatestBackup(ctx context.Context) (string, error) {
	unstructuredBackupList, err := c.dynamicClient.Resource(BackupGVR).Namespace(VeleroNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list backups: %w", err)
	}

	if len(unstructuredBackupList.Items) == 0 {
		return "", fmt.Errorf("no backups found")
	}

	// Find most recent completed backup
	var latestName string
	var latestTime time.Time

	for _, item := range unstructuredBackupList.Items {
		phase, found, err := unstructured.NestedString(item.Object, "status", "phase")
		if err != nil || !found {
			continue
		}

		if phase != string(velerov1.BackupPhaseCompleted) {
			continue
		}

		// Get completion timestamp
		completionStr, found, err := unstructured.NestedString(item.Object, "status", "completionTimestamp")
		if err != nil || !found {
			continue
		}

		completionTime, err := time.Parse(time.RFC3339, completionStr)
		if err != nil {
			continue
		}

		if latestName == "" || completionTime.After(latestTime) {
			latestName = item.GetName()
			latestTime = completionTime
		}
	}

	if latestName == "" {
		return "", fmt.Errorf("no completed backups found")
	}

	return latestName, nil
}

// waitForRestoreCompletion waits for restore to reach a terminal phase.
func (c *Client) waitForRestoreCompletion(ctx context.Context, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for restore: %w", ctx.Err())

		case <-ticker.C:
			unstructuredRestore, err := c.dynamicClient.Resource(RestoreGVR).Namespace(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				c.log.Debug("Failed to get restore status: %v", err)
				continue
			}

			phase, found, err := unstructured.NestedString(unstructuredRestore.Object, "status", "phase")
			if err != nil || !found {
				c.log.Debug("Restore phase not found yet")
				continue
			}

			if phase == string(velerov1.RestorePhaseCompleted) ||
				phase == string(velerov1.RestorePhaseFailed) ||
				phase == string(velerov1.RestorePhasePartiallyFailed) {
				return nil
			}

			c.log.Debug("Restore phase: %s", phase)
		}
	}
}

// verifyRestore checks Velero restore CR status.
func (c *Client) verifyRestore(ctx context.Context, name string) error {
	c.log.Step("Verifying Velero restore status")

	unstructuredRestore, err := c.dynamicClient.Resource(RestoreGVR).Namespace(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get restore: %w", err)
	}

	phase, found, err := unstructured.NestedString(unstructuredRestore.Object, "status", "phase")
	if err != nil || !found {
		return fmt.Errorf("restore status not available")
	}

	switch phase {
	case string(velerov1.RestorePhaseCompleted):
		c.log.Info("Restore status: %s", phase)
		return nil
	case string(velerov1.RestorePhasePartiallyFailed):
		c.log.Warn("Restore status: %s (may be OK if PVs were recreated)", phase)
		return nil
	default:
		return fmt.Errorf("restore failed with status: %s", phase)
	}
}
