package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// Initializer handles first-time database initialization.
type Initializer struct {
	db *gorm.DB
}

// NewInitializer creates a new database initializer.
func NewInitializer(db *gorm.DB) *Initializer {
	return &Initializer{db: db}
}

// InitializeIfNeeded checks if database needs initialization and performs it.
func (i *Initializer) InitializeIfNeeded(ctx context.Context) error {
	logger := zap.L().Named("database.initializer")

	// Check if database is already initialized by looking for existing users
	var userCount int64
	if err := i.db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		logger.Error("Failed to check database initialization status", zap.Error(err))
		return fmt.Errorf("failed to check database initialization status: %w", err)
	}

	// If we already have users, assume database is initialized
	if userCount > 0 {
		logger.Info("Database already initialized", zap.Int64("user_count", userCount))
		return nil
	}

	logger.Info("First-time database startup detected, initializing with essential data...")

	// Seed the database with essential data
	if err := i.seedEssentialData(); err != nil {
		logger.Error("Failed to seed database with essential data", zap.Error(err))
		return fmt.Errorf("failed to seed database: %w", err)
	}

	logger.Info("Database initialization completed successfully")
	return nil
}

// seedEssentialData seeds the database with essential data for first startup.
func (i *Initializer) seedEssentialData() error {
	now := time.Now()
	logger := zap.L().Named("database.initializer")

	// Use a transaction to ensure atomicity
	return i.db.Transaction(func(tx *gorm.DB) error {

		// Generate admin password hash
		adminPasswordHash, err := generatePasswordHash("admin123")
		if err != nil {
			return fmt.Errorf("failed to generate admin password hash: %w", err)
		}

		// Create default admin user
		adminUser := models.User{
			Email:        "admin@smartticket.local",
			Username:     "admin",
			PasswordHash: adminPasswordHash,
			FirstName:    "System",
			LastName:     "Administrator",
			Role:         "admin",
			IsActive:     true,
			Preferences:  `{"timezone": "UTC", "language": "en"}`,
		}

		if err := tx.Create(&adminUser).Error; err != nil {
			return fmt.Errorf("failed to create default admin user: %w", err)
		}
		adminID := adminUser.ID
		logger.Info("Created default admin user", zap.String("email", "admin@smartticket.local"), zap.Uint("id", adminID))

		// Create system roles
		systemRoles := []models.Role{
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:        "admin",
				Description: "System administrator with full access",
				IsSystem:   true,
				// System role (tenant_id = 0)
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:        "tenant_admin",
				Description: "Tenant administrator with tenant-wide access",
				IsSystem:   false,
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:        "engineer",
				Description: "Support engineer with technical access",
				IsSystem:   false,
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:        "customer",
				Description: "Customer with basic access",
				IsSystem:   false,
			},
		}

		// Create roles
		for _, role := range systemRoles {
			if err := tx.Create(&role).Error; err != nil {
				return fmt.Errorf("failed to create role %s: %w", role.Name, err)
			}
		}
		logger.Info("Created system roles", zap.Int("count", len(systemRoles)))

		// Get the admin role ID
		var adminRole models.Role
		if err := tx.Where("name = ? AND tenant_id = ?", "admin", uint(0)).First(&adminRole).Error; err != nil {
			return fmt.Errorf("failed to find admin role: %w", err)
		}

		// Assign admin role to admin user
		adminUserRole := models.UserRole{
			UserID:     adminID,
			RoleID:     adminRole.ID,
			AssignedAt: now,
			AssignedBy: adminID, // Self-assigned by admin
		}

		if err := tx.Create(&adminUserRole).Error; err != nil {
			return fmt.Errorf("failed to assign admin role to admin user: %w", err)
		}
		logger.Info("Assigned admin role to admin user", zap.Uint("user_id", adminID), zap.Uint("role_id", adminRole.ID))

	// Create essential system settings
		systemSettings := []models.SystemSetting{
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Key:         "system.timezone",
				Value:       "UTC",
				Type:        "string",
				Description: "System timezone",
				IsPublic:    true,
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Key:         "system.language",
				Value:       "en",
				Type:        "string",
				Description: "System language",
				IsPublic:    true,
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Key:         "tickets.auto_number",
				Value:       "true",
				Type:        "boolean",
				Description: "Automatically generate ticket numbers",
				IsPublic:    false,
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Key:         "notifications.email_enabled",
				Value:       "false", // Disabled by default for development
				Type:        "boolean",
				Description: "Enable email notifications",
				IsPublic:    false,
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Key:         "security.session_timeout",
				Value:       "3600",
				Type:        "integer",
				Description: "Session timeout in seconds",
				IsPublic:    false,
			},
			{
				BaseModel: models.BaseModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Key:         "system.version",
				Value:       "1.0.0",
				Type:        "string",
				Description: "System version",
				IsPublic:    true,
			},
		}

		// Create system settings
		for _, setting := range systemSettings {
			if err := tx.Create(&setting).Error; err != nil {
				return fmt.Errorf("failed to create system setting %s: %w", setting.Key, err)
			}
		}
		logger.Info("Created system settings", zap.Int("count", len(systemSettings)))

		// Create default LLM provider configuration (disabled by default)
		defaultLLMProvider := models.LLMProvider{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:          "OpenAI GPT",
			ProviderType:  "openai",
			APIEndpoint:   "https://api.openai.com/v1",
			APIKey:        "", // To be configured by admin
			Model:         "gpt-4o-mini",
			MaxTokens:     4096,
			Temperature:   0.7,
			TaskTypes:     `["chat", "generation", "summarization"]`,
			IsDefault:     true,
			IsEnabled:     false, // Disabled by default
			QuotaLimit:    10000,
			QuotaUsed:     0,
			Configuration: `{"model": "gpt-4o-mini", "temperature": 0.7}`,
		}

		if err := tx.Create(&defaultLLMProvider).Error; err != nil {
			return fmt.Errorf("failed to create default LLM provider: %w", err)
		}
		logger.Info("Created default LLM provider", zap.String("name", defaultLLMProvider.Name))

		// Create welcome knowledge article
		welcomeArticle := models.KnowledgeArticle{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			
			Title:    "Welcome to SmartTicket",
			Slug:     "welcome-to-smartticket",
			Content: `# Welcome to SmartTicket

SmartTicket is your self-hosted multi-tenant ticketing and knowledge collaboration platform.

## Getting Started

1. **Admin Account**: Your system administrator account has been created with:
   - Email: admin@smartticket.local
   - Password: admin123

2. **Security**: Please change your default password after first login.

3. **Configuration**: Configure LLM providers in the settings to enable AI features.

## Features

- **Ticket Management**: Create, track, and manage support tickets
- **Knowledge Base**: Build and maintain a knowledge base
- **Multi-tenant Support**: Multiple organizations on the same platform
- **AI Integration**: Connect your preferred LLM providers
- **Data Export**: Full data portability and backup features

## Need Help?

Check the knowledge base for more articles or create a support ticket.

Thank you for choosing SmartTicket!`,
			ContentType:  "markdown",
			Summary:      "Welcome guide for new SmartTicket installations",
			AuthorID:     adminID,
			Status:       "published",
			Visibility:   "public",
			AccessLevel:  "all",
			Category:     "Getting Started",
			Tags:         `["welcome", "getting-started", "admin"]`,
			Views:        0,
			HelpfulVotes: 0,
			Version:      1,
		}

		if err := tx.Create(&welcomeArticle).Error; err != nil {
			return fmt.Errorf("failed to create welcome knowledge article: %w", err)
		}
		logger.Info("Created welcome knowledge article", zap.String("title", welcomeArticle.Title))

		return nil
	})
}

// PrintWelcomeInfo prints welcome information after first-time initialization.
func (i *Initializer) PrintWelcomeInfo() {
	log.Println("🎉 SmartTicket has been initialized successfully!")
	log.Println("")
	log.Println("=== Default Login Information ===")
	log.Println("URL: http://localhost:6533")
	log.Println("Email: admin@smartticket.local")
	log.Println("Password: admin123")
	log.Println("")
	log.Println("⚠️  IMPORTANT SECURITY NOTICE:")
	log.Println("Please change the default password after first login!")
	log.Println("Configure your LLM providers in the admin settings to enable AI features.")
	log.Println("")
	log.Println("📚 Next Steps:")
	log.Println("1. Login with the admin credentials")
	log.Println("2. Change the default password")
	log.Println("3. Configure system settings")
	log.Println("4. Set up LLM providers (optional)")
	log.Println("5. Create additional users and organizations as needed")
}

// generatePasswordHash creates a bcrypt hash for the given password.
func generatePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate password hash: %w", err)
	}
	return string(hash), nil
}
