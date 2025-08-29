// internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"time"

	csstModel "masjidku_backend/internals/features/school/class_subject_books/subject/model"

	"github.com/google/uuid"
)

/* =========================================================
   1) REQUEST DTO — key JSON disamakan dgn model
   ========================================================= */

// Create
// Catatan: masjid_id bisa dikosongkan di request jika diisi dari token di controller.
type CreateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeachersMasjidID      *uuid.UUID `json:"class_section_subject_teachers_masjid_id" validate:"omitempty"`
	ClassSectionSubjectTeachersSectionID     uuid.UUID  `json:"class_section_subject_teachers_section_id" validate:"required"`
	ClassSectionSubjectTeachersSubjectID     uuid.UUID  `json:"class_section_subject_teachers_subject_id" validate:"required"`
	ClassSectionSubjectTeachersTeacherUserID uuid.UUID  `json:"class_section_subject_teachers_teacher_user_id" validate:"required"`
	ClassSectionSubjectTeachersIsActive      *bool      `json:"class_section_subject_teachers_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeachersMasjidID      *uuid.UUID `json:"class_section_subject_teachers_masjid_id" validate:"omitempty"`
	ClassSectionSubjectTeachersSectionID     *uuid.UUID `json:"class_section_subject_teachers_section_id" validate:"omitempty"`
	ClassSectionSubjectTeachersSubjectID     *uuid.UUID `json:"class_section_subject_teachers_subject_id" validate:"omitempty"`
	ClassSectionSubjectTeachersTeacherUserID *uuid.UUID `json:"class_section_subject_teachers_teacher_user_id" validate:"omitempty"`
	ClassSectionSubjectTeachersIsActive      *bool      `json:"class_section_subject_teachers_is_active" validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO — full snake_case seperti di model
   ========================================================= */

type ClassSectionSubjectTeacherResponse struct {
	ClassSectionSubjectTeachersID             uuid.UUID  `json:"class_section_subject_teachers_id"`
	ClassSectionSubjectTeachersMasjidID       uuid.UUID  `json:"class_section_subject_teachers_masjid_id"`
	ClassSectionSubjectTeachersSectionID      uuid.UUID  `json:"class_section_subject_teachers_section_id"`
	ClassSectionSubjectTeachersSubjectID      uuid.UUID  `json:"class_section_subject_teachers_subject_id"`
	ClassSectionSubjectTeachersTeacherUserID  uuid.UUID  `json:"class_section_subject_teachers_teacher_user_id"`
	ClassSectionSubjectTeachersIsActive       bool       `json:"class_section_subject_teachers_is_active"`
	ClassSectionSubjectTeachersCreatedAt      time.Time  `json:"class_section_subject_teachers_created_at"`
	ClassSectionSubjectTeachersUpdatedAt      *time.Time `json:"class_section_subject_teachers_updated_at,omitempty"`
	ClassSectionSubjectTeachersDeletedAt      *time.Time `json:"class_section_subject_teachers_deleted_at,omitempty"`
}

/* =========================================================
   3) MAPPERS — 1:1 ke field model
   ========================================================= */

func (r CreateClassSectionSubjectTeacherRequest) ToModel() csstModel.ClassSectionSubjectTeacherModel {
	m := csstModel.ClassSectionSubjectTeacherModel{
		ClassSectionSubjectTeacherModelSectionID:     r.ClassSectionSubjectTeachersSectionID,
		ClassSectionSubjectTeacherModelSubjectID:     r.ClassSectionSubjectTeachersSubjectID,
		ClassSectionSubjectTeacherModelTeacherUserID: r.ClassSectionSubjectTeachersTeacherUserID,
	}
	// optional from request
	if r.ClassSectionSubjectTeachersMasjidID != nil {
		m.ClassSectionSubjectTeacherModelMasjidID = *r.ClassSectionSubjectTeachersMasjidID
	}
	if r.ClassSectionSubjectTeachersIsActive != nil {
		m.ClassSectionSubjectTeacherModelIsActive = *r.ClassSectionSubjectTeachersIsActive
	}
	return m
}

func (r UpdateClassSectionSubjectTeacherRequest) Apply(m *csstModel.ClassSectionSubjectTeacherModel) {
	if r.ClassSectionSubjectTeachersMasjidID != nil {
		m.ClassSectionSubjectTeacherModelMasjidID = *r.ClassSectionSubjectTeachersMasjidID
	}
	if r.ClassSectionSubjectTeachersSectionID != nil {
		m.ClassSectionSubjectTeacherModelSectionID = *r.ClassSectionSubjectTeachersSectionID
	}
	if r.ClassSectionSubjectTeachersSubjectID != nil {
		m.ClassSectionSubjectTeacherModelSubjectID = *r.ClassSectionSubjectTeachersSubjectID
	}
	if r.ClassSectionSubjectTeachersTeacherUserID != nil {
		m.ClassSectionSubjectTeacherModelTeacherUserID = *r.ClassSectionSubjectTeachersTeacherUserID
	}
	if r.ClassSectionSubjectTeachersIsActive != nil {
		m.ClassSectionSubjectTeacherModelIsActive = *r.ClassSectionSubjectTeachersIsActive
	}
	// updated_at biarkan diisi auto oleh GORM/trigger
}

func FromClassSectionSubjectTeacherModel(m csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if m.ClassSectionSubjectTeacherModelDeletedAt.Valid {
		t := m.ClassSectionSubjectTeacherModelDeletedAt.Time
		deletedAt = &t
	}
	return ClassSectionSubjectTeacherResponse{
		ClassSectionSubjectTeachersID:            m.ClassSectionSubjectTeachersID,
		ClassSectionSubjectTeachersMasjidID:      m.ClassSectionSubjectTeacherModelMasjidID,
		ClassSectionSubjectTeachersSectionID:     m.ClassSectionSubjectTeacherModelSectionID,
		ClassSectionSubjectTeachersSubjectID:     m.ClassSectionSubjectTeacherModelSubjectID,
		ClassSectionSubjectTeachersTeacherUserID: m.ClassSectionSubjectTeacherModelTeacherUserID,
		ClassSectionSubjectTeachersIsActive:      m.ClassSectionSubjectTeacherModelIsActive,
		ClassSectionSubjectTeachersCreatedAt:     m.ClassSectionSubjectTeacherModelCreatedAt,
		ClassSectionSubjectTeachersUpdatedAt:     m.ClassSectionSubjectTeacherModelUpdatedAt,
		ClassSectionSubjectTeachersDeletedAt:     deletedAt,
	}
}

func FromClassSectionSubjectTeacherModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromClassSectionSubjectTeacherModel(r))
	}
	return out
}
