package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/k8s"
	"github.com/berkaybuharali/k8s-lab/cli/pkg/logger"
)

func init() {
	rootCmd.AddCommand(seedDataCmd)
}

var seedDataCmd = &cobra.Command{
	Use:   "seed-data",
	Short: "Seed inventory and orders for Magic Cake",
	Long: `Populate Redis and GCS with test data for Magic Cake agents.

This command:
1. Flushes Redis to remove stale data from previous runs
2. Resets inventory to known state (5 ingredients)
3. Uses Gemini to generate 5 varied orders (names, addresses, cake details)
4. Generates real cake images via Gemini 2.5 Flash Image (exec into commerce pod)
5. Uploads images to GCS and stores orders in Redis`,
	RunE: runSeedData,
}

func runSeedData(cmd *cobra.Command, args []string) error {
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
	log.Info("  Magic Cake - Seed Data")
	log.Info("  Cloud: %s", provider.Name())
	log.Info("==============================================")

	infra, err := getInfrastructureInfo(cfg, provider, log)
	if err != nil {
		return err
	}

	log.Info("Creating tunnel to Kubernetes API...")
	_, cleanup, err := provider.CreateK8sEndpoint(ctx, infra.CPName, infra.CPZone, infra.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create K8s tunnel: %w", err)
	}
	defer cleanup()

	k8sClient, err := k8s.NewClient(cfg.GetKubeconfigPath(), log)
	if err != nil {
		return fmt.Errorf("failed to create K8s client: %w", err)
	}

	// 1. Ensure Redis is ready and flush stale data
	log.Step("Verifying Redis and flushing stale data...")
	redisPod, err := ensureRedisReady(ctx, k8sClient, log)
	if err != nil {
		return err
	}
	if err := execRedisCommands(ctx, k8sClient, redisPod, "FLUSHALL\n", log); err != nil {
		return err
	}

	// 2. Seed inventory
	log.Step("Seeding inventory...")
	if err := execRedisCommands(ctx, k8sClient, redisPod, generateInventoryCommands(), log); err != nil {
		return err
	}

	// 3. Generate order data via Gemini
	log.Step("Generating order data via Gemini...")
	orders, err := generateOrdersWithGemini(ctx, infra.ProjectID, infra.Region, log)
	if err != nil {
		return fmt.Errorf("failed to generate order data: %w", err)
	}
	log.Info("Generated %d orders", len(orders))

	// 4. Generate cake images via commerce pod
	log.Step("Generating cake images via Gemini 2.5 Flash Image...")
	commercePod, err := getAgentPod(ctx, k8sClient, "commerce", log)
	if err != nil {
		return fmt.Errorf("commerce pod not found (is deploy-agents done?): %w", err)
	}

	for i, order := range orders {
		log.Info("  Generating image for order %d/%d: %s", i+1, len(orders), order.OrderID)
		imagePath, err := generateImageViaPod(ctx, k8sClient, commercePod, order, infra.Bucket, log)
		if err != nil {
			// Non-fatal: seed continues with placeholder path
			log.Info("  Warning: image generation failed for %s: %v", order.OrderID, err)
			imagePath = fmt.Sprintf("gs://%s/cakes/orders/%s/cake_1.png", infra.Bucket, order.OrderID)
		}
		orders[i].ImagePath = imagePath
	}

	// 5. Write orders to Redis
	log.Step("Writing orders to Redis...")
	orderCmds := buildOrderRedisCommands(orders)
	if err := execRedisCommands(ctx, k8sClient, redisPod, orderCmds, log); err != nil {
		return err
	}

	log.Info("Seed data complete!")
	log.Info("  Flushed stale Redis data")
	log.Info("  Inventory: 5 ingredients seeded")
	log.Info("  Orders: %d orders with cake images", len(orders))
	return nil
}

// seedOrder holds one generated order record.
type seedOrder struct {
	OrderID      string
	CustomerName string
	Street       string
	Postcode     string
	DeliveryDate string
	Flavor       string
	Nuts         string
	PeopleCount  int
	Concept      string
	ImagePath    string
}

// generateOrdersWithGemini calls the Gemini text API to produce varied order data.
func generateOrdersWithGemini(ctx context.Context, projectID, region string, log *logger.Logger) ([]seedOrder, error) {
	// Get access token via ADC
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}
	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Delivery dates: today+1 (3 orders), today+2 (1 order), today+3 (1 order)
	now := time.Now()
	d1 := now.AddDate(0, 0, 1).Format("2006-01-02")
	d2 := now.AddDate(0, 0, 2).Format("2006-01-02")
	d3 := now.AddDate(0, 0, 3).Format("2006-01-02")
	datesJSON, _ := json.Marshal([]string{d1, d1, d1, d2, d3})

	prompt := fmt.Sprintf(`Generate exactly 5 fictional cake orders for a bakery in Amsterdam.
Return ONLY a valid JSON array (no explanation, no markdown), each element with these exact fields:
- customer_name: realistic first + last name, mix of Dutch names (Jan de Vries, Maria van den Berg) and international (John Doe, Alice Lee)
- street: a real Amsterdam street with house number (e.g. "Herengracht 12")
- postcode: valid Amsterdam format "NNNN XX" where NNNN is 1000-1109 (e.g. "1013 AP")
- flavor: one of chocolate, ananas, banana
- nuts: one of almond, walnut, none
- people_count: integer between 6 and 50
- concept: short theme like birthday, wedding, anniversary, baby shower, graduation, Star Wars, dinosaurs

The 5 orders must use these delivery dates in order: %s
Return only the raw JSON array.`, string(datesJSON))

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"role": "user", "parts": []map[string]string{{"text": prompt}}},
		},
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Use Vertex AI endpoint (cloud-platform scope, consistent with rest of project)
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/gemini-2.0-flash-001:generateContent", region, projectID, region)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Gemini API call failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Gemini response: %w\nRaw: %s", err, string(bodyBytes))
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty Gemini response body: %s", string(bodyBytes))
	}

	rawJSON := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	// Strip markdown code fences if Gemini wrapped it
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var geminiOrders []struct {
		CustomerName string `json:"customer_name"`
		Street       string `json:"street"`
		Postcode     string `json:"postcode"`
		Flavor       string `json:"flavor"`
		Nuts         string `json:"nuts"`
		PeopleCount  int    `json:"people_count"`
		Concept      string `json:"concept"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &geminiOrders); err != nil {
		return nil, fmt.Errorf("failed to parse generated orders JSON: %w\nRaw: %s", err, rawJSON)
	}

	// Delivery dates: 3 + 1 + 1
	now = time.Now()
	dates := []string{
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.AddDate(0, 0, 2).Format("2006-01-02"),
		now.AddDate(0, 0, 3).Format("2006-01-02"),
	}

	orders := make([]seedOrder, 0, len(geminiOrders))
	for i, g := range geminiOrders {
		if i >= 5 {
			break
		}
		date := dates[i]
		dateCompact := strings.ReplaceAll(date, "-", "")
		suffix := strings.ToUpper(uuid.New().String()[:4])
		orders = append(orders, seedOrder{
			OrderID:      fmt.Sprintf("CAKE-%s-%s", dateCompact, suffix),
			CustomerName: g.CustomerName,
			Street:       g.Street,
			Postcode:     g.Postcode,
			DeliveryDate: date,
			Flavor:       g.Flavor,
			Nuts:         g.Nuts,
			PeopleCount:  g.PeopleCount,
			Concept:      g.Concept,
		})
	}

	return orders, nil
}

// getAgentPod returns the name of a running pod for the given agent app label.
func getAgentPod(ctx context.Context, client *k8s.Client, appLabel string, log *logger.Logger) (string, error) {
	podName, err := client.GetFirstPodByLabel(ctx, "agents", "app="+appLabel)
	if err != nil {
		return "", err
	}
	return podName, nil
}

// generateImageViaPod execs into the commerce pod and calls generate_cake_image().
func generateImageViaPod(ctx context.Context, client *k8s.Client, podName string, order seedOrder, bucket string, log *logger.Logger) (string, error) {
	// Build a Python one-liner that generates and uploads the image
	pythonScript := fmt.Sprintf(
		`import os; os.environ.setdefault('GCS_BUCKET','%s'); `+
			`from commerce.tools.gemini_image import generate_cake_image; `+
			`print(generate_cake_image('%s','%s',%d,'%s','%s',1))`,
		bucket,
		order.Flavor, order.Nuts, order.PeopleCount, order.Concept, order.OrderID,
	)

	stdout, stderr, err := client.Exec(
		ctx,
		"agents",
		podName,
		"",
		[]string{"python3", "-c", pythonScript},
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("exec failed: %w\nStderr: %s", err, stderr)
	}

	gcsPath := strings.TrimSpace(stdout)
	if gcsPath == "" {
		return "", fmt.Errorf("empty GCS path returned")
	}
	return gcsPath, nil
}

// buildOrderRedisCommands produces redis-cli HSET commands for all orders.
func buildOrderRedisCommands(orders []seedOrder) string {
	var sb strings.Builder
	createdAt := time.Now().Format(time.RFC3339)

	for _, o := range orders {
		key := fmt.Sprintf("order:%s", o.OrderID)
		cakesJSON := fmt.Sprintf(`[{"flavor":"%s","nuts":"%s","people_count":%d,"concept":"%s"}]`,
			o.Flavor, o.Nuts, o.PeopleCount, o.Concept)
		imagePathsJSON := fmt.Sprintf(`["%s"]`, o.ImagePath)

		people := o.PeopleCount
		totalCakePrice := float64(people) * 5.0
		deliveryFee := 5.0
		if totalCakePrice >= 50.0 {
			deliveryFee = 0.0
		}
		totalPrice := totalCakePrice + deliveryFee

		sb.WriteString(fmt.Sprintf(
			"HSET %s order_id \"%s\" customer_name \"%s\" address \"%s\" postcode \"%s\" "+
				"delivery_date \"%s\" total_cake_price \"%.1f\" delivery_fee \"%.1f\" total_price \"%.1f\" "+
				"status \"confirmed\" created_at \"%s\" cakes '%s' image_paths '%s'\n",
			key, o.OrderID, o.CustomerName, o.Street, o.Postcode, o.DeliveryDate,
			totalCakePrice, deliveryFee, totalPrice, createdAt, cakesJSON, imagePathsJSON,
		))
	}

	return sb.String()
}

// seedDataGcloud is a fallback for image upload if pod exec fails.
func uploadImageGcloud(ctx context.Context, localPath, dest string) error {
	cmd := exec.CommandContext(ctx, "gcloud", "storage", "cp", localPath, dest)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gcloud cp failed: %s\n%s", err, out)
	}
	return nil
}

// ensureRedisReady verifies the Redis deployment and returns the name of a running pod.
func ensureRedisReady(ctx context.Context, client *k8s.Client, log *logger.Logger) (string, error) {
	log.Step("Verifying Redis deployment")
	if err := client.WaitForDeploymentReady(ctx, AgentsNamespace, "redis", 2*time.Minute); err != nil {
		return "", fmt.Errorf("redis deployment not ready: %w", err)
	}

	podName, err := client.GetFirstPodByLabel(ctx, AgentsNamespace, "app=redis")
	if err != nil {
		return "", fmt.Errorf("failed to find redis pod: %w", err)
	}
	log.Debug("Targeting Redis pod: %s", podName)
	return podName, nil
}

// execRedisCommands pipes a payload of redis-cli commands into the Redis pod.
// Agents use the shared Redis instance in the 'agents' namespace.
func execRedisCommands(ctx context.Context, client *k8s.Client, podName, payload string, log *logger.Logger) error {
	_, stderr, err := client.Exec(
		ctx,
		AgentsNamespace,
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

// generateInventoryCommands returns redis-cli HSET commands to set known inventory quantities.
// chocolate: 4, ananas: 1 (LOW), banana: 3, walnut: 2 (LOW), almond: 4.
func generateInventoryCommands() string {
	var sb strings.Builder
	sb.WriteString("HSET inventory:chocolate quantity 4\n")
	sb.WriteString("HSET inventory:ananas quantity 1\n")
	sb.WriteString("HSET inventory:banana quantity 3\n")
	sb.WriteString("HSET inventory:walnut quantity 2\n")
	sb.WriteString("HSET inventory:almond quantity 4\n")
	return sb.String()
}
