//go:build integration
// +build integration

package test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	// Expected IP ranges
	dhcpSubnet   = "192.168.100.0/24"
	staticSubnet = "192.168.101.0/24"

	// Expected static IPs
	client1StaticIP = "192.168.101.10"
	client2StaticIP = "192.168.101.20"

	// DHCP range (from dhcpd.conf)
	dhcpRangeStart = "192.168.100.10"
	dhcpRangeEnd   = "192.168.100.100"
)

// TestDHCPAndStaticIntegration tests the complete DHCP and static IP functionality
// using Docker Compose to run the real services
func TestDHCPAndStaticIntegration(t *testing.T) {
	// Get the test directory (where docker-compose.yml is located)
	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Ensure we're in the test directory
	if !strings.HasSuffix(testDir, "test") {
		testDir = filepath.Join(testDir, "test")
	}

	ctx := context.Background()

	// Build and start the docker-compose stack using make targets
	t.Log("Building Docker images...")
	if err := runMakeTarget("docker-build"); err != nil {
		t.Fatalf("Failed to build Docker images: %v", err)
	}

	t.Log("Starting Docker Compose stack...")
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to start docker-compose: %v", err)
	}

	// Clean up function
	t.Cleanup(func() {
		t.Log("Stopping Docker Compose stack...")
		cmd := exec.Command("docker", "compose", "down", "--remove-orphans")
		cmd.Dir = testDir
		if err := cmd.Run(); err != nil {
			t.Logf("Failed to stop docker-compose: %v", err)
		}
	})

	// Wait for services to stabilize
	time.Sleep(15 * time.Second)

	// Test DHCP functionality
	t.Run("DHCP_Client_Gets_IP", func(t *testing.T) {
		testDHCPAssignment(t, ctx, testDir)
	})

	// Test Static IP functionality
	t.Run("Static_IP_Configuration", func(t *testing.T) {
		testStaticIPConfiguration(t, ctx, testDir)
	})

}

// runMakeTarget runs a make target from the project root
func runMakeTarget(target string) error {
	cmd := exec.Command("make", target)
	cmd.Dir = filepath.Join("..") // Go up one directory to project root
	return cmd.Run()
}

// testDHCPAssignment verifies that DHCP clients receive IP addresses in the expected range
func testDHCPAssignment(t *testing.T, ctx context.Context, testDir string) {
	// Wait a bit longer for DHCP assignment to complete
	time.Sleep(10 * time.Second)

	// Check container logs to see if DHCP is working
	logCmd := exec.Command("docker", "compose", "logs", "client-1")
	logCmd.Dir = testDir
	logOutput, err := logCmd.Output()
	if err != nil {
		t.Logf("Failed to get client-1 logs: %v", err)
	} else {
		t.Logf("Client-1 logs:\n%s", string(logOutput))
	}

	// Check client-1 DHCP assignment (actual interface IP)
	client1IP, err := getContainerActualIP(testDir, "test-client-1-1", "eth0")
	if err != nil {
		t.Errorf("Failed to get client-1 DHCP IP: %v", err)
		return
	}

	// Log the actual IP for debugging
	t.Logf("Client-1 actual IP on eth0 (DHCP): %s", client1IP)

	// Check if the IP is in the expected DHCP range
	if !isIPInRange(client1IP, dhcpRangeStart, dhcpRangeEnd) {
		t.Errorf("Client-1 DHCP IP %s is not in expected range %s-%s", client1IP, dhcpRangeStart, dhcpRangeEnd)
	}

	// Check client-2 DHCP assignment (actual interface IP)
	client2IP, err := getContainerActualIP(testDir, "test-client-2-1", "eth0")
	if err != nil {
		t.Errorf("Failed to get client-2 DHCP IP: %v", err)
		return
	}

	// Log the actual IP for debugging
	t.Logf("Client-2 actual IP on eth0 (DHCP): %s", client2IP)

	if !isIPInRange(client2IP, dhcpRangeStart, dhcpRangeEnd) {
		t.Errorf("Client-2 DHCP IP %s is not in expected range %s-%s", client2IP, dhcpRangeStart, dhcpRangeEnd)
	}

	t.Logf("DHCP assignments verified: client-1=%s, client-2=%s", client1IP, client2IP)
}

// testStaticIPConfiguration verifies that static IP addresses are configured correctly
func testStaticIPConfiguration(t *testing.T, ctx context.Context, testDir string) {
	// Wait for static IP configuration to complete
	time.Sleep(5 * time.Second)

	// Check container logs to see if static configuration is working
	logCmd := exec.Command("docker", "compose", "logs", "client-1")
	logCmd.Dir = testDir
	logOutput, err := logCmd.Output()
	if err != nil {
		t.Logf("Failed to get client-1 logs: %v", err)
	} else {
		t.Logf("Client-1 logs (static config):\n%s", string(logOutput))
	}

	// Check client-1 static IP (actual interface IP)
	client1StaticActual, err := getContainerActualIP(testDir, "test-client-1-1", "eth1")
	if err != nil {
		t.Errorf("Failed to get client-1 static IP: %v", err)
		return
	}

	t.Logf("Client-1 actual IP on eth1 (static): %s (expected: %s)", client1StaticActual, client1StaticIP)

	// Check if the static IP matches exactly what we configured
	if client1StaticActual != client1StaticIP {
		t.Errorf("Client-1 static IP mismatch: expected %s, got %s", client1StaticIP, client1StaticActual)
	}

	// Check client-2 static IP (actual interface IP)
	client2StaticActual, err := getContainerActualIP(testDir, "test-client-2-1", "eth1")
	if err != nil {
		t.Errorf("Failed to get client-2 static IP: %v", err)
		return
	}

	t.Logf("Client-2 actual IP on eth1 (static): %s (expected: %s)", client2StaticActual, client2StaticIP)

	if client2StaticActual != client2StaticIP {
		t.Errorf("Client-2 static IP mismatch: expected %s, got %s", client2StaticIP, client2StaticActual)
	}

	t.Logf("Static IP assignments verified: client-1=%s, client-2=%s", client1StaticActual, client2StaticActual)
}

// getContainerIPOnNetwork retrieves the IP address of a container on a specific network
func getContainerIPOnNetwork(testDir, containerName, networkName string) (string, error) {
	// Use docker inspect to get the IP address
	cmd := exec.Command("docker", "inspect", containerName,
		"--format", fmt.Sprintf("{{.NetworkSettings.Networks.%s.IPAddress}}", networkName))
	cmd.Dir = testDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" || ip == "<no value>" {
		return "", fmt.Errorf("no IP address found for container %s on network %s", containerName, networkName)
	}

	return ip, nil
}

// getContainerActualIP retrieves the actual IP address configured on an interface inside the container
func getContainerActualIP(testDir, containerName, interfaceName string) (string, error) {
	// Use docker exec to run ip command inside the container
	cmd := exec.Command("docker", "exec", containerName, "ip", "addr", "show", interfaceName)
	cmd.Dir = testDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get interface info from container %s: %w", containerName, err)
	}

	// Parse the output to extract the IP address
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "inet ") && !strings.Contains(line, "inet6") {
			// Extract IP from line like "inet 192.168.100.10/24 brd 192.168.100.255 scope global eth0"
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "inet" && i+1 < len(fields) {
					ipWithCidr := fields[i+1]
					ip := strings.Split(ipWithCidr, "/")[0]
					return ip, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no IP address found on interface %s in container %s", interfaceName, containerName)
}

// isIPInRange checks if an IP address is within the specified range
func isIPInRange(ipStr, startStr, endStr string) bool {
	ip := net.ParseIP(ipStr)
	start := net.ParseIP(startStr)
	end := net.ParseIP(endStr)

	if ip == nil || start == nil || end == nil {
		return false
	}

	// Convert to 4-byte representation for comparison
	ip = ip.To4()
	start = start.To4()
	end = end.To4()

	if ip == nil || start == nil || end == nil {
		return false
	}

	// Compare byte by byte
	return compareIP(ip, start) >= 0 && compareIP(ip, end) <= 0
}

// isIPInSubnet checks if an IP address is within the specified subnet
func isIPInSubnet(ipStr, subnetStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	_, subnet, err := net.ParseCIDR(subnetStr)
	if err != nil {
		return false
	}

	return subnet.Contains(ip)
}

// compareIP compares two IP addresses, returns -1, 0, or 1
func compareIP(a, b net.IP) int {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
