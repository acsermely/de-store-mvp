# Federated Storage Network - Makefile
# Provides convenient commands for development and testing

.PHONY: help all build test clean docker-up docker-down webui-test webui-5nodes setup-test-user

# Default target
help:
	@echo "Federated Storage Network - Available Commands"
	@echo "=============================================="
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make install          - Install dependencies for all components"
	@echo "  make setup-test-user  - Create test user in database"
	@echo ""
	@echo "Development (Individual Components):"
	@echo "  make coordinator      - Start coordinator only"
	@echo "  make storage-node     - Start storage node (after init)"
	@echo "  make init-node        - Initialize storage node"
	@echo ""
	@echo "Web UI Testing:"
	@echo "  make webui-test       - Start full Web UI test environment (1 node)"
	@echo "  make webui-5nodes     - Start Web UI test with 5 storage nodes"
	@echo "  make webui-quick      - Quick start (assumes DB already running)"
	@echo "  make webui-stop       - Stop all Web UI test components"
	@echo ""
	@echo "Docker (Full System):"
	@echo "  make docker-up        - Start all services with Docker Compose"
	@echo "  make docker-down      - Stop all Docker services"
	@echo "  make docker-build     - Build Docker images"
	@echo ""
	@echo "Testing:"
	@echo "  make test             - Run all unit tests"
	@echo "  make test-coordinator - Run coordinator tests"
	@echo "  make test-storage     - Run storage node tests"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean            - Clean build artifacts and data"
	@echo "  make clean-all       - Full cleanup (data, configs, Docker)"
	@echo "  make reset-db         - Reset database (WARNING: deletes all data)"
	@echo "  make logs             - Show logs from running services"
	@echo "  make status           - Check status of all services"
	@echo ""
	@echo "Quick Start:"
	@echo "  make webui-test       - Start everything for Web UI testing"
	@echo "  Then open: http://localhost:8080/web/"
	@echo ""

# ============================================
# Setup & Installation
# ============================================

install:
	@echo "Installing dependencies..."
	cd coordinator && go mod tidy
	cd storage-node && go mod tidy
	@echo "✓ Dependencies installed"

setup-test-user:
	@echo "Setting up test user..."
	./setup-test-user.sh

# ============================================
# Individual Component Development
# ============================================

coordinator:
	@echo "Starting coordinator..."
	cd coordinator && MIGRATIONS_PATH=./migrations go run cmd/api/main.go

init-node:
	@echo "Initializing storage node..."
	cd storage-node && go run cmd/storage-node/main.go init --name "Test Node" --coordinator-url http://localhost:8080

storage-node:
	@echo "Starting storage node..."
	cd storage-node && go run cmd/storage-node/main.go start

# ============================================
# Web UI Test Environment
# ============================================

webui-test: docker-up-db
	@echo ""
	@echo "=========================================="
	@echo "Starting Web UI Test Environment"
	@echo "=========================================="
	@echo ""
	@echo "Step 1: ✓ PostgreSQL is running"
	@echo ""
	@echo "Step 2: Starting Coordinator (to run migrations)..."
	@cd coordinator && MIGRATIONS_PATH=./migrations nohup go run cmd/api/main.go > /tmp/coordinator.log 2>&1 &
	@echo "Waiting for coordinator to initialize (running migrations)..."
	@sleep 10
	@echo "✓ Coordinator started on http://localhost:8080"
	@echo ""
	@echo "Step 3: Creating test user..."
	@./setup-test-user.sh || echo "⚠️  Test user setup skipped (may already exist)"
	@echo ""
	@echo "Step 4: Initializing Storage Node..."
	@cd storage-node && go run cmd/storage-node/main.go init --name "Test Node" --coordinator-url http://localhost:8080 2>/dev/null || true
	@echo "Step 5: Starting Storage Node..."
	@cd storage-node && nohup go run cmd/storage-node/main.go start > /tmp/storage-node.log 2>&1 &
	@sleep 2
	@echo "✓ Storage Node started"
	@echo ""
	@echo "=========================================="
	@echo "🎉 Web UI Test Environment Ready!"
	@echo "=========================================="
	@echo ""
	@echo "📱 Web UI: http://localhost:8080/web/"
	@echo "🔑 Test User: test@example.com / testpassword123"
	@echo "💰 Credits: 10,000"
	@echo ""
	@echo "Useful commands:"
	@echo "  make logs        - View all logs"
	@echo "  make status      - Check service status"
	@echo "  make webui-stop  - Stop all services"
	@echo ""
	@echo "Press Ctrl+C to stop..."
	@wait

webui-quick:
	@echo "Starting Web UI (quick mode - assumes DB running)..."
	@cd coordinator && MIGRATIONS_PATH=./migrations nohup go run cmd/api/main.go > /tmp/coordinator.log 2>&1 &
	@sleep 3
	@cd storage-node && nohup go run cmd/storage-node/main.go start > /tmp/storage-node.log 2>&1 &
	@echo "✓ Services started"
	@echo "Log files: /tmp/coordinator.log, /tmp/storage-node.log"
	@echo "🌐 Open: http://localhost:8080/web/"
	@echo "Press Ctrl+C to stop..."
	@wait

webui-5nodes: docker-up-db
	@echo ""
	@echo "=========================================="
	@echo "Starting Web UI Test with 5 Storage Nodes"
	@echo "=========================================="
	@echo ""
	@echo "Step 1: ✓ PostgreSQL is running"
	@echo ""
	@echo "Step 2: Starting Coordinator..."
	@cd coordinator && MIGRATIONS_PATH=./migrations nohup go run cmd/api/main.go > /tmp/coordinator.log 2>&1 &
	@echo "Waiting for coordinator to initialize..."
	@sleep 10
	@echo "✓ Coordinator started on http://localhost:8080"
	@echo ""
	@echo "Step 3: Creating test user..."
	@./setup-test-user.sh || echo "⚠️  Test user setup skipped (may already exist)"
	@echo ""
	@echo "Step 4: Initializing 5 Storage Nodes..."
	@cd storage-node && \
		mkdir -p data/node1 data/node2 data/node3 data/node4 data/node5
	@echo "  Initializing Node 1..."; \
		cd storage-node && cp config.example.toml config.toml && go run cmd/storage-node/main.go init --name "Node 1" --coordinator-url http://localhost:8080 2>/dev/null || true; \
		mv storage-node/config.toml storage-node/config.node1_local.toml 2>/dev/null || true
	@echo "  Initializing Node 2..."; \
		cd storage-node && cp config.example.toml config.toml && go run cmd/storage-node/main.go init --name "Node 2" --coordinator-url http://localhost:8080 2>/dev/null || true; \
		mv storage-node/config.toml storage-node/config.node2_local.toml 2>/dev/null || true
	@echo "  Initializing Node 3..."; \
		cd storage-node && cp config.example.toml config.toml && go run cmd/storage-node/main.go init --name "Node 3" --coordinator-url http://localhost:8080 2>/dev/null || true; \
		mv storage-node/config.toml storage-node/config.node3_local.toml 2>/dev/null || true
	@echo "  Initializing Node 4..."; \
		cd storage-node && cp config.example.toml config.toml && go run cmd/storage-node/main.go init --name "Node 4" --coordinator-url http://localhost:8080 2>/dev/null || true; \
		mv storage-node/config.toml storage-node/config.node4_local.toml 2>/dev/null || true
	@echo "  Initializing Node 5..."; \
		cd storage-node && cp config.example.toml config.toml && go run cmd/storage-node/main.go init --name "Node 5" --coordinator-url http://localhost:8080 2>/dev/null || true; \
		mv storage-node/config.toml storage-node/config.node5_local.toml 2>/dev/null || true
	@echo ""
	@echo "Step 5: Starting Storage Nodes..."
	@cd storage-node && \
		nohup go run cmd/storage-node/main.go --config config.node1_local.toml start > /tmp/storage-node1.log 2>&1 & \
		nohup go run cmd/storage-node/main.go --config config.node2_local.toml start > /tmp/storage-node2.log 2>&1 & \
		nohup go run cmd/storage-node/main.go --config config.node3_local.toml start > /tmp/storage-node3.log 2>&1 & \
		nohup go run cmd/storage-node/main.go --config config.node4_local.toml start > /tmp/storage-node4.log 2>&1 & \
		nohup go run cmd/storage-node/main.go --config config.node5_local.toml start > /tmp/storage-node5.log 2>&1 &
	@sleep 5
	@echo "✓ 5 Storage Nodes started"
	@echo ""
	@echo "=========================================="
	@echo "🎉 5-Node Test Environment Ready!"
	@echo "=========================================="
	@echo ""
	@echo "📱 Web UI: http://localhost:8080/web/"
	@echo "🔑 Test User: test@example.com / testpassword123"
	@echo "💰 Credits: 10,000"
	@echo "🗄️  Storage Nodes: 5"
	@echo ""
	@echo "Useful commands:"
	@echo "  make logs        - View all logs"
	@echo "  make status      - Check service status"
	@echo "  make webui-stop  - Stop all services"
	@echo ""
	@echo "Log files:"
	@echo "  /tmp/coordinator.log"
	@echo "  /tmp/storage-node1.log through /tmp/storage-node5.log"
	@echo ""
	@wait

webui-stop:
	@echo "Stopping all services..."
	@-pkill -f "coordinator.*main.go" 2>/dev/null || true
	@-pkill -f "storage-node.*main.go" 2>/dev/null || true
	@echo "✓ Services stopped"
	@echo "Stopping Docker services..."
	@docker-compose down 2>/dev/null || true

# ============================================
# Docker Operations
# ============================================

docker-up:
	@echo "Starting all services with Docker Compose..."
	docker-compose up -d
	@echo ""
	@echo "✓ Services started"
	@echo "📱 Web UI: http://localhost:8080/web/"
	@echo ""

docker-up-db:
	@echo "Starting PostgreSQL only..."
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker exec de-store-mvp-postgres-1 pg_isready -U postgres > /dev/null 2>&1; then \
			echo "✓ PostgreSQL is ready"; \
			exit 0; \
		fi; \
		echo "  Attempt $$i/10..."; \
		sleep 2; \
	done; \
	echo "❌ PostgreSQL failed to start"; \
	exit 1

docker-down:
	@echo "Stopping all Docker services..."
	docker-compose down
	@echo "✓ Services stopped"

docker-build:
	@echo "Building Docker images..."
	docker-compose build
	@echo "✓ Images built"

docker-logs:
	docker-compose logs -f

# ============================================
# Testing
# ============================================

test: test-coordinator test-storage
	@echo ""
	@echo "=========================================="
	@echo "✅ All Tests Passed!"
	@echo "=========================================="

test-coordinator:
	@echo "Running Coordinator tests..."
	cd coordinator && go test -v ./internal/services/... 2>&1 | tail -30

test-storage:
	@echo "Running Storage Node tests..."
	cd storage-node && go test -v ./internal/services/... 2>&1 | tail -30

test-integration:
	@echo "Running integration tests..."
	./tests/scripts/run-integration-tests.sh

# ============================================
# Utilities
# ============================================

clean:
	@echo "Cleaning build artifacts..."
	@cd coordinator && rm -f coordinator coverage.out coverage.html
	@cd storage-node && rm -f storage-node coverage.out coverage.html
	@rm -rf coordinator/data storage-node/data
	@echo "✓ Clean complete"

clean-all: clean
	@echo "Cleaning everything (data, configs, Docker)..."
	@rm -rf storage-node/data/node* storage-node/config.node*_local.toml
	@docker-compose down -v 2>/dev/null || true
	@docker system prune -f 2>/dev/null || true
	@echo "✓ Full cleanup complete"

reset-db:
	@echo "⚠️  WARNING: This will delete all data in the database!"
	@read -p "Are you sure? (yes/no): " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		docker-compose down -v; \
		docker-compose up -d postgres; \
		sleep 3; \
		make setup-test-user; \
		echo "✓ Database reset complete"; \
	else \
		echo "Cancelled"; \
	fi

logs:
	@echo "Showing logs (Ctrl+C to exit)..."
	@if [ -f /tmp/coordinator.log ] && [ -f /tmp/storage-node.log ]; then \
		tail -f /tmp/coordinator.log /tmp/storage-node.log; \
	elif [ -f /tmp/coordinator.log ]; then \
		tail -f /tmp/coordinator.log; \
	elif [ -f /tmp/storage-node.log ]; then \
		tail -f /tmp/storage-node.log; \
	else \
		echo "No log files found. Services may be running in foreground."; \
	fi

status:
	@echo "=========================================="
	@echo "Service Status"
	@echo "=========================================="
	@echo ""
	@echo "PostgreSQL:"
	@docker-compose ps postgres 2>/dev/null || echo "  Not running (Docker)"
	@pg_isready -h localhost -p 5432 2>/dev/null && echo "  ✓ Running on localhost:5432" || echo "  ✗ Not running on localhost:5432"
	@echo ""
	@echo "Coordinator:"
	@pgrep -f "coordinator.*main.go" > /dev/null && echo "  ✓ Running" || echo "  ✗ Not running"
	@echo ""
	@echo "Storage Nodes:"
	@pgrep -f "storage-node.*main.go" > /dev/null && echo "  ✓ Running" || echo "  ✗ Not running"
	@echo ""
	@echo "Web UI: http://localhost:8080/web/"
	@echo ""

build:
	@echo "Building binaries..."
	cd coordinator && go build -o coordinator cmd/api/main.go
	cd storage-node && go build -o storage-node cmd/storage-node/main.go
	@echo "✓ Binaries built"
	@echo "  coordinator/coordinator"
	@echo "  storage-node/storage-node"

# ============================================
# Development Helpers
# ============================================

fmt:
	@echo "Formatting code..."
	cd coordinator && go fmt ./...
	cd storage-node && go fmt ./...
	@echo "✓ Code formatted"

vet:
	@echo "Running go vet..."
	cd coordinator && go vet ./...
	cd storage-node && go vet ./...
	@echo "✓ Vet complete"

lint: fmt vet
	@echo "✓ Linting complete"

# Run everything for development
dev: docker-up-db setup-test-user
	@echo ""
	@echo "Development environment ready!"
	@echo "Run 'make coordinator' and 'make storage-node' in separate terminals"
	@echo "Or run 'make webui-test' to start everything automatically"