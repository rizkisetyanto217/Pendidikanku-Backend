// internals/features/lembaga/class_subjects/model/class_subject_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectModel struct {
	// PK
	ClassSubjectsID uuid.UUID `json:"class_subjects_id" gorm:"column:class_subjects_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// FKs (NOT NULL sesuai tabel)
	ClassSubjectsMasjidID  uuid.UUID `json:"class_subjects_masjid_id"  gorm:"column:class_subjects_masjid_id;type:uuid;not null"`
	ClassSubjectsClassID   uuid.UUID `json:"class_subjects_class_id"   gorm:"column:class_subjects_class_id;type:uuid;not null"`
	ClassSubjectsSubjectID uuid.UUID `json:"class_subjects_subject_id" gorm:"column:class_subjects_subject_id;type:uuid;not null"`

	// (Baru) relasi opsional ke academic_terms
	ClassSubjectsTermID *uuid.UUID `json:"class_subjects_term_id,omitempty" gorm:"column:class_subjects_term_id;type:uuid"`

	// Metadata kurikulum (opsional)
	ClassSubjectsOrderIndex      *int    `json:"class_subjects_order_index,omitempty"       gorm:"column:class_subjects_order_index"`
	ClassSubjectsHoursPerWeek    *int    `json:"class_subjects_hours_per_week,omitempty"    gorm:"column:class_subjects_hours_per_week"`
	ClassSubjectsMinPassingScore *int    `json:"class_subjects_min_passing_score,omitempty" gorm:"column:class_subjects_min_passing_score"`
	ClassSubjectsWeightOnReport  *int    `json:"class_subjects_weight_on_report,omitempty"  gorm:"column:class_subjects_weight_on_report"`
	ClassSubjectsIsCore          bool    `json:"class_subjects_is_core"                     gorm:"column:class_subjects_is_core;not null;default:false"`
	ClassSubjectsDesc            *string `json:"class_subjects_desc,omitempty"              gorm:"column:class_subjects_desc"`

	// Status & timestamps
	ClassSubjectsIsActive  bool          `json:"class_subjects_is_active"            gorm:"column:class_subjects_is_active;not null;default:true"`
	ClassSubjectsCreatedAt time.Time     `json:"class_subjects_created_at"           gorm:"column:class_subjects_created_at;not null;autoCreateTime"`
	ClassSubjectsUpdatedAt *time.Time    `json:"class_subjects_updated_at,omitempty" gorm:"column:class_subjects_updated_at;autoUpdateTime"`
	ClassSubjectsDeletedAt gorm.DeletedAt `json:"class_subjects_deleted_at,omitempty" gorm:"column:class_subjects_deleted_at;index"`
}

func (ClassSubjectModel) TableName() string { return "class_subjects" }
