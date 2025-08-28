// file: internals/features/users/profile/model/users_profile_formal_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsersProfileFormalModel merepresentasikan tabel users_profile_formal
type UsersProfileFormalModel struct {
	// PK
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	// FK -> users(id)
	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_users_profile_formal_user" json:"user_id"`

	// Data orang tua & wali
	FatherName    *string `gorm:"type:varchar(50)" json:"father_name"`
	FatherPhone   *string `gorm:"type:varchar(20)" json:"father_phone"`
	MotherName    *string `gorm:"type:varchar(50)" json:"mother_name"`
	MotherPhone   *string `gorm:"type:varchar(20)" json:"mother_phone"`
	Guardian      *string `gorm:"type:varchar(50)" json:"guardian"`
	GuardianPhone *string `gorm:"type:varchar(20)" json:"guardian_phone"`

	// Timestamps
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt *time.Time     `gorm:"type:timestamptz" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index" json:"deleted_at,omitempty"`

	// Optional: preload/constraint ke UserModel (jika dibutuhkan)
	// User UserModel `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName override
func (UsersProfileFormalModel) TableName() string {
	return "users_profile_formal"
}
