#!/bin/bash

# SmartTicket Authentication Service E2E Tests
# Tests for user authentication, authorization, and session management

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Test: User Login with Valid Credentials
test_user_login_valid_credentials() {
    log_info "Testing user login with valid credentials"

    local response
    response=$(make_grpc_call "smartticket.v1.AuthService" "Login" "$(cat <<EOF
{
  "email": "${TEST_ADMIN_EMAIL}",
  "password": "${TEST_ADMIN_PASSWORD}",
  "tenant_domain": "${TEST_TENANT_DOMAIN}",
  "remember_me": false
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local access_token
        local refresh_token
        local user_id
        local email

        access_token=$(extract_json_field "${response}" "accessToken")
        refresh_token=$(extract_json_field "${response}" "refreshToken")
        user_id=$(extract_json_field "${response}" "user.id")
        email=$(extract_json_field "${response}" "user.email")

        assert_not_empty "${access_token}" "Access token should not be empty" || return 1
        assert_not_empty "${refresh_token}" "Refresh token should not be empty" || return 1
        assert_not_empty "${user_id}" "User ID should not be empty" || return 1
        assert_equals "${email}" "${TEST_ADMIN_EMAIL}" "Email should match" || return 1

        log_success "Login with valid credentials successful"
        return 0
    else
        log_error "Login with valid credentials failed"
        return 1
    fi
}

# Test: User Login with Invalid Password
test_user_login_invalid_password() {
    log_info "Testing user login with invalid password"

    local response
    response=$(make_grpc_call "smartticket.v1.AuthService" "Login" "$(cat <<EOF
{
  "email": "${TEST_ADMIN_EMAIL}",
  "password": "wrongpassword",
  "tenant_domain": "${TEST_TENANT_DOMAIN}",
  "remember_me": false
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local message_field
        message_field=$(extract_json_field "${response}" "response.message")

        # Check that the response contains an error message
        if [[ "${message_field}" == *"Invalid"* ]] || [[ "${message_field}" == *"error"* ]]; then
            log_success "Login with invalid password correctly failed"
            return 0
        else
            log_error "Login with invalid password should have failed but response was: ${response}"
            return 1
        fi
    else
        log_error "Login with invalid password failed to get response"
        return 1
    fi
}

# Test: User Login with Non-existent User
test_user_login_nonexistent_user() {
    log_info "Testing user login with non-existent user"

    local response
    response=$(make_grpc_call "smartticket.v1.AuthService" "Login" "$(cat <<EOF
{
  "email": "nonexistent@example.com",
  "password": "anypassword",
  "tenant_domain": "${TEST_TENANT_DOMAIN}",
  "remember_me": false
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local message_field
        message_field=$(extract_json_field "${response}" "response.message")

        # Check that the response contains an error message
        if [[ "${message_field}" == *"Invalid"* ]] || [[ "${message_field}" == *"error"* ]]; then
            log_success "Login with non-existent user correctly failed"
            return 0
        else
            log_error "Login with non-existent user should have failed but response was: ${response}"
            return 1
        fi
    else
        log_error "Login with non-existent user failed to get response"
        return 1
    fi
}

# Test: User Login with Invalid Tenant Domain
test_user_login_invalid_tenant() {
    log_info "Testing user login with invalid tenant domain"

    local response
    response=$(make_grpc_call "smartticket.v1.AuthService" "Login" "$(cat <<EOF
{
  "email": "${TEST_ADMIN_EMAIL}",
  "password": "${TEST_ADMIN_PASSWORD}",
  "tenant_domain": "invalid-tenant.example.com",
  "remember_me": false
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local message_field
        message_field=$(extract_json_field "${response}" "response.message")

        # Check that the response contains an error message
        if [[ "${message_field}" == *"Invalid"* ]] || [[ "${message_field}" == *"error"* ]]; then
            log_success "Login with invalid tenant domain correctly failed"
            return 0
        else
            log_error "Login with invalid tenant domain should have failed but response was: ${response}"
            return 1
        fi
    else
        log_error "Login with invalid tenant domain failed to get response"
        return 1
    fi
}

# Test: Token Refresh
test_token_refresh() {
    log_info "Testing token refresh"

    # First login to get tokens
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local old_access_token="${TEST_ACCESS_TOKEN}"
    local old_refresh_token="${TEST_REFRESH_TOKEN}"

    # Wait a moment to ensure different tokens
    sleep 2

    # Refresh the token
    if refresh_token; then
        # Verify new tokens are different
        if [[ "${TEST_ACCESS_TOKEN}" != "${old_access_token}" ]]; then
            log_success "Access token was refreshed correctly"
            return 0
        else
            log_error "Access token was not refreshed"
            return 1
        fi
    else
        log_error "Token refresh failed"
        return 1
    fi
}

# Test: Token Refresh with Invalid Token
test_token_refresh_invalid_token() {
    log_info "Testing token refresh with invalid token"

    local response
    response=$(make_grpc_call "smartticket.v1.AuthService" "RefreshToken" "$(cat <<EOF
{
  "refresh_token": "invalid-refresh-token"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local message_field
        message_field=$(extract_json_field "${response}" "response.message")

        # Check that the response contains an error message
        if [[ "${message_field}" == *"Invalid"* ]] || [[ "${message_field}" == *"error"* ]] || [[ "${message_field}" == *"Unauthenticated"* ]]; then
            log_success "Token refresh with invalid token correctly failed"
            return 0
        else
            log_error "Token refresh with invalid token should have failed but response was: ${response}"
            return 1
        fi
    else
        log_error "Token refresh with invalid token failed to get response"
        return 1
    fi
}

# Test: Create User
test_create_user() {
    log_info "Testing user creation"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local timestamp=$(date +%s)
    local test_email="testuser-${timestamp}@test.smartticket.com"
    local test_username="testuser-${timestamp}"

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "CreateUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "email": "${test_email}",
  "username": "${test_username}",
  "full_name": "Test User ${timestamp}",
  "password": "testpass123",
  "role": "USER_ROLE_CUSTOMER_USER",
  "phone": "+1234567890",
  "timezone": "UTC",
  "language": "en"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local user_id
        local email
        local username

        user_id=$(extract_json_field "${response}" "user.id")
        email=$(extract_json_field "${response}" "user.email")
        username=$(extract_json_field "${response}" "user.username")

        assert_not_empty "${user_id}" "Created user ID should not be empty" || return 1
        assert_equals "${email}" "${test_email}" "Created user email should match" || return 1
        assert_equals "${username}" "${test_username}" "Created user username should match" || return 1

        CURRENT_USER_ID="${user_id}"
        log_success "User creation successful"
        return 0
    else
        log_error "User creation failed"
        return 1
    fi
}

# Test: Get Current User
test_get_current_user() {
    log_info "Testing get current user"

    # Login first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "GetCurrentUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}")
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local user_id
        local email
        local full_name

        user_id=$(extract_json_field "${response}" "user.id")
        email=$(extract_json_field "${response}" "user.email")
        full_name=$(extract_json_field "${response}" "user.fullName")

        assert_equals "${user_id}" "${TEST_USER_ID}" "Current user ID should match" || return 1
        assert_equals "${email}" "${TEST_ADMIN_EMAIL}" "Current user email should match" || return 1
        assert_not_empty "${full_name}" "Current user full name should not be empty" || return 1

        log_success "Get current user successful"
        return 0
    else
        log_error "Get current user failed"
        return 1
    fi
}

# Test: Update Current User Profile
test_update_current_user() {
    log_info "Testing update current user profile"

    # Login first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local new_full_name="Updated Admin Name $(date +%s)"
    local new_phone="+9876543210"

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "UpdateCurrentUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "full_name": "${new_full_name}",
  "phone": "${new_phone}",
  "timezone": "UTC",
  "language": "en"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local updated_full_name
        updated_full_name=$(extract_json_field "${response}" "profile.fullName")

        assert_equals "${updated_full_name}" "${new_full_name}" "Updated full name should match" || return 1

        log_success "Update current user profile successful"
        return 0
    else
        log_error "Update current user profile failed"
        return 1
    fi
}

# Test: Change Password
test_change_password() {
    log_info "Testing password change"

    # Login first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local new_password="newtestpass123"

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "ChangePassword" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "current_password": "${TEST_ADMIN_PASSWORD}",
  "new_password": "${new_password}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        log_success "Password change successful"

        # Try to login with new password
        logout_user
        if login_user "${TEST_ADMIN_EMAIL}" "${new_password}" "${TEST_TENANT_DOMAIN}"; then
            log_success "Login with new password successful"

            # Change back to original password for other tests
            make_grpc_call "smartticket.v1.UserService" "ChangePassword" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "current_password": "${new_password}",
  "new_password": "${TEST_ADMIN_PASSWORD}"
}
EOF
)" >/dev/null 2>&1

            return 0
        else
            log_error "Login with new password failed"
            return 1
        fi
    else
        log_error "Password change failed"
        return 1
    fi
}

# Test: Get User Permissions
test_get_user_permissions() {
    log_info "Testing get user permissions"

    # Login first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "GetUserPermissions" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "user_id": "${TEST_USER_ID}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local permissions_count
        permissions_count=$(echo "${response}" | jq '.permissions | length' 2>/dev/null || echo "0")

        # Admin should have permissions
        if [[ ${permissions_count} -gt 0 ]]; then
            log_success "Get user permissions successful - found ${permissions_count} permissions"
            return 0
        else
            log_error "User should have permissions but none found"
            return 1
        fi
    else
        log_error "Get user permissions failed"
        return 1
    fi
}

# Test: Access Protected Resource Without Token
test_access_without_token() {
    log_info "Testing access to protected resource without token"

    # Ensure no token is set
    local old_token="${TEST_ACCESS_TOKEN}"
    TEST_ACCESS_TOKEN=""

    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "GetCurrentUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}")
}
EOF
)" "false")

    # Restore token
    TEST_ACCESS_TOKEN="${old_token}"

    if [[ $? -eq 0 ]]; then
        log_success "Access without token correctly failed"
        return 0
    else
        log_error "Access without token should have failed"
        return 1
    fi
}

# Main test execution function
run_auth_tests() {
    log_info "Starting Authentication Service E2E Tests"
    log_info "=========================================="

    local tests=(
        "test_user_login_valid_credentials"
        "test_user_login_invalid_password"
        "test_user_login_nonexistent_user"
        "test_user_login_invalid_tenant"
        "test_token_refresh"
        "test_token_refresh_invalid_token"
        "test_create_user"
        "test_get_current_user"
        "test_update_current_user"
        "test_change_password"
        "test_get_user_permissions"
        "test_access_without_token"
    )

    run_test_suite "Authentication Service Tests" "${tests[@]}"
}

# Run tests if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_auth_tests
fi