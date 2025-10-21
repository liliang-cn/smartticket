package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupIntegrationDB creates a properly configured database for integration testing
func setupIntegrationDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Enable foreign key constraints for SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	// Migrate all models
	err = db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Ticket{},
		&models.Message{},
		&models.Attachment{},
		&models.KnowledgeArticle{},
		&models.LLMProvider{},
		&models.ImportExportJob{},
		&models.AuditLog{},
		&models.APIKey{},
		&models.SystemSetting{},
		&models.Product{},
		&models.Service{},
		&models.SLATemplate{},
		&models.SLARule{},
		&models.Permission{},
		&models.Role{},
		&models.RolePermission{},
		&models.UserPermission{},
		&models.UserRole{},
	)
	require.NoError(t, err)

	return db
}

// TestDatabaseServiceIntegration tests the integration between database and services
func TestDatabaseServiceIntegration(t *testing.T) {
	db := setupIntegrationDB(t)
	ctx := context.Background()
	permissionService := services.NewPermissionService(db)

	t.Run("Complete permission workflow", func(t *testing.T) {
		// Create tenant
		tenant := &models.Tenant{
			Name:     "Integration Test Tenant",
			Slug:     "integration-test-tenant",
			Plan:     "enterprise",
			MaxUsers: 1000,
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create user
		user := &models.User{
			TenantID:     tenant.ID,
			Email:        "integration@example.com",
			Username:     "integrationuser",
			FirstName:    "Integration",
			LastName:     "User",
			Role:         "admin",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}
		err = db.Create(user).Error
		require.NoError(t, err)

		// Create permissions via service
		readPermission := &models.Permission{
			Code:        "integration:read",
			Name:        "Integration Read Permission",
			Description: "Read access for integration tests",
			Category:    "integration",
			IsSystem:    false,
		}
		err = permissionService.CreatePermission(ctx, readPermission)
		require.NoError(t, err)

		writePermission := &models.Permission{
			Code:        "integration:write",
			Name:        "Integration Write Permission",
			Description: "Write access for integration tests",
			Category:    "integration",
			IsSystem:    false,
		}
		err = permissionService.CreatePermission(ctx, writePermission)
		require.NoError(t, err)

		// Create role via service
		adminRole := &models.Role{
			TenantID:    tenant.ID,
			Name:        "Integration Admin",
			Description: "Admin role for integration tests",
			IsSystem:    false,
			IsActive:    true,
		}
		err = permissionService.CreateRole(ctx, adminRole)
		require.NoError(t, err)

		// Assign permissions to role
		err = permissionService.AssignPermissionToRole(ctx, adminRole.ID, readPermission.ID)
		require.NoError(t, err)

		err = permissionService.AssignPermissionToRole(ctx, adminRole.ID, writePermission.ID)
		require.NoError(t, err)

		// Assign role to user
		err = permissionService.AssignRoleToUser(ctx, user.ID, adminRole.ID, fmt.Sprintf("%d", tenant.ID))
		require.NoError(t, err)

		// Verify user has permissions through service
		hasRead, err := permissionService.HasPermission(ctx, user.ID, fmt.Sprintf("%d", tenant.ID), "integration:read")
		require.NoError(t, err)
		assert.True(t, hasRead)

		hasWrite, err := permissionService.HasPermission(ctx, user.ID, fmt.Sprintf("%d", tenant.ID), "integration:write")
		require.NoError(t, err)
		assert.True(t, hasWrite)

		hasAdmin, err := permissionService.HasPermission(ctx, user.ID, fmt.Sprintf("%d", tenant.ID), "integration:admin")
		require.NoError(t, err)
		assert.False(t, hasAdmin)

		// Verify database state directly
		var userRoles []models.UserRole
		err = db.Where("user_id = ? AND role_id = ?", user.ID, adminRole.ID).Find(&userRoles).Error
		require.NoError(t, err)
		assert.Len(t, userRoles, 1)

		var rolePermissions []models.RolePermission
		err = db.Where("role_id = ?", adminRole.ID).Find(&rolePermissions).Error
		require.NoError(t, err)
		assert.Len(t, rolePermissions, 2)

		// Test removal of one permission
		err = permissionService.RemovePermissionFromRole(ctx, adminRole.ID, readPermission.ID)
		require.NoError(t, err)

		hasRead, err = permissionService.HasPermission(ctx, user.ID, fmt.Sprintf("%d", tenant.ID), "integration:read")
		require.NoError(t, err)
		assert.False(t, hasRead)

		// Write permission should still work
		hasWrite, err = permissionService.HasPermission(ctx, user.ID, fmt.Sprintf("%d", tenant.ID), "integration:write")
		require.NoError(t, err)
		assert.True(t, hasWrite)
	})
}

// TestTicketWorkflowIntegration tests the complete ticket workflow
func TestTicketWorkflowIntegration(t *testing.T) {
	db := setupIntegrationDB(t)

	t.Run("Complete ticket lifecycle", func(t *testing.T) {
		// Create tenant
		tenant := &models.Tenant{
			Name:     "Ticket Workflow Tenant",
			Slug:     "ticket-workflow-tenant",
			Plan:     "professional",
			MaxUsers: 500,
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create users with different roles
		customer := &models.User{
			TenantID:     tenant.ID,
			Email:        "customer@example.com",
			Username:     "customer",
			FirstName:    "Customer",
			LastName:     "User",
			Role:         "customer",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}
		err = db.Create(customer).Error
		require.NoError(t, err)

		engineer := &models.User{
			TenantID:     tenant.ID,
			Email:        "engineer@example.com",
			Username:     "engineer",
			FirstName:    "Support",
			LastName:     "Engineer",
			Role:         "engineer",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}
		err = db.Create(engineer).Error
		require.NoError(t, err)

		// Create ticket
		ticket := &models.Ticket{
			TenantID:       tenant.ID,
			TicketNumber:   "TKT-001",
			Title:          "Login Issue",
			Description:    "Customer cannot login to the system",
			Status:         "open",
			Priority:       "high",
			Severity:       "medium",
			Category:       "authentication",
			Type:           "incident",
			RequesterName:  "Customer User",
			RequesterEmail: "customer@example.com",
			Tags:           "login,urgent",
		}
		err = db.Create(ticket).Error
		require.NoError(t, err)
		assert.NotZero(t, ticket.ID)

		// Add initial message from customer
		message := &models.Message{
			TicketID:    ticket.ID,
			UserID:      customer.ID,
			Content:     "I've been trying to login for the past hour but keep getting an error message.",
			ContentType: "text",
			IsInternal:  false,
			IsFromAI:    false,
		}
		err = db.Create(message).Error
		require.NoError(t, err)

		// Assign ticket to engineer
		ticket.AssignedTo = &engineer.ID
		ticket.Status = "in_progress"
		err = db.Save(ticket).Error
		require.NoError(t, err)

		// Add message from engineer
		engineerMessage := &models.Message{
			TicketID:    ticket.ID,
			UserID:      engineer.ID,
			Content:     "I'm looking into this issue. Can you please provide the exact error message you're seeing?",
			ContentType: "text",
			IsInternal:  false,
			IsFromAI:    false,
		}
		err = db.Create(engineerMessage).Error
		require.NoError(t, err)

		// Add internal note
		internalNote := &models.Message{
			TicketID:    ticket.ID,
			UserID:      engineer.ID,
			Content:     "Customer mentioned they've been trying for an hour. This might be a high-priority issue affecting multiple users.",
			ContentType: "text",
			IsInternal:  true,
			IsFromAI:    false,
		}
		err = db.Create(internalNote).Error
		require.NoError(t, err)

		// Verify complete workflow state
		var loadedTicket models.Ticket
		err = db.Preload("Messages").Preload("AssignedUser").First(&loadedTicket, ticket.ID).Error
		require.NoError(t, err)

		assert.Equal(t, "in_progress", loadedTicket.Status)
		assert.Equal(t, "high", loadedTicket.Priority)
		assert.NotNil(t, loadedTicket.AssignedTo)
		assert.Equal(t, engineer.Username, loadedTicket.AssignedUser.Username)
		assert.Len(t, loadedTicket.Messages, 3)

		// Check message types
		publicMessages := 0
		internalMessages := 0
		for _, msg := range loadedTicket.Messages {
			if msg.IsInternal {
				internalMessages++
			} else {
				publicMessages++
			}
		}
		assert.Equal(t, 2, publicMessages)
		assert.Equal(t, 1, internalMessages)

		// Resolve ticket
		now := time.Now()
		ticket.Status = "resolved"
		ticket.ResolvedAt = &now
		ticket.ResolutionTime = &now // For test purposes, use current time as resolution time
		err = db.Save(ticket).Error
		require.NoError(t, err)

		// Verify resolution
		var resolvedTicket models.Ticket
		err = db.First(&resolvedTicket, ticket.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "resolved", resolvedTicket.Status)
		assert.NotNil(t, resolvedTicket.ResolvedAt)
		assert.NotNil(t, resolvedTicket.ResolutionTime)
	})
}

// TestMultiTenantIntegration tests multi-tenant data isolation
func TestMultiTenantIntegration(t *testing.T) {
	db := setupIntegrationDB(t)

	t.Run("Data isolation between tenants", func(t *testing.T) {
		// Create two tenants
		tenant1 := &models.Tenant{
			Name:     "Tenant 1",
			Slug:     "tenant-1",
			Plan:     "basic",
			MaxUsers: 100,
			IsActive: true,
		}
		err := db.Create(tenant1).Error
		require.NoError(t, err)

		tenant2 := &models.Tenant{
			Name:     "Tenant 2",
			Slug:     "tenant-2",
			Plan:     "enterprise",
			MaxUsers: 1000,
			IsActive: true,
		}
		err = db.Create(tenant2).Error
		require.NoError(t, err)

		// Create users for each tenant
		user1 := &models.User{
			TenantID: tenant1.ID,
			Email:    "user1@tenant1.com",
			Username: "user1",
			Role:     "admin",
			IsActive: true,
		}
		err = db.Create(user1).Error
		require.NoError(t, err)

		user2 := &models.User{
			TenantID: tenant2.ID,
			Email:    "user2@tenant2.com",
			Username: "user2",
			Role:     "admin",
			IsActive: true,
		}
		err = db.Create(user2).Error
		require.NoError(t, err)

		// Create tickets for each tenant
		ticket1 := &models.Ticket{
			TenantID:       tenant1.ID,
			TicketNumber:   "T1-001",
			Title:          "Tenant 1 Issue",
			Description:    "Issue specific to tenant 1",
			Status:         "open",
			Priority:       "medium",
			RequesterName:  "User 1",
			RequesterEmail: "user1@tenant1.com",
		}
		err = db.Create(ticket1).Error
		require.NoError(t, err)

		ticket2 := &models.Ticket{
			TenantID:       tenant2.ID,
			TicketNumber:   "T2-001",
			Title:          "Tenant 2 Issue",
			Description:    "Issue specific to tenant 2",
			Status:         "open",
			Priority:       "high",
			RequesterName:  "User 2",
			RequesterEmail: "user2@tenant2.com",
		}
		err = db.Create(ticket2).Error
		require.NoError(t, err)

		// Test data isolation: each tenant should only see their own data
		var tenant1Users []models.User
		err = db.Where("tenant_id = ?", tenant1.ID).Find(&tenant1Users).Error
		require.NoError(t, err)
		assert.Len(t, tenant1Users, 1)
		assert.Equal(t, user1.Email, tenant1Users[0].Email)

		var tenant2Users []models.User
		err = db.Where("tenant_id = ?", tenant2.ID).Find(&tenant2Users).Error
		require.NoError(t, err)
		assert.Len(t, tenant2Users, 1)
		assert.Equal(t, user2.Email, tenant2Users[0].Email)

		// Test ticket isolation
		var tenant1Tickets []models.Ticket
		err = db.Where("tenant_id = ?", tenant1.ID).Find(&tenant1Tickets).Error
		require.NoError(t, err)
		assert.Len(t, tenant1Tickets, 1)
		assert.Equal(t, ticket1.TicketNumber, tenant1Tickets[0].TicketNumber)

		var tenant2Tickets []models.Ticket
		err = db.Where("tenant_id = ?", tenant2.ID).Find(&tenant2Tickets).Error
		require.NoError(t, err)
		assert.Len(t, tenant2Tickets, 1)
		assert.Equal(t, ticket2.TicketNumber, tenant2Tickets[0].TicketNumber)

		// Verify cross-tenant access prevention
		var crossTenantTickets []models.Ticket
		err = db.Where("tenant_id = ? AND requester_email = ?", tenant1.ID, "user2@tenant2.com").Find(&crossTenantTickets).Error
		require.NoError(t, err)
		assert.Len(t, crossTenantTickets, 0) // Should not find any tickets

		// Test user can't access other tenant's tickets
		err = db.Where("tenant_id = ? AND (requester_email = ? OR assigned_to = ?)",
			tenant1.ID, "user2@tenant2.com", user2.ID).Find(&crossTenantTickets).Error
		require.NoError(t, err)
		assert.Len(t, crossTenantTickets, 0)
	})
}

// TestTransactionIntegration tests complex transaction scenarios
func TestTransactionIntegration(t *testing.T) {
	db := setupIntegrationDB(t)

	t.Run("Complex transaction with rollback", func(t *testing.T) {
		// Begin transaction
		tx := db.Begin()
		require.NoError(t, tx.Error)

		// Create tenant in transaction
		tenant := &models.Tenant{
			Name:     "Transaction Test Tenant",
			Slug:     "transaction-test-tenant",
			Plan:     "basic",
			MaxUsers: 100,
			IsActive: true,
		}
		err := tx.Create(tenant).Error
		require.NoError(t, err)

		// Create user in transaction
		user := &models.User{
			TenantID: tenant.ID,
			Email:    "txuser@example.com",
			Username: "txuser",
			Role:     "customer",
			IsActive: true,
		}
		err = tx.Create(user).Error
		require.NoError(t, err)

		// Create ticket in transaction
		ticket := &models.Ticket{
			TenantID:       tenant.ID,
			TicketNumber:   "TX-001",
			Title:          "Transaction Test",
			Description:    "Testing transaction rollback",
			Status:         "open",
			Priority:       "medium",
			RequesterName:  "TX User",
			RequesterEmail: "txuser@example.com",
		}
		err = tx.Create(ticket).Error
		require.NoError(t, err)

		// Create message in transaction
		message := &models.Message{
			TicketID:    ticket.ID,
			UserID:      user.ID,
			Content:     "Initial message in transaction",
			ContentType: "text",
			IsInternal:  false,
			IsFromAI:    false,
		}
		err = tx.Create(message).Error
		require.NoError(t, err)

		// Simulate an error condition that would cause rollback
		// For example, trying to create a duplicate tenant slug
		duplicateTenant := &models.Tenant{
			Name:     "Duplicate Tenant",
			Slug:     "transaction-test-tenant", // Same slug as above
			Plan:     "basic",
			MaxUsers: 50,
			IsActive: true,
		}
		err = tx.Create(duplicateTenant).Error
		assert.Error(t, err) // Should fail due to unique constraint

		// Rollback transaction
		err = tx.Rollback().Error
		require.NoError(t, err)

		// Verify that none of the records exist after rollback
		var foundTenant models.Tenant
		err = db.First(&foundTenant, tenant.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		var foundUser models.User
		err = db.First(&foundUser, user.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		var foundTicket models.Ticket
		err = db.First(&foundTicket, ticket.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		var foundMessage models.Message
		err = db.First(&foundMessage, message.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("Successful complex transaction", func(t *testing.T) {
		// Begin transaction
		tx := db.Begin()
		require.NoError(t, tx.Error)

		// Create tenant
		tenant := &models.Tenant{
			Name:     "Successful TX Tenant",
			Slug:     "successful-tx-tenant",
			Plan:     "professional",
			MaxUsers: 500,
			IsActive: true,
		}
		err := tx.Create(tenant).Error
		require.NoError(t, err)

		// Create multiple users
		users := make([]*models.User, 3)
		for i := 0; i < 3; i++ {
			users[i] = &models.User{
				TenantID: tenant.ID,
				Email:    fmt.Sprintf("user%d@successful-tx.com", i+1),
				Username: fmt.Sprintf("user%d", i+1),
				Role:     []string{"customer", "engineer", "admin"}[i],
				IsActive: true,
			}
			err = tx.Create(users[i]).Error
			require.NoError(t, err)
		}

		// Create multiple tickets
		tickets := make([]*models.Ticket, 2)
		for i := 0; i < 2; i++ {
			tickets[i] = &models.Ticket{
				TenantID:       tenant.ID,
				TicketNumber:   fmt.Sprintf("STX-%03d", i+1),
				Title:          fmt.Sprintf("Ticket %d", i+1),
				Description:    fmt.Sprintf("Description for ticket %d", i+1),
				Status:         "open",
				Priority:       []string{"low", "high"}[i],
				RequesterName:  fmt.Sprintf("User %d", i+1),
				RequesterEmail: users[i].Email,
			}
			err = tx.Create(tickets[i]).Error
			require.NoError(t, err)
		}

		// Create relationships
		for i := 0; i < 2; i++ {
			message := &models.Message{
				TicketID:    tickets[i].ID,
				UserID:      users[i].ID,
				Content:     fmt.Sprintf("Message for ticket %d", i+1),
				ContentType: "text",
				IsInternal:  false,
				IsFromAI:    false,
			}
			err = tx.Create(message).Error
			require.NoError(t, err)
		}

		// Commit transaction
		err = tx.Commit().Error
		require.NoError(t, err)

		// Verify all records exist after commit
		var finalTenant models.Tenant
		err = db.First(&finalTenant, tenant.ID).Error
		require.NoError(t, err)
		assert.Equal(t, tenant.Name, finalTenant.Name)

		var finalUsers []models.User
		err = db.Where("tenant_id = ?", tenant.ID).Find(&finalUsers).Error
		require.NoError(t, err)
		assert.Len(t, finalUsers, 3)

		var finalTickets []models.Ticket
		err = db.Where("tenant_id = ?", tenant.ID).Find(&finalTickets).Error
		require.NoError(t, err)
		assert.Len(t, finalTickets, 2)

		var finalMessages []models.Message
		err = db.Joins("JOIN tickets ON messages.ticket_id = tickets.id").
			Where("tickets.tenant_id = ?", tenant.ID).Find(&finalMessages).Error
		require.NoError(t, err)
		assert.Len(t, finalMessages, 2)
	})
}

// TestCascadeOperationsIntegration tests cascade operations and relationships
func TestCascadeOperationsIntegration(t *testing.T) {
	db := setupIntegrationDB(t)

	t.Run("Cascade delete operations", func(t *testing.T) {
		// Create tenant
		tenant := &models.Tenant{
			Name:     "Cascade Test Tenant",
			Slug:     "cascade-test-tenant",
			Plan:     "enterprise",
			MaxUsers: 1000,
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create users
		users := make([]*models.User, 2)
		for i := 0; i < 2; i++ {
			users[i] = &models.User{
				TenantID: tenant.ID,
				Email:    fmt.Sprintf("cascade%d@example.com", i+1),
				Username: fmt.Sprintf("cascade%d", i+1),
				Role:     "customer",
				IsActive: true,
			}
			err = db.Create(users[i]).Error
			require.NoError(t, err)
		}

		// Create tickets
		tickets := make([]*models.Ticket, 3)
		for i := 0; i < 3; i++ {
			tickets[i] = &models.Ticket{
				TenantID:       tenant.ID,
				TicketNumber:   fmt.Sprintf("CAS-%03d", i+1),
				Title:          fmt.Sprintf("Cascade Ticket %d", i+1),
				Description:    "Testing cascade operations",
				Status:         "open",
				Priority:       "medium",
				RequesterName:  fmt.Sprintf("Cascade User %d", i%2+1),
				RequesterEmail: users[i%2].Email,
			}
			err = db.Create(tickets[i]).Error
			require.NoError(t, err)
		}

		// Create messages
		messageCount := 0
		for _, ticket := range tickets {
			for _, user := range users {
				message := &models.Message{
					TicketID:    ticket.ID,
					UserID:      user.ID,
					Content:     fmt.Sprintf("Message from %s on ticket %s", user.Username, ticket.TicketNumber),
					ContentType: "text",
					IsInternal:  false,
					IsFromAI:    false,
				}
				err = db.Create(message).Error
				require.NoError(t, err)
				messageCount++
			}
		}

		// Verify initial state
		var userCount int64
		db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&userCount)
		assert.Equal(t, int64(2), userCount)

		var ticketCount int64
		db.Model(&models.Ticket{}).Where("tenant_id = ?", tenant.ID).Count(&ticketCount)
		assert.Equal(t, int64(3), ticketCount)

		var messageCountBefore int64
		db.Table("messages").
			Joins("JOIN tickets ON messages.ticket_id = tickets.id").
			Where("tickets.tenant_id = ?", tenant.ID).
			Count(&messageCountBefore)
		assert.Equal(t, int64(messageCount), messageCountBefore)

		// Soft delete tenant (note: GORM soft deletes don't automatically cascade to related records)
		err = db.Delete(tenant).Error
		require.NoError(t, err)

		// In GORM, soft deleting the tenant doesn't automatically soft delete related records
		// Each related record needs to be soft deleted individually for proper cascade behavior
		// For this test, we'll verify that tenant is soft deleted but related records remain
		var remainingUsers int64
		db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&remainingUsers)
		assert.Equal(t, int64(2), remainingUsers) // Users are still there (no automatic cascade)

		var remainingTickets int64
		db.Model(&models.Ticket{}).Where("tenant_id = ?", tenant.ID).Count(&remainingTickets)
		assert.Equal(t, int64(3), remainingTickets) // Tickets are still there (no automatic cascade)

		// Verify tenant is soft deleted
		var foundTenant models.Tenant
		err = db.First(&foundTenant, tenant.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// But we can find the tenant with Unscoped
		var deletedTenant int64
		db.Model(&models.Tenant{}).Unscoped().Where("id = ?", tenant.ID).Count(&deletedTenant)
		assert.Equal(t, int64(1), deletedTenant)
	})
}
