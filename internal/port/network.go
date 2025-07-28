// Package port defines the primary ports (interfaces) for the application.
// This follows the Ports and Adapters (Hexagonal Architecture) pattern.
package port

import (
	"context"
)

// NetworkConfigurationManager is the primary port for network configuration.
// This interface defines the contract that all network configuration adapters must implement.
// It follows the Ports and Adapters (Hexagonal Architecture) pattern where this is the "port"
// and specific implementations (DHCP, Static) are the "adapters".
type NetworkConfigurationManager interface {
	// Run starts the network configuration and runs until the context is cancelled.
	// It returns an error if the configuration fails or if the context is cancelled.
	Run(ctx context.Context) error

	// GetInterfaceName returns the name of the network interface managed by this manager.
	GetInterfaceName() string
}
