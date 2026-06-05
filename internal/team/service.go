// Package team manages agent teams and their memberships.
// Teams are collections of users (agents/admins) used for ticket assignment
// and @mention routing. All mutations are admin-only; membership reads are
// available to any authenticated user.
package team

import (
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides team management business logic.
type Service struct {
	db *gorm.DB
}

// NewService creates a new team service.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// CreateRequest is the payload for creating a team.
type CreateRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=120"`
	Description string `json:"description"`
}

// UpdateRequest is the payload for updating a team. All fields are optional.
type UpdateRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=120"`
	Description *string `json:"description"`
}

// AddMemberRequest is the payload for adding a member to a team.
type AddMemberRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// TeamResponse is the JSON view of a team.
type TeamResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MemberResponse is the JSON view of a team member (the user record).
type MemberResponse struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

func toTeamResponse(t *models.Team) TeamResponse {
	return TeamResponse{ID: t.ID, Name: t.Name, Description: t.Description}
}

func toMemberResponse(u *models.User) MemberResponse {
	return MemberResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
	}
}

// ListTeams returns all teams ordered by name.
func (s *Service) ListTeams() ([]TeamResponse, error) {
	var teams []models.Team
	if err := s.db.Order("name ASC").Find(&teams).Error; err != nil {
		return nil, errors.NewDatabaseError("list teams", err)
	}
	out := make([]TeamResponse, len(teams))
	for i := range teams {
		out[i] = toTeamResponse(&teams[i])
	}
	return out, nil
}

// CreateTeam creates a new team.
func (s *Service) CreateTeam(req *CreateRequest) (*TeamResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, errors.NewValidationError("team name is required")
	}
	t := &models.Team{Name: req.Name, Description: strings.TrimSpace(req.Description)}
	if err := s.db.Create(t).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, errors.NewValidationError("a team with that name already exists")
		}
		return nil, errors.NewDatabaseError("create team", err)
	}
	r := toTeamResponse(t)
	return &r, nil
}

// GetTeam returns a single team by ID.
func (s *Service) GetTeam(id uint) (*TeamResponse, error) {
	var t models.Team
	if err := s.db.First(&t, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("team")
		}
		return nil, errors.NewDatabaseError("get team", err)
	}
	r := toTeamResponse(&t)
	return &r, nil
}

// UpdateTeam patches the mutable fields of a team.
func (s *Service) UpdateTeam(id uint, req *UpdateRequest) (*TeamResponse, error) {
	var t models.Team
	if err := s.db.First(&t, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("team")
		}
		return nil, errors.NewDatabaseError("get team", err)
	}
	if req.Name != nil {
		t.Name = strings.TrimSpace(*req.Name)
		if t.Name == "" {
			return nil, errors.NewValidationError("team name cannot be empty")
		}
	}
	if req.Description != nil {
		t.Description = strings.TrimSpace(*req.Description)
	}
	if err := s.db.Save(&t).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, errors.NewValidationError("a team with that name already exists")
		}
		return nil, errors.NewDatabaseError("update team", err)
	}
	r := toTeamResponse(&t)
	return &r, nil
}

// DeleteTeam deletes a team and all its memberships.
func (s *Service) DeleteTeam(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete memberships first to avoid FK issues.
		if err := tx.Where("team_id = ?", id).Delete(&models.TeamMember{}).Error; err != nil {
			return errors.NewDatabaseError("delete team members", err)
		}
		result := tx.Delete(&models.Team{}, id)
		if result.Error != nil {
			return errors.NewDatabaseError("delete team", result.Error)
		}
		if result.RowsAffected == 0 {
			return errors.NewNotFoundError("team")
		}
		return nil
	})
}

// AddMember adds a user to a team. It is idempotent: if the user is already a
// member the operation succeeds without creating a duplicate row.
func (s *Service) AddMember(teamID, userID uint) error {
	// Verify the team and user exist.
	var t models.Team
	if err := s.db.First(&t, teamID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("team")
		}
		return errors.NewDatabaseError("get team", err)
	}
	var u models.User
	if err := s.db.First(&u, userID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("user")
		}
		return errors.NewDatabaseError("get user", err)
	}

	member := &models.TeamMember{TeamID: teamID, UserID: userID}
	if err := s.db.Create(member).Error; err != nil {
		if isUniqueViolation(err) {
			// Already a member — treat as success (idempotent).
			return nil
		}
		return errors.NewDatabaseError("add team member", err)
	}
	return nil
}

// RemoveMember removes a user from a team. It is a no-op if the user is not a
// member (not found is not an error for a remove operation).
func (s *Service) RemoveMember(teamID, userID uint) error {
	result := s.db.Where("team_id = ? AND user_id = ?", teamID, userID).Delete(&models.TeamMember{})
	if result.Error != nil {
		return errors.NewDatabaseError("remove team member", result.Error)
	}
	return nil
}

// ListMembers returns the users belonging to a team.
func (s *Service) ListMembers(teamID uint) ([]MemberResponse, error) {
	// Verify the team exists.
	var t models.Team
	if err := s.db.First(&t, teamID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("team")
		}
		return nil, errors.NewDatabaseError("get team", err)
	}

	// Resolve member user IDs then load users.
	var memberIDs []uint
	if err := s.db.Model(&models.TeamMember{}).
		Where("team_id = ?", teamID).
		Pluck("user_id", &memberIDs).Error; err != nil {
		return nil, errors.NewDatabaseError("list team member ids", err)
	}
	if len(memberIDs) == 0 {
		return []MemberResponse{}, nil
	}

	var users []models.User
	if err := s.db.Where("id IN ?", memberIDs).Find(&users).Error; err != nil {
		return nil, errors.NewDatabaseError("list team members", err)
	}

	out := make([]MemberResponse, len(users))
	for i := range users {
		out[i] = toMemberResponse(&users[i])
	}
	return out, nil
}

// isUniqueViolation reports whether the GORM/SQLite error is a UNIQUE constraint
// violation. We inspect the error text because SQLite does not provide a
// structured error code through the moderncsqlite driver.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "unique violation") ||
		strings.Contains(msg, "duplicate")
}

// UserTeams returns the list of team IDs that a given user belongs to.
// Used internally by the @mention resolver to route mention notifications.
func (s *Service) UserTeams(userID uint) ([]uint, error) {
	var ids []uint
	if err := s.db.Model(&models.TeamMember{}).
		Where("user_id = ?", userID).
		Pluck("team_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("list user teams: %w", err)
	}
	return ids, nil
}
