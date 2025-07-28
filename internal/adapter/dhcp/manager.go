package dhcp

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"golang-dhcpcd/internal/pkg/logging"
	"golang-dhcpcd/internal/port"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Manager is a DHCP network configuration adapter that implements the NetworkConfigurationManager port.
// It handles DHCP lease management for a network interface following the Ports and Adapters pattern.
type Manager struct {
	iface      *net.Interface
	dhcpClient port.DHCPClient
	networkMgr port.NetworkManager
	fileMgr    port.FileManager
}

// Ensure Manager implements the NetworkConfigurationManager port
var _ port.NetworkConfigurationManager = (*Manager)(nil)

// NewManager creates a new DHCP network configuration adapter for the given interface name.
func NewManager(ifaceName string, dhcpClient port.DHCPClient, networkMgr port.NetworkManager, fileMgr port.FileManager) (*Manager, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("interface not found: %w", err)
	}

	return &Manager{
		iface:      iface,
		dhcpClient: dhcpClient,
		networkMgr: networkMgr,
		fileMgr:    fileMgr,
	}, nil
}

// GetInterfaceName returns the name of the network interface managed by this manager.
func (m *Manager) GetInterfaceName() string {
	return m.iface.Name
}

// Run starts and maintains DHCP lease on the interface using the nclient4 library.
// It runs until the context is cancelled.
func (m *Manager) Run(ctx context.Context) error {
	logger := logging.WithComponentAndInterface("dhcp", m.iface.Name).WithField("mac", m.iface.HardwareAddr.String())
	logger.Info("Starting DHCP manager")

	// Start with immediate lease acquisition by using a short timer
	renewalTimer := time.NewTimer(1 * time.Millisecond)
	defer renewalTimer.Stop()

	for {
		// Single channel select for all timing logic
		select {
		case <-ctx.Done():
			logger.Info("DHCP manager stopped due to context cancellation")
			return ctx.Err()
		case <-renewalTimer.C:
			// Get DHCP lease
			lease, err := m.getDHCPLease(ctx, logger)
			if err != nil {
				logger.WithError(err).Error("Failed to get DHCP lease, retrying in 30s")
				renewalTimer.Reset(30 * time.Second)
			} else {
				// Apply lease to interface
				if err := m.applyDHCPLease(ctx, lease); err != nil {
					logger.WithError(err).Error("Failed to apply DHCP lease")
				} else {
					logger.Info("Successfully configured interface")
				}

				// Set up renewal timer
				renewal := lease.IPAddressRenewalTime(30 * time.Second)
				logger.WithField("renewal_time", renewal.String()).Info("Sleeping until renewal")
				renewalTimer.Reset(renewal)
			}
		}
	}
}

// getDHCPLease performs the complete DHCP DISCOVER/OFFER/REQUEST/ACK sequence
func (m *Manager) getDHCPLease(ctx context.Context, logger *logrus.Entry) (*dhcpv4.DHCPv4, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.WithField("attempt", fmt.Sprintf("%d/%d", attempt, maxRetries)).Debug("Attempting DHCP lease")

		// Get lease using the DHCP client port
		ack, err := m.dhcpClient.RequestLease(ctx, m.iface.Name, 15*time.Second)
		if err != nil {
			logger.WithError(err).WithField("attempt", attempt).Error("DHCP lease request failed")
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("DHCP lease request failed after %d attempts: %w", maxRetries, err)
		}

		logger.WithField("ip", ack.YourIPAddr.String()).Info("Successfully obtained DHCP lease")
		return ack, nil
	}

	return nil, fmt.Errorf("failed to get DHCP lease after %d attempts", maxRetries)
}

// applyDHCPLease configures the network interface with the received DHCP lease using netlink
func (m *Manager) applyDHCPLease(ctx context.Context, ack *dhcpv4.DHCPv4) error {
	logger := logging.WithComponentAndInterface("dhcp", m.iface.Name)

	// Extract network configuration from DHCP ACK
	ipAddr := ack.YourIPAddr

	// Get subnet mask
	subnetMask := ack.SubnetMask()
	if subnetMask == nil {
		// Default to /24 if no subnet mask provided
		subnetMask = net.IPv4Mask(255, 255, 255, 0)
	}

	// Create IP network
	ipNet := &net.IPNet{
		IP:   ipAddr,
		Mask: subnetMask,
	}

	logger.WithField("ip", ipNet.String()).Info("Configuring interface with IP")

	// Get netlink interface using network manager port
	link, err := m.networkMgr.GetLinkByName(m.iface.Name)
	if err != nil {
		return fmt.Errorf("failed to get netlink interface: %w", err)
	}

	// Get existing addresses to check for duplicates
	existingAddrs, err := m.networkMgr.ListAddresses(link)
	if err != nil {
		return fmt.Errorf("failed to list existing addresses: %w", err)
	}

	// Check if the target IP is already configured
	targetConfigured := false
	for _, addr := range existingAddrs {
		if addr.IPNet.IP.Equal(ipNet.IP) && addr.IPNet.Mask.String() == ipNet.Mask.String() {
			logger.WithField("ip", ipNet.String()).Info("IP address already configured, skipping")
			targetConfigured = true
			break
		}
	}

	// Only remove existing addresses if target IP is not already configured
	if !targetConfigured {
		// Remove existing IPv4 addresses that don't match our target
		for _, addr := range existingAddrs {
			if !addr.IPNet.IP.Equal(ipNet.IP) {
				if err := m.networkMgr.DeleteAddress(link, &addr); err != nil {
					logger.WithError(err).WithField("address", addr.IPNet.String()).Warn("Failed to remove existing address")
				} else {
					logger.WithField("address", addr.IPNet.String()).Debug("Removed existing address")
				}
			}
		}
	}

	// Get lease time from DHCP ACK
	leaseTime := ack.IPAddressLeaseTime(60 * time.Second) // Default to 60 seconds if not specified
	logger.WithField("lease_time", leaseTime.String()).Debug("Lease time extracted")

	// Add new IP address only if not already configured
	if !targetConfigured {
		addr := &netlink.Addr{
			IPNet:       ipNet,
			ValidLft:    int(leaseTime.Seconds()),
			PreferedLft: int(leaseTime.Seconds()),
		}
		if err := m.networkMgr.AddAddress(link, addr); err != nil {
			return fmt.Errorf("failed to add IP address %s: %w", ipNet.String(), err)
		}
		logger.WithField("ip", ipNet.String()).Info("Successfully added IP address")
	}

	// Configure default gateway if provided
	routers := ack.Router()
	if len(routers) > 0 {
		gateway := routers[0]
		logger.WithField("gateway", gateway.String()).Info("Setting default gateway")

		if err := m.configureDefaultRoute(ctx, link, gateway); err != nil {
			return fmt.Errorf("failed to set default gateway: %w", err)
		}
	}

	// Log DNS servers if provided
	dnsServers := ack.DNS()
	if len(dnsServers) > 0 {
		var dnsStrings []string
		for _, dns := range dnsServers {
			dnsStrings = append(dnsStrings, dns.String())
		}
		logger.WithField("dns_servers", strings.Join(dnsStrings, ", ")).Info("DNS servers received")

		// Configure DNS (write to /etc/resolv.conf)
		if err := m.configureDNS(ctx, dnsServers); err != nil {
			logger.WithError(err).Warn("Failed to configure DNS")
		}
	}

	return nil
}

// configureDefaultRoute configures the default route using netlink
func (m *Manager) configureDefaultRoute(ctx context.Context, link netlink.Link, gateway net.IP) error {
	logger := logging.WithComponentAndInterface("dhcp", m.iface.Name).WithField("gateway", gateway.String())

	// List existing routes to check if our desired route already exists
	routes, err := m.networkMgr.ListRoutes()
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	// Check if the desired default route already exists
	targetRouteExists := false
	for _, route := range routes {
		if (route.Dst == nil || route.Dst.String() == "0.0.0.0/0") &&
			route.Gw != nil && route.Gw.Equal(gateway) &&
			route.LinkIndex == link.Attrs().Index {
			logger.Info("Default route already exists, skipping")
			targetRouteExists = true
			break
		}
	}

	// Only modify routes if the target route doesn't exist
	if !targetRouteExists {
		// Remove existing default routes that don't match our target
		for _, route := range routes {
			if route.Dst == nil || route.Dst.String() == "0.0.0.0/0" {
				// Skip if this is already our desired route
				if route.Gw != nil && route.Gw.Equal(gateway) && route.LinkIndex == link.Attrs().Index {
					continue
				}

				if err := m.networkMgr.DeleteRoute(&route); err != nil {
					logger.WithError(err).Warn("Failed to remove existing default route")
				} else {
					if route.Gw != nil {
						logger.WithField("old_gateway", route.Gw.String()).Debug("Removed existing default route")
					} else {
						logger.Debug("Removed existing default route")
					}
				}
			}
		}

		// Add new default route
		route := &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Gw:        gateway,
		}

		if err := m.networkMgr.AddRoute(route); err != nil {
			return fmt.Errorf("failed to add default route: %w", err)
		}

		logger.Info("Successfully added default route")
	}

	return nil
}

// configureDNS writes DNS servers to /etc/resolv.conf
func (m *Manager) configureDNS(ctx context.Context, dnsServers []net.IP) error {
	logger := logging.WithComponentAndInterface("dhcp", m.iface.Name)

	// Generate the new DNS configuration content
	newContent := "# Generated by golang-dhcpcd\n"
	for _, dns := range dnsServers {
		newContent += fmt.Sprintf("nameserver %s\n", dns.String())
	}

	// Check if the current /etc/resolv.conf already has the same content
	if currentContent, err := m.fileMgr.ReadFile("/etc/resolv.conf"); err == nil {
		if string(currentContent) == newContent {
			logger.Debug("DNS configuration already up to date, skipping")
			return nil
		}
	}

	// Write the new DNS configuration
	if err := m.fileMgr.WriteFile("/etc/resolv.conf", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/resolv.conf: %w", err)
	}

	logger.Info("Updated /etc/resolv.conf with DNS servers")
	return nil
}
