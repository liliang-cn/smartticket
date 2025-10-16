#!/bin/bash

# Ticket Management E2E Tests
# Tests comprehensive ticket lifecycle management

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Global variables for ticket tests
export CREATED_TICKETS=()
export CREATED_TICKET_IDS=()
export TICKET_CATEGORIES=()
export TEST_CUSTOMER_ID=""

# Function to test creating a ticket with different configurations
test_create_ticket() {
    local title="$1"
    local description="$2"
    local priority="$3"
    local severity="$4"
    local expected_success="$5"
    local user_email="$6"
    local user_password="$7"
    local tenant_domain="$8"
    local contact_id="$9"  # Optional contact ID for non-customer users

    log_info "Testing ticket creation: ${title}"
    log_info "Priority: ${priority}, Severity: ${severity}"

    # Login as the specified user
    if ! login_user "${user_email}" "${user_password}" "${tenant_domain}"; then
        log_error "Failed to login as ${user_email}"
        return 1
    fi

    # Determine contact_id based on user role
    local actual_contact_id="${contact_id}"
    if [[ -z "${actual_contact_id}" ]]; then
        if [[ "${user_email}" == *"customer"* ]]; then
            # For customer users, contact_id will be handled automatically
            actual_contact_id=""
        else
            # For non-customer users, need to provide a contact_id
            # Use a placeholder for now - will be set to an actual customer ID later
            actual_contact_id="${TEST_CONTACT_ID:-}"
        fi
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "CreateTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "title": "${title}",
  "description": "${description}",
  "priority": ${priority},
  "severity": ${severity},
  "contactId": "${actual_contact_id}",
  "tags": ["test", "automation", "$(date +%s)"]
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local ticket_id
        local ticket_number
        local ticket_title
        local ticket_priority
        local ticket_severity
        local ticket_status

        ticket_id=$(extract_json_field "${response}" "ticket.id")
        ticket_number=$(extract_json_field "${response}" "ticket.ticketNumber")
        ticket_title=$(extract_json_field "${response}" "ticket.title")
        ticket_priority=$(extract_json_field "${response}" "ticket.priority")
        ticket_severity=$(extract_json_field "${response}" "ticket.severity")
        ticket_status=$(extract_json_field "${response}" "ticket.status")

        if [[ "${expected_success}" == "true" ]]; then
            log_success "✓ Ticket created successfully"
            log_info "  Ticket ID: ${ticket_id}"
            log_info "  Ticket Number: ${ticket_number}"
            log_info "  Title: ${ticket_title}"
            log_info "  Priority: ${ticket_priority}"
            log_info "  Severity: ${ticket_severity}"
            log_info "  Status: ${ticket_status}"

            # Store ticket for later tests
            CREATED_TICKET_IDS+=("${ticket_id}")
            CREATED_TICKETS+=("${ticket_id}:${ticket_number}:${title}")

            # Verify ticket has valid ID and number
            if [[ -n "${ticket_id}" && -n "${ticket_number}" && "${ticket_title}" == "${title}" ]]; then
                log_success "✓ Ticket data validation passed"
            else
                log_error "✗ Ticket data validation failed"
                return 1
            fi

            # Verify initial status
            if [[ "${ticket_status}" == "1" ]]; then  # TICKET_STATUS_NEW = 1
                log_success "✓ Ticket has correct initial status: NEW"
            else
                log_warning "Ticket status: ${ticket_status} (expected: 1 for NEW)"
            fi

            return 0
        else
            log_error "✗ Ticket creation should have failed but succeeded"
            return 1
        fi
    else
        if [[ "${expected_success}" == "false" ]]; then
            log_success "✓ Ticket creation correctly failed"
            return 0
        else
            log_error "✗ Ticket creation failed unexpectedly"
            return 1
        fi
    fi
}

# Function to create a test customer user for ticket testing
create_test_customer() {
    log_info "Creating test customer user for ticket testing"

    local timestamp=$(date +%s)
    local customer_email="test-customer-${timestamp}@example.com"
    local customer_username="testcustomer${timestamp}"
    local customer_name="Test Customer ${timestamp}"

    # Login as tenant admin to create customer user
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as admin to create customer user"
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "CreateUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "email": "${customer_email}",
  "username": "${customer_username}",
  "full_name": "${customer_name}",
  "password": "testpass123",
  "role": "USER_ROLE_CUSTOMER_USER"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        TEST_CUSTOMER_ID=$(extract_json_field "${response}" "user.id")
        log_success "✓ Test customer user created successfully"
        log_info "  Customer ID: ${TEST_CUSTOMER_ID}"
        log_info "  Customer Email: ${customer_email}"
        return 0
    else
        log_error "✗ Failed to create test customer user"
        return 1
    fi
}

# Test: Create tickets with different priorities and severities
test_create_tickets_with_different_properties() {
    log_info "Testing ticket creation with different properties"

    local timestamp=$(date +%s)

    # Test creating tickets with different combinations
    test_create_ticket "Low Priority Issue ${timestamp}" "This is a low priority test ticket" "1" "1" "true" "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}" "${TEST_CUSTOMER_ID}" || return 1
    test_create_ticket "Normal Priority Issue ${timestamp}" "This is a normal priority test ticket" "2" "2" "true" "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}" "${TEST_CUSTOMER_ID}" || return 1
    test_create_ticket "High Priority Issue ${timestamp}" "This is a high priority test ticket" "3" "3" "true" "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}" "${TEST_CUSTOMER_ID}" || return 1
    test_create_ticket "Critical Issue ${timestamp}" "This is a critical priority test ticket" "4" "4" "true" "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}" "${TEST_CUSTOMER_ID}" || return 1

    log_success "✓ Ticket creation with different properties completed"
    return 0
}

# Test: List tickets
test_list_tickets() {
    log_info "Testing ticket listing functionality"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "ListTickets" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": {
    "pageSize": 20
  }
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local total_count
        local first_ticket_title
        total_count=$(extract_json_field "${response}" "pagination.totalCount")
        first_ticket_title=$(extract_json_field "${response}" "tickets[0].title")

        log_success "✓ Tickets listed successfully"
        log_info "  Total tickets: ${total_count}"
        log_info "  First ticket: ${first_ticket_title}"

        # Verify we have tickets
        if [[ "${total_count}" -gt 0 ]]; then
            log_success "✓ Ticket list contains tickets"
        else
            log_warning "Ticket list is empty (this might be expected for a fresh system)"
        fi

        return 0
    else
        log_error "✗ Failed to list tickets"
        return 1
    fi
}

# Test: Get specific ticket
test_get_ticket() {
    log_info "Testing get specific ticket"

    if [[ ${#CREATED_TICKET_IDS[@]} -eq 0 ]]; then
        log_warning "No tickets available for get test"
        return 0
    fi

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local test_ticket_id="${CREATED_TICKET_IDS[0]}"
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "GetTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${test_ticket_id}",
  "includeComments": true
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local ticket_id
        local ticket_title
        local ticket_status
        local ticket_priority
        local ticket_severity

        ticket_id=$(extract_json_field "${response}" "ticket.id")
        ticket_title=$(extract_json_field "${response}" "ticket.title")
        ticket_status=$(extract_json_field "${response}" "ticket.status")
        ticket_priority=$(extract_json_field "${response}" "ticket.priority")
        ticket_severity=$(extract_json_field "${response}" "ticket.severity")

        log_success "✓ Ticket retrieved successfully"
        log_info "  Ticket ID: ${ticket_id}"
        log_info "  Title: ${ticket_title}"
        log_info "  Status: ${ticket_status}"
        log_info "  Priority: ${ticket_priority}"
        log_info "  Severity: ${ticket_severity}"

        # Verify the ticket ID matches
        if [[ "${ticket_id}" == "${test_ticket_id}" ]]; then
            log_success "✓ Ticket ID verification passed"
        else
            log_error "✗ Ticket ID mismatch. Expected: ${test_ticket_id}, Got: ${ticket_id}"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to get ticket"
        return 1
    fi
}

# Test: Update ticket status
test_update_ticket_status() {
    log_info "Testing ticket status update"

    if [[ ${#CREATED_TICKET_IDS[@]} -eq 0 ]]; then
        log_warning "No tickets available for status update test"
        return 0
    fi

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local test_ticket_id="${CREATED_TICKET_IDS[0]}"

    # Update ticket status to IN_PROGRESS (3)
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "UpdateTicketStatus" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${test_ticket_id}",
  "status": 3,
  "comment": "Starting work on this ticket"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local ticket_id
        local ticket_status
        ticket_id=$(extract_json_field "${response}" "ticket.id")
        ticket_status=$(extract_json_field "${response}" "ticket.status")

        log_success "✓ Ticket status updated successfully"
        log_info "  Ticket ID: ${ticket_id}"
        log_info "  New status: ${ticket_status}"

        # Verify the status was updated
        if [[ "${ticket_status}" == "3" ]]; then  # TICKET_STATUS_IN_PROGRESS = 3
            log_success "✓ Ticket status updated correctly to IN_PROGRESS"
        else
            log_error "✗ Status update failed. Expected: 3, Got: ${ticket_status}"
            return 1
        fi

        # Update status to RESOLVED (6)
        response=$(make_grpc_call "smartticket.v1.TicketService" "UpdateTicketStatus" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${test_ticket_id}",
  "status": 6,
  "comment": "Issue has been resolved"
}
EOF)")

        if [[ $? -eq 0 ]]; then
            ticket_status=$(extract_json_field "${response}" "ticket.status")
            log_success "✓ Ticket status updated to RESOLVED"

            if [[ "${ticket_status}" == "6" ]]; then
                log_success "✓ Ticket status updated correctly to RESOLVED"
            else
                log_error "✗ Status update failed. Expected: 6, Got: ${ticket_status}"
                return 1
            fi
        else
            log_error "✗ Failed to update ticket status to RESOLVED"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to update ticket status"
        return 1
    fi
}

# Test: Assign ticket to user
test_assign_ticket() {
    log_info "Testing ticket assignment"

    if [[ ${#CREATED_TICKET_IDS[@]} -eq 0 ]]; then
        log_warning "No tickets available for assignment test"
        return 0
    fi

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local test_ticket_id="${CREATED_TICKET_IDS[0]}"

    # Assign ticket to current user
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "AssignTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${test_ticket_id}",
  "assignedToId": "${TEST_USER_ID}",
  "comment": "Assigning to myself for testing"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local ticket_id
        local assigned_to_id
        local assigned_to_name

        ticket_id=$(extract_json_field "${response}" "ticket.id")
        assigned_to_id=$(extract_json_field "${response}" "ticket.assignedToId")
        assigned_to_name=$(extract_json_field "${response}" "ticket.assignedTo.fullName")

        log_success "✓ Ticket assigned successfully"
        log_info "  Ticket ID: ${ticket_id}"
        log_info "  Assigned to ID: ${assigned_to_id}"
        log_info "  Assigned to name: ${assigned_to_name}"

        # Verify the assignment
        if [[ "${assigned_to_id}" == "${TEST_USER_ID}" ]]; then
            log_success "✓ Ticket assignment verification passed"
        else
            log_error "✗ Assignment verification failed. Expected: ${TEST_USER_ID}, Got: ${assigned_to_id}"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to assign ticket"
        return 1
    fi
}

# Test: Update ticket information
test_update_ticket() {
    log_info "Testing ticket information update"

    if [[ ${#CREATED_TICKET_IDS[@]} -eq 0 ]]; then
        log_warning "No tickets available for update test"
        return 0
    fi

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local test_ticket_id="${CREATED_TICKET_IDS[0]}"
    local updated_title="Updated Ticket Title $(date +%s)"
    local updated_description="This is the updated description for the ticket $(date +%s)"

    # Update ticket information
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "UpdateTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${test_ticket_id}",
  "title": "${updated_title}",
  "description": "${updated_description}",
  "priority": 3,
  "severity": 3,
  "tags": ["updated", "test", "$(date +%s)"]
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local ticket_id
        local ticket_title
        local ticket_description
        local ticket_priority
        local ticket_severity

        ticket_id=$(extract_json_field "${response}" "ticket.id")
        ticket_title=$(extract_json_field "${response}" "ticket.title")
        ticket_description=$(extract_json_field "${response}" "ticket.description")
        ticket_priority=$(extract_json_field "${response}" "ticket.priority")
        ticket_severity=$(extract_json_field "${response}" "ticket.severity")

        log_success "✓ Ticket updated successfully"
        log_info "  Ticket ID: ${ticket_id}"
        log_info "  Updated title: ${ticket_title}"
        log_info "  Updated priority: ${ticket_priority}"
        log_info "  Updated severity: ${ticket_severity}"

        # Verify the updates
        if [[ "${ticket_title}" == "${updated_title}" && "${ticket_priority}" == "3" && "${ticket_severity}" == "3" ]]; then
            log_success "✓ Ticket update verification passed"
        else
            log_error "✗ Ticket update verification failed"
            log_error "  Title expected: ${updated_title}, got: ${ticket_title}"
            log_error "  Priority expected: 3, got: ${ticket_priority}"
            log_error "  Severity expected: 3, got: ${ticket_severity}"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to update ticket"
        return 1
    fi
}

# Test: Add comment to ticket
test_add_comment() {
    log_info "Testing ticket comment functionality"

    if [[ ${#CREATED_TICKET_IDS[@]} -eq 0 ]]; then
        log_warning "No tickets available for comment test"
        return 0
    fi

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local test_ticket_id="${CREATED_TICKET_IDS[0]}"
    local comment_content="This is a test comment added at $(date +%s)"

    # Add comment to ticket
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "AddComment" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${test_ticket_id}",
  "content": "${comment_content}",
  "isInternal": false
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local comment_id
        local comment_content_response
        local comment_author

        comment_id=$(extract_json_field "${response}" "comment.id")
        comment_content_response=$(extract_json_field "${response}" "comment.content")
        comment_author=$(extract_json_field "${response}" "comment.author.fullName")

        log_success "✓ Comment added successfully"
        log_info "  Comment ID: ${comment_id}"
        log_info "  Content: ${comment_content_response}"
        log_info "  Author: ${comment_author}"

        # Verify the comment
        if [[ "${comment_content_response}" == "${comment_content}" ]]; then
            log_success "✓ Comment content verification passed"
        else
            log_error "✗ Comment verification failed"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to add comment"
        return 1
    fi
}

# Test: Get ticket comments
test_get_comments() {
    log_info "Testing get ticket comments"

    if [[ ${#CREATED_TICKET_IDS[@]} -eq 0 ]]; then
        log_warning "No tickets available for comments test"
        return 0
    fi

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local test_ticket_id="${CREATED_TICKET_IDS[0]}"

    # Get ticket comments
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "GetComments" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${test_ticket_id}",
  "pagination": {
    "pageSize": 10
  }
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local total_count
        local first_comment_content

        total_count=$(extract_json_field "${response}" "pagination.totalCount")
        first_comment_content=$(extract_json_field "${response}" "comments[0].content")

        log_success "✓ Comments retrieved successfully"
        log_info "  Total comments: ${total_count}"
        log_info "  First comment: ${first_comment_content}"

        # Verify we have comments
        if [[ "${total_count}" -gt 0 ]]; then
            log_success "✓ Comments list contains comments"
        else
            log_warning "No comments found (this might be expected)"
        fi

        return 0
    else
        log_error "✗ Failed to get comments"
        return 1
    fi
}

# Test: Search tickets
test_search_tickets() {
    log_info "Testing ticket search functionality"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    # Search for tickets with "test" in title
    local response
    response=$(make_grpc_call "smartticket.v1.TicketService" "SearchTickets" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "query": "test",
  "pagination": {
    "pageSize": 10
  }
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local total_matches
        local first_ticket_title

        total_matches=$(extract_json_field "${response}" "totalMatches")
        first_ticket_title=$(extract_json_field "${response}" "tickets[0].title")

        log_success "✓ Search completed successfully"
        log_info "  Total matches: ${total_matches}"
        log_info "  First result: ${first_ticket_title}"

        # Verify search results
        if [[ "${total_matches}" -gt 0 ]]; then
            log_success "✓ Search found results"
        else
            log_warning "Search found no results (this might be expected for a fresh system)"
        fi

        return 0
    else
        log_error "✗ Failed to search tickets"
        return 1
    fi
}

# Test: Customer user creating tickets
test_customer_create_ticket() {
    log_info "Testing customer ticket creation"

    local timestamp=$(date +%s)
    local customer_email="customer$(date +%s)@test.com"
    local customer_password="testpass123"

    # First create a customer user
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as admin to create customer user"
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "CreateUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "email": "${customer_email}",
  "username": "customer$(date +%s)",
  "full_name": "Test Customer ${timestamp}",
  "password": "${customer_password}",
  "role": "USER_ROLE_CUSTOMER_USER"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local customer_id
        customer_id=$(extract_json_field "${response}" "user.id")
        log_success "✓ Customer user created: ${customer_id}"

        # Login as customer
        if ! login_user "${customer_email}" "${customer_password}" "${TEST_TENANT_DOMAIN}"; then
            log_error "Failed to login as customer user"
            return 1
        fi

        # Create ticket as customer
        response=$(make_grpc_call "smartticket.v1.TicketService" "CreateTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${customer_id}"),
  "title": "Customer Support Request ${timestamp}",
  "description": "I need help with my account setup. This is a customer-created ticket.",
  "priority": 2,
  "severity": 2,
  "tags": ["customer-request", "help-needed"]
}
EOF)")

        if [[ $? -eq 0 ]]; then
            local ticket_id
            local ticket_title
            ticket_id=$(extract_json_field "${response}" "ticket.id")
            ticket_title=$(extract_json_field "${response}" "ticket.title")

            log_success "✓ Customer created ticket successfully"
            log_info "  Ticket ID: ${ticket_id}"
            log_info "  Title: ${ticket_title}"

            # Store customer ticket for later tests
            CREATED_TICKET_IDS+=("${ticket_id}")
            CREATED_TICKETS+=("${ticket_id}:CUSTOMER:${ticket_title}")

            return 0
        else
            log_error "✗ Customer failed to create ticket"
            return 1
        fi
    else
        log_error "✗ Failed to create customer user"
        return 1
    fi
}

# Cleanup test data
cleanup_test_tickets() {
    log_info "Cleaning up test tickets"

    # Login as tenant admin for cleanup
    if login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        for ticket_id in "${CREATED_TICKET_IDS[@]}"; do
            local response
            response=$(make_grpc_call "smartticket.v1.TicketService" "DeleteTicket" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticketId": "${ticket_id}"
}
EOF)")

            if [[ $? -eq 0 ]]; then
                log_success "✓ Test ticket deleted: ${ticket_id}"
            else
                log_warning "Failed to delete test ticket: ${ticket_id}"
            fi
        done
    fi
}

# Main test execution
main() {
    log_info "Starting Ticket Management E2E Tests"
    log_info "===================================="

    # Create test customer user for ticket testing
    create_test_customer || log_error "Failed to create test customer user"

    # Run tests
    test_create_tickets_with_different_properties || log_error "Ticket creation tests failed"
    test_list_tickets || log_error "Ticket listing tests failed"
    test_get_ticket || log_error "Get ticket tests failed"
    test_update_ticket_status || log_error "Ticket status update tests failed"
    test_assign_ticket || log_error "Ticket assignment tests failed"
    test_update_ticket || log_error "Ticket update tests failed"
    test_add_comment || log_error "Add comment tests failed"
    test_get_comments || log_error "Get comments tests failed"
    test_search_tickets || log_error "Search tickets tests failed"
    test_customer_create_ticket || log_error "Customer ticket creation tests failed"

    # Cleanup
    cleanup_test_tickets

    log_success "Ticket Management E2E Tests completed"
}

# Run tests if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi