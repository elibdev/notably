#!/bin/bash

# Backend startup script with environment variables
# Usage: ./scripts/start-backend.sh

set -e

# Change to backend directory
cd "$(dirname "$0")/../../backend"

# Set environment variables for DynamoDB
export DYNAMODB_TABLE_NAME=NotablyTest
export DYNAMODB_ENDPOINT_URL=http://localhost:8000

# Additional environment variables that might be needed
export GO_ENV=development
export PORT=8080

echo "Starting backend server with DynamoDB configuration..."
echo "DYNAMODB_TABLE_NAME: $DYNAMODB_TABLE_NAME"
echo "DYNAMODB_ENDPOINT_URL: $DYNAMODB_ENDPOINT_URL"
echo "Server will be available at: http://localhost:$PORT"
echo ""

# Start the backend server
exec go run cmd/server/main.go