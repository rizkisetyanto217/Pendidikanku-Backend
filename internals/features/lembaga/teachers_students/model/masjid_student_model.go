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

	// Unik global (sesuai SQL: NOT NULL UNIQUE)
	MasjidStudentSlug string `gorm:"column:masjid_student_slug;type:varchar(50);not null;uniqueIndex" json:"masjid_student_slug"`

	// Optional fields
	MasjidStudentCode   *string `gorm:"column:masjid_student_code;type:varchar(50)" json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `gorm:"column:masjid_student_status;type:text;not null;default:active" json:"masjid_student_status"` // SQL: TEXT + CHECK ('active','inactive','alumni')
	MasjidStudentNote   *string `gorm:"column:masjid_student_note;type:text" json:"masjid_student_note,omitempty"`

	// Operasional (waktu)
	MasjidStudentJoinedAt *time.Time `gorm:"column:masjid_student_joined_at" json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `gorm:"column:masjid_student_left_at" json:"masjid_student_left_at,omitempty"`

	// timestamps
	MasjidStudentCreatedAt time.Time      `gorm:"column:masjid_student_created_at;autoCreateTime" json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time      `gorm:"column:masjid_student_updated_at;autoUpdateTime" json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt gorm.DeletedAt `gorm:"column:masjid_student_deleted_at;index" json:"masjid_student_deleted_at,omitempty"`
}

func (MasjidStudentModel) TableName() string { return "masjid_students" }

// Enum-like helpers (sesuaikan dengan CHECK constraint di SQL)
const (
	MasjidStudentStatusActive   = "active"
	MasjidStudentStatusInactive = "inactive"
	MasjidStudentStatusAlumni   = "alumni"
)
