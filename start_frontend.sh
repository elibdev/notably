#!/bin/bash

# Start Frontend with Proxy Testing
# This script ensures the frontend starts correctly with proper backend proxy

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting Notably Frontend Setup${NC}"

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -i :$port &>/dev/null; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Function to wait for a service to be ready
wait_for_service() {
    local url=$1
    local timeout=30
    local count=0
    
    echo -e "${YELLOW}⏳ Waiting for service at $url...${NC}"
    
    while [ $count -lt $timeout ]; do
        if curl -f "$url" &>/dev/null; then
            echo -e "${GREEN}✅ Service is ready!${NC}"
            return 0
        fi
        sleep 1
        count=$((count + 1))
        echo -n "."
    done
    
    echo -e "\n${RED}❌ Service at $url is not responding after ${timeout}s${NC}"
    return 1
}

# Step 1: Kill any existing frontend processes
echo -e "${YELLOW}🧹 Cleaning up existing processes...${NC}"
pkill -f "vite" 2>/dev/null || true
pkill -f "npm.*dev" 2>/dev/null || true
sleep 2

# Step 2: Check if backend is running
echo -e "${YELLOW}🔍 Checking backend status...${NC}"
if ! check_port 8080; then
    echo -e "${RED}❌ Backend is not running on port 8080${NC}"
    echo -e "${YELLOW}Please start the backend first:${NC}"
    echo -e "  cd backend && docker compose up -d"
    exit 1
fi

# Test backend API
if ! curl -f http://localhost:8080/health &>/dev/null; then
    echo -e "${RED}❌ Backend health check failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Backend is running and healthy${NC}"

# Step 3: Check if frontend port is available
if check_port 3000; then
    echo -e "${RED}❌ Port 3000 is already in use${NC}"
    echo -e "${YELLOW}Killing process on port 3000...${NC}"
    lsof -ti:3000 | xargs kill -9 2>/dev/null || true
    sleep 2
fi

# Step 4: Start frontend
echo -e "${YELLOW}🎯 Starting frontend on port 3000...${NC}"
cd frontend

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}📦 Installing dependencies...${NC}"
    npm install
fi

# Start Vite in background
echo -e "${YELLOW}🚀 Starting Vite development server...${NC}"
npm run dev &
VITE_PID=$!

# Step 5: Wait for frontend to be ready
if wait_for_service "http://localhost:3000"; then
    echo -e "${GREEN}✅ Frontend is running on http://localhost:3000${NC}"
else
    echo -e "${RED}❌ Frontend failed to start${NC}"
    kill $VITE_PID 2>/dev/null || true
    exit 1
fi

# Step 6: Test proxy functionality
echo -e "${YELLOW}🔗 Testing API proxy...${NC}"
sleep 2

# Test registration endpoint through proxy
TEST_USER_ID="proxy_test_$(date +%s)"
TEST_EMAIL="proxytest@example.com"
TEST_PASSWORD="proxytestpass123"

echo -e "${YELLOW}🧪 Testing registration through proxy...${NC}"
RESPONSE=$(curl -s -w "%{http_code}" -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"user_id\": \"$TEST_USER_ID\", \"email\": \"$TEST_EMAIL\", \"password\": \"$TEST_PASSWORD\"}" \
  -o /tmp/proxy_test_response.json)

if [ "$RESPONSE" = "201" ]; then
    echo -e "${GREEN}✅ Proxy is working! Registration successful${NC}"
    
    # Extract token for login test
    TOKEN=$(cat /tmp/proxy_test_response.json | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$TOKEN" ]; then
        echo -e "${YELLOW}🔑 Testing login through proxy...${NC}"
        LOGIN_RESPONSE=$(curl -s -w "%{http_code}" -X POST http://localhost:3000/api/v1/auth/login \
          -H "Content-Type: application/json" \
          -d "{\"user_id\": \"$TEST_USER_ID\", \"password\": \"$TEST_PASSWORD\"}" \
          -o /tmp/proxy_login_response.json)
        
        if [ "$LOGIN_RESPONSE" = "200" ]; then
            echo -e "${GREEN}✅ Login through proxy successful!${NC}"
        else
            echo -e "${YELLOW}⚠️  Login test failed (code: $LOGIN_RESPONSE)${NC}"
        fi
    fi
else
    echo -e "${RED}❌ Proxy test failed (HTTP code: $RESPONSE)${NC}"
    echo -e "${YELLOW}Response body:${NC}"
    cat /tmp/proxy_test_response.json
    echo
    echo -e "${YELLOW}💡 Checking proxy configuration...${NC}"
    
    # Check if vite.config.js has correct proxy settings
    if grep -q "localhost:8080" frontend/vite.config.js; then
        echo -e "${GREEN}✅ Proxy configuration looks correct${NC}"
    else
        echo -e "${RED}❌ Proxy configuration may be incorrect${NC}"
    fi
fi

# Cleanup test files
rm -f /tmp/proxy_test_response.json /tmp/proxy_login_response.json

# Step 7: Display summary
echo
echo -e "${BLUE}📋 Setup Summary:${NC}"
echo -e "  🌐 Frontend: ${GREEN}http://localhost:3000${NC}"
echo -e "  🔧 Backend:  ${GREEN}http://localhost:8080${NC}"
echo -e "  📊 API Docs: ${GREEN}http://localhost:8080/docs${NC}"
echo -e "  💾 DynamoDB Admin: ${GREEN}http://localhost:8001${NC}"
echo
echo -e "${GREEN}🎉 Frontend is ready! You can now test registration in the UI.${NC}"
echo
echo -e "${YELLOW}💡 To stop the frontend:${NC}"
echo -e "  kill $VITE_PID"
echo
echo -e "${YELLOW}💡 To view frontend logs:${NC}"
echo -e "  tail -f frontend/vite.log"

# Keep the script running and show live logs
echo -e "${BLUE}📝 Frontend logs (Ctrl+C to exit):${NC}"
echo "---"

# Follow the Vite process
wait $VITE_PID