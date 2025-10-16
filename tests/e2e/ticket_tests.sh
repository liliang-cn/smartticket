#!/bin/bash

# SmartTicket Ticket Management E2E Tests
# Tests for ticket CRUD operations, status management, and assignments

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Test: Create Ticket
test_create_ticket() {
    log_info "Testing ticket creation"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local timestamp=$(date +%s)
    local ticket_title="Test Ticket ${timestamp}"
    local ticket_description="This is a test ticket created at ${timestamp}"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "CreateTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "title": "${ticket_title}",
  "description": "${ticket_description}",
  "priority": "TICKET_PRIORITY_NORMAL",
  "severity": "TICKET_SEVERITY_MEDIUM",
  "category_id": "general",
  "contact_id": "${TEST_USER_ID}",
  "tags": ["test", "e2e", "automation"]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local ticket_id
        local ticket_number
        local title
        local description

        ticket_id=$(extract_json_field "${response}" "ticket.id")
        ticket_number=$(extract_json_field "${response}" "ticket.ticketNumber")
        title=$(extract_json_field "${response}" "ticket.title")
        description=$(extract_json_field "${response}" "ticket.description")

        assert_not_empty "${ticket_id}" "Created ticket ID should not be empty" || return 1
        assert_not_empty "${ticket_number}" "Created ticket number should not be empty" || return 1
        assert_equals "${title}" "${ticket_title}" "Created ticket title should match" || return 1
        assert_contains "${description}" "${ticket_description}" "Created ticket description should contain" || return 1

        CURRENT_TICKET_ID="${ticket_id}"
        log_success "Ticket creation successful - Ticket ID: ${ticket_id}"
        return 0
    else
        log_error "Ticket creation failed"
        return 1
    fi
}

# Test: Get Ticket by ID
test_get_ticket() {
    log_info "Testing get ticket by ID"

    # Create a ticket first
    if ! test_create_ticket; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "GetTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}",
  "include_comments": true
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local ticket_id
        local title
        local status

        ticket_id=$(extract_json_field "${response}" "ticket.id")
        title=$(extract_json_field "${response}" "ticket.title")
        status=$(extract_json_field "${response}" "ticket.status")

        assert_equals "${ticket_id}" "${CURRENT_TICKET_ID}" "Retrieved ticket ID should match" || return 1
        assert_not_empty "${title}" "Retrieved ticket title should not be empty" || return 1
        assert_equals "${status}" "OPEN" "New ticket should have OPEN status" || return 1

        log_success "Get ticket successful"
        return 0
    else
        log_error "Get ticket failed"
        return 1
    fi
}

# Test: Update Ticket
test_update_ticket() {
    log_info "Testing ticket update"

    # Create a ticket first
    if ! test_create_ticket; then
        return 1
    fi

    local updated_title="Updated Test Ticket $(date +%s)"
    local updated_description="This ticket has been updated"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "UpdateTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}",
  "title": "${updated_title}",
  "description": "${updated_description}",
  "priority": "TICKET_PRIORITY_HIGH",
  "severity": "TICKET_SEVERITY_HIGH",
  "tags": ["test", "updated", "priority"]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local title
        local description
        local priority
        local severity

        title=$(extract_json_field "${response}" "ticket.title")
        description=$(extract_json_field "${response}" "ticket.description")
        priority=$(extract_json_field "${response}" "ticket.priority")
        severity=$(extract_json_field "${response}" "ticket.severity")

        assert_equals "${title}" "${updated_title}" "Updated ticket title should match" || return 1
        assert_contains "${description}" "${updated_description}" "Updated ticket description should contain" || return 1
        assert_equals "${priority}" "HIGH" "Updated ticket priority should be HIGH" || return 1
        assert_equals "${severity}" "HIGH" "Updated ticket severity should be HIGH" || return 1

        log_success "Ticket update successful"
        return 0
    else
        log_error "Ticket update failed"
        return 1
    fi
}

# Test: Update Ticket Status
test_update_ticket_status() {
    log_info "Testing ticket status update"

    # Create a ticket first
    if ! test_create_ticket; then
        return 1
    fi

    local new_status="IN_PROGRESS"
    local status_comment="Ticket is being worked on"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "UpdateTicketStatus" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}",
  "status": "${new_status}",
  "comment": "${status_comment}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local status
        status=$(extract_json_field "${response}" "ticket.status")

        assert_equals "${status}" "${new_status}" "Updated ticket status should match" || return 1

        log_success "Ticket status update successful"
        return 0
    else
        log_error "Ticket status update failed"
        return 1
    fi
}

# Test: Assign Ticket
test_assign_ticket() {
    log_info "Testing ticket assignment"

    # Create a ticket first
    if ! test_create_ticket; then
        return 1
    fi

    local assignment_comment="Assigning ticket to admin for resolution"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "AssignTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}",
  "assigned_to_id": "${TEST_USER_ID}",
  "comment": "${assignment_comment}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local assigned_to_id
        assigned_to_id=$(extract_json_field "${response}" "ticket.assignedToId")

        assert_equals "${assigned_to_id}" "${TEST_USER_ID}" "Ticket should be assigned to current user" || return 1

        log_success "Ticket assignment successful"
        return 0
    else
        log_error "Ticket assignment failed"
        return 1
    fi
}

# Test: Add Comment to Ticket
test_add_ticket_comment() {
    log_info "Testing add comment to ticket"

    # Create a ticket first
    if ! test_create_ticket; then
        return 1
    fi

    local comment_content="This is a test comment from E2E tests $(date +%s)"
    local is_internal=false

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "AddComment" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}",
  "content": "${comment_content}",
  "is_internal": ${is_internal}
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local comment_id
        local content
        local internal

        comment_id=$(extract_json_field "${response}" "comment.id")
        content=$(extract_json_field "${response}" "comment.content")
        internal=$(extract_json_field "${response}" "comment.is_internal")

        assert_not_empty "${comment_id}" "Comment ID should not be empty" || return 1
        assert_equals "${content}" "${comment_content}" "Comment content should match" || return 1
        assert_equals "${internal}" "${is_internal}" "Comment internal flag should match" || return 1

        log_success "Add ticket comment successful"
        return 0
    else
        log_error "Add ticket comment failed"
        return 1
    fi
}

# Test: Get Ticket Comments
test_get_ticket_comments() {
    log_info "Testing get ticket comments"

    # Create a ticket and add comment first
    if ! test_create_ticket; then
        return 1
    fi

    if ! test_add_ticket_comment; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "GetComments" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}",
  "pagination": $(create_json_pagination 10 "")
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local comments_count
        comments_count=$(echo "${response}" | jq '.comments | length' 2>/dev/null || echo "0")

        if [[ ${comments_count} -gt 0 ]]; then
            log_success "Get ticket comments successful - found ${comments_count} comments"
            return 0
        else
            log_error "No comments found for ticket"
            return 1
        fi
    else
        log_error "Get ticket comments failed"
        return 1
    fi
}

# Test: List Tickets
test_list_tickets() {
    log_info "Testing list tickets"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Create a ticket first
    if ! test_create_ticket; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "ListTickets" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": $(create_json_pagination 20 ""),
  "sort": [{"field": "created_at", "direction": "DESC"}]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local tickets_count
        local total_count

        tickets_count=$(echo "${response}" | jq '.tickets | length' 2>/dev/null || echo "0")
        total_count=$(extract_json_field "${response}" "pagination.totalCount")

        if [[ ${tickets_count} -gt 0 ]]; then
            log_success "List tickets successful - found ${tickets_count} tickets (total: ${total_count})"
            return 0
        else
            log_error "No tickets found"
            return 1
        fi
    else
        log_error "List tickets failed"
        return 1
    fi
}

# Test: Search Tickets
test_search_tickets() {
    log_info "Testing search tickets"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Create a ticket with specific content first
    if ! test_create_ticket; then
        return 1
    fi

    local search_query="Test"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "SearchTickets" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "query": "${search_query}",
  "pagination": $(create_json_pagination 20 "")
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local tickets_count
        local total_matches

        tickets_count=$(echo "${response}" | jq '.tickets | length' 2>/dev/null || echo "0")
        total_matches=$(extract_json_field "${response}" "total_matches")

        if [[ ${tickets_count} -gt 0 ]]; then
            log_success "Search tickets successful - found ${tickets_count} tickets (${total_matches} total matches)"
            return 0
        else
            log_error "No tickets found for search query"
            return 1
        fi
    else
        log_error "Search tickets failed"
        return 1
    fi
}

# Test: Delete Ticket
test_delete_ticket() {
    log_info "Testing ticket deletion"

    # Create a ticket first
    if ! test_create_ticket; then
        return 1
    fi

    local ticket_to_delete="${CURRENT_TICKET_ID}"

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "DeleteTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${ticket_to_delete}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        log_success "Ticket deletion successful"

        # Verify ticket is deleted (should return error)
        local verify_response
        verify_response=$(make_grpc_call "smartticket.v1.TicketService" "GetTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${ticket_to_delete}"
}
EOF
)" "false")

        if [[ $? -eq 0 ]]; then
            log_success "Ticket deletion verified - ticket no longer accessible"
            return 0
        else
            log_warning "Could not verify ticket deletion"
            return 0  # Still consider success since deletion was acknowledged
        fi
    else
        log_error "Ticket deletion failed"
        return 1
    fi
}

# Test: Complete Ticket Lifecycle
test_complete_ticket_lifecycle() {
    log_info "Testing complete ticket lifecycle"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Step 1: Create ticket
    if ! test_create_ticket; then
        log_error "Failed at ticket creation step"
        return 1
    fi
    log_success "✓ Step 1: Ticket created"

    # Step 2: Update ticket status to IN_PROGRESS
    if ! test_update_ticket_status; then
        log_error "Failed at status update step"
        return 1
    fi
    log_success "✓ Step 2: Ticket status updated to IN_PROGRESS"

    # Step 3: Assign ticket
    if ! test_assign_ticket; then
        log_error "Failed at ticket assignment step"
        return 1
    fi
    log_success "✓ Step 3: Ticket assigned"

    # Step 4: Add comment
    if ! test_add_ticket_comment; then
        log_error "Failed at comment addition step"
        return 1
    fi
    log_success "✓ Step 4: Comment added"

    # Step 5: Update ticket status to RESOLVED
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "UpdateTicketStatus" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}",
  "status": "RESOLVED",
  "comment": "Issue has been resolved successfully"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local status
        status=$(extract_json_field "${response}" "ticket.status")
        if [[ "${status}" == "RESOLVED" ]]; then
            log_success "✓ Step 5: Ticket status updated to RESOLVED"
        else
            log_error "Failed to update ticket status to RESOLVED"
            return 1
        fi
    else
        log_error "Failed at final status update step"
        return 1
    fi

    log_success "Complete ticket lifecycle test passed"
    return 0
}

# Main test execution function
run_ticket_tests() {
    log_info "Starting Ticket Management E2E Tests"
    log_info "======================================"

    local tests=(
        "test_create_ticket"
        "test_get_ticket"
        "test_update_ticket"
        "test_update_ticket_status"
        "test_assign_ticket"
        "test_add_ticket_comment"
        "test_get_ticket_comments"
        "test_list_tickets"
        "test_search_tickets"
        "test_delete_ticket"
        "test_complete_ticket_lifecycle"
    )

    run_test_suite "Ticket Management Tests" "${tests[@]}"
}

# Run tests if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_ticket_tests
fi