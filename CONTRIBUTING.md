# Contributing to rentfree

Thank you for your interest in contributing to rentfree! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker (for running Redis integration tests)
- Git

### Setting Up Development Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/LavishGent/rentfree.git
   cd rentfree
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Verify your setup:
   ```bash
   go build ./...
   go test ./...
   ```

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/cache/...

# Run tests with race detector
go test -race ./...
```

### Running Integration Tests

Integration tests require Redis:

```bash
# Start Redis using Docker
docker run -d -p 6379:6379 --platform linux/arm64 redis:alpine

# Run integration tests
go test -v -run "Redis|Integration" ./internal/cache/...

# With custom Redis address
REDIS_TEST_ADDRESS=localhost:6380 go test -v -run "Redis" ./internal/cache/...
```

### Running Examples

```bash
cd examples/basic
go run main.go
```

## Code Style

- Follow standard Go conventions and idioms
- Run `gofmt` on your code before committing
- Keep functions focused and concise
- Add comments for exported functions and types
- Write meaningful commit messages

### Linting

We use `golangci-lint` for code quality:

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## Testing Guidelines

- Write tests for all new functionality
- Maintain or improve existing test coverage
- Use table-driven tests where appropriate
- Mock external dependencies in unit tests
- Use integration tests for Redis functionality
- Ensure tests are deterministic and isolated

### Test Coverage Goals

- New packages: >80% coverage
- Modified packages: Don't decrease coverage
- Critical paths: 100% coverage

## Pull Request Process

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes:**
   - Write code following our style guidelines
   - Add or update tests
   - Update documentation as needed

3. **Verify your changes:**
   ```bash
   go test ./...
   go build ./...
   golangci-lint run
   ```

4. **Commit your changes:**
   ```bash
   git add .
   git commit -m "Add feature: your feature description"
   ```

5. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request:**
   - Provide a clear description of changes
   - Reference any related issues
   - Ensure CI passes

### Pull Request Requirements

- All tests must pass
- Code must be properly formatted
- No linter warnings
- Documentation updated if needed
- Commit messages are clear and descriptive

## Project Structure

```
rentfree/
├── examples/          # Example applications
├── internal/          # Internal implementation
│   ├── cache/        # Cache implementations
│   ├── config/       # Configuration
│   ├── metrics/      # Metrics tracking
│   ├── resilience/   # Circuit breaker, retry, bulkhead
│   └── types/        # Shared types
├── pkg/
│   └── rentfree/     # Public API
└── .github/          # CI/CD workflows
```

## Reporting Issues

### Bug Reports

Include:
- Go version
- Operating system
- Minimal reproduction steps
- Expected vs actual behavior
- Relevant logs or error messages

### Feature Requests

Include:
- Use case description
- Proposed API or behavior
- Alternatives considered
- Potential impact

## Known Limitations

- Memory cache (bigcache) uses global TTL only
- Per-entry TTL only works with Redis layer
- See CHANGELOG.md for full list

## Questions?

For questions or discussions, please open an issue with the "question" label.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
