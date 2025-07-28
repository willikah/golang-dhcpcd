//go:build unit

package config

import (
	"os"
	"path/filepath"
	"testing"

	"golang-dhcpcd/internal/pkg/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("ValidConfig", func(t *testing.T) {
		configContent := `logging:
  level: info
  format: simple

interfaces:
  eth0:
    dhcp: true
  eth1:
    dhcp: false
    static:
      ip: 192.168.1.100
      netmask: 255.255.255.0
      gateway: 192.168.1.1
`
		configFile := filepath.Join(tempDir, "valid.yml")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := Load(configFile)
		require.NoError(t, err)
		assert.Equal(t, "info", config.Logging.Level)
		assert.Equal(t, "simple", config.Logging.Format)
		assert.Len(t, config.Interfaces, 2)
		
		// Test DHCP interface
		eth0, exists := config.Interfaces["eth0"]
		assert.True(t, exists)
		assert.True(t, eth0.DHCP)
		assert.Nil(t, eth0.Static)

		// Test static interface
		eth1, exists := config.Interfaces["eth1"]
		assert.True(t, exists)
		assert.False(t, eth1.DHCP)
		require.NotNil(t, eth1.Static)
		assert.Equal(t, "192.168.1.100", eth1.Static.IP)
		assert.Equal(t, "255.255.255.0", eth1.Static.Netmask)
		assert.Equal(t, "192.168.1.1", eth1.Static.Gateway)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := Load("/nonexistent/config.yml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		configContent := `invalid: yaml: content: [
`
		configFile := filepath.Join(tempDir, "invalid.yml")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = Load(configFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
}

func TestConfig_GetInterfaceConfig(t *testing.T) {
	config := &Config{
		Logging: logging.LogConfig{
			Level:  "info",
			Format: "simple",
		},
		Interfaces: map[string]InterfaceConfig{
			"eth0": {
				DHCP: true,
			},
			"eth1": {
				DHCP: false,
				Static: &StaticConfig{
					IP:      "192.168.1.100",
					Netmask: "255.255.255.0",
					Gateway: "192.168.1.1",
				},
			},
		},
	}

	t.Run("ExistingInterface", func(t *testing.T) {
		ifaceConfig, exists := config.GetInterfaceConfig("eth0")
		assert.True(t, exists)
		assert.True(t, ifaceConfig.DHCP)
	})

	t.Run("NonExistentInterface", func(t *testing.T) {
		_, exists := config.GetInterfaceConfig("eth99")
		assert.False(t, exists)
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := &Config{
			Logging: logging.LogConfig{
				Level:  "info",
				Format: "simple",
			},
			Interfaces: map[string]InterfaceConfig{
				"eth0": {
					DHCP: true,
				},
				"eth1": {
					DHCP: false,
					Static: &StaticConfig{
						IP:      "192.168.1.100",
						Netmask: "255.255.255.0",
						Gateway: "192.168.1.1",
					},
				},
			},
		}

		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("NoInterfaces", func(t *testing.T) {
		config := &Config{
			Logging:    logging.LogConfig{},
			Interfaces: map[string]InterfaceConfig{},
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no interfaces configured")
	})

	t.Run("InterfaceWithoutDHCPOrStatic", func(t *testing.T) {
		config := &Config{
			Logging: logging.LogConfig{},
			Interfaces: map[string]InterfaceConfig{
				"eth0": {
					DHCP:   false,
					Static: nil,
				},
			},
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must specify either dhcp or static configuration")
	})

	t.Run("InterfaceWithBothDHCPAndStatic", func(t *testing.T) {
		config := &Config{
			Logging: logging.LogConfig{},
			Interfaces: map[string]InterfaceConfig{
				"eth0": {
					DHCP: true,
					Static: &StaticConfig{
						IP:      "192.168.1.100",
						Netmask: "255.255.255.0",
					},
				},
			},
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify both dhcp and static configuration")
	})

	t.Run("StaticConfigMissingIP", func(t *testing.T) {
		config := &Config{
			Logging: logging.LogConfig{},
			Interfaces: map[string]InterfaceConfig{
				"eth0": {
					DHCP: false,
					Static: &StaticConfig{
						IP:      "",
						Netmask: "255.255.255.0",
					},
				},
			},
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "static IP address is required")
	})

	t.Run("StaticConfigMissingNetmask", func(t *testing.T) {
		config := &Config{
			Logging: logging.LogConfig{},
			Interfaces: map[string]InterfaceConfig{
				"eth0": {
					DHCP: false,
					Static: &StaticConfig{
						IP:      "192.168.1.100",
						Netmask: "",
					},
				},
			},
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "static netmask is required")
	})

	t.Run("StaticConfigWithOptionalGateway", func(t *testing.T) {
		config := &Config{
			Logging: logging.LogConfig{},
			Interfaces: map[string]InterfaceConfig{
				"eth0": {
					DHCP: false,
					Static: &StaticConfig{
						IP:      "192.168.1.100",
						Netmask: "255.255.255.0",
						// Gateway is optional
					},
				},
			},
		}

		err := config.Validate()
		assert.NoError(t, err)
	})
}
