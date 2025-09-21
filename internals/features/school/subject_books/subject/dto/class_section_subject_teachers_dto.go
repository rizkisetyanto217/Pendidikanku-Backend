// internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"time"

	csstModel "masjidku_backend/internals/features/school/subject_books/subject/model"

	"github.com/google/uuid"
)

/* =========================================================
   1) REQUEST DTO — key JSON = nama kolom
========================================================= */

// Create
// Catatan: class_section_subject_teachers_masjid_id boleh kosong; isi dari token di controller.
type CreateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeachersMasjidID        *uuid.UUID `json:"class_section_subject_teachers_masjid_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeachersSectionID       uuid.UUID  `json:"class_section_subject_teachers_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeachersClassSubjectsID uuid.UUID  `json:"class_section_subject_teachers_class_subjects_id" validate:"required,uuid"`

	// pakai masjid_teachers.masjid_teacher_id
	ClassSectionSubjectTeachersTeacherID uuid.UUID `json:"class_section_subject_teachers_teacher_id" validate:"required,uuid"`

	// >>> SLUG <<<
	ClassSectionSubjectTeachersSlug *string `json:"class_section_subject_teachers_slug" validate:"omitempty,max=160"`

	// Deskripsi (opsional)
	ClassSectionSubjectTeachersDescription *string `json:"class_section_subject_teachers_description" validate:"omitempty"`

	// Override ruangan (opsional)
	ClassSectionSubjectTeachersRoomID *uuid.UUID `json:"class_section_subject_teachers_room_id" validate:"omitempty,uuid"`

	// Link grup pelajaran (opsional)
	ClassSectionSubjectTeachersGroupURL *string `json:"class_section_subject_teachers_group_url" validate:"omitempty,max=2000"`

	// Status
	ClassSectionSubjectTeachersIsActive *bool `json:"class_section_subject_teachers_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeachersMasjidID        *uuid.UUID `json:"class_section_subject_teachers_masjid_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeachersSectionID       *uuid.UUID `json:"class_section_subject_teachers_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeachersClassSubjectsID *uuid.UUID `json:"class_section_subject_teachers_class_subjects_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeachersTeacherID *uuid.UUID `json:"class_section_subject_teachers_teacher_id" validate:"omitempty,uuid"`

	// >>> SLUG <<<
	ClassSectionSubjectTeachersSlug *string `json:"class_section_subject_teachers_slug" validate:"omitempty,max=160"`

	// Deskripsi
	ClassSectionSubjectTeachersDescription *string `json:"class_section_subject_teachers_description" validate:"omitempty"`

	// Override ruangan
	ClassSectionSubjectTeachersRoomID *uuid.UUID `json:"class_section_subject_teachers_room_id" validate:"omitempty,uuid"`

	// Link grup
	ClassSectionSubjectTeachersGroupURL *string `json:"class_section_subject_teachers_group_url" validate:"omitempty,max=2000"`

	// Status
	ClassSectionSubjectTeachersIsActive *bool `json:"class_section_subject_teachers_is_active" validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO — full kolom
========================================================= */

type ClassSectionSubjectTeacherResponse struct {
	ClassSectionSubjectTeachersID              uuid.UUID `json:"class_section_subject_teachers_id"`
	ClassSectionSubjectTeachersMasjidID        uuid.UUID `json:"class_section_subject_teachers_masjid_id"`
	ClassSectionSubjectTeachersSectionID       uuid.UUID `json:"class_section_subject_teachers_section_id"`
	ClassSectionSubjectTeachersClassSubjectsID uuid.UUID `json:"class_section_subject_teachers_class_subjects_id"`

	ClassSectionSubjectTeachersTeacherID uuid.UUID `json:"class_section_subject_teachers_teacher_id"`

	// >>> SLUG & info opsional
	ClassSectionSubjectTeachersSlug        *string    `json:"class_section_subject_teachers_slug,omitempty"`
	ClassSectionSubjectTeachersDescription *string    `json:"class_section_subject_teachers_description,omitempty"`
	ClassSectionSubjectTeachersRoomID      *uuid.UUID `json:"class_section_subject_teachers_room_id,omitempty"`
	ClassSectionSubjectTeachersGroupURL    *string    `json:"class_section_subject_teachers_group_url,omitempty"`

	// Status & audit
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

		// opsional
		ClassSectionSubjectTeachersSlug:        trimPtr(r.ClassSectionSubjectTeachersSlug),
		ClassSectionSubjectTeachersDescription: trimPtr(r.ClassSectionSubjectTeachersDescription),
		ClassSectionSubjectTeachersRoomID:      r.ClassSectionSubjectTeachersRoomID,
		ClassSectionSubjectTeachersGroupURL:    trimPtr(r.ClassSectionSubjectTeachersGroupURL),
	}

	if r.ClassSectionSubjectTeachersMasjidID != nil {
		m.ClassSectionSubjectTeachersMasjidID = *r.ClassSectionSubjectTeachersMasjidID
	}
	if r.ClassSectionSubjectTeachersIsActive != nil {
		m.ClassSectionSubjectTeachersIsActive = *r.ClassSectionSubjectTeachersIsActive
	} else {
		m.ClassSectionSubjectTeachersIsActive = true
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

	// opsional yang bisa dikosongkan: gunakan trimPtr agar "" → nil
	if r.ClassSectionSubjectTeachersSlug != nil {
		m.ClassSectionSubjectTeachersSlug = trimPtr(r.ClassSectionSubjectTeachersSlug)
	}
	if r.ClassSectionSubjectTeachersDescription != nil {
		m.ClassSectionSubjectTeachersDescription = trimPtr(r.ClassSectionSubjectTeachersDescription)
	}
	if r.ClassSectionSubjectTeachersRoomID != nil {
		m.ClassSectionSubjectTeachersRoomID = r.ClassSectionSubjectTeachersRoomID
	}
	if r.ClassSectionSubjectTeachersGroupURL != nil {
		m.ClassSectionSubjectTeachersGroupURL = trimPtr(r.ClassSectionSubjectTeachersGroupURL)
	}

	if r.ClassSectionSubjectTeachersIsActive != nil {
		m.ClassSectionSubjectTeachersIsActive = *r.ClassSectionSubjectTeachersIsActive
	}
	// updated_at dipegang GORM/DB
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

		ClassSectionSubjectTeachersTeacherID: m.ClassSectionSubjectTeachersTeacherID,

		ClassSectionSubjectTeachersSlug:        m.ClassSectionSubjectTeachersSlug,
		ClassSectionSubjectTeachersDescription: m.ClassSectionSubjectTeachersDescription,
		ClassSectionSubjectTeachersRoomID:      m.ClassSectionSubjectTeachersRoomID,
		ClassSectionSubjectTeachersGroupURL:    m.ClassSectionSubjectTeachersGroupURL,

		ClassSectionSubjectTeachersIsActive:  m.ClassSectionSubjectTeachersIsActive,
		ClassSectionSubjectTeachersCreatedAt: m.ClassSectionSubjectTeachersCreatedAt,
		ClassSectionSubjectTeachersUpdatedAt: m.ClassSectionSubjectTeachersUpdatedAt,
		ClassSectionSubjectTeachersDeletedAt: deletedAt,
	}
}

func FromClassSectionSubjectTeacherModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromClassSectionSubjectTeacherModel(r))
	}
	return out
}
