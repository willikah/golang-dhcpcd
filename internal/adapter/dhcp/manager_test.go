//go:build unit

package dhcp

import (
	"context"
	"net"
	"testing"
	"time"

	"golang-dhcpcd/internal/mock"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
	"go.uber.org/mock/gomock"
)

func TestNewManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dhcpClient := mock.NewMockDHCPClient(ctrl)
	networkMgr := mock.NewMockNetworkManager(ctrl)
	fileMgr := mock.NewMockFileManager(ctrl)

	t.Run("ValidInterface", func(t *testing.T) {
		manager, err := NewManager("lo", dhcpClient, networkMgr, fileMgr)
		require.NoError(t, err)
		assert.Equal(t, "lo", manager.GetInterfaceName())
	})

	t.Run("InvalidInterface", func(t *testing.T) {
		_, err := NewManager("nonexistent", dhcpClient, networkMgr, fileMgr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "interface not found")
	})
}

func TestManager_getDHCPLease(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dhcpClient := mock.NewMockDHCPClient(ctrl)
	networkMgr := mock.NewMockNetworkManager(ctrl)
	fileMgr := mock.NewMockFileManager(ctrl)

	manager, err := NewManager("lo", dhcpClient, networkMgr, fileMgr)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("SuccessfulLease", func(t *testing.T) {
		expectedACK := &dhcpv4.DHCPv4{}
		expectedACK.YourIPAddr = net.ParseIP("192.168.1.100")

		dhcpClient.EXPECT().
			RequestLease(ctx, "lo", 15*time.Second).
			Return(expectedACK, nil).
			Times(1)

		ack, err := manager.getDHCPLease(ctx, manager.getLogger())
		require.NoError(t, err)
		assert.Equal(t, expectedACK, ack)
	})

	t.Run("FailedLeaseWithRetries", func(t *testing.T) {
		dhcpClient.EXPECT().
			RequestLease(ctx, "lo", 15*time.Second).
			Return(nil, assert.AnError).
			Times(3)

		ack, err := manager.getDHCPLease(ctx, manager.getLogger())
		assert.Error(t, err)
		assert.Nil(t, ack)
		assert.Contains(t, err.Error(), "DHCP lease request failed after 3 attempts")
	})
}

func TestManager_applyDHCPLease(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dhcpClient := mock.NewMockDHCPClient(ctrl)
	networkMgr := mock.NewMockNetworkManager(ctrl)
	fileMgr := mock.NewMockFileManager(ctrl)

	manager, err := NewManager("lo", dhcpClient, networkMgr, fileMgr)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("SuccessfulIPConfiguration", func(t *testing.T) {
		// Create mock DHCP ACK
		ack := &dhcpv4.DHCPv4{}
		ack.YourIPAddr = net.ParseIP("192.168.1.100")
		// Initialize the Options map before using it
		ack.Options = make(dhcpv4.Options)
		ack.Options.Update(dhcpv4.OptSubnetMask(net.IPv4Mask(255, 255, 255, 0)))

		// Mock netlink operations
		mockLink := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1, Name: "lo"}}

		networkMgr.EXPECT().
			GetLinkByName("lo").
			Return(mockLink, nil)

		networkMgr.EXPECT().
			ListAddresses(mockLink).
			Return([]netlink.Addr{}, nil)

		networkMgr.EXPECT().
			AddAddress(mockLink, gomock.Any()).
			Return(nil)

		err := manager.applyDHCPLease(ctx, ack)
		assert.NoError(t, err)
	})

	t.Run("IPAlreadyConfigured", func(t *testing.T) {
		// Create mock DHCP ACK
		ack := &dhcpv4.DHCPv4{}
		ack.YourIPAddr = net.ParseIP("192.168.1.100")
		// Initialize the Options map before using it
		ack.Options = make(dhcpv4.Options)
		ack.Options.Update(dhcpv4.OptSubnetMask(net.IPv4Mask(255, 255, 255, 0)))

		// Mock netlink operations
		mockLink := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1, Name: "lo"}}

		// Existing address that matches
		existingAddr := netlink.Addr{
			IPNet: &net.IPNet{
				IP:   net.ParseIP("192.168.1.100"),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			},
		}

		networkMgr.EXPECT().
			GetLinkByName("lo").
			Return(mockLink, nil)

		networkMgr.EXPECT().
			ListAddresses(mockLink).
			Return([]netlink.Addr{existingAddr}, nil)

		// Should not call AddAddress since IP is already configured

		err := manager.applyDHCPLease(ctx, ack)
		assert.NoError(t, err)
	})
}

func TestManager_configureDefaultRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dhcpClient := mock.NewMockDHCPClient(ctrl)
	networkMgr := mock.NewMockNetworkManager(ctrl)
	fileMgr := mock.NewMockFileManager(ctrl)

	manager, err := NewManager("lo", dhcpClient, networkMgr, fileMgr)
	require.NoError(t, err)

	ctx := context.Background()
	mockLink := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1, Name: "lo"}}
	gateway := net.ParseIP("192.168.1.1")

	t.Run("AddNewDefaultRoute", func(t *testing.T) {
		networkMgr.EXPECT().
			ListRoutes().
			Return([]netlink.Route{}, nil)

		networkMgr.EXPECT().
			AddRoute(gomock.Any()).
			Return(nil)

		err := manager.configureDefaultRoute(ctx, mockLink, gateway)
		assert.NoError(t, err)
	})

	t.Run("RouteAlreadyExists", func(t *testing.T) {
		existingRoute := netlink.Route{
			LinkIndex: 1,
			Gw:        gateway,
			Dst:       nil, // Default route
		}

		networkMgr.EXPECT().
			ListRoutes().
			Return([]netlink.Route{existingRoute}, nil)

		// Should not call AddRoute since route already exists

		err := manager.configureDefaultRoute(ctx, mockLink, gateway)
		assert.NoError(t, err)
	})
}

func TestManager_configureDNS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dhcpClient := mock.NewMockDHCPClient(ctrl)
	networkMgr := mock.NewMockNetworkManager(ctrl)
	fileMgr := mock.NewMockFileManager(ctrl)

	manager, err := NewManager("lo", dhcpClient, networkMgr, fileMgr)
	require.NoError(t, err)

	ctx := context.Background()
	dnsServers := []net.IP{
		net.ParseIP("8.8.8.8"),
		net.ParseIP("8.8.4.4"),
	}

	t.Run("WriteDNSConfiguration", func(t *testing.T) {
		expectedContent := "# Generated by golang-dhcpcd\nnameserver 8.8.8.8\nnameserver 8.8.4.4\n"

		fileMgr.EXPECT().
			ReadFile("/etc/resolv.conf").
			Return([]byte("old content"), nil)

		fileMgr.EXPECT().
			WriteFile("/etc/resolv.conf", []byte(expectedContent), 0644).
			Return(nil)

		err := manager.configureDNS(ctx, dnsServers)
		assert.NoError(t, err)
	})

	t.Run("DNSAlreadyUpToDate", func(t *testing.T) {
		expectedContent := "# Generated by golang-dhcpcd\nnameserver 8.8.8.8\nnameserver 8.8.4.4\n"

		fileMgr.EXPECT().
			ReadFile("/etc/resolv.conf").
			Return([]byte(expectedContent), nil)

		// Should not call WriteFile since content is already correct

		err := manager.configureDNS(ctx, dnsServers)
		assert.NoError(t, err)
	})
}

// Helper method for the manager to get logger (for testing)
func (m *Manager) getLogger() *logrus.Entry {
	return logrus.NewEntry(logrus.New())
}
