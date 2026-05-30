package mcp

import (
	"context"
	"mime/multipart"

	"github.com/stretchr/testify/mock"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/importexport"
	"github.com/company/smartticket/internal/knowledge"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/product"
	servicemgmt "github.com/company/smartticket/internal/service"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/internal/ticket"
	"github.com/company/smartticket/internal/user"
)

// MockBackend is a testify-based mock implementation of Backend, shared by all
// domain tool tests. Each domain test file sets up expectations with
// (*MockBackend).On("MethodName", args...).Return(...) and asserts via
// AssertExpectations. Methods not exercised by a given test simply have no
// expectation registered.
type MockBackend struct {
	mock.Mock
}

// Ensure MockBackend satisfies the Backend interface.
var _ Backend = (*MockBackend)(nil)

// --- Ticket domain ---

func (m *MockBackend) CreateTicket(userID uint, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error) {
	args := m.Called(userID, req)
	return getPtr[ticket.TicketResponse](args, 0), args.Error(1)
}

func (m *MockBackend) GetTicket(ticketID uint) (*ticket.TicketResponse, error) {
	args := m.Called(ticketID)
	return getPtr[ticket.TicketResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ListTickets(page, pageSize int, filters map[string]interface{}) (*ticket.TicketListResponse, error) {
	args := m.Called(page, pageSize, filters)
	return getPtr[ticket.TicketListResponse](args, 0), args.Error(1)
}

func (m *MockBackend) UpdateTicket(ticketID, userID uint, req *ticket.UpdateTicketRequest) (*ticket.TicketResponse, error) {
	args := m.Called(ticketID, userID, req)
	return getPtr[ticket.TicketResponse](args, 0), args.Error(1)
}

func (m *MockBackend) DeleteTicket(ticketID uint) error {
	return m.Called(ticketID).Error(0)
}

func (m *MockBackend) AssignTicket(ticketID, assignedTo uint) error {
	return m.Called(ticketID, assignedTo).Error(0)
}

func (m *MockBackend) GetTicketStats() (map[string]interface{}, error) {
	args := m.Called()
	return getMap(args, 0), args.Error(1)
}

// --- Knowledge domain ---

func (m *MockBackend) CreateKnowledgeArticle(userID uint, req *knowledge.CreateKnowledgeArticleRequest) (*knowledge.KnowledgeArticleResponse, error) {
	args := m.Called(userID, req)
	return getPtr[knowledge.KnowledgeArticleResponse](args, 0), args.Error(1)
}

func (m *MockBackend) GetKnowledgeArticle(id uint) (*knowledge.KnowledgeArticleResponse, error) {
	args := m.Called(id)
	return getPtr[knowledge.KnowledgeArticleResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ListKnowledgeArticles(page, pageSize int, filters map[string]interface{}) (*knowledge.KnowledgeArticleListResponse, error) {
	args := m.Called(page, pageSize, filters)
	return getPtr[knowledge.KnowledgeArticleListResponse](args, 0), args.Error(1)
}

func (m *MockBackend) UpdateKnowledgeArticle(id, userID uint, req *knowledge.UpdateKnowledgeArticleRequest) (*knowledge.KnowledgeArticleResponse, error) {
	args := m.Called(id, userID, req)
	return getPtr[knowledge.KnowledgeArticleResponse](args, 0), args.Error(1)
}

func (m *MockBackend) DeleteKnowledgeArticle(id, userID uint) error {
	return m.Called(id, userID).Error(0)
}

func (m *MockBackend) GetKnowledgeArticleStats() (*knowledge.KnowledgeArticleStatsResponse, error) {
	args := m.Called()
	return getPtr[knowledge.KnowledgeArticleStatsResponse](args, 0), args.Error(1)
}

// --- Product domain ---

func (m *MockBackend) CreateProduct(req *product.CreateProductRequest) (*product.ProductResponse, error) {
	args := m.Called(req)
	return getPtr[product.ProductResponse](args, 0), args.Error(1)
}

func (m *MockBackend) GetProduct(productID uint) (*product.ProductResponse, error) {
	args := m.Called(productID)
	return getPtr[product.ProductResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ListProducts(req *product.ListProductsRequest) ([]product.ProductResponse, int64, error) {
	args := m.Called(req)
	var list []product.ProductResponse
	if v := args.Get(0); v != nil {
		list = v.([]product.ProductResponse)
	}
	return list, int64(args.Int(1)), args.Error(2)
}

func (m *MockBackend) UpdateProduct(productID uint, req *product.UpdateProductRequest) (*product.ProductResponse, error) {
	args := m.Called(productID, req)
	return getPtr[product.ProductResponse](args, 0), args.Error(1)
}

func (m *MockBackend) DeleteProduct(productID uint) error {
	return m.Called(productID).Error(0)
}

func (m *MockBackend) ActivateProduct(productID uint) error {
	return m.Called(productID).Error(0)
}

func (m *MockBackend) DeactivateProduct(productID uint) error {
	return m.Called(productID).Error(0)
}

// --- Service domain ---

func (m *MockBackend) CreateService(req *servicemgmt.CreateServiceRequest) (*servicemgmt.ServiceResponse, error) {
	args := m.Called(req)
	return getPtr[servicemgmt.ServiceResponse](args, 0), args.Error(1)
}

func (m *MockBackend) GetService(serviceID uint) (*servicemgmt.ServiceResponse, error) {
	args := m.Called(serviceID)
	return getPtr[servicemgmt.ServiceResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ListServices(req *servicemgmt.ListServicesRequest) ([]servicemgmt.ServiceResponse, int64, error) {
	args := m.Called(req)
	var list []servicemgmt.ServiceResponse
	if v := args.Get(0); v != nil {
		list = v.([]servicemgmt.ServiceResponse)
	}
	return list, int64(args.Int(1)), args.Error(2)
}

func (m *MockBackend) UpdateService(serviceID uint, req *servicemgmt.UpdateServiceRequest) (*servicemgmt.ServiceResponse, error) {
	args := m.Called(serviceID, req)
	return getPtr[servicemgmt.ServiceResponse](args, 0), args.Error(1)
}

func (m *MockBackend) DeleteService(serviceID uint) error {
	return m.Called(serviceID).Error(0)
}

func (m *MockBackend) ActivateService(serviceID uint) error {
	return m.Called(serviceID).Error(0)
}

func (m *MockBackend) DeactivateService(serviceID uint) error {
	return m.Called(serviceID).Error(0)
}

// --- SLA domain ---

func (m *MockBackend) CreateSLATemplate(req *sla.CreateSLATemplateRequest) (*sla.SLATemplateResponse, error) {
	args := m.Called(req)
	return getPtr[sla.SLATemplateResponse](args, 0), args.Error(1)
}

func (m *MockBackend) GetSLATemplate(templateID uint) (*sla.SLATemplateResponse, error) {
	args := m.Called(templateID)
	return getPtr[sla.SLATemplateResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ListSLATemplates(req *sla.ListSLATemplatesRequest) ([]sla.SLATemplateResponse, int64, error) {
	args := m.Called(req)
	var list []sla.SLATemplateResponse
	if v := args.Get(0); v != nil {
		list = v.([]sla.SLATemplateResponse)
	}
	return list, int64(args.Int(1)), args.Error(2)
}

func (m *MockBackend) UpdateSLATemplate(templateID uint, req *sla.UpdateSLATemplateRequest) (*sla.SLATemplateResponse, error) {
	args := m.Called(templateID, req)
	return getPtr[sla.SLATemplateResponse](args, 0), args.Error(1)
}

func (m *MockBackend) DeleteSLATemplate(templateID uint) error {
	return m.Called(templateID).Error(0)
}

func (m *MockBackend) SetDefaultSLATemplate(templateID uint) error {
	return m.Called(templateID).Error(0)
}

func (m *MockBackend) ActivateSLATemplate(templateID uint) error {
	return m.Called(templateID).Error(0)
}

func (m *MockBackend) DeactivateSLATemplate(templateID uint) error {
	return m.Called(templateID).Error(0)
}

func (m *MockBackend) CreateSLARule(req *sla.CreateSLARuleRequest) (*sla.SLARuleResponse, error) {
	args := m.Called(req)
	return getPtr[sla.SLARuleResponse](args, 0), args.Error(1)
}

func (m *MockBackend) GetSLARule(ruleID uint) (*sla.SLARuleResponse, error) {
	args := m.Called(ruleID)
	return getPtr[sla.SLARuleResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ListSLARules(req *sla.ListSLARulesRequest) ([]sla.SLARuleResponse, int64, error) {
	args := m.Called(req)
	var list []sla.SLARuleResponse
	if v := args.Get(0); v != nil {
		list = v.([]sla.SLARuleResponse)
	}
	return list, int64(args.Int(1)), args.Error(2)
}

func (m *MockBackend) UpdateSLARule(ruleID uint, req *sla.UpdateSLARuleRequest) (*sla.SLARuleResponse, error) {
	args := m.Called(ruleID, req)
	return getPtr[sla.SLARuleResponse](args, 0), args.Error(1)
}

func (m *MockBackend) DeleteSLARule(ruleID uint) error {
	return m.Called(ruleID).Error(0)
}

func (m *MockBackend) ActivateSLARule(ruleID uint) error {
	return m.Called(ruleID).Error(0)
}

func (m *MockBackend) DeactivateSLARule(ruleID uint) error {
	return m.Called(ruleID).Error(0)
}

// --- Import/Export domain ---

func (m *MockBackend) CreateImportJob(userID uint, file *multipart.FileHeader, req *importexport.ImportRequest) (*importexport.JobResponse, error) {
	args := m.Called(userID, file, req)
	return getPtr[importexport.JobResponse](args, 0), args.Error(1)
}

func (m *MockBackend) CreateExportJob(userID uint, req *importexport.ExportRequest) (*importexport.JobResponse, error) {
	args := m.Called(userID, req)
	return getPtr[importexport.JobResponse](args, 0), args.Error(1)
}

func (m *MockBackend) GetJob(jobID uint) (*importexport.JobResponse, error) {
	args := m.Called(jobID)
	return getPtr[importexport.JobResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ListJobs(page, pageSize int, filters map[string]interface{}) (*importexport.JobListResponse, error) {
	args := m.Called(page, pageSize, filters)
	return getPtr[importexport.JobListResponse](args, 0), args.Error(1)
}

func (m *MockBackend) CancelJob(jobID, userID uint) error {
	return m.Called(jobID, userID).Error(0)
}

func (m *MockBackend) DeleteJob(jobID, userID uint) error {
	return m.Called(jobID, userID).Error(0)
}

func (m *MockBackend) GetJobStats() (map[string]interface{}, error) {
	args := m.Called()
	return getMap(args, 0), args.Error(1)
}

// --- User domain ---

func (m *MockBackend) CreateUser(req *user.CreateUserRequest) (*auth.UserInfo, error) {
	args := m.Called(req)
	return getPtr[auth.UserInfo](args, 0), args.Error(1)
}

func (m *MockBackend) GetUser(userID uint) (*auth.UserInfo, error) {
	args := m.Called(userID)
	return getPtr[auth.UserInfo](args, 0), args.Error(1)
}

func (m *MockBackend) UpdateUser(userID uint, req *user.UpdateUserRequest) (*auth.UserInfo, error) {
	args := m.Called(userID, req)
	return getPtr[auth.UserInfo](args, 0), args.Error(1)
}

func (m *MockBackend) DeleteUser(userID uint) error {
	return m.Called(userID).Error(0)
}

func (m *MockBackend) ListUsers(req *user.UserListRequest) (*user.UserListResponse, error) {
	args := m.Called(req)
	return getPtr[user.UserListResponse](args, 0), args.Error(1)
}

func (m *MockBackend) ActivateUser(userID uint) error {
	return m.Called(userID).Error(0)
}

func (m *MockBackend) DeactivateUser(userID uint) error {
	return m.Called(userID).Error(0)
}

func (m *MockBackend) ChangeUserPassword(userID uint, newPassword string) error {
	return m.Called(userID, newPassword).Error(0)
}

func (m *MockBackend) GetUserStats() (map[string]interface{}, error) {
	args := m.Called()
	return getMap(args, 0), args.Error(1)
}

// --- RBAC domain ---

func (m *MockBackend) GetUserPermissions(ctx context.Context, userID uint) ([]models.Permission, error) {
	args := m.Called(ctx, userID)
	return getSlice[models.Permission](args, 0), args.Error(1)
}

func (m *MockBackend) GetUserRoles(ctx context.Context, userID uint) ([]models.Role, error) {
	args := m.Called(ctx, userID)
	return getSlice[models.Role](args, 0), args.Error(1)
}

func (m *MockBackend) GetRolePermissions(ctx context.Context, roleID uint) ([]models.Permission, error) {
	args := m.Called(ctx, roleID)
	return getSlice[models.Permission](args, 0), args.Error(1)
}

func (m *MockBackend) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	args := m.Called(ctx)
	return getSlice[models.Permission](args, 0), args.Error(1)
}

func (m *MockBackend) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	args := m.Called(ctx)
	return getSlice[models.Role](args, 0), args.Error(1)
}

func (m *MockBackend) GetRoleByID(ctx context.Context, id uint) (*models.Role, error) {
	args := m.Called(ctx, id)
	return getPtr[models.Role](args, 0), args.Error(1)
}

func (m *MockBackend) GetPermissionByID(ctx context.Context, id uint) (*models.Permission, error) {
	args := m.Called(ctx, id)
	return getPtr[models.Permission](args, 0), args.Error(1)
}

func (m *MockBackend) CreateRole(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockBackend) CreatePermission(ctx context.Context, permission *models.Permission) error {
	return m.Called(ctx, permission).Error(0)
}

func (m *MockBackend) UpdateRole(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockBackend) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	return m.Called(ctx, permission).Error(0)
}

func (m *MockBackend) DeleteRole(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockBackend) DeletePermission(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockBackend) HasPermission(ctx context.Context, userID uint, permissionCode string) (bool, error) {
	args := m.Called(ctx, userID, permissionCode)
	return args.Bool(0), args.Error(1)
}

func (m *MockBackend) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

func (m *MockBackend) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

func (m *MockBackend) AssignPermissionToUser(ctx context.Context, userID, permissionID uint) error {
	return m.Called(ctx, userID, permissionID).Error(0)
}

func (m *MockBackend) RemovePermissionFromUser(ctx context.Context, userID, permissionID uint) error {
	return m.Called(ctx, userID, permissionID).Error(0)
}

func (m *MockBackend) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	return m.Called(ctx, roleID, permissionID).Error(0)
}

func (m *MockBackend) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	return m.Called(ctx, roleID, permissionID).Error(0)
}

// --- generic helpers for nil-safe return extraction ---

func getPtr[T any](args mock.Arguments, idx int) *T {
	if v := args.Get(idx); v != nil {
		return v.(*T)
	}
	return nil
}

func getSlice[T any](args mock.Arguments, idx int) []T {
	if v := args.Get(idx); v != nil {
		return v.([]T)
	}
	return nil
}

func getMap(args mock.Arguments, idx int) map[string]interface{} {
	if v := args.Get(idx); v != nil {
		return v.(map[string]interface{})
	}
	return nil
}

// --- shared test session helpers ---

// newTestSession builds a *Session for tests holding the given permission codes.
func newTestSession(perms ...string) *Session {
	m := make(map[string]bool, len(perms))
	for _, p := range perms {
		m[p] = true
	}
	return &Session{UserID: 1, Permissions: m}
}

// ctxWithSession returns a context carrying the given test session.
func ctxWithSession(s *Session) context.Context {
	return WithSession(context.Background(), s)
}
