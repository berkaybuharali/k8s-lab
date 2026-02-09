package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/berkaybuharali/k8s-lab/cli/pkg/ui"
)

var uiPort int
var uiBrowser string

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the web dashboard",
	Long: `Start the web dashboard for managing the Kubernetes lab.

This command starts a local web server that provides a visual interface
to all cluster lifecycle operations (deploy, destroy, backup, restore).

The UI will automatically open in your default browser.
Keep this terminal open while using the UI.`,
	RunE: runUI,
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().IntVarP(&uiPort, "port", "p", 3000, "Port to run the UI on")
	uiCmd.Flags().StringVarP(&uiBrowser, "browser", "b", "", "Browser to open (e.g. chrome, firefox). Uses system default if not set")
}

func runUI(cmd *cobra.Command, args []string) error {
	cfg := GetConfig(cmd)
	log := GetLogger(cmd)
	provider := GetProvider(cmd)

	// UI needs cloud provider for operations
	if err := RequireCloud(provider); err != nil {
		return err
	}

	// Create server
	server, err := ui.NewServer(uiPort, provider.Name(), cfg, log, provider)
	if err != nil {
		return err
	}

	// Handle OS signals
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Open browser (best effort)
	go func() {
		time.Sleep(500 * time.Millisecond) // Wait for server to start
		url := fmt.Sprintf("http://localhost:%d", uiPort)
		log.Info("Opening %s in browser...", url)
		openBrowser(url, uiBrowser)
	}()

	return server.Start(ctx)
}

// browserApps maps short names to macOS application names.
var browserApps = map[string]string{
	"chrome":  "Google Chrome",
	"firefox": "Firefox",
	"safari":  "Safari",
	"edge":    "Microsoft Edge",
	"brave":   "Brave Browser",
	"arc":     "Arc",
}

func openBrowser(url string, browser string) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		if browser != "" {
			app := browser
			if mapped, ok := browserApps[browser]; ok {
				app = mapped
			}
			err = exec.Command("open", "-a", app, url).Start()
		} else {
			err = exec.Command("open", url).Start()
		}
	case "linux":
		if browser != "" {
			err = exec.Command(browser, url).Start()
		} else {
			err = exec.Command("xdg-open", url).Start()
		}
	case "windows":
		if browser != "" {
			err = exec.Command("cmd", "/c", "start", browser, url).Start()
		} else {
			err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		}
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}
}
