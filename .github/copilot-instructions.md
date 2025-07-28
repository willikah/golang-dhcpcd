# golang-dhcpcd AI Assistant Instructions

## Project Overview
This is a Go DHCP client daemon that configures network interfaces with either DHCP or static IP addresses. The project follows **Hexagonal Architecture (Ports and Adapters)** pattern with clean separation between business logic and infrastructure concerns.

## Architecture Patterns

### Hexagonal Architecture Implementation
- **Primary Port**: `internal/port/network.go` defines `NetworkConfigurationManager` interface
- **Adapters**: `internal/adapter/{dhcp,static}/manager.go` implement the port for different protocols
- **Factory Pattern**: `cmd/serve.go:createNetworkConfigurationManager()` creates appropriate adapters based on config

### Key Directory Structure
```
cmd/           # CLI entry points (Cobra-based)
internal/
  port/        # Business interfaces (hexagonal ports)
  adapter/     # Infrastructure implementations (hexagonal adapters)
    dhcp/      # DHCP client using insomniacslk/dhcp + netlink
    static/    # Static IP using netlink directly
  pkg/         # Shared utilities (config, logging)
  types/       # Domain types
test/          # Integration tests using Docker Compose
```

## Critical Development Patterns

### Network Management Approach
- **Never use `ip` commands**: All network operations use `github.com/vishvananda/netlink` library
- **Address deduplication**: Both adapters check existing IPs before applying changes (see `configureDefaultRoute()` pattern)
- **Graceful renewal**: DHCP manager uses single timer-based renewal loop with context cancellation

### Logging Architecture
- **Structured logging**: Uses logrus with component/interface context via `logging.WithComponentAndInterface()`
- **Custom formatter**: `CompactFormatter` for concise output showing `[TIME][LEVEL][component][interface]`
- **Error context**: Always include interface name and relevant network details in log fields

### Configuration System
- **YAML-based**: Single config file with validation in `config.Validate()`
- **Interface-centric**: Config organized by interface name with `dhcp: bool` or `static: {ip, netmask, gateway}`
- **Type conversion**: Config types (`config.StaticConfig`) convert to domain types (`types.StaticIPConfig`)

## Development Workflow

### Building and Testing
```bash
make build              # Standard build with version from git tags
make test               # All tests (unit + integration)
make test-unit          # Unit tests only (tags=unit, uses mocks)
make test-integration   # Docker-based integration tests
make generate           # Generate mocks and embedded files
```

### Mock-based Unit Testing
- Unit tests use build tag `//go:build unit` and uber/mock for dependency injection
- Mocks auto-generated in `internal/mock/` via `make generate`
- Tests cover business logic without requiring real network interfaces or files
- Integration tests in `test/` directory use real Docker environment

### Testing Patterns
- **Unit Tests**: Use mocks from `internal/mock/`, build tag `//go:build unit`
- **Integration Tests**: Real DHCP/network operations with Docker Compose
- **Mock Generation**: `internal/mock/generate.go` creates mocks for all ports
- **Test Structure**: Arrange-Act-Assert pattern with gomock for behavior verification

### Docker Development
- Multi-stage Dockerfile with distroless final image
- Requires `--privileged` and `--network host` for network interface access
- Test environment via `make up/down` using docker-compose

## Code Conventions

### Error Handling
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Log errors with structured fields before returning
- Retry logic: See `getDHCPLease()` for retry pattern with exponential backoff

### Interface Management
- Always use `netlink.LinkByName()` to get interface handles
- Check existing configuration before applying changes to avoid unnecessary operations
- Use `net.Interface` for metadata, `netlink.Link` for operations

### Concurrency Patterns
- Each interface runs in separate goroutine with shared context for cancellation
- Use `sync.WaitGroup` in `serve.go` to coordinate multiple interface managers
- Timer-based operations use single `time.Timer` with `Reset()` for efficiency

## Key Files for Understanding
- `internal/port/network.go`: Core business interface
- `cmd/serve.go`: Application orchestration and adapter factory
- `internal/adapter/dhcp/manager.go`: DHCP implementation with lease renewal
- `internal/adapter/static/manager.go`: Static IP with monitoring
- `test/integration_test.go`: Real-world usage patterns and validation

## Common Tasks
- **Adding new adapter**: Implement `NetworkConfigurationManager`, add factory case in `createNetworkConfigurationManager()`
- **Network debugging**: Use `logging.WithComponentAndInterface()` with relevant fields (IP, gateway, etc.)
- **Config changes**: Update `config.go` types, add validation in `Validate()`, handle in adapters
- **Testing network features**: Add to `test/integration_test.go` with Docker environment
