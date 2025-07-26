# golang-dhcpcd

[![CI](https://github.com/willikah/golang-dhcpcd/actions/workflows/ci.yml/badge.svg)](https://github.com/willikah/golang-dhcpcd/actions/workflows/ci.yml)
[![Security Scan](https://github.com/willikah/golang-dhcpcd/actions/workflows/security.yml/badge.svg)](https://github.com/willikah/golang-dhcpcd/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/willikah/golang-dhcpcd)](https://goreportcard.com/report/github.com/willikah/golang-dhcpcd)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A modern DHCP client and static IP configuration daemon written in Go. This tool allows you to configure network interfaces with either DHCP or static IP addresses using a simple YAML configuration file.

## Features

- **DHCP Client**: Full-featured DHCP client with lease management and renewal
- **Static IP Configuration**: Configure static IP addresses with gateway and DNS
- **Multi-Interface Support**: Configure multiple network interfaces simultaneously
- **Structured Logging**: Comprehensive logging with different levels and formats
- **Configuration Validation**: YAML configuration with validation
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Containerized**: Ready-to-use Docker images available

## Installation

### From Releases
Download the latest binary from the [releases page](https://github.com/willikah/golang-dhcpcd/releases).

### From Source
```bash
git clone https://github.com/willikah/golang-dhcpcd.git
cd golang-dhcpcd
make build
```

### Using Go Install
```bash
go install github.com/willikah/golang-dhcpcd@latest
```

### Using Docker
```bash
docker pull ghcr.io/willikah/golang-dhcpcd:latest
```

## Usage

### Basic Usage
```bash
# Run with configuration file
golang-dhcpcd serve -f config.yml

# Show version
golang-dhcpcd version

# Show help
golang-dhcpcd --help
```

### Configuration

Create a `config.yml` file:

```yaml
logging:
  level: info      # debug, info, warn, error
  format: simple   # simple, json

interfaces:
  eth0:
    dhcp: true
  
  eth1:
    dhcp: false
    static:
      ip: 192.168.1.100
      netmask: 255.255.255.0
      gateway: 192.168.1.1
```

### Docker Usage
```bash
# Create your configuration
cat > golang-dhcpcd.yml << EOF
logging:
  level: info
  format: simple

interfaces:
  eth0:
    dhcp: true
EOF

# Run with Docker
docker run --rm --privileged \
  --network host \
  -v $(pwd)/golang-dhcpcd.yml:/app/config.yml \
  ghcr.io/willikah/golang-dhcpcd:latest \
  serve -f /app/config.yml
```

## Configuration Reference

### Logging Configuration
```yaml
logging:
  level: info      # Log level: debug, info, warn, error
  format: simple   # Log format: simple, json
```

### Interface Configuration
```yaml
interfaces:
  <interface_name>:
    dhcp: true|false
    static:          # Only used when dhcp: false
      ip: "x.x.x.x"
      netmask: "x.x.x.x"
      gateway: "x.x.x.x"
```

## Development

### Prerequisites
- Go 1.23 or later
- Docker and Docker Compose (for integration tests)
- Make

### Building
```bash
make build
```

### Testing
```bash
# Run unit tests
make test

# Run integration tests (requires Docker)
make test-integration

# Run all tests
make test-all

# Run linting
make lint
```

### Development with Docker
```bash
# Build Docker images
make docker-build

# Start test environment
make up

# View logs
make logs

# Stop test environment
make down

# Check for network conflicts
make check-conflicts
```

## Architecture

```
┌─────────────────┐
│   Configuration │
│   (YAML)        │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐    ┌──────────────┐    ┌──────────────┐
│   CLI Command   │────│   Logging    │────│   Validation │
│   (Cobra)       │    │   (Logrus)   │    │              │
└─────────┬───────┘    └──────────────┘    └──────────────┘
          │
          ▼
┌─────────────────┐
│   Interface     │
│   Manager       │
└─────────┬───────┘
          │
          ├─────────────────────┬─────────────────────
          ▼                     ▼
┌─────────────────┐    ┌─────────────────┐
│   DHCP Client   │    │  Static Client  │
│   (DHCPv4)      │    │  (Netlink)      │
└─────────────────┘    └─────────────────┘
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Quick Start for Contributors
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run tests: `make test-all`
6. Submit a pull request

## Security

If you discover a security vulnerability, please send an email to the maintainer instead of opening a public issue.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [insomniacslk/dhcp](https://github.com/insomniacslk/dhcp) - DHCP library
- [vishvananda/netlink](https://github.com/vishvananda/netlink) - Network interface management
- [spf13/cobra](https://github.com/spf13/cobra) - CLI framework
- [sirupsen/logrus](https://github.com/sirupsen/logrus) - Structured logging

## Support

- [GitHub Issues](https://github.com/willikah/golang-dhcpcd/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/willikah/golang-dhcpcd/discussions) - Questions and general discussion
