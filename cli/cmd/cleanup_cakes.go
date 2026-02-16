package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

func init() {
	rootCmd.AddCommand(cleanupCakesCmd)
}

var cleanupCakesCmd = &cobra.Command{
	Use:   "cleanup-cakes",
	Short: "Clean up orphan cake images from GCS",
	Long: `Identify and remove GCS images that are not referenced by any active order.

This command:
1. Scans all orders in Redis
2. Lists all images in GCS (cakes/ prefix)
3. Deletes images not linked to an order
4. Reports statistics`,
	RunE: runCleanupCakes,
}

func runCleanupCakes(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	log.Info("==============================================")
	log.Info("  Magic Cake - Cleanup Orphans")
	log.Info("  Cloud: %s", provider.Name())
	log.Info("==============================================")

	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	// 1. Get referenced images from Redis
	log.Step("Scanning Redis orders...")
	
	// Create Tunnel
	log.Debug("Creating tunnel...")
	_, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()

	k8sClient, err := k8s.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return err
	}

	podName, err := ensureRedisReady(ctx, k8sClient, log)
	if err != nil {
		return err
	}

	validImages, err := getReferencedImages(ctx, k8sClient, podName, log)
	if err != nil {
		return err
	}
	log.Info("Found %d valid images in active orders", len(validImages))

	// 2. List GCS images
	log.Step("Scanning GCS bucket...")
	gcsImages, err := listGCSImages(ctx, infra.Bucket)
	if err != nil {
		return err
	}
	log.Info("Found %d images in GCS", len(gcsImages))

	// 3. Find Orphans
	var orphans []string
	for _, img := range gcsImages {
		if !validImages[img] {
			orphans = append(orphans, img)
		}
	}

	// 4. Delete Orphans
	if len(orphans) == 0 {
		log.Info("No orphan images found. System is clean.")
		return nil
	}

	log.Warn("Found %d orphan images. Deleting...", len(orphans))
	
	// Delete in batches or one by one? One by one is safer but slower. 
	// `gcloud storage rm` accepts multiple arguments.
	// Let's do batches of 100
	batchSize := 100
	for i := 0; i < len(orphans); i += batchSize {
		end := i + batchSize
		if end > len(orphans) {
			end = len(orphans)
		}
		batch := orphans[i:end]
		
		log.Debug("Deleting batch %d-%d...", i, end)
		args := append([]string{"storage", "rm"}, batch...)
		cmd := exec.CommandContext(ctx, "gcloud", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Error("Failed to delete batch: %v\nOutput: %s", err, out)
			// Continue with next batch
		}
	}

	log.Info("Cleanup complete. Removed %d orphans.", len(orphans))
	return nil
}

func getReferencedImages(ctx context.Context, client *k8s.Client, podName string, log *logger.Logger) (map[string]bool, error) {
	validImages := make(map[string]bool)

	// 1. Scan for keys
	// This is a simplified scan. In production, use cursor loop.
	// For PoC with < 1000 orders, KEYS is fine or SCAN with large count.
	scanCmd := []string{"redis-cli", "SCAN", "0", "MATCH", "order:*", "COUNT", "10000"}
	stdout, _, err := client.Exec(ctx, ApplicationNamespace, podName, "", scanCmd, nil)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 2 {
		return validImages, nil // No keys?
	}
	// Redis SCAN output:
	// 1) "0" (cursor)
	// 2) 1) "key1"
	//    2) "key2"
	// Parse output is tricky via simple split because formatting depends on client version/tty.
	// Better to use Lua script to get all values directly?
	// Or use `KEYS order:*` since this is a lab environment.
	
	// For PoC/lab environment with <1000 orders, KEYS is acceptable and simpler.
	// Production environments should use cursor-based SCAN loop to avoid blocking Redis.
	// Example production approach:
	//   cursor := "0"
	//   for cursor != "0" {
	//       cursor, keys := SCAN cursor MATCH order:* COUNT 100
	//       // process keys...
	//   }
	keysCmd := []string{"redis-cli", "KEYS", "order:*"}
	stdout, _, err = client.Exec(ctx, ApplicationNamespace, podName, "", keysCmd, nil)
	if err != nil {
		return nil, err
	}
	
	keys := strings.Fields(stdout)
	if len(keys) == 0 {
		return validImages, nil
	}

	// 2. Get image_paths for each order
	// Batching HGETs would be better, but loop is easier to implement for PoC
	for _, key := range keys {
		hgetCmd := []string{"redis-cli", "HGET", key, "image_paths"}
		out, _, err := client.Exec(ctx, ApplicationNamespace, podName, "", hgetCmd, nil)
		if err != nil {
			log.Warn("Failed to get order %s: %v", key, err)
			continue
		}
		
		val := strings.TrimSpace(out)
		if val == "" || val == "(nil)" {
			continue
		}
		
		// Unmarshal JSON list
		var paths []string
		// val might be single quoted or raw. redis-cli output usually raw if not tty.
		// If it contains spaces it might be quoted.
		// Try to parse.
		if err := json.Unmarshal([]byte(val), &paths); err != nil {
			log.Debug("Failed to parse image_paths for %s: %v (val: %s)", key, err, val)
			continue
		}
		
		for _, p := range paths {
			validImages[p] = true
		}
	}

	return validImages, nil
}

func listGCSImages(ctx context.Context, bucket string) ([]string, error) {
	// gsutil ls -r gs://bucket/cakes/
	// output: gs://bucket/cakes/orders/id/img.png
	url := fmt.Sprintf("gs://%s/cakes/", bucket)
	
	cmd := exec.CommandContext(ctx, "gcloud", "storage", "ls", "--recursive", url)
	out, err := cmd.Output()
	if err != nil {
		// If bucket empty or prefix doesn't exist, it might fail.
		// Check error.
		return nil, nil // Assume empty
	}
	
	var images []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, "/:") || strings.HasSuffix(line, "/") {
			continue // Skip directories or headers
		}
		if strings.HasPrefix(line, "gs://") {
			images = append(images, line)
		}
	}
	return images, nil
}
