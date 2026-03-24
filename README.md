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
- **Admin API**: Full admin API for quota management, user administration, and system configuration
- **Multi-Spec Architecture**: Separate OpenAPI specs for user-facing and admin APIs

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

    // Example: Get current quota status
    quota, err := client.GetQuota(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Upload used: %d%%\n", quota.Upload.Percentage)
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

### Admin API

```go
import "go.lumeweb.com/portal-sdk/admin"

// Create admin client
adminClient := admin.NewClient(
    admin.WithJWT("admin-jwt-token"),
    admin.WithEndpoint("https://api.pinner.xyz"),
)

// List quota plans
plans, total, err := adminClient.Quota().ListPlans(ctx)
if err != nil {
    log.Fatal(err)
}

for _, plan := range plans {
    fmt.Printf("Plan: %s - Upload: %d GB\\n", plan.Name, plan.UploadTotalLimit/1024/1024/1024)
}

// Create a quota allowance for a user
expiry := time.Now().Add(30 * 24 * time.Hour)
allowance, err := adminClient.Quota().CreateAllowance(ctx,
    123,              // user ID
    "manual",         // source
    "one-time",       // type
    100*1024*1024,    // upload bytes
    500*1024*1024,    // download bytes
    10*1024*1024,     // storage bytes
    expiry,           // expiry date
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Allowance created with ID: %s\\n", allowance.AllowanceID)
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

The SDK follows a multi-spec, layered architecture:

### Packages

1. **Account Package** (`account/`)
   - User-facing account API
   - JWT authentication and 2FA
   - Quota status and history
   - Profile and settings management

2. **Admin Package** (`admin/`)
   - Admin quota API (plans, allowances, config, stats, reconcile, cleanup)
   - Extensible for future admin APIs (billing, user CRUD, etc.)

### Layered Architecture

1. **OpenAPI Specifications** (`specs/`)
   - `specs/account.yaml` - User-facing account API specification
   - `specs/admin/quota.yaml` - Admin quota API specification
   - Source of truth for all generated client code

2. **Generated Client Layer**
   - `internal/client/client.gen.go` - Account API generated client
   - `internal/admin/client.gen.go` - Admin API generated client
   - Auto-generated by `oapi-codegen` from swagger specs
   - Do NOT edit directly - changes will be overwritten

3. **Public API Layer**
   - `account/account.go` - Public account client with high-level abstractions
   - `admin/admin.go` - Public admin client with service pattern

4. **Testing Layer**
   - `account_test.go` - Account API unit tests
   - `admin/admin_test.go` - Admin API unit tests
   - Generated mocks for interfaces

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

```bash
go generate ./...
```

This regenerates `internal/client/client.gen.go` from `specs/account.yaml` and `internal/admin/client.gen.go` from `specs/admin/quota.yaml`.

### Update OpenAPI Specs from Running Services

When working on local development with Portal API services running, use the swagger-update skill to fetch and update specs:

```bash
# Fetch specs from local services (with Host header spoofing for vhosts)
curl -H "Host: account.localhost:8080" http://localhost:8080/swagger.yaml -o specs/account.yaml
curl -H "Host: admin.localhost:8080" http://localhost:8080/swagger.yaml -o specs/admin/quota.yaml

# Regenerate client code
go generate ./...

# Fix any code changes needed for new structures
# Run tests to verify
go test ./...
```

Note: This requires Portal API services running on localhost:8080 with proper vhost routing configured.

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