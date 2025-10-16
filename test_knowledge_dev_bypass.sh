#!/bin/bash

# Test Knowledge Service with development bypass

PROTO_DIR="/Users/liliang/Things/AI/projects/smartticket/proto"
GATEWAY_ADDR="localhost:6533"

# Test tenant and user IDs (actual IDs from database)
TENANT_ID="de57f60e-80a3-4a87-af40-3f99723c6530"
USER_ID="818bcee0-2176-477d-b39b-ed636f73e19b"

echo "Testing Knowledge Service with development bypass..."
echo "Gateway: $GATEWAY_ADDR"
echo "Tenant ID: $TENANT_ID"
echo "User ID: $USER_ID"
echo ""

# Test CreateArticle with development bypass
echo "Testing CreateArticle with development bypass..."
grpcurl -plaintext \
  -proto "${PROTO_DIR}/smartticket/knowledge.proto" \
  -proto "${PROTO_DIR}/smartticket/common.proto" \
  -import-path "${PROTO_DIR}" \
  -d '{
    "title": "Test Article",
    "content": "This is a test article content with sufficient length to be meaningful. It contains enough detail to test the article creation functionality properly.",
    "summary": "Test article summary for development testing",
    "language": "en",
    "visibility": 1,
    "tags": ["test", "article", "knowledge", "development"]
  }' \
  -H "x-dev-bypass: true" \
  -H "x-tenant-id: ${TENANT_ID}" \
  -H "x-user-id: ${USER_ID}" \
  "${GATEWAY_ADDR}" \
  smartticket.v1.KnowledgeService/CreateArticle

echo ""
echo "CreateArticle test completed."
echo ""

# Test ListArticles with development bypass
echo "Testing ListArticles with development bypass..."
grpcurl -plaintext \
  -proto "${PROTO_DIR}/smartticket/knowledge.proto" \
  -proto "${PROTO_DIR}/smartticket/common.proto" \
  -import-path "${PROTO_DIR}" \
  -d '{}' \
  -H "x-dev-bypass: true" \
  -H "x-tenant-id: ${TENANT_ID}" \
  -H "x-user-id: ${USER_ID}" \
  "${GATEWAY_ADDR}" \
  smartticket.v1.KnowledgeService/ListArticles

echo ""
echo "ListArticles test completed."
echo ""

echo "All Knowledge service tests completed successfully!"