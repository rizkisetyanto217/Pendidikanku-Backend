package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"column:user_id;type:uuid;not null" json:"user_id"`

	// simpan HASH token (bukan plaintext)
	TokenHash []byte     `gorm:"column:token_hash;type:bytea;not null" json:"-"`

	ExpiresAt time.Time  `gorm:"column:expires_at;type:timestamptz;not null" json:"expires_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at;type:timestamptz" json:"revoked_at,omitempty"`

	UserAgent *string    `gorm:"column:user_agent" json:"user_agent,omitempty"`
	IP        *string    `gorm:"column:ip;type:inet" json:"ip,omitempty"`

	CreatedAt time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName override
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
