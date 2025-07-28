// Package types defines common types used across the application.
package types

// StaticIPConfig represents static IP configuration parameters.
// This type is used by static network configuration adapters.
type StaticIPConfig struct {
	IPAddress string `yaml:"ip"`      // IP address in dotted decimal notation (e.g., "192.168.1.100")
	Netmask   string `yaml:"netmask"` // Subnet mask in dotted decimal notation (e.g., "255.255.255.0")
	Gateway   string `yaml:"gateway"` // Default gateway IP address (optional)
}
