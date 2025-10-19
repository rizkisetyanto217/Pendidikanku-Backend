// file: internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	csstModel "masjidku_backend/internals/features/school/classes/class_section_subject_teachers/model"

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
   - Disinkronkan dgn builder snapshot di DB:
   - key: csb_id, book_id, title, author, slug, image_url, is_primary, created_at
========================================================= */

type BookSnap struct {
	CSBID     uuid.UUID  `json:"csb_id"`
	BookID    uuid.UUID  `json:"book_id"`
	Title     string     `json:"title"`
	Author    *string    `json:"author,omitempty"`
	Slug      *string    `json:"slug,omitempty"`
	ImageURL  *string    `json:"image_url,omitempty"`
	IsPrimary bool       `json:"is_primary"` // jika di snapshot diset; default false kalau tidak ada
	CreatedAt *time.Time `json:"created_at,omitempty"`
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
   2) RESPONSE DTO — sinkron dgn SQL/model terbaru
========================================================= */

type ClassSectionSubjectTeacherResponse struct {
	// IDs
	ClassSectionSubjectTeacherID             uuid.UUID `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherMasjidID       uuid.UUID `json:"class_section_subject_teacher_masjid_id"`
	ClassSectionSubjectTeacherSectionID      uuid.UUID `json:"class_section_subject_teacher_section_id"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherTeacherID      uuid.UUID `json:"class_section_subject_teacher_teacher_id"`

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
	ClassSectionsSubjectTeacherDeliveryMode   string `json:"class_sections_subject_teacher_delivery_mode"`

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

	// Class Subject snapshot + turunan (generated)
	ClassSectionSubjectTeacherClassSubjectSnapshot *datatypes.JSON `json:"class_section_subject_teacher_class_subject_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSubjectNameSnap *string         `json:"class_section_subject_teacher_class_subject_name_snap,omitempty"`
	ClassSectionSubjectTeacherClassSubjectCodeSnap *string         `json:"class_section_subject_teacher_class_subject_code_snap,omitempty"`
	ClassSectionSubjectTeacherClassSubjectSlugSnap *string         `json:"class_section_subject_teacher_class_subject_slug_snap,omitempty"`
	ClassSectionSubjectTeacherClassSubjectURLSnap  *string         `json:"class_section_subject_teacher_class_subject_url_snap,omitempty"`

	// ===== NEW: BOOKS snapshot (raw + turunan/generated) =====
	// Raw JSONB exactly from table (biar kompatibel dgn existing API)
	ClassSectionSubjectTeacherBooksSnapshot datatypes.JSON `json:"class_section_subject_teacher_books_snapshot"`

	// Turunan (generated column di DB, kalau ada)
	ClassSectionSubjectTeacherBooksCount       *int    `json:"class_section_subject_teacher_books_count,omitempty"`
	ClassSectionSubjectTeacherPrimaryBookTitle *string `json:"class_section_subject_teacher_primary_book_title,omitempty"`

	// Versi parsed yang enak dipakai di FE (dibangun di mapper)
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
		ClassSectionSubjectTeacherSectionID:      r.ClassSectionSubjectTeacherSectionID,
		ClassSectionSubjectTeacherClassSubjectID: r.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherTeacherID:      r.ClassSectionSubjectTeacherTeacherID,

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

	// --- parse snapshot books JSONB → []BookSnap (optional, aman kalau gagal) ---
	var parsedBooks []BookSnap
	if len(m.ClassSectionSubjectTeacherBooksSnapshot) > 0 {
		_ = json.Unmarshal(m.ClassSectionSubjectTeacherBooksSnapshot, &parsedBooks)
	}

	return ClassSectionSubjectTeacherResponse{
		// IDs
		ClassSectionSubjectTeacherID:             m.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherMasjidID:       m.ClassSectionSubjectTeacherMasjidID,
		ClassSectionSubjectTeacherSectionID:      m.ClassSectionSubjectTeacherSectionID,
		ClassSectionSubjectTeacherClassSubjectID: m.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherTeacherID:      m.ClassSectionSubjectTeacherTeacherID,

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
		ClassSectionsSubjectTeacherDeliveryMode:   string(m.ClassSectionsSubjectTeacherDeliveryMode),

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

		// Snapshots: Class Subject
		ClassSectionSubjectTeacherClassSubjectSnapshot: m.ClassSectionSubjectTeacherClassSubjectSnapshot,
		ClassSectionSubjectTeacherClassSubjectNameSnap: m.ClassSectionSubjectTeacherClassSubjectNameSnap,
		ClassSectionSubjectTeacherClassSubjectCodeSnap: m.ClassSectionSubjectTeacherClassSubjectCodeSnap,
		ClassSectionSubjectTeacherClassSubjectSlugSnap: m.ClassSectionSubjectTeacherClassSubjectSlugSnap,
		ClassSectionSubjectTeacherClassSubjectURLSnap:  m.ClassSectionSubjectTeacherClassSubjectURLSnap,

		// NEW: Books snapshot (raw + parsed + generated)
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
