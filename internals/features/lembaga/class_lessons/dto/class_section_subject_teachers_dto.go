// internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/lembaga/class_lessons/model"

	"github.com/google/uuid"
)

// ===========================
// Create Request (tanpa masjid_id)
// ===========================
type CreateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherModelSectionID     uuid.UUID `json:"class_section_subject_teachers_section_id" validate:"required"`
	ClassSectionSubjectTeacherModelSubjectID     uuid.UUID `json:"class_section_subject_teachers_subject_id" validate:"required"`
	ClassSectionSubjectTeacherModelTeacherUserID uuid.UUID `json:"class_section_subject_teachers_teacher_user_id" validate:"required"`
	ClassSectionSubjectTeacherModelIsActive      *bool     `json:"class_section_subject_teachers_is_active"`
}

// ===========================
// Update Request (partial)
// ===========================
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherModelSectionID     *uuid.UUID `json:"class_section_subject_teachers_section_id"`
	ClassSectionSubjectTeacherModelSubjectID     *uuid.UUID `json:"class_section_subject_teachers_subject_id"`
	ClassSectionSubjectTeacherModelTeacherUserID *uuid.UUID `json:"class_section_subject_teachers_teacher_user_id"`
	ClassSectionSubjectTeacherModelIsActive      *bool      `json:"class_section_subject_teachers_is_active"`
}


// ===========================
// Response DTO
// ===========================
type ClassSectionSubjectTeacherResponse struct {
	ID         uuid.UUID  `json:"id"`
	MasjidID   uuid.UUID  `json:"masjid_id"`
	SectionID  uuid.UUID  `json:"section_id"`
	SubjectID  uuid.UUID  `json:"subject_id"`
	TeacherUID uuid.UUID  `json:"teacher_user_id"`
	IsActive   bool       `json:"is_active"`

	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt *time.Time  `json:"updated_at,omitempty"`
	DeletedAt *time.Time  `json:"deleted_at,omitempty"`
}

// ===========================
// Mapper
// ===========================
func FromClassSectionSubjectTeacherModel(m model.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAtPtr *time.Time
	if m.ClassSectionSubjectTeacherModelDeletedAt.Valid {
		t := m.ClassSectionSubjectTeacherModelDeletedAt.Time
		deletedAtPtr = &t
	}

	return ClassSectionSubjectTeacherResponse{
		ID:         m.ClassSectionSubjectTeachersID,
		MasjidID:   m.ClassSectionSubjectTeacherModelMasjidID,
		SectionID:  m.ClassSectionSubjectTeacherModelSectionID,
		SubjectID:  m.ClassSectionSubjectTeacherModelSubjectID,
		TeacherUID: m.ClassSectionSubjectTeacherModelTeacherUserID,
		IsActive:   m.ClassSectionSubjectTeacherModelIsActive,
		CreatedAt:  m.ClassSectionSubjectTeacherModelCreatedAt,
		UpdatedAt:  m.ClassSectionSubjectTeacherModelUpdatedAt,
		DeletedAt:  deletedAtPtr,
	}
}

func FromClassSectionSubjectTeacherModels(rows []model.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromClassSectionSubjectTeacherModel(r))
	}
	return out
}
