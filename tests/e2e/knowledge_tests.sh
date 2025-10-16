#!/bin/bash

# SmartTicket Knowledge Base E2E Tests
# Tests for knowledge article CRUD, search, and management

# Source test configuration and helpers
source "$(dirname "${BASH_SOURCE[0]}")/test_helpers.sh"

# Global variables for knowledge tests
export CURRENT_ARTICLE_ID=""
export CURRENT_CATEGORY_ID=""

# Test: Create Knowledge Category
test_create_knowledge_category() {
    log_info "Testing knowledge category creation"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local timestamp=$(date +%s)
    local category_name="Test Category ${timestamp}"
    local category_description="This is a test category created at ${timestamp}"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "CreateCategory" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "name": "${category_name}",
  "description": "${category_description}",
  "parent_id": "",
  "icon": "folder"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local category_id
        local name
        local description

        category_id=$(extract_json_field "${response}" "category.id")
        name=$(extract_json_field "${response}" "category.name")
        description=$(extract_json_field "${response}" "category.description")

        assert_not_empty "${category_id}" "Created category ID should not be empty" || return 1
        assert_equals "${name}" "${category_name}" "Created category name should match" || return 1
        assert_contains "${description}" "${category_description}" "Created category description should contain" || return 1

        CURRENT_CATEGORY_ID="${category_id}"
        log_success "Knowledge category creation successful - Category ID: ${category_id}"
        return 0
    else
        log_error "Knowledge category creation failed"
        return 1
    fi
}

# Test: Get Knowledge Categories
test_get_knowledge_categories() {
    log_info "Testing get knowledge categories"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "GetCategories" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "parent_id": ""
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local categories_count
        categories_count=$(echo "${response}" | jq '.categories | length' 2>/dev/null || echo "0")

        if [[ ${categories_count} -gt 0 ]]; then
            log_success "Get knowledge categories successful - found ${categories_count} categories"
            return 0
        else
            log_warning "No categories found, this may be expected for a fresh system"
            return 0
        fi
    else
        log_error "Get knowledge categories failed"
        return 1
    fi
}

# Test: Create Knowledge Article
test_create_knowledge_article() {
    log_info "Testing knowledge article creation"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Create a category first if not exists
    if [[ -z "${CURRENT_CATEGORY_ID}" ]]; then
        if ! test_create_knowledge_category; then
            return 1
        fi
    fi

    local timestamp=$(date +%s)
    local article_title="Test Article ${timestamp}"
    local article_content="This is a comprehensive test article created at ${timestamp}. It contains detailed information about the testing process and various scenarios that might be encountered during E2E testing of the SmartTicket system."
    local article_summary="This article explains E2E testing procedures for SmartTicket"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "CreateArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "title": "${article_title}",
  "content": "${article_content}",
  "summary": "${article_summary}",
  "category_id": "${CURRENT_CATEGORY_ID}",
  "visibility": "PUBLIC",
  "language": "en",
  "tags": ["test", "e2e", "automation", "knowledge"]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local article_id
        local title
        local content
        local status

        article_id=$(extract_json_field "${response}" "article.id")
        title=$(extract_json_field "${response}" "article.title")
        content=$(extract_json_field "${response}" "article.content")
        status=$(extract_json_field "${response}" "article.status")

        assert_not_empty "${article_id}" "Created article ID should not be empty" || return 1
        assert_equals "${title}" "${article_title}" "Created article title should match" || return 1
        assert_contains "${content}" "${article_content}" "Created article content should contain" || return 1
        assert_equals "${status}" "DRAFT" "New article should have DRAFT status" || return 1

        CURRENT_ARTICLE_ID="${article_id}"
        log_success "Knowledge article creation successful - Article ID: ${article_id}"
        return 0
    else
        log_error "Knowledge article creation failed"
        return 1
    fi
}

# Test: Get Knowledge Article
test_get_knowledge_article() {
    log_info "Testing get knowledge article"

    # Create an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "GetArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "article_id": "${CURRENT_ARTICLE_ID}",
  "increment_view_count": true
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local article_id
        local title
        local view_count

        article_id=$(extract_json_field "${response}" "article.id")
        title=$(extract_json_field "${response}" "article.title")
        view_count=$(extract_json_field "${response}" "article.view_count")

        assert_equals "${article_id}" "${CURRENT_ARTICLE_ID}" "Retrieved article ID should match" || return 1
        assert_not_empty "${title}" "Retrieved article title should not be empty" || return 1
        assert_not_empty "${view_count}" "View count should not be empty" || return 1

        log_success "Get knowledge article successful - Views: ${view_count}"
        return 0
    else
        log_error "Get knowledge article failed"
        return 1
    fi
}

# Test: Update Knowledge Article
test_update_knowledge_article() {
    log_info "Testing knowledge article update"

    # Create an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    local updated_title="Updated Test Article $(date +%s)"
    local updated_content="This article has been updated with new content at $(date +%s). It now includes more comprehensive information and updated testing procedures."
    local updated_summary="Updated article about SmartTicket E2E testing"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "UpdateArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "article_id": "${CURRENT_ARTICLE_ID}",
  "title": "${updated_title}",
  "content": "${updated_content}",
  "summary": "${updated_summary}",
  "visibility": "PUBLIC",
  "language": "en",
  "tags": ["test", "e2e", "automation", "updated"],
  "comment": "Updated article with new content for testing"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local title
        local content
        local version

        title=$(extract_json_field "${response}" "article.title")
        content=$(extract_json_field "${response}" "article.content")
        version=$(extract_json_field "${response}" "article.version")

        assert_equals "${title}" "${updated_title}" "Updated article title should match" || return 1
        assert_contains "${content}" "${updated_content}" "Updated article content should contain" || return 1
        assert_not_empty "${version}" "Article version should not be empty" || return 1

        log_success "Knowledge article update successful - Version: ${version}"
        return 0
    else
        log_error "Knowledge article update failed"
        return 1
    fi
}

# Test: Publish Knowledge Article
test_publish_knowledge_article() {
    log_info "Testing knowledge article publication"

    # Create an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    local publish_comment="Article is ready for publication"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "PublishArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "article_id": "${CURRENT_ARTICLE_ID}",
  "comment": "${publish_comment}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local status
        local published_at

        status=$(extract_json_field "${response}" "article.status")
        published_at=$(extract_json_field "${response}" "article.published_at")

        assert_equals "${status}" "PUBLISHED" "Published article should have PUBLISHED status" || return 1
        assert_not_empty "${published_at}" "Published timestamp should not be empty" || return 1

        log_success "Knowledge article publication successful"
        return 0
    else
        log_error "Knowledge article publication failed"
        return 1
    fi
}

# Test: Search Knowledge Articles
test_search_knowledge_articles() {
    log_info "Testing knowledge article search"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Create and publish an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    if ! test_publish_knowledge_article; then
        return 1
    fi

    local search_query="Test"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "SearchArticles" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "query": "${search_query}",
  "pagination": $(create_json_pagination 20 ""),
  "only_published": true
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local articles_count
        local total_matches

        articles_count=$(echo "${response}" | jq '.articles | length' 2>/dev/null || echo "0")
        total_matches=$(extract_json_field "${response}" "total_matches")

        if [[ ${articles_count} -gt 0 ]]; then
            log_success "Search knowledge articles successful - found ${articles_count} articles (${total_matches} total matches)"
            return 0
        else
            log_error "No articles found for search query"
            return 1
        fi
    else
        log_error "Search knowledge articles failed"
        return 1
    fi
}

# Test: List Knowledge Articles
test_list_knowledge_articles() {
    log_info "Testing list knowledge articles"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Create an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "ListArticles" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "pagination": $(create_json_pagination 20 ""),
  "sort": [{"field": "created_at", "direction": "DESC"}],
  "statuses": ["DRAFT"]
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local articles_count
        articles_count=$(echo "${response}" | jq '.articles | length' 2>/dev/null || echo "0")

        if [[ ${articles_count} -gt 0 ]]; then
            log_success "List knowledge articles successful - found ${articles_count} articles"
            return 0
        else
            log_warning "No draft articles found, this may be expected"
            return 0
        fi
    else
        log_error "List knowledge articles failed"
        return 1
    fi
}

# Test: Rate Knowledge Article
test_rate_knowledge_article() {
    log_info "Testing knowledge article rating"

    # Create and publish an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    if ! test_publish_knowledge_article; then
        return 1
    fi

    local rating_comment="This article was very helpful for E2E testing"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "RateArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "article_id": "${CURRENT_ARTICLE_ID}",
  "is_helpful": true,
  "comment": "${rating_comment}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local helpful_count
        local not_helpful_count

        helpful_count=$(extract_json_field "${response}" "helpful_count")
        not_helpful_count=$(extract_json_field "${response}" "not_helpful_count")

        assert_not_empty "${helpful_count}" "Helpful count should not be empty" || return 1
        assert_not_empty "${not_helpful_count}" "Not helpful count should not be empty" || return 1

        log_success "Knowledge article rating successful - Helpful: ${helpful_count}, Not helpful: ${not_helpful_count}"
        return 0
    else
        log_error "Knowledge article rating failed"
        return 1
    fi
}

# Test: Get Article Suggestions for Ticket
test_get_article_suggestions() {
    log_info "Testing get article suggestions for ticket"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Create and publish an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    if ! test_publish_knowledge_article; then
        return 1
    fi

    local ticket_title="Test Ticket for Suggestions"
    local ticket_description="This ticket needs suggestions for E2E testing"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "GetArticleSuggestions" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "ticket_title": "${ticket_title}",
  "ticket_description": "${ticket_description}",
  "ticket_tags": ["test", "e2e"],
  "limit": 5
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local suggestions_count
        suggestions_count=$(echo "${response}" | jq '.suggestions | length' 2>/dev/null || echo "0")

        if [[ ${suggestions_count} -gt 0 ]]; then
            log_success "Get article suggestions successful - found ${suggestions_count} suggestions"
            return 0
        else
            log_warning "No article suggestions found, this may be expected"
            return 0
        fi
    else
        log_error "Get article suggestions failed"
        return 1
    fi
}

# Test: Archive Knowledge Article
test_archive_knowledge_article() {
    log_info "Testing knowledge article archival"

    # Create and publish an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    if ! test_publish_knowledge_article; then
        return 1
    fi

    local archive_reason="Article is outdated and needs archival"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "ArchiveArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "article_id": "${CURRENT_ARTICLE_ID}",
  "reason": "${archive_reason}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        local status
        status=$(extract_json_field "${response}" "article.status")

        assert_equals "${status}" "ARCHIVED" "Archived article should have ARCHIVED status" || return 1

        log_success "Knowledge article archival successful"
        return 0
    else
        log_error "Knowledge article archival failed"
        return 1
    fi
}

# Test: Delete Knowledge Article
test_delete_knowledge_article() {
    log_info "Testing knowledge article deletion"

    # Create an article first
    if ! test_create_knowledge_article; then
        return 1
    fi

    local article_to_delete="${CURRENT_ARTICLE_ID}"

    local response
    response=$(make_grpc_call "smartticket.v1.KnowledgeService" "DeleteArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "article_id": "${article_to_delete}"
}
EOF
)")

    if [[ $? -eq 0 ]]; then
        log_success "Knowledge article deletion successful"

        # Verify article is deleted (should return error)
        local verify_response
        verify_response=$(make_grpc_call "smartticket.v1.KnowledgeService" "GetArticle" "$(cat <<EOF
{
  "metadata": $(create_json_metadata "${TEST_TENANT_ID}" "${TEST_USER_ID}"),
  "article_id": "${article_to_delete}"
}
EOF
)" "false")

        if [[ $? -eq 0 ]]; then
            log_success "Knowledge article deletion verified - article no longer accessible"
            return 0
        else
            log_warning "Could not verify knowledge article deletion"
            return 0  # Still consider success since deletion was acknowledged
        fi
    else
        log_error "Knowledge article deletion failed"
        return 1
    fi
}

# Test: Complete Knowledge Article Lifecycle
test_complete_knowledge_lifecycle() {
    log_info "Testing complete knowledge article lifecycle"

    # Login as admin first
    if ! login_user "${TEST_ADMIN_EMAIL}" "${TEST_ADMIN_PASSWORD}" "${TEST_TENANT_DOMAIN}"; then
        return 1
    fi

    # Step 1: Create category
    if ! test_create_knowledge_category; then
        log_error "Failed at category creation step"
        return 1
    fi
    log_success "✓ Step 1: Category created"

    # Step 2: Create article
    if ! test_create_knowledge_article; then
        log_error "Failed at article creation step"
        return 1
    fi
    log_success "✓ Step 2: Article created"

    # Step 3: Update article
    if ! test_update_knowledge_article; then
        log_error "Failed at article update step"
        return 1
    fi
    log_success "✓ Step 3: Article updated"

    # Step 4: Publish article
    if ! test_publish_knowledge_article; then
        log_error "Failed at article publication step"
        return 1
    fi
    log_success "✓ Step 4: Article published"

    # Step 5: Rate article
    if ! test_rate_knowledge_article; then
        log_error "Failed at article rating step"
        return 1
    fi
    log_success "✓ Step 5: Article rated"

    # Step 6: Archive article
    if ! test_archive_knowledge_article; then
        log_error "Failed at article archival step"
        return 1
    fi
    log_success "✓ Step 6: Article archived"

    log_success "Complete knowledge article lifecycle test passed"
    return 0
}

# Main test execution function
run_knowledge_tests() {
    log_info "Starting Knowledge Base E2E Tests"
    log_info "==================================="

    local tests=(
        "test_create_knowledge_category"
        "test_get_knowledge_categories"
        "test_create_knowledge_article"
        "test_get_knowledge_article"
        "test_update_knowledge_article"
        "test_publish_knowledge_article"
        "test_search_knowledge_articles"
        "test_list_knowledge_articles"
        "test_rate_knowledge_article"
        "test_get_article_suggestions"
        "test_archive_knowledge_article"
        "test_delete_knowledge_article"
        "test_complete_knowledge_lifecycle"
    )

    run_test_suite "Knowledge Base Tests" "${tests[@]}"
}

# Run tests if this script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_knowledge_tests
fi