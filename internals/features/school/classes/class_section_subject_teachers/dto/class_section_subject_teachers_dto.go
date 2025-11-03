// file: internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"encoding/json"
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
   0) TYPES untuk Snapshot Buku (parsed)
========================================================= */

type BookSnap struct {
	CSBID     uuid.UUID  `json:"csb_id"`
	BookID    uuid.UUID  `json:"book_id"`
	Title     string     `json:"title"`
	Author    *string    `json:"author,omitempty"`
	Slug      *string    `json:"slug,omitempty"`
	ImageURL  *string    `json:"image_url,omitempty"`
	IsPrimary bool       `json:"is_primary"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

/* =========================================================
   1) REQUEST DTO
========================================================= */

// Create
type CreateClassSectionSubjectTeacherRequest struct {
	// Biasanya diisi dari context auth pada controller
	ClassSectionSubjectTeacherSchoolID *uuid.UUID `json:"class_section_subject_teacher_school_id"  validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSectionID          uuid.UUID `json:"class_section_subject_teacher_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeacherClassSubjectBookID uuid.UUID `json:"class_section_subject_teacher_class_subject_book_id" validate:"required,uuid"`
	// pakai school_teachers.school_teacher_id
	ClassSectionSubjectTeacherTeacherID uuid.UUID `json:"class_section_subject_teacher_teacher_id" validate:"required,uuid"`

	// ➕ Asisten (opsional) — dipakai di controller untuk build snapshot JSON (bukan FK)
	ClassSectionSubjectTeacherAssistantTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_teacher_id" validate:"omitempty,uuid"`

	// Opsional
	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `json:"class_section_subject_teacher_room_id" validate:"omitempty,uuid"`
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
	ClassSectionSubjectTeacherSectionID          *uuid.UUID `json:"class_section_subject_teacher_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSubjectBookID *uuid.UUID `json:"class_section_subject_teacher_class_subject_book_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherTeacherID          *uuid.UUID `json:"class_section_subject_teacher_teacher_id" validate:"omitempty,uuid"`

	// ➕ Asisten (opsional) — untuk rebuild snapshot asisten saat update (jika perlu)
	ClassSectionSubjectTeacherAssistantTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSlug         *string                      `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription  *string                      `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherRoomID       *uuid.UUID                   `json:"class_section_subject_teacher_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL     *string                      `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherCapacity     *int                         `json:"class_section_subject_teacher_capacity" validate:"omitempty"`
	ClassSectionSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO — sinkron SQL/model terbaru
========================================================= */

type ClassSectionSubjectTeacherResponse struct {
	// IDs
	ClassSectionSubjectTeacherID                 uuid.UUID `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID           uuid.UUID `json:"class_section_subject_teacher_school_id"`
	ClassSectionSubjectTeacherSectionID          uuid.UUID `json:"class_section_subject_teacher_section_id"`
	ClassSectionSubjectTeacherClassSubjectBookID uuid.UUID `json:"class_section_subject_teacher_class_subject_book_id"`
	ClassSectionSubjectTeacherTeacherID          uuid.UUID `json:"class_section_subject_teacher_teacher_id"`

	// Identitas / fasilitas
	ClassSectionSubjectTeacherName        string     `json:"class_section_subject_teacher_name"`
	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `json:"class_section_subject_teacher_room_id,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url,omitempty"`

	// Agregat & kapasitas
	ClassSectionSubjectTeacherTotalAttendance int    `json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherCapacity        *int   `json:"class_section_subject_teacher_capacity,omitempty"`
	ClassSectionSubjectTeacherEnrolledCount   int    `json:"class_section_subject_teacher_enrolled_count"`
	ClassSectionSubjectTeacherDeliveryMode    string `json:"class_section_subject_teacher_delivery_mode"`

	// Room snapshot + turunan (generated)
	ClassSectionSubjectTeacherRoomSnapshot     *datatypes.JSON `json:"class_section_subject_teacher_room_snapshot,omitempty"`
	ClassSectionSubjectTeacherRoomNameSnap     *string         `json:"class_section_subject_teacher_room_name_snap,omitempty"`
	ClassSectionSubjectTeacherRoomSlugSnap     *string         `json:"class_section_subject_teacher_room_slug_snap,omitempty"`
	ClassSectionSubjectTeacherRoomLocationSnap *string         `json:"class_section_subject_teacher_room_location_snap,omitempty"`

	// People snapshots + turunan (generated)
	ClassSectionSubjectTeacherTeacherSnapshot          *datatypes.JSON `json:"class_section_subject_teacher_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherSnapshot *datatypes.JSON `json:"class_section_subject_teacher_assistant_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherTeacherNameSnap          *string         `json:"class_section_subject_teacher_teacher_name_snap,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherNameSnap *string         `json:"class_section_subject_teacher_assistant_teacher_name_snap,omitempty"`

	// ===== Snapshot CSB gabungan (opsional dikirim) =====
	// Raw snapshot gabungan (jika controller ingin expose)
	ClassSectionSubjectTeacherClassSubjectBookSnapshot *datatypes.JSON `json:"class_section_subject_teacher_class_subject_book_snapshot,omitempty"`

	// Turunan dari snapshot: BOOK*
	ClassSectionSubjectTeacherBookTitleSnap    *string `json:"class_section_subject_teacher_book_title_snap,omitempty"`
	ClassSectionSubjectTeacherBookAuthorSnap   *string `json:"class_section_subject_teacher_book_author_snap,omitempty"`
	ClassSectionSubjectTeacherBookSlugSnap     *string `json:"class_section_subject_teacher_book_slug_snap,omitempty"`
	ClassSectionSubjectTeacherBookImageURLSnap *string `json:"class_section_subject_teacher_book_image_url_snap,omitempty"`

	// Turunan dari snapshot: SUBJECT*
	ClassSectionSubjectTeacherSubjectNameSnap *string `json:"class_section_subject_teacher_subject_name_snap,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeSnap *string `json:"class_section_subject_teacher_subject_code_snap,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugSnap *string `json:"class_section_subject_teacher_subject_slug_snap,omitempty"`

	// ===== BOOKS snapshot array (raw + turunan) =====
	ClassSectionSubjectTeacherBooksSnapshot    datatypes.JSON `json:"class_section_subject_teacher_books_snapshot"`
	ClassSectionSubjectTeacherBooksCount       *int           `json:"class_section_subject_teacher_books_count,omitempty"`
	ClassSectionSubjectTeacherPrimaryBookTitle *string        `json:"class_section_subject_teacher_primary_book_title,omitempty"`

	// Versi parsed yang enak dipakai di FE
	BooksInUse []BookSnap `json:"books_in_use,omitempty"`

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
		ClassSectionSubjectTeacherSectionID:          r.ClassSectionSubjectTeacherSectionID,
		ClassSectionSubjectTeacherClassSubjectBookID: r.ClassSectionSubjectTeacherClassSubjectBookID,
		ClassSectionSubjectTeacherTeacherID:          r.ClassSectionSubjectTeacherTeacherID,

		// opsional
		ClassSectionSubjectTeacherSlug:        trimLowerPtr(r.ClassSectionSubjectTeacherSlug), // slug → lowercase
		ClassSectionSubjectTeacherDescription: trimPtr(r.ClassSectionSubjectTeacherDescription),
		ClassSectionSubjectTeacherRoomID:      r.ClassSectionSubjectTeacherRoomID,
		ClassSectionSubjectTeacherGroupURL:    trimPtr(r.ClassSectionSubjectTeacherGroupURL),
	}

	// Auto-derive NAME dari SLUG (lowercase)
	if r.ClassSectionSubjectTeacherSlug != nil {
		s := strings.ToLower(strings.TrimSpace(*r.ClassSectionSubjectTeacherSlug))
		if s != "" {
			m.ClassSectionSubjectTeacherName = s
		}
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
	if r.ClassSectionSubjectTeacherSectionID != nil {
		m.ClassSectionSubjectTeacherSectionID = *r.ClassSectionSubjectTeacherSectionID
	}
	if r.ClassSectionSubjectTeacherClassSubjectBookID != nil {
		m.ClassSectionSubjectTeacherClassSubjectBookID = *r.ClassSectionSubjectTeacherClassSubjectBookID
	}
	if r.ClassSectionSubjectTeacherTeacherID != nil {
		m.ClassSectionSubjectTeacherTeacherID = *r.ClassSectionSubjectTeacherTeacherID
	}

	if r.ClassSectionSubjectTeacherSlug != nil {
		cleaned := trimLowerPtr(r.ClassSectionSubjectTeacherSlug)
		m.ClassSectionSubjectTeacherSlug = cleaned
		// Jika slug diupdate menjadi non-empty, ikutkan pembaruan name agar tetap sama persis
		if cleaned != nil {
			m.ClassSectionSubjectTeacherName = *cleaned
		}
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
	if r.ClassSectionSubjectTeacherDeliveryMode != nil {
		m.ClassSectionSubjectTeacherDeliveryMode = *r.ClassSectionSubjectTeacherDeliveryMode
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

	// Parse snapshot books JSONB → []BookSnap (opsional)
	var parsedBooks []BookSnap
	if len(m.ClassSectionSubjectTeacherBooksSnapshot) > 0 {
		_ = json.Unmarshal(m.ClassSectionSubjectTeacherBooksSnapshot, &parsedBooks)
	}

	return ClassSectionSubjectTeacherResponse{
		// IDs
		ClassSectionSubjectTeacherID:                 m.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherSchoolID:           m.ClassSectionSubjectTeacherSchoolID,
		ClassSectionSubjectTeacherSectionID:          m.ClassSectionSubjectTeacherSectionID,
		ClassSectionSubjectTeacherClassSubjectBookID: m.ClassSectionSubjectTeacherClassSubjectBookID,
		ClassSectionSubjectTeacherTeacherID:          m.ClassSectionSubjectTeacherTeacherID,

		// Identitas / fasilitas
		ClassSectionSubjectTeacherName:        m.ClassSectionSubjectTeacherName,
		ClassSectionSubjectTeacherSlug:        m.ClassSectionSubjectTeacherSlug,
		ClassSectionSubjectTeacherDescription: m.ClassSectionSubjectTeacherDescription,
		ClassSectionSubjectTeacherRoomID:      m.ClassSectionSubjectTeacherRoomID,
		ClassSectionSubjectTeacherGroupURL:    m.ClassSectionSubjectTeacherGroupURL,

		// Agregat
		ClassSectionSubjectTeacherTotalAttendance: m.ClassSectionSubjectTeacherTotalAttendance,
		ClassSectionSubjectTeacherCapacity:        m.ClassSectionSubjectTeacherCapacity,
		ClassSectionSubjectTeacherEnrolledCount:   m.ClassSectionSubjectTeacherEnrolledCount,
		ClassSectionSubjectTeacherDeliveryMode:    string(m.ClassSectionSubjectTeacherDeliveryMode),

		// Snapshots: Room
		ClassSectionSubjectTeacherRoomSnapshot:     m.ClassSectionSubjectTeacherRoomSnapshot,
		ClassSectionSubjectTeacherRoomNameSnap:     m.ClassSectionSubjectTeacherRoomNameSnap,
		ClassSectionSubjectTeacherRoomSlugSnap:     m.ClassSectionSubjectTeacherRoomSlugSnap,
		ClassSectionSubjectTeacherRoomLocationSnap: m.ClassSectionSubjectTeacherRoomLocationSnap,

		// Snapshots: People
		ClassSectionSubjectTeacherTeacherSnapshot:          m.ClassSectionSubjectTeacherTeacherSnapshot,
		ClassSectionSubjectTeacherAssistantTeacherSnapshot: m.ClassSectionSubjectTeacherAssistantTeacherSnapshot,
		ClassSectionSubjectTeacherTeacherNameSnap:          m.ClassSectionSubjectTeacherTeacherNameSnap,
		ClassSectionSubjectTeacherAssistantTeacherNameSnap: m.ClassSectionSubjectTeacherAssistantTeacherNameSnap,

		// Snapshot CSB gabungan (opsional expose)
		ClassSectionSubjectTeacherClassSubjectBookSnapshot: m.ClassSectionSubjectTeacherClassSubjectBookSnapshot,

		// Turunan BOOK
		ClassSectionSubjectTeacherBookTitleSnap:    m.ClassSectionSubjectTeacherBookTitleSnap,
		ClassSectionSubjectTeacherBookAuthorSnap:   m.ClassSectionSubjectTeacherBookAuthorSnap,
		ClassSectionSubjectTeacherBookSlugSnap:     m.ClassSectionSubjectTeacherBookSlugSnap,
		ClassSectionSubjectTeacherBookImageURLSnap: m.ClassSectionSubjectTeacherBookImageURLSnap,

		// Turunan SUBJECT
		ClassSectionSubjectTeacherSubjectNameSnap: m.ClassSectionSubjectTeacherSubjectNameSnap,
		ClassSectionSubjectTeacherSubjectCodeSnap: m.ClassSectionSubjectTeacherSubjectCodeSnap,
		ClassSectionSubjectTeacherSubjectSlugSnap: m.ClassSectionSubjectTeacherSubjectSlugSnap,

		// Books snapshot (raw + parsed)
		ClassSectionSubjectTeacherBooksSnapshot:    m.ClassSectionSubjectTeacherBooksSnapshot,
		ClassSectionSubjectTeacherBooksCount:       m.ClassSectionSubjectTeacherBooksCount,
		ClassSectionSubjectTeacherPrimaryBookTitle: m.ClassSectionSubjectTeacherPrimaryBookTitle,
		BooksInUse: parsedBooks,

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
