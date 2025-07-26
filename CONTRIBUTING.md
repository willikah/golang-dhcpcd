# Contributing to golang-dhcpcd

Thank you for your interest in contributing to golang-dhcpcd! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/golang-dhcpcd.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Test your changes
6. Commit and push your changes
7. Create a pull request

## Development Setup

### Prerequisites
- Go 1.23 or later
- Docker and Docker Compose
- Make

### Building the Project
```bash
make build
```

### Running Tests
```bash
# Run unit tests
make test

# Run integration tests (requires Docker)
make test-integration

# Run all tests
make test-all
```

### Code Style
- Follow standard Go conventions
- Use `gofmt` to format your code
- Run `make lint` to check for style issues
- Write meaningful commit messages

## Testing

### Unit Tests
- Write unit tests for all new functionality
- Ensure existing tests continue to pass
- Aim for good test coverage

### Integration Tests
- Integration tests are located in the `test/` directory
- They use Docker Compose to test real DHCP and static IP functionality
- Run with `make test-integration`

## Pull Request Process

1. Update documentation if needed
2. Add tests for new functionality
3. Ensure all tests pass
4. Update the README.md if needed
5. Follow the pull request template
6. Wait for review and address feedback

## Code Review Guidelines

### For Contributors
- Keep pull requests focused and small
- Write clear commit messages
- Respond to feedback promptly
- Update your branch if needed

### For Reviewers
- Be constructive and respectful
- Focus on code quality and maintainability
- Test the changes if possible
- Approve when ready

## Reporting Issues

When reporting issues, please include:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Configuration files (remove sensitive data)
- Relevant logs

## Feature Requests

When requesting features:
- Describe the use case
- Explain why the feature would be useful
- Consider if it fits the project's scope
- Provide examples if applicable

## Security Issues

If you discover a security vulnerability, please:
1. Do NOT create a public issue
2. Email the maintainer directly
3. Provide details about the vulnerability
4. Allow time for the issue to be addressed

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project.

## Questions?

If you have questions about contributing, feel free to:
- Open an issue for discussion
- Contact the maintainers
- Check existing documentation

Thank you for contributing! 🎉
