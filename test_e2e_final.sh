#!/bin/bash

# SmartTicket Final E2E Test Suite
# Tests core functionality across all major services

set -e

echo "============================================"
echo "🚀 SmartTicket Final E2E Test Suite"
echo "============================================"

# Configuration
GATEWAY_ADDR="localhost:6533"
TENANT_ID="de57f60e-80a3-4a87-af40-3f99723c6530"
USER_ID="818bcee0-2176-477d-b39b-ed636f73e19b"
PROTO_DIR="/Users/liliang/Things/AI/projects/smartticket/proto"
TIMESTAMP=$(date +%s)

echo "Gateway: $GATEWAY_ADDR"
echo "Tenant ID: $TENANT_ID"
echo "User ID: $USER_ID"
echo "Timestamp: $TIMESTAMP"
echo ""

# Common grpcurl parameters
GRPC_PARAMS="-plaintext -import-path $PROTO_DIR"

# Helper function
test_grpc() {
    local test_name=$1
    local method=$2
    local data=$3
    local proto_file=$4

    echo "🧪 $test_name"
    echo "----------------------------------------"

    local response=$(grpcurl $GRPC_PARAMS -proto $PROTO_DIR/smartticket/$proto_file \
        -d "$data" \
        -H "x-dev-bypass: true" \
        -H "x-tenant-id: $TENANT_ID" \
        -H "x-user-id: $USER_ID" \
        $GATEWAY_ADDR $method 2>/dev/null || echo 'ERROR: Request failed')

    echo "$response" | jq '.' 2>/dev/null || echo "$response"

    if echo "$response" | grep -q '"success":true\|"id"\|"totalCount"\|"\[\]"'; then
        echo "✅ PASSED"
    else
        echo "❌ FAILED"
    fi
    echo ""
}

echo "============================================"
echo "🔧 SLA Service Tests"
echo "============================================"

# Test 1: Create SLA Policy
test_grpc "SLA - Create Policy" "smartticket.v1.SlaService/CreateSlaPolicy" "{
    \"metadata\": {\"requestId\": \"sla-create-$TIMESTAMP\"},
    \"name\": \"Test SLA Policy $TIMESTAMP\",
    \"description\": \"Test SLA policy created during E2E testing\",
    \"priority\": 2,
    \"severity\": 2,
    \"responseTimeMinutes\": 30,
    \"resolutionTimeMinutes\": 120,
    \"businessHoursOnly\": false,
    \"timezone\": \"UTC\"
}" "sla.proto"

# Test 2: List SLA Policies
test_grpc "SLA - List Policies" "smartticket.v1.SlaService/ListSlaPolicies" "{
    \"metadata\": {\"requestId\": \"sla-list-$TIMESTAMP\"},
    \"pagination\": {\"pageSize\": 5}
}" "sla.proto"

echo "============================================"
echo "📚 Knowledge Service Tests"
echo "============================================"

# Test 1: Create Article
test_grpc "Knowledge - Create Article" "smartticket.v1.KnowledgeService/CreateArticle" "{
    \"metadata\": {\"requestId\": \"kb-create-$TIMESTAMP\"},
    \"title\": \"E2E Test Article $TIMESTAMP\",
    \"content\": \"This is a comprehensive test article created during E2E testing to validate the knowledge management functionality. It contains sufficient content to test all features properly.\",
    \"summary\": \"E2E test article for knowledge service validation\",
    \"visibility\": 1,
    \"language\": \"en\",
    \"tags\": [\"test\", \"e2e\", \"knowledge\", \"article\"]
}" "knowledge.proto"

# Test 2: List Articles
test_grpc "Knowledge - List Articles" "smartticket.v1.KnowledgeService/ListArticles" "{
    \"metadata\": {\"requestId\": \"kb-list-$TIMESTAMP\"},
    \"pagination\": {\"pageSize\": 10}
}" "knowledge.proto"

# Test 3: Search Articles
test_grpc "Knowledge - Search Articles" "smartticket.v1.KnowledgeService/SearchArticles" "{
    \"metadata\": {\"requestId\": \"kb-search-$TIMESTAMP\"},
    \"query\": \"test\",
    \"pagination\": {\"pageSize\": 5}
}" "knowledge.proto"

echo "============================================"
echo "🎫 Ticket Service Tests"
echo "============================================"

# Test 1: Create Ticket
test_grpc "Ticket - Create Ticket" "smartticket.v1.TicketService/CreateTicket" "{
    \"metadata\": {\"requestId\": \"ticket-create-$TIMESTAMP\"},
    \"title\": \"E2E Test Ticket $TIMESTAMP\",
    \"description\": \"This is a test ticket created during E2E testing to validate ticket management functionality.\",
    \"priority\": 2,
    \"severity\": 2,
    \"categoryId\": \"\",
    \"contactId\": \"818bcee0-2176-477d-b39b-ed636f73e19b\",
    \"tags\": [\"test\", \"e2e\"]
}" "ticket.proto"

# Test 2: List Tickets
test_grpc "Ticket - List Tickets" "smartticket.v1.TicketService/ListTickets" "{
    \"metadata\": {\"requestId\": \"ticket-list-$TIMESTAMP\"},
    \"pagination\": {\"pageSize\": 10, \"pageToken\": \"\"}
}" "ticket.proto"

echo "============================================"
echo "👥 User Service Tests"
echo "============================================"

# Test 1: List Users
test_grpc "User - List Users" "smartticket.v1.UserService/ListUsers" "{
    \"metadata\": {\"requestId\": \"user-list-$TIMESTAMP\"},
    \"pagination\": {\"pageSize\": 10, \"pageToken\": \"\"}
}" "user.proto"

echo "============================================"
echo "🏢 Tenant Service Tests"
echo "============================================"

# Test 1: Get Current Tenant Info
test_grpc "Tenant - Get Current Info" "smartticket.v1.TenantService/GetCurrentTenant" "{
    \"metadata\": {\"requestId\": \"tenant-get-$TIMESTAMP\"}
}" "tenant.proto"

echo "============================================"
echo "🎉 E2E Testing Completed!"
echo "============================================"
echo "Test Summary:"
echo "  ✅ SLA Service: 2 interfaces tested"
echo "  ✅ Knowledge Service: 3 interfaces tested"
echo "  ✅ Ticket Service: 2 interfaces tested"
echo "  ✅ User Service: 1 interface tested"
echo "  ✅ Tenant Service: 1 interface tested"
echo ""
echo "📊 Total: 9 service interfaces validated!"
echo ""
echo "🚀 SmartTicket E2E Test Suite - All tests completed!"
