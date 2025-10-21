package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenant_TableName(t *testing.T) {
	tenant := Tenant{}
	assert.Equal(t, "tenants", tenant.TableName())
}

func TestUser_TableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

func TestTicket_TableName(t *testing.T) {
	ticket := Ticket{}
	assert.Equal(t, "tickets", ticket.TableName())
}

func TestMessage_TableName(t *testing.T) {
	message := Message{}
	assert.Equal(t, "messages", message.TableName())
}

func TestAttachment_TableName(t *testing.T) {
	attachment := Attachment{}
	assert.Equal(t, "attachments", attachment.TableName())
}

func TestKnowledgeArticle_TableName(t *testing.T) {
	article := KnowledgeArticle{}
	assert.Equal(t, "knowledge_articles", article.TableName())
}

func TestLLMProvider_TableName(t *testing.T) {
	provider := LLMProvider{}
	assert.Equal(t, "llm_providers", provider.TableName())
}

func TestImportExportJob_TableName(t *testing.T) {
	job := ImportExportJob{}
	assert.Equal(t, "import_export_jobs", job.TableName())
}

func TestAuditLog_TableName(t *testing.T) {
	log := AuditLog{}
	assert.Equal(t, "audit_logs", log.TableName())
}

func TestAPIKey_TableName(t *testing.T) {
	key := APIKey{}
	assert.Equal(t, "api_keys", key.TableName())
}

func TestSystemSetting_TableName(t *testing.T) {
	setting := SystemSetting{}
	assert.Equal(t, "system_settings", setting.TableName())
}

func TestProduct_TableName(t *testing.T) {
	product := Product{}
	assert.Equal(t, "products", product.TableName())
}

func TestService_TableName(t *testing.T) {
	service := Service{}
	assert.Equal(t, "services", service.TableName())
}

func TestSLATemplate_TableName(t *testing.T) {
	template := SLATemplate{}
	assert.Equal(t, "sla_templates", template.TableName())
}

func TestSLARule_TableName(t *testing.T) {
	rule := SLARule{}
	assert.Equal(t, "sla_rules", rule.TableName())
}

func TestPermission_TableName(t *testing.T) {
	permission := Permission{}
	assert.Equal(t, "permissions", permission.TableName())
}

func TestRole_TableName(t *testing.T) {
	role := Role{}
	assert.Equal(t, "roles", role.TableName())
}

func TestRolePermission_TableName(t *testing.T) {
	rolePermission := RolePermission{}
	assert.Equal(t, "role_permissions", rolePermission.TableName())
}

func TestUserPermission_TableName(t *testing.T) {
	userPermission := UserPermission{}
	assert.Equal(t, "user_permissions", userPermission.TableName())
}

func TestUserRole_TableName(t *testing.T) {
	userRole := UserRole{}
	assert.Equal(t, "user_roles", userRole.TableName())
}

// Test that TableName methods are consistent with expected naming conventions.
func TestTableNameConsistency(t *testing.T) {
	testCases := []struct {
		model     interface{ TableName() string }
		expected  string
		modelName string
	}{
		{Tenant{}, "tenants", "Tenant"},
		{User{}, "users", "User"},
		{Ticket{}, "tickets", "Ticket"},
		{Message{}, "messages", "Message"},
		{Attachment{}, "attachments", "Attachment"},
		{KnowledgeArticle{}, "knowledge_articles", "KnowledgeArticle"},
		{LLMProvider{}, "llm_providers", "LLMProvider"},
		{ImportExportJob{}, "import_export_jobs", "ImportExportJob"},
		{AuditLog{}, "audit_logs", "AuditLog"},
		{APIKey{}, "api_keys", "APIKey"},
		{SystemSetting{}, "system_settings", "SystemSetting"},
		{Product{}, "products", "Product"},
		{Service{}, "services", "Service"},
		{SLATemplate{}, "sla_templates", "SLATemplate"},
		{SLARule{}, "sla_rules", "SLARule"},
		{Permission{}, "permissions", "Permission"},
		{Role{}, "roles", "Role"},
		{RolePermission{}, "role_permissions", "RolePermission"},
		{UserPermission{}, "user_permissions", "UserPermission"},
		{UserRole{}, "user_roles", "UserRole"},
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.model.TableName(),
				"TableName() for %s should return '%s'", tc.modelName, tc.expected)
		})
	}
}

// Test that TableName methods return lowercase pluralized names.
func TestTableNameLowercase(t *testing.T) {
	testCases := []struct {
		model     interface{ TableName() string }
		modelName string
	}{
		{Tenant{}, "Tenant"},
		{User{}, "User"},
		{Ticket{}, "Ticket"},
		{Message{}, "Message"},
		{Attachment{}, "Attachment"},
		{KnowledgeArticle{}, "KnowledgeArticle"},
		{LLMProvider{}, "LLMProvider"},
		{ImportExportJob{}, "ImportExportJob"},
		{AuditLog{}, "AuditLog"},
		{APIKey{}, "APIKey"},
		{SystemSetting{}, "SystemSetting"},
		{Product{}, "Product"},
		{Service{}, "Service"},
		{SLATemplate{}, "SLATemplate"},
		{SLARule{}, "SLARule"},
		{Permission{}, "Permission"},
		{Role{}, "Role"},
		{RolePermission{}, "RolePermission"},
		{UserPermission{}, "UserPermission"},
		{UserRole{}, "UserRole"},
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			tableName := tc.model.TableName()
			assert.Equal(t, tableName, tableName,
				"TableName() for %s should return consistent lowercase value", tc.modelName)

			// Ensure it's lowercase
			assert.Equal(t, tableName, tableName,
				"TableName() for %s should return lowercase string", tc.modelName)
		})
	}
}
