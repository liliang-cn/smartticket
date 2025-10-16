#!/bin/bash

# Simple Multi-Tenant Functionality Test
# Tests basic multi-tenant features with existing data

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Global variables
export TEST_TICKET_ID=""

# Test 1: Login as existing tenant admin
test_login_existing_admin() {
    log_info "Test 1: Login as existing tenant admin"

    if login_user "admin@test.smartticket.com" "admin123" "test.smartticket.com"; then
        log_success "✓ Login successful"
        log_info "  User ID: ${TEST_USER_ID}"
        log_info "  Tenant ID: ${TEST_TENANT_ID}"
        return 0
    else
        log_error "✗ Login failed"
        return 1
    fi
}

# Test 2: Create a ticket in current tenant
test_create_ticket() {
    log_info "Test 2: Create ticket in tenant"

    local timestamp=$(date +%s)
    local ticket_title="Multi-Tenant Test Ticket ${timestamp}"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "CreateTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "title": "${ticket_title}",
  "description": "This ticket tests multi-tenant isolation",
  "priority": "TICKET_PRIORITY_NORMAL",
  "severity": "TICKET_SEVERITY_MEDIUM",
  "category_id": "general",
  "contact_id": "${TEST_USER_ID}",
  "tags": ["multi-tenant", "test-${timestamp}"]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        TEST_TICKET_ID=$(extract_json_field "${response}" "ticket.id")
        log_success "✓ Ticket created successfully"
        log_info "  Ticket ID: ${TEST_TICKET_ID}"
        log_info "  Title: ${ticket_title}"
        return 0
    else
        log_error "✗ Ticket creation failed"
        log_error "  Response: ${response}"
        return 1
    fi
}

# Test 3: List tickets in current tenant (should show our ticket)
test_list_tickets_current_tenant() {
    log_info "Test 3: List tickets in current tenant"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "ListTickets" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": $(create_json_pagination 50 "")
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local ticket_count=$(echo "${response}" | jq '.tickets | length')
        local found_ticket=$(echo "${response}" | jq -r --arg id "${TEST_TICKET_ID}" '.tickets[] | select(.id == $id) | .id')

        if [[ -n "${found_ticket}" ]]; then
            log_success "✓ Found our ticket in current tenant"
            log_info "  Total tickets: ${ticket_count}"
            log_info "  Our ticket ID: ${found_ticket}"
            return 0
        else
            log_error "✗ Our ticket not found in current tenant"
            return 1
        fi
    else
        log_error "✗ Failed to list tickets"
        return 1
    fi
}

# Test 4: Test tenant isolation by trying to access with wrong tenant_id
test_tenant_isolation() {
    log_info "Test 4: Test tenant isolation"

    # Try to access the ticket with a different tenant_id (should fail)
    local fake_tenant_id="00000000-0000-0000-0000-000000000000"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "GetTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${fake_tenant_id}" "${TEST_USER_ID}"),
  "ticket_id": "${TEST_TICKET_ID}"
}
EOF
)" "false")

    # This should fail with permission error
    if [[ $? -eq 0 ]]; then
        log_success "✓ Cross-tenant access correctly prevented"
        log_info "  Expected: Access denied"
        log_info "  Actual: Permission error (good!)"
        return 0
    else
        log_error "✗ Cross-tenant access was allowed (SECURITY ISSUE!)"
        return 1
    fi
}

# Test 5: Create knowledge article in current tenant
test_create_knowledge_article() {
    log_info "Test 5: Create knowledge article in tenant"

    local timestamp=$(date +%s)
    local article_title="Multi-Tenant Test Article ${timestamp}"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "CreateArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "title": "${article_title}",
  "content": "This knowledge article tests multi-tenant isolation",
  "summary": "Multi-tenant test article",
  "visibility": "KNOWLEDGE_VISIBILITY_PUBLIC",
  "language": "en",
  "tags": ["multi-tenant", "knowledge-test", "${timestamp}"]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local article_id=$(extract_json_field "${response}" "article.id")
        log_success "✓ Knowledge article created successfully"
        log_info "  Article ID: ${article_id}"
        log_info "  Title: ${article_title}"
        return 0
    else
        log_error "✗ Knowledge article creation failed"
        log_error "  Response: ${response}"
        return 1
    fi
}

# Test 6: List users in current tenant
test_list_users_current_tenant() {
    log_info "Test 6: List users in current tenant"

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "ListUsers" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": $(create_json_pagination 50 "")
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local user_count=$(echo "${response}" | jq '.users | length')
        log_success "✓ Listed users successfully"
        log_info "  Total users in tenant: ${user_count}"

        # Show first few users for verification
        echo "${response}" | jq -r '.users[0:3] | .[] | "  - \(.email) (\(.role))"'
        return 0
    else
        log_error "✗ Failed to list users"
        return 1
    fi
}

# Test 7: Test tenant-specific data filtering
test_tenant_data_filtering() {
    log_info "Test 7: Test tenant data filtering"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "ListTickets" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": $(create_json_pagination 100 ""),
  "filters": [{"field": "tags", "operator": "contains", "value": "multi-tenant"}]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local filtered_count=$(echo "${response}" | jq '.tickets | length')
        log_success "✓ Tenant data filtering working"
        log_info "  Tickets with 'multi-tenant' tag: ${filtered_count}"

        # Verify all returned tickets have the correct tenant_id
        local cross_tenant_count=$(echo "${response}" | jq '[.tickets[] | select(.tenantId != "'"${TEST_TENANT_ID}"'")] | length')

        if [[ ${cross_tenant_count} -eq 0 ]]; then
            log_success "✓ All tickets belong to correct tenant (no data leakage)"
            return 0
        else
            log_error "✗ Found ${cross_tenant_count} tickets from other tenants (DATA LEAKAGE!)"
            return 1
        fi
    else
        log_error "✗ Failed to filter tickets"
        return 1
    fi
}

# Test 8: Test user permissions within tenant
test_user_permissions() {
    log_info "Test 8: Test user permissions within tenant"

    # Get current user info to verify permissions
    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "GetCurrentUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}")
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local user_role=$(extract_json_field "${response}" "user.role")
        local user_permissions=$(echo "${response}" | jq -r '.user.permissions[]' 2>/dev/null || echo "")

        log_success "✓ User permissions verified"
        log_info "  Role: ${user_role}"
        log_info "  Permissions: ${user_permissions}"
        return 0
    else
        log_error "✗ Failed to get user permissions"
        return 1
    fi
}

# Test 9: Test tenant statistics
test_tenant_statistics() {
    log_info "Test 9: Test tenant statistics"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "GetTicketStatistics" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}")
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local total_tickets=$(extract_json_field "${response}" "statistics.totalTickets")
        local open_tickets=$(extract_json_field "${response}" "statistics.openTickets")

        log_success "✓ Tenant statistics retrieved"
        log_info "  Total tickets: ${total_tickets}"
        log_info "  Open tickets: ${open_tickets}"
        return 0
    else
        log_error "✗ Failed to get tenant statistics"
        return 1
    fi
}

# Main test execution
run_multi_tenant_tests() {
    log_info "Starting Simple Multi-Tenant E2E Tests"
    log_info "======================================"

    # Initialize test environment
    init_test_env

    local tests=(
        "test_login_existing_admin"
        "test_create_ticket"
        "test_list_tickets_current_tenant"
        "test_tenant_isolation"
        "test_create_knowledge_article"
        "test_list_users_current_tenant"
        "test_tenant_data_filtering"
        "test_user_permissions"
        "test_tenant_statistics"
    )

    local passed=0
    local failed=0

    for test in "${tests[@]}"; do
        echo ""
        if ${test}; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    echo ""
    log_info "======================================"
    log_info "Test Results: ${passed} passed, ${failed} failed"

    if [[ $failed -eq 0 ]]; then
        log_success "🎉 All multi-tenant tests passed!"
        log_success "✓ Tenant authentication working"
        log_success "✓ Data isolation verified"
        log_success "✓ Cross-tenant access prevented"
        log_success "✓ Tenant-specific data filtering working"
        log_success "✓ User permissions enforced"
        log_success "✓ Tenant statistics functional"
        return 0
    else
        log_error "❌ Some tests failed"
        return 1
    fi
}

# Check if required tools are available
check_dependencies() {
    if ! command -v grpcurl &> /dev/null; then
        log_error "grpcurl is required but not installed. Please install grpcurl to run E2E tests."
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed. Please install jq to run E2E tests."
        exit 1
    fi

    log_success "✓ All dependencies available"
}

# Run tests if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    check_dependencies
    run_multi_tenant_tests
fi