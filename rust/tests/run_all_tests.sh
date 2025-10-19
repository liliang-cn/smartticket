#!/bin/bash

echo "🧪 SmartTicket 统一测试运行器"
echo "================================"

cd "$(dirname "$0")/.."

# 测试计数
TOTAL=0
PASSED=0
FAILED=0

run_test() {
    local name="$1"
    local cmd="$2"

    echo ""
    echo "🧪 $name"
    echo "----------------------------------------"

    ((TOTAL++))

    if eval "$cmd"; then
        echo "✅ $name 通过"
        ((PASSED++))
        return 0
    else
        echo "❌ $name 失败"
        ((FAILED++))
        return 1
    fi
}

# 测试1: Rust单元测试
run_test "Rust单元测试" "cargo test --all --lib --quiet"

# 测试2: 集成测试  
run_test "集成测试" "cargo test --all --bins --quiet"

# 测试3: grpcurl全部68个接口测试
run_test "grpcurl全部68个接口测试" "bash tests/grpc/grpcurl_all_68.sh"

# 测试4: E2E综合测试
run_test "E2E综合测试" "bash tests/e2e/final_100_percent_test.sh"

# 结果
echo ""
echo "================================"
echo "📊 测试结果"
echo "================================"
echo "总计: $TOTAL"
echo -e "通过: \033[0;32m$PASSED\033[0m"
echo -e "失败: \033[0;31m$FAILED\033[0m"

if [ $FAILED -eq 0 ]; then
    echo -e "成功率: \033[0;32m100%\033[0m"
    echo "🎉 所有测试通过！"
else
    SUCCESS_RATE=$((PASSED * 100 / TOTAL))
    echo -e "成功率: \033[0;31m$SUCCESS_RATE%\033[0m"
fi

exit $FAILED
