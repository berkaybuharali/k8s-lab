// cli/pkg/cloud/provider_test.go
package cloud

import (
	"context"
	"testing"
)

// TestRegistry verifies that the provider registry works correctly.
func TestRegistry(t *testing.T) {
	// Save original registry and restore after test
	// This prevents test pollution
	original := Registry
	defer func() { Registry = original }()

	// Start with clean registry
	Registry = make(map[string]Provider)

	// Create a mock provider for testing
	mock := &mockProvider{name: "test-cloud"}

	// Test Register
	Register("test-cloud", mock)

	// Test Get - should find the registered provider
	retrieved := Get("test-cloud")
	if retrieved == nil {
		t.Fatal("Expected to retrieve registered provider, got nil")
	}
	if retrieved.Name() != "test-cloud" {
		t.Errorf("Expected provider name 'test-cloud', got '%s'", retrieved.Name())
	}

	// Test Get - non-existent provider
	notFound := Get("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for non-existent provider")
	}

	// Test List
	names := List()
	if len(names) != 1 {
		t.Errorf("Expected 1 provider in list, got %d", len(names))
	}
	if names[0] != "test-cloud" {
		t.Errorf("Expected provider name 'test-cloud', got '%s'", names[0])
	}
}

// TestRegisterDuplicate verifies that registering duplicate providers panics.
func TestRegisterDuplicate(t *testing.T) {
	// Save original registry and restore after test
	original := Registry
	defer func() { Registry = original }()

	// Start with clean registry
	Registry = make(map[string]Provider)

	mock := &mockProvider{name: "test"}

	// First registration should succeed
	Register("test", mock)

	// Second registration should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when registering duplicate provider")
		}
	}()

	Register("test", mock) // Should panic
}

// mockProvider is a test implementation of the Provider interface.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Validate(ctx context.Context) error {
	return nil
}

func (m *mockProvider) EnsureStateBucket(ctx context.Context, bucket, project string) error {
	return nil
}

func (m *mockProvider) GetProjectID(dir string) (string, error) {
	return "test-project", nil
}

func (m *mockProvider) CreateTalosEndpoint(ctx context.Context, instance, zone, projectID string) (string, func(), error) {
	noopCleanup := func() {}
	return "localhost:50000", noopCleanup, nil
}

func (m *mockProvider) CreateK8sEndpoint(ctx context.Context, instance, zone, projectID string) (string, func(), error) {
	noopCleanup := func() {}
	return "localhost:6443", noopCleanup, nil
}
