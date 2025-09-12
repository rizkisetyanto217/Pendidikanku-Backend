// internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"time"

	csstModel "masjidku_backend/internals/features/school/subject_books/subject/model"

	"github.com/google/uuid"
)

/* =========================================================
   1) REQUEST DTO — key JSON disamakan dgn kolom
   ========================================================= */

// Create
// Catatan: masjid_id bisa dikosongkan di request jika diisi dari token di controller.
type CreateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeachersMasjidID     *uuid.UUID `json:"class_section_subject_teachers_masjid_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeachersSectionID    uuid.UUID  `json:"class_section_subject_teachers_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeachersClassSubjectsID uuid.UUID `json:"class_section_subject_teachers_class_subjects_id" validate:"required,uuid"`

	// pakai masjid_teachers.masjid_teacher_id
	ClassSectionSubjectTeachersTeacherID uuid.UUID `json:"class_section_subject_teachers_teacher_id" validate:"required,uuid"`

	ClassSectionSubjectTeachersIsActive *bool `json:"class_section_subject_teachers_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeachersMasjidID        *uuid.UUID `json:"class_section_subject_teachers_masjid_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeachersSectionID       *uuid.UUID `json:"class_section_subject_teachers_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeachersClassSubjectsID *uuid.UUID `json:"class_section_subject_teachers_class_subjects_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeachersTeacherID *uuid.UUID `json:"class_section_subject_teachers_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeachersIsActive *bool `json:"class_section_subject_teachers_is_active" validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO — full snake_case seperti kolom
   ========================================================= */

type ClassSectionSubjectTeacherResponse struct {
	ClassSectionSubjectTeachersID            uuid.UUID  `json:"class_section_subject_teachers_id"`
	ClassSectionSubjectTeachersMasjidID      uuid.UUID  `json:"class_section_subject_teachers_masjid_id"`
	ClassSectionSubjectTeachersSectionID     uuid.UUID  `json:"class_section_subject_teachers_section_id"`
	ClassSectionSubjectTeachersClassSubjectsID uuid.UUID `json:"class_section_subject_teachers_class_subjects_id"`

	ClassSectionSubjectTeachersTeacherID uuid.UUID `json:"class_section_subject_teachers_teacher_id"`

	ClassSectionSubjectTeachersIsActive  bool       `json:"class_section_subject_teachers_is_active"`
	ClassSectionSubjectTeachersCreatedAt time.Time  `json:"class_section_subject_teachers_created_at"`
	ClassSectionSubjectTeachersUpdatedAt *time.Time `json:"class_section_subject_teachers_updated_at,omitempty"`
	ClassSectionSubjectTeachersDeletedAt *time.Time `json:"class_section_subject_teachers_deleted_at,omitempty"`
}

/* =========================================================
   3) MAPPERS
   ========================================================= */

func (r CreateClassSectionSubjectTeacherRequest) ToModel() csstModel.ClassSectionSubjectTeacherModel {
	m := csstModel.ClassSectionSubjectTeacherModel{
		ClassSectionSubjectTeachersSectionID:       r.ClassSectionSubjectTeachersSectionID,
		ClassSectionSubjectTeachersClassSubjectsID: r.ClassSectionSubjectTeachersClassSubjectsID,
		ClassSectionSubjectTeachersTeacherID:       r.ClassSectionSubjectTeachersTeacherID,
	}
	if r.ClassSectionSubjectTeachersMasjidID != nil {
		m.ClassSectionSubjectTeachersMasjidID = *r.ClassSectionSubjectTeachersMasjidID
	}
	if r.ClassSectionSubjectTeachersIsActive != nil {
		m.ClassSectionSubjectTeachersIsActive = *r.ClassSectionSubjectTeachersIsActive
	}
	return m
}

func (r UpdateClassSectionSubjectTeacherRequest) Apply(m *csstModel.ClassSectionSubjectTeacherModel) {
	if r.ClassSectionSubjectTeachersMasjidID != nil {
		m.ClassSectionSubjectTeachersMasjidID = *r.ClassSectionSubjectTeachersMasjidID
	}
	if r.ClassSectionSubjectTeachersSectionID != nil {
		m.ClassSectionSubjectTeachersSectionID = *r.ClassSectionSubjectTeachersSectionID
	}
	if r.ClassSectionSubjectTeachersClassSubjectsID != nil {
		m.ClassSectionSubjectTeachersClassSubjectsID = *r.ClassSectionSubjectTeachersClassSubjectsID
	}
	if r.ClassSectionSubjectTeachersTeacherID != nil {
		m.ClassSectionSubjectTeachersTeacherID = *r.ClassSectionSubjectTeachersTeacherID
	}
	if r.ClassSectionSubjectTeachersIsActive != nil {
		m.ClassSectionSubjectTeachersIsActive = *r.ClassSectionSubjectTeachersIsActive
	}
	// updated_at: dipegang GORM/trigger DB
}

func FromClassSectionSubjectTeacherModel(m csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if m.ClassSectionSubjectTeachersDeletedAt.Valid {
		t := m.ClassSectionSubjectTeachersDeletedAt.Time
		deletedAt = &t
	}
	return ClassSectionSubjectTeacherResponse{
		ClassSectionSubjectTeachersID:              m.ClassSectionSubjectTeachersID,
		ClassSectionSubjectTeachersMasjidID:        m.ClassSectionSubjectTeachersMasjidID,
		ClassSectionSubjectTeachersSectionID:       m.ClassSectionSubjectTeachersSectionID,
		ClassSectionSubjectTeachersClassSubjectsID: m.ClassSectionSubjectTeachersClassSubjectsID,
		ClassSectionSubjectTeachersTeacherID:       m.ClassSectionSubjectTeachersTeacherID,
		ClassSectionSubjectTeachersIsActive:        m.ClassSectionSubjectTeachersIsActive,
		ClassSectionSubjectTeachersCreatedAt:       m.ClassSectionSubjectTeachersCreatedAt,
		ClassSectionSubjectTeachersUpdatedAt:       m.ClassSectionSubjectTeachersUpdatedAt,
		ClassSectionSubjectTeachersDeletedAt:       deletedAt,
	}
}

func FromClassSectionSubjectTeacherModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromClassSectionSubjectTeacherModel(r))
	}
	return out
}


