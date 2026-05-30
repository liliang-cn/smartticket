// Package mcp exposes SmartTicket's service-layer operations as Model Context
// Protocol (MCP) tools, served over stdio and Streamable HTTP transports.
//
// The package is organized around the Backend interface, which abstracts every
// domain operation the tools can perform. The only implementation today is
// DirectBackend (in-process delegation to the service layer), but the interface
// leaves room for an HTTPBackend in the future.
package mcp

import (
	"context"
	"mime/multipart"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/customer"
	"github.com/company/smartticket/internal/importexport"
	"github.com/company/smartticket/internal/knowledge"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/product"
	servicemgmt "github.com/company/smartticket/internal/service"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/internal/ticket"
	"github.com/company/smartticket/internal/user"
)

// Backend abstracts all SmartTicket domain operations that MCP tools can invoke.
//
// Method signatures mirror the underlying service-layer methods. Where a service
// method requires an "acting user" (for audit/authorship), the Backend method
// carries an explicit userID uint parameter sourced from the authenticated
// Session. Request/response types are reused directly from the service packages;
// MCP tools translate their own MCP-specific input structs into these types.
//
// Methods are grouped by domain.
type Backend interface {
	// --- Ticket domain --- (actor scopes customer isolation)
	CreateTicket(actor authz.Actor, userID uint, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error)
	GetTicket(actor authz.Actor, ticketID uint) (*ticket.TicketResponse, error)
	ListTickets(actor authz.Actor, page, pageSize int, filters map[string]interface{}) (*ticket.TicketListResponse, error)
	UpdateTicket(actor authz.Actor, ticketID, userID uint, req *ticket.UpdateTicketRequest) (*ticket.TicketResponse, error)
	DeleteTicket(actor authz.Actor, ticketID uint) error
	AssignTicket(actor authz.Actor, ticketID, assignedTo uint) error
	GetTicketStats(actor authz.Actor) (map[string]interface{}, error)

	// --- Knowledge domain ---
	CreateKnowledgeArticle(userID uint, req *knowledge.CreateKnowledgeArticleRequest) (*knowledge.KnowledgeArticleResponse, error)
	GetKnowledgeArticle(id uint) (*knowledge.KnowledgeArticleResponse, error)
	ListKnowledgeArticles(page, pageSize int, filters map[string]interface{}) (*knowledge.KnowledgeArticleListResponse, error)
	UpdateKnowledgeArticle(id, userID uint, req *knowledge.UpdateKnowledgeArticleRequest) (*knowledge.KnowledgeArticleResponse, error)
	DeleteKnowledgeArticle(id, userID uint) error
	GetKnowledgeArticleStats() (*knowledge.KnowledgeArticleStatsResponse, error)

	// --- Product domain ---
	CreateProduct(req *product.CreateProductRequest) (*product.ProductResponse, error)
	GetProduct(productID uint) (*product.ProductResponse, error)
	ListProducts(req *product.ListProductsRequest) ([]product.ProductResponse, int64, error)
	UpdateProduct(productID uint, req *product.UpdateProductRequest) (*product.ProductResponse, error)
	DeleteProduct(productID uint) error
	ActivateProduct(productID uint) error
	DeactivateProduct(productID uint) error

	// --- Service domain ---
	CreateService(req *servicemgmt.CreateServiceRequest) (*servicemgmt.ServiceResponse, error)
	GetService(serviceID uint) (*servicemgmt.ServiceResponse, error)
	ListServices(req *servicemgmt.ListServicesRequest) ([]servicemgmt.ServiceResponse, int64, error)
	UpdateService(serviceID uint, req *servicemgmt.UpdateServiceRequest) (*servicemgmt.ServiceResponse, error)
	DeleteService(serviceID uint) error
	ActivateService(serviceID uint) error
	DeactivateService(serviceID uint) error

	// --- SLA domain ---
	CreateSLATemplate(req *sla.CreateSLATemplateRequest) (*sla.SLATemplateResponse, error)
	GetSLATemplate(templateID uint) (*sla.SLATemplateResponse, error)
	ListSLATemplates(req *sla.ListSLATemplatesRequest) ([]sla.SLATemplateResponse, int64, error)
	UpdateSLATemplate(templateID uint, req *sla.UpdateSLATemplateRequest) (*sla.SLATemplateResponse, error)
	DeleteSLATemplate(templateID uint) error
	SetDefaultSLATemplate(templateID uint) error
	ActivateSLATemplate(templateID uint) error
	DeactivateSLATemplate(templateID uint) error
	CreateSLARule(req *sla.CreateSLARuleRequest) (*sla.SLARuleResponse, error)
	GetSLARule(ruleID uint) (*sla.SLARuleResponse, error)
	ListSLARules(req *sla.ListSLARulesRequest) ([]sla.SLARuleResponse, int64, error)
	UpdateSLARule(ruleID uint, req *sla.UpdateSLARuleRequest) (*sla.SLARuleResponse, error)
	DeleteSLARule(ruleID uint) error
	ActivateSLARule(ruleID uint) error
	DeactivateSLARule(ruleID uint) error

	// --- Import/Export domain ---
	CreateImportJob(userID uint, file *multipart.FileHeader, req *importexport.ImportRequest) (*importexport.JobResponse, error)
	CreateExportJob(userID uint, req *importexport.ExportRequest) (*importexport.JobResponse, error)
	GetJob(jobID uint) (*importexport.JobResponse, error)
	ListJobs(page, pageSize int, filters map[string]interface{}) (*importexport.JobListResponse, error)
	CancelJob(jobID, userID uint) error
	DeleteJob(jobID, userID uint) error
	GetJobStats() (map[string]interface{}, error)

	// --- User domain ---
	CreateUser(req *user.CreateUserRequest) (*auth.UserInfo, error)
	GetUser(userID uint) (*auth.UserInfo, error)
	UpdateUser(userID uint, req *user.UpdateUserRequest) (*auth.UserInfo, error)
	DeleteUser(userID uint) error
	ListUsers(req *user.UserListRequest) (*user.UserListResponse, error)
	ActivateUser(userID uint) error
	DeactivateUser(userID uint) error
	ChangeUserPassword(userID uint, newPassword string) error
	GetUserStats() (map[string]interface{}, error)

	// --- RBAC domain ---
	GetUserPermissions(ctx context.Context, userID uint) ([]models.Permission, error)
	GetUserRoles(ctx context.Context, userID uint) ([]models.Role, error)
	GetRolePermissions(ctx context.Context, roleID uint) ([]models.Permission, error)
	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	GetAllRoles(ctx context.Context) ([]models.Role, error)
	GetRoleByID(ctx context.Context, id uint) (*models.Role, error)
	GetPermissionByID(ctx context.Context, id uint) (*models.Permission, error)
	CreateRole(ctx context.Context, role *models.Role) error
	CreatePermission(ctx context.Context, permission *models.Permission) error
	UpdateRole(ctx context.Context, role *models.Role) error
	UpdatePermission(ctx context.Context, permission *models.Permission) error
	DeleteRole(ctx context.Context, id uint) error
	DeletePermission(ctx context.Context, id uint) error
	HasPermission(ctx context.Context, userID uint, permissionCode string) (bool, error)
	AssignRoleToUser(ctx context.Context, userID, roleID uint) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error
	AssignPermissionToUser(ctx context.Context, userID, permissionID uint) error
	RemovePermissionFromUser(ctx context.Context, userID, permissionID uint) error
	AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error

	// --- Customer domain --- (team-only; gated by customer:read/write)
	CreateCustomer(req *customer.CreateCustomerRequest) (*customer.CustomerResponse, error)
	GetCustomer(customerID uint) (*customer.CustomerResponse, error)
	ListCustomers(req *customer.ListCustomersRequest) ([]customer.CustomerResponse, int64, error)
	UpdateCustomer(customerID uint, req *customer.UpdateCustomerRequest) (*customer.CustomerResponse, error)
	DeleteCustomer(customerID uint) error
	ListCustomerUsers(customerID uint) ([]customer.CustomerUserResponse, error)
}

// Session holds the authenticated identity and effective permission set for a
// single MCP connection. It is created once during authentication and injected
// into the request context so that each tool handler can enforce RBAC.
type Session struct {
	UserID      uint
	Role        string
	CustomerID  *uint
	Permissions map[string]bool
}

// Can reports whether the session holds the named permission code.
func (s *Session) Can(code string) bool {
	if s == nil || s.Permissions == nil {
		return false
	}
	return s.Permissions[code]
}

// Actor builds the authorization Actor for this session, used to scope
// customer-isolated queries at the service layer.
func (s *Session) Actor() authz.Actor {
	if s == nil {
		return authz.Actor{}
	}
	return authz.Actor{UserID: s.UserID, Role: s.Role, CustomerID: s.CustomerID}
}

// sessionActor returns the authorization Actor for the session in ctx, or a
// zero Actor when none is present. registerTool runs RequirePermission first,
// so an authenticated session is normally present by the time tools call this.
func sessionActor(ctx context.Context) authz.Actor {
	s, _ := SessionFromContext(ctx)
	return s.Actor()
}

// sessionKey is the unexported context key under which a *Session is stored.
type sessionKey struct{}

// WithSession returns a copy of ctx carrying the given Session.
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// SessionFromContext extracts the Session previously stored with WithSession.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(*Session)
	return s, ok
}
