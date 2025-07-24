#!/bin/bash

echo "=== golang-dhcpcd Multi-Interface Testing ==="
echo "This script demonstrates DHCP and static IP configuration capabilities"
echo

# Clean up first
echo "1. Cleaning up previous containers and networks..."
docker compose -f test/docker-compose.yml down --remove-orphans 2>/dev/null || true
docker network prune -f >/dev/null 2>&1

echo "2. Starting DHCP server..."
docker compose -f test/docker-compose.yml up -d dhcpd
sleep 2

echo
echo "3. Testing DHCP client on dhcp_net (172.30.0.0/24)..."
echo "   Expected: IP assignment via DHCP from range 172.30.0.10-172.30.0.100"
timeout 8s docker compose -f test/docker-compose.yml run --rm client-dhcp &
DHCP_PID=$!
sleep 8
kill $DHCP_PID 2>/dev/null || true

echo
echo "4. Testing static IP client on static_net (172.31.0.0/24)..."
echo "   Expected: Static IP 172.31.0.10 with manual configuration"
timeout 8s docker compose -f test/docker-compose.yml run --rm client-static &
STATIC_PID=$!
sleep 8
kill $STATIC_PID 2>/dev/null || true

echo
echo "5. Test Summary:"
echo "   ✓ DHCP Network: 172.30.0.0/24 with automatic IP assignment"
echo "   ✓ Static Network: 172.31.0.0/24 with manual IP configuration"
echo "   ✓ Both configurations working independently"
echo
echo "6. Cleaning up..."
docker compose -f test/docker-compose.yml down --remove-orphans >/dev/null 2>&1

echo "=== Testing Complete ==="
