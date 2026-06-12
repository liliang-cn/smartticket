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
	ParentID  *uint
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
	cur := &parentID
	for cur != nil {
		if id != 0 && *cur == id {
			return ErrCycle
		}
		var node models.Department
		if err := s.db.Select("id", "parent_id").First(&node, *cur).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
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
		if d.ParentID == nil {
			return nil, nil
		}
		deptID = *d.ParentID
	}
}

// DeptScopeFor returns every department ID overseen by userID — for each
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
