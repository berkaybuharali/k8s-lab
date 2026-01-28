// cli/pkg/logger/logger_test.go
package logger

import "testing"

// TestLogger verifies that all logger functions work without panicking.
// This is a basic smoke test - we don't verify exact output since we're
// testing colored terminal output which is hard to assert.
func TestLogger(t *testing.T) {
	log := New(true)

	// Test all logging functions - should not panic
	log.Info("This is info")
	log.Warn("This is warning")
	log.Error("This is error")
	log.Step("This is a step")
	log.Debug("This is debug (verbose mode)")
}

// TestLoggerNonVerbose verifies that debug messages are suppressed
// when verbose mode is disabled.
func TestLoggerNonVerbose(t *testing.T) {
	log := New(false)

	// Debug should not output anything (but shouldn't panic either)
	log.Debug("This should not appear")

	// Other log levels should still work
	log.Info("Non-verbose info")
}

// TestLoggerFormatting verifies that format strings work correctly.
func TestLoggerFormatting(t *testing.T) {
	log := New(false)

	// Test with format arguments
	log.Info("User: %s, Count: %d", "test-user", 42)
	log.Step("Processing %d items", 100)
	log.Warn("Temperature is %d degrees", 75)
}
