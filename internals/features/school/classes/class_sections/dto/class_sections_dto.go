// file: internals/features/school/classes/sections/main/dto/class_section_dto.go
package dto

import (
	"strings"
	"time"

	m "masjidku_backend/internals/features/school/classes/class_sections/model"

	"github.com/google/uuid"
)

/* ===================== Requests ===================== */

// CREATE (server bisa generate slug jika kosong)
type CreateClassSectionRequest struct {
	ClassSectionsMasjidID    uuid.UUID  `json:"class_sections_masjid_id"`                 // required
	ClassSectionsClassID     uuid.UUID  `json:"class_sections_class_id"`                  // required
	ClassSectionsTeacherID   *uuid.UUID `json:"class_sections_teacher_id,omitempty"`      // masjid_teachers.*
	ClassSectionsClassRoomID *uuid.UUID `json:"class_sections_class_room_id,omitempty"`  // class_rooms.class_room_id (opsional)

	ClassSectionsSlug string  `json:"class_sections_slug,omitempty"` // optional
	ClassSectionsName string  `json:"class_sections_name"`           // required
	ClassSectionsCode *string `json:"class_sections_code,omitempty"`

	ClassSectionsCapacity *int    `json:"class_sections_capacity,omitempty"`  // >= 0
	ClassSectionsSchedule *string `json:"class_sections_schedule,omitempty"`  // teks bebas, ex: "Jumat 19:00–21:00"
	ClassSectionsGroupURL *string `json:"class_sections_group_url,omitempty"` // URL (WA group, dsb.)
}

func (r *CreateClassSectionRequest) ToModel() *m.ClassSectionModel {
	out := &m.ClassSectionModel{
		ClassSectionsClassID:     r.ClassSectionsClassID,
		ClassSectionsMasjidID:    r.ClassSectionsMasjidID,
		ClassSectionsTeacherID:   r.ClassSectionsTeacherID,
		ClassSectionsClassRoomID: r.ClassSectionsClassRoomID,

		ClassSectionsSlug: strings.TrimSpace(r.ClassSectionsSlug),
		ClassSectionsName: strings.TrimSpace(r.ClassSectionsName),
		ClassSectionsCode: nil,

		ClassSectionsCapacity:  r.ClassSectionsCapacity,
		ClassSectionsSchedule:  nil,
		ClassSectionsGroupURL:  nil,
		ClassSectionsIsActive:  true, // default aktif
		// ClassSectionsTotalStudents default 0 dari DB
	}

	// optional code
	if r.ClassSectionsCode != nil {
		c := strings.TrimSpace(*r.ClassSectionsCode)
		out.ClassSectionsCode = &c
	}

	// optional schedule
	if r.ClassSectionsSchedule != nil {
		s := strings.TrimSpace(*r.ClassSectionsSchedule)
		out.ClassSectionsSchedule = &s
	}

	// optional group url
	if r.ClassSectionsGroupURL != nil {
		u := strings.TrimSpace(*r.ClassSectionsGroupURL)
		out.ClassSectionsGroupURL = &u
	}

	return out
}

// UPDATE (PATCH semantics: semua optional; kosong "" bisa dipakai utk clear)
type UpdateClassSectionRequest struct {
	ClassSectionsMasjidID    *uuid.UUID `json:"class_sections_masjid_id,omitempty"`
	ClassSectionsClassID     *uuid.UUID `json:"class_sections_class_id,omitempty"`
	ClassSectionsTeacherID   *uuid.UUID `json:"class_sections_teacher_id,omitempty"`     // masjid_teachers.*
	ClassSectionsClassRoomID *uuid.UUID `json:"class_sections_class_room_id,omitempty"`  // class_rooms.*

	ClassSectionsSlug *string `json:"class_sections_slug,omitempty"`
	ClassSectionsName *string `json:"class_sections_name,omitempty"`
	ClassSectionsCode *string `json:"class_sections_code,omitempty"`

	ClassSectionsCapacity *int    `json:"class_sections_capacity,omitempty"`
	ClassSectionsSchedule *string `json:"class_sections_schedule,omitempty"`
	ClassSectionsGroupURL *string `json:"class_sections_group_url,omitempty"`

	ClassSectionsIsActive *bool `json:"class_sections_is_active,omitempty"`
}

// ApplyToModel menerapkan patch ke model (TrimSpace untuk string).
// Catatan: jika field pointer diberikan & bernilai "" (string kosong),
// akan disimpan kosong (bukan NULL). Jika ingin benar-benar menghapus (NULL),
// kirimkan field sebagai tidak-ada (pointer = nil) atau terapkan logika khusus di controller.
func (r *UpdateClassSectionRequest) ApplyToModel(dst *m.ClassSectionModel) {
	if r.ClassSectionsMasjidID != nil {
		dst.ClassSectionsMasjidID = *r.ClassSectionsMasjidID
	}
	if r.ClassSectionsClassID != nil {
		dst.ClassSectionsClassID = *r.ClassSectionsClassID
	}
	if r.ClassSectionsTeacherID != nil {
		dst.ClassSectionsTeacherID = r.ClassSectionsTeacherID
	}
	if r.ClassSectionsClassRoomID != nil {
		dst.ClassSectionsClassRoomID = r.ClassSectionsClassRoomID
	}

	if r.ClassSectionsSlug != nil {
		s := strings.TrimSpace(*r.ClassSectionsSlug)
		dst.ClassSectionsSlug = s
	}
	if r.ClassSectionsName != nil {
		n := strings.TrimSpace(*r.ClassSectionsName)
		dst.ClassSectionsName = n
	}
	if r.ClassSectionsCode != nil {
		c := strings.TrimSpace(*r.ClassSectionsCode)
		// simpan sebagai pointer; "" akan jadi non-NULL empty string
		dst.ClassSectionsCode = &c
	}

	if r.ClassSectionsCapacity != nil {
		dst.ClassSectionsCapacity = r.ClassSectionsCapacity
	}
	if r.ClassSectionsSchedule != nil {
		s := strings.TrimSpace(*r.ClassSectionsSchedule)
		dst.ClassSectionsSchedule = &s
	}
	if r.ClassSectionsGroupURL != nil {
		u := strings.TrimSpace(*r.ClassSectionsGroupURL)
		dst.ClassSectionsGroupURL = &u
	}

	if r.ClassSectionsIsActive != nil {
		dst.ClassSectionsIsActive = *r.ClassSectionsIsActive
	}
}

/* ===================== Queries ===================== */

type ListClassSectionQuery struct {
	Limit      int        `query:"limit"`
	Offset     int        `query:"offset"`
	ActiveOnly *bool      `query:"active_only"`
	Search     *string    `query:"search"`      // name/code/slug (controller handles)
	ClassID    *uuid.UUID `query:"class_id"`    // filter by class
	TeacherID  *uuid.UUID `query:"teacher_id"`  // filter by masjid_teacher
	RoomID     *uuid.UUID `query:"room_id"`     // filter by class_room
	Sort       *string    `query:"sort"`        // name_asc|name_desc|created_at_asc|created_at_desc
}

/* ===================== Responses ===================== */

type UserLite struct {
	ID       uuid.UUID `json:"id"`
	UserName string    `json:"user_name"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
	FullName string    `json:"full_name"`
}

type ClassSectionResponse struct {
	ClassSectionsID           uuid.UUID  `json:"class_sections_id"`
	ClassSectionsClassID      uuid.UUID  `json:"class_sections_class_id"`
	ClassSectionsMasjidID     uuid.UUID  `json:"class_sections_masjid_id"`
	ClassSectionsTeacherID    *uuid.UUID `json:"class_sections_teacher_id,omitempty"`
	ClassSectionsClassRoomID  *uuid.UUID `json:"class_sections_class_room_id,omitempty"`

	ClassSectionsSlug         string   `json:"class_sections_slug"`
	ClassSectionsName         string   `json:"class_sections_name"`
	ClassSectionsCode         *string  `json:"class_sections_code,omitempty"`

	ClassSectionsCapacity     *int     `json:"class_sections_capacity,omitempty"`
	ClassSectionsSchedule     *string  `json:"class_sections_schedule,omitempty"`
	ClassSectionsGroupURL     *string  `json:"class_sections_group_url,omitempty"`

	// Denormalized counter
	ClassSectionsTotalStudents int       `json:"class_sections_total_students"`

	ClassSectionsIsActive     bool       `json:"class_sections_is_active"`
	ClassSectionsCreatedAt    time.Time  `json:"class_sections_created_at"`
	ClassSectionsUpdatedAt    time.Time  `json:"class_sections_updated_at"`
	ClassSectionsDeletedAt    *time.Time `json:"class_sections_deleted_at,omitempty"`

	Teacher *UserLite `json:"teacher,omitempty"` // enrichment (optional)
}

// builder with teacher name only
func NewClassSectionResponse(src *m.ClassSectionModel, teacherName string) *ClassSectionResponse {
	var deletedAt *time.Time
	if !src.ClassSectionsDeletedAt.Time.IsZero() {
		t := src.ClassSectionsDeletedAt.Time
		deletedAt = &t
	}

	return &ClassSectionResponse{
		ClassSectionsID:           src.ClassSectionsID,
		ClassSectionsClassID:      src.ClassSectionsClassID,
		ClassSectionsMasjidID:     src.ClassSectionsMasjidID,
		ClassSectionsTeacherID:    src.ClassSectionsTeacherID,
		ClassSectionsClassRoomID:  src.ClassSectionsClassRoomID,

		ClassSectionsSlug:         src.ClassSectionsSlug,
		ClassSectionsName:         src.ClassSectionsName,
		ClassSectionsCode:         src.ClassSectionsCode,

		ClassSectionsCapacity:     src.ClassSectionsCapacity,
		ClassSectionsSchedule:     src.ClassSectionsSchedule,
		ClassSectionsGroupURL:     src.ClassSectionsGroupURL,

		ClassSectionsTotalStudents: src.ClassSectionsTotalStudents,

		ClassSectionsIsActive:     src.ClassSectionsIsActive,
		ClassSectionsCreatedAt:    src.ClassSectionsCreatedAt,
		ClassSectionsUpdatedAt:    src.ClassSectionsUpdatedAt,
		ClassSectionsDeletedAt:    deletedAt,
	}
}

// builder with full Teacher object (preferred)
func NewClassSectionResponseWithTeacher(src *m.ClassSectionModel, t *UserLite) *ClassSectionResponse {
	teacherName := ""
	if t != nil {
		teacherName = t.FullName
	}
	resp := NewClassSectionResponse(src, teacherName)
	resp.Teacher = t
	return resp
}

func MapClassSectionsWithTeachers(models []m.ClassSectionModel, users map[uuid.UUID]UserLite) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(models))
	for i := range models {
		row := &models[i]
		var t *UserLite
		if row.ClassSectionsTeacherID != nil {
			if u, ok := users[*row.ClassSectionsTeacherID]; ok {
				uCopy := u
				t = &uCopy
			}
		}
		out = append(out, *NewClassSectionResponseWithTeacher(row, t))
	}
	return out
}
