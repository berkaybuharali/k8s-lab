// cli/pkg/prerequisites/prerequisite_test.go
package prerequisites

import (
	"context"
	"strings"
	"testing"
)

// TestBinaryPrerequisite_Name verifies Name() returns correct name
func TestBinaryPrerequisite_Name(t *testing.T) {
	prereq := &BinaryPrerequisite{
		name:       "Test Tool",
		binaryName: "testtool",
	}

	if prereq.Name() != "Test Tool" {
		t.Errorf("Expected 'Test Tool', got '%s'", prereq.Name())
	}
}

// TestBinaryPrerequisite_Required verifies Required() returns correct value
func TestBinaryPrerequisite_Required(t *testing.T) {
	prereq := &BinaryPrerequisite{
		required: true,
	}

	if !prereq.Required() {
		t.Error("Expected Required() to return true")
	}
}

// TestBinaryPrerequisite_Check_Exists verifies Check succeeds for existing binary
func TestBinaryPrerequisite_Check_Exists(t *testing.T) {
	// Use 'ls' which should exist on all Unix systems
	prereq := &BinaryPrerequisite{
		name:       "ls",
		binaryName: "ls",
	}

	ctx := context.Background()
	if err := prereq.Check(ctx); err != nil {
		t.Errorf("Expected Check() to succeed for 'ls', got error: %v", err)
	}
}

// TestBinaryPrerequisite_Check_NotExists verifies Check fails for missing binary
func TestBinaryPrerequisite_Check_NotExists(t *testing.T) {
	prereq := &BinaryPrerequisite{
		name:        "NonExistentTool",
		binaryName:  "this-binary-definitely-does-not-exist-12345",
		installHint: "brew install fake",
	}

	ctx := context.Background()
	err := prereq.Check(ctx)
	if err == nil {
		t.Error("Expected Check() to fail for non-existent binary")
	}

	// Error should contain install hint
	if err != nil && !strings.Contains(err.Error(), "brew install fake") {
		t.Errorf("Expected error to contain install hint, got: %v", err)
	}
}

// TestGetCommandPrereqs verifies command prerequisites are returned correctly
func TestGetCommandPrereqs(t *testing.T) {
	tests := []struct {
		name     string
		cmdName  string
		expected int // Number of prerequisites expected
	}{
		{"infra command", "infra", 1},       // terraform only (Talos uses SDK)
		{"platform command", "platform", 1}, // kubectl
		{"unknown command", "unknown", 0},   // no prerequisites
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prereqs := GetCommandPrereqs(tt.cmdName)
			if len(prereqs) != tt.expected {
				t.Errorf("Expected %d prerequisites, got %d", tt.expected, len(prereqs))
			}
		})
	}
}

// TestGetCloudPrereqs verifies cloud prerequisites are returned correctly
func TestGetCloudPrereqs(t *testing.T) {
	tests := []struct {
		name      string
		cloudName string
		expected  int
	}{
		{"gcp cloud", "gcp", 1},     // gcloud
		{"empty cloud", "", 0},      // no prerequisites
		{"unknown cloud", "foo", 0}, // no prerequisites
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prereqs := GetCloudPrereqs(tt.cloudName)
			if len(prereqs) != tt.expected {
				t.Errorf("Expected %d prerequisites, got %d", tt.expected, len(prereqs))
			}
		})
	}
}

// TestCheckAll_AllSatisfied verifies CheckAll succeeds when all prerequisites exist
func TestCheckAll_AllSatisfied(t *testing.T) {
	ctx := context.Background()

	// Use binaries that should exist on all Unix systems
	prereqs := []Prerequisite{
		&BinaryPrerequisite{name: "ls", binaryName: "ls"},
		&BinaryPrerequisite{name: "cat", binaryName: "cat"},
	}

	err := CheckAll(ctx, prereqs...)
	if err != nil {
		t.Errorf("Expected CheckAll to succeed, got error: %v", err)
	}
}

// TestCheckAll_SomeMissing verifies CheckAll lists all missing prerequisites
func TestCheckAll_SomeMissing(t *testing.T) {
	ctx := context.Background()

	prereqs := []Prerequisite{
		&BinaryPrerequisite{name: "ls", binaryName: "ls"}, // exists
		&BinaryPrerequisite{name: "fake1", binaryName: "nonexistent1"},
		&BinaryPrerequisite{name: "fake2", binaryName: "nonexistent2"},
	}

	err := CheckAll(ctx, prereqs...)
	if err == nil {
		t.Fatal("Expected CheckAll to fail with missing prerequisites")
	}

	// Error should mention both missing tools
	errMsg := err.Error()
	if !strings.Contains(errMsg, "fake1") || !strings.Contains(errMsg, "fake2") {
		t.Errorf("Expected error to list all missing prerequisites, got: %v", err)
	}
}

// TestCheckAll_Empty verifies CheckAll succeeds with no prerequisites
func TestCheckAll_Empty(t *testing.T) {
	ctx := context.Background()
	err := CheckAll(ctx)
	if err != nil {
		t.Errorf("Expected CheckAll with no prerequisites to succeed, got: %v", err)
	}
}

// TestPrerequisitesDefinitions verifies all pre-defined prerequisites are properly configured
func TestPrerequisitesDefinitions(t *testing.T) {
	// Verify command-specific tools (Talosctl removed - using Go SDK instead)
	commandTools := []Prerequisite{Terraform, Kubectl, Velero}
	for _, tool := range commandTools {
		if tool.Name() == "" {
			t.Errorf("Tool has empty name: %v", tool)
		}
		if !tool.Required() {
			t.Errorf("Expected %s to be required", tool.Name())
		}
	}

	// Verify cloud-specific tools
	cloudTools := []Prerequisite{Gcloud}
	for _, tool := range cloudTools {
		if tool.Name() == "" {
			t.Errorf("Cloud tool has empty name: %v", tool)
		}
		if !tool.Required() {
			t.Errorf("Expected %s to be required", tool.Name())
		}
	}
}
