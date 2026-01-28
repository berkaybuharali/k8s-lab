// cli/pkg/logger/logger.go
// Package logger provides colored, leveled logging for CLI output.
// It replicates the bash script logging functions (log_info, log_error, etc.)
// with ANSI color codes for terminal output.
package logger

import (
	"fmt"
	"io"
	"os"
)

// ANSI color codes for terminal output.
// These match the colors used in scripts/lib/common.sh.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
)

// Logger handles formatted output to stderr (like bash scripts).
// All output goes to stderr to keep stdout clean for data/JSON.
type Logger struct {
	out     io.Writer
	verbose bool
}

// New creates a new logger instance.
// By default, output goes to stderr (matching bash script behavior).
func New(verbose bool) *Logger {
	return &Logger{
		out:     os.Stderr,
		verbose: verbose,
	}
}

// Info logs an informational message in green.
// Equivalent to bash: log_info "message"
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s[INFO]%s %s\n", colorGreen, colorReset, msg)
}

// Warn logs a warning message in yellow.
// Equivalent to bash: log_warn "message"
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s[WARN]%s %s\n", colorYellow, colorReset, msg)
}

// Error logs an error message in red.
// Equivalent to bash: log_error "message"
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s[ERROR]%s %s\n", colorRed, colorReset, msg)
}

// Step logs a step/action message in blue.
// Equivalent to bash: log_step "message"
func (l *Logger) Step(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s[STEP]%s %s\n", colorBlue, colorReset, msg)
}

// Debug logs a debug message only if verbose mode is enabled.
// Use this for detailed information that's not needed in normal operation.
func (l *Logger) Debug(format string, args ...interface{}) {
	if !l.verbose {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s[DEBUG]%s %s\n", colorBlue, colorReset, msg)
}

// Fatal logs an error message and exits with code 1.
// Equivalent to bash: log_error "message" && exit 1
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.Error(format, args...)
	os.Exit(1)
}
