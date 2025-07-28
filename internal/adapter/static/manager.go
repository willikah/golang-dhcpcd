package static

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"golang-dhcpcd/internal/pkg/config"
	"golang-dhcpcd/internal/pkg/logging"
	"golang-dhcpcd/internal/port"
	"golang-dhcpcd/internal/types"

	"github.com/vishvananda/netlink"
)

// Manager is a static IP network configuration adapter that implements the NetworkConfigurationManager port.
// It handles static IP configuration for a network interface following the Ports and Adapters pattern.
type Manager struct {
	iface        *net.Interface
	config       config.InterfaceConfig
	staticConfig types.StaticIPConfig
	networkMgr   port.NetworkManager
}

// Ensure Manager implements the NetworkConfigurationManager port
var _ port.NetworkConfigurationManager = (*Manager)(nil)

// NewManager creates a new static IP network configuration adapter for the given interface name and configuration.
func NewManager(ifaceName string, ifaceConfig config.InterfaceConfig, networkMgr port.NetworkManager) (*Manager, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("interface not found: %w", err)
	}

	if ifaceConfig.Static == nil {
		return nil, fmt.Errorf("interface configuration does not have static IP settings")
	}

	// Convert config.StaticConfig to types.StaticIPConfig
	staticConfig := types.StaticIPConfig{
		IPAddress: ifaceConfig.Static.IP,
		Netmask:   ifaceConfig.Static.Netmask,
		Gateway:   ifaceConfig.Static.Gateway,
	}

	// Validate configuration at creation time
	manager := &Manager{
		iface:        iface,
		config:       ifaceConfig,
		staticConfig: staticConfig,
		networkMgr:   networkMgr,
	}
	return manager, nil
}

// GetInterfaceName returns the name of the network interface managed by this manager.
func (m *Manager) GetInterfaceName() string {
	return m.iface.Name
}

// Run configures the interface with static IP settings and maintains the configuration.
// It runs until the context is cancelled. This method implements the NetworkConfigurationManager port.
func (m *Manager) Run(ctx context.Context) error {
	logger := logging.WithComponentAndInterface("static", m.iface.Name).WithField("mac", m.iface.HardwareAddr.String())
	logger.Info("Starting static IP configuration")

	// Apply static IP configuration
	if err := m.applyStaticConfig(ctx, m.staticConfig); err != nil {
		return fmt.Errorf("failed to apply static configuration: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"ip":      m.staticConfig.IPAddress,
		"netmask": m.staticConfig.Netmask,
		"gateway": m.staticConfig.Gateway,
	}).Info("Static IP configuration applied successfully")

	// Monitor interface status and reapply configuration if needed
	return m.monitorInterface(ctx, m.staticConfig)
}

// applyStaticConfig applies the static IP configuration to the interface using netlink.
func (m *Manager) applyStaticConfig(ctx context.Context, config types.StaticIPConfig) error {
	logger := logging.WithComponentAndInterface("static", m.iface.Name)

	// Get netlink interface using network manager port
	link, err := m.networkMgr.GetLinkByName(m.iface.Name)
	if err != nil {
		return fmt.Errorf("failed to get netlink interface: %w", err)
	}

	// Parse IP address and netmask
	ip := net.ParseIP(config.IPAddress)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", config.IPAddress)
	}

	mask := net.ParseIP(config.Netmask)
	if mask == nil {
		return fmt.Errorf("invalid netmask: %s", config.Netmask)
	}

	// Create IP network
	ipNet := &net.IPNet{
		IP:   ip,
		Mask: net.IPMask(mask.To4()),
	}

	logger.WithField("ip", ipNet.String()).Info("Configuring interface with IP")

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

	// Add new IP address only if not already configured
	if !targetConfigured {
		addr := &netlink.Addr{
			IPNet: ipNet,
		}
		if err := m.networkMgr.AddAddress(link, addr); err != nil {
			return fmt.Errorf("failed to add IP address %s: %w", ipNet.String(), err)
		}
		logger.WithField("ip", ipNet.String()).Info("Successfully added IP address")
	}

	// Configure default gateway if specified
	if config.Gateway != "" {
		gateway := net.ParseIP(config.Gateway)
		if gateway == nil {
			return fmt.Errorf("invalid gateway address: %s", config.Gateway)
		}

		logger.WithField("gateway", gateway.String()).Info("Setting default gateway")

		if err := m.configureDefaultRoute(ctx, link, gateway); err != nil {
			return fmt.Errorf("failed to set default gateway: %w", err)
		}
	}

	return nil
}

// configureDefaultRoute configures the default gateway for the interface.
func (m *Manager) configureDefaultRoute(ctx context.Context, link netlink.Link, gateway net.IP) error {
	logger := logging.WithComponentAndInterface("static", m.iface.Name).WithField("gateway", gateway.String())

	// Check if default route already exists with this gateway
	routes, err := m.networkMgr.ListRoutes()
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	// Check for existing default route
	hasDefaultRoute := false
	for _, route := range routes {
		// Check if this is a default route (0.0.0.0/0)
		if route.Dst == nil && route.Gw != nil {
			if route.Gw.Equal(gateway) && route.LinkIndex == link.Attrs().Index {
				logger.Debug("Default route already configured, skipping")
				hasDefaultRoute = true
				break
			} else {
				// Remove conflicting default route
				if err := m.networkMgr.DeleteRoute(&route); err != nil {
					logger.WithError(err).WithField("existing_gateway", route.Gw.String()).
						Warn("Failed to remove existing default route")
				} else {
					logger.WithField("existing_gateway", route.Gw.String()).
						Debug("Removed conflicting default route")
				}
			}
		}
	}

	// Add new default route if not already present
	if !hasDefaultRoute {
		route := &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Gw:        gateway,
		}

		if err := m.networkMgr.AddRoute(route); err != nil {
			// Check if the error is because the route already exists
			if strings.Contains(err.Error(), "file exists") {
				logger.WithField("gateway", gateway.String()).
					Debug("Default route already exists, ignoring error")
			} else {
				return fmt.Errorf("failed to add default route: %w", err)
			}
		} else {
			logger.Info("Successfully configured default route")
		}
	}

	return nil
}

// monitorInterface monitors the interface and reapplies configuration if needed.
func (m *Manager) monitorInterface(ctx context.Context, config types.StaticIPConfig) error {
	logger := logging.WithComponentAndInterface("static", m.iface.Name)
	logger.Info("Starting interface monitoring")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Interface monitoring stopped due to context cancellation")
			return ctx.Err()
		case <-ticker.C:
			if err := m.checkAndRepairConfiguration(ctx, config); err != nil {
				logger.WithError(err).Error("Configuration check failed")
			}
		}
	}
}

// checkAndRepairConfiguration checks if the static configuration is still applied and repairs if needed.
func (m *Manager) checkAndRepairConfiguration(ctx context.Context, config types.StaticIPConfig) error {
	logger := logging.WithComponentAndInterface("static", m.iface.Name)

	// Refresh interface information
	iface, err := net.InterfaceByName(m.iface.Name)
	if err != nil {
		return fmt.Errorf("interface %s not found: %w", m.iface.Name, err)
	}
	m.iface = iface

	// Get netlink interface using network manager port
	link, err := m.networkMgr.GetLinkByName(m.iface.Name)
	if err != nil {
		return fmt.Errorf("failed to get netlink interface: %w", err)
	}

	// Check if interface is up
	if m.iface.Flags&net.FlagUp == 0 {
		logger.Warn("Interface is down, bringing it up")
		if err := m.networkMgr.SetLinkUp(link); err != nil {
			return fmt.Errorf("failed to bring interface up: %w", err)
		}
	}

	// Get current IP addresses using network manager port
	addrs, err := m.networkMgr.ListAddresses(link)
	if err != nil {
		return fmt.Errorf("failed to get interface addresses: %w", err)
	}

	// Check if our static IP is configured
	expectedIP := net.ParseIP(config.IPAddress)
	hasStaticIP := false

	for _, addr := range addrs {
		if addr.IPNet.IP.Equal(expectedIP) {
			hasStaticIP = true
			break
		}
	}

	// Reapply configuration if static IP is missing
	if !hasStaticIP {
		logger.WithField("ip", config.IPAddress).
			Warn("Static IP not found on interface, reapplying configuration")
		if err := m.applyStaticConfig(ctx, config); err != nil {
			return fmt.Errorf("failed to reapply static configuration: %w", err)
		}
		logger.Info("Static configuration reapplied successfully")
	}

	return nil
}
