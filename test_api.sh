#!/bin/bash -euo pipefail

# API Test Script for SaveAny-Bot HTTP API

API_URL="http://localhost:8080"
TOKEN="test-token-12345"
HEADERS=(-H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json")

echo "=== Testing SaveAny-Bot HTTP API ==="
echo

# Test 1: Health Check (no auth required)
echo "1. Testing health check endpoint..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "${API_URL}/health")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "200" ]; then
    echo "✓ Health check passed"
    echo "  Response: $BODY"
else
    echo "✗ Health check failed (HTTP $HTTP_STATUS)"
    echo "  Response: $BODY"
fi
echo

# Test 2: Unauthorized request (missing token)
echo "2. Testing unauthorized request..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "${API_URL}/api/v1/tasks")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)

if [ "$HTTP_STATUS" = "401" ]; then
    echo "✓ Correctly rejected unauthorized request"
else
    echo "✗ Unexpected response (HTTP $HTTP_STATUS)"
fi
echo

# Test 3: Invalid token
echo "3. Testing invalid token..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer wrong-token" \
    "${API_URL}/api/v1/tasks")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)

if [ "$HTTP_STATUS" = "401" ]; then
    echo "✓ Correctly rejected invalid token"
else
    echo "✗ Unexpected response (HTTP $HTTP_STATUS)"
fi
echo

# Test 4: List tasks (authorized)
echo "4. Testing list tasks endpoint..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    "${HEADERS[@]}" \
    "${API_URL}/api/v1/tasks")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "200" ]; then
    echo "✓ List tasks successful"
    echo "  Response: $BODY"
else
    echo "✗ List tasks failed (HTTP $HTTP_STATUS)"
    echo "  Response: $BODY"
fi
echo

# Test 5: Create task with missing parameters
echo "5. Testing create task with missing parameters..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -X POST \
    "${HEADERS[@]}" \
    -d '{"telegram_url":""}' \
    "${API_URL}/api/v1/tasks")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)

if [ "$HTTP_STATUS" = "400" ]; then
    echo "✓ Correctly rejected invalid request"
else
    echo "✗ Unexpected response (HTTP $HTTP_STATUS)"
fi
echo

# Test 6: Get non-existent task
echo "6. Testing get non-existent task..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    "${HEADERS[@]}" \
    "${API_URL}/api/v1/tasks/nonexistent")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)

if [ "$HTTP_STATUS" = "404" ]; then
    echo "✓ Correctly returned 404 for non-existent task"
else
    echo "✗ Unexpected response (HTTP $HTTP_STATUS)"
fi
echo

echo "=== API Tests Complete ==="
echo
echo "Note: Full integration testing requires:"
echo "  - Valid Telegram bot token in config"
echo "  - Bot running and connected to Telegram"
echo "  - Valid Telegram message URL to test downloads"
