package cmd

import (
	"fmt"
	"golang-dhcpcd/internal/pkg/config"
	"golang-dhcpcd/internal/pkg/dhcpc"
	"golang-dhcpcd/internal/pkg/logging"
	"golang-dhcpcd/internal/pkg/static"
	"sync"

	"github.com/spf13/cobra"
)

var (
	configFlag string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run as DHCP client or configure static IP for all interfaces",
	Run: func(cmd *cobra.Command, args []string) {
		// Load and validate configuration
		cfg, err := config.Load(configFlag)
		if err != nil {
			fmt.Printf("Config error: %v\n", err)
			return
		}

		if err := cfg.Validate(); err != nil {
			fmt.Printf("Config validation error: %v\n", err)
			return
		}

		// Initialize logging
		logging.InitLogger(cfg.Logging)

		logger := logging.GetLogger()
		logger.WithField("config_file", configFlag).Info("Starting daemon")

		// Start interface configuration in goroutines
		var wg sync.WaitGroup
		for ifaceName, ifaceConfig := range cfg.Interfaces {
			wg.Add(1)
			go func(name string, config config.InterfaceConfig) {
				defer wg.Done()

				ifaceLogger := logging.WithInterface(name)

				if config.DHCP {
					ifaceLogger.WithField("component", "dhcp").Info("Starting DHCP client")
					if err := runDHCP(name); err != nil {
						ifaceLogger.WithField("component", "dhcp").WithError(err).Error("DHCP client failed")
					}
				} else if config.Static != nil {
					ifaceLogger.WithField("component", "static").
						WithField("ip", config.Static.IP).
						WithField("netmask", config.Static.Netmask).
						WithField("gateway", config.Static.Gateway).
						Info("Configuring static IP")
					if err := runStaticConfig(name, config.Static); err != nil {
						ifaceLogger.WithField("component", "static").WithError(err).Error("Static configuration failed")
					}
				}
			}(ifaceName, ifaceConfig)
		}

		// Wait for all goroutines to complete
		wg.Wait()
	},
}

func init() {
	serveCmd.Flags().StringVarP(&configFlag, "config", "f", "", "Path to config file (YAML)")
	if err := serveCmd.MarkFlagRequired("config"); err != nil {
		panic(err) // This should never happen during initialization
	}
	rootCmd.AddCommand(serveCmd)
}

// runDHCP runs the real DHCP client on the specified interface
func runDHCP(ifaceName string) error {
	client, err := dhcpc.NewClient(ifaceName)
	if err != nil {
		return err
	}
	return client.Run()
}

// runStaticConfig configures static IP on the specified interface
func runStaticConfig(ifaceName string, staticConfig *config.StaticConfig) error {
	logger := logging.WithComponentAndInterface("static", ifaceName)

	// Create static client
	client, err := static.NewClient(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to create static client: %w", err)
	} // Convert config.StaticConfig to static.Config
	staticClientConfig := static.Config{
		IPAddress: staticConfig.IP,
		Netmask:   staticConfig.Netmask,
		Gateway:   staticConfig.Gateway,
	}

	logger.WithField("config", staticClientConfig).Debug("Created static client configuration")

	// Run static configuration
	return client.Run(staticClientConfig)
}
