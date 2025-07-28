package config

import (
	"fmt"
	"os"

	"golang-dhcpcd/internal/pkg/logging"

	"gopkg.in/yaml.v3"
)

// InterfaceConfig represents the configuration for a network interface
type InterfaceConfig struct {
	DHCP   bool          `yaml:"dhcp,omitempty"`
	Static *StaticConfig `yaml:"static,omitempty"`
}

// StaticConfig represents static IP configuration
type StaticConfig struct {
	IP      string `yaml:"ip"`
	Netmask string `yaml:"netmask"`
	Gateway string `yaml:"gateway"`
}

// Config represents the main configuration structure
type Config struct {
	Logging    logging.LogConfig          `yaml:"logging"`
	Interfaces map[string]InterfaceConfig `yaml:"interfaces"`
}

// Load loads configuration from a YAML file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return &config, nil
}

// GetInterfaceConfig returns the configuration for a specific interface
func (c *Config) GetInterfaceConfig(interfaceName string) (InterfaceConfig, bool) {
	config, exists := c.Interfaces[interfaceName]
	return config, exists
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Interfaces) == 0 {
		return fmt.Errorf("no interfaces configured")
	}

	for name, iface := range c.Interfaces {
		if !iface.DHCP && iface.Static == nil {
			return fmt.Errorf("interface %s: must specify either dhcp or static configuration", name)
		}
		if iface.DHCP && iface.Static != nil {
			return fmt.Errorf("interface %s: cannot specify both dhcp and static configuration", name)
		}
		if iface.Static != nil {
			if err := validateStaticConfig(name, iface.Static); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateStaticConfig(interfaceName string, static *StaticConfig) error {
	if static.IP == "" {
		return fmt.Errorf("interface %s: static IP address is required", interfaceName)
	}
	if static.Netmask == "" {
		return fmt.Errorf("interface %s: static netmask is required", interfaceName)
	}
	return nil
}
