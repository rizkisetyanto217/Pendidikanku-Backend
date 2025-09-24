// file: internals/features/school/enrolments/user_classes/model/user_class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   Constants
========================= */

const (
	UserClassStatusActive    = "active"
	UserClassStatusInactive  = "inactive"
	UserClassStatusCompleted = "completed"

	UserClassResultPassed = "passed"
	UserClassResultFailed = "failed"
)

/* =========================
   Model
========================= */

type UserClassModel struct {
	// PK
	UserClassID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_id" json:"user_class_id"`

	// Identitas siswa pada tenant
	UserClassMasjidStudentID uuid.UUID `gorm:"type:uuid;not null;column:user_class_masjid_student_id" json:"user_class_masjid_student_id"`

	// Kelas & tenant
	UserClassClassID  uuid.UUID `gorm:"type:uuid;not null;column:user_class_class_id" json:"user_class_class_id"`
	UserClassMasjidID uuid.UUID `gorm:"type:uuid;not null;column:user_class_masjid_id" json:"user_class_masjid_id"`

	// Lifecycle enrolment
	UserClassStatus string  `gorm:"type:text;not null;default:'active';column:user_class_status" json:"user_class_status"`
	UserClassResult *string `gorm:"type:text;column:user_class_result,omitempty" json:"user_class_result,omitempty"`

	// Billing ringan
	UserClassRegisterPaidAt *time.Time `gorm:"type:timestamptz;column:user_class_register_paid_at" json:"user_class_register_paid_at,omitempty"`
	UserClassPaidUntil      *time.Time `gorm:"type:timestamptz;column:user_class_paid_until" json:"user_class_paid_until,omitempty"`
	UserClassPaidGraceDays  int16      `gorm:"type:smallint;not null;default:0;column:user_class_paid_grace_days" json:"user_class_paid_grace_days"`

	// Jejak waktu enrolment
	UserClassJoinedAt    *time.Time `gorm:"type:timestamptz;column:user_class_joined_at" json:"user_class_joined_at,omitempty"`
	UserClassLeftAt      *time.Time `gorm:"type:timestamptz;column:user_class_left_at" json:"user_class_left_at,omitempty"`
	UserClassCompletedAt *time.Time `gorm:"type:timestamptz;column:user_class_completed_at" json:"user_class_completed_at,omitempty"`

	// Audit
	UserClassCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_created_at" json:"user_class_created_at"`
	UserClassUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_updated_at" json:"user_class_updated_at"`
	UserClassDeletedAt gorm.DeletedAt `gorm:"column:user_class_deleted_at;index" json:"user_class_deleted_at,omitempty"`
}

func (UserClassModel) TableName() string { return "user_class" }
