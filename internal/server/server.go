package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	smartticketweb "github.com/company/smartticket/web"

	"github.com/company/smartticket/internal/api/handlers"
	"github.com/company/smartticket/internal/aiassist"
	"github.com/company/smartticket/internal/api/middleware"
	"github.com/company/smartticket/internal/attachment"
	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/automation"
	"github.com/company/smartticket/internal/branding"
	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/customer"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/email"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/importexport"
	"github.com/company/smartticket/internal/knowledge"
	"github.com/company/smartticket/internal/knowledgebase"
	"github.com/company/smartticket/internal/llm"
	"github.com/company/smartticket/internal/macro"
	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/notification"
	"github.com/company/smartticket/internal/product"
	"github.com/company/smartticket/internal/realtime"
	servicemgmt "github.com/company/smartticket/internal/service"
	"github.com/company/smartticket/internal/services"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/internal/subscription"
	"github.com/company/smartticket/internal/ticket"
	"github.com/company/smartticket/internal/user"
	"github.com/company/smartticket/internal/survey"
	"github.com/company/smartticket/internal/widget"
)

// Server represents the HTTP server.
type Server struct {
	config               *config.Config
	router               *gin.Engine
	server               *http.Server
	db                   *database.Database
	authService          *auth.Service
	permissionMiddleware *middleware.PermissionMiddleware
	kbStore              *knowledgebase.Store
	hub                  *realtime.Hub
	bus                  *automation.Bus
	// cancelCtx stops background goroutines (e.g. the automation scheduler) on Shutdown.
	cancelCtx context.CancelFunc
	// uiFS is the embedded single-page frontend, served for non-API GET routes.
	// nil in API-only builds (built without the `embedui` tag).
	uiFS fs.FS
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

	// Serve the embedded console when the binary was built with the `embedui`
	// tag (single-binary deployment); otherwise the server runs API-only.
	if uiFS, ok := smartticketweb.DistFS(); ok {
		server.uiFS = uiFS
		logger.Info("Embedded web console enabled (serving SPA from binary)")
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

	// Set up custom error handlers. NoRoute also serves the embedded SPA (when
	// present) so client-side routes resolve to the app shell.
	s.router.NoRoute(s.handleNoRoute)
	s.router.NoMethod(errors.MethodNotAllowedHandler)
}

// setupRoutes configures server routes.
func (s *Server) setupRoutes() {
	// Initialize the in-process realtime hub (shared across all WS connections).
	s.hub = realtime.NewHub()
	go s.hub.Run()
	// Initialize the domain-event bus (single shared instance for the server lifetime).
	s.bus = automation.NewBus()

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

	// In-app notification module; injected into the ticket service so ticket
	// events (reply/assign/status) emit notifications without coupling packages.
	notificationService := notification.NewService(s.db.DB)
	notificationHandlers := notification.NewHandlers(notificationService)
	ticketService.SetNotifier(notificationService)
	ticketService.SetBus(s.bus)
	ticketService.SetHub(s.hub)

	// Bidirectional email (opt-in): outbound ticket replies via Resend/SMTP, and
	// inbound email→ticket via a signed webhook (registered as a public route).
	var emailInbound *email.InboundHandler
	var emailSvc *email.Service // kept for CSAT survey delivery
	if s.config.Email.Enabled {
		emailSvc = email.NewService(email.Options{
			Provider:     s.config.Email.Provider,
			FromName:     s.config.Email.FromName,
			FromAddress:  s.config.Email.FromAddress,
			ResendAPIKey: s.config.Email.Resend.APIKey,
			SMTP: email.SMTPOptions{
				Host:     s.config.Email.SMTP.Host,
				Port:     s.config.Email.SMTP.Port,
				Username: s.config.Email.SMTP.Username,
				Password: s.config.Email.SMTP.Password,
				TLS:      s.config.Email.SMTP.TLS,
			},
		})
		ticketService.SetMailer(emailSvc)
		if s.config.Email.Inbound.Enabled {
			emailInbound = email.NewInboundHandler(ticketService, s.config.Email.Inbound.Secret)
		}
		// Fully self-hosted inbound: poll a mailbox over IMAP (no webhook/DNS).
		if s.config.Email.IMAP.Enabled {
			poller := email.NewPoller(email.IMAPOptions{
				Host:         s.config.Email.IMAP.Host,
				Port:         s.config.Email.IMAP.Port,
				Username:     s.config.Email.IMAP.Username,
				Password:     s.config.Email.IMAP.Password,
				Mailbox:      s.config.Email.IMAP.Mailbox,
				TLS:          s.config.Email.IMAP.TLS,
				PollInterval: time.Duration(s.config.Email.IMAP.PollSeconds) * time.Second,
			}, ticketService)
			go poller.Run(context.Background())
		}
	}

	productService := product.NewService(s.db.DB)
	customerService := customer.NewService(s.db.DB)
	serviceManagementService := servicemgmt.NewService(s.db.DB)
	slaService := sla.NewService(s.db.DB)
	subscriptionService := subscription.NewService(s.db.DB)
	importExportService := importexport.NewService(s.db.DB, s.config.Storage.DataPath)
	attachmentService := attachment.NewService(s.db.DB, s.config.Storage.DataPath, s.config.Storage.MaxFileSize, s.config.Storage.AllowedExtensions)
	brandingService := branding.NewService(s.db.DB, s.config.Storage.DataPath)
	macroService := macro.NewService(s.db.DB)
	surveyService := survey.NewService(s.db.DB)

	authHandlers := auth.NewHandlers(s.authService)
	userHandlers := user.NewHandlers(userService)
	ticketHandlers := ticket.NewHandlers(ticketService)
	productHandlers := product.NewHandlers(productService)
	customerHandlers := customer.NewHandlers(customerService)
	serviceHandlers := servicemgmt.NewHandlers(serviceManagementService)
	slaHandlers := sla.NewHandlers(slaService, slaCalculator)
	subscriptionHandlers := subscription.NewHandlers(subscriptionService)
	importExportHandlers := importexport.NewHandlers(importExportService)
	attachmentHandlers := attachment.NewHandlers(attachmentService)
	brandingHandlers := branding.NewHandlers(brandingService)
	macroHandlers := macro.NewHandlers(macroService)
	surveyHandlers := survey.NewHandlers(surveyService)
	permissionHandlers := handlers.NewPermissionHandler(permissionService)
	roleHandlers := handlers.NewRoleHandler(permissionService)

	// AI foundation: LLM providers + CortexDB knowledge store.
	// Encryption key from SMARTTICKET_SECRET_KEY; dev fallback = SHA-256(JWT secret).
	var llmHandlers *llm.Handlers
	var llmServiceRef *llm.Service
	secretKey, kerr := llm.LoadKey(s.config.SecretKeyRaw)
	if kerr != nil {
		logger.Warn("SMARTTICKET_SECRET_KEY not set or invalid; deriving encryption key from JWT secret — changing the JWT secret will make stored LLM API keys unrecoverable")
		sum := sha256.Sum256([]byte(s.config.JWT.Secret))
		secretKey = sum[:]
	}
	cipher, cerr := llm.NewCipher(secretKey)
	if cerr != nil {
		logger.Error("llm cipher init failed; LLM provider endpoints disabled", zap.Error(cerr))
	} else {
		llmService := llm.NewService(s.db.DB, cipher)
		llmServiceRef = llmService
		llmHandlers = llm.NewHandlers(llmService)

		embedder := knowledgebase.NewProviderEmbedder(func(ctx context.Context, texts []string) ([][]float32, error) {
			ep, key, err := llmService.ResolveEmbedding()
			if err != nil {
				return nil, err
			}
			return llm.NewClient(ep.APIEndpoint, key).Embed(ctx, ep.Model, ep.Dimensions, texts)
		}, 1024)
		kbStore, kerr2 := knowledgebase.Open("./data/cortex.db", embedder)
		if kerr2 != nil {
			logger.Warn("cortexdb unavailable", zap.Error(kerr2)) // non-fatal
		} else {
			s.kbStore = kbStore
			llmHandlers.SetCortexProbe(func(ctx context.Context, vec []float32) error {
				if kbStore == nil || !kbStore.Healthy() {
					return fmt.Errorf("cortexdb not open")
				}
				return nil // full embed->store->recall round-trip lands with the ingest API (next slice)
			})
		}
	}

	// Knowledge service/handlers depend on the (optional) AI store + LLM service.
	knowledgeService := knowledge.NewService(s.db.DB, s.kbStore, llmServiceRef)
	knowledgeHandlers := knowledge.NewHandlers(knowledgeService)

	// AI feature toggles (admin-configurable) + the BYO-LLM-backed assistant.
	// Settings always exist; the suggester is wired only when an LLM is available.
	aiSettings := aiassist.NewSettingsStore(s.db.DB)
	aiSettingsHandlers := aiassist.NewSettingsHandlers(aiSettings)
	// autoAssistantRef is set inside the llmServiceRef block so the automation
	// effector can optionally trigger AI actions; nil = AI actions are no-ops.
	var autoAssistantRef *aiassist.Assistant
	if llmServiceRef != nil {
		kb := aiassist.KBSearcherFunc(func(ctx context.Context, q string, k int) []string {
			hits, err := knowledgeService.Search(ctx, q, k, true)
			if err != nil {
				return nil
			}
			out := make([]string, 0, len(hits))
			for _, h := range hits {
				out = append(out, h.Title+": "+h.Snippet)
			}
			return out
		})
		assistant, aerr := aiassist.NewAssistant(aiassist.NewGenerator(llmServiceRef), kb, aiSettings, "./data/agentgo.db")
		if aerr != nil {
			logger.Warn("AI assistant unavailable; suggested replies disabled", zap.Error(aerr))
		} else {
			ticketService.SetSuggester(assistant)
			autoAssistantRef = assistant

			// Wire the auto-resolve orchestrator onto the event bus.
			actions := &ticketAIActions{
				svc:   ticketService,
				notif: notificationService,
				db:    s.db.DB,
			}
			resolver := aiassist.NewAutoResolver(assistant, aiSettings, actions)
			resolver.Subscribe(s.bus)
		}
	}

	// Automation engine: generic trigger→condition→action rules (admin-configurable).
	// The engine subscribes to all ticket domain events and executes matching rules.
	autoSvc := automation.NewService(s.db.DB)
	autoEffector := &automationEffector{
		svc:       ticketService,
		notif:     notificationService,
		assistant: autoAssistantRef,
		db:        s.db.DB,
	}
	autoExec := automation.NewExecutor(autoEffector)
	loadView := func(ticketID uint) (automation.TicketView, error) {
		var tkt models.Ticket
		if err := s.db.DB.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
			return automation.TicketView{}, err
		}
		var tags []string
		if tkt.Tags != "" {
			_ = json.Unmarshal([]byte(tkt.Tags), &tags)
		}
		return automation.TicketView{
			Status:        tkt.Status,
			Priority:      tkt.Priority,
			Severity:      tkt.Severity,
			Channel:       tkt.Channel,
			CustomerEmail: tkt.RequesterEmail,
			Tags:          tags,
		}, nil
	}
	autoEngine := automation.NewEngine(autoSvc, autoExec, loadView)
	autoEngine.Subscribe(s.bus)

	autoScheduler := automation.NewScheduler(s.bus, autoSvc, autoEngine, automation.SchedulerConfig{
		SilentWindowSeconds: 86400,
		// Wire the live settings store so toggling AutoResolveEnabled in /settings
		// takes effect on the next scheduler tick without a server restart.
		Settings: (*aiSettingsAdapter)(aiSettings),
	})
	schedCtx, schedCancel := context.WithCancel(context.Background())
	s.cancelCtx = schedCancel
	go autoScheduler.Run(schedCtx)

	autoHandlers := automation.NewHandlers(autoSvc)

	// CSAT resolve subscriber: on ticket resolved, create a survey and deliver
	// it via the appropriate channel (widget push or email).
	// Defined here (in the server wiring layer) so survey package stays free of
	// ticket/email/hub imports. Failures never block the resolve path.
	{
		hub := s.hub
		baseURL := s.config.App.BaseURL
		db := s.db.DB
		mailer := emailSvc // nil when email is not configured
		s.bus.Subscribe(automation.EventTicketResolved, func(ev automation.Event) {
			sv, err := surveyService.CreateForTicket(ev.TicketID)
			if err != nil {
				logger.Warn("csat: failed to create survey", zap.Uint("ticket_id", ev.TicketID), zap.Error(err))
				return
			}

			// Load the ticket to determine the delivery channel.
			var tkt models.Ticket
			if err := db.Where("id = ?", ev.TicketID).First(&tkt).Error; err != nil {
				logger.Warn("csat: failed to load ticket", zap.Uint("ticket_id", ev.TicketID), zap.Error(err))
				return
			}

			if tkt.Channel == "web_widget" {
				// Notify the embedded widget via WebSocket broadcast.
				payload, _ := json.Marshal(map[string]string{
					"type":  "survey",
					"token": sv.Token,
				})
				hub.Broadcast(fmt.Sprintf("widget:%d", ev.TicketID), payload)
				return
			}

			// Email/web channels: send the survey link by email when available.
			if mailer != nil && tkt.RequesterEmail != "" {
				surveyLink := strings.TrimRight(baseURL, "/") + "/survey/" + sv.Token
				body := "We'd love to hear your feedback on your recent support ticket.\n\n" +
					"Please rate your experience (1–5) at:\n" + surveyLink +
					"\n\nThank you!"
				mailer.SendTicketReply(
					context.Background(),
					tkt.RequesterEmail,
					tkt.TicketNumber,
					"How did we do? — "+tkt.Title,
					tkt.ID,
					body,
					"Support Team",
				)
			}
		})
	}

	// Health check endpoints (no authentication required)
	health := s.router.Group("/")
	{
		health.GET("/health", s.healthCheck)
		health.GET("/healthz", s.healthCheck) // Alternative health endpoint
		health.GET("/api/v1/health", s.healthCheck)

		// API version info (no authentication required)
		health.GET("/version", s.versionInfo)
		health.GET("/api/v1/version", s.versionInfo)

		// Swagger documentation
		health.GET("/swagger/*any", s.serveSwaggerUI)
		health.GET("/swagger.yaml", s.serveSwaggerYAML)
	}

	// Widget service: manages visitor sessions + conversation tokens.
	// Wired here (before route registration) so the widget WS handler can reuse
	// the same ParseToken function without importing the service.
	widgetService := widget.NewService(s.db.DB, ticketService, s.config.JWT.Secret)
	widgetHandlers := widget.NewHandlers(widgetService)

	// widgetCORS is a permissive CORS middleware applied to public widget endpoints
	// and the widget bundle itself. These routes are intended to be embedded on
	// arbitrary customer domains, so Access-Control-Allow-Origin: * is correct and
	// safe — all sensitive operations are scoped by the conversation token, not by
	// the browser's same-origin policy.
	widgetCORS := func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}

	// Serve the compiled widget JS bundle. Path is configurable via the
	// SMARTTICKET_WIDGET_JS_PATH environment variable (via Viper); falls back to
	// the local build output so `go run` during development just works.
	widgetJSPath := s.config.WidgetJSPath
	if widgetJSPath == "" {
		widgetJSPath = "./web-widget/dist/widget.js"
	}
	s.router.GET("/widget.js", widgetCORS, func(c *gin.Context) {
		data, err := os.ReadFile(widgetJSPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "widget bundle not found — run `pnpm build` in web-widget/",
				"path":  widgetJSPath,
			})
			return
		}
		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", data)
	})

	// Demo/smoke-test page: embeds the widget so it can be tested in a browser
	// without deploying to a real customer site.
	s.router.GET("/widget/demo", widgetCORS, func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		origin := scheme + "://" + c.Request.Host
		html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>SmartTicket Widget Demo</title>
  <style>
    body { font-family: system-ui, sans-serif; padding: 40px; background: #f8fafc; }
    h1 { color: #1e293b; font-size: 24px; }
    p  { color: #64748b; margin-top: 8px; line-height: 1.6; }
    code { background: #e2e8f0; padding: 2px 6px; border-radius: 4px; font-size: 13px; }
  </style>
</head>
<body>
  <h1>SmartTicket Widget Demo</h1>
  <p>The chat widget should appear in the bottom-right corner. Click the launcher button to open it.</p>
  <p>This page is served from <code>` + origin + `</code>.</p>
  <script src="` + origin + `/widget.js" data-key="demo" async></script>
</body>
</html>`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	// Public WebSocket endpoint for the customer-facing chat widget.
	// Validates the conversation_token JWT, then subscribes to widget:<ticketID>
	// — the same room the ticket service broadcasts to on web_widget messages.
	hub := s.hub
	s.router.GET("/widget/ws", widgetCORS, func(c *gin.Context) {
		token := c.Query("conversation_token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_token is required"})
			return
		}
		ticketID, err := widget.ParseToken(token, s.config.JWT.Secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired conversation token"})
			return
		}
		room := fmt.Sprintf("widget:%d", ticketID)
		realtime.ServeWSPublic(hub, room, c.Writer, c.Request)
	})

	// Widget REST endpoints (public — no JWT auth required; secured by the
	// conversation token instead).
	widgetGroup := s.router.Group("/widget")
	widgetGroup.Use(widgetCORS)
	{
		widgetGroup.POST("/session", widgetHandlers.StartSession)
		widgetGroup.POST("/messages", widgetHandlers.PostMessage)
		widgetGroup.GET("/messages", widgetHandlers.History)
	}

	// API routes group - public authentication endpoints
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

			// Branding: read is public so the login page and app shell can
			// render the white-label config before authentication.
			public.GET("/settings/branding", brandingHandlers.Get)
			public.GET("/settings/branding/logo", brandingHandlers.ServeLogo)

			// Inbound email webhook (email→ticket). Public but authenticated by
			// a shared secret inside the handler; only mounted when configured.
			if emailInbound != nil {
				public.POST("/email/inbound", emailInbound.Handle)
			}

			// CSAT survey public endpoints — accessed by customers via the
			// survey link; no authentication required.
			public.GET("/survey/:token", surveyHandlers.GetSurvey)
			public.POST("/survey/:token", surveyHandlers.SubmitSurvey)
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
				tickets.GET("/:id/sla", ticketHandlers.GetTicketSLA)
				tickets.GET("/:id/events", ticketHandlers.GetTicketEvents)
				tickets.GET("/:id/messages", ticketHandlers.GetTicketMessages)
				tickets.POST("/:id/messages", ticketHandlers.CreateTicketMessage)
				tickets.POST("/:id/suggest-reply", ticketHandlers.SuggestReply)
				tickets.POST("/:id/attachments", attachmentHandlers.Upload)
				tickets.GET("/:id/attachments", attachmentHandlers.List)
			}

			// WebSocket endpoint for agents/admins to receive real-time ticket updates.
			// Subscribes to room "ticket:<id>". The actor must be able to view the ticket
			// (enforced by attempting GetTicket before upgrading the connection).
			protected.GET("/ws/tickets/:id", func(c *gin.Context) {
				idStr := c.Param("id")
				var ticketID uint
				if _, err := fmt.Sscanf(idStr, "%d", &ticketID); err != nil || ticketID == 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
					return
				}
				// Build actor from auth middleware context values.
				actor := wsActorFromContext(c)
				if _, err := ticketService.GetTicket(actor, ticketID); err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found or access denied"})
					return
				}
				room := fmt.Sprintf("ticket:%d", ticketID)
				realtime.ServeWS(hub, room, c.Writer, c.Request)
			})

			// Attachment download (by attachment id, customer-isolated).
			protected.GET("/attachments/:id/download", attachmentHandlers.Download)

			// In-app notification routes (per authenticated user).
			notif := protected.Group("/notifications")
			{
				notif.GET("", notificationHandlers.List)
				notif.GET("/unread-count", notificationHandlers.UnreadCount)
				notif.POST("/:id/read", notificationHandlers.MarkRead)
				notif.POST("/read-all", notificationHandlers.MarkAllRead)
			}

			// Customer organization management routes (team-only).
			customers := protected.Group("/customers")
			customers.Use(s.adminMiddleware())
			{
				customers.POST("", customerHandlers.CreateCustomer)
				customers.GET("", customerHandlers.ListCustomers)
				customers.GET("/:id", customerHandlers.GetCustomer)
				customers.PUT("/:id", customerHandlers.UpdateCustomer)
				customers.DELETE("/:id", customerHandlers.DeleteCustomer)
				customers.GET("/:id/users", customerHandlers.ListCustomerUsers)
			}

			// CSAT survey stats (any authenticated team/agent user).
			protected.GET("/survey/stats", surveyHandlers.GetStats)

			// Branding / white-label settings (admin-only writes; reads are
			// public, registered above).
			// AI settings: read for any authenticated user (the agent UI needs to
			// know which AI features are on); writes are admin-only below.
			protected.GET("/settings/ai", aiSettingsHandlers.Get)

			settings := protected.Group("/settings")
			settings.Use(s.adminMiddleware())
			{
				settings.PUT("/ai", aiSettingsHandlers.Update)
				settings.PUT("/branding", brandingHandlers.Update)
				settings.POST("/branding/logo", brandingHandlers.UploadLogo)
				settings.DELETE("/branding/logo", brandingHandlers.DeleteLogo)
			}

			// LLM provider management routes (admin-only).
			if llmHandlers != nil {
				llmGroup := protected.Group("/llm/providers")
				llmGroup.Use(s.adminMiddleware())
				{
					llmGroup.GET("", llmHandlers.List)
					llmGroup.POST("", llmHandlers.Create)
					llmGroup.GET("/:id", llmHandlers.Get)
					llmGroup.PUT("/:id", llmHandlers.Update)
					llmGroup.DELETE("/:id", llmHandlers.Delete)
					llmGroup.POST("/:id/test", llmHandlers.Test)
				}
			}

			// Automation rules (admin-only CRUD + reorder).
			autoGroup := protected.Group("/automations")
			autoGroup.Use(s.adminMiddleware())
			{
				autoGroup.GET("", autoHandlers.ListRules)
				autoGroup.POST("", autoHandlers.CreateRule)
				autoGroup.POST("/reorder", autoHandlers.ReorderRules)
				autoGroup.GET("/:id", autoHandlers.GetRule)
				autoGroup.PUT("/:id", autoHandlers.UpdateRule)
				autoGroup.DELETE("/:id", autoHandlers.DeleteRule)
			}

			// Macro / canned responses (any authenticated team or admin user).
			macros := protected.Group("/macros")
			{
				macros.GET("", macroHandlers.List)
				macros.POST("", macroHandlers.Create)
				macros.GET("/:id", macroHandlers.Get)
				macros.PUT("/:id", macroHandlers.Update)
				macros.DELETE("/:id", macroHandlers.Delete)
				macros.POST("/:id/apply", func(c *gin.Context) {
					// Assemble RenderContext from ticket + user before delegating to
					// the macro handler so macro package doesn't import ticket/user.
					idStr := c.Query("ticket_id")
					if idStr != "" {
						if tid, err := strconv.ParseUint(idStr, 10, 64); err == nil && tid > 0 {
							// Build a minimal actor for ticket lookup (team user, no customer scope).
							actor := authz.Actor{
								UserID: c.GetUint("user_id"),
								Role:   c.GetString("user_role"),
							}
							if tkt, err := ticketService.GetTicket(actor, uint(tid)); err == nil {
								// Resolve the acting user's display name.
								agentName := ""
								if ui, uerr := userService.GetUser(actor.UserID); uerr == nil {
									agentName = ui.FirstName + " " + ui.LastName
								}
								customerName := tkt.RequesterName
								if tkt.CustomerName != "" {
									customerName = tkt.CustomerName
								}
								rctx := macro.RenderContext{
									CustomerName:  customerName,
									AgentName:     agentName,
									TicketID:      fmt.Sprintf("%d", tkt.ID),
									TicketSubject: tkt.Title,
								}
								c.Set("macro_render_ctx", rctx)
							}
						}
					}
					macroHandlers.Apply(c)
				})
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

				// AI: semantic search + RAG ask (any authenticated role).
				knowledge.POST("/search", knowledgeHandlers.SearchKnowledge)
				knowledge.POST("/ask", knowledgeHandlers.AskKnowledge)

				// AI: full re-index (admin only).
				knowledgeAdmin := knowledge.Group("")
				knowledgeAdmin.Use(s.adminMiddleware())
				knowledgeAdmin.POST("/reindex", knowledgeHandlers.ReindexKnowledge)
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

		// Subscription / licensing routes (admin only)
		subscriptions := admin.Group("/subscriptions")
		{
			subscriptions.GET("", subscriptionHandlers.ListSubscriptions)
			subscriptions.POST("", subscriptionHandlers.CreateSubscription)
			subscriptions.GET("/:id", subscriptionHandlers.GetSubscription)
			subscriptions.PUT("/:id", subscriptionHandlers.UpdateSubscription)
			subscriptions.DELETE("/:id", subscriptionHandlers.DeleteSubscription)
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

	// Cancel the background context so the scheduler goroutine exits cleanly.
	if s.cancelCtx != nil {
		s.cancelCtx()
	}

	if s.kbStore != nil {
		if err := s.kbStore.Close(); err != nil {
			logger.Warn("failed to close cortexdb store", zap.Error(err))
		}
	}

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

// aiSettingsAdapter wraps *aiassist.SettingsStore and satisfies
// automation.AutoResolveSettingsReader. The type alias avoids exposing the
// aiassist import in the public scheduler API while keeping the dependency
// explicit at the server wiring layer.
type aiSettingsAdapter aiassist.SettingsStore

// AutoResolveEnabled implements automation.AutoResolveSettingsReader.
// Returns false on any DB error so the scheduler degrades safely.
func (a *aiSettingsAdapter) AutoResolveEnabled() bool {
	s, err := (*aiassist.SettingsStore)(a).Get()
	if err != nil || s == nil {
		return false
	}
	return s.AutoResolveEnabled
}

// handleNoRoute serves the embedded single-page app for unmatched non-API GET
// routes (so client-side routing works on hard refresh / deep links), and
// returns the standard JSON 404 for API and infrastructure paths. In API-only
// builds (no embedded UI) every unmatched route gets the JSON 404.
func (s *Server) handleNoRoute(c *gin.Context) {
	p := c.Request.URL.Path
	if s.uiFS == nil || c.Request.Method != http.MethodGet ||
		strings.HasPrefix(p, "/api/") ||
		strings.HasPrefix(p, "/swagger") ||
		p == "/metrics" || p == "/health" || p == "/healthz" || p == "/version" {
		errors.NotFoundHandler(c)
		return
	}

	rel := strings.TrimPrefix(path.Clean(p), "/")
	if rel == "" {
		rel = "index.html"
	}
	data, err := fs.ReadFile(s.uiFS, rel)
	if err != nil {
		// Unknown asset → serve the SPA shell so the client router can handle it.
		rel = "index.html"
		data, err = fs.ReadFile(s.uiFS, rel)
		if err != nil {
			errors.NotFoundHandler(c)
			return
		}
	}

	ctype := mime.TypeByExtension(path.Ext(rel))
	if ctype == "" {
		ctype = http.DetectContentType(data)
	}
	// Vite emits content-hashed asset filenames, so they can cache forever.
	if strings.HasPrefix(rel, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	}
	c.Data(http.StatusOK, ctype, data)
}

// serveSwaggerYAML serves the authoritative (swag-generated) OpenAPI spec.
func (s *Server) serveSwaggerYAML(c *gin.Context) {
	openAPIPath := "./docs/swagger.yaml"
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
