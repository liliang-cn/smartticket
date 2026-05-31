package mcp

import (
	"context"
	"mime/multipart"

	"gorm.io/gorm"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/customer"
	"github.com/company/smartticket/internal/importexport"
	"github.com/company/smartticket/internal/knowledge"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/notification"
	"github.com/company/smartticket/internal/product"
	servicemgmt "github.com/company/smartticket/internal/service"
	"github.com/company/smartticket/internal/services"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/internal/ticket"
	"github.com/company/smartticket/internal/user"
)

// DirectBackend implements Backend by delegating directly to in-process service
// instances. This is the default backend for self-hosted single-binary
// deployment.
type DirectBackend struct {
	ticket       *ticket.Service
	knowledge    *knowledge.Service
	product      *product.Service
	service      *servicemgmt.Service
	sla          *sla.Service
	importexport *importexport.Service
	user         *user.Service
	customer     *customer.Service
	permission   *services.PermissionService
}

// NewDirectBackend constructs a DirectBackend, wiring up each domain service the
// same way internal/server does. It needs the *gorm.DB plus the shared
// auth.Service so the user service can reuse it for token/user info.
func NewDirectBackend(db *gorm.DB, authService *auth.Service, permissionService *services.PermissionService) *DirectBackend {
	slaCalculator := sla.NewCalculator(db)
	authRepo := auth.NewRepository(db)

	// Wire in-app notifications so ticket actions performed via MCP (e.g. an
	// agent posting a reply) emit notifications just like the REST path.
	ticketSvc := ticket.NewService(db, slaCalculator)
	ticketSvc.SetNotifier(notification.NewService(db))

	return &DirectBackend{
		ticket:       ticketSvc,
		knowledge:    knowledge.NewService(db, nil, nil),
		product:      product.NewService(db),
		service:      servicemgmt.NewService(db),
		sla:          sla.NewService(db),
		importexport: importexport.NewService(db),
		user:         user.NewService(db, authRepo, authService),
		customer:     customer.NewService(db),
		permission:   permissionService,
	}
}

// --- Ticket domain ---

func (b *DirectBackend) CreateTicket(actor authz.Actor, userID uint, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error) {
	return b.ticket.CreateTicket(actor, userID, req)
}

func (b *DirectBackend) GetTicket(actor authz.Actor, ticketID uint) (*ticket.TicketResponse, error) {
	return b.ticket.GetTicket(actor, ticketID)
}

func (b *DirectBackend) ListTickets(actor authz.Actor, page, pageSize int, filters map[string]interface{}) (*ticket.TicketListResponse, error) {
	return b.ticket.ListTickets(actor, page, pageSize, filters)
}

func (b *DirectBackend) UpdateTicket(actor authz.Actor, ticketID, userID uint, req *ticket.UpdateTicketRequest) (*ticket.TicketResponse, error) {
	return b.ticket.UpdateTicket(actor, ticketID, userID, req)
}

func (b *DirectBackend) DeleteTicket(actor authz.Actor, ticketID uint) error {
	return b.ticket.DeleteTicket(actor, ticketID)
}

func (b *DirectBackend) AssignTicket(actor authz.Actor, ticketID, assignedTo uint) error {
	return b.ticket.AssignTicket(actor, ticketID, assignedTo)
}

func (b *DirectBackend) GetTicketStats(actor authz.Actor) (map[string]interface{}, error) {
	return b.ticket.GetTicketStats(actor)
}

func (b *DirectBackend) CreateMessage(actor authz.Actor, ticketID, userID uint, req *ticket.CreateMessageRequest) (*ticket.MessageResponse, error) {
	return b.ticket.CreateMessage(actor, ticketID, userID, req)
}

func (b *DirectBackend) ListMessages(actor authz.Actor, ticketID uint) ([]ticket.MessageResponse, error) {
	return b.ticket.ListMessages(actor, ticketID)
}

// --- Knowledge domain ---

func (b *DirectBackend) CreateKnowledgeArticle(userID uint, req *knowledge.CreateKnowledgeArticleRequest) (*knowledge.KnowledgeArticleResponse, error) {
	return b.knowledge.CreateKnowledgeArticle(userID, req)
}

func (b *DirectBackend) GetKnowledgeArticle(id uint) (*knowledge.KnowledgeArticleResponse, error) {
	return b.knowledge.GetKnowledgeArticle(id)
}

func (b *DirectBackend) ListKnowledgeArticles(page, pageSize int, filters map[string]interface{}) (*knowledge.KnowledgeArticleListResponse, error) {
	return b.knowledge.ListKnowledgeArticles(page, pageSize, filters)
}

func (b *DirectBackend) UpdateKnowledgeArticle(id, userID uint, req *knowledge.UpdateKnowledgeArticleRequest) (*knowledge.KnowledgeArticleResponse, error) {
	return b.knowledge.UpdateKnowledgeArticle(id, userID, req)
}

func (b *DirectBackend) DeleteKnowledgeArticle(id, userID uint) error {
	return b.knowledge.DeleteKnowledgeArticle(id, userID)
}

func (b *DirectBackend) GetKnowledgeArticleStats() (*knowledge.KnowledgeArticleStatsResponse, error) {
	return b.knowledge.GetKnowledgeArticleStats()
}

// --- Product domain ---

func (b *DirectBackend) CreateProduct(req *product.CreateProductRequest) (*product.ProductResponse, error) {
	return b.product.CreateProduct(req)
}

func (b *DirectBackend) GetProduct(productID uint) (*product.ProductResponse, error) {
	return b.product.GetProduct(productID)
}

func (b *DirectBackend) ListProducts(req *product.ListProductsRequest) ([]product.ProductResponse, int64, error) {
	return b.product.ListProducts(req)
}

func (b *DirectBackend) UpdateProduct(productID uint, req *product.UpdateProductRequest) (*product.ProductResponse, error) {
	return b.product.UpdateProduct(productID, req)
}

func (b *DirectBackend) DeleteProduct(productID uint) error {
	return b.product.DeleteProduct(productID)
}

func (b *DirectBackend) ActivateProduct(productID uint) error {
	return b.product.ActivateProduct(productID)
}

func (b *DirectBackend) DeactivateProduct(productID uint) error {
	return b.product.DeactivateProduct(productID)
}

// --- Service domain ---

func (b *DirectBackend) CreateService(req *servicemgmt.CreateServiceRequest) (*servicemgmt.ServiceResponse, error) {
	return b.service.CreateService(req)
}

func (b *DirectBackend) GetService(serviceID uint) (*servicemgmt.ServiceResponse, error) {
	return b.service.GetService(serviceID)
}

func (b *DirectBackend) ListServices(req *servicemgmt.ListServicesRequest) ([]servicemgmt.ServiceResponse, int64, error) {
	return b.service.ListServices(req)
}

func (b *DirectBackend) UpdateService(serviceID uint, req *servicemgmt.UpdateServiceRequest) (*servicemgmt.ServiceResponse, error) {
	return b.service.UpdateService(serviceID, req)
}

func (b *DirectBackend) DeleteService(serviceID uint) error {
	return b.service.DeleteService(serviceID)
}

func (b *DirectBackend) ActivateService(serviceID uint) error {
	return b.service.ActivateService(serviceID)
}

func (b *DirectBackend) DeactivateService(serviceID uint) error {
	return b.service.DeactivateService(serviceID)
}

// --- SLA domain ---

func (b *DirectBackend) CreateSLATemplate(req *sla.CreateSLATemplateRequest) (*sla.SLATemplateResponse, error) {
	return b.sla.CreateSLATemplate(req)
}

func (b *DirectBackend) GetSLATemplate(templateID uint) (*sla.SLATemplateResponse, error) {
	return b.sla.GetSLATemplate(templateID)
}

func (b *DirectBackend) ListSLATemplates(req *sla.ListSLATemplatesRequest) ([]sla.SLATemplateResponse, int64, error) {
	return b.sla.ListSLATemplates(req)
}

func (b *DirectBackend) UpdateSLATemplate(templateID uint, req *sla.UpdateSLATemplateRequest) (*sla.SLATemplateResponse, error) {
	return b.sla.UpdateSLATemplate(templateID, req)
}

func (b *DirectBackend) DeleteSLATemplate(templateID uint) error {
	return b.sla.DeleteSLATemplate(templateID)
}

func (b *DirectBackend) SetDefaultSLATemplate(templateID uint) error {
	return b.sla.SetDefaultSLATemplate(templateID)
}

func (b *DirectBackend) ActivateSLATemplate(templateID uint) error {
	return b.sla.ActivateSLATemplate(templateID)
}

func (b *DirectBackend) DeactivateSLATemplate(templateID uint) error {
	return b.sla.DeactivateSLATemplate(templateID)
}

func (b *DirectBackend) CreateSLARule(req *sla.CreateSLARuleRequest) (*sla.SLARuleResponse, error) {
	return b.sla.CreateSLARule(req)
}

func (b *DirectBackend) GetSLARule(ruleID uint) (*sla.SLARuleResponse, error) {
	return b.sla.GetSLARule(ruleID)
}

func (b *DirectBackend) ListSLARules(req *sla.ListSLARulesRequest) ([]sla.SLARuleResponse, int64, error) {
	return b.sla.ListSLARules(req)
}

func (b *DirectBackend) UpdateSLARule(ruleID uint, req *sla.UpdateSLARuleRequest) (*sla.SLARuleResponse, error) {
	return b.sla.UpdateSLARule(ruleID, req)
}

func (b *DirectBackend) DeleteSLARule(ruleID uint) error {
	return b.sla.DeleteSLARule(ruleID)
}

func (b *DirectBackend) ActivateSLARule(ruleID uint) error {
	return b.sla.ActivateSLARule(ruleID)
}

func (b *DirectBackend) DeactivateSLARule(ruleID uint) error {
	return b.sla.DeactivateSLARule(ruleID)
}

// --- Import/Export domain ---

func (b *DirectBackend) CreateImportJob(userID uint, file *multipart.FileHeader, req *importexport.ImportRequest) (*importexport.JobResponse, error) {
	return b.importexport.CreateImportJob(userID, file, req)
}

func (b *DirectBackend) CreateExportJob(userID uint, req *importexport.ExportRequest) (*importexport.JobResponse, error) {
	return b.importexport.CreateExportJob(userID, req)
}

func (b *DirectBackend) GetJob(jobID uint) (*importexport.JobResponse, error) {
	return b.importexport.GetJob(jobID)
}

func (b *DirectBackend) ListJobs(page, pageSize int, filters map[string]interface{}) (*importexport.JobListResponse, error) {
	return b.importexport.ListJobs(page, pageSize, filters)
}

func (b *DirectBackend) CancelJob(jobID, userID uint) error {
	return b.importexport.CancelJob(jobID, userID)
}

func (b *DirectBackend) DeleteJob(jobID, userID uint) error {
	return b.importexport.DeleteJob(jobID, userID)
}

func (b *DirectBackend) GetJobStats() (map[string]interface{}, error) {
	return b.importexport.GetJobStats()
}

// --- User domain ---

func (b *DirectBackend) CreateUser(req *user.CreateUserRequest) (*auth.UserInfo, error) {
	return b.user.CreateUser(req)
}

func (b *DirectBackend) GetUser(userID uint) (*auth.UserInfo, error) {
	return b.user.GetUser(userID)
}

func (b *DirectBackend) UpdateUser(userID uint, req *user.UpdateUserRequest) (*auth.UserInfo, error) {
	return b.user.UpdateUser(userID, req)
}

func (b *DirectBackend) DeleteUser(userID uint) error {
	return b.user.DeleteUser(userID)
}

func (b *DirectBackend) ListUsers(req *user.UserListRequest) (*user.UserListResponse, error) {
	return b.user.ListUsers(req)
}

func (b *DirectBackend) ActivateUser(userID uint) error {
	return b.user.ActivateUser(userID)
}

func (b *DirectBackend) DeactivateUser(userID uint) error {
	return b.user.DeactivateUser(userID)
}

func (b *DirectBackend) ChangeUserPassword(userID uint, newPassword string) error {
	return b.user.ChangeUserPassword(userID, newPassword)
}

func (b *DirectBackend) GetUserStats() (map[string]interface{}, error) {
	return b.user.GetUserStats()
}

// --- RBAC domain ---

func (b *DirectBackend) GetUserPermissions(ctx context.Context, userID uint) ([]models.Permission, error) {
	return b.permission.GetUserPermissions(ctx, userID)
}

func (b *DirectBackend) GetUserRoles(ctx context.Context, userID uint) ([]models.Role, error) {
	return b.permission.GetUserRoles(ctx, userID)
}

func (b *DirectBackend) GetRolePermissions(ctx context.Context, roleID uint) ([]models.Permission, error) {
	return b.permission.GetRolePermissions(ctx, roleID)
}

func (b *DirectBackend) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	return b.permission.GetAllPermissions(ctx)
}

func (b *DirectBackend) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	return b.permission.GetAllRoles(ctx)
}

func (b *DirectBackend) GetRoleByID(ctx context.Context, id uint) (*models.Role, error) {
	return b.permission.GetRoleByID(ctx, id)
}

func (b *DirectBackend) GetPermissionByID(ctx context.Context, id uint) (*models.Permission, error) {
	return b.permission.GetPermissionByID(ctx, id)
}

func (b *DirectBackend) CreateRole(ctx context.Context, role *models.Role) error {
	return b.permission.CreateRole(ctx, role)
}

func (b *DirectBackend) CreatePermission(ctx context.Context, permission *models.Permission) error {
	return b.permission.CreatePermission(ctx, permission)
}

func (b *DirectBackend) UpdateRole(ctx context.Context, role *models.Role) error {
	return b.permission.UpdateRole(ctx, role)
}

func (b *DirectBackend) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	return b.permission.UpdatePermission(ctx, permission)
}

func (b *DirectBackend) DeleteRole(ctx context.Context, id uint) error {
	return b.permission.DeleteRole(ctx, id)
}

func (b *DirectBackend) DeletePermission(ctx context.Context, id uint) error {
	return b.permission.DeletePermission(ctx, id)
}

func (b *DirectBackend) HasPermission(ctx context.Context, userID uint, permissionCode string) (bool, error) {
	return b.permission.HasPermission(ctx, userID, permissionCode)
}

func (b *DirectBackend) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	return b.permission.AssignRoleToUser(ctx, userID, roleID)
}

func (b *DirectBackend) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	return b.permission.RemoveRoleFromUser(ctx, userID, roleID)
}

func (b *DirectBackend) AssignPermissionToUser(ctx context.Context, userID, permissionID uint) error {
	return b.permission.AssignPermissionToUser(ctx, userID, permissionID)
}

func (b *DirectBackend) RemovePermissionFromUser(ctx context.Context, userID, permissionID uint) error {
	return b.permission.RemovePermissionFromUser(ctx, userID, permissionID)
}

func (b *DirectBackend) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	return b.permission.AssignPermissionToRole(ctx, roleID, permissionID)
}

func (b *DirectBackend) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	return b.permission.RemovePermissionFromRole(ctx, roleID, permissionID)
}

// --- Customer domain ---

func (b *DirectBackend) CreateCustomer(req *customer.CreateCustomerRequest) (*customer.CustomerResponse, error) {
	return b.customer.CreateCustomer(req)
}

func (b *DirectBackend) GetCustomer(customerID uint) (*customer.CustomerResponse, error) {
	return b.customer.GetCustomer(customerID)
}

func (b *DirectBackend) ListCustomers(req *customer.ListCustomersRequest) ([]customer.CustomerResponse, int64, error) {
	return b.customer.ListCustomers(req)
}

func (b *DirectBackend) UpdateCustomer(customerID uint, req *customer.UpdateCustomerRequest) (*customer.CustomerResponse, error) {
	return b.customer.UpdateCustomer(customerID, req)
}

func (b *DirectBackend) DeleteCustomer(customerID uint) error {
	return b.customer.DeleteCustomer(customerID)
}

func (b *DirectBackend) ListCustomerUsers(customerID uint) ([]customer.CustomerUserResponse, error) {
	return b.customer.ListCustomerUsers(customerID)
}

// Ensure DirectBackend satisfies the Backend interface.
var _ Backend = (*DirectBackend)(nil)
