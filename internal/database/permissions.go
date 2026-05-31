package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// permissionDef describes a single permission code in the catalog.
type permissionDef struct {
	Code     string
	Name     string
	Category string
}

// permissionCatalog is the full set of permission codes the system recognizes.
// Codes follow the resource:action convention used throughout the app and by
// the MCP tools (e.g. ticket:write, customer:read, admin:system).
var permissionCatalog = []permissionDef{
	{"ticket:read", "Read tickets", "tickets"},
	{"ticket:write", "Create/update tickets", "tickets"},
	{"knowledge:read", "Read knowledge articles", "knowledge"},
	{"knowledge:write", "Create/update knowledge articles", "knowledge"},
	{"product:read", "Read products", "products"},
	{"product:write", "Manage products", "products"},
	{"service:read", "Read services", "services"},
	{"service:write", "Manage services", "services"},
	{"sla:read", "Read SLA templates/rules", "sla"},
	{"sla:write", "Manage SLA templates/rules", "sla"},
	{"importexport:read", "Read import/export jobs", "importexport"},
	{"importexport:write", "Create/manage import/export jobs", "importexport"},
	{"user:read", "Read users", "users"},
	{"user:write", "Manage users", "users"},
	{"customer:read", "Read customers", "customers"},
	{"customer:write", "Manage customers", "customers"},
	{"rbac:read", "Read roles/permissions", "rbac"},
	{"rbac:write", "Manage roles/permissions", "rbac"},
	{"llm:read", "Read LLM providers", "llm"},
	{"llm:write", "Manage LLM providers", "llm"},
	{"subscription:read", "Read subscriptions", "subscription"},
	{"subscription:write", "Manage subscriptions", "subscription"},
	{"settings:read", "Read deployment settings/branding", "settings"},
	{"settings:write", "Manage deployment settings/branding", "settings"},
	{"admin:system", "Full system administration", "system"},
}

// roleGrants maps each standard role to the permission codes it is granted.
// "*" grants every code in the catalog.
var roleGrants = map[string][]string{
	"admin": {"*"},
	"engineer": {
		"ticket:read", "ticket:write",
		"knowledge:read", "knowledge:write",
		"product:read", "product:write",
		"service:read", "service:write",
		"sla:read", "sla:write",
		"importexport:read", "importexport:write",
		"customer:read", "customer:write",
		"user:read",
		"rbac:read",
		"llm:read",
		"subscription:read", "subscription:write",
		"settings:read",
	},
	"support": {
		"ticket:read", "ticket:write",
		"knowledge:read", "knowledge:write",
		"customer:read",
		"product:read", "service:read", "sla:read",
		"subscription:read",
	},
	"sales": {
		"customer:read", "customer:write",
		"ticket:read",
		"knowledge:read",
		"product:read", "service:read",
		"subscription:read",
	},
	"customer": {
		"ticket:read", "ticket:write",
		"knowledge:read",
	},
}

// standardRoleDefs are the built-in roles, created idempotently.
var standardRoleDefs = []models.Role{
	{Name: "admin", Description: "System administrator with full access", IsSystem: true},
	{Name: "engineer", Description: "Support engineer with technical access"},
	{Name: "support", Description: "Support agent handling tickets and knowledge"},
	{Name: "sales", Description: "Sales representative managing customers"},
	{Name: "customer", Description: "Customer with basic access"},
}

// EnsureRolesAndPermissions idempotently creates the standard roles, the full
// permission catalog, and the role→permission grants. It is safe to call on
// every startup and from the createadmin bootstrap path. It runs in its own
// transaction unless db is already a transaction handle (GORM nests safely via
// SAVEPOINT).
func EnsureRolesAndPermissions(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Roles.
		roleByName := map[string]models.Role{}
		for _, r := range standardRoleDefs {
			var role models.Role
			if err := tx.Where(models.Role{Name: r.Name}).
				Attrs(models.Role{Description: r.Description, IsSystem: r.IsSystem}).
				FirstOrCreate(&role).Error; err != nil {
				return fmt.Errorf("failed to ensure role %q: %w", r.Name, err)
			}
			roleByName[r.Name] = role
		}

		// Permissions.
		permByCode := map[string]models.Permission{}
		for _, p := range permissionCatalog {
			var perm models.Permission
			if err := tx.Where(models.Permission{Code: p.Code}).
				Attrs(models.Permission{Name: p.Name, Category: p.Category, IsSystem: true}).
				FirstOrCreate(&perm).Error; err != nil {
				return fmt.Errorf("failed to ensure permission %q: %w", p.Code, err)
			}
			permByCode[p.Code] = perm
		}

		// Role → permission grants.
		for roleName, codes := range roleGrants {
			role, ok := roleByName[roleName]
			if !ok {
				continue
			}
			grantCodes := codes
			if len(codes) == 1 && codes[0] == "*" {
				grantCodes = grantCodes[:0]
				for _, p := range permissionCatalog {
					grantCodes = append(grantCodes, p.Code)
				}
			}
			for _, code := range grantCodes {
				perm, ok := permByCode[code]
				if !ok {
					return fmt.Errorf("role %q references unknown permission %q", roleName, code)
				}
				var rp models.RolePermission
				if err := tx.Where(models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).
					FirstOrCreate(&rp).Error; err != nil {
					return fmt.Errorf("failed to grant %q to role %q: %w", code, roleName, err)
				}
			}
		}
		return nil
	})
}
