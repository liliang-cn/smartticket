// Package apikey issues and validates long-lived machine credentials. Each key
// binds a service-account user; authentication resolves that user so all RBAC
// checks downstream behave exactly as for a JWT-authenticated request.
package apikey

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/utils"
	"gorm.io/gorm"
)

const keyPrefixLabel = "stk_live" // GenerateAPIKey appends "_<token>"

var (
	ErrInvalid = errors.New("invalid api key")
	ErrRevoked = errors.New("api key revoked")
	ErrExpired = errors.New("api key expired")
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func hashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Create issues a new key bound to userID. The plaintext is returned ONCE and
// never stored. expiresAt nil = never. createdBy is the admin's user ID.
func (s *Service) Create(name string, userID uint, expiresAt *time.Time, createdBy uint) (string, *models.APIKey, error) {
	plaintext := utils.GenerateAPIKey(keyPrefixLabel, 32) // -> "stk_live_<token>"
	key := &models.APIKey{
		Name:      name,
		KeyHash:   hashKey(plaintext),
		KeyPrefix: plaintext[:12],
		UserID:    userID,
		IsActive:  true,
		ExpiresAt: expiresAt,
		CreatorID: createdBy,
	}
	if err := s.db.Create(key).Error; err != nil {
		return "", nil, err
	}
	return plaintext, key, nil
}

// Authenticate resolves the service-account user for a plaintext key, or an
// error (ErrInvalid / ErrRevoked / ErrExpired). LastUsedAt is bumped async.
func (s *Service) Authenticate(plaintext string) (*models.User, error) {
	var key models.APIKey
	if err := s.db.Where("key_hash = ?", hashKey(plaintext)).First(&key).Error; err != nil {
		return nil, ErrInvalid
	}
	if !key.IsActive {
		return nil, ErrRevoked
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpired
	}
	// Resolve the bound user, mirroring the JWT path which rejects inactive
	// accounts on every request: a deactivated service account must not keep
	// authenticating via its API keys.
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", key.UserID, true).First(&user).Error; err != nil {
		return nil, ErrInvalid
	}
	now := time.Now()
	go func() { _ = s.db.Model(&models.APIKey{}).Where("id = ?", key.ID).Update("last_used_at", &now).Error }()
	return &user, nil
}

func (s *Service) List() ([]models.APIKey, error) {
	var keys []models.APIKey
	err := s.db.Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (s *Service) Revoke(id uint) error {
	return s.db.Model(&models.APIKey{}).Where("id = ?", id).Update("is_active", false).Error
}
