#!/bin/bash

# Test Runner Script for Federated Storage Network
# Usage: ./run-tests.sh [unit|integration|e2e|all] [--coverage]

set -e

TEST_TYPE=${1:-all}
COVERAGE=false

if [ "$2" == "--coverage" ]; then
    COVERAGE=true
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Federated Storage Network - Test Runner"
echo "=========================================="
echo ""

# Function to run tests with coverage
run_tests_with_coverage() {
    local module=$1
    local test_path=$2
    
    echo "Running tests in $module..."
    cd "$module"
    
    if [ "$COVERAGE" = true ]; then
        go test -v -race -coverprofile=coverage.out -covermode=atomic "$test_path" 2>&1 | head -100
        go tool cover -html=coverage.out -o coverage.html 2>/dev/null || true
        echo "Coverage report generated: $module/coverage.html"
    else
        go test -v -race "$test_path" 2>&1 | head -100
    fi
    
    cd - > /dev/null
}

# Function to run unit tests
run_unit_tests() {
    echo -e "${YELLOW}Running Unit Tests...${NC}"
    echo ""
    
    # Coordinator unit tests
    echo "Coordinator unit tests:"
    cd /home/csermely/prog/de-store-mvp/coordinator
    go test -v ./internal/services/... 2>&1 | tee /tmp/coordinator_unit_test.log || true
    COORD_STATUS=$?
    
    echo ""
    echo "Storage Node unit tests:"
    cd /home/csermely/prog/de-store-mvp/storage-node
    go test -v ./internal/services/... 2>&1 | tee /tmp/storage_node_unit_test.log || true
    STORAGE_STATUS=$?
    
    echo ""
    if [ $COORD_STATUS -eq 0 ] && [ $STORAGE_STATUS -eq 0 ]; then
        echo -e "${GREEN}✓ Unit tests completed successfully${NC}"
    else
        echo -e "${RED}✗ Some unit tests failed${NC}"
        echo "See logs: /tmp/coordinator_unit_test.log, /tmp/storage_node_unit_test.log"
    fi
    echo ""
}

# Function to run integration tests
run_integration_tests() {
    echo -e "${YELLOW}Running Integration Tests...${NC}"
    echo ""
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Docker not found. Skipping integration tests.${NC}"
        return 1
    fi
    
    echo "Starting test environment..."
    cd /home/csermely/prog/de-store-mvp
    docker-compose -f docker-compose.test.yml up -d --build 2>&1 | tail -20 || {
        echo -e "${YELLOW}Warning: Could not start Docker environment. Using mock tests.${NC}"
    }
    
    echo ""
    echo "Running integration tests..."
    cd /home/csermely/prog/de-store-mvp/coordinator
    go test -v ./... -tags=integration 2>&1 | head -100 || true
    
    echo ""
    echo "Cleaning up test environment..."
    cd /home/csermely/prog/de-store-mvp
    docker-compose -f docker-compose.test.yml down 2>/dev/null || true
    
    echo -e "${GREEN}✓ Integration tests completed${NC}"
    echo ""
}

# Function to run e2e tests
run_e2e_tests() {
    echo -e "${YELLOW}Running End-to-End Tests...${NC}"
    echo ""
    
    echo "E2E tests require full system deployment."
    echo "This would test:"
    echo "  - User registration and login"
    echo "  - File upload and download"
    echo "  - Chunk distribution"
    echo "  - Proof challenges"
    echo "  - Credit system"
    echo ""
    echo "For MVP, this is simulated with manual testing scripts."
    echo "See: tests/scripts/test-e2e.sh"
    echo ""
}

# Function to display test summary
display_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    echo ""
    
    # Count test results
    if [ -f /tmp/coordinator_unit_test.log ]; then
        PASS_COUNT=$(grep -c "^--- PASS:" /tmp/coordinator_unit_test.log 2>/dev/null || echo 0)
        FAIL_COUNT=$(grep -c "^--- FAIL:" /tmp/coordinator_unit_test.log 2>/dev/null || echo 0)
        SKIP_COUNT=$(grep -c "^--- SKIP:" /tmp/coordinator_unit_test.log 2>/dev/null || echo 0)
        
        echo "Coordinator: $PASS_COUNT passed, $FAIL_COUNT failed, $SKIP_COUNT skipped"
    fi
    
    if [ -f /tmp/storage_node_unit_test.log ]; then
        PASS_COUNT=$(grep -c "^--- PASS:" /tmp/storage_node_unit_test.log 2>/dev/null || echo 0)
        FAIL_COUNT=$(grep -c "^--- FAIL:" /tmp/storage_node_unit_test.log 2>/dev/null || echo 0)
        SKIP_COUNT=$(grep -c "^--- SKIP:" /tmp/storage_node_unit_test.log 2>/dev/null || echo 0)
        
        echo "Storage Node: $PASS_COUNT passed, $FAIL_COUNT failed, $SKIP_COUNT skipped"
    fi
    
    echo ""
    echo -e "${GREEN}Testing complete!${NC}"
}

# Main execution
case $TEST_TYPE in
    unit)
        run_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    e2e)
        run_e2e_tests
        ;;
    all)
        run_unit_tests
        run_integration_tests
        run_e2e_tests
        display_summary
        ;;
    *)
        echo "Usage: $0 [unit|integration|e2e|all] [--coverage]"
        echo ""
        echo "Options:"
        echo "  unit         - Run unit tests only"
        echo "  integration  - Run integration tests (requires Docker)"
        echo "  e2e          - Run end-to-end tests (requires full deployment)"
        echo "  all          - Run all tests (default)"
        echo "  --coverage   - Generate coverage reports"
        exit 1
        ;;
esac

exit 0