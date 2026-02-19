#!/bin/bash

# Setup Test User Script for Federated Storage Network
# This script creates a test user with 10,000 credits in the database

set -e

echo "=========================================="
echo "Federated Storage Network - Test User Setup"
echo "=========================================="
echo ""

# Use docker exec to run psql commands inside the container
PSQL="docker exec -e PGPASSWORD=postgres de-store-mvp-postgres-1 psql -U postgres"

# Check if PostgreSQL container is running
if ! docker ps | grep -q de-store-mvp-postgres-1; then
    echo "❌ PostgreSQL container is not running"
    echo ""
    echo "Please start PostgreSQL first:"
    echo "  docker-compose up -d postgres"
    echo ""
    exit 1
fi

echo "✓ PostgreSQL container is running"
echo ""

DB_NAME="${DB_NAME:-coordinator}"

# Check if database exists
echo "Checking database..."
if ! $PSQL -d "$DB_NAME" -c "SELECT 1" > /dev/null 2>&1; then
    echo "❌ Database '$DB_NAME' does not exist"
    echo ""
    echo "Creating database..."
    $PSQL -c "CREATE DATABASE $DB_NAME;"
    echo "✓ Database created"
fi

echo "✓ Database exists"
echo ""

# Check if users table exists
echo "Checking database schema..."
if ! $PSQL -d "$DB_NAME" -c "SELECT 1 FROM users LIMIT 1" > /dev/null 2>&1; then
    echo "⚠️  Users table not found. The coordinator needs to run first to create the schema."
    echo "   Please start the coordinator: make coordinator"
    echo "   Then run this script again."
    echo ""
    exit 1
fi

echo "✓ Database schema is ready"
echo ""

# Test user details
TEST_EMAIL="test@example.com"
TEST_PASSWORD="testpassword123"
TEST_CREDITS=10000

echo "Creating test user..."
echo "  Email: $TEST_EMAIL"
echo "  Password: $TEST_PASSWORD"
echo "  Credits: $TEST_CREDITS"
echo ""

# Generate password hash (using bcrypt)
# For simplicity, we'll insert directly with a pre-computed bcrypt hash
PASSWORD_HASH='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

# Remove existing test user if it exists
$PSQL -d "$DB_NAME" -c "DELETE FROM users WHERE email = '$TEST_EMAIL';"
echo "✓ Removed existing test user (if any)"

# Create fresh test user
echo "Creating new user..."
$PSQL -d "$DB_NAME" -c "
    INSERT INTO users (id, email, password_hash, credits, created_at, updated_at)
    VALUES (
        gen_random_uuid(),
        '$TEST_EMAIL',
        '$PASSWORD_HASH',
        $TEST_CREDITS,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    );
"
echo "✓ User created successfully"

echo ""
echo "=========================================="
echo "✅ Test User Setup Complete!"
echo "=========================================="
echo ""
echo "You can now login with:"
echo "  Email:    test@example.com"
echo "  Password: testpassword123"
echo "  Credits:  $TEST_CREDITS"
echo ""
echo "Web UI: http://localhost:8080/web/"
echo ""

# Verify user was created
echo "Verifying user in database..."
$PSQL -d "$DB_NAME" -c "
    SELECT id, email, credits, created_at 
    FROM users 
    WHERE email = '$TEST_EMAIL';
" | grep -v "^\s*$" || true

echo ""
echo "Done! 🎉"