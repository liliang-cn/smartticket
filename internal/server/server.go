package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/api/handlers"
	"github.com/company/smartticket/internal/api/middleware"
	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/importexport"
	"github.com/company/smartticket/internal/knowledge"
	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/product"
	servicemgmt "github.com/company/smartticket/internal/service"
	"github.com/company/smartticket/internal/services"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/internal/ticket"
	"github.com/company/smartticket/internal/user"
)

// Server represents the HTTP server.
type Server struct {
	config               *config.Config
	router               *gin.Engine
	server               *http.Server
	db                   *database.Database
	authService          *auth.Service
	permissionMiddleware *middleware.PermissionMiddleware
}

// NewServer creates a new HTTP server instance.
func NewServer(cfg *config.Config, db *database.Database) *Server {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsTest() {
		gin.SetMode(gin.TestMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	// Enable trailing slash redirect for consistent API behavior
	router.RedirectTrailingSlash = true
	router.RedirectFixedPath = true

	// Initialize auth service
	authService := auth.NewService(
		db.DB,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.JWT.Issuer,
	)

	server := &Server{
		config:      cfg,
		router:      router,
		db:          db,
		authService: authService,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware configures server middleware.
func (s *Server) setupMiddleware() {
	// Custom recovery middleware with error handling
	s.router.Use(errors.RecoveryMiddleware())

	// Error handling middleware (should be early in the chain)
	s.router.Use(errors.ErrorMiddleware())

	// Request ID middleware (should be before other middleware)
	s.router.Use(s.requestIDMiddleware())

	// Structured logging middleware
	s.router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID from context
		requestID, exists := c.Get("request_id")
		var requestIDStr string
		if !exists {
			requestIDStr = ""
		} else if requestIDVal, ok := requestID.(string); ok {
			requestIDStr = requestIDVal
		} else {
			requestIDStr = ""
		}

		// Get client info
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		userAgent := c.Request.UserAgent()

		// Build full path
		if raw != "" {
			path = path + "?" + raw
		}

		// Log request
		logger.LogRequest(method, path, clientIP, userAgent, statusCode, latency, requestIDStr)
	})

	// CORS middleware
	s.setupCORS()

	// Rate limiting middleware
	s.setupRateLimiting()

	// Security headers middleware
	s.setupSecurityHeaders()

	// Validation middleware for request binding
	s.router.Use(errors.ValidationMiddleware())

	// Set up custom error handlers
	s.router.NoRoute(errors.NotFoundHandler)
	s.router.NoMethod(errors.MethodNotAllowedHandler)
}

// setupRoutes configures server routes.
func (s *Server) setupRoutes() {
	// Initialize services and handlers
	authRepo := auth.NewRepository(s.db.DB)
	userService := user.NewService(s.db.DB, authRepo, s.authService)

	// Initialize permission service
	permissionService := services.NewPermissionService(s.db.DB)
	permissionMiddleware := middleware.NewPermissionMiddleware(permissionService)
	s.permissionMiddleware = permissionMiddleware

	// Initialize SLA calculator
	slaCalculator := sla.NewCalculator(s.db.DB)

	// Initialize ticket service with SLA calculator
	ticketService := ticket.NewService(s.db.DB, slaCalculator)
	productService := product.NewService(s.db.DB)
	serviceManagementService := servicemgmt.NewService(s.db.DB)
	slaService := sla.NewService(s.db.DB)
	knowledgeService := knowledge.NewService(s.db.DB)
	importExportService := importexport.NewService(s.db.DB)

	authHandlers := auth.NewHandlers(s.authService)
	userHandlers := user.NewHandlers(userService)
	ticketHandlers := ticket.NewHandlers(ticketService)
	productHandlers := product.NewHandlers(productService)
	serviceHandlers := servicemgmt.NewHandlers(serviceManagementService)
	slaHandlers := sla.NewHandlers(slaService, slaCalculator)
	knowledgeHandlers := knowledge.NewHandlers(knowledgeService)
	importExportHandlers := importexport.NewHandlers(importExportService)
	permissionHandlers := handlers.NewPermissionHandler(permissionService)
	roleHandlers := handlers.NewRoleHandler(permissionService)

	// Health check endpoints (no tenant validation required)
	health := s.router.Group("/")
	{
		health.GET("/health", s.healthCheck)
		health.GET("/healthz", s.healthCheck) // Alternative health endpoint
		health.GET("/api/v1/health", s.healthCheck)

		// API version info (no tenant validation required)
		health.GET("/version", s.versionInfo)
		health.GET("/api/v1/version", s.versionInfo)

		// Swagger documentation
		health.GET("/swagger/*any", s.serveSwaggerUI)
		health.GET("/swagger.yaml", s.serveSwaggerYAML)
	}

	// API routes group - authentication endpoints without tenant isolation
	authPublic := s.router.Group("/api/v1/auth")
	{
		authPublic.POST("/login", authHandlers.Login)
		authPublic.POST("/refresh", authHandlers.RefreshToken)
	}

	// API routes group for all other endpoints
	api := s.router.Group("/api/v1")
	{
		// Public endpoints (no auth required)
		public := api.Group("/")
		{
			public.GET("/info", s.appInfo)
		}

		// Protected endpoints (auth required)
		protected := api.Group("/")
		protected.Use(s.authMiddleware())
		{
			// Authentication endpoints (auth required)
			protectedAuth := protected.Group("/auth")
			{
				protectedAuth.POST("/logout", authHandlers.Logout)
				protectedAuth.GET("/profile", authHandlers.GetProfile)
				protectedAuth.POST("/change-password", authHandlers.ChangePassword)
				protectedAuth.GET("/me", authHandlers.GetMe)
				protectedAuth.GET("/validate", authHandlers.ValidateToken)
			}

			// User management routes (new implementation)
			users := protected.Group("/users")
			{
				users.GET("", userHandlers.ListUsers)
				users.GET("/stats", userHandlers.GetUserStats)
				users.POST("", userHandlers.CreateUser)
				users.GET("/:id", userHandlers.GetUser)
				users.PUT("/:id", userHandlers.UpdateUser)
				users.DELETE("/:id", userHandlers.DeleteUser)
				users.POST("/:id/activate", userHandlers.ActivateUser)
				users.POST("/:id/deactivate", userHandlers.DeactivateUser)

				// User permission routes
				users.GET("/:id/permissions", permissionHandlers.GetUserPermissions)
				users.POST("/:id/permissions/assign", permissionHandlers.AssignPermissionToUser)
				users.DELETE("/:id/permissions/:permissionId", permissionHandlers.RemovePermissionFromUser)

				// User role routes
				users.GET("/:id/roles", roleHandlers.GetUserRoles)
				users.POST("/:id/roles/assign", roleHandlers.AssignRoleToUser)
				users.DELETE("/:id/roles/:roleId", roleHandlers.RemoveRoleFromUser)
			}

			// Permission management routes
			permissions := protected.Group("/permissions")
			{
				permissions.GET("", permissionHandlers.GetAllPermissions)
				permissions.GET("/:id", permissionHandlers.GetPermissionByID)
				permissions.POST("", permissionHandlers.CreatePermission)
				permissions.PUT("/:id", permissionHandlers.UpdatePermission)
				permissions.DELETE("/:id", permissionHandlers.DeletePermission)
			}

			// Role management routes
			roles := protected.Group("/roles")
			{
				roles.GET("", roleHandlers.GetAllRoles)
				roles.GET("/:id", roleHandlers.GetRoleByID)
				roles.POST("", roleHandlers.CreateRole)
				roles.PUT("/:id", roleHandlers.UpdateRole)
				roles.DELETE("/:id", roleHandlers.DeleteRole)

				// Role permission routes
				roles.GET("/:id/permissions", roleHandlers.GetRolePermissions)
				roles.POST("/:id/permissions/assign", roleHandlers.AssignPermissionToRole)
				roles.DELETE("/:id/permissions/:permissionId", roleHandlers.RemovePermissionFromRole)
			}

			// Ticket routes
			tickets := protected.Group("/tickets")
			{
				tickets.GET("/", ticketHandlers.ListTickets)
				tickets.GET("/stats", ticketHandlers.GetTicketStats)
				tickets.GET("/my", ticketHandlers.GetMyTickets)
				tickets.POST("/", ticketHandlers.CreateTicket)
				tickets.GET("/:id", ticketHandlers.GetTicket)
				tickets.PUT("/:id", ticketHandlers.UpdateTicket)
				tickets.DELETE("/:id", ticketHandlers.DeleteTicket)
				tickets.POST("/:id/assign", ticketHandlers.AssignTicket)
				// TODO: Implement message routes in next phase
				// tickets.GET("/:id/messages", s.getTicketMessages)
				// tickets.POST("/:id/messages", s.createTicketMessage)
			}

			// Knowledge base routes
			knowledge := protected.Group("/knowledge")
			{
				knowledge.GET("/articles", knowledgeHandlers.ListKnowledgeArticles)
				knowledge.GET("/articles/stats", knowledgeHandlers.GetKnowledgeArticleStats)
				knowledge.GET("/articles/:id", knowledgeHandlers.GetKnowledgeArticle)
				knowledge.POST("/articles", knowledgeHandlers.CreateKnowledgeArticle)
				knowledge.PUT("/articles/:id", knowledgeHandlers.UpdateKnowledgeArticle)
				knowledge.DELETE("/articles/:id", knowledgeHandlers.DeleteKnowledgeArticle)
			}

			// Import/Export routes
			data := protected.Group("/data")
			{
				data.GET("/jobs", importExportHandlers.ListImportExportJobs)
				data.GET("/jobs/stats", importExportHandlers.GetImportExportStats)
				data.GET("/jobs/:id", importExportHandlers.GetImportExportJob)
				data.POST("/jobs/import", importExportHandlers.CreateImportJob)
				data.POST("/jobs/export", importExportHandlers.CreateExportJob)
				data.POST("/jobs/:id/cancel", importExportHandlers.CancelImportExportJob)
				data.DELETE("/jobs/:id", importExportHandlers.DeleteImportExportJob)
				data.GET("/jobs/:id/download", importExportHandlers.DownloadExportFile)
				data.GET("/templates/import", importExportHandlers.GetImportTemplate)
			}
		}
	}

	// Admin routes
	admin := s.router.Group("/api/v1/admin")
	admin.Use(s.authMiddleware())
	admin.Use(s.adminMiddleware())
	{
		admin.GET("/stats", s.getSystemStats)

		// Product management routes (admin only)
		products := admin.Group("/products")
		{
			products.GET("", productHandlers.ListProducts)
			products.POST("", productHandlers.CreateProduct)
			products.GET("/:id", productHandlers.GetProduct)
			products.PUT("/:id", productHandlers.UpdateProduct)
			products.DELETE("/:id", productHandlers.DeleteProduct)
			products.POST("/:id/activate", productHandlers.ActivateProduct)
			products.POST("/:id/deactivate", productHandlers.DeactivateProduct)
		}

		// Service management routes (admin only)
		services := admin.Group("/services")
		{
			services.GET("", serviceHandlers.ListServices)
			services.POST("", serviceHandlers.CreateService)
			services.GET("/:id", serviceHandlers.GetService)
			services.PUT("/:id", serviceHandlers.UpdateService)
			services.DELETE("/:id", serviceHandlers.DeleteService)
			services.POST("/:id/activate", serviceHandlers.ActivateService)
			services.POST("/:id/deactivate", serviceHandlers.DeactivateService)
		}

		// SLA management routes (admin only)
		slaTemplates := admin.Group("/sla-templates")
		{
			slaTemplates.GET("", slaHandlers.ListSLATemplates)
			slaTemplates.POST("", slaHandlers.CreateSLATemplate)
			slaTemplates.GET("/:id", slaHandlers.GetSLATemplate)
			slaTemplates.PUT("/:id", slaHandlers.UpdateSLATemplate)
			slaTemplates.DELETE("/:id", slaHandlers.DeleteSLATemplate)
		}

		slaRules := admin.Group("/sla-rules")
		{
			slaRules.GET("", slaHandlers.ListSLARules)
			slaRules.POST("", slaHandlers.CreateSLARule)
			slaRules.GET("/:id", slaHandlers.GetSLARule)
			slaRules.PUT("/:id", slaHandlers.UpdateSLARule)
			slaRules.DELETE("/:id", slaHandlers.DeleteSLARule)
			slaRules.POST("/:id/activate", slaHandlers.ActivateSLARule)
			slaRules.POST("/:id/deactivate", slaHandlers.DeactivateSLARule)
		}

	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := s.config.GetServerAddress()

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.config.Server.IdleTimeout) * time.Second,
	}

	logger.Info("Starting HTTP server",
		zap.String("address", addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	logger.Info("Shutting down HTTP server...")

	return s.server.Shutdown(ctx)
}

// GetRouter returns the Gin router (useful for testing).
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// GetConfig returns the server configuration.
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// serveSwaggerYAML serves the complete OpenAPI specification.
func (s *Server) serveSwaggerYAML(c *gin.Context) {
	openAPIPath := "./docs/api/complete-openapi.yaml"
	yamlContent, err := os.ReadFile(openAPIPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OpenAPI specification not found",
			"path":  openAPIPath,
		})
		return
	}

	c.Data(http.StatusOK, "application/vnd.oai.openapi", yamlContent)
}

// serveSwaggerUI serves the Swagger UI HTML page.
func (s *Server) serveSwaggerUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>SmartTicket API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/swagger.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// getSystemStats handles system statistics requests.
func (s *Server) getSystemStats(c *gin.Context) {
	// Get database stats
	dbStats := s.db.Stats()

	// Get system info
	sysInfo := map[string]interface{}{
		"go_version":    runtime.Version(),
		"go_os":         runtime.GOOS,
		"go_arch":       runtime.GOARCH,
		"num_cpu":       runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
		"memory_stats": map[string]interface{}{
			"alloc":       runtime.MemStats{}.Alloc,
			"total_alloc": runtime.MemStats{}.TotalAlloc,
			"sys":         runtime.MemStats{}.Sys,
			"num_gc":      runtime.MemStats{}.NumGC,
		},
	}

	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"database":  dbStats,
			"system":    sysInfo,
			"timestamp": time.Now().UTC(),
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetDB returns the database connection.
func (s *Server) GetDB() *database.Database {
	return s.db
}
