// file: internals/features/school/class_subjects/model/class_subject_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectModel struct {
	// PK & tenant
	ClassSubjectID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_subject_id"      json:"class_subject_id"`
	ClassSubjectMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_subject_masjid_id"                           json:"class_subject_masjid_id"`

	// FK eksplisit (tenant-safe via constraint komposit)
	ClassSubjectClassID   uuid.UUID `gorm:"type:uuid;not null;column:class_subject_class_id"   json:"class_subject_class_id"`
	ClassSubjectSubjectID uuid.UUID `gorm:"type:uuid;not null;column:class_subject_subject_id" json:"class_subject_subject_id"`

	// Identitas & atribut
	ClassSubjectSlug             *string `gorm:"type:varchar(160);column:class_subject_slug"              json:"class_subject_slug,omitempty"`
	ClassSubjectOrderIndex       *int    `gorm:"column:class_subject_order_index"                         json:"class_subject_order_index,omitempty"`
	ClassSubjectHoursPerWeek     *int    `gorm:"column:class_subject_hours_per_week"                      json:"class_subject_hours_per_week,omitempty"`
	ClassSubjectMinPassingScore  *int    `gorm:"column:class_subject_min_passing_score"                   json:"class_subject_min_passing_score,omitempty"`
	ClassSubjectWeightOnReport   *int    `gorm:"column:class_subject_weight_on_report"                    json:"class_subject_weight_on_report,omitempty"`
	ClassSubjectIsCore           bool    `gorm:"not null;default:false;column:class_subject_is_core"      json:"class_subject_is_core"`
	ClassSubjectDesc             *string `gorm:"type:text;column:class_subject_desc"                      json:"class_subject_desc,omitempty"`

	// Bobot penilaian (0..100)
	ClassSubjectWeightAssignment      *int `gorm:"column:class_subject_weight_assignment"       json:"class_subject_weight_assignment,omitempty"`
	ClassSubjectWeightQuiz            *int `gorm:"column:class_subject_weight_quiz"             json:"class_subject_weight_quiz,omitempty"`
	ClassSubjectWeightMid             *int `gorm:"column:class_subject_weight_mid"              json:"class_subject_weight_mid,omitempty"`
	ClassSubjectWeightFinal           *int `gorm:"column:class_subject_weight_final"            json:"class_subject_weight_final,omitempty"`
	ClassSubjectMinAttendancePercent  *int `gorm:"column:class_subject_min_attendance_percent"  json:"class_subject_min_attendance_percent,omitempty"`

	// Status & audit
	ClassSubjectIsActive  bool           `gorm:"not null;default:true;column:class_subject_is_active"  json:"class_subject_is_active"`
	ClassSubjectCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_subject_created_at" json:"class_subject_created_at"`
	ClassSubjectUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_subject_updated_at" json:"class_subject_updated_at"`
	ClassSubjectDeletedAt gorm.DeletedAt `gorm:"column:class_subject_deleted_at;index"                 json:"class_subject_deleted_at,omitempty"`
}

func (ClassSubjectModel) TableName() string { return "class_subjects" }
