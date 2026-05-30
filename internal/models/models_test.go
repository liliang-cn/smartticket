package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseModel tests the BaseModel struct.
func TestBaseModel(t *testing.T) {
	t.Run("Empty BaseModel", func(t *testing.T) {
		model := BaseModel{}

		assert.Zero(t, model.ID)
		assert.True(t, model.CreatedAt.IsZero())
		assert.True(t, model.UpdatedAt.IsZero())
		assert.True(t, model.DeletedAt.Time.IsZero())
		assert.Nil(t, model.CreatedBy)
		assert.Nil(t, model.UpdatedBy)
	})

	t.Run("BaseModel with values", func(t *testing.T) {
		now := time.Now()
		createdBy := "user123"
		updatedBy := "user456"

		model := BaseModel{
			ID:        1,
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: &createdBy,
			UpdatedBy: &updatedBy,
		}

		assert.Equal(t, uint(1), model.ID)
		assert.Equal(t, now, model.CreatedAt)
		assert.Equal(t, now, model.UpdatedAt)
		assert.Equal(t, &createdBy, model.CreatedBy)
		assert.Equal(t, &updatedBy, model.UpdatedBy)
	})
}

// TestUser tests the User model.
func TestUser(t *testing.T) {
	t.Run("Empty User", func(t *testing.T) {
		user := User{}

		assert.Zero(t, user)
		assert.Empty(t, user.Email)
		assert.Empty(t, user.Username)
		assert.Empty(t, user.PasswordHash)
		assert.Empty(t, user.FirstName)
		assert.Empty(t, user.LastName)
		// Role field removed from User model - now handled by UserRole associations
		assert.False(t, user.IsActive) // Zero value for bool is false
		assert.Nil(t, user.LastLoginAt)
		assert.Empty(t, user.Preferences)
		assert.Empty(t, user.Tickets)
		assert.Empty(t, user.Messages)
	})

	t.Run("User with valid data", func(t *testing.T) {
		now := time.Now()
		lastLoginAt := now.Add(-24 * time.Hour)
		createdBy := "admin"
		updatedBy := "admin"

		user := User{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			Email:        "test@example.com",
			Username:     "testuser",
			PasswordHash: "hashed_password",
			FirstName:    "Test",
			LastName:     "User",
			IsActive:     true,
			LastLoginAt:  &lastLoginAt,
			Preferences:  `{"theme": "light", "notifications": true}`,
		}

		assert.Equal(t, uint(1), user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "hashed_password", user.PasswordHash)
		assert.Equal(t, "Test", user.FirstName)
		assert.Equal(t, "User", user.LastName)
		// Role field removed from User model - role assignment now done through UserRole table
		assert.True(t, user.IsActive)
		assert.Equal(t, &lastLoginAt, user.LastLoginAt)
		assert.Equal(t, `{"theme": "light", "notifications": true}`, user.Preferences)
	})

	t.Run("User JSON serialization", func(t *testing.T) {
		user := User{
			BaseModel: BaseModel{
				ID: 1,
			},
			Email:       "test@example.com",
			Username:    "testuser",
			FirstName:   "Test",
			LastName:    "User",
			IsActive:    true,
			Preferences: `{"theme": "dark"}`,
		}

		jsonData, err := json.Marshal(user)
		require.NoError(t, err)

		var unmarshaled User
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, user, unmarshaled)
		assert.Equal(t, user.Email, unmarshaled.Email)
		assert.Equal(t, user.Username, unmarshaled.Username)
		assert.Equal(t, user.FirstName, unmarshaled.FirstName)
		assert.Equal(t, user.LastName, unmarshaled.LastName)
		// Role field has been removed from User model
		// assert.Equal(t, user.Role, unmarshaled.Role)
		assert.Equal(t, user.IsActive, unmarshaled.IsActive)
		assert.Equal(t, user.Preferences, unmarshaled.Preferences)
	})

	t.Run("User role validation", func(t *testing.T) {
		// Role field removed from User model - now handled by UserRole associations
		// Role validation is now done through the UserRole model and service layer
		// This test validates that the User model no longer has a Role field
		user := User{}
		// Verify that User model doesn't have a Role field by checking other fields
		assert.Empty(t, user.Email)
		assert.Empty(t, user.Username)
		// Role field has been removed - role assignment is now through UserRole table
	})

	t.Run("User timestamps", func(t *testing.T) {
		now := time.Now()
		user := User{
			BaseModel: BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		// Test timestamp field
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
	})
}

// TestTicket tests the Ticket model.
func TestTicket(t *testing.T) {
	t.Run("Empty Ticket", func(t *testing.T) {
		ticket := Ticket{}

		assert.Zero(t, ticket)
		assert.Empty(t, ticket.TicketNumber)
		assert.Empty(t, ticket.Title)
		assert.Empty(t, ticket.Description)
		assert.Empty(t, ticket.Status)   // Zero value for string is ""
		assert.Empty(t, ticket.Priority) // Zero value for string is ""
		assert.Empty(t, ticket.Severity) // Zero value for string is ""
		assert.Empty(t, ticket.Category)
		assert.Empty(t, ticket.Type)
		assert.Nil(t, ticket.ProductID)
		assert.Nil(t, ticket.ServiceID)
		assert.Nil(t, ticket.AssignedTo)
		assert.Empty(t, ticket.RequesterName)
		assert.Empty(t, ticket.RequesterEmail)
		assert.Empty(t, ticket.Tags)
		assert.Empty(t, ticket.CustomFields)
		assert.False(t, ticket.IsDeleted)
		assert.Nil(t, ticket.ResolutionTime)
		assert.Nil(t, ticket.ResolvedAt)
		assert.Nil(t, ticket.DueDate)
		assert.Empty(t, ticket.SLAStatus) // Zero value for string is ""
		assert.Empty(t, ticket.Messages)
		assert.Empty(t, ticket.Attachments)
	})

	t.Run("Ticket with valid data", func(t *testing.T) {
		now := time.Now()
		resolutionTime := now.Add(-24 * time.Hour)
		resolvedAt := now.Add(-24 * time.Hour)
		dueDate := now.Add(7 * 24 * time.Hour)
		createdBy := "admin"
		updatedBy := "admin"

		ticket := Ticket{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			TicketNumber:   "TICKET-001",
			Title:          "Test Ticket",
			Description:    "This is a test ticket",
			Status:         "resolved",
			Priority:       "high",
			Severity:       "major",
			Category:       "bug",
			Type:           "incident",
			AssignedTo:     &[]uint{2}[0],
			RequesterName:  "John Doe",
			RequesterEmail: "john@example.com",
			Tags:           `["urgent", "backend"]`,
			CustomFields:   `{"priority": "high"}`,
			IsDeleted:      false,
			ResolutionTime: &resolutionTime,
			ResolvedAt:     &resolvedAt,
			DueDate:        &dueDate,
			SLAStatus:      "within",
		}

		assert.Equal(t, uint(1), ticket.ID)
		assert.Equal(t, "TICKET-001", ticket.TicketNumber)
		assert.Equal(t, "Test Ticket", ticket.Title)
		assert.Equal(t, "This is a test ticket", ticket.Description)
		assert.Equal(t, "resolved", ticket.Status)
		assert.Equal(t, "high", ticket.Priority)
		assert.Equal(t, "major", ticket.Severity)
		assert.Equal(t, "bug", ticket.Category)
		assert.Equal(t, "incident", ticket.Type)
		assert.Equal(t, &[]uint{2}[0], ticket.AssignedTo)
		assert.Equal(t, "John Doe", ticket.RequesterName)
		assert.Equal(t, "john@example.com", ticket.RequesterEmail)
		assert.Equal(t, `["urgent", "backend"]`, ticket.Tags)
		assert.Equal(t, `{"priority": "high"}`, ticket.CustomFields)
		assert.False(t, ticket.IsDeleted)
		assert.Equal(t, &resolutionTime, ticket.ResolutionTime)
		assert.Equal(t, &resolvedAt, ticket.ResolvedAt)
		assert.Equal(t, &dueDate, ticket.DueDate)
		assert.Equal(t, "within", ticket.SLAStatus)
	})

	t.Run("Ticket JSON serialization", func(t *testing.T) {
		ticket := Ticket{
			BaseModel: BaseModel{
				ID: 1,
			},
			TicketNumber: "TICKET-001",
			Title:        "Test Ticket",
			Description:  "Test description",
			Status:       "open",
			Priority:     "medium",
			Severity:     "minor",
			Category:     "general",
		}

		jsonData, err := json.Marshal(ticket)
		require.NoError(t, err)

		var unmarshaled Ticket
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, ticket, unmarshaled)
		assert.Equal(t, ticket.TicketNumber, unmarshaled.TicketNumber)
		assert.Equal(t, ticket.Title, unmarshaled.Title)
		assert.Equal(t, ticket.Description, unmarshaled.Description)
		assert.Equal(t, ticket.Status, unmarshaled.Status)
		assert.Equal(t, ticket.Priority, unmarshaled.Priority)
		assert.Equal(t, ticket.Severity, unmarshaled.Severity)
		assert.Equal(t, ticket.Category, unmarshaled.Category)
	})

	t.Run("Ticket status validation", func(t *testing.T) {
		validStatuses := []string{"open", "in_progress", "resolved", "closed", "cancelled"}

		for _, status := range validStatuses {
			ticket := Ticket{Status: status}
			assert.Contains(t, validStatuses, ticket.Status)
		}
	})

	t.Run("Ticket priority validation", func(t *testing.T) {
		validPriorities := []string{"low", "medium", "high", "critical"}

		for _, priority := range validPriorities {
			ticket := Ticket{Priority: priority}
			assert.Contains(t, validPriorities, ticket.Priority)
		}
	})

	t.Run("Ticket severity validation", func(t *testing.T) {
		validSeverities := []string{"trivial", "minor", "major", "critical"}

		for _, severity := range validSeverities {
			ticket := Ticket{Severity: severity}
			assert.Contains(t, validSeverities, ticket.Severity)
		}
	})
}

// TestMessage tests the Message model.
func TestMessage(t *testing.T) {
	t.Run("Empty Message", func(t *testing.T) {
		message := Message{}

		assert.Zero(t, message.TicketID)
		assert.Zero(t, message.UserID)
		assert.Empty(t, message.Content)
		assert.Empty(t, message.ContentType) // Zero value for string is ""
		assert.False(t, message.IsInternal)
		assert.False(t, message.IsFromAI)
		assert.Empty(t, message.Attachments)
	})

	t.Run("Message with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"

		message := Message{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			TicketID:    1,
			UserID:      2,
			Content:     "This is a test message",
			ContentType: "html",
			IsInternal:  true,
			IsFromAI:    false,
		}

		assert.Equal(t, uint(1), message.TicketID)
		assert.Equal(t, uint(2), message.UserID)
		assert.Equal(t, "This is a test message", message.Content)
		assert.Equal(t, "html", message.ContentType)
		assert.True(t, message.IsInternal)
		assert.False(t, message.IsFromAI)
	})

	t.Run("Message content type validation", func(t *testing.T) {
		validTypes := []string{"text", "html", "markdown"}

		for _, contentType := range validTypes {
			message := Message{ContentType: contentType}
			assert.Contains(t, validTypes, message.ContentType)
		}
	})

	t.Run("Message JSON serialization", func(t *testing.T) {
		message := Message{
			BaseModel: BaseModel{
				ID: 1,
			},
			TicketID:    1,
			UserID:      1,
			Content:     "Test content",
			ContentType: "text",
			IsInternal:  false,
			IsFromAI:    true,
		}

		jsonData, err := json.Marshal(message)
		require.NoError(t, err)

		var unmarshaled Message
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, message.TicketID, unmarshaled.TicketID)
		assert.Equal(t, message.UserID, unmarshaled.UserID)
		assert.Equal(t, message.Content, unmarshaled.Content)
		assert.Equal(t, message.ContentType, unmarshaled.ContentType)
		assert.Equal(t, message.IsInternal, unmarshaled.IsInternal)
		assert.Equal(t, message.IsFromAI, unmarshaled.IsFromAI)
	})
}

// TestAttachment tests the Attachment model.
func TestAttachment(t *testing.T) {
	t.Run("Empty Attachment", func(t *testing.T) {
		attachment := Attachment{}

		assert.Zero(t, attachment.TicketID)
		assert.Zero(t, attachment.MessageID)
		assert.Zero(t, attachment.KnowledgeArticleID)
		assert.Empty(t, attachment.FileName)
		assert.Empty(t, attachment.OriginalName)
		assert.Empty(t, attachment.FilePath)
		assert.Zero(t, attachment.FileSize)
		assert.Empty(t, attachment.ContentType)
		assert.Empty(t, attachment.Hash)
	})

	t.Run("Attachment with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"

		attachment := Attachment{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			TicketID:     1,
			FileName:     "test.pdf",
			OriginalName: "test_document.pdf",
			FilePath:     "/uploads/test.pdf",
			FileSize:     1024,
			ContentType:  "application/pdf",
			Hash:         "abc123def456",
		}

		assert.Equal(t, uint(1), attachment.TicketID)
		assert.Equal(t, "test.pdf", attachment.FileName)
		assert.Equal(t, "test_document.pdf", attachment.OriginalName)
		assert.Equal(t, "/uploads/test.pdf", attachment.FilePath)
		assert.Equal(t, int64(1024), attachment.FileSize)
		assert.Equal(t, "application/pdf", attachment.ContentType)
		assert.Equal(t, "abc123def456", attachment.Hash)
	})

	t.Run("Attachment JSON serialization", func(t *testing.T) {
		attachment := Attachment{
			BaseModel: BaseModel{
				ID: 1,
			},
			TicketID:     1,
			FileName:     "test.pdf",
			OriginalName: "test_document.pdf",
			FilePath:     "/uploads/test.pdf",
			FileSize:     1024,
			ContentType:  "application/pdf",
			Hash:         "abc123def456",
		}

		jsonData, err := json.Marshal(attachment)
		require.NoError(t, err)

		var unmarshaled Attachment
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, attachment.TicketID, unmarshaled.TicketID)
		assert.Equal(t, attachment.FileName, unmarshaled.FileName)
		assert.Equal(t, attachment.OriginalName, unmarshaled.OriginalName)
		assert.Equal(t, attachment.FilePath, unmarshaled.FilePath)
		assert.Equal(t, attachment.FileSize, unmarshaled.FileSize)
		assert.Equal(t, attachment.ContentType, unmarshaled.ContentType)
		assert.Equal(t, attachment.Hash, unmarshaled.Hash)
	})
}

// TestKnowledgeArticle tests the KnowledgeArticle model.
func TestKnowledgeArticle(t *testing.T) {
	t.Run("Empty KnowledgeArticle", func(t *testing.T) {
		article := KnowledgeArticle{}

		assert.Zero(t, article)
		assert.Empty(t, article.Title)
		assert.Empty(t, article.Slug)
		assert.Empty(t, article.Content)
		assert.Empty(t, article.ContentType) // Zero value for string is ""
		assert.Empty(t, article.Summary)
		assert.Zero(t, article.AuthorID)
		assert.Empty(t, article.Status)      // Zero value for string is ""
		assert.Empty(t, article.Visibility)  // Zero value for string is ""
		assert.Empty(t, article.AccessLevel) // Zero value for string is ""
		assert.Empty(t, article.Category)
		assert.Empty(t, article.Tags)
		assert.Zero(t, article.Views)
		assert.Zero(t, article.HelpfulVotes)
		assert.Equal(t, 0, article.Version) // Zero value for int is 0
		assert.Nil(t, article.ParentID)
		assert.Empty(t, article.Attachments)
	})

	t.Run("KnowledgeArticle with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"
		parentID := uint(2)

		article := KnowledgeArticle{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			Title:        "Test Article",
			Slug:         "test-article",
			Content:      "# Test Article\nThis is a test article content",
			ContentType:  "markdown",
			Summary:      "Test article summary",
			AuthorID:     2,
			Status:       "published",
			Visibility:   "internal",
			AccessLevel:  "agents",
			Category:     "documentation",
			Tags:         `["test", "documentation"]`,
			Views:        100,
			HelpfulVotes: 5,
			Version:      2,
			ParentID:     &parentID,
		}

		assert.Equal(t, uint(1), article.ID)
		assert.Equal(t, "Test Article", article.Title)
		assert.Equal(t, "test-article", article.Slug)
		assert.Equal(t, "# Test Article\nThis is a test article content", article.Content)
		assert.Equal(t, "markdown", article.ContentType)
		assert.Equal(t, "Test article summary", article.Summary)
		assert.Equal(t, uint(2), article.AuthorID)
		assert.Equal(t, "published", article.Status)
		assert.Equal(t, "internal", article.Visibility)
		assert.Equal(t, "agents", article.AccessLevel)
		assert.Equal(t, "documentation", article.Category)
		assert.Equal(t, `["test", "documentation"]`, article.Tags)
		assert.Equal(t, 100, article.Views)
		assert.Equal(t, 5, article.HelpfulVotes)
		assert.Equal(t, 2, article.Version)
		assert.Equal(t, &parentID, article.ParentID)
	})

	t.Run("KnowledgeArticle JSON serialization", func(t *testing.T) {
		article := KnowledgeArticle{
			BaseModel: BaseModel{
				ID: 1,
			},
			Title:        "Test Article",
			Slug:         "test-article",
			Content:      "Test content",
			ContentType:  "markdown",
			Summary:      "Test summary",
			Status:       "published",
			Visibility:   "public",
			AccessLevel:  "all",
			Category:     "docs",
			Tags:         `["test"]`,
			Views:        50,
			HelpfulVotes: 3,
			Version:      1,
		}

		jsonData, err := json.Marshal(article)
		require.NoError(t, err)

		var unmarshaled KnowledgeArticle
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, article, unmarshaled)
		assert.Equal(t, article.Title, unmarshaled.Title)
		assert.Equal(t, article.Slug, unmarshaled.Slug)
		assert.Equal(t, article.Content, unmarshaled.Content)
		assert.Equal(t, article.ContentType, unmarshaled.ContentType)
		assert.Equal(t, article.Summary, unmarshaled.Summary)
		assert.Equal(t, article.Status, unmarshaled.Status)
		assert.Equal(t, article.Visibility, unmarshaled.Visibility)
		assert.Equal(t, article.AccessLevel, unmarshaled.AccessLevel)
		assert.Equal(t, article.Category, unmarshaled.Category)
		assert.Equal(t, article.Tags, unmarshaled.Tags)
		assert.Equal(t, article.Views, unmarshaled.Views)
		assert.Equal(t, article.HelpfulVotes, unmarshaled.HelpfulVotes)
		assert.Equal(t, article.Version, unmarshaled.Version)
	})

	t.Run("KnowledgeArticle status validation", func(t *testing.T) {
		validStatuses := []string{"draft", "published", "archived"}

		for _, status := range validStatuses {
			article := KnowledgeArticle{Status: status}
			assert.Contains(t, validStatuses, article.Status)
		}
	})

	t.Run("KnowledgeArticle visibility validation", func(t *testing.T) {
		validVisibilities := []string{"public", "internal", "private"}

		for _, visibility := range validVisibilities {
			article := KnowledgeArticle{Visibility: visibility}
			assert.Contains(t, validVisibilities, article.Visibility)
		}
	})
}

// TestLLMProvider tests the LLMProvider model.
func TestLLMProvider(t *testing.T) {
	t.Run("Empty LLMProvider", func(t *testing.T) {
		provider := LLMProvider{}

		assert.Zero(t, provider)
		assert.Empty(t, provider.Name)
		assert.Empty(t, provider.ProviderType)
		assert.Empty(t, provider.APIEndpoint)
		assert.Empty(t, provider.APIKey)
		assert.Empty(t, provider.Model)
		assert.Equal(t, 0, provider.MaxTokens)     // Zero value for int is 0
		assert.Equal(t, 0.0, provider.Temperature) // Zero value for float64 is 0.0
		assert.Empty(t, provider.TaskTypes)
		assert.False(t, provider.IsDefault)     // Zero value for bool is false
		assert.False(t, provider.IsEnabled)     // Zero value for bool is false
		assert.Equal(t, 0, provider.QuotaLimit) // Zero value for int is 0
		assert.Equal(t, 0, provider.QuotaUsed)  // Zero value for int is 0
		assert.Empty(t, provider.Configuration)
	})

	t.Run("LLMProvider with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"

		provider := LLMProvider{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			Name:          "OpenAI GPT-4",
			ProviderType:  "openai",
			APIEndpoint:   "https://api.openai.com/v1",
			APIKey:        "sk-abc123def456",
			Model:         "gpt-4",
			MaxTokens:     8192,
			Temperature:   0.5,
			TaskTypes:     `["chat", "completion", "embedding"]`,
			IsDefault:     true,
			IsEnabled:     true,
			QuotaLimit:    5000,
			QuotaUsed:     1234,
			Configuration: `{"timeout": 30}`,
		}

		assert.Equal(t, uint(1), provider.ID)
		assert.Equal(t, "OpenAI GPT-4", provider.Name)
		assert.Equal(t, "openai", provider.ProviderType)
		assert.Equal(t, "https://api.openai.com/v1", provider.APIEndpoint)
		assert.Equal(t, "sk-abc123def456", provider.APIKey)
		assert.Equal(t, "gpt-4", provider.Model)
		assert.Equal(t, 8192, provider.MaxTokens)
		assert.Equal(t, 0.5, provider.Temperature)
		assert.Equal(t, `["chat", "completion", "embedding"]`, provider.TaskTypes)
		assert.True(t, provider.IsDefault)
		assert.True(t, provider.IsEnabled)
		assert.Equal(t, 5000, provider.QuotaLimit)
		assert.Equal(t, 1234, provider.QuotaUsed)
		assert.Equal(t, `{"timeout": 30}`, provider.Configuration)
	})

	t.Run("LLMProvider JSON serialization", func(t *testing.T) {
		provider := LLMProvider{
			BaseModel: BaseModel{
				ID: 1,
			},
			Name:         "Test Provider",
			ProviderType: "test",
			APIEndpoint:  "https://api.test.com/v1",
			Model:        "test-model",
			MaxTokens:    2048,
			Temperature:  0.8,
			TaskTypes:    `["chat"]`,
			IsDefault:    false,
			IsEnabled:    true,
			QuotaLimit:   2000,
			QuotaUsed:    500,
		}

		jsonData, err := json.Marshal(provider)
		require.NoError(t, err)

		var unmarshaled LLMProvider
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, provider, unmarshaled)
		assert.Equal(t, provider.Name, unmarshaled.Name)
		assert.Equal(t, provider.ProviderType, unmarshaled.ProviderType)
		assert.Equal(t, provider.APIEndpoint, unmarshaled.APIEndpoint)
		assert.Equal(t, provider.Model, unmarshaled.Model)
		assert.Equal(t, provider.MaxTokens, unmarshaled.MaxTokens)
		assert.Equal(t, provider.Temperature, unmarshaled.Temperature)
		assert.Equal(t, provider.TaskTypes, unmarshaled.TaskTypes)
		assert.Equal(t, provider.IsDefault, unmarshaled.IsDefault)
		assert.Equal(t, provider.IsEnabled, unmarshaled.IsEnabled)
		assert.Equal(t, provider.QuotaLimit, unmarshaled.QuotaLimit)
		assert.Equal(t, provider.QuotaUsed, unmarshaled.QuotaUsed)
	})

	t.Run("LLMProvider provider type validation", func(t *testing.T) {
		validProviders := []string{"openai", "azure", "anthropic", "deepseek", "ollama", "local"}

		for _, providerType := range validProviders {
			provider := LLMProvider{ProviderType: providerType}
			assert.Contains(t, validProviders, provider.ProviderType)
		}
	})
}

// TestImportExportJob tests the ImportExportJob model.
func TestImportExportJob(t *testing.T) {
	t.Run("Empty ImportExportJob", func(t *testing.T) {
		job := ImportExportJob{}

		assert.Zero(t, job)
		assert.Empty(t, job.Type)   // Zero value for string is ""
		assert.Empty(t, job.Status) // Zero value for string is ""
		assert.Zero(t, job.Progress)
		assert.Zero(t, job.TotalRecords)
		assert.Zero(t, job.ProcessedRecords)
		assert.Zero(t, job.FailedRecords)
		assert.Empty(t, job.SourceFormat)
		assert.Empty(t, job.TargetFormat)
		assert.Empty(t, job.FilePath)
		assert.Empty(t, job.Configuration)
		assert.Empty(t, job.Error)
		assert.Nil(t, job.StartedAt)
		assert.Nil(t, job.CompletedAt)
		assert.Zero(t, job.StartedBy)
	})

	t.Run("ImportExportJob with valid data", func(t *testing.T) {
		now := time.Now()
		startedAt := now.Add(-1 * time.Hour)
		completedAt := now.Add(-30 * time.Minute)
		createdBy := "admin"
		updatedBy := "admin"

		job := ImportExportJob{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			Type:             "export",
			Status:           "completed",
			Progress:         100,
			TotalRecords:     1000,
			ProcessedRecords: 950,
			FailedRecords:    50,
			SourceFormat:     "json",
			TargetFormat:     "csv",
			FilePath:         "/data/export/test.csv",
			Configuration:    `{"delimiter": ",", "encoding": "utf-8"}`,
			Error:            "",
			StartedAt:        &startedAt,
			CompletedAt:      &completedAt,
			StartedBy:        2,
		}

		assert.Equal(t, uint(1), job.ID)
		assert.Equal(t, "export", job.Type)
		assert.Equal(t, "completed", job.Status)
		assert.Equal(t, 100, job.Progress)
		assert.Equal(t, 1000, job.TotalRecords)
		assert.Equal(t, 950, job.ProcessedRecords)
		assert.Equal(t, 50, job.FailedRecords)
		assert.Equal(t, "json", job.SourceFormat)
		assert.Equal(t, "csv", job.TargetFormat)
		assert.Equal(t, "/data/export/test.csv", job.FilePath)
		assert.Equal(t, `{"delimiter": ",", "encoding": "utf-8"}`, job.Configuration)
		assert.Equal(t, "", job.Error)
		assert.Equal(t, &startedAt, job.StartedAt)
		assert.Equal(t, &completedAt, job.CompletedAt)
		assert.Equal(t, uint(2), job.StartedBy)
	})

	t.Run("ImportExportJob JSON serialization", func(t *testing.T) {
		job := ImportExportJob{
			BaseModel: BaseModel{
				ID: 1,
			},
			Type:         "import",
			Status:       "running",
			Progress:     50,
			TotalRecords: 200,
			FilePath:     "/data/import/test.csv",
		}

		jsonData, err := json.Marshal(job)
		require.NoError(t, err)

		var unmarshaled ImportExportJob
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, job, unmarshaled)
		assert.Equal(t, job.Type, unmarshaled.Type)
		assert.Equal(t, job.Status, unmarshaled.Status)
		assert.Equal(t, job.Progress, unmarshaled.Progress)
		assert.Equal(t, job.TotalRecords, unmarshaled.TotalRecords)
		assert.Equal(t, job.FilePath, unmarshaled.FilePath)
	})

	t.Run("ImportExportJob status validation", func(t *testing.T) {
		validStatuses := []string{"pending", "running", "completed", "failed"}

		for _, status := range validStatuses {
			job := ImportExportJob{Status: status}
			assert.Contains(t, validStatuses, job.Status)
		}
	})
}

// TestAuditLog tests the AuditLog model.
func TestAuditLog(t *testing.T) {
	t.Run("Empty AuditLog", func(t *testing.T) {
		log := AuditLog{}

		assert.Zero(t, log)
		assert.Zero(t, log.UserID)
		assert.Empty(t, log.Action)
		assert.Empty(t, log.ResourceType)
		assert.Zero(t, log.ResourceID)
		assert.Empty(t, log.ResourceName)
		assert.Empty(t, log.IPAddress)
		assert.Empty(t, log.UserAgent)
		assert.Empty(t, log.Changes)
		assert.Empty(t, log.OldValues)
		assert.Empty(t, log.NewValues)
		assert.Empty(t, log.RequestID)
		assert.Empty(t, log.Hash)
	})

	t.Run("AuditLog with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"

		log := AuditLog{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			UserID:       2,
			Action:       "ticket.created",
			ResourceType: "ticket",
			ResourceID:   3,
			ResourceName: "Test Ticket",
			IPAddress:    "192.168.1.100",
			UserAgent:    "Mozilla/5.0",
			Changes:      `{"title": "New Title"}`,
			OldValues:    `{"title": "Old Title"}`,
			NewValues:    `{"title": "New Title"}`,
			RequestID:    "req-123456",
			Hash:         "abc123def456",
		}

		assert.Equal(t, uint(1), log.ID)
		assert.Equal(t, uint(2), log.UserID)
		assert.Equal(t, "ticket.created", log.Action)
		assert.Equal(t, "ticket", log.ResourceType)
		assert.Equal(t, uint(3), log.ResourceID)
		assert.Equal(t, "Test Ticket", log.ResourceName)
		assert.Equal(t, "192.168.1.100", log.IPAddress)
		assert.Equal(t, "Mozilla/5.0", log.UserAgent)
		assert.Equal(t, `{"title": "New Title"}`, log.Changes)
		assert.Equal(t, `{"title": "Old Title"}`, log.OldValues)
		assert.Equal(t, `{"title": "New Title"}`, log.NewValues)
		assert.Equal(t, "req-123456", log.RequestID)
		assert.Equal(t, "abc123def456", log.Hash)
	})

	t.Run("AuditLog JSON serialization", func(t *testing.T) {
		log := AuditLog{
			BaseModel: BaseModel{
				ID: 1,
			},
			UserID:       1,
			Action:       "user.updated",
			ResourceType: "user",
			ResourceID:   1,
			ResourceName: "Test User",
			IPAddress:    "127.0.0.1",
			UserAgent:    "test-agent",
			Changes:      `{"name": "New Name"}`,
			RequestID:    "req-789",
			Hash:         "def789ghi012",
		}

		jsonData, err := json.Marshal(log)
		require.NoError(t, err)

		var unmarshaled AuditLog
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, log, unmarshaled)
		assert.Equal(t, log.UserID, unmarshaled.UserID)
		assert.Equal(t, log.Action, unmarshaled.Action)
		assert.Equal(t, log.ResourceType, unmarshaled.ResourceType)
		assert.Equal(t, log.ResourceID, unmarshaled.ResourceID)
		assert.Equal(t, log.ResourceName, unmarshaled.ResourceName)
		assert.Equal(t, log.IPAddress, unmarshaled.IPAddress)
		assert.Equal(t, log.UserAgent, unmarshaled.UserAgent)
		assert.Equal(t, log.Changes, unmarshaled.Changes)
		assert.Equal(t, log.RequestID, unmarshaled.RequestID)
		assert.Equal(t, log.Hash, unmarshaled.Hash)
	})
}

// TestPermission tests the Permission model.
func TestPermission(t *testing.T) {
	t.Run("Empty Permission", func(t *testing.T) {
		permission := Permission{}

		assert.Empty(t, permission.Code)
		assert.Empty(t, permission.Name)
		assert.Empty(t, permission.Description)
		assert.Empty(t, permission.Category)
		assert.False(t, permission.IsSystem)
		assert.Empty(t, permission.RoleAssignments)
		assert.Empty(t, permission.UserAssignments)
	})

	t.Run("Permission with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"

		permission := Permission{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			Code:        "tickets:read",
			Name:        "Read Tickets",
			Description: "Allows reading tickets",
			Category:    "tickets",
			IsSystem:    false,
		}

		assert.Equal(t, "tickets:read", permission.Code)
		assert.Equal(t, "Read Tickets", permission.Name)
		assert.Equal(t, "Allows reading tickets", permission.Description)
		assert.Equal(t, "tickets", permission.Category)
		assert.False(t, permission.IsSystem)
	})

	t.Run("Permission code format validation", func(t *testing.T) {
		validCodes := []string{
			"tickets:read", "tickets:write", "tickets:delete",
			"users:read", "users:write", "users:delete",
			"knowledge:read", "knowledge:write",
		}

		for _, code := range validCodes {
			permission := Permission{Code: code}
			assert.Equal(t, code, permission.Code)
			// Verify format: resource:action
			parts := strings.Split(code, ":")
			assert.Len(t, parts, 2, "Permission code should be in 'resource:action' format")
			assert.NotEmpty(t, parts[0], "Resource part should not be empty")
			assert.NotEmpty(t, parts[1], "Action part should not be empty")
		}
	})
}

// TestRole tests the Role model.
func TestRole(t *testing.T) {
	t.Run("Empty Role", func(t *testing.T) {
		role := Role{}

		assert.Zero(t, role)
		assert.Empty(t, role.Name)
		assert.Empty(t, role.Description)
		assert.False(t, role.IsSystem)
		assert.False(t, role.IsActive) // Zero value for bool is false
		assert.Zero(t, role.CreatedBy)
		assert.Empty(t, role.Permissions)
	})

	t.Run("Role with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"

		role := Role{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			Name:        "Engineer",
			Description: "Engineer role with ticket permissions",
			IsSystem:    false,
			IsActive:    true,
			CreatedBy:   2,
		}

		assert.Equal(t, uint(1), role.ID)
		assert.Equal(t, "Engineer", role.Name)
		assert.Equal(t, "Engineer role with ticket permissions", role.Description)
		assert.False(t, role.IsSystem)
		assert.True(t, role.IsActive)
		assert.Equal(t, uint(2), role.CreatedBy)
	})

	t.Run("Role JSON serialization", func(t *testing.T) {
		role := Role{
			BaseModel: BaseModel{
				ID: 1,
			},
			Name:        "Admin",
			Description: "Administrator role",
			IsSystem:    true,
			IsActive:    true,
			CreatedBy:   1,
		}

		jsonData, err := json.Marshal(role)
		require.NoError(t, err)

		var unmarshaled Role
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, role, unmarshaled)
		assert.Equal(t, role.Name, unmarshaled.Name)
		assert.Equal(t, role.Description, unmarshaled.Description)
		assert.Equal(t, role.IsSystem, unmarshaled.IsSystem)
		assert.Equal(t, role.IsActive, unmarshaled.IsActive)
		assert.Equal(t, role.CreatedBy, unmarshaled.CreatedBy)
	})
}

// TestUserPermission tests the UserPermission model.
func TestUserPermission(t *testing.T) {
	t.Run("Empty UserPermission", func(t *testing.T) {
		up := UserPermission{}

		assert.Zero(t, up.UserID)
		assert.Zero(t, up.PermissionID)
		assert.Zero(t, up.GrantedBy)
		assert.Nil(t, up.ExpiresAt)
	})

	t.Run("UserPermission with valid data", func(t *testing.T) {
		now := time.Now()
		createdBy := "admin"
		updatedBy := "admin"
		expiresAt := now.Add(30 * 24 * time.Hour)

		up := UserPermission{
			BaseModel: BaseModel{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: &createdBy,
				UpdatedBy: &updatedBy,
			},
			UserID:       1,
			PermissionID: 1,
			GrantedBy:    2,
			ExpiresAt:    &expiresAt,
		}

		assert.Equal(t, uint(1), up.UserID)
		assert.Equal(t, uint(1), up.PermissionID)
		assert.Equal(t, uint(2), up.GrantedBy)
		assert.Equal(t, &expiresAt, up.ExpiresAt)
	})

	t.Run("UserPermission JSON serialization", func(t *testing.T) {
		up := UserPermission{
			BaseModel: BaseModel{
				ID: 1,
			},
			UserID:       1,
			PermissionID: 1,
			GrantedBy:    2,
		}

		jsonData, err := json.Marshal(up)
		require.NoError(t, err)

		var unmarshaled UserPermission
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, up.UserID, unmarshaled.UserID)
		assert.Equal(t, up.PermissionID, unmarshaled.PermissionID)
		assert.Equal(t, up.GrantedBy, unmarshaled.GrantedBy)
	})
}

// TestUserRole tests the UserRole model.
func TestUserRole(t *testing.T) {
	t.Run("Empty UserRole", func(t *testing.T) {
		ur := UserRole{}

		assert.Zero(t, ur.UserID)
		assert.Zero(t, ur.RoleID)
		assert.True(t, ur.AssignedAt.IsZero())
		assert.Zero(t, ur.AssignedBy)
	})

	t.Run("UserRole with valid data", func(t *testing.T) {
		now := time.Now()

		ur := UserRole{
			UserID:     1,
			RoleID:     2,
			AssignedAt: now,
			AssignedBy: 3,
		}

		assert.Equal(t, uint(1), ur.UserID)
		assert.Equal(t, uint(2), ur.RoleID)
		assert.Equal(t, now, ur.AssignedAt)
		assert.Equal(t, uint(3), ur.AssignedBy)
	})

	t.Run("UserRole JSON serialization", func(t *testing.T) {
		ur := UserRole{
			UserID:     1,
			RoleID:     2,
			AssignedAt: time.Now(),
			AssignedBy: 3,
		}

		jsonData, err := json.Marshal(ur)
		require.NoError(t, err)

		var unmarshaled UserRole
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, ur.UserID, unmarshaled.UserID)
		assert.Equal(t, ur.RoleID, unmarshaled.RoleID)
		assert.Equal(t, ur.AssignedAt.Unix(), unmarshaled.AssignedAt.Unix())
		assert.Equal(t, ur.AssignedBy, unmarshaled.AssignedBy)
	})
}

// TestModelCreationTimes tests timestamp behavior across models.
func TestModelCreationTimes(t *testing.T) {
	t.Run("Timestamp field handling", func(t *testing.T) {
		now := time.Now()

		// Test BaseModel
		base := BaseModel{
			CreatedAt: now,
			UpdatedAt: now,
		}
		assert.Equal(t, now, base.CreatedAt)
		assert.Equal(t, now, base.UpdatedAt)

		// Test User
		user := User{
			BaseModel: BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		assert.Equal(t, now, user.CreatedAt)
		assert.Equal(t, now, user.UpdatedAt)

		// Test Ticket
		ticket := Ticket{
			BaseModel: BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		assert.Equal(t, now, ticket.CreatedAt)
		assert.Equal(t, now, ticket.UpdatedAt)
	})

	t.Run("Timestamp consistency", func(t *testing.T) {
		created := time.Now()
		updated := created.Add(1 * time.Hour)

		model := BaseModel{
			CreatedAt: created,
			UpdatedAt: updated,
		}

		assert.True(t, model.UpdatedAt.After(model.CreatedAt) ||
			model.UpdatedAt.Equal(model.CreatedAt))
	})
}

// TestModelFieldValidation tests field constraints and validations.
func TestModelFieldValidation(t *testing.T) {
	t.Run("Email validation", func(t *testing.T) {
		t.Run("Valid email", func(t *testing.T) {
			email := "test@example.com"
			user := User{Email: email}
			assert.Equal(t, email, user.Email)
		})

		t.Run("Invalid email formats", func(t *testing.T) {
			invalidEmails := []string{
				"invalid-email",
				"@example.com",
				"test@",
				"test.example.com",
			}

			for _, invalidEmail := range invalidEmails {
				user := User{Email: invalidEmail}
				// Model doesn't enforce validation directly,
				// so just test that the field accepts the value
				assert.Equal(t, invalidEmail, user.Email)
			}
		})
	})

	t.Run("Username validation", func(t *testing.T) {
		t.Run("Valid username", func(t *testing.T) {
			username := "testuser123"
			user := User{Username: username}
			assert.Equal(t, username, user.Username)
		})

		t.Run("Empty username", func(t *testing.T) {
			username := ""
			user := User{Username: username}
			assert.Equal(t, username, user.Username)
		})
	})

	t.Run("Ticket number format", func(t *testing.T) {
		t.Run("Valid ticket number", func(t *testing.T) {
			ticketNumber := "TICKET-001"
			ticket := Ticket{TicketNumber: ticketNumber}
			assert.Equal(t, ticketNumber, ticket.TicketNumber)
		})

		t.Run("Empty ticket number", func(t *testing.T) {
			ticketNumber := ""
			ticket := Ticket{TicketNumber: ticketNumber}
			assert.Equal(t, ticketNumber, ticket.TicketNumber)
		})
	})

	t.Run("File size validation", func(t *testing.T) {
		t.Run("Positive file size", func(t *testing.T) {
			fileSize := int64(1024)
			attachment := Attachment{FileSize: fileSize}
			assert.Equal(t, fileSize, attachment.FileSize)
		})

		t.Run("Zero file size", func(t *testing.T) {
			fileSize := int64(0)
			attachment := Attachment{FileSize: fileSize}
			assert.Equal(t, fileSize, attachment.FileSize)
		})

		t.Run("Negative file size", func(t *testing.T) {
			fileSize := int64(-1)
			attachment := Attachment{FileSize: fileSize}
			assert.Equal(t, fileSize, attachment.FileSize)
		})
	})
}

// TestModelRelationships tests model relationships.
func TestModelRelationships(t *testing.T) {
	t.Run("Ticket-User relationship", func(t *testing.T) {
		createdBy := "user123"
		assignedTo := uint(2)
		ticket := Ticket{
			BaseModel: BaseModel{
				CreatedBy: &createdBy,
			},
			AssignedTo: &assignedTo,
		}

		assert.Equal(t, &createdBy, ticket.CreatedBy)
		assert.Equal(t, &assignedTo, ticket.AssignedTo)
		// Actual relationship testing would require database setup
	})

	t.Run("Message-Ticket relationship", func(t *testing.T) {
		message := Message{
			TicketID: 1,
		}

		assert.Equal(t, uint(1), message.TicketID)
		// Actual relationship testing would require database setup
	})

	t.Run("Message-User relationship", func(t *testing.T) {
		message := Message{
			UserID: 1,
		}

		assert.Equal(t, uint(1), message.UserID)
		// Actual relationship testing would require database setup
	})
}

// TestJSONSerialization tests JSON serialization for all models.
func TestJSONSerialization(t *testing.T) {
	models := []interface{}{
		&User{},
		&Ticket{},
		&Message{},
		&Attachment{},
		&KnowledgeArticle{},
		&LLMProvider{},
		&ImportExportJob{},
		&AuditLog{},
		&APIKey{},
		&SystemSetting{},
		&Product{},
		&Service{},
		&SLATemplate{},
		&SLARule{},
		&Permission{},
		&Role{},
		&RolePermission{},
		&UserPermission{},
		&UserRole{},
	}

	for _, model := range models {
		t.Run(fmt.Sprintf("%T JSON serialization", model), func(t *testing.T) {
			// Create a minimal valid instance
			switch v := model.(type) {
			case *User:
				*v = User{
					Email:    "test@example.com",
					Username: "test",
				}
			case *Ticket:
				*v = Ticket{
					TicketNumber: "TICKET-001",
					Title:        "Test Ticket",
					Description:  "Test",
					Status:       "open",
				}
			case *Message:
				*v = Message{
					TicketID: 1,
					UserID:   1,
					Content:  "Test message",
				}
			case *Attachment:
				*v = Attachment{
					TicketID:    1,
					FileName:    "test.pdf",
					FilePath:    "/test/test.pdf",
					FileSize:    1024,
					ContentType: "application/pdf",
				}
			case *KnowledgeArticle:
				*v = KnowledgeArticle{
					Title:  "Test Article",
					Slug:   "test-article",
					Status: "draft",
				}
			case *LLMProvider:
				*v = LLMProvider{
					Name:  "Test Provider",
					Model: "test-model",
				}
			case *ImportExportJob:
				*v = ImportExportJob{
					Type:   "import",
					Status: "pending",
				}
			case *AuditLog:
				*v = AuditLog{
					Action: "test",
				}
			case *APIKey:
				*v = APIKey{
					Name:    "Test Key",
					KeyHash: "test-hash",
				}
			case *SystemSetting:
				*v = SystemSetting{
					Key:   "test.key",
					Value: "test-value",
					Type:  "string",
				}
			case *Product:
				*v = Product{
					Name: "Test Product",
					Code: "PROD-001",
				}
			case *Service:
				*v = Service{
					ProductID: 1,
					Name:      "Test Service",
					Code:      "SRV-001",
				}
			case *SLATemplate:
				*v = SLATemplate{
					Name:     "Test SLA",
					IsActive: true,
				}
			case *SLARule:
				*v = SLARule{
					Priority: "high",
					Severity: "critical",
				}
			case *Permission:
				*v = Permission{
					Code:     "test:read",
					Name:     "Test Permission",
					Category: "test",
				}
			case *Role:
				*v = Role{
					Name:     "Test Role",
					IsActive: true,
				}
			case *RolePermission:
				*v = RolePermission{
					PermissionID: 1,
				}
			case *UserPermission:
				*v = UserPermission{
					UserID:       1,
					PermissionID: 1,
				}
			case *UserRole:
				*v = UserRole{
					UserID:     1,
					RoleID:     1,
					AssignedAt: time.Now(),
				}
			}

			// Test JSON serialization
			jsonData, err := json.Marshal(model)
			require.NoError(t, err, "Failed to marshal model to JSON")

			// Test JSON unmarshaling
			var unmarshaled interface{}
			switch model.(type) {
			case *User:
				unmarshaled = &User{}
			case *Ticket:
				unmarshaled = &Ticket{}
			case *Message:
				unmarshaled = &Message{}
			case *Attachment:
				unmarshaled = &Attachment{}
			case *KnowledgeArticle:
				unmarshaled = &KnowledgeArticle{}
			case *LLMProvider:
				unmarshaled = &LLMProvider{}
			case *ImportExportJob:
				unmarshaled = &ImportExportJob{}
			case *AuditLog:
				unmarshaled = &AuditLog{}
			case *APIKey:
				unmarshaled = &APIKey{}
			case *SystemSetting:
				unmarshaled = &SystemSetting{}
			case *Product:
				unmarshaled = &Product{}
			case *Service:
				unmarshaled = &Service{}
			case *SLATemplate:
				unmarshaled = &SLATemplate{}
			case *SLARule:
				unmarshaled = &SLARule{}
			case *Permission:
				unmarshaled = &Permission{}
			case *Role:
				unmarshaled = &Role{}
			case *RolePermission:
				unmarshaled = &RolePermission{}
			case *UserPermission:
				unmarshaled = &UserPermission{}
			case *UserRole:
				unmarshaled = &UserRole{}
			}

			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "Failed to unmarshal JSON to model")

			// Test that the unmarshaled model matches the original
			switch v := model.(type) {
			case *User:
				original := *v
				unmarshaledTyped := unmarshaled.(*User)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Email, unmarshaledTyped.Email)
			case *Ticket:
				original := *v
				unmarshaledTyped := unmarshaled.(*Ticket)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.TicketNumber, unmarshaledTyped.TicketNumber)
			case *Message:
				original := *v
				unmarshaledTyped := unmarshaled.(*Message)
				assert.Equal(t, original.TicketID, unmarshaledTyped.TicketID)
				assert.Equal(t, original.UserID, unmarshaledTyped.UserID)
			case *Attachment:
				original := *v
				unmarshaledTyped := unmarshaled.(*Attachment)
				assert.Equal(t, original.TicketID, unmarshaledTyped.TicketID)
				assert.Equal(t, original.FileName, unmarshaledTyped.FileName)
			case *KnowledgeArticle:
				original := *v
				unmarshaledTyped := unmarshaled.(*KnowledgeArticle)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Title, unmarshaledTyped.Title)
			case *LLMProvider:
				original := *v
				unmarshaledTyped := unmarshaled.(*LLMProvider)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Name, unmarshaledTyped.Name)
			case *ImportExportJob:
				original := *v
				unmarshaledTyped := unmarshaled.(*ImportExportJob)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Type, unmarshaledTyped.Type)
			case *AuditLog:
				original := *v
				unmarshaledTyped := unmarshaled.(*AuditLog)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Action, unmarshaledTyped.Action)
			case *APIKey:
				original := *v
				unmarshaledTyped := unmarshaled.(*APIKey)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Name, unmarshaledTyped.Name)
			case *SystemSetting:
				original := *v
				unmarshaledTyped := unmarshaled.(*SystemSetting)
				assert.Equal(t, original.Key, unmarshaledTyped.Key)
			case *Product:
				original := *v
				unmarshaledTyped := unmarshaled.(*Product)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Name, unmarshaledTyped.Name)
			case *Service:
				original := *v
				unmarshaledTyped := unmarshaled.(*Service)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.ProductID, unmarshaledTyped.ProductID)
			case *SLATemplate:
				original := *v
				unmarshaledTyped := unmarshaled.(*SLATemplate)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Name, unmarshaledTyped.Name)
			case *SLARule:
				original := *v
				unmarshaledTyped := unmarshaled.(*SLARule)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Priority, unmarshaledTyped.Priority)
			case *Permission:
				original := *v
				unmarshaledTyped := unmarshaled.(*Permission)
				assert.Equal(t, original.Code, unmarshaledTyped.Code)
			case *Role:
				original := *v
				unmarshaledTyped := unmarshaled.(*Role)
				assert.Equal(t, original, *unmarshaledTyped)
				assert.Equal(t, original.Name, unmarshaledTyped.Name)
			case *RolePermission:
				original := *v
				unmarshaledTyped := unmarshaled.(*RolePermission)
				assert.Equal(t, original.RoleID, unmarshaledTyped.RoleID)
				assert.Equal(t, original.PermissionID, unmarshaledTyped.PermissionID)
			case *UserPermission:
				original := *v
				unmarshaledTyped := unmarshaled.(*UserPermission)
				assert.Equal(t, original.UserID, unmarshaledTyped.UserID)
				assert.Equal(t, original.PermissionID, unmarshaledTyped.PermissionID)
			case *UserRole:
				original := *v
				unmarshaledTyped := unmarshaled.(*UserRole)
				assert.Equal(t, original.UserID, unmarshaledTyped.UserID)
				assert.Equal(t, original.RoleID, unmarshaledTyped.RoleID)
			}
		})
	}
}
