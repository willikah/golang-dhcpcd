package static

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang-dhcpcd/internal/pkg/logging"

	"github.com/vishvananda/netlink"
)

// Client handles static IP configuration for a network interface.
type Client struct {
	Iface *net.Interface
}

// Config represents static IP configuration parameters.
type Config struct {
	IPAddress string `yaml:"ip"`
	Netmask   string `yaml:"netmask"`
	Gateway   string `yaml:"gateway"`
}

// NewClient creates a new static IP client for the given interface name.
func NewClient(ifaceName string) (*Client, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("interface not found: %w", err)
	}
	return &Client{Iface: iface}, nil
}

// Run configures the interface with static IP settings and maintains the configuration.
func (c *Client) Run(config Config) error {
	logger := logging.WithComponentAndInterface("static", c.Iface.Name).WithField("mac", c.Iface.HardwareAddr.String())
	logger.Info("Starting static IP configuration")

	// Validate configuration
	if err := c.validateConfig(config); err != nil {
		return fmt.Errorf("invalid static configuration: %w", err)
	}

	// Apply static IP configuration
	if err := c.applyStaticConfig(config); err != nil {
		return fmt.Errorf("failed to apply static configuration: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"ip":      config.IPAddress,
		"netmask": config.Netmask,
		"gateway": config.Gateway,
	}).Info("Static IP configuration applied successfully")

	// Monitor interface status and reapply configuration if needed
	return c.monitorInterface(config)
}

// validateConfig validates the static IP configuration parameters.
func (c *Client) validateConfig(config Config) error {
	// Validate IP address
	ip := net.ParseIP(config.IPAddress)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", config.IPAddress)
	}
	if ip.To4() == nil {
		return fmt.Errorf("only IPv4 addresses are supported: %s", config.IPAddress)
	}

	// Validate netmask
	mask := net.ParseIP(config.Netmask)
	if mask == nil {
		return fmt.Errorf("invalid netmask: %s", config.Netmask)
	}
	if mask.To4() == nil {
		return fmt.Errorf("only IPv4 netmasks are supported: %s", config.Netmask)
	}

	// Validate gateway
	if config.Gateway != "" {
		gw := net.ParseIP(config.Gateway)
		if gw == nil {
			return fmt.Errorf("invalid gateway address: %s", config.Gateway)
		}
		if gw.To4() == nil {
			return fmt.Errorf("only IPv4 gateway addresses are supported: %s", config.Gateway)
		}
	}

	return nil
}

// applyStaticConfig applies the static IP configuration to the interface using netlink.
func (c *Client) applyStaticConfig(config Config) error {
	logger := logging.WithComponentAndInterface("static", c.Iface.Name)

	// Get netlink interface
	link, err := netlink.LinkByName(c.Iface.Name)
	if err != nil {
		return fmt.Errorf("failed to get netlink interface: %w", err)
	} // Parse IP address and netmask
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
	existingAddrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
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
				if err := netlink.AddrDel(link, &addr); err != nil {
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
		if err := netlink.AddrAdd(link, addr); err != nil {
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

		if err := c.configureDefaultRoute(link, gateway); err != nil {
			return fmt.Errorf("failed to set default gateway: %w", err)
		}
	}

	return nil
}

// configureDefaultRoute configures the default gateway for the interface.
func (c *Client) configureDefaultRoute(link netlink.Link, gateway net.IP) error {
	logger := logging.WithComponentAndInterface("static", c.Iface.Name).WithField("gateway", gateway.String())

	// Check if default route already exists with this gateway
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
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
				if err := netlink.RouteDel(&route); err != nil {
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

		if err := netlink.RouteAdd(route); err != nil {
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
func (c *Client) monitorInterface(config Config) error {
	logger := logging.WithComponentAndInterface("static", c.Iface.Name)
	logger.Info("Starting interface monitoring")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := c.checkAndRepairConfiguration(config); err != nil {
			logger.WithError(err).Error("Configuration check failed")
		}
	}

	// This line is unreachable, but required for compilation
	return nil
}

// checkAndRepairConfiguration checks if the static configuration is still applied and repairs if needed.
func (c *Client) checkAndRepairConfiguration(config Config) error {
	logger := logging.WithComponentAndInterface("static", c.Iface.Name)

	// Refresh interface information
	iface, err := net.InterfaceByName(c.Iface.Name)
	if err != nil {
		return fmt.Errorf("interface %s not found: %w", c.Iface.Name, err)
	}
	c.Iface = iface

	// Get netlink interface
	link, err := netlink.LinkByName(c.Iface.Name)
	if err != nil {
		return fmt.Errorf("failed to get netlink interface: %w", err)
	}

	// Check if interface is up
	if c.Iface.Flags&net.FlagUp == 0 {
		logger.Warn("Interface is down, bringing it up")
		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to bring interface up: %w", err)
		}
	}

	// Get current IP addresses using netlink
	addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
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
		if err := c.applyStaticConfig(config); err != nil {
			return fmt.Errorf("failed to reapply static configuration: %w", err)
		}
		logger.Info("Static configuration reapplied successfully")
	}

	return nil
}
