// Package port defines the primary ports (interfaces) for the application.
// This follows the Ports and Adapters (Hexagonal Architecture) pattern.
package port

import (
	"context"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/vishvananda/netlink"
)

// DHCPClient is a port for DHCP client operations.
// This interface abstracts DHCP lease acquisition and management.
type DHCPClient interface {
	// RequestLease performs DHCP DISCOVER/OFFER/REQUEST/ACK sequence
	RequestLease(ctx context.Context, interfaceName string, timeout time.Duration) (*dhcpv4.DHCPv4, error)
}

// NetworkManager is a port for network interface operations.
// This interface abstracts netlink operations for network configuration.
type NetworkManager interface {
	// GetLinkByName returns a network link by interface name
	GetLinkByName(interfaceName string) (netlink.Link, error)

	// ListAddresses returns IPv4 addresses configured on the link
	ListAddresses(link netlink.Link) ([]netlink.Addr, error)

	// AddAddress adds an IP address to the interface
	AddAddress(link netlink.Link, addr *netlink.Addr) error

	// DeleteAddress removes an IP address from the interface
	DeleteAddress(link netlink.Link, addr *netlink.Addr) error

	// ListRoutes returns IPv4 routes
	ListRoutes() ([]netlink.Route, error)

	// AddRoute adds a route
	AddRoute(route *netlink.Route) error

	// DeleteRoute removes a route
	DeleteRoute(route *netlink.Route) error

	// SetLinkUp brings the interface up
	SetLinkUp(link netlink.Link) error
}

// FileManager is a port for file system operations.
// This interface abstracts file read/write operations.
type FileManager interface {
	// ReadFile reads the contents of a file
	ReadFile(filename string) ([]byte, error)

	// WriteFile writes data to a file with specified permissions
	WriteFile(filename string, data []byte, perm int) error

	// FileExists checks if a file exists
	FileExists(filename string) bool
}
