// file: internals/features/school/sections/dto/class_section_dto.go
package dto

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/classes/class_sections/model"
)

/* =========================================================
   PATCH FIELD — tri-state (absent | null | value)
   ========================================================= */

type PatchFieldCS[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchFieldCS[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

func (p PatchFieldCS[T]) Get() (*T, bool) { return p.Value, p.Present }

/* =========================================================
   CREATE REQUEST / RESPONSE
   ========================================================= */

type ClassSectionCreateRequest struct {
	// Wajib
	ClassSectionMasjidID uuid.UUID `json:"class_section_masjid_id" form:"class_section_masjid_id" validate:"required"`
	ClassSectionClassID  uuid.UUID `json:"class_section_class_id"  form:"class_section_class_id"  validate:"required"`
	ClassSectionSlug     string    `json:"class_section_slug"      form:"class_section_slug"      validate:"min=1,max=160"`
	ClassSectionName     string    `json:"class_section_name"      form:"class_section_name"      validate:"required,min=1,max=100"`

	// Opsional
	ClassSectionTeacherID          *uuid.UUID `json:"class_section_teacher_id"           form:"class_section_teacher_id"`
	ClassSectionAssistantTeacherID *uuid.UUID `json:"class_section_assistant_teacher_id" form:"class_section_assistant_teacher_id"`
	ClassSectionClassRoomID        *uuid.UUID `json:"class_section_class_room_id"        form:"class_section_class_room_id"`
	ClassSectionLeaderStudentID    *uuid.UUID `json:"class_section_leader_student_id"    form:"class_section_leader_student_id"`

	ClassSectionCode     *string `json:"class_section_code"     form:"class_section_code"     validate:"omitempty,max=50"`
	ClassSectionSchedule *string `json:"class_section_schedule" form:"class_section_schedule"`
	ClassSectionCapacity *int    `json:"class_section_capacity" form:"class_section_capacity"`

	ClassSectionTotalStudents *int    `json:"class_section_total_students" form:"class_section_total_students" validate:"omitempty,min=0"`
	ClassSectionGroupURL      *string `json:"class_section_group_url"      form:"class_section_group_url"`

	// Image (opsional)
	ClassSectionImageURL       *string `json:"class_section_image_url"        form:"class_section_image_url"`
	ClassSectionImageObjectKey *string `json:"class_section_image_object_key" form:"class_section_image_object_key"`

	// Status
	ClassSectionIsActive *bool `json:"class_section_is_active" form:"class_section_is_active"`
}

func (r *ClassSectionCreateRequest) Normalize() {
	trimPtr := func(pp **string, lower bool) {
		if pp == nil || *pp == nil {
			return
		}
		v := strings.TrimSpace(**pp)
		if v == "" {
			*pp = nil
			return
		}
		if lower {
			v = strings.ToLower(v)
		}
		*pp = &v
	}

	r.ClassSectionSlug = strings.ToLower(strings.TrimSpace(r.ClassSectionSlug))
	r.ClassSectionName = strings.TrimSpace(r.ClassSectionName)
	trimPtr(&r.ClassSectionCode, false)
	trimPtr(&r.ClassSectionSchedule, false)
	trimPtr(&r.ClassSectionGroupURL, false)
	trimPtr(&r.ClassSectionImageURL, false)
	trimPtr(&r.ClassSectionImageObjectKey, false)
}

func (r ClassSectionCreateRequest) ToModel() *m.ClassSectionModel {
	now := time.Now()
	cs := &m.ClassSectionModel{
		ClassSectionMasjidID: r.ClassSectionMasjidID,
		ClassSectionClassID:  r.ClassSectionClassID,
		ClassSectionSlug:     r.ClassSectionSlug,
		ClassSectionName:     r.ClassSectionName,

		ClassSectionTeacherID:          r.ClassSectionTeacherID,
		ClassSectionAssistantTeacherID: r.ClassSectionAssistantTeacherID,
		ClassSectionClassRoomID:        r.ClassSectionClassRoomID,
		ClassSectionLeaderStudentID:    r.ClassSectionLeaderStudentID,

		ClassSectionCode:     r.ClassSectionCode,
		ClassSectionSchedule: r.ClassSectionSchedule,
		ClassSectionCapacity: r.ClassSectionCapacity,

		ClassSectionGroupURL:       r.ClassSectionGroupURL,
		ClassSectionImageURL:       r.ClassSectionImageURL,
		ClassSectionImageObjectKey: r.ClassSectionImageObjectKey,

		ClassSectionCreatedAt: now,
		ClassSectionUpdatedAt: now,
	}
	if r.ClassSectionIsActive != nil {
		cs.ClassSectionIsActive = *r.ClassSectionIsActive
	} else {
		cs.ClassSectionIsActive = true
	}
	if r.ClassSectionTotalStudents != nil {
		cs.ClassSectionTotalStudents = *r.ClassSectionTotalStudents
	}
	return cs
}

type ClassSectionResponse struct {
	ClassSectionID       uuid.UUID `json:"class_section_id"`
	ClassSectionMasjidID uuid.UUID `json:"class_section_masjid_id"`

	ClassSectionClassID            uuid.UUID  `json:"class_section_class_id"`
	ClassSectionTeacherID          *uuid.UUID `json:"class_section_teacher_id"`
	ClassSectionAssistantTeacherID *uuid.UUID `json:"class_section_assistant_teacher_id"`
	ClassSectionClassRoomID        *uuid.UUID `json:"class_section_class_room_id"`

	ClassSectionLeaderStudentID *uuid.UUID `json:"class_section_leader_student_id"`

	ClassSectionSlug string  `json:"class_section_slug"`
	ClassSectionName string  `json:"class_section_name"`
	ClassSectionCode *string `json:"class_section_code"`

	ClassSectionSchedule *string `json:"class_section_schedule"`

	ClassSectionCapacity      *int `json:"class_section_capacity"`
	ClassSectionTotalStudents int  `json:"class_section_total_students"`

	ClassSectionGroupURL *string `json:"class_section_group_url"`

	ClassSectionImageURL                *string    `json:"class_section_image_url"`
	ClassSectionImageObjectKey          *string    `json:"class_section_image_object_key"`
	ClassSectionImageURLOld             *string    `json:"class_section_image_url_old"`
	ClassSectionImageObjectKeyOld       *string    `json:"class_section_image_object_key_old"`
	ClassSectionImageDeletePendingUntil *time.Time `json:"class_section_image_delete_pending_until"`

	ClassSectionIsActive  bool       `json:"class_section_is_active"`
	ClassSectionCreatedAt time.Time  `json:"class_section_created_at"`
	ClassSectionUpdatedAt time.Time  `json:"class_section_updated_at"`
	ClassSectionDeletedAt *time.Time `json:"class_section_deleted_at,omitempty"`
}

func FromModelClassSection(cs *m.ClassSectionModel) ClassSectionResponse {
	var deletedAt *time.Time
	if cs.ClassSectionDeletedAt.Valid {
		t := cs.ClassSectionDeletedAt.Time
		deletedAt = &t
	}
	return ClassSectionResponse{
		ClassSectionID:       cs.ClassSectionID,
		ClassSectionMasjidID: cs.ClassSectionMasjidID,

		ClassSectionClassID:            cs.ClassSectionClassID,
		ClassSectionTeacherID:          cs.ClassSectionTeacherID,
		ClassSectionAssistantTeacherID: cs.ClassSectionAssistantTeacherID,
		ClassSectionClassRoomID:        cs.ClassSectionClassRoomID,

		ClassSectionLeaderStudentID: cs.ClassSectionLeaderStudentID,

		ClassSectionSlug: cs.ClassSectionSlug,
		ClassSectionName: cs.ClassSectionName,
		ClassSectionCode: cs.ClassSectionCode,

		ClassSectionSchedule: cs.ClassSectionSchedule,

		ClassSectionCapacity:      cs.ClassSectionCapacity,
		ClassSectionTotalStudents: cs.ClassSectionTotalStudents,

		ClassSectionGroupURL: cs.ClassSectionGroupURL,

		ClassSectionImageURL:                cs.ClassSectionImageURL,
		ClassSectionImageObjectKey:          cs.ClassSectionImageObjectKey,
		ClassSectionImageURLOld:             cs.ClassSectionImageURLOld,
		ClassSectionImageObjectKeyOld:       cs.ClassSectionImageObjectKeyOld,
		ClassSectionImageDeletePendingUntil: cs.ClassSectionImageDeletePendingUntil,

		ClassSectionIsActive:  cs.ClassSectionIsActive,
		ClassSectionCreatedAt: cs.ClassSectionCreatedAt,
		ClassSectionUpdatedAt: cs.ClassSectionUpdatedAt,
		ClassSectionDeletedAt: deletedAt,
	}
}

/* =========================================================
   PATCH REQUEST — tri-state
   ========================================================= */

type ClassSectionPatchRequest struct {
	// String wajib di model
	ClassSectionSlug PatchFieldCS[string] `json:"class_section_slug"`
	ClassSectionName PatchFieldCS[string] `json:"class_section_name"`

	// UUID pointer di model (null → clear)
	ClassSectionTeacherID          PatchFieldCS[uuid.UUID] `json:"class_section_teacher_id"`
	ClassSectionAssistantTeacherID PatchFieldCS[uuid.UUID] `json:"class_section_assistant_teacher_id"`
	ClassSectionClassRoomID        PatchFieldCS[uuid.UUID] `json:"class_section_class_room_id"`
	ClassSectionLeaderStudentID    PatchFieldCS[uuid.UUID] `json:"class_section_leader_student_id"`

	// Pointer string
	ClassSectionCode     PatchFieldCS[*string] `json:"class_section_code"`
	ClassSectionSchedule PatchFieldCS[*string] `json:"class_section_schedule"`
	ClassSectionGroupURL PatchFieldCS[*string] `json:"class_section_group_url"`

	// Image
	ClassSectionImageURL       PatchFieldCS[*string] `json:"class_section_image_url"`
	ClassSectionImageObjectKey PatchFieldCS[*string] `json:"class_section_image_object_key"`

	// Old / delete pending
	ClassSectionImageURLOld             PatchFieldCS[*string]    `json:"class_section_image_url_old"`
	ClassSectionImageObjectKeyOld       PatchFieldCS[*string]    `json:"class_section_image_object_key_old"`
	ClassSectionImageDeletePendingUntil PatchFieldCS[*time.Time] `json:"class_section_image_delete_pending_until"`

	// Numerik & status
	ClassSectionCapacity      PatchFieldCS[int]  `json:"class_section_capacity"`
	ClassSectionTotalStudents PatchFieldCS[int]  `json:"class_section_total_students"`
	ClassSectionIsActive      PatchFieldCS[bool] `json:"class_section_is_active"`
}

func (p *ClassSectionPatchRequest) Normalize() {
	trim := func(pp **string, lower bool) {
		if pp == nil || *pp == nil {
			return
		}
		v := strings.TrimSpace(**pp)
		if v == "" {
			*pp = nil
			return
		}
		if lower {
			v = strings.ToLower(v)
		}
		*pp = &v
	}

	if p.ClassSectionSlug.Present && p.ClassSectionSlug.Value != nil {
		v := strings.ToLower(strings.TrimSpace(*p.ClassSectionSlug.Value))
		p.ClassSectionSlug.Value = &v
	}
	if p.ClassSectionName.Present && p.ClassSectionName.Value != nil {
		v := strings.TrimSpace(*p.ClassSectionName.Value)
		p.ClassSectionName.Value = &v
	}

	if p.ClassSectionCode.Present {
		trim(p.ClassSectionCode.Value, false)
	}
	if p.ClassSectionSchedule.Present {
		trim(p.ClassSectionSchedule.Value, false)
	}
	if p.ClassSectionGroupURL.Present {
		trim(p.ClassSectionGroupURL.Value, false)
	}
	if p.ClassSectionImageURL.Present {
		trim(p.ClassSectionImageURL.Value, false)
	}
	if p.ClassSectionImageObjectKey.Present {
		trim(p.ClassSectionImageObjectKey.Value, false)
	}
	if p.ClassSectionImageURLOld.Present {
		trim(p.ClassSectionImageURLOld.Value, false)
	}
	if p.ClassSectionImageObjectKeyOld.Present {
		trim(p.ClassSectionImageObjectKeyOld.Value, false)
	}
}

func (p ClassSectionPatchRequest) Apply(cs *m.ClassSectionModel) {
	// slug & name
	if p.ClassSectionSlug.Present && p.ClassSectionSlug.Value != nil {
		cs.ClassSectionSlug = *p.ClassSectionSlug.Value
	}
	if p.ClassSectionName.Present && p.ClassSectionName.Value != nil {
		cs.ClassSectionName = *p.ClassSectionName.Value
	}

	// UUID pointer fields
	if p.ClassSectionTeacherID.Present {
		if p.ClassSectionTeacherID.Value == nil {
			cs.ClassSectionTeacherID = nil
		} else {
			v := *p.ClassSectionTeacherID.Value
			cs.ClassSectionTeacherID = &v
		}
	}
	if p.ClassSectionAssistantTeacherID.Present {
		if p.ClassSectionAssistantTeacherID.Value == nil {
			cs.ClassSectionAssistantTeacherID = nil
		} else {
			v := *p.ClassSectionAssistantTeacherID.Value
			cs.ClassSectionAssistantTeacherID = &v
		}
	}
	if p.ClassSectionClassRoomID.Present {
		if p.ClassSectionClassRoomID.Value == nil {
			cs.ClassSectionClassRoomID = nil
		} else {
			v := *p.ClassSectionClassRoomID.Value
			cs.ClassSectionClassRoomID = &v
		}
	}
	if p.ClassSectionLeaderStudentID.Present {
		if p.ClassSectionLeaderStudentID.Value == nil {
			cs.ClassSectionLeaderStudentID = nil
		} else {
			v := *p.ClassSectionLeaderStudentID.Value
			cs.ClassSectionLeaderStudentID = &v
		}
	}

	// **string → pointer string di model
	if p.ClassSectionCode.Present {
		if p.ClassSectionCode.Value == nil {
			cs.ClassSectionCode = nil
		} else {
			cs.ClassSectionCode = *p.ClassSectionCode.Value
		}
	}
	if p.ClassSectionSchedule.Present {
		if p.ClassSectionSchedule.Value == nil {
			cs.ClassSectionSchedule = nil
		} else {
			cs.ClassSectionSchedule = *p.ClassSectionSchedule.Value
		}
	}
	if p.ClassSectionGroupURL.Present {
		if p.ClassSectionGroupURL.Value == nil {
			cs.ClassSectionGroupURL = nil
		} else {
			cs.ClassSectionGroupURL = *p.ClassSectionGroupURL.Value
		}
	}
	if p.ClassSectionImageURL.Present {
		if p.ClassSectionImageURL.Value == nil {
			cs.ClassSectionImageURL = nil
		} else {
			cs.ClassSectionImageURL = *p.ClassSectionImageURL.Value
		}
	}
	if p.ClassSectionImageObjectKey.Present {
		if p.ClassSectionImageObjectKey.Value == nil {
			cs.ClassSectionImageObjectKey = nil
		} else {
			cs.ClassSectionImageObjectKey = *p.ClassSectionImageObjectKey.Value
		}
	}
	if p.ClassSectionImageURLOld.Present {
		if p.ClassSectionImageURLOld.Value == nil {
			cs.ClassSectionImageURLOld = nil
		} else {
			cs.ClassSectionImageURLOld = *p.ClassSectionImageURLOld.Value
		}
	}
	if p.ClassSectionImageObjectKeyOld.Present {
		if p.ClassSectionImageObjectKeyOld.Value == nil {
			cs.ClassSectionImageObjectKeyOld = nil
		} else {
			cs.ClassSectionImageObjectKeyOld = *p.ClassSectionImageObjectKeyOld.Value
		}
	}
	if p.ClassSectionImageDeletePendingUntil.Present {
		if p.ClassSectionImageDeletePendingUntil.Value == nil {
			cs.ClassSectionImageDeletePendingUntil = nil
		} else {
			cs.ClassSectionImageDeletePendingUntil = *p.ClassSectionImageDeletePendingUntil.Value
		}
	}

	// numerik & status
	if p.ClassSectionCapacity.Present {
		if p.ClassSectionCapacity.Value == nil {
			cs.ClassSectionCapacity = nil
		} else {
			v := *p.ClassSectionCapacity.Value
			cs.ClassSectionCapacity = &v
		}
	}
	if p.ClassSectionTotalStudents.Present && p.ClassSectionTotalStudents.Value != nil {
		cs.ClassSectionTotalStudents = *p.ClassSectionTotalStudents.Value
	}
	if p.ClassSectionIsActive.Present && p.ClassSectionIsActive.Value != nil {
		cs.ClassSectionIsActive = *p.ClassSectionIsActive.Value
	}

	cs.ClassSectionUpdatedAt = time.Now()
}

/* =========================================================
   LIST QUERY + HELPERS
   ========================================================= */

type ListClassSectionQuery struct {
	Limit     int        `query:"limit"`
	Offset    int        `query:"offset"`
	Q         string     `query:"q"`
	Active    *bool      `query:"active"`
	ClassID   *uuid.UUID `query:"class_id"`
	TeacherID *uuid.UUID `query:"teacher_id"`
	RoomID    *uuid.UUID `query:"room_id"`
	CreatedGt *time.Time `query:"created_gt"`
	CreatedLt *time.Time `query:"created_lt"`
}

func ToClassSectionResponses(rows []m.ClassSectionModel) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromModelClassSection(&rows[i]))
	}
	return out
}

type PaginationMeta struct {
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	Count      int   `json:"count"`
	NextOffset *int  `json:"next_offset,omitempty"`
	PrevOffset *int  `json:"prev_offset,omitempty"`
	HasMore    bool  `json:"has_more"`
}

func NewPaginationMeta(total int64, limit, offset, count int) PaginationMeta {
	if limit <= 0 {
		limit = 20
	}
	meta := PaginationMeta{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Count:   count,
		HasMore: int64(offset+count) < total,
	}
	if offset > 0 {
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		meta.PrevOffset = &prev
	}
	if meta.HasMore {
		next := offset + count
		meta.NextOffset = &next
	}
	return meta
}

// DecodePatchClassSectionFromRequest:
//   - multipart/form-data:
//     a) jika ada field "payload" -> unmarshal JSON,
//     b) jika tidak ada -> baca form key-value via DecodePatchClassSectionMultipart.
//   - application/json -> BodyParser biasa.
func DecodePatchClassSectionFromRequest(c *fiber.Ctx, out *ClassSectionPatchRequest) error {
	ct := strings.ToLower(c.Get("Content-Type"))
	if strings.Contains(ct, "multipart/form-data") {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := json.Unmarshal([]byte(s), out); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "payload JSON tidak valid: "+err.Error())
			}
		} else if err := DecodePatchClassSectionMultipart(c, out); err != nil {
			return err
		}
	} else {
		if err := c.BodyParser(out); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Body JSON tidak valid")
		}
	}
	out.Normalize()
	return nil
}

// DecodePatchClassSectionMultipart: map form key-value -> tri-state.
func DecodePatchClassSectionMultipart(c *fiber.Ctx, r *ClassSectionPatchRequest) error {
	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Form-data tidak ditemukan")
	}

	// helpers
	get := func(k string) (string, bool) {
		if vs, ok := form.Value[k]; ok {
			if len(vs) == 0 {
				return "", true
			}
			return vs[0], true
		}
		return "", false
	}
	setStr := func(field *PatchFieldCS[string], key string, lower bool) {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if lower {
				v = strings.ToLower(v)
			}
			field.Value = &v
		}
	}
	setStrPtr := func(field *PatchFieldCS[*string], key string, lower bool) {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				field.Value = nil
				return
			}
			if lower {
				v = strings.ToLower(v)
			}
			ptr := new(*string)
			*ptr = &v
			field.Value = ptr
		}
	}
	setUUID := func(field *PatchFieldCS[uuid.UUID], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				field.Value = nil
				return nil
			}
			u, e := uuid.Parse(v)
			if e != nil {
				return fiber.NewError(fiber.StatusBadRequest, label+" invalid UUID")
			}
			field.Value = &u
		}
		return nil
	}
	setTimePtr := func(field *PatchFieldCS[*time.Time], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			s := strings.TrimSpace(v)
			if s == "" {
				field.Value = nil
				return nil
			}
			if t, e := time.Parse(time.RFC3339, s); e == nil {
				pp := new(*time.Time)
				*pp = &t
				field.Value = pp
				return nil
			}
			if t, e := time.Parse("2006-01-02", s); e == nil {
				pp := new(*time.Time)
				*pp = &t
				field.Value = pp
				return nil
			}
			return fiber.NewError(fiber.StatusBadRequest, label+" format invalid (pakai RFC3339 atau YYYY-MM-DD)")
		}
		return nil
	}
	setInt := func(field *PatchFieldCS[int], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				field.Value = nil
				return nil
			}
			x, e := strconv.Atoi(v)
			if e != nil {
				return fiber.NewError(fiber.StatusBadRequest, label+" harus int")
			}
			field.Value = &x
		}
		return nil
	}
	setBool := func(field *PatchFieldCS[bool], key string) error {
		if v, ok := get(key); ok {
			field.Present = true
			s := strings.ToLower(strings.TrimSpace(v))
			if s == "" {
				field.Value = nil
				return nil
			}
			switch s {
			case "1", "true", "on", "yes", "y":
				b := true
				field.Value = &b
			case "0", "false", "off", "no", "n":
				b := false
				field.Value = &b
			default:
				return fiber.NewError(fiber.StatusBadRequest, key+" harus boolean (true/false)")
			}
		}
		return nil
	}

	// mapping field
	setStr(&r.ClassSectionSlug, "class_section_slug", true)
	setStr(&r.ClassSectionName, "class_section_name", false)

	_ = setUUID(&r.ClassSectionTeacherID, "class_section_teacher_id", "class_section_teacher_id")
	_ = setUUID(&r.ClassSectionAssistantTeacherID, "class_section_assistant_teacher_id", "class_section_assistant_teacher_id")
	_ = setUUID(&r.ClassSectionClassRoomID, "class_section_class_room_id", "class_section_class_room_id")
	_ = setUUID(&r.ClassSectionLeaderStudentID, "class_section_leader_student_id", "class_section_leader_student_id")

	setStrPtr(&r.ClassSectionCode, "class_section_code", false)
	setStrPtr(&r.ClassSectionSchedule, "class_section_schedule", false)
	setStrPtr(&r.ClassSectionGroupURL, "class_section_group_url", false)

	setStrPtr(&r.ClassSectionImageURL, "class_section_image_url", false)
	setStrPtr(&r.ClassSectionImageObjectKey, "class_section_image_object_key", false)
	setStrPtr(&r.ClassSectionImageURLOld, "class_section_image_url_old", false)
	setStrPtr(&r.ClassSectionImageObjectKeyOld, "class_section_image_object_key_old", false)

	if err := setTimePtr(&r.ClassSectionImageDeletePendingUntil, "class_section_image_delete_pending_until", "class_section_image_delete_pending_until"); err != nil {
		return err
	}

	if err := setInt(&r.ClassSectionCapacity, "class_section_capacity", "class_section_capacity"); err != nil {
		return err
	}
	if err := setInt(&r.ClassSectionTotalStudents, "class_section_total_students", "class_section_total_students"); err != nil {
		return err
	}
	if err := setBool(&r.ClassSectionIsActive, "class_section_is_active"); err != nil {
		return err
	}

	return nil
}
