# CI/CD Workflows

This directory contains GitHub Actions workflows for continuous integration and testing.

## Workflows

### `ci.yml` - Comprehensive CI Pipeline
- **Triggers**: Pull requests and pushes to `main`
- **Jobs**:
  - **Test**: Runs unit tests with coverage reporting
  - **Build**: Builds the server binary
  - **Lint**: Code quality checks with golangci-lint
  - **Integration Test**: End-to-end testing with real Loki instance

## Timestamp Bug Protection

The workflow includes specific tests for the timestamp parsing bug (issue #3) that was fixed:

- Unit tests verify that timestamps show correct years (2023, 2024, etc.) instead of 2262
- Integration tests confirm the fix works with real Loki data
- Regression tests prevent the bug from being reintroduced

## Test Coverage

Our tests cover:
- Timestamp parsing with nanosecond precision
- Multiple timestamp formats and edge cases
- Invalid timestamp fallback behavior
- Empty result handling
- Stream formatting and labeling
- Integration with real Loki instances

## Running Tests Locally

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run tests with race detection
go test -race ./...

# Build verification
go build ./cmd/loki-mcp
```

## Adding New Tests

When adding new functionality:

1. Add unit tests to the appropriate `*_test.go` file
2. For timestamp-related changes, add regression tests to prevent bug #3
3. Ensure tests pass locally before submitting PR
4. CI will automatically run on PR creation
