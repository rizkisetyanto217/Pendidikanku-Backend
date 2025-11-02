// ===================================
// model/lecture_sessions_material.go
// ===================================

package model

import (
	"time"

	lsmodel "schoolku_backend/internals/features/schools/lecture_sessions/main/model"

	"gorm.io/gorm"
)

type LectureSessionsMaterialModel struct {
	LectureSessionsMaterialID               string  `gorm:"column:lecture_sessions_material_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"lecture_sessions_material_id"`
	LectureSessionsMaterialSummary          *string `gorm:"column:lecture_sessions_material_summary;type:text" json:"lecture_sessions_material_summary,omitempty"`
	LectureSessionsMaterialTranscriptFull   *string `gorm:"column:lecture_sessions_material_transcript_full;type:text" json:"lecture_sessions_material_transcript_full,omitempty"`
	LectureSessionsMaterialLectureSessionID string  `gorm:"column:lecture_sessions_material_lecture_session_id;type:uuid;not null" json:"lecture_sessions_material_lecture_session_id"`
	LectureSessionsMaterialSchoolID         string  `gorm:"column:lecture_sessions_material_school_id;type:uuid;not null" json:"lecture_sessions_material_school_id"`

	LectureSessionsMaterialCreatedAt time.Time      `gorm:"column:lecture_sessions_material_created_at;autoCreateTime" json:"lecture_sessions_material_created_at"`
	LectureSessionsMaterialUpdatedAt time.Time      `gorm:"column:lecture_sessions_material_updated_at;autoUpdateTime" json:"lecture_sessions_material_updated_at"`
	LectureSessionsMaterialDeletedAt gorm.DeletedAt `gorm:"column:lecture_sessions_material_deleted_at;index" json:"-"`

	// Relasi (opsional)
	LectureSession lsmodel.LectureSessionModel `gorm:"foreignKey:LectureSessionsMaterialLectureSessionID;references:LectureSessionID" json:"-"`
}

func (LectureSessionsMaterialModel) TableName() string { return "lecture_sessions_materials" }
