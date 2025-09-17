// file: internals/features/school/enrolments/user_classes/model/user_class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	UserClassesStatusActive    = "active"
	UserClassesStatusInactive  = "inactive"
	UserClassesStatusCompleted = "completed"
)

type UserClassesModel struct {
	// PK
	UserClassesID uuid.UUID `json:"user_classes_id" gorm:"type:uuid;primaryKey;column:user_classes_id"`

	// Identitas siswa (tenant)
	UserClassesMasjidStudentID uuid.UUID `json:"user_classes_masjid_student_id" gorm:"type:uuid;not null;column:user_classes_masjid_student_id"`

	// Kelas & Tenant
	UserClassesClassID  uuid.UUID `json:"user_classes_class_id"  gorm:"type:uuid;not null;column:user_classes_class_id"`
	UserClassesMasjidID uuid.UUID `json:"user_classes_masjid_id" gorm:"type:uuid;not null;column:user_classes_masjid_id"`

	// Lifecycle enrolment
	UserClassesStatus string  `json:"user_classes_status" gorm:"type:text;not null;default:'active';column:user_classes_status"` // 'active'|'inactive'|'completed'
	UserClassesResult *string `json:"user_classes_result,omitempty" gorm:"type:text;column:user_classes_result"`                  // 'passed'|'failed'|NULL

	// Billing ringan
	UserClassesRegisterPaidAt *time.Time `json:"user_classes_register_paid_at,omitempty" gorm:"type:timestamptz;column:user_classes_register_paid_at"`
	UserClassesPaidUntil      *time.Time `json:"user_classes_paid_until,omitempty"       gorm:"type:timestamptz;column:user_classes_paid_until"`
	UserClassesPaidGraceDays  int16      `json:"user_classes_paid_grace_days"            gorm:"type:smallint;not null;default:0;column:user_classes_paid_grace_days"`

	// Jejak waktu enrolment
	UserClassesJoinedAt    *time.Time `json:"user_classes_joined_at,omitempty"    gorm:"type:timestamptz;column:user_classes_joined_at"`
	UserClassesLeftAt      *time.Time `json:"user_classes_left_at,omitempty"      gorm:"type:timestamptz;column:user_classes_left_at"`
	UserClassesCompletedAt *time.Time `json:"user_classes_completed_at,omitempty" gorm:"type:timestamptz;column:user_classes_completed_at"`

	// Audit
	UserClassesCreatedAt time.Time  `json:"user_classes_created_at"           gorm:"type:timestamptz;not null;default:now();column:user_classes_created_at"`
	UserClassesUpdatedAt time.Time  `json:"user_classes_updated_at"           gorm:"type:timestamptz;not null;default:now();column:user_classes_updated_at"`
	UserClassesDeletedAt gorm.DeletedAt `json:"user_classes_deleted_at,omitempty" gorm:"type:timestamptz;column:user_classes_deleted_at"`
}

func (UserClassesModel) TableName() string { return "user_classes" }

// Hooks sederhana
func (m *UserClassesModel) BeforeCreate(tx *gorm.DB) error {
	if m.UserClassesID == uuid.Nil {
		m.UserClassesID = uuid.New()
	}
	now := time.Now()
	if m.UserClassesCreatedAt.IsZero() {
		m.UserClassesCreatedAt = now
	}
	if m.UserClassesUpdatedAt.IsZero() {
		m.UserClassesUpdatedAt = now
	}
	return nil
}

func (m *UserClassesModel) BeforeUpdate(tx *gorm.DB) error {
	m.UserClassesUpdatedAt = time.Now()
	return nil
}