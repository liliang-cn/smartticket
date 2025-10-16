#!/bin/bash

# User Management E2E Tests
# Tests user creation, role management, and permissions

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Global variables for user tests
export TEST_USERS=()
export CREATED_USER_IDS=()

# Function to test creating a user with different roles
test_create_user_with_role() {
    local role="$1"
    local email="$2"
    local username="$3"
    local full_name="$4"
    local expected_success="$5"

    log_info "Testing user creation with role: ${role}"
    log_info "Email: ${email}, Username: ${username}"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "CreateUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "email": "${email}",
  "username": "${username}",
  "full_name": "${full_name}",
  "password": "testpass123",
  "role": "${role}"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local user_id
        user_id=$(extract_json_field "${response}" "user.id")
        local created_role
        created_role=$(extract_json_field "${response}" "user.role")

        if [[ "${expected_success}" == "true" ]]; then
            log_success "✓ User created successfully - ID: ${user_id}, Role: ${created_role}"
            CREATED_USER_IDS+=("${user_id}")

            # Store user info for later tests
            TEST_USERS+=("${email}:${username}:${role}:${user_id}")

            # Verify role matches expected
            if [[ "${created_role}" == "${role}" ]]; then
                log_success "✓ User role assigned correctly: ${created_role}"
            else
                log_error "✗ Role mismatch. Expected: ${role}, Got: ${created_role}"
                return 1
            fi
            return 0
        else
            log_error "✗ User creation should have failed but succeeded"
            return 1
        fi
    else
        if [[ "${expected_success}" == "false" ]]; then
            log_success "✓ User creation correctly failed for role: ${role}"
            return 0
        else
            log_error "✗ User creation failed unexpectedly for role: ${role}"
            return 1
        fi
    fi
}

# Test: Create users with different roles
test_create_users() {
    log_info "Testing user creation with different roles"

    local timestamp=$(date +%s)

    # Test creating users with valid roles
    test_create_user_with_role "USER_ROLE_TENANT_ADMIN" "admin${timestamp}@test.com" "admin${timestamp}" "Admin User ${timestamp}" "true" || return 1
    test_create_user_with_role "USER_ROLE_SUPPORT_ENGINEER" "support${timestamp}@test.com" "support${timestamp}" "Support Engineer ${timestamp}" "true" || return 1
    test_create_user_with_role "USER_ROLE_CUSTOMER_USER" "customer${timestamp}@test.com" "customer${timestamp}" "Customer User ${timestamp}" "true" || return 1
    test_create_user_with_role "USER_ROLE_SALES" "sales${timestamp}@test.com" "sales${timestamp}" "Sales User ${timestamp}" "true" || return 1

    # Test that regular admin cannot create Super Admin users
    test_create_user_with_role "USER_ROLE_SUPER_ADMIN" "super${timestamp}@test.com" "super${timestamp}" "Super Admin User ${timestamp}" "false" || return 1

    log_success "✓ User creation tests completed"
    return 0
}

# Test: List users
test_list_users() {
    log_info "Testing user listing functionality"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "ListUsers" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": {
    "pageSize": 20
  }
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local total_count
        total_count=$(extract_json_field "${response}" "pagination.totalCount")
        local first_user_email
        first_user_email=$(extract_json_field "${response}" "users[0].email")

        log_success "✓ Users listed successfully"
        log_info "  Total users: ${total_count}"
        log_info "  First user: ${first_user_email}"

        # Verify we have users
        if [[ "${total_count}" -gt 0 ]]; then
            log_success "✓ User list contains users"
        else
            log_error "✗ User list is empty"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to list users"
        return 1
    fi
}

# Test: Get specific user
test_get_user() {
    log_info "Testing get specific user"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    if [[ ${#CREATED_USER_IDS[@]} -eq 0 ]]; then
        log_warning "No users available for get test"
        return 0
    fi

    local test_user_id="${CREATED_USER_IDS[0]}"
    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "GetUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "userId": "${test_user_id}"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local user_email
        local user_role
        user_email=$(extract_json_field "${response}" "user.email")
        user_role=$(extract_json_field "${response}" "user.role")

        log_success "✓ User retrieved successfully"
        log_info "  Email: ${user_email}"
        log_info "  Role: ${user_role}"

        return 0
    else
        log_error "✗ Failed to get user"
        return 1
    fi
}

# Test: Update user information
test_update_user() {
    log_info "Testing user update functionality"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    if [[ ${#CREATED_USER_IDS[@]} -eq 0 ]]; then
        log_warning "No users available for update test"
        return 0
    fi

    local test_user_id="${CREATED_USER_IDS[0]}"
    local updated_name="Updated Name $(date +%s)"

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "UpdateUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "userId": "${test_user_id}",
  "fullName": "${updated_name}",
  "role": "USER_ROLE_SUPPORT_ENGINEER"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local updated_name_response
        updated_name_response=$(extract_json_field "${response}" "user.fullName")
        local updated_role
        updated_role=$(extract_json_field "${response}" "user.role")

        log_success "✓ User updated successfully"
        log_info "  Updated name: ${updated_name_response}"
        log_info "  Updated role: ${updated_role}"

        # Verify the update
        if [[ "${updated_name_response}" == "${updated_name}" ]]; then
            log_success "✓ User name updated correctly"
        else
            log_error "✗ User name update failed"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to update user"
        return 1
    fi
}

# Test: Update user status (activate/deactivate)
test_update_user_status() {
    log_info "Testing user status update"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    if [[ ${#CREATED_USER_IDS[@]} -eq 0 ]]; then
        log_warning "No users available for status update test"
        return 0
    fi

    local test_user_id="${CREATED_USER_IDS[0]}"

    # Deactivate user
    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "UpdateUserStatus" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "userId": "${test_user_id}",
  "isActive": false,
  "reason": "Test deactivation"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local is_active
        is_active=$(extract_json_field "${response}" "user.isActive")

        log_success "✓ User deactivated successfully"
        log_info "  User active status: ${is_active}"

        if [[ "${is_active}" == "false" ]]; then
            log_success "✓ User status updated correctly"
        else
            log_error "✗ User status update failed"
            return 1
        fi

        # Reactivate user
        response=$(make_grpc_call "smartticket.v1.UserService" "UpdateUserStatus" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "userId": "${test_user_id}",
  "isActive": true,
  "reason": "Test reactivation"
}
EOF)")

        if [[ $? -eq 0 ]]; then
            log_success "✓ User reactivated successfully"
        else
            log_error "✗ Failed to reactivate user"
            return 1
        fi

        return 0
    else
        log_error "✗ Failed to update user status"
        return 1
    fi
}

# Test: Get user permissions
test_get_user_permissions() {
    log_info "Testing get user permissions"

    # Login as tenant admin
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        log_error "Failed to login as tenant admin"
        return 1
    fi

    if [[ ${#CREATED_USER_IDS[@]} -eq 0 ]]; then
        log_warning "No users available for permissions test"
        return 0
    fi

    local test_user_id="${CREATED_USER_IDS[0]}"

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "GetUserPermissions" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "userId": "${test_user_id}"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local permission_count
        permission_count=$(extract_json_field "${response}" "permissions" | jq '. | length' 2>/dev/null || echo "0")

        log_success "✓ User permissions retrieved successfully"
        log_info "  Permission count: ${permission_count}"

        return 0
    else
        log_error "✗ Failed to get user permissions"
        return 1
    fi
}

# Test: User login with different roles
test_user_role_login() {
    log_info "Testing user login with different roles"

    for user_info in "${TEST_USERS[@]}"; do
        IFS=':' read -r email username role user_id <<< "${user_info}"

        log_info "Testing login for user: ${email} (role: ${role})"

        local response
        response=$(make_grpc_call "smartticket.v1.AuthService" "Login" "$(cat <<EOF
{
  "email": "${email}",
  "password": "testpass123",
  "tenantDomain": "${TEST_TENANT_DOMAIN}"
}
EOF)")

        if [[ $? -eq 0 ]]; then
            local logged_in_role
            logged_in_role=$(extract_json_field "${response}" "user.role")
            local is_active
            is_active=$(extract_json_field "${response}" "user.isActive")

            log_success "✓ User login successful: ${email}"
            log_info "  Role: ${logged_in_role}, Active: ${is_active}"

            if [[ "${logged_in_role}" == "${role}" ]]; then
                log_success "✓ User role preserved correctly during login"
            else
                log_error "✗ Role mismatch during login. Expected: ${role}, Got: ${logged_in_role}"
                return 1
            fi
        else
            log_error "✗ User login failed: ${email}"
            return 1
        fi
    done

    return 0
}

# Test: Super Admin can create users in any tenant
test_super_admin_create_users() {
    log_info "Testing Super Admin user creation capabilities"

    # Login as Super Admin
    if ! login_user "superadmin@smartticket.system" "admin123" "test.smartticket.com"; then
        log_error "Failed to login as Super Admin"
        return 1
    fi

    local timestamp=$(date +%s)
    local test_email="superadmin_user_${timestamp}@test.com"

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "CreateUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "email": "${test_email}",
  "username": "superadmin_user_${timestamp}",
  "full_name": "Super Admin Created User ${timestamp}",
  "password": "testpass123",
  "role": "USER_ROLE_SUPPORT_ENGINEER"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local user_id
        user_id=$(extract_json_field "${response}" "user.id")
        local user_role
        user_role=$(extract_json_field "${response}" "user.role")

        log_success "✓ Super Admin successfully created user"
        log_info "  User ID: ${user_id}"
        log_info "  Role: ${user_role}"

        return 0
    else
        log_error "✗ Super Admin failed to create user"
        return 1
    fi
}

# Cleanup test data
cleanup_test_users() {
    log_info "Cleaning up test users"

    # Login as tenant admin for cleanup
    if login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        for user_id in "${CREATED_USER_IDS[@]}"; do
            local response
            response=$(make_grpc_call "smartticket.v1.UserService" "DeleteUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "userId": "${user_id}"
}
EOF)")

            if [[ $? -eq 0 ]]; then
                log_success "✓ Test user deleted: ${user_id}"
            else
                log_warning "Failed to delete test user: ${user_id}"
            fi
        done
    fi
}

# Main test execution
main() {
    log_info "Starting User Management E2E Tests"
    log_info "=================================="

    # Run tests
    test_create_users || log_error "User creation tests failed"
    test_list_users || log_error "User listing tests failed"
    test_get_user || log_error "Get user tests failed"
    test_update_user || log_error "User update tests failed"
    test_update_user_status || log_error "User status update tests failed"
    test_get_user_permissions || log_error "User permissions tests failed"
    test_user_role_login || log_error "User role login tests failed"
    test_super_admin_create_users || log_error "Super Admin user creation tests failed"

    # Cleanup
    cleanup_test_users

    log_success "User Management E2E Tests completed"
}

# Run tests if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi