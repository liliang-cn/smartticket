# Customer Organizations — Design

**Date:** 2026-05-30
**Status:** Approved (brainstorming) — implementing directly per user request
**Branch:** `feat/customer-organizations`

## 1. Goal

Model a **customer** as an organization/company that has multiple contact users,
within the single-org deployment. Customer-side users log in and collaborate on
their company's tickets; different customers cannot see each other's content; the
operator's own team (admin/engineer) sees everything. This is a business entity
(the operator's clients) — NOT a return to multi-tenancy.

## 2. Decisions (from brainstorming)

| Topic | Decision |
|-------|----------|
| Customer entity | New `Customer` (company). Customer-role users each belong to exactly one Customer (`User.CustomerID`); team users (admin/engineer) belong to none. |
| Ticket linkage | `Ticket.CustomerID`. Company-internal shared visibility: all contacts of a customer see that customer's tickets; team sees all. |
| Isolation | Different customers cannot see each other's content. Team sees all customers. |
| Management (v1) | Team-only. admin/engineer create customers and their contact accounts via the API. Customer users can log in and view/create their company's tickets, but cannot manage users. |
| Enforcement | Service layer, via an `Actor` parameter — single source of truth shared by REST and MCP. |
| Scope (v1) | Customer isolation applies to **tickets and their messages**. Knowledge base keeps its existing published/internal visibility (not customer-scoped in v1). |
| Interface | Server-side API (REST for the future product UI; MCP tools kept consistent). CLI stays minimal (`createadmin` bootstrap only). |

## 3. Data Model

**New `Customer`** (`internal/models`):
```
Customer {
  BaseModel
  Name        string  // required
  Code        string  // optional, uniqueIndex (short reference code)
  Domain      string  // optional (email domain; reserved for future self-registration)
  IsActive    bool    // default true
  Description string
  Users    []User    `foreignKey:CustomerID`
  Tickets  []Ticket  `foreignKey:CustomerID`
}
```
- `User`: add `CustomerID *uint` (nullable, indexed) + `Customer *Customer`. Null for team users.
- `Ticket`: add `CustomerID *uint` (nullable, indexed) + `Customer *Customer`.
- Migration: add `Customer` to AutoMigrate **before** `User`/`Ticket`. Adding nullable
  columns is additive; existing rows get `CustomerID = NULL` (team/historical) — no backfill required.

## 4. Actor & Visibility Enforcement

**`Actor`** (`internal/authz` — new small package, or `internal/auth`):
```
Actor { UserID uint; Role string; CustomerID *uint }
func (a Actor) IsTeam() bool      // role == admin || engineer
func (a Actor) IsCustomer() bool  // role == customer (CustomerID must be non-nil)
```

**Construction (shared semantics, two entry points):**
- **REST:** built from the authenticated `*models.User` in the gin context (set by the
  permission middleware). Helper `authz.ActorFromUser(*models.User) Actor`.
- **MCP:** `Authenticator.Authenticate` additionally loads the user's `Role` and
  `CustomerID` into `Session`; tools derive the same `Actor` from the session.

**Ticket service scoping** (ticket methods take an `actor authz.Actor` argument):
- `actor.IsCustomer()`:
  - All list/get/stats queries append `WHERE customer_id = ?` (= `actor.CustomerID`).
  - `GetTicket` on a ticket outside the actor's customer → **NotFound** (not Forbidden;
    avoids existence disclosure).
  - `CreateTicket` forces `CustomerID = actor.CustomerID` (a customer cannot file for
    another company).
  - `Update`/`Delete` restricted to the actor's customer.
  - `AssignTicket` (assign to an engineer) is **team-only** → customer actor rejected
    (Forbidden).
- `actor.IsTeam()`: unrestricted; team may set `CustomerID` explicitly when creating.

**Messages:** customers see messages only for tickets they can access (transitively
scoped through ticket access). `Message.IsInternal == true` notes are **hidden from
customer actors** and visible only to team actors.

## 5. API Surface (REST, for the UI)

**Customer management (team-only):**
- `POST   /api/v1/customers` — create
- `GET    /api/v1/customers` — list
- `GET    /api/v1/customers/:id` — get
- `PUT    /api/v1/customers/:id` — update
- `DELETE /api/v1/customers/:id` — soft delete
- `GET    /api/v1/customers/:id/users` — list the customer's contacts

**User management (team-only), extended:**
- `POST /api/v1/users` accepts optional `customer_id` + `role`.
  - `role == customer` requires `customer_id`.
  - team roles (`admin`/`engineer`) forbid `customer_id`.

**Tickets:** existing endpoints, now Actor-scoped (see §4). Customer-created tickets
auto-set `customer_id`; team may pass `customer_id`.

**Permission codes:** `customer:read` / `customer:write` for customer management
(team). Tickets continue to use `ticket:read` / `ticket:write`; the `customer` role is
expected to carry `ticket:read` + `ticket:write` for its own company's tickets.
(Permission seeding gap is tracked separately — see the existing memory note.)

## 6. MCP Consistency

Because enforcement is at the service layer, MCP inherits isolation automatically once
its ticket Backend calls pass the `Actor`:
- Extend `Session` with `Role` + `CustomerID`; ticket tools build the `Actor` and pass
  it through `Backend` → service. **Required for security** (otherwise a customer MCP
  token could read all tickets).
- Add customer-management MCP tools (`customer_create/get/list/update/delete`) for
  parity, gated by `customer:write`/`customer:read`.

## 7. Access Control Summary

| Action | admin | engineer | customer |
|--------|-------|----------|----------|
| Manage customers | ✅ | ✅ | ❌ |
| Create users (team or customer contacts) | ✅ | ✅ | ❌ |
| See all tickets | ✅ | ✅ | ❌ (only own customer) |
| Create/view own company's tickets | ✅ | ✅ | ✅ (own customer) |
| Assign ticket to engineer | ✅ | ✅ | ❌ |
| See internal notes | ✅ | ✅ | ❌ |

## 8. Testing

- **Service Actor scoping** (in-memory SQLite): customer sees only own customer's
  tickets; team sees all; cross-customer `GetTicket` → NotFound; customer `CreateTicket`
  auto-sets `customer_id`; customer `AssignTicket` → Forbidden; internal notes hidden
  from customer.
- **Customer CRUD** service + handler tests.
- **User create** with `customer_id` validation (customer requires it; team forbids it).
- **MCP**: ticket tools enforce isolation via Actor (customer session scoped); customer
  tools gated by permission.
- Full suite green; `go build/vet/test ./...`.

## 9. Out of Scope (v1)

- Customer self-registration / email-domain auto-association (Domain field reserved).
- Customer-admin sub-role (self-service contact management).
- Customer isolation for knowledge base (keeps existing visibility model).
- Many-to-many users↔customers.
