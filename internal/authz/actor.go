// Package authz provides the authorization Actor used to scope queries by the
// caller's identity and role. It is the single source of truth shared by the
// REST handlers and the MCP tools so customer-isolation rules cannot diverge.
package authz

import "github.com/company/smartticket/internal/models"

// Role constants for the built-in roles.
const (
	RoleAdmin    = "admin"
	RoleEngineer = "engineer"
	RoleCustomer = "customer"
)

// Actor represents the authenticated caller for authorization decisions.
//
//	UserID     the acting user's ID.
//	Role       the user's role (admin / engineer / customer).
//	CustomerID the customer organization the user belongs to; non-nil only for
//	           customer-role users.
type Actor struct {
	UserID     uint
	Role       string
	CustomerID *uint
}

// IsTeam reports whether the actor is operator-side staff (admin or engineer),
// who may see and manage all customers' content.
func (a Actor) IsTeam() bool {
	return a.Role == RoleAdmin || a.Role == RoleEngineer
}

// IsCustomer reports whether the actor is a customer-side user, whose view is
// restricted to their own customer organization.
func (a Actor) IsCustomer() bool {
	return a.Role == RoleCustomer
}

// IsAdmin reports whether the actor holds the admin role.
func (a Actor) IsAdmin() bool {
	return a.Role == RoleAdmin
}

// ActorFromUser builds an Actor from an authenticated user record.
func ActorFromUser(u *models.User) Actor {
	if u == nil {
		return Actor{}
	}
	return Actor{UserID: u.ID, Role: u.Role, CustomerID: u.CustomerID}
}
