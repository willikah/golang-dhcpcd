// Package dhcp provides DHCP client adapter implementation.
package dhcp

import (
	"context"
	"fmt"
	"time"

	"golang-dhcpcd/internal/port"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
)

// ClientAdapter is an adapter that implements the DHCPClient port using insomniacslk/dhcp library.
type ClientAdapter struct{}

// Ensure ClientAdapter implements the DHCPClient port
var _ port.DHCPClient = (*ClientAdapter)(nil)

// NewClientAdapter creates a new DHCP client adapter.
func NewClientAdapter() *ClientAdapter {
	return &ClientAdapter{}
}

// RequestLease performs the complete DHCP DISCOVER/OFFER/REQUEST/ACK sequence.
func (c *ClientAdapter) RequestLease(ctx context.Context, interfaceName string, timeout time.Duration) (*dhcpv4.DHCPv4, error) {
	// Create DHCP client
	client, err := nclient4.New(interfaceName, nclient4.WithTimeout(timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to create DHCP client: %w", err)
	}
	defer client.Close()

	// Get lease (DISCOVER/OFFER/REQUEST/ACK)
	lease, err := client.Request(ctx)
	if err != nil {
		return nil, fmt.Errorf("DHCP lease request failed: %w", err)
	}

	return lease.ACK, nil
}
