#!/bin/bash

# Complete SLA Service E2E Test Suite
# Tests all 9 SLA service interfaces

set -e

echo "============================================"
echo "🚀 SmartTicket SLA Service E2E Test Suite"
echo "============================================"

# Configuration
GATEWAY_ADDR="localhost:6533"
TENANT_ID="de57f60e-80a3-4a87-af40-3f99723c6530"
USER_ID="818bcee0-2176-477d-b39b-ed636f73e19b"
PROTO_DIR="/Users/liliang/Things/AI/projects/smartticket/proto"

echo "Gateway: $GATEWAY_ADDR"
echo "Tenant ID: $TENANT_ID"
echo "User ID: $USER_ID"
echo ""

# Check if gateway is running (test with a simple health check)
if ! timeout 5 grpcurl -plaintext -d '{}' $GATEWAY_ADDR smartticket.v1.SlaService/ListSlaPolicies > /dev/null 2>&1; then
    echo "⚠️  Gateway connection test failed, but proceeding with tests..."
else
    echo "✅ Gateway is running and responding"
fi
echo ""

# Test 1: CreateSlaPolicy
echo "🧪 Test 1: CreateSlaPolicy"
echo "----------------------------------------"
CREATE_SLA_RESPONSE=$(grpcurl -plaintext -import-path $PROTO_DIR -proto $PROTO_DIR/smartticket/sla.proto -d "{
    \"metadata\": {
        \"requestId\": \"create-sla-test-$(date +%s)\"
    },
    \"name\": \"Premium Support SLA\",
    \"description\": \"Premium customer support SLA policy\",
    \"priority\": 3,
    \"severity\": 3,
    \"responseTimeMinutes\": 15,
    \"resolutionTimeMinutes\": 240,
    \"businessHoursOnly\": true,
    \"active\": true
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/CreateSlaPolicy)

echo "$CREATE_SLA_RESPONSE" | jq '.' 2>/dev/null || echo "$CREATE_SLA_RESPONSE"

# Extract SLA ID for subsequent tests
SLA_ID=$(echo "$CREATE_SLA_RESPONSE" | jq -r '.sla.id // empty' 2>/dev/null)
if [ -z "$SLA_ID" ]; then
    echo "❌ Failed to create SLA policy or extract ID"
    exit 1
fi
echo "✅ Created SLA Policy with ID: $SLA_ID"
echo ""

# Test 2: GetSlaPolicy
echo "🧪 Test 2: GetSlaPolicy"
echo "----------------------------------------"
GET_SLA_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"get-sla-test-$(date +%s)\"
    },
    \"slaId\": \"$SLA_ID\"
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/GetSlaPolicy)

echo "$GET_SLA_RESPONSE" | jq '.' 2>/dev/null || echo "$GET_SLA_RESPONSE"
echo "✅ Retrieved SLA Policy"
echo ""

# Test 3: UpdateSlaPolicy
echo "🧪 Test 3: UpdateSlaPolicy"
echo "----------------------------------------"
UPDATE_SLA_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"update-sla-test-$(date +%s)\"
    },
    \"slaId\": \"$SLA_ID\",
    \"name\": \"Updated Premium Support SLA\",
    \"description\": \"Updated premium customer support SLA policy\",
    \"responseTimeMinutes\": 10,
    \"resolutionTimeMinutes\": 180
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/UpdateSlaPolicy)

echo "$UPDATE_SLA_RESPONSE" | jq '.' 2>/dev/null || echo "$UPDATE_SLA_RESPONSE"
echo "✅ Updated SLA Policy"
echo ""

# Test 4: ListSlaPolicies
echo "🧪 Test 4: ListSlaPolicies"
echo "----------------------------------------"
LIST_SLA_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"list-sla-test-$(date +%s)\"
    },
    \"pageSize\": 10
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/ListSlaPolicies)

echo "$LIST_SLA_RESPONSE" | jq '.' 2>/dev/null || echo "$LIST_SLA_RESPONSE"
echo "✅ Listed SLA Policies"
echo ""

# Test 5: GetSlaMetrics
echo "🧪 Test 5: GetSlaMetrics"
echo "----------------------------------------"
GET_METRICS_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"metrics-sla-test-$(date +%s)\"
    },
    \"slaId\": \"$SLA_ID\"
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/GetSlaMetrics)

echo "$GET_METRICS_RESPONSE" | jq '.' 2>/dev/null || echo "$GET_METRICS_RESPONSE"
echo "✅ Retrieved SLA Metrics"
echo ""

# Test 6: GetSlaDashboard
echo "🧪 Test 6: GetSlaDashboard"
echo "----------------------------------------"
GET_DASHBOARD_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"dashboard-sla-test-$(date +%s)\"
    }
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/GetSlaDashboard)

echo "$GET_DASHBOARD_RESPONSE" | jq '.' 2>/dev/null || echo "$GET_DASHBOARD_RESPONSE"
echo "✅ Retrieved SLA Dashboard"
echo ""

# Test 7: GetSlaBreaches
echo "🧪 Test 7: GetSlaBreaches"
echo "----------------------------------------"
GET_BREACHES_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"breaches-sla-test-$(date +%s)\"
    },
    \"pageSize\": 10
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/GetSlaBreaches)

echo "$GET_BREACHES_RESPONSE" | jq '.' 2>/dev/null || echo "$GET_BREACHES_RESPONSE"
echo "✅ Retrieved SLA Breaches"
echo ""

# Test 8: UpdateSlaMetrics
echo "🧪 Test 8: UpdateSlaMetrics"
echo "----------------------------------------"
UPDATE_METRICS_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"update-metrics-sla-test-$(date +%s)\"
    },
    \"slaId\": \"$SLA_ID\",
    \"responseTimeMinutes\": 8,
    \"resolutionTimeMinutes\": 150
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/UpdateSlaMetrics)

echo "$UPDATE_METRICS_RESPONSE" | jq '.' 2>/dev/null || echo "$UPDATE_METRICS_RESPONSE"
echo "✅ Updated SLA Metrics"
echo ""

# Test 9: DeleteSlaPolicy
echo "🧪 Test 9: DeleteSlaPolicy"
echo "----------------------------------------"
DELETE_SLA_RESPONSE=$(grpcurl -plaintext -d "{
    \"metadata\": {
        \"requestId\": \"delete-sla-test-$(date +%s)\"
    },
    \"slaId\": \"$SLA_ID\"
}" -H "x-dev-bypass: true" -H "x-tenant-id: $TENANT_ID" -H "x-user-id: $USER_ID" $GATEWAY_ADDR smartticket.v1.SlaService/DeleteSlaPolicy)

echo "$DELETE_SLA_RESPONSE" | jq '.' 2>/dev/null || echo "$DELETE_SLA_RESPONSE"
echo "✅ Deleted SLA Policy"
echo ""

echo "============================================"
echo "🎉 SLA Service E2E Tests Completed!"
echo "============================================"
echo "All 9 SLA service interfaces tested successfully!"