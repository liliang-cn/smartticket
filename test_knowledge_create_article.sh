#!/bin/bash

# Test CreateArticle interface using grpcurl with proto files

PROTO_DIR="/Users/liliang/Things/AI/projects/smartticket/proto"
GATEWAY_ADDR="localhost:6533"

# Test tenant and user IDs
TENANT_ID="123e4567-e89b-12d3-a456-426614174000"
USER_ID="123e4567-e89b-12d3-a456-426614174001"

echo "Testing CreateArticle interface..."
echo "Gateway: $GATEWAY_ADDR"
echo "Tenant ID: $TENANT_ID"
echo "User ID: $USER_ID"
echo ""

# Test CreateArticle
grpcurl -plaintext \
  -proto "${PROTO_DIR}/smartticket/knowledge.proto" \
  -proto "${PROTO_DIR}/smartticket/common.proto" \
  -import-path "${PROTO_DIR}" \
  -d '{
    "title": "Test Article",
    "content": "This is a test article content with sufficient length to be meaningful.",
    "summary": "Test article summary",
    "language": "en",
    "visibility": 1,
    "tags": ["test", "article", "knowledge"]
  }' \
  -H "x-tenant-id: ${TENANT_ID}" \
  -H "x-user-id: ${USER_ID}" \
  "${GATEWAY_ADDR}" \
  smartticket.v1.KnowledgeService/CreateArticle

echo ""
echo "CreateArticle test completed."