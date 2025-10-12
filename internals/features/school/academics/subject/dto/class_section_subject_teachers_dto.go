// file: internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"strings"
	"time"

	csstModel "masjidku_backend/internals/features/school/academics/subject/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =========================================================
   Helpers
========================================================= */

func trimLowerPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*p))
	if s == "" {
		return nil
	}
	return &s
}


/* =========================================================
   1) REQUEST DTO
========================================================= */

// Create
// Catatan: class_section_subject_teacher_masjid_id biasanya diisi dari token pada controller.
type CreateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherMasjidID       *uuid.UUID `json:"class_section_subject_teacher_masjid_id"  validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSectionID      uuid.UUID  `json:"class_section_subject_teacher_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID  `json:"class_section_subject_teacher_class_subject_id" validate:"required,uuid"`
	// pakai masjid_teachers.masjid_teacher_id
	ClassSectionSubjectTeacherTeacherID uuid.UUID `json:"class_section_subject_teacher_teacher_id" validate:"required,uuid"`

	// ➕ Asisten (opsional) — dipakai di controller untuk buat snapshot JSON (bukan FK kolom)
	ClassSectionSubjectTeacherAssistantTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_teacher_id" validate:"omitempty,uuid"`

	// Opsional
	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `json:"class_section_subject_teacher_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherCapacity    *int       `json:"class_section_subject_teacher_capacity" validate:"omitempty"` // >=0 divalidasi di DB (CHECK)
	// enum: offline|online|hybrid
	ClassSectionsSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_sections_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	// Status
	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherMasjidID       *uuid.UUID `json:"class_section_subject_teacher_masjid_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSectionID      *uuid.UUID `json:"class_section_subject_teacher_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSubjectID *uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherTeacherID      *uuid.UUID `json:"class_section_subject_teacher_teacher_id" validate:"omitempty,uuid"`

	// ➕ Asisten (opsional) — untuk rebuild snapshot asisten saat update (jika perlu)
	ClassSectionSubjectTeacherAssistantTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSlug          *string                      `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription   *string                      `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherRoomID        *uuid.UUID                   `json:"class_section_subject_teacher_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL      *string                      `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherCapacity      *int                         `json:"class_section_subject_teacher_capacity" validate:"omitempty"`
	ClassSectionsSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_sections_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO — sesuai kolom di DB/model terbaru
========================================================= */

type ClassSectionSubjectTeacherResponse struct {
	// IDs
	ClassSectionSubjectTeacherID             uuid.UUID `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherMasjidID       uuid.UUID `json:"class_section_subject_teacher_masjid_id"`
	ClassSectionSubjectTeacherSectionID      uuid.UUID `json:"class_section_subject_teacher_section_id"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherTeacherID      uuid.UUID `json:"class_section_subject_teacher_teacher_id"`

	// Identitas / fasilitas
	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `json:"class_section_subject_teacher_room_id,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url,omitempty"`

	// Agregat & kapasitas
	ClassSectionSubjectTeacherTotalAttendance int    `json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherCapacity        *int   `json:"class_section_subject_teacher_capacity,omitempty"`
	ClassSectionSubjectTeacherEnrolledCount   int    `json:"class_section_subject_teacher_enrolled_count"`
	ClassSectionsSubjectTeacherDeliveryMode   string `json:"class_sections_subject_teacher_delivery_mode"`

	// Room snapshot + derived
	ClassSectionSubjectTeacherRoomSnapshot     *datatypes.JSON `json:"class_section_subject_teacher_room_snapshot,omitempty"`
	ClassSectionSubjectTeacherRoomNameSnap     *string         `json:"class_section_subject_teacher_room_name_snap,omitempty"`
	ClassSectionSubjectTeacherRoomSlugSnap     *string         `json:"class_section_subject_teacher_room_slug_snap,omitempty"`
	ClassSectionSubjectTeacherRoomLocationSnap *string         `json:"class_section_subject_teacher_room_location_snap,omitempty"`

	// People snapshots + derived
	ClassSectionSubjectTeacherTeacherSnapshot          *datatypes.JSON `json:"class_section_subject_teacher_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherSnapshot *datatypes.JSON `json:"class_section_subject_teacher_assistant_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherTeacherNameSnap          *string         `json:"class_section_subject_teacher_teacher_name_snap,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherNameSnap *string         `json:"class_section_subject_teacher_assistant_teacher_name_snap,omitempty"`

	// Status & audit
	ClassSectionSubjectTeacherIsActive  bool       `json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time  `json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time  `json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt *time.Time `json:"class_section_subject_teacher_deleted_at,omitempty"`
}

/* =========================================================
   3) MAPPERS
========================================================= */

func (r CreateClassSectionSubjectTeacherRequest) ToModel() csstModel.ClassSectionSubjectTeacherModel {
	m := csstModel.ClassSectionSubjectTeacherModel{
		ClassSectionSubjectTeacherSectionID:      r.ClassSectionSubjectTeacherSectionID,
		ClassSectionSubjectTeacherClassSubjectID: r.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherTeacherID:      r.ClassSectionSubjectTeacherTeacherID,

		// opsional
		ClassSectionSubjectTeacherSlug:        trimLowerPtr(r.ClassSectionSubjectTeacherSlug), // slug → lowercase
		ClassSectionSubjectTeacherDescription: trimPtr(r.ClassSectionSubjectTeacherDescription),
		ClassSectionSubjectTeacherRoomID:      r.ClassSectionSubjectTeacherRoomID,
		ClassSectionSubjectTeacherGroupURL:    trimPtr(r.ClassSectionSubjectTeacherGroupURL),
	}

	if r.ClassSectionSubjectTeacherMasjidID != nil {
		m.ClassSectionSubjectTeacherMasjidID = *r.ClassSectionSubjectTeacherMasjidID
	}
	if r.ClassSectionSubjectTeacherIsActive != nil {
		m.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	} else {
		m.ClassSectionSubjectTeacherIsActive = true
	}
	if r.ClassSectionSubjectTeacherCapacity != nil {
		m.ClassSectionSubjectTeacherCapacity = r.ClassSectionSubjectTeacherCapacity
	}
	if r.ClassSectionsSubjectTeacherDeliveryMode != nil {
		m.ClassSectionsSubjectTeacherDeliveryMode = *r.ClassSectionsSubjectTeacherDeliveryMode
	}

	return m
}

func (r UpdateClassSectionSubjectTeacherRequest) Apply(m *csstModel.ClassSectionSubjectTeacherModel) {
	if r.ClassSectionSubjectTeacherMasjidID != nil {
		m.ClassSectionSubjectTeacherMasjidID = *r.ClassSectionSubjectTeacherMasjidID
	}
	if r.ClassSectionSubjectTeacherSectionID != nil {
		m.ClassSectionSubjectTeacherSectionID = *r.ClassSectionSubjectTeacherSectionID
	}
	if r.ClassSectionSubjectTeacherClassSubjectID != nil {
		m.ClassSectionSubjectTeacherClassSubjectID = *r.ClassSectionSubjectTeacherClassSubjectID
	}
	if r.ClassSectionSubjectTeacherTeacherID != nil {
		m.ClassSectionSubjectTeacherTeacherID = *r.ClassSectionSubjectTeacherTeacherID
	}

	if r.ClassSectionSubjectTeacherSlug != nil {
		m.ClassSectionSubjectTeacherSlug = trimLowerPtr(r.ClassSectionSubjectTeacherSlug)
	}
	if r.ClassSectionSubjectTeacherDescription != nil {
		m.ClassSectionSubjectTeacherDescription = trimPtr(r.ClassSectionSubjectTeacherDescription)
	}
	if r.ClassSectionSubjectTeacherRoomID != nil {
		m.ClassSectionSubjectTeacherRoomID = r.ClassSectionSubjectTeacherRoomID
	}
	if r.ClassSectionSubjectTeacherGroupURL != nil {
		m.ClassSectionSubjectTeacherGroupURL = trimPtr(r.ClassSectionSubjectTeacherGroupURL)
	}
	if r.ClassSectionSubjectTeacherCapacity != nil {
		m.ClassSectionSubjectTeacherCapacity = r.ClassSectionSubjectTeacherCapacity
	}
	if r.ClassSectionsSubjectTeacherDeliveryMode != nil {
		m.ClassSectionsSubjectTeacherDeliveryMode = *r.ClassSectionsSubjectTeacherDeliveryMode
	}

	if r.ClassSectionSubjectTeacherIsActive != nil {
		m.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	}
	// updated_at dikelola oleh DB / GORM
}

func FromClassSectionSubjectTeacherModel(m csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if m.ClassSectionSubjectTeacherDeletedAt.Valid {
		t := m.ClassSectionSubjectTeacherDeletedAt.Time
		deletedAt = &t
	}
	return ClassSectionSubjectTeacherResponse{
		// IDs
		ClassSectionSubjectTeacherID:             m.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherMasjidID:       m.ClassSectionSubjectTeacherMasjidID,
		ClassSectionSubjectTeacherSectionID:      m.ClassSectionSubjectTeacherSectionID,
		ClassSectionSubjectTeacherClassSubjectID: m.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherTeacherID:      m.ClassSectionSubjectTeacherTeacherID,

		// Identitas / fasilitas
		ClassSectionSubjectTeacherSlug:        m.ClassSectionSubjectTeacherSlug,
		ClassSectionSubjectTeacherDescription: m.ClassSectionSubjectTeacherDescription,
		ClassSectionSubjectTeacherRoomID:      m.ClassSectionSubjectTeacherRoomID,
		ClassSectionSubjectTeacherGroupURL:    m.ClassSectionSubjectTeacherGroupURL,

		// Agregat
		ClassSectionSubjectTeacherTotalAttendance: m.ClassSectionSubjectTeacherTotalAttendance,
		ClassSectionSubjectTeacherCapacity:        m.ClassSectionSubjectTeacherCapacity,
		ClassSectionSubjectTeacherEnrolledCount:   m.ClassSectionSubjectTeacherEnrolledCount,
		ClassSectionsSubjectTeacherDeliveryMode:   string(m.ClassSectionsSubjectTeacherDeliveryMode),

		// Snapshots
		ClassSectionSubjectTeacherRoomSnapshot:     m.ClassSectionSubjectTeacherRoomSnapshot,
		ClassSectionSubjectTeacherRoomNameSnap:     m.ClassSectionSubjectTeacherRoomNameSnap,
		ClassSectionSubjectTeacherRoomSlugSnap:     m.ClassSectionSubjectTeacherRoomSlugSnap,
		ClassSectionSubjectTeacherRoomLocationSnap: m.ClassSectionSubjectTeacherRoomLocationSnap,

		ClassSectionSubjectTeacherTeacherSnapshot:          m.ClassSectionSubjectTeacherTeacherSnapshot,
		ClassSectionSubjectTeacherAssistantTeacherSnapshot: m.ClassSectionSubjectTeacherAssistantTeacherSnapshot,
		ClassSectionSubjectTeacherTeacherNameSnap:          m.ClassSectionSubjectTeacherTeacherNameSnap,
		ClassSectionSubjectTeacherAssistantTeacherNameSnap: m.ClassSectionSubjectTeacherAssistantTeacherNameSnap,

		// Status & audit
		ClassSectionSubjectTeacherIsActive:  m.ClassSectionSubjectTeacherIsActive,
		ClassSectionSubjectTeacherCreatedAt: m.ClassSectionSubjectTeacherCreatedAt,
		ClassSectionSubjectTeacherUpdatedAt: m.ClassSectionSubjectTeacherUpdatedAt,
		ClassSectionSubjectTeacherDeletedAt: deletedAt,
	}
}

func FromClassSectionSubjectTeacherModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromClassSectionSubjectTeacherModel(r))
	}
	return out
}
