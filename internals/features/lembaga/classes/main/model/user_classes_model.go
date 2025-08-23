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

	// FK komposit ke classes (class_id, class_masjid_id) — ditopang di DB
	UserClassesClassID  uuid.UUID `json:"user_classes_class_id" gorm:"type:uuid;not null;column:user_classes_class_id"`
	UserClassesMasjidID uuid.UUID `json:"user_classes_masjid_id" gorm:"type:uuid;not null;column:user_classes_masjid_id"`

	// FK komposit ke academic_terms (academic_terms_id, academic_terms_masjid_id) — ditopang di DB
	UserClassesTermID uuid.UUID `json:"user_classes_term_id" gorm:"type:uuid;not null;column:user_classes_term_id"`

	// (Opsional) jejak opening untuk trace kuota/harga per term
	UserClassesOpeningID *uuid.UUID `json:"user_classes_opening_id,omitempty" gorm:"type:uuid;column:user_classes_opening_id"`

	// Status enrolment (dibatasi oleh CHECK di DB)
	UserClassesStatus string `json:"user_classes_status" gorm:"type:text;not null;default:'active';column:user_classes_status"`

	// Snapshot biaya per siswa (NULL = ikut default/override lain)
	UserClassesFeeOverrideMonthlyIDR *int    `json:"user_classes_fee_override_monthly_idr,omitempty" gorm:"column:user_classes_fee_override_monthly_idr"`
	UserClassesNotes                 *string `json:"user_classes_notes,omitempty" gorm:"column:user_classes_notes"`

	// Timestamps
	UserClassesCreatedAt time.Time       `json:"user_classes_created_at" gorm:"column:user_classes_created_at;autoCreateTime"`
	UserClassesUpdatedAt *time.Time      `json:"user_classes_updated_at,omitempty" gorm:"column:user_classes_updated_at;autoUpdateTime"`
	UserClassesDeletedAt gorm.DeletedAt  `json:"user_classes_deleted_at,omitempty" gorm:"column:user_classes_deleted_at;index"`
}

func (UserClassesModel) TableName() string {
	return "user_classes"
}
