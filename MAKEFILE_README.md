# Makefile for Federated Storage Network

A comprehensive Makefile for easy development and testing of the Federated Storage Network.

## Quick Start - Web UI Testing

The easiest way to test the Web UI:

```bash
# Start everything with one command
make webui-test
```

This will:
1. ✅ Start PostgreSQL in Docker
2. ✅ Create test user (test@example.com / testpassword123)
3. ✅ Start Coordinator
4. ✅ Initialize and start Storage Node
5. ✅ Display Web UI URL

Then open: **http://localhost:8080/web/**

## Available Commands

### Setup & Installation

```bash
make install              # Install Go dependencies
make setup-test-user      # Create test user in database
```

### Web UI Testing (Recommended)

```bash
make webui-test           # Start complete Web UI test environment
make webui-quick          # Quick start (assumes DB running)
make webui-stop           # Stop all services
```

### Individual Components

```bash
make coordinator          # Start coordinator only
make init-node            # Initialize storage node
make storage-node         # Start storage node
```

### Docker Operations

```bash
make docker-up            # Start all services with Docker Compose
make docker-up-db         # Start PostgreSQL only
make docker-down          # Stop all Docker services
make docker-build         # Build Docker images
make docker-logs          # View Docker logs
```

### Testing

```bash
make test                 # Run all unit tests
make test-coordinator     # Run coordinator tests only
make test-storage         # Run storage node tests only
make test-integration     # Run integration tests
```

### Utilities

```bash
make build                # Build binaries
make clean                # Clean build artifacts
make reset-db             # Reset database (WARNING: deletes data)
make logs                 # Show service logs
make status               # Check service status
make fmt                  # Format Go code
make vet                  # Run go vet
make lint                 # Run all linting
```

## Usage Examples

### Example 1: First Time Setup

```bash
# Install dependencies
make install

# Start the full Web UI environment
make webui-test

# Open browser
# http://localhost:8080/web/
# Login: test@example.com / testpassword123
```

### Example 2: Daily Development

```bash
# Check if services are running
make status

# If not, start them
make webui-quick

# Run tests while developing
make test

# Check logs
make logs

# Clean up when done
make webui-stop
```

### Example 3: Using Docker Compose

```bash
# Start everything with Docker
make docker-up

# View logs
make docker-logs

# Stop everything
make docker-down
```

### Example 4: Reset Everything

```bash
# WARNING: This deletes all data!
make reset-db

# Or manually:
make docker-down
make clean
make docker-up-db
make setup-test-user
make webui-test
```

## Command Details

### `make webui-test`

The main command for testing the Web UI. It:

1. Starts PostgreSQL in Docker
2. Waits for it to be ready
3. Creates test user with 10,000 credits
4. Starts coordinator in background
5. Initializes storage node
6. Starts storage node in background
7. Displays success message with Web UI URL

**Test User Credentials:**
- Email: `test@example.com`
- Password: `testpassword123`
- Credits: 10,000

### `make webui-quick`

Fast start when PostgreSQL is already running. Skips:
- Docker operations
- Test user creation

Useful when you've already run `make webui-test` once and just need to restart the services.

### `make webui-stop`

Stops all running services (coordinator and storage node). Note: This doesn't stop PostgreSQL Docker container (use `make docker-down` for that).

### `make status`

Shows current status of:
- PostgreSQL (Docker and localhost)
- Coordinator
- Storage Node

Example output:
```
==========================================
Service Status
==========================================

PostgreSQL:
  ✓ Running on localhost:5432

Coordinator:
  ✓ Running

Storage Node:
  ✓ Running

Web UI: http://localhost:8080/web/
```

### `make reset-db`

**WARNING: Destructive operation!**

Resets the entire database:
1. Stops all Docker services
2. Removes Docker volumes (deletes all data)
3. Starts fresh PostgreSQL
4. Creates test user
5. Requires confirmation

## Workflow Examples

### Web UI Development Workflow

```bash
# Terminal 1: Start everything
make webui-test

# Terminal 2: Edit web files
cd coordinator/web/static
# Edit index.html or app.js

# Browser: http://localhost:8080/web/
# Test your changes immediately (no restart needed for static files)

# When done
make webui-stop
```

### Backend Development Workflow

```bash
# Terminal 1: Database
make docker-up-db

# Terminal 2: Coordinator (for watching logs)
make coordinator

# Terminal 3: Storage Node (for watching logs)
make init-node  # Only first time
make storage-node

# Terminal 4: Testing
make test
make test-coordinator
```

### Testing Workflow

```bash
# Run all tests
make test

# Run specific test suites
make test-coordinator
make test-storage

# Check coverage (after running tests with coverage)
cd coordinator && go tool cover -html=coverage.out
```

## Troubleshooting

### "PostgreSQL not running"

```bash
# Start PostgreSQL
make docker-up-db

# Check status
make status
```

### "Port 8080 already in use"

```bash
# Stop all services
make webui-stop

# Or find and kill process
lsof -ti:8080 | xargs kill -9
```

### "Database connection failed"

```bash
# Reset everything
make reset-db

# Then start again
make webui-test
```

### "Test user doesn't exist"

```bash
make setup-test-user
```

## Environment Variables

The Makefile respects these environment variables:

- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database user (default: postgres)
- `DB_PASS` - Database password (default: postgres)
- `DB_NAME` - Database name (default: coordinator)

Example:
```bash
DB_PASS=mysecretpassword make webui-test
```

## Tips

1. **Use `make webui-test`** for the complete one-command start
2. **Use `make status`** frequently to check what's running
3. **Use `make webui-stop`** before restarting to avoid port conflicts
4. **Use `make clean`** if you encounter weird build issues
5. **Use `make reset-db`** if you want to start fresh with empty data

## Integration with IDE

You can integrate these commands with your IDE:

**VS Code** (tasks.json):
```json
{
    "label": "Start Web UI Test Environment",
    "type": "shell",
    "command": "make webui-test"
}
```

**JetBrains GoLand**:
- Create Run Configuration → Makefile → Target: `webui-test`

## Summary

The Makefile provides a simple, consistent interface for all development tasks:

- **One command to start**: `make webui-test`
- **One command to stop**: `make webui-stop`
- **One command to test**: `make test`
- **One command to check**: `make status`

No need to remember multiple commands or switch between directories!