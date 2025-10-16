#!/bin/bash

# Complete tenant setup by creating admin users for the test tenants
# This script should be run after setup_test_tenants.sh

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Load tenant information
if [[ ! -f "/tmp/test_tenants.env" ]]; then
    log_error "Test tenant information not found. Run setup_test_tenants.sh first."
    exit 1
fi

source "/tmp/test_tenants.env"

# Function to complete tenant setup by creating admin user
complete_tenant_setup() {
    local tenant_id="$1"
    local tenant_domain="$2"
    local admin_email="$3"
    local admin_password="$4"
    local setup_token="$5"
    local tenant_name="$6"

    log_info "Completing setup for tenant: ${tenant_name} (${tenant_domain})"
    log_info "Admin user: ${admin_email}"

    # Use the tenant setup token to create the admin user
    # This would typically be done via a tenant setup endpoint
    # For now, we'll create the admin user directly as Super Admin

    # Login as Super Admin
    if ! login_user "superadmin@smartticket.system" "admin123" "test.smartticket.com"; then
        log_error "Failed to login as Super Admin"
        return 1
    fi

    # Create admin user in the tenant
    local response
    response=$(make_grpc_call "smartticket.v1.UserService" "CreateUser" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${tenant_id}" "${TEST_USER_ID}"),
  "email": "${admin_email}",
  "username": "admin",
  "full_name": "Tenant Administrator",
  "password": "${admin_password}",
  "role": "USER_ROLE_TENANT_ADMIN"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local user_id
        user_id=$(extract_json_field "${response}" "user.id")
        log_success "✓ Admin user created successfully - ID: ${user_id}"

        # Update the tenant info file with user ID
        local tenant_var_name=$(echo "${tenant_name}" | tr '[:lower:]' '[:upper:]' | tr ' ' '_')
        echo "TENANT_${tenant_var_name}_USER_ID=${user_id}" >> "/tmp/test_tenants.env"

        return 0
    else
        log_error "✗ Failed to create admin user for ${tenant_name}"
        return 1
    fi
}

# Main completion function
complete_all_tenants() {
    log_info "Completing tenant setup for admin users"
    log_info "======================================="

    # Complete Tenant A setup
    if [[ -n "${TENANT_TENANT_A_TEST_COMPANY_ID}" ]]; then
        if complete_tenant_setup \
            "${TENANT_TENANT_A_TEST_COMPANY_ID}" \
            "${TENANT_TENANT_A_TEST_COMPANY_DOMAIN}" \
            "${TENANT_TENANT_A_TEST_COMPANY_EMAIL}" \
            "${TENANT_TENANT_A_TEST_COMPANY_PASSWORD}" \
            "${TENANT_TENANT_A_TEST_COMPANY_SETUP_TOKEN}" \
            "Tenant A Test Company"; then
            log_success "✓ Tenant A setup completed"
        else
            log_error "✗ Failed to complete Tenant A setup"
            return 1
        fi
    else
        log_error "✗ Tenant A information not found"
        return 1
    fi

    # Complete Tenant B setup
    if [[ -n "${TENANT_TENANT_B_TEST_COMPANY_ID}" ]]; then
        if complete_tenant_setup \
            "${TENANT_TENANT_B_TEST_COMPANY_ID}" \
            "${TENANT_TENANT_B_TEST_COMPANY_DOMAIN}" \
            "${TENANT_TENANT_B_TEST_COMPANY_EMAIL}" \
            "${TENANT_TENANT_B_TEST_COMPANY_PASSWORD}" \
            "${TENANT_TENANT_B_TEST_COMPANY_SETUP_TOKEN}" \
            "Tenant B Test Company"; then
            log_success "✓ Tenant B setup completed"
        else
            log_error "✗ Failed to complete Tenant B setup"
            return 1
        fi
    else
        log_error "✗ Tenant B information not found"
        return 1
    fi

    log_success "✓ All tenant setups completed"
    log_info "Updated tenant information:"
    cat "/tmp/test_tenants.env"

    return 0
}

# Run completion if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    complete_all_tenants
fi