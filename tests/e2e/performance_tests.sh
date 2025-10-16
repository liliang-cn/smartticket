#!/bin/bash

# SmartTicket Performance and Load E2E Tests
# Tests for system performance under load, response times, and concurrent operations

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Performance test configuration
export LOAD_TEST_USERS=${LOAD_TEST_USERS:-10}
export LOAD_TEST_DURATION=${LOAD_TEST_DURATION:-30}
export LOAD_TEST_RAMP_UP=${LOAD_TEST_RAMP_UP:-5}
export RESPONSE_TIME_THRESHOLD=${RESPONSE_TIME_THRESHOLD:-2000}  # 2 seconds in milliseconds
export CONCURRENT_OPS_THRESHOLD=${CONCURRENT_OPS_THRESHOLD:-50}   # Max concurrent operations

# Performance metrics (using simple variables for compatibility)
export PERFORMANCE_RESULTS_FILE="${TEST_RESULTS_DIR}/performance_results_$(date +%Y%m%d_%H%M%S).json"

# Store performance metrics in a temporary file for compatibility
PERFORMANCE_METRICS_FILE="${TEST_RESULTS_DIR}/performance_metrics_$$.tmp"

# Function to store performance metrics
store_performance_metric() {
    local metric_name="$1"
    local metric_value="$2"
    echo "${metric_name}=${metric_value}" >> "${PERFORMANCE_METRICS_FILE}"
}

# Function to retrieve performance metrics
get_performance_metric() {
    local metric_name="$1"
    grep "^${metric_name}=" "${PERFORMANCE_METRICS_FILE}" 2>/dev/null | cut -d'=' -f2- || echo ""
}

# Function to measure response time
measure_response_time() {
    local service="$1"
    local method="$2"
    local data="$3"
    local start_time=$(date +%s%N)  # Nanoseconds

    local response
    response=$(make_grpc_call "${service}" "${method}" "${data}")
    local exit_code=$?

    local end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds

    echo "{\"duration_ms\": ${duration}, \"success\": $([ $exit_code -eq 0 ] && echo "true" || echo "false"), \"response\": ${response}}"
}

# Function to run concurrent operations
run_concurrent_operations() {
    local operation_count="$1"
    local operation_function="$2"
    local concurrent_limit="$3"

    log_info "Running ${operation_count} concurrent operations (limit: ${concurrent_limit})"

    local pids=()
    local temp_files=()
    local start_time=$(date +%s)

    # Start concurrent operations
    for ((i=1; i<=operation_count; i++)); do
        # Limit concurrent operations
        while [[ ${#pids[@]} -ge ${concurrent_limit} ]]; do
            for j in "${!pids[@]}"; do
                if ! kill -0 "${pids[$j]}" 2>/dev/null; then
                    wait "${pids[$j]}"
                    unset "pids[$j]"
                fi
            done
            pids=("${pids[@]}")  # Reindex array
            sleep 0.1
        done

        local temp_file="${TEST_RESULTS_DIR}/op_${i}_$$.tmp"
        temp_files+=("${temp_file}")

        # Run operation in background
        (
            echo "$(${operation_function} ${i})" > "${temp_file}"
        ) &
        pids+=($!)
    done

    # Wait for all operations to complete
    for pid in "${pids[@]}"; do
        wait "${pid}"
    done

    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))

    # Collect results
    local successful_ops=0
    local failed_ops=0
    local total_response_time=0
    local max_response_time=0
    local min_response_time=999999

    for temp_file in "${temp_files[@]}"; do
        if [[ -f "${temp_file}" ]]; then
            local result
            result=$(cat "${temp_file}")
            local success
            success=$(echo "${result}" | jq -r '.success' 2>/dev/null || echo "false")
            local duration
            duration=$(echo "${result}" | jq -r '.duration_ms' 2>/dev/null || echo "0")

            if [[ "${success}" == "true" ]]; then
                ((successful_ops++))
                total_response_time=$((total_response_time + duration))

                if [[ ${duration} -gt ${max_response_time} ]]; then
                    max_response_time=${duration}
                fi

                if [[ ${duration} -lt ${min_response_time} ]]; then
                    min_response_time=${duration}
                fi
            else
                ((failed_ops++))
            fi

            rm -f "${temp_file}"
        fi
    done

    local avg_response_time=0
    if [[ ${successful_ops} -gt 0 ]]; then
        avg_response_time=$((total_response_time / successful_ops))
    fi

    log_info "Concurrent operations completed:"
    log_info "  Total: ${operation_count}, Successful: ${successful_ops}, Failed: ${failed_ops}"
    log_info "  Duration: ${total_duration}s, Throughput: $(( successful_ops / total_duration )) ops/sec"
    log_info "  Response times - Avg: ${avg_response_time}ms, Min: ${min_response_time}ms, Max: ${max_response_time}ms"

    # Store metrics
    store_performance_metric "concurrent_ops_${operation_count}" "{\"total\": ${operation_count}, \"successful\": ${successful_ops}, \"failed\": ${failed_ops}, \"duration_s\": ${total_duration}, \"avg_response_ms\": ${avg_response_time}, \"min_response_ms\": ${min_response_time}, \"max_response_ms\": ${max_response_time}}"

    return 0
}

# Function: Create ticket operation for load testing
load_test_create_ticket() {
    local iteration="$1"

    # Login first (in real scenario, tokens would be reused)
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        echo "{\"duration_ms\": 0, \"success\": false}"
        return 1
    fi

    local test_data
    test_data=$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "title": "Load Test Ticket ${iteration}",
  "description": "This is a load test ticket created at $(date +%s) with iteration ${iteration}",
  "priority": "NORMAL",
  "severity": "MEDIUM",
  "category_id": "general",
  "contact_id": "${TEST_USER_ID}",
  "tags": ["load-test", "perf-test", "iteration-${iteration}"]
}
EOF
)

    measure_response_time "smartticket.v1.TicketService" "CreateTicket" "${test_data}"
}

# Function: Get ticket operation for load testing
load_test_get_ticket() {
    local iteration="$1"

    if [[ -z "${CURRENT_TICKET_ID}" ]]; then
        # Create a ticket first
        local result
        result=$(load_test_create_ticket 1)
        local success
        success=$(echo "${result}" | jq -r '.success' 2>/dev/null || echo "false")

        if [[ "${success}" == "true" ]]; then
            CURRENT_TICKET_ID=$(echo "${result}" | jq -r '.response.ticket.id' 2>/dev/null || echo "")
        fi
    fi

    if [[ -n "${CURRENT_TICKET_ID}" ]]; then
        local test_data
        test_data=$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_id": "${CURRENT_TICKET_ID}"
}
EOF
)

        measure_response_time "smartticket.v1.TicketService" "GetTicket" "${test_data}"
    else
        echo "{\"duration_ms\": 0, \"success\": false}"
        return 1
    fi
}

# Function: List tickets operation for load testing
load_test_list_tickets() {
    local iteration="$1"

    local test_data
    test_data=$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": $(create_json_pagination 20 ""),
  "sort": [{"field": "created_at", "direction": "DESC"}]
}
EOF
)

    measure_response_time "smartticket.v1.TicketService" "ListTickets" "${test_data}"
}

# Function: Search tickets operation for load testing
load_test_search_tickets() {
    local iteration="$1"

    local test_data
    test_data=$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "query": "load test",
  "pagination": $(create_json_pagination 20 "")
}
EOF
)

    measure_response_time "smartticket.v1.TicketService" "SearchTickets" "${test_data}"
}

# Test: Authentication Performance
test_authentication_performance() {
    log_info "Testing authentication performance"

    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local iterations=10
    local total_duration=0
    local successful_logins=0

    for i in $(seq 1 ${iterations}); do
        logout_user

        local start_time=$(date +%s%N)
        if login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
            local end_time=$(date +%s%N)
            local duration=$(( (end_time - start_time) / 1000000 ))
            total_duration=$((total_duration + duration))
            ((successful_logins++))
        fi
    done

    if [[ ${successful_logins} -gt 0 ]]; then
        local avg_duration=$((total_duration / successful_logins))
        log_success "Authentication performance: ${avg_duration}ms avg over ${successful_logins}/${iterations} logins"

        store_performance_metric "auth_performance" "{\"iterations\": ${iterations}, \"successful\": ${successful_logins}, \"avg_response_ms\": ${avg_duration}}"

        if [[ ${avg_duration} -lt ${RESPONSE_TIME_THRESHOLD} ]]; then
            return 0
        else
            log_error "Authentication response time ${avg_duration}ms exceeds threshold ${RESPONSE_TIME_THRESHOLD}ms"
            return 1
        fi
    else
        log_error "No successful logins in performance test"
        return 1
    fi
}

# Test: Ticket CRUD Performance
test_ticket_crud_performance() {
    log_info "Testing ticket CRUD performance"

    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local operations=20
    local successful_ops=0
    local total_duration=0

    for i in $(seq 1 ${operations}); do
        local result
        result=$(load_test_create_ticket ${i})
        local success
        success=$(echo "${result}" | jq -r '.success' 2>/dev/null || echo "false")
        local duration
        duration=$(echo "${result}" | jq -r '.duration_ms' 2>/dev/null || echo "0")

        if [[ "${success}" == "true" ]]; then
            ((successful_ops++))
            total_duration=$((total_duration + duration))
        fi
    done

    if [[ ${successful_ops} -gt 0 ]]; then
        local avg_duration=$((total_duration / successful_ops))
        log_success "Ticket creation performance: ${avg_duration}ms avg over ${successful_ops}/${operations} operations"

        store_performance_metric "ticket_create_performance" "{\"operations\": ${operations}, \"successful\": ${successful_ops}, \"avg_response_ms\": ${avg_duration}}"

        if [[ ${avg_duration} -lt ${RESPONSE_TIME_THRESHOLD} ]]; then
            return 0
        else
            log_error "Ticket creation response time ${avg_duration}ms exceeds threshold ${RESPONSE_TIME_THRESHOLD}ms"
            return 1
        fi
    else
        log_error "No successful ticket creations in performance test"
        return 1
    fi
}

# Test: Concurrent Load Test
test_concurrent_load() {
    log_info "Testing concurrent load - ${LOAD_TEST_USERS} concurrent operations"

    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Create one ticket first for get operations
    load_test_create_ticket 1 >/dev/null

    run_concurrent_operations ${LOAD_TEST_USERS} "load_test_get_ticket" ${CONCURRENT_OPS_THRESHOLD}

    # Check if we had acceptable success rate
    local metrics
    metrics=$(get_performance_metric "concurrent_ops_${LOAD_TEST_USERS}")
    local successful
    successful=$(echo "${metrics}" | jq -r '.successful' 2>/dev/null || echo "0")
    local total
    total=$(echo "${metrics}" | jq -r '.total' 2>/dev/null || echo "0")
    local avg_response
    avg_response=$(echo "${metrics}" | jq -r '.avg_response_ms' 2>/dev/null || echo "0")

    if [[ ${total} -gt 0 ]]; then
        local success_rate=$(( successful * 100 / total ))
        log_info "Concurrent load test results:"
        log_info "  Success rate: ${success_rate}% (${successful}/${total})"
        log_info "  Average response time: ${avg_response}ms"

        if [[ ${success_rate} -ge 90 && ${avg_response} -lt ${RESPONSE_TIME_THRESHOLD} ]]; then
            log_success "Concurrent load test passed"
            return 0
        else
            log_error "Concurrent load test failed - success rate: ${success_rate}%, avg response: ${avg_response}ms"
            return 1
        fi
    else
        log_error "No operations completed in concurrent load test"
        return 1
    fi
}

# Test: Sustained Load Test
test_sustained_load() {
    log_info "Testing sustained load for ${LOAD_TEST_DURATION} seconds"

    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local end_time=$(( $(date +%s) + LOAD_TEST_DURATION ))
    local total_operations=0
    local successful_operations=0
    local start_time=$(date +%s)

    while [[ $(date +%s) -lt ${end_time} ]]; do
        # Mix of different operations
        local operations=("load_test_create_ticket" "load_test_get_ticket" "load_test_list_tickets" "load_test_search_tickets")
        local op="${operations[$(( total_operations % ${#operations[@]} ))]}"

        local result
        result=$(${op} ${total_operations})
        local success
        success=$(echo "${result}" | jq -r '.success' 2>/dev/null || echo "false")

        ((total_operations++))
        if [[ "${success}" == "true" ]]; then
            ((successful_operations++))
        fi

        # Small delay to prevent overwhelming the system
        sleep 0.1
    done

    local actual_duration=$(($(date +%s) - start_time))
    local throughput=$(( successful_operations / actual_duration ))

    log_success "Sustained load test completed:"
    log_success "  Duration: ${actual_duration}s (target: ${LOAD_TEST_DURATION}s)"
    log_success "  Operations: ${total_operations}, Successful: ${successful_operations}"
    log_success "  Throughput: ${throughput} ops/sec"

    store_performance_metric "sustained_load" "{\"duration_s\": ${actual_duration}, \"total_operations\": ${total_operations}, \"successful_operations\": ${successful_operations}, \"throughput_ops_per_sec\": ${throughput}}"

    if [[ ${throughput} -gt 5 ]]; then
        return 0
    else
        log_error "Sustained load throughput ${throughput} ops/sec below minimum threshold"
        return 1
    fi
}

# Test: Memory and Resource Usage
test_resource_usage() {
    log_info "Testing resource usage under load"

    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Get initial memory usage (if available)
    local initial_memory=0
    if command -v ps >/dev/null 2>&1; then
        # Get memory usage of the grpc server process
        local grpc_pid
        grpc_pid=$(pgrep -f "smartticket.*grpc" | head -1)
        if [[ -n "${grpc_pid}" ]]; then
            initial_memory=$(ps -o rss= -p "${grpc_pid}" 2>/dev/null || echo "0")
        fi
    fi

    # Run load test
    run_concurrent_operations 50 "load_test_get_ticket" 10

    # Get final memory usage
    local final_memory=0
    if command -v ps >/dev/null 2>&1 && [[ ${initial_memory} -gt 0 ]]; then
        local grpc_pid
        grpc_pid=$(pgrep -f "smartticket.*grpc" | head -1)
        if [[ -n "${grpc_pid}" ]]; then
            final_memory=$(ps -o rss= -p "${grpc_pid}" 2>/dev/null || echo "0")
        fi
    fi

    if [[ ${initial_memory} -gt 0 && ${final_memory} -gt 0 ]]; then
        local memory_increase=$((final_memory - initial_memory))
        local memory_increase_percent=$(( memory_increase * 100 / initial_memory ))

        log_info "Resource usage test:"
        log_info "  Initial memory: ${initial_memory}KB"
        log_info "  Final memory: ${final_memory}KB"
        log_info "  Memory increase: ${memory_increase}KB (${memory_increase_percent}%)"

        store_performance_metric "resource_usage" "{\"initial_memory_kb\": ${initial_memory}, \"final_memory_kb\": ${final_memory}, \"memory_increase_kb\": ${memory_increase}, \"memory_increase_percent\": ${memory_increase_percent}}"

        if [[ ${memory_increase_percent} -lt 50 ]]; then
            log_success "Memory usage within acceptable limits"
            return 0
        else
            log_warning "Memory usage increased by ${memory_increase_percent}% - may indicate memory leak"
            return 1
        fi
    else
        log_warning "Could not measure memory usage - test inconclusive"
        return 0
    fi
}

# Function to generate performance report
generate_performance_report() {
    log_info "Generating performance report"

    cat > "${PERFORMANCE_RESULTS_FILE}" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "test_configuration": {
    "load_test_users": ${LOAD_TEST_USERS},
    "load_test_duration": ${LOAD_TEST_DURATION},
    "response_time_threshold_ms": ${RESPONSE_TIME_THRESHOLD},
    "concurrent_ops_threshold": ${CONCURRENT_OPS_THRESHOLD}
  },
  "environment": {
    "grpc_server": "${GRPC_SERVER_ADDRESS}",
    "test_tenant": "${TEST_TENANT_DOMAIN}"
  },
  "metrics": {
EOF

    # Read metrics from the temporary file
    local first=true
    if [[ -f "${PERFORMANCE_METRICS_FILE}" ]]; then
        while IFS='=' read -r metric_name metric_value; do
            if [[ -n "${metric_name}" && -n "${metric_value}" ]]; then
                if [[ "${first}" == "true" ]]; then
                    first=false
                else
                    echo "," >> "${PERFORMANCE_RESULTS_FILE}"
                fi
                echo "    \"${metric_name}\": ${metric_value}" >> "${PERFORMANCE_RESULTS_FILE}"
            fi
        done < "${PERFORMANCE_METRICS_FILE}"
    fi

    # Cleanup temporary metrics file
    rm -f "${PERFORMANCE_METRICS_FILE}"

    cat >> "${PERFORMANCE_RESULTS_FILE}" << EOF
  }
}
EOF

    log_success "Performance report generated: ${PERFORMANCE_RESULTS_FILE}"
}

# Main test execution function
run_performance_tests() {
    log_info "Starting Performance and Load E2E Tests"
    log_info "========================================="

    local tests=(
        "test_authentication_performance"
        "test_ticket_crud_performance"
        "test_concurrent_load"
        "test_sustained_load"
        "test_resource_usage"
    )

    # Run performance tests
    local test_results=0
    for test in "${tests[@]}"; do
        if ! run_test "${test}" "${test}"; then
            ((test_results++))
        fi
    done

    # Generate performance report
    generate_performance_report

    if [[ ${test_results} -eq 0 ]]; then
        log_success "All performance tests passed"
        return 0
    else
        log_error "${test_results} performance tests failed"
        return 1
    fi
}

# Run tests if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_performance_tests
fi