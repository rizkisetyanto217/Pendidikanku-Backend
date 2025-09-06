package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidStudentModel struct {
	// PK
	MasjidStudentID uuid.UUID `gorm:"column:masjid_student_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_student_id"`

	// FK
	MasjidStudentMasjidID uuid.UUID `gorm:"column:masjid_student_masjid_id;type:uuid;not null;index" json:"masjid_student_masjid_id"`
	MasjidStudentUserID   uuid.UUID `gorm:"column:masjid_student_user_id;type:uuid;not null;index" json:"masjid_student_user_id"`

	// Optional fields
	MasjidStudentCode   *string `gorm:"column:masjid_student_code;type:varchar(50)" json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `gorm:"column:masjid_student_status;type:varchar(20);not null;default:active" json:"masjid_student_status"`
	MasjidStudentNote   *string `gorm:"column:masjid_student_note" json:"masjid_student_note,omitempty"`

	// timestamps
	MasjidStudentCreatedAt time.Time      `gorm:"column:masjid_student_created_at;autoCreateTime" json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time      `gorm:"column:masjid_student_updated_at;autoUpdateTime" json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt gorm.DeletedAt `gorm:"column:masjid_student_deleted_at;index" json:"masjid_student_deleted_at,omitempty"`
}

// TableName override
func (MasjidStudentModel) TableName() string {
	return "masjid_students"
}

// (Opsional) enum-like helpers
const (
	MasjidStudentStatusActive   = "active"
	MasjidStudentStatusInactive = "inactive"
	MasjidStudentStatusAlumni   = "alumni"
)
