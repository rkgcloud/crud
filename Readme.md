# CRUD Application

A secure, production-ready CRUD application built with Go, featuring comprehensive security middleware, health checks, and enterprise-grade reliability.

## Features

- 🔒 **Security**: Rate limiting, CORS protection, security headers, input validation
- 📊 **Monitoring**: Health checks, metrics, structured logging
- 🧪 **Testing**: Comprehensive unit and integration tests
- 🚀 **Production Ready**: Graceful shutdown, configuration management
- 🔧 **CI/CD**: Automated testing, security scanning, and deployment

## Quick Start

### Local Development

1. **Start PostgreSQL database:**
   ```shell
   make run-db
   ```

2. **Set environment variables:**
   ```shell
   export KO_DATA_PATH=$(pwd)/kodata
   export DATABASE_URL="host=localhost user=postgres password=mysecretpassword dbname=postgres sslmode=disable"
   export SECRET="your-32-character-secret-key-here"
   ```

3. **Run the application:**
   ```shell
   make run
   ```

### Testing

```shell
# Run unit tests
make test

# Run integration tests (requires running app)
make test-integration

# Run all tests
make test-all
```

### Building

```shell
# Build the application
make build

# The binary will be available at .bin/crud
```

## Health Endpoints

The application provides several health check endpoints:

- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe with database connectivity check
- `GET /health/` - Comprehensive health status
- `GET /health/metrics` - Application metrics and runtime information

## Configuration

All configuration is environment-based with sensible defaults:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DEBUG` | `false` | Debug mode |
| `SECRET` | *(required)* | Session secret (min 32 chars) |
| `DATABASE_URL` | `host=localhost...` | PostgreSQL connection string |
| `RATE_LIMIT_PER_MINUTE` | `60` | Rate limit per IP per minute |
| `ALLOWED_ORIGINS` | `http://localhost:8080` | CORS allowed origins |

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment:

### Workflow Jobs

1. **Lint** - Code quality checks with golangci-lint
2. **Test** - Unit tests with PostgreSQL service
3. **Build** - Application build verification
4. **Security** - Security scanning with Gosec
5. **Integration** - End-to-end integration tests

### Security Features

- **Gosec** security scanning
- **Dependency scanning** with GitHub's security features
- **SARIF** upload for security findings
- **Codecov** integration for test coverage

### Running CI Locally

You can run the same checks locally:

```shell
# Lint
golangci-lint run

# Test with coverage
make test

# Build
make build

# Integration tests
./hack/integration-test.sh
```

## Kubernetes Deployment

### Install CRUD database
```shell
make db-deploy
```

### Install CRUD application
```shell
make deploy
```

### Connect from cluster
```shell
kubectl port-forward service/go-postgres-crud-service 8080:8080
```

## Security

This application implements multiple layers of security:

- **Rate Limiting**: IP-based rate limiting with configurable limits
- **CORS Protection**: Configurable CORS with secure defaults  
- **Security Headers**: CSP, XSS protection, frame options, HSTS
- **Input Validation**: Comprehensive validation for all user inputs
- **Session Security**: Secure session configuration with HttpOnly, Secure, SameSite
- **SQL Injection Protection**: Parameterized queries and GORM protections
- **OAuth Security**: CSRF-protected OAuth flow with secure state tokens

## Architecture

```
├── cmd/                    # Application entrypoints
├── pkg/
│   ├── config/            # Configuration management
│   ├── controllers/       # HTTP request handlers
│   ├── middleware/        # Security middleware
│   ├── models/           # Data models
│   ├── health/           # Health checks and metrics
│   ├── auth/             # Authentication types
│   ├── session/          # Session management
│   └── database/         # Database connectivity
├── hack/                 # Development scripts and utilities
├── .github/workflows/    # CI/CD pipelines
└── kodata/              # Static assets and templates
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `make test-all` to verify
5. Submit a pull request

The CI pipeline will automatically run all checks on your pull request.


