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

	// SLUG (opsional; unik per tenant saat alive — diatur di DB)
	ClassSubjectsSlug *string `json:"class_subjects_slug,omitempty" gorm:"column:class_subjects_slug;size:160"`

	// Metadata kurikulum (opsional)
	ClassSubjectsOrderIndex      *int    `json:"class_subjects_order_index,omitempty"       gorm:"column:class_subjects_order_index"`
	ClassSubjectsHoursPerWeek    *int    `json:"class_subjects_hours_per_week,omitempty"    gorm:"column:class_subjects_hours_per_week"`
	ClassSubjectsMinPassingScore *int    `json:"class_subjects_min_passing_score,omitempty" gorm:"column:class_subjects_min_passing_score"`
	ClassSubjectsWeightOnReport  *int    `json:"class_subjects_weight_on_report,omitempty"  gorm:"column:class_subjects_weight_on_report"`
	ClassSubjectsIsCore          bool    `json:"class_subjects_is_core"                     gorm:"column:class_subjects_is_core;not null;default:false"`
	ClassSubjectsDesc            *string `json:"class_subjects_desc,omitempty"              gorm:"column:class_subjects_desc"`

	// Bobot penilaian (opsional) – SMALLINT di DB; di Go boleh *int16 atau *int
	ClassSubjectsWeightAssignment   *int16 `json:"class_subjects_weight_assignment,omitempty"   gorm:"column:class_subjects_weight_assignment"`
	ClassSubjectsWeightQuiz         *int16 `json:"class_subjects_weight_quiz,omitempty"         gorm:"column:class_subjects_weight_quiz"`
	ClassSubjectsWeightMid          *int16 `json:"class_subjects_weight_mid,omitempty"          gorm:"column:class_subjects_weight_mid"`
	ClassSubjectsWeightFinal        *int16 `json:"class_subjects_weight_final,omitempty"        gorm:"column:class_subjects_weight_final"`
	ClassSubjectsMinAttendancePct   *int16 `json:"class_subjects_min_attendance_percent,omitempty" gorm:"column:class_subjects_min_attendance_percent"`

	// Single image (2-slot + retensi)
	ClassSubjectsImageURL                 *string    `json:"class_subjects_image_url,omitempty"                  gorm:"column:class_subjects_image_url"`
	ClassSubjectsImageObjectKey           *string    `json:"class_subjects_image_object_key,omitempty"           gorm:"column:class_subjects_image_object_key"`
	ClassSubjectsImageURLOld              *string    `json:"class_subjects_image_url_old,omitempty"              gorm:"column:class_subjects_image_url_old"`
	ClassSubjectsImageObjectKeyOld        *string    `json:"class_subjects_image_object_key_old,omitempty"       gorm:"column:class_subjects_image_object_key_old"`
	ClassSubjectsImageDeletePendingUntil  *time.Time `json:"class_subjects_image_delete_pending_until,omitempty" gorm:"column:class_subjects_image_delete_pending_until"`

	// Status & timestamps
	ClassSubjectsIsActive  bool           `json:"class_subjects_is_active"            gorm:"column:class_subjects_is_active;not null;default:true"`
	ClassSubjectsCreatedAt time.Time      `json:"class_subjects_created_at"           gorm:"column:class_subjects_created_at;not null;autoCreateTime"`
	// Kalau mau strict sesuai DB yang NOT NULL, pakai time.Time (bukan *time.Time)
	ClassSubjectsUpdatedAt time.Time      `json:"class_subjects_updated_at"           gorm:"column:class_subjects_updated_at;not null;autoUpdateTime"`
	ClassSubjectsDeletedAt gorm.DeletedAt `json:"class_subjects_deleted_at,omitempty" gorm:"column:class_subjects_deleted_at;index"`
}

func (ClassSubjectModel) TableName() string { return "class_subjects" }
