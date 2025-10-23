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

	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/server"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "smartticket",
		Short: "SmartTicket is a self-hosted multi-tenant ticketing platform",
		Long: `SmartTicket is a self-hosted multi-tenant ticketing and knowledge
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
		&models.Tenant{},
		&models.SystemSetting{},
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

	migrator := database.NewMigrator(db.DB)
	if err := migrator.AutoMigrate(dbModels...); err != nil {
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

	// TODO: Fix database model foreign key constraints before re-enabling initialization
	// Initialize database if this is first startup
	// logger.Debug("Checking if database initialization is needed")
	// initializer := database.NewInitializer(db.DB)
	// if err := initializer.InitializeIfNeeded(context.Background()); err != nil {
	// 	logger.Error("Failed to initialize database", zap.Error(err))
	// 	return fmt.Errorf("failed to initialize database: %w", err)
	// }

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

	// Run SQL migrations from files
	logger.Debug("Running SQL migrations from files")
	migrator := database.NewMigrator(db.DB)
	if err := migrator.RunMigrations("migrations"); err != nil {
		logger.Error("Failed to run SQL migrations", zap.Error(err))
		return fmt.Errorf("failed to run SQL migrations: %w", err)
	}

	// Auto-migrate all models in correct order to avoid foreign key issues
	logger.Debug("Running GORM auto-migrations")
	dbModels := []interface{}{
		// Base tables first (no foreign key dependencies)
		&models.Tenant{},
		&models.SystemSetting{},
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

	if err := migrator.AutoMigrate(dbModels...); err != nil {
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
