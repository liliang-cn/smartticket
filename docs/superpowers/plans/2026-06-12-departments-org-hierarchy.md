# Departments / Org Hierarchy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a department org tree (orthogonal to teams) that powers "find the supervisor" escalation and lets department managers see their subtree's tickets, via a centralized `authz` scoping helper (no Casbin).

**Architecture:** A nested `Department` table (`parent_id`, `manager_id`) + `User.DepartmentID`. `internal/department` provides CRUD, `SupervisorOf`, and `DeptScopeFor` (subtree dept IDs for a manager). Escalation and ticket visibility consume these via small interfaces injected into `ticket.Service` (same optional-injection pattern as notifier/mailer), so `ticket` does not hard-depend on `department`. Data scoping is centralized in `authz.Scope(q, actor, opts)`.

**Tech Stack:** Go 1.21+, GIN, GORM, modernc SQLite, testify; React/TS frontend.

**Spec:** `docs/superpowers/specs/2026-06-12-departments-org-hierarchy-design.md`

**Conventions (verified):**
- Test DB: `sqlite "github.com/company/smartticket/internal/database/moderncsqlite"`, `gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})`, testify `require`.
- CRUD module = `internal/<mod>/{service.go, handlers.go}`; admin routes under `protected.Group("/...")` + `.Use(s.adminMiddleware())` in `internal/server/server.go`; models registered in BOTH `dbModels` slices in `cmd/server/main.go` (runServer ~172 and runMigrate ~351).
- `authz.Actor{UserID, Role, CustomerID}` in `internal/authz/actor.go`; `ActorFromUser(u)`. `ticket.scopeToActor(q, actor)` currently only does customer isolation.
- `ticket.Service` has optional injected deps (`notifier Notifier`, `mailer Mailer`) set via setters; `EscalateAutomation` (ticket/service.go) bumps priority one rank. `notification.Service.Notify(ctx, userIDs []uint, ntype, title, body, refType string, refID uint)`.
- No data migration required (new table + new nullable column).

---

## Part 1 — Department model + service

### Task 1: Department model + User.DepartmentID

**Files:** Modify `internal/models/models.go`, `cmd/server/main.go`.

- [ ] **Step 1:** Add to `internal/models/models.go`:

```go
// Department is a node in the org reporting tree (orthogonal to Team). ParentID
// nests it; ManagerID is the department's lead (a staff User). Used for "find
// supervisor" escalation and manager subtree visibility.
type Department struct {
	BaseModel
	Name      string      `gorm:"size:120;not null" json:"name"`
	ParentID  *uint       `gorm:"index" json:"parent_id"`
	Parent    *Department `gorm:"foreignKey:ParentID" json:"-"`
	ManagerID *uint       `gorm:"index" json:"manager_id"`
	Manager   *User       `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
}
```
And add a field to the existing `User` struct (after `CustomerID *uint`):
```go
	DepartmentID *uint `gorm:"index" json:"department_id,omitempty"`
```

- [ ] **Step 2:** Register migration in BOTH `dbModels` slices in `cmd/server/main.go`:
```go
		&models.Department{},
```

- [ ] **Step 3:** `go build ./...` → clean.

- [ ] **Step 4:** Commit:
```bash
git add internal/models/models.go cmd/server/main.go
git commit -m "feat(department): Department model + User.DepartmentID"
```

---

### Task 2: department service — CRUD, cycle guard, SupervisorOf, DeptScopeFor (TDD)

**Files:** Create `internal/department/service.go`, `internal/department/service_test.go`.

- [ ] **Step 1:** Write `internal/department/service_test.go`:

```go
package department

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Department{}))
	return db
}

func staff(t *testing.T, db *gorm.DB, name string) models.User {
	t.Helper()
	u := models.User{Email: name + "@x.local", Username: name, PasswordHash: "-", Role: "engineer", IsActive: true}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func TestCreateRejectsCycle(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	root, err := svc.Create(CreateInput{Name: "root"})
	require.NoError(t, err)
	child, err := svc.Create(CreateInput{Name: "child", ParentID: &root.ID})
	require.NoError(t, err)
	// Making root a child of its own descendant must be rejected.
	err = svc.Update(root.ID, UpdateInput{ParentID: &child.ID})
	require.ErrorIs(t, err, ErrCycle)
}

func TestSupervisorOfWalksUp(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	mgrRoot := staff(t, db, "rootmgr")
	mgrChild := staff(t, db, "childmgr")
	agent := staff(t, db, "agent")

	root, _ := svc.Create(CreateInput{Name: "root", ManagerID: &mgrRoot.ID})
	child, _ := svc.Create(CreateInput{Name: "child", ParentID: &root.ID, ManagerID: &mgrChild.ID})
	// agent reports into child dept
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", agent.ID).Update("department_id", child.ID).Error)
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", mgrChild.ID).Update("department_id", child.ID).Error)

	// agent's supervisor = child dept manager
	sup, err := svc.SupervisorOf(agent.ID)
	require.NoError(t, err)
	require.NotNil(t, sup)
	require.Equal(t, mgrChild.ID, sup.ID)

	// childmgr IS the child manager → supervisor walks up to root dept manager
	sup2, err := svc.SupervisorOf(mgrChild.ID)
	require.NoError(t, err)
	require.NotNil(t, sup2)
	require.Equal(t, mgrRoot.ID, sup2.ID)

	// rootmgr is top → no supervisor
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", mgrRoot.ID).Update("department_id", root.ID).Error)
	sup3, err := svc.SupervisorOf(mgrRoot.ID)
	require.NoError(t, err)
	require.Nil(t, sup3)
}

func TestDeptScopeForManagerSubtree(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	mgr := staff(t, db, "mgr")
	root, _ := svc.Create(CreateInput{Name: "root", ManagerID: &mgr.ID})
	child, _ := svc.Create(CreateInput{Name: "child", ParentID: &root.ID})
	grand, _ := svc.Create(CreateInput{Name: "grand", ParentID: &child.ID})
	other, _ := svc.Create(CreateInput{Name: "other"})

	scope, err := svc.DeptScopeFor(mgr.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{root.ID, child.ID, grand.ID}, scope)
	require.NotContains(t, scope, other.ID)

	// non-manager → empty scope
	plain := staff(t, db, "plain")
	scope2, err := svc.DeptScopeFor(plain.ID)
	require.NoError(t, err)
	require.Empty(t, scope2)
}
```

- [ ] **Step 2:** Run `go test ./internal/department/ -v` → FAIL (undefined NewService).

- [ ] **Step 3:** Implement `internal/department/service.go`:

```go
// Package department manages the org reporting tree: CRUD over Department nodes,
// "find supervisor" resolution, and the set of department IDs a manager oversees
// (their subtree) for data-scoping decisions.
package department

import (
	"errors"

	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

var (
	ErrCycle    = errors.New("department parent would create a cycle")
	ErrNotFound = errors.New("department not found")
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateInput struct {
	Name      string
	ParentID  *uint
	ManagerID *uint
}

type UpdateInput struct {
	Name      *string
	ParentID  *uint // pointer-to-pointer semantics omitted for simplicity: nil = leave unchanged
	ManagerID *uint
}

func (s *Service) Create(in CreateInput) (*models.Department, error) {
	d := &models.Department{Name: in.Name, ParentID: in.ParentID, ManagerID: in.ManagerID}
	if in.ParentID != nil {
		if err := s.guardParent(0, *in.ParentID); err != nil {
			return nil, err
		}
	}
	if err := s.db.Create(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Update(id uint, in UpdateInput) error {
	updates := map[string]any{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.ManagerID != nil {
		updates["manager_id"] = *in.ManagerID
	}
	if in.ParentID != nil {
		if err := s.guardParent(id, *in.ParentID); err != nil {
			return err
		}
		updates["parent_id"] = *in.ParentID
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&models.Department{}).Where("id = ?", id).Updates(updates).Error
}

func (s *Service) Delete(id uint) error { return s.db.Delete(&models.Department{}, id).Error }

func (s *Service) List() ([]models.Department, error) {
	var ds []models.Department
	err := s.db.Order("parent_id, id").Find(&ds).Error
	return ds, err
}

// guardParent rejects setting node `id`'s parent to `parentID` when parentID is
// id itself or any descendant of id (which would create a cycle). id==0 means a
// new node (no descendants yet) so only the self-check is relevant.
func (s *Service) guardParent(id, parentID uint) error {
	if id != 0 && parentID == id {
		return ErrCycle
	}
	// Walk up from parentID; if we reach id, parentID is a descendant of id.
	cur := &parentID
	for cur != nil {
		if id != 0 && *cur == id {
			return ErrCycle
		}
		var node models.Department
		if err := s.db.Select("id", "parent_id").First(&node, *cur).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil // parent chain ends / unknown — no cycle
			}
			return err
		}
		cur = node.ParentID
	}
	return nil
}

// SupervisorOf returns the manager a user reports to: the manager of the user's
// department, unless the user IS that manager, in which case it walks up to the
// parent department's manager. Returns nil at the top of the tree / no dept.
func (s *Service) SupervisorOf(userID uint) (*models.User, error) {
	var u models.User
	if err := s.db.Select("id", "department_id").First(&u, userID).Error; err != nil {
		return nil, err
	}
	if u.DepartmentID == nil {
		return nil, nil
	}
	deptID := *u.DepartmentID
	for {
		var d models.Department
		if err := s.db.First(&d, deptID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		if d.ManagerID != nil && *d.ManagerID != userID {
			var mgr models.User
			if err := s.db.First(&mgr, *d.ManagerID).Error; err != nil {
				return nil, err
			}
			return &mgr, nil
		}
		// user is this dept's manager (or no manager) → go up
		if d.ParentID == nil {
			return nil, nil
		}
		deptID = *d.ParentID
	}
}

// DeptScopeFor returns every department ID overseen by userID — i.e. for each
// department the user manages, that department plus all descendants. Empty if
// the user manages nothing.
func (s *Service) DeptScopeFor(userID uint) ([]uint, error) {
	var roots []models.Department
	if err := s.db.Where("manager_id = ?", userID).Find(&roots).Error; err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, nil
	}
	var all []models.Department
	if err := s.db.Select("id", "parent_id").Find(&all).Error; err != nil {
		return nil, err
	}
	children := map[uint][]uint{}
	for _, d := range all {
		if d.ParentID != nil {
			children[*d.ParentID] = append(children[*d.ParentID], d.ID)
		}
	}
	seen := map[uint]bool{}
	var stack []uint
	for _, r := range roots {
		stack = append(stack, r.ID)
	}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[n] {
			continue
		}
		seen[n] = true
		stack = append(stack, children[n]...)
	}
	out := make([]uint, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
}
```

- [ ] **Step 4:** Run `go test ./internal/department/ -v` → all PASS.

- [ ] **Step 5:** Commit:
```bash
git add internal/department/service.go internal/department/service_test.go
git commit -m "feat(department): CRUD, cycle guard, SupervisorOf, DeptScopeFor"
```

---

### Task 3: department admin CRUD handlers + routes

**Files:** Create `internal/department/handlers.go`, `internal/department/handlers_test.go`; modify `internal/server/server.go`.

- [ ] **Step 1:** Write `internal/department/handlers_test.go` — TDD: POST create returns 201 with id; GET list returns the created dept. Mirror the structure of `internal/apikey/handlers_test.go` (gin.New(), httptest, `c.Set("user_id", uint(1))` where needed, reuse `newTestDB`/`staff` from service_test.go).

```go
package department

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateAndListHandlers(t *testing.T) {
	db := newTestDB(t)
	h := NewHandlers(NewService(db))
	r := gin.New()
	r.POST("/admin/departments", h.Create)
	r.GET("/admin/departments", h.List)

	req := httptest.NewRequest(http.MethodPost, "/admin/departments", strings.NewReader(`{"name":"Support"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/admin/departments", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), "Support")
}
```

- [ ] **Step 2:** Run `go test ./internal/department/ -run TestCreateAndList -v` → FAIL (undefined NewHandlers).

- [ ] **Step 3:** Implement `internal/department/handlers.go` with `Handlers{svc}`, `NewHandlers`, and methods `Create` (bind `{name, parent_id?, manager_id?}` → svc.Create → 201), `List` (svc.List → `{"departments": [...]}`), `Update` (`PUT /:id` bind `{name?, parent_id?, manager_id?}` → svc.Update; map `ErrCycle` → 400), `Delete` (`DELETE /:id`). Use `fmt.Sscanf` for `:id` parse (mirror apikey handlers). Map `ErrCycle` to `http.StatusBadRequest` with `{"error":"cycle"}`.

- [ ] **Step 4:** Run `go test ./internal/department/ -v` → all PASS.

- [ ] **Step 5:** Register routes in `internal/server/server.go` (construct `departmentService := department.NewService(s.db.DB)` near other services — KEEP this variable reachable; later tasks inject it into ticket. Then):
```go
			deptHandlers := department.NewHandlers(departmentService)
			adminDepts := protected.Group("/admin/departments")
			adminDepts.Use(s.adminMiddleware())
			{
				adminDepts.GET("", deptHandlers.List)
				adminDepts.POST("", deptHandlers.Create)
				adminDepts.PUT("/:id", deptHandlers.Update)
				adminDepts.DELETE("/:id", deptHandlers.Delete)
			}
```
Add import `"github.com/company/smartticket/internal/department"`. Run `go build ./...`.

- [ ] **Step 6:** Commit:
```bash
git add internal/department/handlers.go internal/department/handlers_test.go internal/server/server.go
git commit -m "feat(department): admin CRUD handlers and routes"
```

---

## Part 2 — Escalation finds the supervisor

### Task 4: EscalateAutomation notifies the supervisor

**Files:** Modify `internal/ticket/service.go` (add `SupervisorResolver` optional dep + setter, extend `EscalateAutomation`); modify `internal/server/server.go` (inject); test `internal/ticket/escalation_supervisor_test.go`.

- [ ] **Step 1:** In `internal/ticket/service.go`, define a narrow interface + optional field + setter (mirror the existing `notifier`/`mailer` optional pattern):
```go
// SupervisorResolver resolves the manager a user reports to (see department svc).
type SupervisorResolver interface {
	SupervisorOf(userID uint) (*models.User, error)
}
```
Add field `supervisors SupervisorResolver` to `Service` struct and a setter:
```go
func (s *Service) SetSupervisors(r SupervisorResolver) { s.supervisors = r }
```

- [ ] **Step 2:** Extend `EscalateAutomation` — after the existing priority bump succeeds, notify the assignee's supervisor (best-effort; never fails the escalation):
```go
func (s *Service) EscalateAutomation(ticketID uint) error {
	escalate := map[string]string{"low": "medium", "medium": "high", "high": "critical"}
	var tkt models.Ticket
	if err := s.db.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
		return fmt.Errorf("EscalateAutomation: load: %w", err)
	}
	if next, ok := escalate[tkt.Priority]; ok {
		if err := s.SetFieldAutomation(ticketID, "priority", next); err != nil {
			return err
		}
	}
	// Best-effort: notify the assignee's supervisor so escalation reaches a person.
	if s.supervisors != nil && tkt.AssignedTo != nil && s.notifier != nil {
		if sup, err := s.supervisors.SupervisorOf(*tkt.AssignedTo); err == nil && sup != nil {
			body := fmt.Sprintf("Ticket %s was escalated and needs your attention.", tkt.TicketNumber)
			s.notifier.Notify(context.Background(), []uint{sup.ID}, "ticket_escalated",
				"Ticket escalated", body, "ticket", tkt.ID)
		}
	}
	return nil
}
```
VERIFIED: the existing `ticket.Notifier` interface is exactly `Notify(ctx context.Context, userIDs []uint, ntype, title, body, refType string, refID uint)` (internal/ticket/notifier.go) — so reuse it directly as above; do NOT add a new interface method. `context` and `fmt` are already imported in ticket/service.go.

- [ ] **Step 3:** Write `internal/ticket/escalation_supervisor_test.go`: a fake `SupervisorResolver` returning a known user and a fake notifier capturing calls; assert that escalating a ticket with an assignee notifies the supervisor, and that a ticket with no assignee (or nil resolver) does NOT panic and still bumps priority. Reuse the package's existing test DB/setup helpers (read an existing `internal/ticket/*_test.go` for the setup pattern).

- [ ] **Step 4:** Run `go test ./internal/ticket/ -run Escalat -v` → PASS. Run `go build ./...`.

- [ ] **Step 5:** Inject in `internal/server/server.go`: after both `ticketService` and `departmentService` exist:
```go
	ticketService.SetSupervisors(departmentService)
```
(`department.Service.SupervisorOf` satisfies `ticket.SupervisorResolver`.) Run `go build ./...`.

- [ ] **Step 6:** Commit:
```bash
git add internal/ticket/service.go internal/ticket/escalation_supervisor_test.go internal/server/server.go
git commit -m "feat(department): escalation notifies the assignee's supervisor"
```

---

## Part 3 — Centralized data scoping

### Task 5: authz.Actor.DeptScope + authz.Scope helper (TDD)

**Files:** Modify `internal/authz/actor.go`; create `internal/authz/scope.go`, `internal/authz/scope_test.go`.

- [ ] **Step 1:** Add field to `authz.Actor` in `internal/authz/actor.go`:
```go
	// DeptScope is the set of department IDs a manager oversees (their subtree).
	// Empty for non-managers. Filled by callers that have department data.
	DeptScope []uint
```

- [ ] **Step 2:** Write `internal/authz/scope_test.go`:
```go
package authz

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func scopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Ticket{}))
	return db
}

func countScoped(t *testing.T, db *gorm.DB, actor Actor, opts ScopeOptions) int64 {
	t.Helper()
	q := Scope(db, db.Model(&models.Ticket{}), actor, opts)
	var n int64
	require.NoError(t, q.Count(&n).Error)
	return n
}

func TestScopeCustomerIsolation(t *testing.T) {
	db := scopeTestDB(t)
	cid := uint(7)
	require.NoError(t, db.Create(&models.Ticket{CustomerID: &cid, Title: "a"}).Error)
	other := uint(8)
	require.NoError(t, db.Create(&models.Ticket{CustomerID: &other, Title: "b"}).Error)
	opts := ScopeOptions{CustomerColumn: "customer_id", AssigneeColumn: "assigned_to"}

	cust := Actor{UserID: 1, Role: RoleCustomer, CustomerID: &cid}
	require.Equal(t, int64(1), countScoped(t, db, cust, opts))

	// customer with nil CustomerID sees nothing
	require.Equal(t, int64(0), countScoped(t, db, Actor{UserID: 2, Role: RoleCustomer}, opts))

	// admin sees all
	require.Equal(t, int64(2), countScoped(t, db, Actor{UserID: 3, Role: RoleAdmin}, opts))
}

func TestScopeDepartmentIsolation(t *testing.T) {
	db := scopeTestDB(t)
	// two staff users in different departments
	d1, d2 := uint(10), uint(20)
	u1 := models.User{Email: "u1@x", Username: "u1", PasswordHash: "-", Role: "engineer", DepartmentID: &d1}
	u2 := models.User{Email: "u2@x", Username: "u2", PasswordHash: "-", Role: "engineer", DepartmentID: &d2}
	require.NoError(t, db.Create(&u1).Error)
	require.NoError(t, db.Create(&u2).Error)
	require.NoError(t, db.Create(&models.Ticket{Title: "t1", AssignedTo: &u1.ID}).Error)
	require.NoError(t, db.Create(&models.Ticket{Title: "t2", AssignedTo: &u2.ID}).Error)

	mgr := Actor{UserID: 99, Role: RoleEngineer, DeptScope: []uint{d1}}

	// isolation OFF → staff sees all
	require.Equal(t, int64(2), countScoped(t, db, mgr, ScopeOptions{CustomerColumn: "customer_id", AssigneeColumn: "assigned_to", DepartmentIsolation: false}))

	// isolation ON → manager sees only tickets assigned to users in dept d1
	require.Equal(t, int64(1), countScoped(t, db, mgr, ScopeOptions{CustomerColumn: "customer_id", AssigneeColumn: "assigned_to", DepartmentIsolation: true}))
}
```

- [ ] **Step 3:** Implement `internal/authz/scope.go`:
```go
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
	// non-admin staff
	if opts.DepartmentIsolation && opts.AssigneeColumn != "" {
		sub := db.Model(&models.User{}).Select("id")
		if len(actor.DeptScope) > 0 {
			sub = sub.Where("department_id IN ?", actor.DeptScope)
		} else {
			sub = sub.Where("1 = 0") // staff with no managed depts see nothing under isolation
		}
		return q.Where(opts.AssigneeColumn+" IN (?)", sub)
	}
	return q // isolation off → staff see all (unchanged behavior)
}
```

- [ ] **Step 4:** Run `go test ./internal/authz/ -v` → all PASS.

- [ ] **Step 5:** Commit:
```bash
git add internal/authz/actor.go internal/authz/scope.go internal/authz/scope_test.go
git commit -m "feat(authz): centralized Scope helper (customer + department subtree)"
```

---

### Task 6: ticket delegates to authz.Scope + DepartmentIsolation setting + my_department endpoint

**Files:** Modify `internal/ticket/service.go` (scopeToActor delegates; ListTickets accepts a department-scope filter), `internal/server/server.go` (read setting, fill DeptScope, route), `internal/models/models.go` if a setting field is needed. Test `internal/ticket/scope_department_test.go`.

- [ ] **Step 1 — DepartmentIsolation setting:** Use the generic `SystemSetting` key/value table (`models.SystemSetting{Key string, Value string, ...}`, table `system_settings`). Convention: key `"department_isolation"`, value `"true"`/`"false"` (absent = false). Add a tiny read helper (in the server layer or department service): `departmentIsolation(db) bool` doing `db.Where("key = ?", "department_isolation").First(&s)` and `s.Value == "true"`. The admin toggle is set by upserting that row (expose a `PUT /api/v1/settings/department-isolation {enabled bool}` admin route, or fold into the departments page settings — keep it minimal). Inject the bool into ticket via `SetDepartmentIsolation(fn func() bool)` so it reflects changes without restart.

- [ ] **Step 2 — inject a DeptScoper into ticket:** Add another optional interface to `ticket.Service`:
```go
// DeptScoper yields the department IDs a manager oversees (their subtree).
type DeptScoper interface {
	DeptScopeFor(userID uint) ([]uint, error)
}
```
Add field `deptScoper DeptScoper` + setter `SetDeptScoper`. In `server.go` call `ticketService.SetDeptScoper(departmentService)` (department.Service.DeptScopeFor satisfies it).

- [ ] **Step 3 — delegate scopeToActor:** Replace `ticket.scopeToActor` body so it enriches the actor with DeptScope (via `s.deptScoper`, best-effort) and delegates to `authz.Scope`:
```go
func (s *Service) scopeToActor(q *gorm.DB, actor authz.Actor) *gorm.DB {
	if s.deptScoper != nil && actor.IsTeam() && !actor.IsAdmin() && len(actor.DeptScope) == 0 {
		if scope, err := s.deptScoper.DeptScopeFor(actor.UserID); err == nil {
			actor.DeptScope = scope
		}
	}
	return authz.Scope(s.db, q, actor, authz.ScopeOptions{
		CustomerColumn:      "customer_id",
		AssigneeColumn:      "assigned_to",
		DepartmentIsolation: s.departmentIsolation(),
	})
}
```
where `s.departmentIsolation()` returns the setting (inject the bool via a setter `SetDepartmentIsolation(func() bool)` or a simple `departmentIsolationFn func() bool` field so it can change at runtime without restart; default returns false when unset). NOTE: `scopeToActor` is currently a free function — convert call sites to the method `s.scopeToActor(...)` (search `scopeToActor(` in the ticket package and update; they're already inside `*Service` methods so `s.scopeToActor` is in scope).

- [ ] **Step 4 — my_department endpoint:** In the ticket list handler/route, support `?scope=my_department`: when present and the actor is staff, force department-subtree filtering regardless of the global isolation toggle (compute DeptScope and apply the assignee filter). Implement as a `ListTicketsForDepartment(actor)` service method that calls `authz.Scope` with `DepartmentIsolation: true` after filling DeptScope. Wire it in the existing tickets GET handler when the query param is set.

- [ ] **Step 5 — tests** `internal/ticket/scope_department_test.go`: with a fake DeptScoper + two assigned tickets, assert: isolation off → ListTickets returns both for a manager; isolation on → returns only subtree; `my_department` path → only subtree even when isolation off; admin → all; customer isolation still works (regression). Reuse existing ticket test setup helpers.

- [ ] **Step 6:** `go test ./internal/ticket/ ./internal/authz/ -count=1` → PASS. `go build ./...`.

- [ ] **Step 7:** Commit:
```bash
git add internal/ticket/service.go internal/ticket/scope_department_test.go internal/server/server.go
git commit -m "feat(department): ticket scoping via authz.Scope + isolation toggle + my_department view"
```

---

## Part 4 — Frontend

### Task 7: Departments admin page (tree editor) + user department field

**Files:** Create `web/src/pages/departments.tsx`, `web/src/features/departments/api.ts`; modify `web/src/App.tsx`, `web/src/components/app-shell.tsx`, `web/src/locales/*`; modify the user create/edit form to include a Department select.

- [ ] **Step 1:** Mirror `web/src/features/apikeys/api.ts` and `web/src/pages/api-keys.tsx` (built earlier) for conventions. Build `features/departments/api.ts` with TanStack Query hooks: `useDepartments`, `useCreateDepartment`, `useUpdateDepartment`, `useDeleteDepartment` over `/admin/departments`.

- [ ] **Step 2:** Build `departments.tsx`: render departments as an indented tree (compute nesting from `parent_id`), each node showing Name + Manager + actions (Add child, Edit name/manager/parent, Delete). Create/Edit dialogs with Name input, Parent select (other departments), Manager select (staff users). Use the project's existing user-list query to populate the manager dropdown (find the existing users query hook). On `ErrCycle` (400) show an inline error toast.

- [ ] **Step 3:** Add the Department select to the user edit form (find `web/src/pages/users.tsx` or the user dialog) — a dropdown bound to `department_id`, sending it on user update. Confirm the user update endpoint accepts `department_id` (the User model now has it; verify the user update handler whitelists it — if not, add `department_id` to the allowed update fields in the user service/handler).

- [ ] **Step 4:** Wire route `/departments` (AdminOnly) in `App.tsx` and a nav entry in `app-shell.tsx` (lucide `Building2` or `Network` icon, `admin: true`, `labelKey: "nav.departments"`). Add `web/src/locales/*/departments.json` for all 7 languages + `nav.departments` in all 7 `common.json` (mirror what was done for api-keys/webhooks).

- [ ] **Step 5:** `cd web && pnpm build` → no TS errors.

- [ ] **Step 6:** Commit:
```bash
git add web/src/pages/departments.tsx web/src/features/departments web/src/App.tsx web/src/components/app-shell.tsx web/src/locales web/src/pages/users.tsx
git commit -m "feat(department): admin departments tree page + user department field"
```

### Task 8: Ticket list "My department" filter

**Files:** Modify the tickets list page (`web/src/pages/tickets.tsx` or equivalent).

- [ ] **Step 1:** Add a "My department" toggle/filter visible to staff that, when on, calls the tickets list with `?scope=my_department`. Mirror existing ticket filter controls. Hide it for customer-role users.
- [ ] **Step 2:** `cd web && pnpm build` → no TS errors.
- [ ] **Step 3:** Commit:
```bash
git add web/src/pages/tickets.tsx
git commit -m "feat(department): my-department filter on ticket list"
```

---

## Part 5 — Finalize

### Task 9: OpenAPI regen + full verification

- [ ] **Step 1:** Add Swagger annotations to `internal/department/handlers.go` (Create/List/Update/Delete, `@Tags departments`, `@Security BearerAuth`), mirroring `internal/auth/handlers.go` style.
- [ ] **Step 2:** Regenerate: `swag init -g cmd/server/main.go --parseDependency --parseInternal -o docs` (install swag if missing; the `--parseDependency` flag is REQUIRED). Confirm `/admin/departments` paths appear in `docs/swagger.yaml`. If regen fails, note it and proceed (annotations still committed).
- [ ] **Step 3:** Full sweep: `go build ./...` (clean), `go test ./... 2>&1 | tail -40` (report pass/fail; distinguish this feature's failures from pre-existing), `cd web && pnpm build` (rolldown-vite, sub-second). Report all results.
- [ ] **Step 4:** Commit:
```bash
git add docs/ internal/department/handlers.go
git commit -m "docs(api): regenerate OpenAPI for departments"
```

---

## Self-Review Notes
- **Spec coverage:** Department model + User.DepartmentID (Task 1); supervisorOf + DeptScopeFor + cycle guard (Task 2); admin CRUD (Task 3); escalation→supervisor notify (Task 4); authz.Scope centralized helper (Task 5); ticket delegates + isolation toggle + my_department (Task 6); frontend tree + user dept field (Task 7) + ticket filter (Task 8); OpenAPI (Task 9). All spec sections covered.
- **Decoupling:** `ticket` depends on `department` only via the `SupervisorResolver` and `DeptScoper` interfaces it defines (injected in server.go), so no import cycle. `authz.Scope` imports only `models` + gorm.
- **Type consistency:** `department.Service` methods (`Create/Update/Delete/List/SupervisorOf/DeptScopeFor`) used consistently across Tasks 2-6; `authz.ScopeOptions{CustomerColumn, AssigneeColumn, DepartmentIsolation}` consistent across Tasks 5-6.
- **No migration:** new `Department` table + nullable `User.DepartmentID`; AutoMigrate handles both.
- **Open detail for implementer (Task 4 & 6):** confirm the exact existing `ticket.Notifier` interface shape before adding `NotifyEscalation`, and pick the lowest-friction existing setting mechanism for `department_isolation` — both flagged inline.
