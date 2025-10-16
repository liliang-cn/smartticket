#!/bin/bash

# Setup script for multi-tenant test environment
# Creates test tenants using Super Admin credentials

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Test tenant configuration (using unique timestamps)
TIMESTAMP=$(date +%s)
TENANT_A_DOMAIN="tenant-a-${TIMESTAMP}.example.com"
TENANT_B_DOMAIN="tenant-b-${TIMESTAMP}.example.com"
TENANT_A_EMAIL="admin@${TENANT_A_DOMAIN}"
TENANT_B_EMAIL="admin@${TENANT_B_DOMAIN}"
TENANT_A_PASSWORD="tenantapass123"
TENANT_B_PASSWORD="tenantbpass123"

# Function to create a test tenant
create_test_tenant() {
    local tenant_name="$1"
    local tenant_domain="$2"
    local admin_email="$3"
    local admin_password="$4"

    log_info "Creating test tenant: ${tenant_name} (${tenant_domain})"

    # Login as Super Admin
    if ! login_user "superadmin@smartticket.system" "admin123" "test.smartticket.com"; then
        log_error "Failed to login as Super Admin"
        return 1
    fi

    # Create tenant
    local response
    response=$(make_grpc_call "smartticket.v1.TenantService" "CreateTenant" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "name": "${tenant_name}",
  "domain": "${tenant_domain}",
  "subscriptionTier": "SUBSCRIPTION_TIER_STANDARD",
  "maxUsers": 50,
  "dataResidencyRegion": "EU",
  "contactEmail": "${admin_email}"
}
EOF)")

    if [[ $? -eq 0 ]]; then
        local tenant_id
        local setup_token

        tenant_id=$(extract_json_field "${response}" "tenant.id")
        setup_token=$(extract_json_field "${response}" "setupToken")

        log_success "✓ Tenant created successfully - ID: ${tenant_id}"
        log_success "  Setup Token: ${setup_token:0:20}..."

        # Save tenant info to file for later use
        local tenant_var_name=$(echo "${tenant_name}" | tr '[:lower:]' '[:upper:]' | tr ' ' '_')
        echo "TENANT_${tenant_var_name}_ID=${tenant_id}" >> "/tmp/test_tenants.env"
        echo "TENANT_${tenant_var_name}_DOMAIN=${tenant_domain}" >> "/tmp/test_tenants.env"
        echo "TENANT_${tenant_var_name}_EMAIL=${admin_email}" >> "/tmp/test_tenants.env"
        echo "TENANT_${tenant_var_name}_PASSWORD=${admin_password}" >> "/tmp/test_tenants.env"
        echo "TENANT_${tenant_var_name}_SETUP_TOKEN=${setup_token}" >> "/tmp/test_tenants.env"

        return 0
    else
        log_error "✗ Failed to create tenant: ${tenant_name}"
        return 1
    fi
}

# Main setup function
setup_test_tenants() {
    log_info "Setting up multi-tenant test environment"
    log_info "======================================"

    # Clear any existing tenant info
    > "/tmp/test_tenants.env"

    # Create Tenant A
    if create_test_tenant "Tenant A Test Company" "${TENANT_A_DOMAIN}" "${TENANT_A_EMAIL}" "${TENANT_A_PASSWORD}"; then
        log_success "✓ Tenant A setup completed"
    else
        log_error "✗ Failed to setup Tenant A"
        return 1
    fi

    # Create Tenant B
    if create_test_tenant "Tenant B Test Company" "${TENANT_B_DOMAIN}" "${TENANT_B_EMAIL}" "${TENANT_B_PASSWORD}"; then
        log_success "✓ Tenant B setup completed"
    else
        log_error "✗ Failed to setup Tenant B"
        return 1
    fi

    log_success "✓ Test tenant setup completed"
    log_info "Tenant information saved to: /tmp/test_tenants.env"

    # Display tenant info
    cat "/tmp/test_tenants.env"

    return 0
}

# Run setup if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    setup_test_tenants
fi