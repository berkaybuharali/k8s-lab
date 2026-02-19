package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
)

var (
	skipBuild bool
)

func init() {
	rootCmd.AddCommand(deployAgentsCmd)
	deployAgentsCmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip building and pushing Docker images")
}

var deployAgentsCmd = &cobra.Command{
	Use:   "deploy-agents",
	Short: "Deploy Magic Cake agents",
	Long: `Build and deploy the Commerce and Supply Chain agents.

This command:
1. Builds Docker images for both agent systems (skip with --skip-build)
2. Pushes images to Artifact Registry
3. Patches manifests with your Project ID and API keys
4. Deploys to Kubernetes (agents namespace)

Requires GOOGLE_API_KEY environment variable.

Use --skip-build to only deploy manifests without rebuilding images.`,
	RunE: runDeployAgents,
}

func runDeployAgents(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	// Check prerequisites
	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	// Check for Docker
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is required for building agent images")
	}

	// Check environment variables
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GOOGLE_API_KEY environment variable is required (get one from aistudio.google.com/apikey, enable Maps APIs in GCP Console)")
	}

	log.Info("==============================================")
	log.Info("  Magic Cake - Agent Deployment")
	log.Info("  Cloud: %s", provider.Name())
	log.Info("==============================================")

	// Get infrastructure info
	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	repoRoot := cfg.GetRepoRoot()
	artifactRegistry := fmt.Sprintf("%s-docker.pkg.dev/%s/k8s-lab", infra.Region, infra.ProjectID)

	// 1. Build and Push Images (unless --skip-build)
	if !skipBuild {
		images := []struct {
			Name       string
			DockerFile string
			Context    string
		}{
			{
				Name:       "commerce",
				DockerFile: "agents/commerce/Dockerfile",
				Context:    ".",
			},
			{
				Name:       "supply-chain",
				DockerFile: "agents/supply_chain/Dockerfile",
				Context:    ".",
			},
		}

		for _, img := range images {
			fullImageName := fmt.Sprintf("%s/%s:latest", artifactRegistry, img.Name)
			log.Step("Building and pushing %s...", img.Name)

			// Docker Build and Push
			// Use --push to push directly to registry (more reliable with buildkit than --load + push)
			// Build for linux/amd64 since GCP VMs are amd64 (even on arm64 Mac)
			buildPushCmd := exec.CommandContext(ctx, "docker", "build",
				"--platform", "linux/amd64",
				"--push",
				"-t", fullImageName,
				"-f", filepath.Join(repoRoot, img.DockerFile),
				repoRoot,
			)
			buildPushCmd.Stdout = os.Stdout
			buildPushCmd.Stderr = os.Stderr
			if err := buildPushCmd.Run(); err != nil {
				return fmt.Errorf("failed to build and push %s: %w", img.Name, err)
			}
			log.Debug("Build and push complete for %s", img.Name)
		}
	} else {
		log.Info("Skipping build (--skip-build flag set)")
	}

	// 2. Prepare Manifests
	log.Step("Preparing manifests...")
	tempDir, err := os.MkdirTemp("", "k8s-lab-agents")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Prepare replacements
	replacements := map[string]string{
		"YOUR_PROJECT_ID":      infra.ProjectID,
		"YOUR_REGION":          infra.Region,
		"YOUR_BUCKET":          infra.Bucket,
		"YOUR_GOOGLE_API_KEY":  apiKey,
	}

	// Helper to patch and write file
	patchFile := func(srcPath, destName string, customReplacements map[string]string) (string, error) {
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return "", err
		}
		strContent := string(content)

		// Global replacements
		for k, v := range replacements {
			strContent = strings.ReplaceAll(strContent, k, v)
		}

		// Custom replacements (e.g. image tag)
		for k, v := range customReplacements {
			strContent = strings.ReplaceAll(strContent, k, v)
		}

		destPath := filepath.Join(tempDir, destName)
		if err := os.WriteFile(destPath, []byte(strContent), 0644); err != nil {
			return "", err
		}
		return destPath, nil
	}

	manifestsDir := filepath.Join(repoRoot, "apps", "agents")

	// Patch ConfigMap
	cmPath, err := patchFile(filepath.Join(manifestsDir, "configmap.yaml.example"), "configmap.yaml", nil)
	if err != nil {
		return err
	}

	// Patch Commerce
	commercePath, err := patchFile(filepath.Join(manifestsDir, "commerce.yaml.example"), "commerce.yaml", nil)
	if err != nil {
		return err
	}

	// Patch Supply Chain
	supplyChainPath, err := patchFile(filepath.Join(manifestsDir, "supply-chain.yaml.example"), "supply-chain.yaml", nil)
	if err != nil {
		return err
	}

	// 3. Apply Manifests
	log.Step("Applying manifests...")

	// Create Tunnel
	log.Info("Creating tunnel to Kubernetes API...")
	k8sEndpoint, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()
	log.Debug("K8s API accessible at: %s", k8sEndpoint)

	// Create Client
	k8sClient, err := k8s.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return fmt.Errorf("failed to create K8s client: %w", err)
	}

	// Apply Namespace
	log.Debug("Applying namespace...")
	if err := k8sClient.ApplyManifest(ctx, filepath.Join(manifestsDir, "namespace.yaml")); err != nil {
		return fmt.Errorf("failed to apply namespace: %w", err)
	}
	log.Debug("Namespace applied")

	// Deploy Redis into the agents namespace alongside agent pods
	log.Step("Deploying Redis...")
	if err := k8sClient.ApplyManifest(ctx, filepath.Join(repoRoot, "apps", "redis.yaml")); err != nil {
		return fmt.Errorf("failed to apply redis: %w", err)
	}
	if err := k8sClient.WaitForDeploymentReady(ctx, "agents", "redis", 3*time.Minute); err != nil {
		return fmt.Errorf("redis deployment failed: %w", err)
	}
	log.Debug("Redis ready")

	// Apply GCR Credential Sync (RBAC + CronJob that refreshes the imagePullSecret every 10 min)
	log.Debug("Applying credential sync resources...")
	if err := k8sClient.ApplyManifest(ctx, filepath.Join(manifestsDir, "gcr-credential-sync.yaml")); err != nil {
		return fmt.Errorf("failed to apply credential sync: %w", err)
	}

	// CronJobs don't run on apply, only on schedule. Trigger one-off run immediately.
	log.Step("Creating imagePullSecret...")
	kubeconfigPath := cfg.GetKubeconfigPath()
	triggerCmd := exec.CommandContext(ctx, "kubectl",
		"--kubeconfig", kubeconfigPath,
		"-n", "agents",
		"create", "job", "gcr-credential-sync-init",
		"--from=cronjob/gcr-credential-sync",
	)
	// Ignore error: job may already exist from a previous run
	_ = triggerCmd.Run()

	if err := k8sClient.WaitForSecret(ctx, "agents", "artifact-registry", 2*time.Minute); err != nil {
		return fmt.Errorf("imagePullSecret not ready: %w", err)
	}
	log.Debug("imagePullSecret ready")

	// Apply ConfigMap
	log.Debug("Applying configmap...")
	if err := k8sClient.ApplyManifest(ctx, cmPath); err != nil {
		return fmt.Errorf("failed to apply configmap: %w", err)
	}
	log.Debug("Configmap applied")

	// Apply Commerce
	log.Debug("Applying commerce deployment...")
	if err := k8sClient.ApplyManifest(ctx, commercePath); err != nil {
		return fmt.Errorf("failed to apply commerce: %w", err)
	}
	log.Info("Commerce deployment applied")

	// Apply Supply Chain
	log.Debug("Applying supply-chain deployment...")
	if err := k8sClient.ApplyManifest(ctx, supplyChainPath); err != nil {
		return fmt.Errorf("failed to apply supply-chain: %w", err)
	}
	log.Info("Supply-chain deployment applied")

	// 4. Wait for Readiness
	log.Step("Waiting for agents to be ready...")
	log.Debug("Waiting for commerce to be ready...")

	if err := k8sClient.WaitForDeploymentReady(ctx, "agents", "commerce", 5*time.Minute); err != nil {
		return fmt.Errorf("commerce agent failed to start: %w", err)
	}
	log.Debug("Commerce is ready")

	log.Debug("Waiting for supply-chain to be ready...")

	if err := k8sClient.WaitForDeploymentReady(ctx, "agents", "supply-chain", 5*time.Minute); err != nil {
		return fmt.Errorf("supply-chain agent failed to start: %w", err)
	}
	log.Debug("Supply-chain is ready")

	log.Info("Agents deployed successfully!")
	log.Info("Next: k8s-lab seed-data --cloud %s", provider.Name())

	return nil
}
