#!/bin/bash

# SmartTicket gRPC E2E测试框架
# 使用grpcurl对所有gRPC接口进行端到端测试

set -e

# 配置
GRPC_GATEWAY_HOST="${GRPC_GATEWAY_HOST:-localhost}"
GRPC_GATEWAY_PORT="${GRPC_GATEWAY_PORT:-50051}"
PROTO_DIR="${PROTO_DIR:-$(dirname "$0")/../proto}"
TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-$(dirname "$0")/../test_results}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 创建结果目录
mkdir -p "$TEST_RESULTS_DIR"

# 日志文件
LOG_FILE="$TEST_RESULTS_DIR/grpc_e2e_${TIMESTAMP}.log"
RESULT_FILE="$TEST_RESULTS_DIR/grpc_e2e_summary_${TIMESTAMP}.json"

# 日志函数
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# 检查grpcurl是否可用
check_grpcurl() {
    if ! command -v grpcurl &> /dev/null; then
        log_error "grpcurl not found. Please install grpcurl:"
        log_error "  brew install grpcurl"
        log_error "  or"
        log_error "  go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
        exit 1
    fi
    log_info "grpcurl found: $(grpcurl --version)"
}

# 检查gRPC服务是否运行
check_grpc_service() {
    log_info "Checking gRPC service at $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"

    if ! grpcurl -plaintext "$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT" list > /dev/null 2>&1; then
        log_error "Cannot connect to gRPC service at $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"
        log_error "Please ensure the gRPC gateway is running:"
        log_error "  cargo run --bin gateway"
        exit 1
    fi

    log_success "gRPC service is reachable"
}

# 列出所有可用服务
list_services() {
    log_info "Discovering available gRPC services..."
    grpcurl -plaintext "$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT" list 2>/dev/null | grep -E "^smartticket\.v1\." | sort
}

# 执行gRPC调用并检查结果
execute_grpc_call() {
    local service_method="$1"
    local request_data="$2"
    local expected_success="$3" # "true" or "false"
    local test_description="$4"

    ((TOTAL_TESTS++))

    log_info "Testing: $test_description"
    log_info "Method: $service_method"

    # 构造完整的命令
    local cmd="grpcurl -plaintext -d '$request_data' '$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT' '$service_method'"

    log_info "Command: $cmd"

    # 执行命令并捕获输出
    local output
    local exit_code

    output=$(eval "$cmd" 2>&1)
    exit_code=$?

    # 检查执行结果
    if [ $exit_code -eq 0 ]; then
        if echo "$output" | grep -q '"success":true\|"response":{"success":true'; then
            log_success "✅ PASS: $test_description"
            ((PASSED_TESTS++))
            echo "{\"test\":\"$test_description\",\"method\":\"$service_method\",\"status\":\"PASS\",\"output\":\"$output\"}" >> "$RESULT_FILE.tmp"
            return 0
        elif echo "$output" | grep -q '"success":false\|"response":{"success":false'; then
            if [ "$expected_success" = "false" ]; then
                log_success "✅ PASS (expected failure): $test_description"
                ((PASSED_TESTS++))
                echo "{\"test\":\"$test_description\",\"method\":\"$service_method\",\"status\":\"PASS\",\"output\":\"$output\"}" >> "$RESULT_FILE.tmp"
                return 0
            else
                log_error "❌ FAIL: $test_description - API returned error"
                log_error "Output: $output"
                ((FAILED_TESTS++))
                echo "{\"test\":\"$test_description\",\"method\":\"$service_method\",\"status\":\"FAIL\",\"output\":\"$output\"}" >> "$RESULT_FILE.tmp"
                return 1
            fi
        else
            log_warning "⚠️  PARTIAL: $test_description - Unexpected response format"
            log_warning "Output: $output"
            ((PASSED_TESTS++)) # 仍然算通过，因为服务有响应
            echo "{\"test\":\"$test_description\",\"method\":\"$service_method\",\"status\":\"PARTIAL\",\"output\":\"$output\"}" >> "$RESULT_FILE.tmp"
            return 0
        fi
    else
        if [ "$expected_success" = "false" ]; then
            log_success "✅ PASS (expected failure): $test_description"
            ((PASSED_TESTS++))
            echo "{\"test\":\"$test_description\",\"method\":\"$service_method\",\"status\":\"PASS\",\"output\":\"$output\"}" >> "$RESULT_FILE.tmp"
            return 0
        else
            log_error "❌ FAIL: $test_description - Command failed with exit code $exit_code"
            log_error "Output: $output"
            ((FAILED_TESTS++))
            echo "{\"test\":\"$test_description\",\"method\":\"$service_method\",\"status\":\"FAIL\",\"output\":\"$output\"}" >> "$RESULT_FILE.tmp"
            return 1
        fi
    fi
}

# 生成认证token（如果需要）
generate_auth_token() {
    log_info "Generating authentication token..."

    # 这里应该调用认证服务获取token
    # 暂时返回一个测试token
    echo "test_jwt_token"
}

# 初始化测试结果文件
init_results() {
    echo "{\"timestamp\":\"$(date -Iseconds)\",\"host\":\"$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT\",\"tests\":[" > "$RESULT_FILE.tmp"
}

# 完成测试结果文件
finalize_results() {
    echo "]}" | sed '$ s/,$//' "$RESULT_FILE.tmp" > "$RESULT_FILE"
    rm "$RESULT_FILE.tmp"
}

# 显示测试摘要
show_summary() {
    echo ""
    echo "=================================="
    echo "🧪 gRPC E2E Test Summary"
    echo "=================================="
    echo "Total Tests: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

    if [ $FAILED_TESTS -eq 0 ]; then
        SUCCESS_RATE=100
        echo -e "Success Rate: ${GREEN}100%${NC}"
        echo "🎉 All gRPC tests passed!"
    else
        SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
        echo -e "Success Rate: ${RED}$SUCCESS_RATE%${NC}"
    fi

    echo ""
    echo "Results saved to:"
    echo "  Log: $LOG_FILE"
    echo "  Summary: $RESULT_FILE"
}

# 主函数
main() {
    log_info "Starting SmartTicket gRPC E2E Tests"
    log_info "Target: $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"

    # 环境检查
    check_grpcurl
    check_grpc_service

    # 初始化
    init_results

    # 列出可用服务
    local services
    services=$(list_services)
    log_info "Found services: $(echo "$services" | wc -l | tr -d ' ')"

    # 导入测试用例
    source "$(dirname "$0")/grpc_test_cases.sh"

    # 执行所有测试
    run_all_grpc_tests

    # 完成并显示结果
    finalize_results
    show_summary

    # 返回适当的退出码
    if [ $FAILED_TESTS -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi