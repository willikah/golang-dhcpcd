//go:build unit

package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManagerAdapter(t *testing.T) {
	adapter := NewManagerAdapter()
	assert.NotNil(t, adapter)
}

func TestManagerAdapter_GetLinkByName(t *testing.T) {
	adapter := NewManagerAdapter()
	
	t.Run("ValidInterface", func(t *testing.T) {
		// Test with loopback interface which should exist on most systems
		link, err := adapter.GetLinkByName("lo")
		if err != nil {
			t.Skip("Loopback interface not available, skipping test")
		}
		assert.NoError(t, err)
		assert.NotNil(t, link)
		assert.Equal(t, "lo", link.Attrs().Name)
	})

	t.Run("InvalidInterface", func(t *testing.T) {
		_, err := adapter.GetLinkByName("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get netlink interface")
	})
}

func TestManagerAdapter_ListAddresses(t *testing.T) {
	adapter := NewManagerAdapter()
	
	// Test with loopback interface which should exist on most systems
	link, err := adapter.GetLinkByName("lo")
	if err != nil {
		t.Skip("Loopback interface not available, skipping test")
	}

	addresses, err := adapter.ListAddresses(link)
	assert.NoError(t, err)
	assert.NotNil(t, addresses)
	// Loopback typically has at least 127.0.0.1
}

// Note: AddAddress, DeleteAddress, AddRoute, DeleteRoute, and SetLinkUp 
// require elevated privileges and would modify system state, so they're 
// better tested in integration tests rather than unit tests.
// These tests just verify the adapter creation.

func TestManagerAdapter_Methods_Exist(t *testing.T) {
	adapter := NewManagerAdapter()
	
	// Just verify the adapter was created successfully
	// The actual network operations require real interfaces and privileges
	assert.NotNil(t, adapter)
	
	// We could test with mock interfaces, but netlink.Link is an interface
	// and creating proper mocks would be complex for unit tests
	// Integration tests would be more appropriate for these methods
}
