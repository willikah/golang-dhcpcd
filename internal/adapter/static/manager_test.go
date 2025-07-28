//go:build unit

package static

import (
	"context"
	"net"
	"testing"

	"golang-dhcpcd/internal/mock"
	"golang-dhcpcd/internal/pkg/config"
	"golang-dhcpcd/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
	"go.uber.org/mock/gomock"
)

func TestNewManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	networkMgr := mock.NewMockNetworkManager(ctrl)

	t.Run("ValidStaticConfig", func(t *testing.T) {
		ifaceConfig := config.InterfaceConfig{
			DHCP: false,
			Static: &config.StaticConfig{
				IP:      "192.168.1.100",
				Netmask: "255.255.255.0",
				Gateway: "192.168.1.1",
			},
		}

		manager, err := NewManager("lo", ifaceConfig, networkMgr)
		require.NoError(t, err)
		assert.Equal(t, "lo", manager.GetInterfaceName())
		assert.Equal(t, "192.168.1.100", manager.staticConfig.IPAddress)
		assert.Equal(t, "255.255.255.0", manager.staticConfig.Netmask)
		assert.Equal(t, "192.168.1.1", manager.staticConfig.Gateway)
	})

	t.Run("InvalidInterface", func(t *testing.T) {
		ifaceConfig := config.InterfaceConfig{
			DHCP: false,
			Static: &config.StaticConfig{
				IP:      "192.168.1.100",
				Netmask: "255.255.255.0",
				Gateway: "192.168.1.1",
			},
		}

		_, err := NewManager("nonexistent", ifaceConfig, networkMgr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "interface not found")
	})

	t.Run("MissingStaticConfig", func(t *testing.T) {
		ifaceConfig := config.InterfaceConfig{
			DHCP:   false,
			Static: nil,
		}

		_, err := NewManager("lo", ifaceConfig, networkMgr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "interface configuration does not have static IP settings")
	})
}

func TestManager_applyStaticConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	networkMgr := mock.NewMockNetworkManager(ctrl)

	ifaceConfig := config.InterfaceConfig{
		DHCP: false,
		Static: &config.StaticConfig{
			IP:      "192.168.1.100",
			Netmask: "255.255.255.0",
			Gateway: "192.168.1.1",
		},
	}

	manager, err := NewManager("lo", ifaceConfig, networkMgr)
	require.NoError(t, err)

	ctx := context.Background()
	staticConfig := types.StaticIPConfig{
		IPAddress: "192.168.1.100",
		Netmask:   "255.255.255.0",
		Gateway:   "192.168.1.1",
	}

	t.Run("SuccessfulConfiguration", func(t *testing.T) {
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

		// Expect gateway configuration
		networkMgr.EXPECT().
			ListRoutes().
			Return([]netlink.Route{}, nil)

		networkMgr.EXPECT().
			AddRoute(gomock.Any()).
			Return(nil)

		err := manager.applyStaticConfig(ctx, staticConfig)
		assert.NoError(t, err)
	})

	t.Run("IPAlreadyConfigured", func(t *testing.T) {
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

		// Should still configure gateway even if IP exists
		networkMgr.EXPECT().
			ListRoutes().
			Return([]netlink.Route{}, nil)

		networkMgr.EXPECT().
			AddRoute(gomock.Any()).
			Return(nil)

		err := manager.applyStaticConfig(ctx, staticConfig)
		assert.NoError(t, err)
	})

	t.Run("InvalidIPAddress", func(t *testing.T) {
		invalidConfig := types.StaticIPConfig{
			IPAddress: "invalid-ip",
			Netmask:   "255.255.255.0",
			Gateway:   "192.168.1.1",
		}

		mockLink := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1, Name: "lo"}}

		networkMgr.EXPECT().
			GetLinkByName("lo").
			Return(mockLink, nil)

		err := manager.applyStaticConfig(ctx, invalidConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid IP address")
	})

	t.Run("InvalidNetmask", func(t *testing.T) {
		invalidConfig := types.StaticIPConfig{
			IPAddress: "192.168.1.100",
			Netmask:   "invalid-netmask",
			Gateway:   "192.168.1.1",
		}

		mockLink := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1, Name: "lo"}}

		networkMgr.EXPECT().
			GetLinkByName("lo").
			Return(mockLink, nil)

		err := manager.applyStaticConfig(ctx, invalidConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid netmask")
	})
}

func TestManager_configureDefaultRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	networkMgr := mock.NewMockNetworkManager(ctrl)

	ifaceConfig := config.InterfaceConfig{
		DHCP: false,
		Static: &config.StaticConfig{
			IP:      "192.168.1.100",
			Netmask: "255.255.255.0",
			Gateway: "192.168.1.1",
		},
	}

	manager, err := NewManager("lo", ifaceConfig, networkMgr)
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

	t.Run("RemoveConflictingRoute", func(t *testing.T) {
		conflictingRoute := netlink.Route{
			LinkIndex: 2, // Different interface
			Gw:        net.ParseIP("192.168.1.2"), // Different gateway
			Dst:       nil, // Default route
		}

		networkMgr.EXPECT().
			ListRoutes().
			Return([]netlink.Route{conflictingRoute}, nil)

		networkMgr.EXPECT().
			DeleteRoute(&conflictingRoute).
			Return(nil)

		networkMgr.EXPECT().
			AddRoute(gomock.Any()).
			Return(nil)

		err := manager.configureDefaultRoute(ctx, mockLink, gateway)
		assert.NoError(t, err)
	})
}
