// Package main provides the SmartTicket server CLI application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"strings"

	"crypto/sha256"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/customer"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/knowledge"
	"github.com/company/smartticket/internal/knowledgebase"
	"github.com/company/smartticket/internal/linbit"
	"github.com/company/smartticket/internal/llm"
	"github.com/company/smartticket/internal/logger"
	mcpserver "github.com/company/smartticket/internal/mcp"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/server"
	"github.com/company/smartticket/internal/services"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "smartticket",
		Short: "SmartTicket is a self-hosted single-tenant ticketing platform",
		Long: `SmartTicket is a self-hosted single-tenant ticketing and knowledge
collaboration platform designed for enterprise deployment.`,
		Version: fmt.Sprintf("%s (built %s)", version, buildTime),
	}

	// Add serve command
	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the SmartTicket server",
		Long:  "Start the SmartTicket HTTP server and begin serving requests",
		RunE:  runServe,
	}
	serveCmd.Flags().String("config", "", "Configuration file path")
	rootCmd.AddCommand(serveCmd)

	// Add migrate command
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long:  "Run database migrations to set up the database schema",
		RunE:  runMigrate,
	}
	migrateCmd.Flags().String("config", "", "Configuration file path")
	rootCmd.AddCommand(migrateCmd)

	// Add mcp command
	var mcpCmd = &cobra.Command{
		Use:   "mcp",
		Short: "Start the SmartTicket MCP server",
		Long: `Start the SmartTicket Model Context Protocol (MCP) server, exposing
ticketing and knowledge operations as MCP tools. Serves stdio by default, or
Streamable HTTP when --http is provided.`,
		RunE: runMCP,
	}
	mcpCmd.Flags().String("config", "", "Configuration file path")
	mcpCmd.Flags().String("http", "", "Serve Streamable HTTP on this address (default :43517 when flag is set without a value); if unset, serve stdio")
	mcpCmd.Flags().Lookup("http").NoOptDefVal = ":43517"
	mcpCmd.Flags().String("token", os.Getenv("SMARTTICKET_MCP_TOKEN"), "JWT credential for stdio transport (default from SMARTTICKET_MCP_TOKEN)")
	mcpCmd.Flags().String("toolsets", "", "Comma-separated toolsets to enable (default: all)")
	rootCmd.AddCommand(mcpCmd)

	// Add createadmin command
	var createAdminCmd = &cobra.Command{
		Use:   "createadmin",
		Short: "Create or update an administrator account",
		Long: `Create (or update, if the email already exists) an administrator user
and assign the admin role. Optionally set the deployment's organization name.`,
		RunE: runCreateAdmin,
	}
	createAdminCmd.Flags().String("config", "", "Configuration file path")
	createAdminCmd.Flags().String("email", "", "Administrator email (required)")
	createAdminCmd.Flags().String("password", "", "Administrator password (required)")
	createAdminCmd.Flags().String("username", "", "Username (defaults to the email local-part)")
	createAdminCmd.Flags().String("name", "", "Full name (defaults to \"Administrator\")")
	createAdminCmd.Flags().String("org", "", "Organization/team name to record for this deployment")
	_ = createAdminCmd.MarkFlagRequired("email")
	_ = createAdminCmd.MarkFlagRequired("password")
	rootCmd.AddCommand(createAdminCmd)

	// Add importlinbit command
	var importLinbitCmd = &cobra.Command{
		Use:   "importlinbit",
		Short: "Import LINBIT UG9 docs as knowledge articles and create the LINBIT customer",
		Long: `Fetch the LINBIT UG9 English documentation from GitHub, import each
AsciiDoc file as a published knowledge article (auto-indexed into CortexDB when
an embedding provider is configured), and create a "LINBIT" customer.`,
		RunE: runImportLinbit,
	}
	importLinbitCmd.Flags().String("config", "", "Configuration file path")
	importLinbitCmd.Flags().String("prefix", "", "Only import files whose name starts with this (e.g. \"drbd-\")")
	importLinbitCmd.Flags().Int("max-kb", 0, "Skip files larger than this many KB (0 = no limit)")
	rootCmd.AddCommand(importLinbitCmd)

	// Add version command
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			// Initialize a simple console logger for version output
			consoleLogger, _ := zap.NewDevelopment()
			defer func() {
				_ = consoleLogger.Sync()
			}()

			consoleLogger.Info("SmartTicket version",
				zap.String("version", version),
				zap.String("built", buildTime),
				zap.String("go", runtime.Version()),
			)
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}

func runServe(cmd *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.LoadFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	if err := logger.InitializeGlobalLogger(&cfg.Logger); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// Log server startup
	logger.Info("Starting SmartTicket server",
		zap.String("environment", cfg.Environment),
		zap.String("version", version),
		zap.String("server_address", cfg.GetServerAddress()),
		zap.String("database", cfg.Database.ConnectionURL),
	)

	// Initialize database connection
	logger.Debug("Initializing database connection")
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Test database connection
	logger.Debug("Testing database connection")
	if !db.IsHealthy() {
		logger.Error("Database connection is not healthy")
		return fmt.Errorf("database connection is not healthy")
	}
	logger.Info("Database connection established successfully")

	// Auto-migrate all models in correct order to avoid foreign key issues
	logger.Debug("Running database migrations")
	dbModels := []interface{}{
		// Base tables first (no foreign key dependencies)
		&models.SystemSetting{},
		&models.Customer{},
		&models.Product{},
		&models.Service{},
		&models.SLATemplate{},
		&models.SLARule{},
		&models.LLMProvider{},

		// Core business tables (only depend on base tables)
		&models.User{},
		&models.KnowledgeArticle{},
		&models.APIKey{},

		// Permission system tables (depend on users)
		&models.Permission{},
		&models.Role{},

		// Relationship tables (depend on core tables)
		&models.RolePermission{},
		&models.UserPermission{},
		&models.UserRole{},

		// Dependent business tables (depend on core tables)
		&models.Ticket{},
		&models.Message{},
		&models.Attachment{},
		&models.ImportExportJob{},
		&models.AuditLog{},
	}

	// Run GORM AutoMigrate for all models
	if err := db.DB.AutoMigrate(dbModels...); err != nil {
		logger.Error("Failed to auto-migrate models", zap.Error(err))
		return fmt.Errorf("failed to auto-migrate models: %w", err)
	}

	logger.Info("Database models migrated successfully", zap.Int("model_count", len(dbModels)))

	// Re-enable foreign key constraints after migration is complete
	logger.Debug("Re-enabling foreign key constraints")
	if err := db.EnableForeignKeys(); err != nil {
		logger.Error("Failed to enable foreign key constraints", zap.Error(err))
		return fmt.Errorf("failed to enable foreign key constraints: %w", err)
	}
	logger.Info("Foreign key constraints enabled successfully")

	// Initialize database if this is first startup
	logger.Debug("Checking if database initialization is needed")
	initializer := database.NewInitializer(db.DB)
	if err := initializer.InitializeIfNeeded(context.Background()); err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Set up HTTP server
	logger.Debug("Setting up HTTP server")
	httpServer := server.NewServer(cfg, db)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server")
		if err := httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", zap.Error(err))
			cancel()
		}
	}()

	logger.Info("Server started successfully",
		zap.String("address", cfg.GetServerAddress()),
		zap.String("environment", cfg.Environment),
	)
	logger.Info("Press Ctrl+C to stop the server")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Server context cancelled")
	}

	// Graceful shutdown with timeout
	logger.Info("Initiating graceful shutdown")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
		return err
	}

	logger.Info("Server shutdown complete")
	return nil
}

func runMigrate(cmd *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.LoadFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	if err := logger.InitializeGlobalLogger(&cfg.Logger); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("Running database migrations",
		zap.String("environment", cfg.Environment),
		zap.String("database", cfg.Database.ConnectionURL),
	)

	// Initialize database connection
	logger.Debug("Initializing database connection for migrations")
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Test database connection
	if !db.IsHealthy() {
		logger.Error("Database connection is not healthy")
		return fmt.Errorf("database connection is not healthy")
	}

	// Auto-migrate all models in correct order to avoid foreign key issues
	logger.Debug("Running GORM auto-migrations")
	dbModels := []interface{}{
		// Base tables first (no foreign key dependencies)
		&models.SystemSetting{},
		&models.Customer{},
		&models.Product{},
		&models.Service{},
		&models.SLATemplate{},
		&models.SLARule{},
		&models.LLMProvider{},

		// Core business tables (only depend on base tables)
		&models.User{},
		&models.KnowledgeArticle{},
		&models.APIKey{},

		// Permission system tables (depend on users)
		&models.Permission{},
		&models.Role{},

		// Relationship tables (depend on core tables)
		&models.RolePermission{},
		&models.UserPermission{},
		&models.UserRole{},

		// Dependent business tables (depend on core tables)
		&models.Ticket{},
		&models.Message{},
		&models.Attachment{},
		&models.ImportExportJob{},
		&models.AuditLog{},
	}

	// Run GORM AutoMigrate
	if err := db.DB.AutoMigrate(dbModels...); err != nil {
		logger.Error("Failed to auto-migrate models", zap.Error(err))
		return fmt.Errorf("failed to auto-migrate models: %w", err)
	}

	// Re-enable foreign key constraints after migration is complete
	logger.Debug("Re-enabling foreign key constraints")
	if err := db.EnableForeignKeys(); err != nil {
		logger.Error("Failed to enable foreign key constraints", zap.Error(err))
		return fmt.Errorf("failed to enable foreign key constraints: %w", err)
	}
	logger.Info("Foreign key constraints enabled successfully")

	logger.Info("Database migration completed successfully",
		zap.Int("model_count", len(dbModels)),
	)
	return nil
}

func runCreateAdmin(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := logger.InitializeGlobalLogger(&cfg.Logger); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	email, _ := cmd.Flags().GetString("email")
	password, _ := cmd.Flags().GetString("password")
	username, _ := cmd.Flags().GetString("username")
	name, _ := cmd.Flags().GetString("name")
	org, _ := cmd.Flags().GetString("org")

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return fmt.Errorf("both --email and --password are required and must be non-empty")
	}
	if username == "" {
		username = strings.SplitN(email, "@", 2)[0]
	}
	if name == "" {
		name = "Administrator"
	}

	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Ensure the tables this command touches exist (idempotent).
	if err := db.DB.AutoMigrate(&models.SystemSetting{}, &models.User{}, &models.Role{}, &models.UserRole{}, &models.Permission{}, &models.RolePermission{}); err != nil {
		return fmt.Errorf("failed to migrate required tables: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	// Ensure all standard roles, the permission catalog, and role→permission
	// grants exist. Bootstrapping via createadmin creates a user before serve
	// runs, which makes InitializeIfNeeded skip its seeding; without this, the
	// engineer/customer roles and permission grants would be missing.
	if err := database.EnsureRolesAndPermissions(db.DB); err != nil {
		return fmt.Errorf("failed to seed roles and permissions: %w", err)
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Look up the admin role for assignment to this user.
		var adminRole models.Role
		if err := tx.Where(models.Role{Name: "admin"}).First(&adminRole).Error; err != nil {
			return fmt.Errorf("failed to load admin role: %w", err)
		}

		// Create or update the user.
		var user models.User
		err := tx.Where("email = ?", email).First(&user).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			user = models.User{
				Email:        email,
				Username:     username,
				PasswordHash: string(hash),
				FirstName:    name,
				Role:         "admin",
				IsActive:     true,
				Preferences:  `{"timezone": "UTC", "language": "en"}`,
			}
			if err := tx.Create(&user).Error; err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
		case err != nil:
			return fmt.Errorf("failed to look up user: %w", err)
		default:
			user.PasswordHash = string(hash)
			user.Username = username
			user.FirstName = name
			user.Role = "admin"
			user.IsActive = true
			if err := tx.Save(&user).Error; err != nil {
				return fmt.Errorf("failed to update user: %w", err)
			}
		}

		// Ensure the admin role is assigned to the user.
		var userRole models.UserRole
		if err := tx.Where(models.UserRole{UserID: user.ID, RoleID: adminRole.ID}).
			Attrs(models.UserRole{AssignedAt: now, AssignedBy: user.ID}).
			FirstOrCreate(&userRole).Error; err != nil {
			return fmt.Errorf("failed to assign admin role: %w", err)
		}

		// Optionally record the organization/team name.
		if org = strings.TrimSpace(org); org != "" {
			var setting models.SystemSetting
			if err := tx.Where(models.SystemSetting{Key: "system.organization_name"}).
				Assign(models.SystemSetting{
					Value:       org,
					Type:        "string",
					Description: "Organization/team name for this deployment",
					IsPublic:    true,
				}).
				FirstOrCreate(&setting).Error; err != nil {
				return fmt.Errorf("failed to set organization name: %w", err)
			}
		}

		logger.Info("Administrator account ready",
			zap.String("email", email),
			zap.String("username", username),
			zap.Uint("user_id", user.ID),
			zap.String("organization", org),
		)
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("Administrator %q is ready (role: admin)", email)
	if org != "" {
		fmt.Printf("; organization set to %q", org)
	}
	fmt.Println()
	return nil
}

func runImportLinbit(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := logger.InitializeGlobalLogger(&cfg.Logger); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if !db.IsHealthy() {
		return fmt.Errorf("database connection is not healthy")
	}

	// Ensure the tables this command touches exist (idempotent).
	if err := db.DB.AutoMigrate(&models.User{}, &models.Customer{}, &models.KnowledgeArticle{}); err != nil {
		return fmt.Errorf("failed to migrate required tables: %w", err)
	}

	// Build the AI stack the same way server.go does, so created articles are
	// auto-indexed into CortexDB when an embedding provider is configured.
	var llmService *llm.Service
	var kbStore *knowledgebase.Store

	secretKey, kerr := llm.LoadKey(cfg.SecretKeyRaw)
	if kerr != nil {
		logger.Warn("SMARTTICKET_SECRET_KEY not set or invalid; deriving encryption key from JWT secret")
		sum := sha256.Sum256([]byte(cfg.JWT.Secret))
		secretKey = sum[:]
	}
	cipher, cerr := llm.NewCipher(secretKey)
	if cerr != nil {
		logger.Warn("llm cipher init failed; AI indexing disabled", zap.Error(cerr))
	} else {
		llmService = llm.NewService(db.DB, cipher)
		embedder := knowledgebase.NewProviderEmbedder(func(ctx context.Context, texts []string) ([][]float32, error) {
			ep, key, err := llmService.ResolveEmbedding()
			if err != nil {
				return nil, err
			}
			return llm.NewClient(ep.APIEndpoint, key).Embed(ctx, ep.Model, ep.Dimensions, texts)
		}, 1024)
		store, oerr := knowledgebase.Open("./data/cortex.db", embedder)
		if oerr != nil {
			logger.Warn("cortexdb unavailable; AI indexing disabled", zap.Error(oerr))
		} else {
			kbStore = store
			defer func() { _ = kbStore.Close() }()
		}
	}

	knowledgeService := knowledge.NewService(db.DB, kbStore, llmService)
	customerService := customer.NewService(db.DB)

	aiReady := kbStore != nil && kbStore.Healthy() && kbStore.DB().HasEmbedder()

	// Resolve an author user id: prefer the first admin, fall back to any user.
	var authorID uint
	var admin models.User
	if err := db.DB.Where("role = ?", "admin").Order("id asc").First(&admin).Error; err == nil {
		authorID = admin.ID
	} else {
		var anyUser models.User
		if err := db.DB.Order("id asc").First(&anyUser).Error; err == nil {
			authorID = anyUser.ID
		} else {
			return fmt.Errorf("no users found; run 'createadmin' first so imported articles have an author")
		}
	}

	importer := linbit.NewImporter(db.DB, knowledgeService, customerService, aiReady)
	importer.NamePrefix, _ = cmd.Flags().GetString("prefix")
	maxKB, _ := cmd.Flags().GetInt("max-kb")
	importer.MaxBytes = maxKB * 1024
	res, err := importer.Run(context.Background(), authorID)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	logger.Info("LINBIT import complete",
		zap.Int("files_found", res.FilesFound),
		zap.Int("articles_created", res.ArticlesCreated),
		zap.Int("articles_skipped", res.ArticlesSkipped),
		zap.Bool("customer_created", res.CustomerCreated),
		zap.Bool("embedding_active", res.EmbeddingActive),
	)

	fmt.Printf("LINBIT import: %d files found, %d created, %d skipped; customer %s\n",
		res.FilesFound, res.ArticlesCreated, res.ArticlesSkipped,
		map[bool]string{true: "created", false: "already exists"}[res.CustomerCreated])
	if !res.EmbeddingActive {
		fmt.Println("No embedding provider configured — run reindex after configuring one")
	}
	return nil
}

func runMCP(cmd *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.LoadFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	if err := logger.InitializeGlobalLogger(&cfg.Logger); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// Initialize database connection
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if !db.IsHealthy() {
		return fmt.Errorf("database connection is not healthy")
	}

	// Construct shared services and the MCP backend.
	authService := auth.NewService(
		db.DB,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.JWT.Issuer,
	)
	permissionService := services.NewPermissionService(db.DB)

	backend := mcpserver.NewDirectBackend(db.DB, authService, permissionService)
	authn := mcpserver.NewAuthenticator(authService, permissionService)

	// Parse toolsets flag.
	toolsetsFlag, _ := cmd.Flags().GetString("toolsets")
	var toolsets []string
	if strings.TrimSpace(toolsetsFlag) != "" {
		for _, t := range strings.Split(toolsetsFlag, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				toolsets = append(toolsets, trimmed)
			}
		}
	}

	mcpSrv := mcpserver.NewMCPServer(backend, toolsets)

	httpAddr, _ := cmd.Flags().GetString("http")
	token, _ := cmd.Flags().GetString("token")

	ctx := context.Background()

	if httpAddr != "" {
		logger.Info("Starting MCP server (Streamable HTTP)", zap.String("address", httpAddr))
		return mcpserver.RunHTTP(ctx, mcpSrv, authn, httpAddr)
	}

	logger.Info("Starting MCP server (stdio)")
	return mcpserver.RunStdio(ctx, mcpSrv, authn, token)
}
