// file: internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"strings"
	"time"

	csstModel "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/model"

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
func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

/* =========================================================
   1) REQUEST DTO (FOLLOW SQL/MODEL)
========================================================= */

// Create
type CreateClassSectionSubjectTeacherRequest struct {
	// Biasanya diisi dari context auth pada controller
	ClassSectionSubjectTeacherSchoolID *uuid.UUID `json:"class_section_subject_teacher_school_id"  validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherClassSectionID     uuid.UUID `json:"class_section_subject_teacher_class_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeacherClassSubjectBookID uuid.UUID `json:"class_section_subject_teacher_class_subject_book_id" validate:"required,uuid"`

	// pakai school_teachers.school_teacher_id
	ClassSectionSubjectTeacherSchoolTeacherID uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"required,uuid"`

	// ➕ Asisten (opsional)
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	// Opsional
	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherClassRoomID *uuid.UUID `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherCapacity    *int       `json:"class_section_subject_teacher_capacity" validate:"omitempty"` // >=0 divalidasi di DB (CHECK)

	// enum: offline|online|hybrid
	ClassSectionSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	// Status
	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherSchoolID           *uuid.UUID `json:"class_section_subject_teacher_school_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSectionID     *uuid.UUID `json:"class_section_subject_teacher_class_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSubjectBookID *uuid.UUID `json:"class_section_subject_teacher_class_subject_book_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSchoolTeacherID    *uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"omitempty,uuid"`

	// ➕ Asisten (opsional)
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSlug         *string                      `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription  *string                      `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherClassRoomID  *uuid.UUID                   `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL     *string                      `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherCapacity     *int                         `json:"class_section_subject_teacher_capacity" validate:"omitempty"`
	ClassSectionSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

/*
	=========================================================
	  2) RESPONSE DTO — sinkron SQL/model terbaru

=========================================================
*/
type ClassSectionSubjectTeacherResponse struct {
	/* ===== IDs & Relations ===== */
	ClassSectionSubjectTeacherID                       uuid.UUID  `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID                 uuid.UUID  `json:"class_section_subject_teacher_school_id"`
	ClassSectionSubjectTeacherClassSectionID           uuid.UUID  `json:"class_section_subject_teacher_class_section_id"`
	ClassSectionSubjectTeacherClassSubjectBookID       uuid.UUID  `json:"class_section_subject_teacher_class_subject_book_id"`
	ClassSectionSubjectTeacherSchoolTeacherID          uuid.UUID  `json:"class_section_subject_teacher_school_teacher_id"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id,omitempty"`
	ClassSectionSubjectTeacherClassRoomID              *uuid.UUID `json:"class_section_subject_teacher_class_room_id,omitempty"`

	/* ===== Identitas & Fasilitas ===== */
	ClassSectionSubjectTeacherSlug        *string `json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string `json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string `json:"class_section_subject_teacher_group_url,omitempty"`

	/* ===== Agregat & kapasitas ===== */
	ClassSectionSubjectTeacherTotalAttendance  int    `json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherCapacity         *int   `json:"class_section_subject_teacher_capacity,omitempty"`
	ClassSectionSubjectTeacherEnrolledCount    int    `json:"class_section_subject_teacher_enrolled_count"`
	ClassSectionSubjectTeacherTotalAssessments int    `json:"class_section_subject_teacher_total_assessments"`
	ClassSectionSubjectTeacherDeliveryMode     string `json:"class_section_subject_teacher_delivery_mode"`

	/* ===== SECTION snapshots (varchar/text) ===== */
	ClassSectionSubjectTeacherClassSectionSlugSnapshot *string `json:"class_section_subject_teacher_class_section_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionNameSnapshot *string `json:"class_section_subject_teacher_class_section_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionCodeSnapshot *string `json:"class_section_subject_teacher_class_section_code_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionURLSnapshot  *string `json:"class_section_subject_teacher_class_section_url_snapshot,omitempty"`

	/* ===== ROOM snapshot ===== */
	ClassSectionSubjectTeacherClassRoomSlugSnapshot *string         `json:"class_section_subject_teacher_class_room_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassRoomSnapshot     *datatypes.JSON `json:"class_section_subject_teacher_class_room_snapshot,omitempty"`
	// generated
	ClassSectionSubjectTeacherClassRoomNameSnapshot     *string `json:"class_section_subject_teacher_class_room_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassRoomSlugSnapshotGen  *string `json:"class_section_subject_teacher_class_room_slug_snapshot_gen,omitempty"`
	ClassSectionSubjectTeacherClassRoomLocationSnapshot *string `json:"class_section_subject_teacher_class_room_location_snapshot,omitempty"`

	/* ===== PEOPLE snapshots ===== */
	ClassSectionSubjectTeacherSchoolTeacherSlugSnapshot          *string         `json:"class_section_subject_teacher_school_teacher_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherSnapshot              *datatypes.JSON `json:"class_section_subject_teacher_school_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSlugSnapshot *string         `json:"class_section_subject_teacher_assistant_school_teacher_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot     *datatypes.JSON `json:"class_section_subject_teacher_assistant_school_teacher_snapshot,omitempty"`
	// generated names
	ClassSectionSubjectTeacherSchoolTeacherNameSnapshot          *string `json:"class_section_subject_teacher_school_teacher_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot *string `json:"class_section_subject_teacher_assistant_school_teacher_name_snapshot,omitempty"`

	/* ===== CLASS_SUBJECT_BOOK snapshot ===== */
	ClassSectionSubjectTeacherClassSubjectBookSlugSnapshot *string         `json:"class_section_subject_teacher_class_subject_book_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSubjectBookSnapshot     *datatypes.JSON `json:"class_section_subject_teacher_class_subject_book_snapshot,omitempty"`

	/* ===== Generated dari CSB snapshot (BOOK*) ===== */
	ClassSectionSubjectTeacherBookTitleSnapshot    *string `json:"class_section_subject_teacher_book_title_snapshot,omitempty"`
	ClassSectionSubjectTeacherBookAuthorSnapshot   *string `json:"class_section_subject_teacher_book_author_snapshot,omitempty"`
	ClassSectionSubjectTeacherBookSlugSnapshot     *string `json:"class_section_subject_teacher_book_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherBookImageURLSnapshot *string `json:"class_section_subject_teacher_book_image_url_snapshot,omitempty"`

	/* ===== Generated dari CSB snapshot (SUBJECT*) ===== */
	ClassSectionSubjectTeacherSubjectIDSnapshot   *uuid.UUID `json:"class_section_subject_teacher_subject_id_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectNameSnapshot *string    `json:"class_section_subject_teacher_subject_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeSnapshot *string    `json:"class_section_subject_teacher_subject_code_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugSnapshot *string    `json:"class_section_subject_teacher_subject_slug_snapshot,omitempty"`

	/* ===== Status & audit ===== */
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
		ClassSectionSubjectTeacherClassSectionID:     r.ClassSectionSubjectTeacherClassSectionID,
		ClassSectionSubjectTeacherClassSubjectBookID: r.ClassSectionSubjectTeacherClassSubjectBookID,
		ClassSectionSubjectTeacherSchoolTeacherID:    r.ClassSectionSubjectTeacherSchoolTeacherID,

		// opsional
		ClassSectionSubjectTeacherSlug:        trimLowerPtr(r.ClassSectionSubjectTeacherSlug), // slug → lowercase
		ClassSectionSubjectTeacherDescription: trimPtr(r.ClassSectionSubjectTeacherDescription),
		ClassSectionSubjectTeacherClassRoomID: r.ClassSectionSubjectTeacherClassRoomID,
		ClassSectionSubjectTeacherGroupURL:    trimPtr(r.ClassSectionSubjectTeacherGroupURL),
	}

	if r.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
		m.ClassSectionSubjectTeacherAssistantSchoolTeacherID = r.ClassSectionSubjectTeacherAssistantSchoolTeacherID
	}
	if r.ClassSectionSubjectTeacherSchoolID != nil {
		m.ClassSectionSubjectTeacherSchoolID = *r.ClassSectionSubjectTeacherSchoolID
	}
	if r.ClassSectionSubjectTeacherIsActive != nil {
		m.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	} else {
		m.ClassSectionSubjectTeacherIsActive = true
	}
	if r.ClassSectionSubjectTeacherCapacity != nil {
		m.ClassSectionSubjectTeacherCapacity = r.ClassSectionSubjectTeacherCapacity
	}
	if r.ClassSectionSubjectTeacherDeliveryMode != nil {
		m.ClassSectionSubjectTeacherDeliveryMode = *r.ClassSectionSubjectTeacherDeliveryMode
	}

	return m
}

func (r UpdateClassSectionSubjectTeacherRequest) Apply(m *csstModel.ClassSectionSubjectTeacherModel) {
	if r.ClassSectionSubjectTeacherSchoolID != nil {
		m.ClassSectionSubjectTeacherSchoolID = *r.ClassSectionSubjectTeacherSchoolID
	}
	if r.ClassSectionSubjectTeacherClassSectionID != nil {
		m.ClassSectionSubjectTeacherClassSectionID = *r.ClassSectionSubjectTeacherClassSectionID
	}
	if r.ClassSectionSubjectTeacherClassSubjectBookID != nil {
		m.ClassSectionSubjectTeacherClassSubjectBookID = *r.ClassSectionSubjectTeacherClassSubjectBookID
	}
	if r.ClassSectionSubjectTeacherSchoolTeacherID != nil {
		m.ClassSectionSubjectTeacherSchoolTeacherID = *r.ClassSectionSubjectTeacherSchoolTeacherID
	}

	if r.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
		m.ClassSectionSubjectTeacherAssistantSchoolTeacherID = r.ClassSectionSubjectTeacherAssistantSchoolTeacherID
	}

	if r.ClassSectionSubjectTeacherSlug != nil {
		m.ClassSectionSubjectTeacherSlug = trimLowerPtr(r.ClassSectionSubjectTeacherSlug)
	}
	if r.ClassSectionSubjectTeacherDescription != nil {
		m.ClassSectionSubjectTeacherDescription = trimPtr(r.ClassSectionSubjectTeacherDescription)
	}
	if r.ClassSectionSubjectTeacherClassRoomID != nil {
		m.ClassSectionSubjectTeacherClassRoomID = r.ClassSectionSubjectTeacherClassRoomID
	}
	if r.ClassSectionSubjectTeacherGroupURL != nil {
		m.ClassSectionSubjectTeacherGroupURL = trimPtr(r.ClassSectionSubjectTeacherGroupURL)
	}
	if r.ClassSectionSubjectTeacherCapacity != nil {
		m.ClassSectionSubjectTeacherCapacity = r.ClassSectionSubjectTeacherCapacity
	}
	if r.ClassSectionSubjectTeacherDeliveryMode != nil {
		m.ClassSectionSubjectTeacherDeliveryMode = *r.ClassSectionSubjectTeacherDeliveryMode
	}
	if r.ClassSectionSubjectTeacherIsActive != nil {
		m.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	}
}

func FromClassSectionSubjectTeacherModel(m csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if m.ClassSectionSubjectTeacherDeletedAt.Valid {
		t := m.ClassSectionSubjectTeacherDeletedAt.Time
		deletedAt = &t
	}

	return ClassSectionSubjectTeacherResponse{
		// IDs & Relations
		ClassSectionSubjectTeacherID:                       m.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherSchoolID:                 m.ClassSectionSubjectTeacherSchoolID,
		ClassSectionSubjectTeacherClassSectionID:           m.ClassSectionSubjectTeacherClassSectionID,
		ClassSectionSubjectTeacherClassSubjectBookID:       m.ClassSectionSubjectTeacherClassSubjectBookID,
		ClassSectionSubjectTeacherSchoolTeacherID:          m.ClassSectionSubjectTeacherSchoolTeacherID,
		ClassSectionSubjectTeacherAssistantSchoolTeacherID: m.ClassSectionSubjectTeacherAssistantSchoolTeacherID,
		ClassSectionSubjectTeacherClassRoomID:              m.ClassSectionSubjectTeacherClassRoomID,

		// Identitas / fasilitas
		ClassSectionSubjectTeacherSlug:        m.ClassSectionSubjectTeacherSlug,
		ClassSectionSubjectTeacherDescription: m.ClassSectionSubjectTeacherDescription,
		ClassSectionSubjectTeacherGroupURL:    m.ClassSectionSubjectTeacherGroupURL,

		// Agregat & kapasitas
		ClassSectionSubjectTeacherTotalAttendance:  m.ClassSectionSubjectTeacherTotalAttendance,
		ClassSectionSubjectTeacherCapacity:         m.ClassSectionSubjectTeacherCapacity,
		ClassSectionSubjectTeacherEnrolledCount:    m.ClassSectionSubjectTeacherEnrolledCount,
		ClassSectionSubjectTeacherTotalAssessments: m.ClassSectionSubjectTeacherTotalAssessments,
		ClassSectionSubjectTeacherDeliveryMode:     string(m.ClassSectionSubjectTeacherDeliveryMode),

		// SECTION snapshots
		ClassSectionSubjectTeacherClassSectionSlugSnapshot: m.ClassSectionSubjectTeacherClassSectionSlugSnapshot,
		ClassSectionSubjectTeacherClassSectionNameSnapshot: m.ClassSectionSubjectTeacherClassSectionNameSnapshot,
		ClassSectionSubjectTeacherClassSectionCodeSnapshot: m.ClassSectionSubjectTeacherClassSectionCodeSnapshot,
		ClassSectionSubjectTeacherClassSectionURLSnapshot:  m.ClassSectionSubjectTeacherClassSectionURLSnapshot,

		// ROOM snapshot + generated
		ClassSectionSubjectTeacherClassRoomSlugSnapshot:     m.ClassSectionSubjectTeacherClassRoomSlugSnapshot,
		ClassSectionSubjectTeacherClassRoomSnapshot:         m.ClassSectionSubjectTeacherClassRoomSnapshot,
		ClassSectionSubjectTeacherClassRoomNameSnapshot:     m.ClassSectionSubjectTeacherClassRoomNameSnapshot,
		ClassSectionSubjectTeacherClassRoomSlugSnapshotGen:  m.ClassSectionSubjectTeacherClassRoomSlugSnapshotGen,
		ClassSectionSubjectTeacherClassRoomLocationSnapshot: m.ClassSectionSubjectTeacherClassRoomLocationSnapshot,

		// PEOPLE snapshots + generated names
		ClassSectionSubjectTeacherSchoolTeacherSlugSnapshot:          m.ClassSectionSubjectTeacherSchoolTeacherSlugSnapshot,
		ClassSectionSubjectTeacherSchoolTeacherSnapshot:              m.ClassSectionSubjectTeacherSchoolTeacherSnapshot,
		ClassSectionSubjectTeacherAssistantSchoolTeacherSlugSnapshot: m.ClassSectionSubjectTeacherAssistantSchoolTeacherSlugSnapshot,
		ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot:     m.ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot,
		ClassSectionSubjectTeacherSchoolTeacherNameSnapshot:          m.ClassSectionSubjectTeacherSchoolTeacherNameSnapshot,
		ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot: m.ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot,

		// CSB snapshot
		ClassSectionSubjectTeacherClassSubjectBookSlugSnapshot: m.ClassSectionSubjectTeacherClassSubjectBookSlugSnapshot,
		ClassSectionSubjectTeacherClassSubjectBookSnapshot:     m.ClassSectionSubjectTeacherClassSubjectBookSnapshot,

		// BOOK* generated
		ClassSectionSubjectTeacherBookTitleSnapshot:    m.ClassSectionSubjectTeacherBookTitleSnapshot,
		ClassSectionSubjectTeacherBookAuthorSnapshot:   m.ClassSectionSubjectTeacherBookAuthorSnapshot,
		ClassSectionSubjectTeacherBookSlugSnapshot:     m.ClassSectionSubjectTeacherBookSlugSnapshot,
		ClassSectionSubjectTeacherBookImageURLSnapshot: m.ClassSectionSubjectTeacherBookImageURLSnapshot,

		// SUBJECT* generated
		ClassSectionSubjectTeacherSubjectIDSnapshot:   m.ClassSectionSubjectTeacherSubjectIDSnapshot,
		ClassSectionSubjectTeacherSubjectNameSnapshot: m.ClassSectionSubjectTeacherSubjectNameSnapshot,
		ClassSectionSubjectTeacherSubjectCodeSnapshot: m.ClassSectionSubjectTeacherSubjectCodeSnapshot,
		ClassSectionSubjectTeacherSubjectSlugSnapshot: m.ClassSectionSubjectTeacherSubjectSlugSnapshot,

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
