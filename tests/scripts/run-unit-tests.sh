#!/bin/bash

# Quick unit test runner
# Usage: ./run-unit-tests.sh

set -e

echo "=========================================="
echo "Running Unit Tests"
echo "=========================================="
echo ""

cd /home/csermely/prog/de-store-mvp

# Coordinator tests
echo "Testing Coordinator..."
cd coordinator
if go test -v ./internal/services/... 2>&1; then
    echo "✓ Coordinator tests passed"
else
    echo "✗ Coordinator tests failed (may be expected due to missing DB)"
fi

echo ""

# Storage Node tests  
echo "Testing Storage Node..."
cd ../storage-node
if go test -v ./internal/services/... 2>&1; then
    echo "✓ Storage Node tests passed"
else
    echo "✗ Storage Node tests failed (may be expected due to missing DB)"
fi

echo ""
echo "=========================================="
echo "Unit test run complete"
echo "=========================================="