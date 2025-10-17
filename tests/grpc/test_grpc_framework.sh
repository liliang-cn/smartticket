#!/bin/bash

# 简化的grpcurl测试框架验证脚本
# 用于验证测试框架是否正常工作

set -e

# 配置
GRPC_GATEWAY_HOST="${GRPC_GATEWAY_HOST:-localhost}"
GRPC_GATEWAY_PORT="${GRPC_GATEWAY_PORT:-50051}"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 检查grpcurl
check_grpcurl() {
    if ! command -v grpcurl &> /dev/null; then
        log_error "grpcurl not found. Please install grpcurl:"
        log_error "  brew install grpcurl"
        return 1
    fi
    log_success "grpcurl found: $(grpcurl --version)"
    return 0
}

# 检查gRPC服务
check_grpc_service() {
    log_info "Checking gRPC service at $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"

    if ! grpcurl -plaintext "$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT" list > /dev/null 2>&1; then
        log_error "Cannot connect to gRPC service at $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"
        log_error "Please ensure the gRPC gateway is running:"
        log_error "  cargo run --bin gateway"
        return 1
    fi

    log_success "gRPC service is reachable"
    return 0
}

# 列出服务
list_services() {
    log_info "Discovering available gRPC services..."
    local services
    services=$(grpcurl -plaintext "$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT" list 2>/dev/null | grep -E "^smartticket\.v1\." | sort)

    if [ -z "$services" ]; then
        log_warning "No smartticket.v1 services found"
        return 1
    fi

    echo "Found services:"
    echo "$services" | while read -r service; do
        echo "  - $service"
    done

    local service_count=$(echo "$services" | wc -l | tr -d ' ')
    log_info "Found $service_count smartticket.v1 services"
    return 0
}

# 测试简单调用
test_simple_call() {
    log_info "Testing simple gRPC call..."

    # 尝试调用TenantService的ListTenants方法
    local test_data='{
      "metadata": {
        "tenant_id": "test_tenant_123",
        "user_id": "test_user_123",
        "request_id": "test_req_123",
        "client_ip_address": "127.0.0.1",
        "user_agent": "grpcurl-test"
      },
      "pagination": {
        "page_size": 1
      }
    }'

    if grpcurl -plaintext -d "$test_data" "$GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT" "smartticket.v1.TenantService/ListTenants" > /dev/null 2>&1; then
        log_success "Basic gRPC call successful"
        return 0
    else
        log_warning "Basic gRPC call failed (service might not be fully implemented)"
        return 1
    fi
}

# 主函数
main() {
    echo "🧪 SmartTicket gRPC Framework Test"
    echo "=================================="
    echo "Target: $GRPC_GATEWAY_HOST:$GRPC_GATEWAY_PORT"
    echo ""

    local all_checks_passed=true

    # 检查grpcurl
    if ! check_grpcurl; then
        all_checks_passed=false
    fi

    echo ""

    # 检查gRPC服务
    if ! check_grpc_service; then
        all_checks_passed=false
    fi

    echo ""

    # 列出服务
    if ! list_services; then
        all_checks_passed=false
    fi

    echo ""

    # 测试简单调用
    if ! test_simple_call; then
        all_checks_passed=false
    fi

    echo ""
    echo "=================================="
    if $all_checks_passed; then
        log_success "🎉 All framework checks passed!"
        echo "The gRPC E2E test framework is ready to use."
        echo ""
        echo "To run the full gRPC test suite:"
        echo "  bash tests/grpc/grpc_e2e_test.sh"
        exit 0
    else
        log_error "❌ Some framework checks failed."
        echo "Please fix the issues before running the full test suite."
        exit 1
    fi
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi