// file: internals/features/school/classes/class_materials/model/student_class_material_progress_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =======================================================
   ENUM: material_progress_status_enum (Go-side)
======================================================= */

type MaterialProgressStatus string

const (
	MaterialProgressStatusNotStarted MaterialProgressStatus = "not_started"
	MaterialProgressStatusInProgress MaterialProgressStatus = "in_progress"
	MaterialProgressStatusCompleted  MaterialProgressStatus = "completed"
	MaterialProgressStatusSkipped    MaterialProgressStatus = "skipped"
)

/* =======================================================
   MODEL: student_class_material_progresses
======================================================= */

type StudentClassMaterialProgressModel struct {
	StudentClassMaterialProgressID uuid.UUID `json:"student_class_material_progress_id" gorm:"column:student_class_material_progress_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// scope / tenant
	StudentClassMaterialProgressSchoolID uuid.UUID `json:"student_class_material_progress_school_id" gorm:"column:student_class_material_progress_school_id;type:uuid;not null"`

	// murid & enrollment (SCSST)
	StudentClassMaterialProgressStudentID uuid.UUID `json:"student_class_material_progress_student_id" gorm:"column:student_class_material_progress_student_id;type:uuid;not null"`
	StudentClassMaterialProgressSCSSTID   uuid.UUID `json:"student_class_material_progress_scsst_id" gorm:"column:student_class_material_progress_scsst_id;type:uuid;not null"`

	// materi yang dilacak
	StudentClassMaterialProgressClassMaterialID uuid.UUID `json:"student_class_material_progress_class_material_id" gorm:"column:student_class_material_progress_class_material_id;type:uuid;not null"`

	// status & progress (umum untuk article, video, pdf, dll.)
	StudentClassMaterialProgressStatus       MaterialProgressStatus `json:"student_class_material_progress_status" gorm:"column:student_class_material_progress_status;type:material_progress_status_enum;not null;default:'in_progress'"`
	StudentClassMaterialProgressLastPercent  int32                  `json:"student_class_material_progress_last_percent" gorm:"column:student_class_material_progress_last_percent;not null;default:0"`
	StudentClassMaterialProgressIsCompleted  bool                   `json:"student_class_material_progress_is_completed" gorm:"column:student_class_material_progress_is_completed;not null;default:false"`
	StudentClassMaterialProgressViewDuration *int64                 `json:"student_class_material_progress_view_duration_sec" gorm:"column:student_class_material_progress_view_duration_sec"`
	StudentClassMaterialProgressOpenCount    int32                  `json:"student_class_material_progress_open_count" gorm:"column:student_class_material_progress_open_count;not null;default:0"`

	// timestamps aktivitas
	StudentClassMaterialProgressFirstStartedAt *time.Time `json:"student_class_material_progress_first_started_at" gorm:"column:student_class_material_progress_first_started_at"`
	StudentClassMaterialProgressLastActivityAt *time.Time `json:"student_class_material_progress_last_activity_at" gorm:"column:student_class_material_progress_last_activity_at"`
	StudentClassMaterialProgressCompletedAt    *time.Time `json:"student_class_material_progress_completed_at" gorm:"column:student_class_material_progress_completed_at"`

	// fleksibel: posisi terakhir video / scroll / halaman / dsb (per-type)
	StudentClassMaterialProgressExtra datatypes.JSON `json:"student_class_material_progress_extra" gorm:"column:student_class_material_progress_extra;type:jsonb"`

	// timestamps row
	StudentClassMaterialProgressCreatedAt time.Time `json:"student_class_material_progress_created_at" gorm:"column:student_class_material_progress_created_at;not null;default:now()"`
	StudentClassMaterialProgressUpdatedAt time.Time `json:"student_class_material_progress_updated_at" gorm:"column:student_class_material_progress_updated_at;not null;default:now()"`
}

func (StudentClassMaterialProgressModel) TableName() string {
	return "student_class_material_progresses"
}
