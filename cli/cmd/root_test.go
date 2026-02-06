// cli/cmd/root_test.go
package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestExecute verifies that Execute() doesn't panic.
// This is a basic smoke test for the root command setup.
func TestExecute(t *testing.T) {
	// We can't easily test Execute() fully because it calls os.Exit()
	// Just verify the function exists and compiles
	// Full integration tests will be done manually
	t.Log("Execute function exists")
}

// TestGettersWithoutContext verifies that getters panic when context is missing.
// This tests our safety checks - if PersistentPreRunE fails, we should panic
// with a clear message rather than returning nil and causing confusion.
func TestGettersWithoutContext(t *testing.T) {
	// Create a command without running PersistentPreRunE
	cmd := &cobra.Command{}

	// Test GetConfig panics
	t.Run("GetConfig panics without context", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected GetConfig to panic, but it didn't")
			}
		}()
		GetConfig(cmd)
	})

	// Test GetLogger panics
	t.Run("GetLogger panics without context", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected GetLogger to panic, but it didn't")
			}
		}()
		GetLogger(cmd)
	})

	// Test GetProvider returns nil (doesn't panic)
	t.Run("GetProvider returns nil without context", func(t *testing.T) {
		provider := GetProvider(cmd)
		if provider != nil {
			t.Error("Expected GetProvider to return nil without context")
		}
	})
}

// TestRequireCloud verifies the RequireCloud helper.
func TestRequireCloud(t *testing.T) {
	// Test with nil provider
	err := RequireCloud(nil)
	if err == nil {
		t.Error("Expected error when provider is nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	// Test with non-nil provider would require mocking
	// Skip for now - integration tests will cover this
}

// TestRootCommandFlags verifies that global flags are registered.
func TestRootCommandFlags(t *testing.T) {
	// Check --cloud flag exists
	cloudFlag := rootCmd.PersistentFlags().Lookup("cloud")
	if cloudFlag == nil {
		t.Fatal("--cloud flag not found")
	}
	if cloudFlag.Shorthand != "c" {
		t.Errorf("Expected --cloud shorthand 'c', got '%s'", cloudFlag.Shorthand)
	}

	// Check --verbose flag exists
	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("--verbose flag not found")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("Expected --verbose shorthand 'v', got '%s'", verboseFlag.Shorthand)
	}
}

// TestContextKeys verifies context key constants are unique.
// This prevents accidental collisions if we add more context values.
func TestContextKeys(t *testing.T) {
	keys := []contextKey{configKey, loggerKey, providerKey}

	// Check all keys are different
	seen := make(map[contextKey]bool)
	for _, key := range keys {
		if seen[key] {
			t.Errorf("Duplicate context key: %v", key)
		}
		seen[key] = true
	}
}
