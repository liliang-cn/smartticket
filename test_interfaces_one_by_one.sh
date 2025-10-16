#!/bin/bash

# 逐个接口测试 SLA 和知识库功能
# 一个接口一个接口测试，不简化

echo "🧪 SmartTicket 接口逐个测试"
echo "=============================="

# 配置
GRPC_URL="localhost:6533"
TENANT_DOMAIN="test.smartticket.com"
ADMIN_EMAIL="admin@test.smartticket.com"
ADMIN_PASSWORD="admin123"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试计数器
TEST_COUNT=0
PASS_COUNT=0

# 清理函数
cleanup() {
    echo -e "${YELLOW}🧹 清理进程...${NC}"
    pkill -f "cargo run --bin gateway" 2>/dev/null || true
    lsof -ti:6533 -ti:7218 | xargs kill -9 2>/dev/null || true
    exit 0
}

# 信号处理
trap cleanup SIGINT SIGTERM

# 启动gateway函数
start_gateway() {
    echo -e "${BLUE}🚀 启动gateway服务...${NC}"
    cleanup > /dev/null 2>&1
    sleep 2

    RUST_LOG=debug cargo run --bin gateway > gateway_debug.log 2>&1 &
    GATEWAY_PID=$!

    # 等待服务启动
    for i in {1..30}; do
        if curl -s "http://localhost:7218/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Gateway服务启动成功 (PID: $GATEWAY_PID)${NC}"
            return 0
        fi
        sleep 1
        echo -e "${YELLOW}⏳ 等待gateway启动... ($i/30)${NC}"
    done

    echo -e "${RED}❌ Gateway服务启动失败${NC}"
    echo "=== Gateway日志 ==="
    tail -20 gateway_debug.log
    return 1
}

# 获取认证token
get_auth_token() {
    echo -e "${BLUE}🔐 获取认证token...${NC}"

    AUTH_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/user.proto \
        -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\", \"tenantDomain\": \"$TENANT_DOMAIN\"}" \
        "$GRPC_URL" smartticket.v1.AuthService.Login 2>/dev/null)

    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ 认证失败${NC}"
        echo "响应: $AUTH_RESPONSE"
        return 1
    fi

    ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.accessToken')
    USER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.id')
    TENANT_ID=$(echo "$AUTH_RESPONSE" | jq -r '.user.tenantId')

    if [ "$ACCESS_TOKEN" = "null" ] || [ "$USER_ID" = "null" ] || [ "$TENANT_ID" = "null" ]; then
        echo -e "${RED}❌ 认证响应解析失败${NC}"
        echo "认证响应: $AUTH_RESPONSE"
        return 1
    fi

    echo -e "${GREEN}✅ 认证成功${NC}"
    echo "User ID: $USER_ID"
    echo "Tenant ID: $TENANT_ID"
    return 0
}

# 测试单个接口
test_interface() {
    local test_name="$1"
    local test_command="$2"
    local expected_field="$3"

    TEST_COUNT=$((TEST_COUNT + 1))
    echo ""
    echo -e "${BLUE}🧪 测试 $TEST_COUNT: $test_name${NC}"
    echo -e "${YELLOW}命令: $test_command${NC}"

    # 执行测试
    RESPONSE=$(eval "$test_command" 2>&1)
    EXIT_CODE=$?

    if [ $EXIT_CODE -ne 0 ]; then
        echo -e "${RED}❌ 命令执行失败 (退出码: $EXIT_CODE)${NC}"
        echo "错误输出: $RESPONSE"
        echo "=== Gateway日志 (最后10行) ==="
        tail -10 gateway_debug.log
        return 1
    fi

    # 检查响应是否为空
    if [ -z "$RESPONSE" ]; then
        echo -e "${RED}❌ 响应为空${NC}"
        return 1
    fi

    # 检查是否是有效的JSON
    if ! echo "$RESPONSE" | jq . > /dev/null 2>&1; then
        echo -e "${YELLOW}⚠️ 响应不是有效JSON，可能是错误信息${NC}"
        echo "原始响应: $RESPONSE"

        # 检查是否包含错误信息
        if echo "$RESPONSE" | grep -qi "error\|failed\|denied"; then
            echo -e "${RED}❌ 响应包含错误信息${NC}"
            return 1
        fi
    fi

    # 检查期望字段
    if [ -n "$expected_field" ]; then
        FIELD_VALUE=$(echo "$RESPONSE" | jq -r ".$expected_field" 2>/dev/null)
        if [ "$FIELD_VALUE" = "null" ] || [ -z "$FIELD_VALUE" ]; then
            echo -e "${RED}❌ 期望字段 '$expected_field' 为空或不存在${NC}"
            echo "完整响应: $RESPONSE | jq ."
            return 1
        else
            echo -e "${GREEN}✅ 字段 '$expected_field' 值: $FIELD_VALUE${NC}"
        fi
    fi

    echo -e "${GREEN}✅ 通过: $test_name${NC}"
    PASS_COUNT=$((PASS_COUNT + 1))
    return 0
}

# 主测试流程
main() {
    # 启动gateway
    if ! start_gateway; then
        exit 1
    fi

    # 获取认证token
    if ! get_auth_token; then
        cleanup
        exit 1
    fi

    echo ""
    echo "=============================="
    echo -e "${BLUE}🎯 开始SLA接口测试${NC}"
    echo "=============================="

    # SLA接口测试
    test_interface "创建SLA策略" \
        "grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
        -d '{\"name\": \"测试SLA策略\", \"description\": \"测试用的SLA策略\", \"responseTimeMinutes\": 60, \"resolutionTimeMinutes\": 480, \"businessHoursOnly\": true}' \
        -H \"authorization: Bearer $ACCESS_TOKEN\" \
        -H \"x-tenant-id: $TENANT_ID\" \
        -H \"x-user-id: $USER_ID\" \
        $GRPC_URL smartticket.v1.TicketService.CreateSLAPolicy" \
        "id"

    # 如果上一个测试成功，保存SLA ID用于后续测试
    if [ $? -eq 0 ]; then
        SLA_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
            -d '{"name": "测试SLA策略", "description": "测试用的SLA策略", "responseTimeMinutes": 60, "resolutionTimeMinutes": 480, "businessHoursOnly": true}' \
            -H "authorization: Bearer $ACCESS_TOKEN" \
            -H "x-tenant-id: $TENANT_ID" \
            -H "x-user-id: $USER_ID" \
            $GRPC_URL smartticket.v1.TicketService.CreateSLAPolicy 2>/dev/null)

        SLA_ID=$(echo "$SLA_RESPONSE" | jq -r '.id' 2>/dev/null)
        if [ "$SLA_ID" != "null" ] && [ -n "$SLA_ID" ]; then
            echo -e "${GREEN}✅ SLA策略创建成功，ID: $SLA_ID${NC}"

            # 测试获取SLA策略
            test_interface "获取SLA策略详情" \
                "grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
                -d '{\"id\": \"$SLA_ID\"}' \
                -H \"authorization: Bearer $ACCESS_TOKEN\" \
                -H \"x-tenant-id: $TENANT_ID\" \
                -H \"x-user-id: $USER_ID\" \
                $GRPC_URL smartticket.v1.TicketService.GetSLAPolicy" \
                "id"

            # 测试列出SLA策略
            test_interface "列出SLA策略" \
                "grpcurl -plaintext -import-path proto -proto smartticket/ticket.proto \
                -d '{}' \
                -H \"authorization: Bearer $ACCESS_TOKEN\" \
                -H \"x-tenant-id: $TENANT_ID\" \
                -H \"x-user-id: $USER_ID\" \
                $GRPC_URL smartticket.v1.TicketService.ListSLAPolicies" \
                "slaPolicies"
        fi
    fi

    echo ""
    echo "=============================="
    echo -e "${BLUE}🎯 开始知识库接口测试${NC}"
    echo "=============================="

    # 知识库接口测试
    test_interface "创建知识分类" \
        "grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
        -d '{\"name\": \"测试分类\", \"description\": \"测试用的知识分类\"}' \
        -H \"authorization: Bearer $ACCESS_TOKEN\" \
        -H \"x-tenant-id: $TENANT_ID\" \
        -H \"x-user-id: $USER_ID\" \
        $GRPC_URL smartticket.v1.KnowledgeService.CreateCategory" \
        "id"

    # 如果分类创建成功，保存分类ID
    if [ $? -eq 0 ]; then
        CATEGORY_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
            -d '{"name": "测试分类", "description": "测试用的知识分类"}' \
            -H "authorization: Bearer $ACCESS_TOKEN" \
            -H "x-tenant-id: $TENANT_ID" \
            -H "x-user-id: $USER_ID" \
            $GRPC_URL smartticket.v1.KnowledgeService.CreateCategory 2>/dev/null)

        CATEGORY_ID=$(echo "$CATEGORY_RESPONSE" | jq -r '.id' 2>/dev/null)
        if [ "$CATEGORY_ID" != "null" ] && [ -n "$CATEGORY_ID" ]; then
            echo -e "${GREEN}✅ 知识分类创建成功，ID: $CATEGORY_ID${NC}"

            # 测试创建知识文章
            test_interface "创建知识文章" \
                "grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
                -d \"{\\\"title\\\": \\\"测试文章\\\", \\\"content\\\": \\\"# 测试文章\\\\n\\\\n这是一个测试文章的内容。\\\", \\\"summary\\\": \\\"测试文章摘要\\\", \\\"categoryId\\\": \\\"$CATEGORY_ID\\\"}\" \
                -H \"authorization: Bearer $ACCESS_TOKEN\" \
                -H \"x-tenant-id: $TENANT_ID\" \
                -H \"x-user-id: $USER_ID\" \
                $GRPC_URL smartticket.v1.KnowledgeService.CreateArticle" \
                "id"

            # 如果文章创建成功，保存文章ID
            if [ $? -eq 0 ]; then
                ARTICLE_RESPONSE=$(grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
                    -d "{\"title\": \"测试文章\", \"content\": \"# 测试文章\\n\\n这是一个测试文章的内容。\", \"summary\": \"测试文章摘要\", \"categoryId\": \"$CATEGORY_ID\"}" \
                    -H "authorization: Bearer $ACCESS_TOKEN" \
                    -H "x-tenant-id: $TENANT_ID" \
                    -H "x-user-id: $USER_ID" \
                    $GRPC_URL smartticket.v1.KnowledgeService.CreateArticle 2>/dev/null)

                ARTICLE_ID=$(echo "$ARTICLE_RESPONSE" | jq -r '.id' 2>/dev/null)
                if [ "$ARTICLE_ID" != "null" ] && [ -n "$ARTICLE_ID" ]; then
                    echo -e "${GREEN}✅ 知识文章创建成功，ID: $ARTICLE_ID${NC}"

                    # 测试获取知识文章
                    test_interface "获取知识文章" \
                        "grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
                        -d '{\"id\": \"$ARTICLE_ID\"}' \
                        -H \"authorization: Bearer $ACCESS_TOKEN\" \
                        -H \"x-tenant-id: $TENANT_ID\" \
                        -H \"x-user-id: $USER_ID\" \
                        $GRPC_URL smartticket.v1.KnowledgeService.GetArticle" \
                        "id"

                    # 测试列出知识文章
                    test_interface "列出知识文章" \
                        "grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
                        -d '{\"pagination\": {\"pageSize\": 10}}' \
                        -H \"authorization: Bearer $ACCESS_TOKEN\" \
                        -H \"x-tenant-id: $TENANT_ID\" \
                        -H \"x-user-id: $USER_ID\" \
                        $GRPC_URL smartticket.v1.KnowledgeService.ListArticles" \
                        "articles"

                    # 测试搜索知识文章
                    test_interface "搜索知识文章" \
                        "grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
                        -d '{\"query\": \"测试\", \"pagination\": {\"pageSize\": 10}}' \
                        -H \"authorization: Bearer $ACCESS_TOKEN\" \
                        -H \"x-tenant-id: $TENANT_ID\" \
                        -H \"x-user-id: $USER_ID\" \
                        $GRPC_URL smartticket.v1.KnowledgeService.SearchArticles" \
                        "articles"
                fi
            fi

            # 测试获取分类列表
            test_interface "获取知识分类列表" \
                "grpcurl -plaintext -import-path proto -proto smartticket/knowledge.proto \
                -d '{}' \
                -H \"authorization: Bearer $ACCESS_TOKEN\" \
                -H \"x-tenant-id: $TENANT_ID\" \
                -H \"x-user-id: $USER_ID\" \
                $GRPC_URL smartticket.v1.KnowledgeService.GetCategories" \
                "categories"
        fi
    fi

    # 输出测试结果
    echo ""
    echo "=============================="
    echo -e "${BLUE}📊 测试结果统计${NC}"
    echo "=============================="
    echo "总测试数: $TEST_COUNT"
    echo -e "通过: ${GREEN}$PASS_COUNT${NC}"
    echo -e "失败: ${RED}$((TEST_COUNT - PASS_COUNT))${NC}"

    if [ $PASS_COUNT -eq $TEST_COUNT ]; then
        echo ""
        echo -e "${GREEN}🎉 所有接口测试通过！${NC}"
    else
        echo ""
        echo -e "${RED}❌ 部分接口测试失败${NC}"
    fi

    # 清理
    cleanup
}

# 执行主函数
main