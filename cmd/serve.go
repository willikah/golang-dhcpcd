package cmd

import (
	"context"
	"fmt"
	"golang-dhcpcd/internal/adapter/dhcp"
	infraDhcp "golang-dhcpcd/internal/adapter/infrastructure/dhcp"
	"golang-dhcpcd/internal/adapter/infrastructure/file"
	"golang-dhcpcd/internal/adapter/infrastructure/network"
	"golang-dhcpcd/internal/adapter/static"
	"golang-dhcpcd/internal/pkg/config"
	"golang-dhcpcd/internal/pkg/logging"
	"golang-dhcpcd/internal/port"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	configFlag string
)

// createNetworkConfigurationManager creates a network configuration manager for the given interface and configuration
func createNetworkConfigurationManager(ifaceName string, ifaceConfig config.InterfaceConfig) (port.NetworkConfigurationManager, error) {
	logger := logging.GetLogger()

	// Create shared infrastructure adapters
	networkMgr := network.NewManagerAdapter()
	fileMgr := file.NewManagerAdapter()

	if ifaceConfig.DHCP {
		// Create DHCP infrastructure adapter
		dhcpClient := infraDhcp.NewClientAdapter()

		// Create DHCP network configuration adapter
		manager, err := dhcp.NewManager(ifaceName, dhcpClient, networkMgr, fileMgr)
		if err != nil {
			return nil, err
		}
		logger.WithField("interface", ifaceName).Info("Created DHCP network configuration adapter")
		return manager, nil
	} else if ifaceConfig.Static != nil {
		// Create static network configuration adapter
		manager, err := static.NewManager(ifaceName, ifaceConfig, networkMgr)
		if err != nil {
			return nil, err
		}
		logger.WithField("interface", ifaceName).WithFields(map[string]interface{}{
			"ip":      ifaceConfig.Static.IP,
			"netmask": ifaceConfig.Static.Netmask,
			"gateway": ifaceConfig.Static.Gateway,
		}).Info("Created static network configuration adapter")
		return manager, nil
	}

	return nil, fmt.Errorf("invalid interface configuration: must specify either DHCP or static")
}

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

		// Create context for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigChan
			logger.WithField("signal", sig.String()).Info("Received shutdown signal")
			cancel()
		}()

		// Create network configuration managers for all interfaces
		var managers []port.NetworkConfigurationManager

		for ifaceName, ifaceConfig := range cfg.Interfaces {
			manager, err := createNetworkConfigurationManager(ifaceName, ifaceConfig)
			if err != nil {
				logger.WithField("interface", ifaceName).WithError(err).Error("Failed to create network configuration adapter")
				continue
			}
			managers = append(managers, manager)
		}

		if len(managers) == 0 {
			logger.Warn("No network configuration adapters created")
			return
		}

		logger.WithField("adapter_count", len(managers)).Info("Starting network configuration adapters")

		// Start all network configuration adapters concurrently
		var wg sync.WaitGroup
		for _, manager := range managers {
			wg.Add(1)
			go func(mgr port.NetworkConfigurationManager) {
				defer wg.Done()

				if err := mgr.Run(ctx); err != nil {
					if err != context.Canceled {
						logger.WithField("interface", mgr.GetInterfaceName()).WithError(err).Error("Network configuration adapter failed")
					}
				}
			}(manager)
		}

		// Wait for all adapters to complete
		wg.Wait()
		logger.Info("All network configuration adapters stopped")
	},
}

func init() {
	serveCmd.Flags().StringVarP(&configFlag, "config", "f", "", "Path to config file (YAML)")
	if err := serveCmd.MarkFlagRequired("config"); err != nil {
		panic(err) // This should never happen during initialization
	}
	rootCmd.AddCommand(serveCmd)
}
