# Federated Storage Network MVP

A distributed storage system with encrypted file storage, proof-of-storage verification, and credit-based economics.

## Architecture

### Components

1. **Coordinator** - Central API server
   - User authentication (JWT-based)
   - File management
   - Credit system
   - Node management
   - PostgreSQL database

2. **Storage Node** - Distributed storage participant
   - Stores encrypted file chunks
   - Responds to proof challenges
   - Earns credits for storage
   - SQLite database

## Quick Start

### 🚀 Option 1: Using Makefile (Easiest - Recommended for Testing)

The simplest way to start the Web UI test environment:

```bash
# Start everything with one command
make webui-test

# This starts:
# - PostgreSQL database (Docker)
# - Coordinator API (port 8080)
# - Storage Node
# - Test user with 10,000 credits
#
# Then open: http://localhost:8080/web/
```

**Credentials:**
- Email: `test@example.com`
- Password: `testpassword123`
- Credits: 10,000

Other useful commands:
```bash
make help              # Show all available commands
make status            # Check what's running
make webui-stop        # Stop all services
make test              # Run all tests
```

### Option 2: Using Docker Compose (Full System)

Start the complete system with all services:

```bash
# Build and start all services
docker-compose up --build

# This starts:
# - PostgreSQL database
# - Coordinator API (port 8080)
# - 3 Storage Nodes
# - Web UI available at http://localhost:8080/web/
```

### Option 3: Manual Setup

#### 1. Start PostgreSQL

```bash
# Using Docker
docker-compose up -d postgres

# Or use your local PostgreSQL
```

#### 2. Setup Test User

```bash
# Create test user with 10,000 credits
./setup-test-user.sh

# Credentials:
# Email: test@example.com
# Password: testpassword123
```

#### 3. Start Coordinator

```bash
cd coordinator

# Install dependencies
go mod tidy

# Start the coordinator
go run cmd/api/main.go

# The Web UI will be available at http://localhost:8080/web/
```

#### 4. Start Storage Node

```bash
cd storage-node

# Install dependencies
go mod tidy

# Initialize the storage node
go run cmd/storage-node/main.go init --name "My Node" --coordinator-url http://localhost:8080

# Start the storage node
go run cmd/storage-node/main.go start
```

### Option 3: Quick Web UI Test

If you just want to test the Web UI:

```bash
# 1. Start infrastructure
docker-compose up -d postgres

# 2. Setup test user
./setup-test-user.sh

# 3. Start coordinator
cd coordinator && go run cmd/api/main.go &

# 4. Start one storage node
cd ../storage-node
go run cmd/storage-node/main.go init --name "Test Node" --coordinator-url http://localhost:8080
go run cmd/storage-node/main.go start &

# 5. Open Web UI
open http://localhost:8080/web/
# or
xdg-open http://localhost:8080/web/
```

## Web UI

A simple web interface is available for testing uploads and downloads:

**URL**: `http://localhost:8080/web/` or `http://localhost:8080/`

### Features
- User authentication (login/register)
- Drag & drop file upload
- File listing and management
- Download files
- Credit balance display
- Real-time upload progress

### Test User (Pre-configured)
After running `./setup-test-user.sh`:
- **Email**: `test@example.com`
- **Password**: `testpassword123`
- **Credits**: 10,000

The Web UI includes step-by-step instructions right on the page!

## API Endpoints

### Web UI
- `GET /` - Web UI (redirects to /web/)
- `GET /web/*` - Static web UI files

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get JWT token
- `GET /api/v1/auth/profile` - Get user profile
- `POST /api/v1/auth/credits/purchase` - Purchase credits (mock)

### Files
- `GET /api/v1/files` - List user's files
- `GET /api/v1/files/:id/download` - Download file
- `DELETE /api/v1/files/:id` - Delete file
- `POST /api/v1/files/upload/initiate` - Start upload
- `POST /api/v1/files/upload/:id/chunk` - Upload chunk
- `POST /api/v1/files/upload/:id/complete` - Complete upload

### Storage Nodes
- `POST /api/v1/nodes/register` - Register storage node
- `GET /api/v1/nodes` - List active nodes
- `POST /api/v1/nodes/heartbeat` - Send heartbeat
- `GET /api/v1/nodes/balance` - Get node earnings

## Storage Node CLI

```bash
# Initialize a new storage node
storage-node init --name "Node Name" --coordinator-url http://localhost:8080

# Start the storage node
storage-node start

# List stored chunks
storage-node chunks list

# Drain node (stop accepting new chunks)
storage-node drain
```

## Configuration

### Coordinator (`coordinator/config.toml`)

```toml
[server]
host = "0.0.0.0"
port = 8080

[database]
host = "localhost"
port = 5432
user = "postgres"
password = "postgres"
database = "coordinator"

[storage]
chunk_size_bytes = 262144  # 256KB
default_replicas = 3
storage_credit_per_gb_month = 100
```

### Storage Node (`storage-node/config.toml`)

```toml
[node]
name = "My Storage Node"
data_dir = "./data"
max_storage_gb = 100

[coordinator]
url = "http://localhost:8080"

[storage]
chunk_dir = "./data/chunks"
```

## Features

- **End-to-End Encryption** - Files are encrypted before being distributed
- **Chunk Distribution** - Files are split into 256KB chunks and distributed across 3+ nodes
- **Proof of Storage** - Nodes must prove they're storing data through challenges
- **Credit System** - Users pay credits for storage, nodes earn credits
- **Heartbeat Monitoring** - Automatic node health monitoring
- **Re-replication** - Automatic recovery when nodes fail

## Development

### Project Structure

```
coordinator/
├── cmd/api/              # Main entry point
├── internal/
│   ├── config/          # Configuration
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # JWT auth, etc.
│   ├── models/          # Database models
│   ├── p2p/            # libp2p networking
│   ├── services/       # Business logic
│   └── storage/        # PostgreSQL
└── migrations/         # Database migrations

storage-node/
├── cmd/storage-node/    # CLI entry point
├── internal/
│   ├── config/         # Configuration
│   ├── models/         # Database models
│   ├── p2p/           # libp2p networking
│   ├── services/      # Business logic
│   └── storage/       # SQLite
└── migrations/        # Database migrations
```

### Testing

```bash
# Coordinator
cd coordinator
go test ./...

# Storage Node
cd storage-node
go test ./...
```

## Roadmap

### MVP (Completed)
- [x] Basic upload/download flow
- [x] Chunk encryption and distribution
- [x] Proof-of-storage challenges
- [x] Credit system
- [x] Node registration and heartbeat

### Post-MVP
- [ ] Real payment processing (Stripe)
- [ ] Web UI for file management
- [ ] Node operator dashboard
- [ ] Advanced erasure coding
- [ ] Bandwidth optimization
- [ ] Mobile client
- [ ] Federation (multiple coordinators)

## License

MIT License