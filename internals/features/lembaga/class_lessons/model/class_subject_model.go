// internals/features/lembaga/class_subjects/model/class_subject_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type ClassSubjectModel struct {
	// PK
	ClassSubjectsID uuid.UUID `json:"class_subjects_id" gorm:"column:class_subjects_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// FKs
	ClassSubjectsMasjidID  uuid.UUID `json:"class_subjects_masjid_id"  gorm:"column:class_subjects_masjid_id;type:uuid;not null;index:idx_cs_masjid_active,where:class_subjects_is_active = TRUE;index:idx_cs_masjid_class_year_active,priority:1;index:idx_cs_masjid_subject_year_active,priority:1"`
	ClassSubjectsClassID   uuid.UUID `json:"class_subjects_class_id"   gorm:"column:class_subjects_class_id;type:uuid;not null;index:idx_cs_masjid_class_year_active,priority:2;index:idx_cs_class_order,priority:1,where:class_subjects_is_active = TRUE"`
	ClassSubjectsSubjectID uuid.UUID `json:"class_subjects_subject_id" gorm:"column:class_subjects_subject_id;type:uuid;not null;index:idx_cs_masjid_subject_year_active,priority:2"`

	// Metadata kurikulum (opsional)
	ClassSubjectsOrderIndex      *int    `json:"class_subjects_order_index,omitempty"      gorm:"column:class_subjects_order_index;check:(class_subjects_order_index IS NULL OR class_subjects_order_index >= 0)"`
	ClassSubjectsHoursPerWeek    *int    `json:"class_subjects_hours_per_week,omitempty"   gorm:"column:class_subjects_hours_per_week;check:(class_subjects_hours_per_week IS NULL OR class_subjects_hours_per_week >= 0)"`
	ClassSubjectsMinPassingScore *int    `json:"class_subjects_min_passing_score,omitempty" gorm:"column:class_subjects_min_passing_score;check:(class_subjects_min_passing_score IS NULL OR (class_subjects_min_passing_score BETWEEN 0 AND 100))"`
	ClassSubjectsWeightOnReport  *int    `json:"class_subjects_weight_on_report,omitempty" gorm:"column:class_subjects_weight_on_report;check:(class_subjects_weight_on_report IS NULL OR class_subjects_weight_on_report >= 0)"`
	ClassSubjectsIsCore          bool    `json:"class_subjects_is_core"                    gorm:"column:class_subjects_is_core;not null;default:false"`
	ClassSubjectsAcademicYear    *string `json:"class_subjects_academic_year,omitempty"    gorm:"column:class_subjects_academic_year;type:text;index:idx_cs_masjid_class_year_active,priority:3;index:idx_cs_masjid_subject_year_active,priority:3"`
	ClassSubjectsDesc            *string `json:"class_subjects_desc,omitempty"             gorm:"column:class_subjects_desc"`

	// Status & timestamps
	ClassSubjectsIsActive  bool       `json:"class_subjects_is_active"            gorm:"column:class_subjects_is_active;not null;default:true;index:idx_cs_masjid_active,where:class_subjects_is_active = TRUE;index:idx_cs_class_order,priority:2,where:class_subjects_is_active = TRUE"`
	ClassSubjectsCreatedAt time.Time  `json:"class_subjects_created_at"           gorm:"column:class_subjects_created_at;not null;default:CURRENT_TIMESTAMP"`
	ClassSubjectsUpdatedAt *time.Time `json:"class_subjects_updated_at,omitempty" gorm:"column:class_subjects_updated_at"`
	ClassSubjectsDeletedAt *time.Time `json:"class_subjects_deleted_at,omitempty" gorm:"column:class_subjects_deleted_at;index"`
}

func (ClassSubjectModel) TableName() string { return "class_subjects" }
