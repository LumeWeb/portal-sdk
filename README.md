# Portal SDK

Go SDK for the Portal Account API. A type-safe, easy-to-use client library generated from OpenAPI specifications.

[![Go Report Card](https://goreportcard.com/badge/go.lumeweb.com/portal-sdk)](https://goreportcard.com/report/go.lumeweb.com/portal-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)

## Features

- **Type-Safe Client**: Auto-generated from OpenAPI specification ensures API contract compliance
- **JWT Authentication**: Built-in JWT token handling for secure API access
- **2FA Support**: Complete OTP (One-Time Password) generation, verification, and management
- **Operation Polling**: Built-in support for long-running operations with configurable timeouts
- **Functional Options Pattern**: Clean, extensible client configuration
- **Builder Pattern**: Intuitive query building for filters, sorting, and pagination
- **Comprehensive Error Handling**: Centralized error mapping with custom error types

## Installation

```bash
go get go.lumeweb.com/portal-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go.lumeweb.com/portal-sdk/account"
)

func main() {
    // Create a new client with JWT token
    client := account.NewClient(
        account.WithJWT("your-jwt-token"),
        account.WithEndpoint("https://account.pinner.xyz"),
    )

    ctx := context.Background()

    // Ping the API
    resp, err := client.Ping(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("API Status: %s\n", resp.Status)
}
```

## Usage

### Authentication

#### Login with Username/Password

```go
loginResp, err := client.Login(ctx, account.LoginRequest{
    Username: "user@example.com",
    Password: "password123",
})
if err != nil {
    log.Fatal(err)
}

// Use the JWT token for subsequent requests
client = account.NewClient(
    account.WithJWT(loginResp.JWT),
)
```

#### Two-Factor Authentication (OTP)

Generate an OTP code:

```go
otpResp, err := client.GenerateOTP(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("OTP Code: %s\n", otpResp.Code)
```

Validate OTP:

```go
_, err := client.ValidateOTP(ctx, account.ValidateOTPRequest{
    Code: "123456",
})
if err != nil {
    log.Fatal(err)
}
```

Disable OTP:

```go
_, err := client.DisableOTP(ctx, account.DisableOTPRequest{
    Password: "password123",
})
if err != nil {
    log.Fatal(err)
}
```

### Long-Running Operations

Many operations in the Portal API are asynchronous. Use `WaitForOperation` to poll until completion:

```go
import (
    "errors"
    "fmt"
    "time"
)

// Start an operation
startResp, err := client.StartSomeOperation(ctx, params)
if err != nil {
    log.Fatal(err)
}

// Wait for the operation to complete
op, err := client.WaitForOperation(ctx, startResp.OperationID,
    account.WithPollInterval(2*time.Second),
    account.WithPollTimeout(5*time.Minute),
    account.WithPollSettledStates(account.OperationStatusCompleted),
)
if err != nil {
    if errors.Is(err, account.ErrOperationTimeout) {
        log.Fatal("Operation timed out")
    }
    log.Fatal(err)
}

fmt.Printf("Operation completed: %s\n", op.Status)
```

### List Operations with Filters, Sorting, and Pagination

```go
import (
    "fmt"
    "time"

    "go.lumeweb.com/queryutil"
)

// Build filters
filters := []queryutil.Filter{
    queryutil.NewFilter("status").Equals("completed"),
    queryutil.NewFilter("created_at").Gte(time.Now().AddDate(0, -1, 0)),
}

// Build sorts
sorts := []queryutil.Sort{
    queryutil.NewSort("created_at").Desc(),
}

// Configure pagination
pagination := &queryutil.Pagination{
    Page:     1,
    PageSize: 20,
}

// List operations
ops, err := client.ListOperations(ctx,
    account.WithFilters(filters...),
    account.WithSorts(sorts...),
    account.WithPagination(pagination),
    account.WithSearch("backup"),
)
if err != nil {
    log.Fatal(err)
}

for _, op := range ops {
    fmt.Printf("%s: %s\n", op.ID, op.Status)
}
```

### Client Configuration Options

```go
client := account.NewClient(
    account.WithJWT("your-jwt-token"),
    account.WithEndpoint("https://api.example.com"),
    account.WithDisableFollowRedirect(),
)
```

## Architecture

The SDK follows a layered architecture:

1. **OpenAPI Specification** (`swagger.yaml`) - Source of truth for API contract
2. **Generated Client** (`client/client.gen.go`) - Auto-generated HTTP client
3. **Public API** (`account.go`) - High-level abstractions and convenience methods
4. **Testing Layer** (`account_test.go`) - Unit tests with HTTP mocking

## Development

### Prerequisites

- Go 1.25 or later
- `oapi-codegen` for code generation (configured in go.mod)
- `mockery` for mock generation

### Build

```bash
go build ./...
```

### Generate Client Code

```go generate ./...
```

This regenerates `client/client.gen.go` from `swagger.yaml`.

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...

# Run tests with coverage profile
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Generate Mocks

```bash
mockery
```

### Dependency Management

```bash
go mod tidy
go mod verify
```

### Release Management

This project uses [Knope](https://github.com/knope-dev/knope) for release management:

```bash
# Prepare a release (updates changelog, bumps version, commits changes)
knope prepare-release

# Create a release (creates GitHub release, tags commit)
knope release

# Document a change (creates a change file for changelog)
knope document-change
```

## API Documentation

### Operation Status

Operations have the following statuses:

- `pending` - Operation is queued
- `running` - Operation is in progress
- `completed` - Operation finished successfully
- `failed` - Operation failed
- `error` - Operation encountered an error

### Error Handling

The SDK provides custom error types for common scenarios:

```go
import "errors"

if errors.Is(err, account.ErrUnauthorized) {
    // Handle authentication errors
}

if errors.Is(err, account.ErrOperationTimeout) {
    // Handle timeout errors
}
```

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code follows the project's existing patterns
3. New features include tests
4. API contract changes must be made on the server side first; manually update `swagger.yaml` from the server and run `go generate ./...` to regenerate the client code

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.