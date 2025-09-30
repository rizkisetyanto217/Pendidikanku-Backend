// file: internals/features/school/class_subjects/model/class_subject_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectModel struct {
	// ============ PK & tenant ============
	ClassSubjectID       uuid.UUID `gorm:"column:class_subject_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_subject_id"`
	ClassSubjectMasjidID uuid.UUID `gorm:"column:class_subject_masjid_id;type:uuid;not null"                            json:"class_subject_masjid_id"`

	// ============ FK eksplisit ============
	ClassSubjectClassID   uuid.UUID `gorm:"column:class_subject_class_id;type:uuid;not null"   json:"class_subject_class_id"`
	ClassSubjectSubjectID uuid.UUID `gorm:"column:class_subject_subject_id;type:uuid;not null" json:"class_subject_subject_id"`

	// ============ Identitas & atribut ============
	ClassSubjectSlug            *string `gorm:"column:class_subject_slug;type:varchar(160)" json:"class_subject_slug,omitempty"`
	ClassSubjectOrderIndex      *int    `gorm:"column:class_subject_order_index"            json:"class_subject_order_index,omitempty"`
	ClassSubjectHoursPerWeek    *int    `gorm:"column:class_subject_hours_per_week"         json:"class_subject_hours_per_week,omitempty"`
	ClassSubjectMinPassingScore *int    `gorm:"column:class_subject_min_passing_score"      json:"class_subject_min_passing_score,omitempty"`
	ClassSubjectWeightOnReport  *int    `gorm:"column:class_subject_weight_on_report"       json:"class_subject_weight_on_report,omitempty"`
	ClassSubjectIsCore          bool    `gorm:"column:class_subject_is_core;not null;default:false" json:"class_subject_is_core"`
	ClassSubjectDesc            *string `gorm:"column:class_subject_desc;type:text"          json:"class_subject_desc,omitempty"`

	// ============ Bobot penilaian (0..100) ============
	ClassSubjectWeightAssignment     *int `gorm:"column:class_subject_weight_assignment"      json:"class_subject_weight_assignment,omitempty"`
	ClassSubjectWeightQuiz           *int `gorm:"column:class_subject_weight_quiz"            json:"class_subject_weight_quiz,omitempty"`
	ClassSubjectWeightMid            *int `gorm:"column:class_subject_weight_mid"             json:"class_subject_weight_mid,omitempty"`
	ClassSubjectWeightFinal          *int `gorm:"column:class_subject_weight_final"           json:"class_subject_weight_final,omitempty"`
	ClassSubjectMinAttendancePercent *int `gorm:"column:class_subject_min_attendance_percent" json:"class_subject_min_attendance_percent,omitempty"`

	// ============ Snapshots: subjects ============
	ClassSubjectSubjectNameSnapshot *string `gorm:"column:class_subject_subject_name_snapshot;type:varchar(160)" json:"class_subject_subject_name_snapshot,omitempty"`
	ClassSubjectSubjectCodeSnapshot *string `gorm:"column:class_subject_subject_code_snapshot;type:varchar(80)"  json:"class_subject_subject_code_snapshot,omitempty"`
	ClassSubjectSubjectSlugSnapshot *string `gorm:"column:class_subject_subject_slug_snapshot;type:varchar(160)" json:"class_subject_subject_slug_snapshot,omitempty"`
	ClassSubjectSubjectURLSnapshot  *string `gorm:"column:class_subject_subject_url_snapshot;type:text"          json:"class_subject_subject_url_snapshot,omitempty"`

	// ============ Snapshots: class_parent (prefix parent) ============
	ClassSubjectParentCodeSnapshot  *string `gorm:"column:class_subject_parent_code_snapshot;type:varchar(80)"   json:"class_subject_parent_code_snapshot,omitempty"`
	ClassSubjectParentSlugSnapshot  *string `gorm:"column:class_subject_parent_slug_snapshot;type:varchar(160)"  json:"class_subject_parent_slug_snapshot,omitempty"`
	ClassSubjectParentLevelSnapshot *int16  `gorm:"column:class_subject_parent_level_snapshot"                   json:"class_subject_parent_level_snapshot,omitempty"`
	ClassSubjectParentURLSnapshot   *string `gorm:"column:class_subject_parent_url_snapshot;type:text"           json:"class_subject_parent_url_snapshot,omitempty"`
	ClassSubjectParentNameSnapshot  *string `gorm:"column:class_subject_parent_name_snapshot;type:varchar(160)"  json:"class_subject_parent_name_snapshot,omitempty"`

	// ============ Snapshots: classes (prefix class_) ============
	ClassSubjectClassSlugSnapshot         *string `gorm:"column:class_subject_class_slug_snapshot;type:varchar(160)"        json:"class_subject_class_slug_snapshot,omitempty"`
	ClassSubjectClassImageURLSnapshot     *string `gorm:"column:class_subject_class_image_url_snapshot;type:text"           json:"class_subject_class_image_url_snapshot,omitempty"`
	ClassSubjectClassStatusSnapshot       *string `gorm:"column:class_subject_class_status_snapshot;type:class_status_enum" json:"class_subject_class_status_snapshot,omitempty"`
	ClassSubjectClassDeliveryModeSnapshot *string `gorm:"column:class_subject_class_delivery_mode_snapshot;type:class_delivery_mode_enum" json:"class_subject_class_delivery_mode_snapshot,omitempty"`

	// ============ Status & audit ============
	ClassSubjectIsActive  bool           `gorm:"column:class_subject_is_active;not null;default:true"               json:"class_subject_is_active"`
	ClassSubjectCreatedAt time.Time      `gorm:"column:class_subject_created_at;type:timestamptz;not null;default:now()" json:"class_subject_created_at"`
	ClassSubjectUpdatedAt time.Time      `gorm:"column:class_subject_updated_at;type:timestamptz;not null;default:now()" json:"class_subject_updated_at"`
	ClassSubjectDeletedAt gorm.DeletedAt `gorm:"column:class_subject_deleted_at;index"                              json:"class_subject_deleted_at,omitempty"`
}

func (ClassSubjectModel) TableName() string { return "class_subjects" }
