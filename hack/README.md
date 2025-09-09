# Hack Directory

This directory contains development scripts and utilities for the CRUD application.

## Scripts

### `lint.sh`
Basic Go linting script that serves as a fallback when golangci-lint has configuration issues.

**Usage:**
```bash
# Run linting checks
./hack/lint.sh

# Or via make target
make lint
```

**Features:**
- Code formatting verification with `gofmt`
- Static analysis with `go vet`
- Optional tools (if installed): `ineffassign`, `misspell`, `gosec`
- Colored output with detailed reporting
- Graceful handling of missing optional tools

### `integration-test.sh`
Comprehensive integration test script that validates the application's health endpoints, security headers, and API functionality.

**Usage:**
```bash
# Run with default settings (localhost:8080)
./hack/integration-test.sh

# Run with custom host/port
./hack/integration-test.sh --host 127.0.0.1 --port 3000

# Show help
./hack/integration-test.sh --help
```

**Features:**
- Health endpoint validation (`/health/live`, `/health/ready`, `/health/`, `/health/metrics`)
- Security header verification (CSP, XSS protection, frame options)
- CORS header validation
- Rate limiting testing
- Public endpoint accessibility checks
- Colored output with detailed reporting

**Requirements:**
- Running CRUD application
- `curl` command available
- `python3` for JSON validation (optional)

## Make Targets

The following make targets use scripts from this directory:

- `make test-integration` - Run integration tests (builds app first)
- `make test-all` - Run both unit and integration tests

## CI/CD Integration

These scripts are automatically executed in the GitHub Actions CI pipeline:
- Integration tests run after successful build
- Application is started in background for testing
- Comprehensive validation of all endpoints and security features

## Adding New Scripts

When adding new development scripts to this directory:

1. Make them executable: `chmod +x hack/your-script.sh`
2. Add appropriate copyright header
3. Include help/usage information
4. Update this README
5. Consider adding a corresponding make target
6. Test locally before committing
