// Package network provides network management adapter implementation.
package network

import (
	"fmt"

	"golang-dhcpcd/internal/port"

	"github.com/vishvananda/netlink"
)

// ManagerAdapter is an adapter that implements the NetworkManager port using vishvananda/netlink library.
type ManagerAdapter struct{}

// Ensure ManagerAdapter implements the NetworkManager port
var _ port.NetworkManager = (*ManagerAdapter)(nil)

// NewManagerAdapter creates a new network manager adapter.
func NewManagerAdapter() *ManagerAdapter {
	return &ManagerAdapter{}
}

// GetLinkByName returns a network link by interface name.
func (n *ManagerAdapter) GetLinkByName(interfaceName string) (netlink.Link, error) {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get netlink interface %s: %w", interfaceName, err)
	}
	return link, nil
}

// ListAddresses returns IPv4 addresses configured on the link.
func (n *ManagerAdapter) ListAddresses(link netlink.Link) ([]netlink.Addr, error) {
	addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses: %w", err)
	}
	return addrs, nil
}

// AddAddress adds an IP address to the interface.
func (n *ManagerAdapter) AddAddress(link netlink.Link, addr *netlink.Addr) error {
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to add address %s: %w", addr.IPNet.String(), err)
	}
	return nil
}

// DeleteAddress removes an IP address from the interface.
func (n *ManagerAdapter) DeleteAddress(link netlink.Link, addr *netlink.Addr) error {
	if err := netlink.AddrDel(link, addr); err != nil {
		return fmt.Errorf("failed to delete address %s: %w", addr.IPNet.String(), err)
	}
	return nil
}

// ListRoutes returns IPv4 routes.
func (n *ManagerAdapter) ListRoutes() ([]netlink.Route, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}
	return routes, nil
}

// AddRoute adds a route.
func (n *ManagerAdapter) AddRoute(route *netlink.Route) error {
	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}
	return nil
}

// DeleteRoute removes a route.
func (n *ManagerAdapter) DeleteRoute(route *netlink.Route) error {
	if err := netlink.RouteDel(route); err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	return nil
}

// SetLinkUp brings the interface up.
func (n *ManagerAdapter) SetLinkUp(link netlink.Link) error {
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set link up: %w", err)
	}
	return nil
}
