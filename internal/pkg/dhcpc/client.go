package dhcpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang-dhcpcd/internal/pkg/logging"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
	"github.com/vishvananda/netlink"
)

// Client wraps the dhcpv4 client for a network interface.
type Client struct {
	Iface *net.Interface
}

// NewClient creates a new DHCP client for the given interface name.
func NewClient(ifaceName string) (*Client, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("interface not found: %w", err)
	}
	return &Client{Iface: iface}, nil
}

// Run starts and maintains DHCP lease on the interface using the nclient4 library.
func (c *Client) Run() error {
	logger := logging.WithComponentAndInterface("dhcp", c.Iface.Name).WithField("mac", c.Iface.HardwareAddr.String())
	logger.Info("Starting DHCP client")

	const maxRetries = 3
	const retryDelay = 2 * time.Second

	for {
		var lease *dhcpv4.DHCPv4

		// Retry DISCOVER/OFFER up to maxRetries times
		for attempt := 1; attempt <= maxRetries; attempt++ {
			logger.WithField("attempt", fmt.Sprintf("%d/%d", attempt, maxRetries)).Debug("Attempting to get DHCP lease")

			// Create DHCP client using the nclient4 library
			client, err := nclient4.New(c.Iface.Name, nclient4.WithTimeout(15*time.Second))
			if err != nil {
				logger.WithError(err).Error("Failed to create DHCP client")
				if attempt < maxRetries {
					logger.WithField("delay", retryDelay).Debug("Retrying...")
					time.Sleep(retryDelay)
					continue
				}
				return fmt.Errorf("failed to create DHCP client after %d attempts: %w", maxRetries, err)
			}
			defer client.Close()

			logger.Debug("Created DHCP client")

			// Perform DHCP DISCOVER/OFFER exchange
			offer, err := client.DiscoverOffer(context.Background())
			if err != nil {
				logger.WithError(err).WithField("attempt", attempt).Error("DISCOVER/OFFER failed")
				client.Close()
				if attempt < maxRetries {
					logger.WithField("delay", retryDelay).Debug("Retrying...")
					time.Sleep(retryDelay)
					continue
				}
			} else {
				lease = offer
				logger.WithFields(map[string]interface{}{
					"attempt": attempt,
					"ip":      offer.YourIPAddr.String(),
				}).Info("Successfully received OFFER")
				client.Close()
				break
			}
		}

		// If no valid offer received after all retries, wait and restart
		if lease == nil {
			logger.WithField("attempts", maxRetries).Warn("All attempts failed, waiting before full retry")
			time.Sleep(30 * time.Second)
			continue
		}

		// Perform REQUEST/ACK exchange with retry mechanism
		var ack *dhcpv4.DHCPv4
		for attempt := 1; attempt <= maxRetries; attempt++ {
			// Create a new client for REQUEST/ACK
			client, err := nclient4.New(c.Iface.Name, nclient4.WithTimeout(10*time.Second))
			if err != nil {
				logger.WithError(err).Error("Failed to create DHCP client for REQUEST")
				break
			}

			// Send REQUEST and wait for ACK
			leasedPacket, err := client.RequestFromOffer(context.Background(), lease)
			client.Close()

			if err != nil {
				logger.WithError(err).WithField("attempt", attempt).Error("REQUEST/ACK failed")
				if attempt < maxRetries {
					logger.WithField("delay", retryDelay).Debug("Retrying REQUEST...")
					time.Sleep(retryDelay)
					continue
				}
				break
			}

			// Successfully received ACK - extract the DHCP packet from the lease
			ack = leasedPacket.ACK
			logger.WithField("ip", ack.YourIPAddr.String()).Info("Received ACK")
			break
		}

		// If no valid ACK received after all retries, restart the whole process
		if ack == nil {
			logger.Error("Failed to receive ACK after all attempts, restarting DHCP process")
			continue
		}

		leaseTime := ack.IPAddressLeaseTime(60 * time.Second)
		logger.WithFields(map[string]interface{}{
			"ip":         ack.YourIPAddr.String(),
			"lease_time": leaseTime.String(),
		}).Info("Lease acquired")

		// Apply the DHCP lease to the network interface
		if err := c.applyDHCPLease(ack); err != nil {
			logger.WithError(err).Error("Failed to apply lease to interface")
			logger.Warn("Continuing without interface configuration")
		} else {
			logger.Info("Successfully configured interface")
		}

		// Sleep until lease renewal time
		renewal := ack.IPAddressRenewalTime(30 * time.Second)
		logger.WithField("renewal_time", renewal.String()).Info("Sleeping for renewal time")
		time.Sleep(renewal)
	}
}

// applyDHCPLease configures the network interface with the received DHCP lease using netlink
func (c *Client) applyDHCPLease(ack *dhcpv4.DHCPv4) error {
	logger := logging.WithComponentAndInterface("dhcp", c.Iface.Name)

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

	// Get netlink interface
	link, err := netlink.LinkByName(c.Iface.Name)
	if err != nil {
		return fmt.Errorf("failed to get netlink interface: %w", err)
	}

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
		if err := netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("failed to add IP address %s: %w", ipNet.String(), err)
		}
		logger.WithField("ip", ipNet.String()).Info("Successfully added IP address")
	}

	// Configure default gateway if provided
	routers := ack.Router()
	if len(routers) > 0 {
		gateway := routers[0]
		logger.WithField("gateway", gateway.String()).Info("Setting default gateway")

		if err := c.configureDefaultRoute(link, gateway); err != nil {
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
		if err := c.configureDNS(dnsServers); err != nil {
			logger.WithError(err).Warn("Failed to configure DNS")
		}
	}

	return nil
}

// configureDefaultRoute configures the default route using netlink
func (c *Client) configureDefaultRoute(link netlink.Link, gateway net.IP) error {
	logger := logging.WithComponentAndInterface("dhcp", c.Iface.Name).WithField("gateway", gateway.String())

	// List existing routes to check if our desired route already exists
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
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

				if err := netlink.RouteDel(&route); err != nil {
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

		if err := netlink.RouteAdd(route); err != nil {
			return fmt.Errorf("failed to add default route: %w", err)
		}

		logger.Info("Successfully added default route")
	}

	return nil
}

// configureDNS writes DNS servers to /etc/resolv.conf
func (c *Client) configureDNS(dnsServers []net.IP) error {
	logger := logging.WithComponentAndInterface("dhcp", c.Iface.Name)

	// Generate the new DNS configuration content
	newContent := "# Generated by golang-dhcpcd\n"
	for _, dns := range dnsServers {
		newContent += fmt.Sprintf("nameserver %s\n", dns.String())
	}

	// Check if the current /etc/resolv.conf already has the same content
	if currentContent, err := os.ReadFile("/etc/resolv.conf"); err == nil {
		if string(currentContent) == newContent {
			logger.Debug("DNS configuration already up to date, skipping")
			return nil
		}
	}

	// Write the new DNS configuration
	if err := os.WriteFile("/etc/resolv.conf", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/resolv.conf: %w", err)
	}

	logger.Info("Updated /etc/resolv.conf with DNS servers")
	return nil
}
