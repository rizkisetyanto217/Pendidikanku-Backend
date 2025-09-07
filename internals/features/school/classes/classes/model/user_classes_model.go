// file: internals/features/lembaga/classes/user_classes/main/model/user_class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	UserClassStatusActive   = "active"
	UserClassStatusInactive = "inactive"
	UserClassStatusEnded    = "ended"
)

type UserClassesModel struct {
	// PK
	UserClassesID uuid.UUID `json:"user_classes_id" gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_classes_id"`

	// FK: users(id)
	UserClassesUserID uuid.UUID `json:"user_classes_user_id" gorm:"type:uuid;not null;column:user_classes_user_id"`

	// FK komposit ke classes (class_id, class_masjid_id) — constraint di DB
	UserClassesClassID  uuid.UUID `json:"user_classes_class_id" gorm:"type:uuid;not null;column:user_classes_class_id"`
	UserClassesMasjidID uuid.UUID `json:"user_classes_masjid_id" gorm:"type:uuid;not null;column:user_classes_masjid_id"`

	// FK komposit ke academic_terms (academic_terms_id, academic_terms_masjid_id) — constraint di DB
	UserClassesTermID uuid.UUID `json:"user_classes_term_id" gorm:"type:uuid;not null;column:user_classes_term_id"`

	// Relasi opsional ke masjid_students
	UserClassesMasjidStudentID *uuid.UUID `json:"user_classes_masjid_student_id,omitempty" gorm:"type:uuid;column:user_classes_masjid_student_id"`

	// Status enrolment (dibatasi CHECK di DB)
	UserClassesStatus string `json:"user_classes_status" gorm:"type:text;not null;default:'active';column:user_classes_status"`

	// Jejak waktu enrolment per kelas
	UserClassesJoinedAt *time.Time `json:"user_classes_joined_at,omitempty" gorm:"column:user_classes_joined_at"`
	UserClassesLeftAt   *time.Time `json:"user_classes_left_at,omitempty" gorm:"column:user_classes_left_at"`

	// Timestamps
	UserClassesCreatedAt time.Time      `json:"user_classes_created_at" gorm:"column:user_classes_created_at;autoCreateTime"`
	UserClassesUpdatedAt time.Time      `json:"user_classes_updated_at" gorm:"column:user_classes_updated_at;autoUpdateTime"`
	UserClassesDeletedAt gorm.DeletedAt `json:"user_classes_deleted_at,omitempty" gorm:"column:user_classes_deleted_at;index"`
}

func (UserClassesModel) TableName() string {
	return "user_classes"
}
