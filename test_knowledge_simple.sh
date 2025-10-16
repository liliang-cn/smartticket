#!/bin/bash

echo "🧪 Simple Knowledge Service Test"
echo "================================"

# Start gateway in background
echo "🚀 Starting gateway service..."
RUST_LOG=info cargo run --bin gateway > gateway_test.log 2>&1 &
GATEWAY_PID=$!
sleep 10

# Check if gateway is running
if curl -s http://localhost:7218/health > /dev/null 2>&1; then
    echo "✅ Gateway started successfully"
else
    echo "❌ Gateway failed to start"
    cat gateway_test.log
    exit 1
fi

# Get auth token
echo "🔐 Getting authentication token..."
AUTH_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/user.proto \
  -d '{"email": "admin@test.smartticket.com", "password": "admin123", "tenantDomain": "test.smartticket.com"}' \
  localhost:6533 smartticket.v1.AuthService.Login 2>/dev/null)

if [ $? -ne 0 ]; then
    echo "❌ Authentication failed"
    cat gateway_test.log
    exit 1
fi

ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.accessToken')
USER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.id')
TENANT_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.tenantId')

echo "✅ Authentication successful"

# Test knowledge category creation
echo "📚 Testing knowledge category creation..."
CATEGORY_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
  -d '{"name": "Test Category", "description": "Test category for E2E"}' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.KnowledgeService.CreateCategory 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "✅ Category creation successful"
    CATEGORY_ID=$(echo "$CATEGORY_RESPONSE" | jq -r '.id')
    echo "Category ID: $CATEGORY_ID"
else
    echo "❌ Category creation failed"
fi

# Test knowledge article creation
echo "📄 Testing knowledge article creation..."
ARTICLE_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
  -d "{\"title\": \"Test Article\", \"content\": \"# Test Article\\n\\nThis is a test article.\", \"summary\": \"Test article summary\", \"categoryId\": \"$CATEGORY_ID\"}" \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.KnowledgeService.CreateArticle 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "✅ Article creation successful"
    ARTICLE_ID=$(echo "$ARTICLE_RESPONSE" | jq -r '.id')
    echo "Article ID: $ARTICLE_ID"
else
    echo "❌ Article creation failed"
fi

# Test article listing
echo "📋 Testing article listing..."
LIST_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
  -d '{"pagination": {"pageSize": 10}}' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.KnowledgeService.ListArticles 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "✅ Article listing successful"
    ARTICLE_COUNT=$(echo "$LIST_RESPONSE" | jq '.articles | length')
    echo "Found $ARTICLE_COUNT articles"
else
    echo "❌ Article listing failed"
fi

# Test article search
echo "🔍 Testing article search..."
SEARCH_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
  -d '{"query": "test", "pagination": {"pageSize": 10}}' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  -H "x-user-id: $USER_ID" \
  localhost:6533 smartticket.v1.KnowledgeService.SearchArticles 2>/dev/null)

if [ $? -eq 0 ]; then
    echo "✅ Article search successful"
    SEARCH_COUNT=$(echo "$SEARCH_RESPONSE" | jq '.articles | length')
    echo "Found $SEARCH_COUNT search results"
else
    echo "❌ Article search failed"
fi

# Cleanup
echo "🧹 Cleaning up..."
kill $GATEWAY_PID 2>/dev/null

echo ""
echo "================================"
echo "🎉 Knowledge Service Test Complete"
echo "================================"