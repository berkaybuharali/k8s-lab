package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

func init() {
	rootCmd.AddCommand(seedInventoryCmd)
}

var seedInventoryCmd = &cobra.Command{
	Use:   "seed-inventory",
	Short: "Seed inventory and orders for Magic Cake",
	Long: `Populate Redis with inventory and test orders for the Magic Cake agents.

This command:
1. Resets inventory to known state (5 items)
2. Creates 7 test orders across next 3 days
3. Uploads dummy cake images to GCS
4. Verifies data in Redis`,
	RunE: runSeedInventory,
}

func runSeedInventory(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)
	ctx := cmd.Context()

	if err := RequireCloud(provider); err != nil {
		return err
	}

	// Check prerequisites (kubectl + gcloud)
	if err := checkToolsPrerequisites(cfg, log); err != nil {
		return err
	}

	log.Info("==============================================")
	log.Info("  Magic Cake - Seed Inventory")
	log.Info("  Cloud: %s", provider.Name())
	log.Info("==============================================")

	// Get infrastructure info
	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	// Create Tunnel
	log.Info("Creating tunnel to Kubernetes API...")
	_, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()

	// Create Client
	k8sClient, err := k8s.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return fmt.Errorf("failed to create K8s client: %w", err)
	}

	// Ensure Redis Ready
	podName, err := ensureRedisReady(ctx, k8sClient, log)
	if err != nil {
		return err
	}

	// Seed Inventory
	log.Step("Seeding Inventory...")
	inventoryCmds := generateInventoryCommands()
	if err := execRedisCommands(ctx, k8sClient, podName, inventoryCmds, log); err != nil {
		return err
	}

	// Seed Orders + Images
	log.Step("Seeding Orders and Images...")
	
	// Create dummy image file
	dummyImg, err := createDummyImage()
	if err != nil {
		return err
	}
	defer os.Remove(dummyImg)

	orders, orderCmds := generateOrderCommands(infra.Bucket)
	
	// Execute Redis commands for orders
	if err := execRedisCommands(ctx, k8sClient, podName, orderCmds, log); err != nil {
		return err
	}

	// Upload images
	log.Step("Uploading images to GCS...")
	for _, order := range orders {
		// gs://{bucket}/cakes/orders/{order_id}/cake_1.png
		dest := fmt.Sprintf("gs://%s/cakes/orders/%s/cake_1.png", infra.Bucket, order.ID)
		
		log.Debug("Uploading image for order %s", order.ID)
		cmd := exec.CommandContext(ctx, "gcloud", "storage", "cp", dummyImg, dest)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to upload image: %s\nOutput: %s", err, out)
		}
	}

	log.Info("Inventory and Orders seeded successfully!")
	return nil
}

func execRedisCommands(ctx context.Context, client *k8s.Client, podName, payload string, log *logger.Logger) error {
	// Agents use the shared Redis instance in the 'application' namespace.
	// This Redis instance serves both traditional applications (via seed-redis)
	// and Magic Cake agents (inventory, orders). The 'application' namespace
	// effectively serves as the shared data layer for the entire lab.
	_, stderr, err := client.Exec(
		ctx,
		ApplicationNamespace,
		podName,
		"", 
		[]string{"redis-cli"},
		strings.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to execute redis commands: %w\nStderr: %s", err, stderr)
	}
	return nil
}

func generateInventoryCommands() string {
	var sb strings.Builder
	// 5 items, max 5 per type. 
	// chocolate: 4, ananas: 1 (LOW), banana: 3, walnut: 2 (LOW), almond: 4
	sb.WriteString("HSET inventory:chocolate quantity 4\n")
	sb.WriteString("HSET inventory:ananas quantity 1\n")
	sb.WriteString("HSET inventory:banana quantity 3\n")
	sb.WriteString("HSET inventory:walnut quantity 2\n")
	sb.WriteString("HSET inventory:almond quantity 4\n")
	return sb.String()
}

type seededOrder struct {
	ID string
}

func generateOrderCommands(bucketName string) ([]seededOrder, string) {
	var sb strings.Builder
	var orders []seededOrder

	// 7 orders: Today+1 (4), Today+2 (2), Today+3 (1)
	now := time.Now()
	createdAt := now.Format(time.RFC3339)
	dates := []string{
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.AddDate(0, 0, 2).Format("2006-01-02"),
		now.AddDate(0, 0, 2).Format("2006-01-02"),
		now.AddDate(0, 0, 3).Format("2006-01-02"),
	}

	addresses := []struct{ Street, Postcode string }{
		{"Herengracht 502", "1017 CB"},
		{"Prinsengracht 263", "1016 GV"},
		{"Keizersgracht 174", "1016 DW"},
		{"Damrak 1", "1012 LG"},
		{"Rokin 92", "1012 KZ"},
		{"Singel 140", "1015 AG"},
		{"Amstel 51", "1018 EJ"},
	}
	
	names := []string{"Jan de Vries", "Maria van den Berg", "John Doe", "Mary Steling", "Peter Pan", "Alice Wonderland", "Bob Builder"}

	for i, date := range dates {
		// Generate ID: CAKE-{YYYYMMDD}-{XXXX}
		dateCompact := strings.ReplaceAll(date, "-", "")
		suffix := uuid.New().String()[:4] // first 4 chars of UUID
		orderID := fmt.Sprintf("CAKE-%s-%s", dateCompact, strings.ToUpper(suffix))
		
		orders = append(orders, seededOrder{ID: orderID})
		
		addr := addresses[i%len(addresses)]
		name := names[i%len(names)]
		
		// Construct Redis HSET command
		// Need proper JSON escaping
		key := fmt.Sprintf("order:%s", orderID)
		
		cakesJson := fmt.Sprintf(`[{"flavor":"chocolate","nuts":"almond","people_count":10,"concept":"birthday %d"}]`, i)
		imagePathsJson := fmt.Sprintf(`["gs://%s/cakes/orders/%s/cake_1.png"]`, bucketName, orderID)
		
		// HSET key field value field value ...
		// We use single quotes for JSON strings to avoid shell issues with double quotes inside
		// Updated HSET with all required fields (10 people * 5 EUR = 50 EUR, free delivery since >= 50 EUR)
		sb.WriteString(fmt.Sprintf(
			"HSET %s order_id \"%s\" customer_name \"%s\" address \"%s\" postcode \"%s\" "+
			"delivery_date \"%s\" total_cake_price \"50.0\" delivery_fee \"0.0\" total_price \"50.0\" "+
			"status \"confirmed\" created_at \"%s\" cakes '%s' image_paths '%s'\n",
			key, orderID, name, addr.Street, addr.Postcode, date, createdAt, cakesJson, imagePathsJson))
	}
	return orders, sb.String()
}

func createDummyImage() (string, error) {
	f, err := os.CreateTemp("", "cake-*.png")
	if err != nil {
		return "", err
	}
	// Write some bytes
	f.Write([]byte("fake image content"))
	f.Close()
	return f.Name(), nil
}
