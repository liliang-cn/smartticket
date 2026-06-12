package authz

import (
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// ScopeOptions configures Scope for a particular resource's columns.
type ScopeOptions struct {
	// CustomerColumn matches actor.CustomerID for customer isolation (e.g. "customer_id").
	CustomerColumn string
	// AssigneeColumn holds the staff user id a row is assigned to (e.g. "assigned_to").
	// Used for department-subtree scoping. Empty disables dept scoping.
	AssigneeColumn string
	// DepartmentIsolation, when true, restricts staff to their department subtree.
	DepartmentIsolation bool
}

// Scope restricts q to what actor may see: customers to their own organization;
// admins to everything; and (when DepartmentIsolation is on) non-admin staff to
// rows assigned to users within actor.DeptScope. db is needed for the subquery.
func Scope(db *gorm.DB, q *gorm.DB, actor Actor, opts ScopeOptions) *gorm.DB {
	if actor.IsCustomer() {
		if actor.CustomerID != nil && opts.CustomerColumn != "" {
			return q.Where(opts.CustomerColumn+" = ?", *actor.CustomerID)
		}
		return q.Where("1 = 0") // misconfigured customer → see nothing (IDOR guard)
	}
	if actor.IsAdmin() {
		return q
	}
	if opts.DepartmentIsolation && opts.AssigneeColumn != "" {
		sub := db.Model(&models.User{}).Select("id")
		if len(actor.DeptScope) > 0 {
			sub = sub.Where("department_id IN ?", actor.DeptScope)
		} else {
			sub = sub.Where("1 = 0") // staff with no managed depts see nothing under isolation
		}
		return q.Where(opts.AssigneeColumn+" IN (?)", sub)
	}
	return q
}
