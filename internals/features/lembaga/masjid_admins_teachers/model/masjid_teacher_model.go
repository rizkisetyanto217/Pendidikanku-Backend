package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidTeacherModel struct {
	// PK
	MasjidTeacherID uuid.UUID `gorm:"column:masjid_teacher_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_teacher_id"`

	// FK
	MasjidTeacherMasjidID uuid.UUID `gorm:"column:masjid_teacher_masjid_id;type:uuid;not null;index" json:"masjid_teacher_masjid_id"`
	MasjidTeacherUserID   uuid.UUID `gorm:"column:masjid_teacher_user_id;type:uuid;not null;index" json:"masjid_teacher_user_id"`

	// timestamps
	MasjidTeacherCreatedAt time.Time      `gorm:"column:masjid_teacher_created_at;autoCreateTime" json:"masjid_teacher_created_at"`
	MasjidTeacherUpdatedAt time.Time      `gorm:"column:masjid_teacher_updated_at;autoUpdateTime" json:"masjid_teacher_updated_at"`
	MasjidTeacherDeletedAt gorm.DeletedAt `gorm:"column:masjid_teacher_deleted_at;index" json:"masjid_teacher_deleted_at,omitempty"`
}

// TableName override
func (MasjidTeacherModel) TableName() string {
	return "masjid_teachers"
}
