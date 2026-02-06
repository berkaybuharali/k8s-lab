// cli/cmd/seed_redis.go
// Package cmd implements the seed-redis command.
package cmd

import (
	"context"
	"fmt"
	"strconv"

	"strings"

	"time"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

func init() {
	rootCmd.AddCommand(seedRedisCmd)
}

var seedRedisCmd = &cobra.Command{
	Use:   "seed-redis",
	Short: "Seed Redis with test data",
	Long: `Populates Redis with sample data for testing backup/restore.

This command:
1. Verifies Redis is running
2. Connects to the Redis pod
3. Inserts 100+ keys (users, config, queue)
4. Verifies data persistence

Example:
  k8s-lab seed-redis --cloud gcp`,
	RunE: runSeedRedis,
}

func runSeedRedis(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	log.Info("==============================================")
	log.Info("  Kubernetes Lab - Seed Redis Data")
	log.Info("  Cloud: %s", provider.Name())
	log.Info("==============================================")
	log.Info("")

	// Check tools prerequisites (kubeconfig)
	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	// Get infrastructure info
	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	// Create K8s API tunnel
	log.Info("")
	log.Info("Creating tunnel to Kubernetes API...")
	k8sEndpoint, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()
	log.Debug("K8s API accessible at: %s", k8sEndpoint)

	// Create K8s client
	k8sClient, err := k8s.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return fmt.Errorf("failed to create K8s client: %w", err)
	}

	// 1. Ensure Redis is Ready
	podName, err := ensureRedisReady(ctx, k8sClient, log)
	if err != nil {
		return err
	}

	// 2. Seed Data
	if err := seedRedisData(ctx, k8sClient, podName, log); err != nil {
		return err
	}

	// 3. Verify Key Count
	if err := verifyRedisKeyCount(ctx, k8sClient, podName, log); err != nil {
		return err
	}

	printSeedSuccess(log)
	return nil
}

// ensureRedisReady verifies the Redis deployment and returns the name of a running pod.
func ensureRedisReady(ctx context.Context, client *k8s.Client, log *logger.Logger) (string, error) {
	log.Step("Verifying Redis deployment")
	if err := client.WaitForDeploymentReady(ctx, ApplicationNamespace, "redis", 2*time.Minute); err != nil {
		return "", fmt.Errorf("redis deployment not ready: %w", err)
	}

	podName, err := client.GetFirstPodByLabel(ctx, ApplicationNamespace, "app=redis")
	if err != nil {
		return "", fmt.Errorf("failed to find redis pod: %w", err)
	}
	log.Debug("Targeting Redis pod: %s", podName)
	return podName, nil
}

// seedRedisData generates and inserts the test data payload into the Redis pod.
func seedRedisData(ctx context.Context, client *k8s.Client, podName string, log *logger.Logger) error {
	log.Step("Generating test data")
	payload := generateRedisTestData()

	log.Step("Seeding data into Redis...")
	log.Debug("Sending %d bytes of commands", len(payload))

	_, stderr, err := client.Exec(
		ctx,
		ApplicationNamespace,
		podName,
		"", // Default to first container
		[]string{"redis-cli"},
		strings.NewReader(payload),
	)

	if err != nil {
		return fmt.Errorf("failed to execute redis-cli: %w\nStderr: %s", err, stderr)
	}
	return nil
}

// verifyRedisKeyCount checks if the number of keys in Redis meets the minimum requirement.
func verifyRedisKeyCount(ctx context.Context, client *k8s.Client, podName string, log *logger.Logger) error {
	log.Step("Verifying seeded data")
	stdout, _, err := client.Exec(
		ctx,
		ApplicationNamespace,
		podName,
		"", // Default to first container
		[]string{"redis-cli", "DBSIZE"},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to verify data size: %w", err)
	}

	countStr := strings.TrimSpace(stdout)
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return fmt.Errorf("failed to parse Redis key count '%s': %w", countStr, err)
	}

	if count < 100 {
		return fmt.Errorf("expected at least 100 keys, found %d", count)
	}

	log.Info("Total keys in Redis: %d", count)
	return nil
}

// generateRedisTestData creates a bulk command payload for redis-cli.
// It uses raw strings for JSON to keep the code clean.
func generateRedisTestData() string {
	var sb strings.Builder

	// Standard keys from bash script
	sb.WriteString(`SET user:1 '{"name":"Alice","email":"alice@example.com"}'` + "\n")
	sb.WriteString(`SET user:2 '{"name":"Bob","email":"bob@example.com"}'` + "\n")
	sb.WriteString(`SET user:3 '{"name":"Charlie","email":"charlie@example.com"}'` + "\n")
	sb.WriteString("SET counter:visits 1000\n")
	sb.WriteString("SET config:app:version \"1.0.0\"\n")
	sb.WriteString("LPUSH queue:tasks \"task1\" \"task2\" \"task3\"\n")

	// Bulk keys (user:4 to user:103)
	for i := 4; i <= 103; i++ {
		sb.WriteString(fmt.Sprintf(`SET user:%d '{"name":"User%d","generated":true}'`+"\n", i, i))
	}

	return sb.String()
}

func printSeedSuccess(log *logger.Logger) {
	log.Info("")
	log.Info("==============================================")
	log.Info("  Redis seeded successfully!")
	log.Info("==============================================")
	log.Info("")
	log.Info("Next steps:")
	log.Info("  k8s-lab backup --cloud gcp         # Create backup")
	log.Info("")
}
