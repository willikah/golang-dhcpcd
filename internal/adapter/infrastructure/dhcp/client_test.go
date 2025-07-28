//go:build unit

package dhcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClientAdapter(t *testing.T) {
	adapter := NewClientAdapter()
	assert.NotNil(t, adapter)
}

func TestClientAdapter_RequestLease(t *testing.T) {
	t.Skip("Skipping integration test - requires real network interface")
	
	// This test would require a real network interface and DHCP server
	// In a real test environment, you might want to test with a mock network interface
	// or use integration tests instead of unit tests for this functionality
	
	adapter := NewClientAdapter()
	ctx := context.Background()
	
	// This would fail in unit tests since "nonexistent" interface doesn't exist
	_, err := adapter.RequestLease(ctx, "nonexistent", 5*time.Second)
	assert.Error(t, err)
}
