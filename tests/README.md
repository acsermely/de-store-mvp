# Testing Guide for Federated Storage Network

## Running Tests

### Unit Tests Only
```bash
./tests/scripts/run-unit-tests.sh
```

### Integration Tests (requires Docker)
```bash
./tests/scripts/run-integration-tests.sh
```

### End-to-End Tests (requires Docker Compose)
```bash
./tests/scripts/run-e2e-tests.sh
```

### All Tests
```bash
./tests/scripts/run-all-tests.sh
```

### With Coverage Report
```bash
./tests/scripts/run-all-tests.sh --coverage
```

## Test Categories

### Unit Tests
Fast, isolated tests for individual functions and methods.
- **Location**: `tests/unit/`, `coordinator/internal/*/..._test.go`, `storage-node/internal/*/..._test.go`
- **Runtime**: < 2 minutes
- **Dependencies**: None (uses mocks)

### Integration Tests
Tests for component interactions with real dependencies.
- **Location**: `tests/integration/`
- **Runtime**: 5-10 minutes
- **Dependencies**: Docker, PostgreSQL, SQLite

### End-to-End Tests
Full system tests simulating real user workflows.
- **Location**: `tests/e2e/`
- **Runtime**: 10-20 minutes
- **Dependencies**: Docker Compose, full system stack

### Load Tests
Performance and stress testing.
- **Location**: `tests/load/`
- **Runtime**: 30-60 minutes
- **Dependencies**: Docker Compose, monitoring tools

## Test Data

Test fixtures are located in `tests/fixtures/`:
- `test-files/1KB.bin` - 1 kilobyte random data
- `test-files/1MB.bin` - 1 megabyte random data
- `test-files/10MB.bin` - 10 megabytes random data

## Writing New Tests

### Unit Test Example
```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := "test data"
    expected := "expected result"
    
    // Act
    result := FunctionUnderTest(input)
    
    // Assert
    assert.Equal(t, expected, result)
}
```

### Integration Test Example
```go
func TestComponentIntegration(t *testing.T) {
    // Setup test database
    db := testsupport.SetupTestDB(t)
    defer db.Close()
    
    // Run test
    // ...
}
```

### E2E Test Example
```go
func TestFileUploadDownload(t *testing.T) {
    // Setup test environment
    env := e2e.SetupTestEnvironment(t)
    defer env.Teardown()
    
    // Register user
    user := env.RegisterUser("test@example.com", "password123")
    
    // Upload file
    fileID := env.UploadFile(user.Token, "test-data.txt", []byte("Hello World"))
    
    // Download and verify
    data := env.DownloadFile(user.Token, fileID)
    assert.Equal(t, "Hello World", string(data))
}
```

## Continuous Integration

Tests are run automatically on:
1. Every pull request (unit + integration)
2. Merge to main (unit + integration + e2e)
3. Nightly builds (unit + integration + e2e + load)

## Troubleshooting

### Tests fail with "database connection refused"
Make sure PostgreSQL is running: `docker-compose up -d postgres`

### Storage node tests fail
Ensure SQLite development libraries are installed: `apt-get install libsqlite3-dev`

### P2P tests timeout
libp2p requires network access. Disable firewall or run in Docker.

### Coverage report not generated
Install coverage tools: `go install github.com/axw/gocov/gocov@latest`