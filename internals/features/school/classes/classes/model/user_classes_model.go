package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   Constants (status & result)
   ========================================================= */

const (
	// Lifecycle status
	UserClassStatusActive    = "active"
	UserClassStatusInactive  = "inactive"
	UserClassStatusCompleted = "completed"

	// Outcome (hanya valid saat status = completed)
	UserClassResultPassed = "passed"
	UserClassResultFailed = "failed"
)

/* =========================================================
   Model
   ========================================================= */

type UserClassesModel struct {
	// PK
	UserClassesID uuid.UUID `json:"user_classes_id" gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_classes_id"`

	// FK wajib ke masjid_students (single, NOT NULL)
	UserClassesMasjidStudentID uuid.UUID `json:"user_classes_masjid_student_id" gorm:"type:uuid;not null;column:user_classes_masjid_student_id"`

	// FK komposit ke classes (class_id, class_masjid_id) — constraint di DB
	UserClassesClassID  uuid.UUID `json:"user_classes_class_id" gorm:"type:uuid;not null;column:user_classes_class_id"`
	UserClassesMasjidID uuid.UUID `json:"user_classes_masjid_id" gorm:"type:uuid;not null;column:user_classes_masjid_id"`

	// Lifecycle enrolment (dibatasi CHECK di DB)
	UserClassesStatus string `json:"user_classes_status" gorm:"type:text;not null;default:'active';column:user_classes_status"`

	// Outcome (hasil akhir) — hanya terisi saat status = completed
	// NULL untuk active/inactive
	UserClassesResult *string `json:"user_classes_result,omitempty" gorm:"type:text;column:user_classes_result"`

	// Jejak waktu enrolment per kelas
	UserClassesJoinedAt    *time.Time `json:"user_classes_joined_at,omitempty" gorm:"column:user_classes_joined_at"`
	UserClassesLeftAt      *time.Time `json:"user_classes_left_at,omitempty" gorm:"column:user_classes_left_at"`
	UserClassesCompletedAt *time.Time `json:"user_classes_completed_at,omitempty" gorm:"column:user_classes_completed_at"`

	// Timestamps
	UserClassesCreatedAt time.Time      `json:"user_classes_created_at" gorm:"column:user_classes_created_at;autoCreateTime"`
	UserClassesUpdatedAt time.Time      `json:"user_classes_updated_at" gorm:"column:user_classes_updated_at;autoUpdateTime"`
	UserClassesDeletedAt gorm.DeletedAt `json:"user_classes_deleted_at,omitempty" gorm:"column:user_classes_deleted_at;index"`
}

func (UserClassesModel) TableName() string { return "user_classes" }
