// file: internals/features/authz/models/user_role.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole struct {
	UserRoleID uuid.UUID  `gorm:"column:user_role_id;type:uuid;primaryKey" json:"user_role_id"`
	UserID     uuid.UUID  `gorm:"column:user_id;type:uuid;not null"        json:"user_id"`
	RoleID     uuid.UUID  `gorm:"column:role_id;type:uuid;not null"        json:"role_id"`
	MasjidID   *uuid.UUID `gorm:"column:masjid_id;type:uuid"               json:"masjid_id,omitempty"`
	AssignedAt *time.Time `gorm:"column:assigned_at"                       json:"assigned_at,omitempty"`
	AssignedBy *uuid.UUID `gorm:"column:assigned_by;type:uuid"             json:"assigned_by,omitempty"`
	DeletedAt  *time.Time `gorm:"column:deleted_at"                        json:"deleted_at,omitempty"`
}

func (UserRole) TableName() string { return "user_roles" }
