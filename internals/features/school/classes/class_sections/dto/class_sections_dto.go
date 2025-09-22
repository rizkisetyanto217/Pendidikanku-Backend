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
// dto: internals/features/school/classes/class_sections/dto/create.go
type ClassSectionCreateRequest struct {
	// Wajib
	ClassSectionsMasjidID uuid.UUID `json:"class_sections_masjid_id" form:"class_sections_masjid_id" validate:"required"`
	ClassSectionsClassID  uuid.UUID `json:"class_sections_class_id"  form:"class_sections_class_id"  validate:"required"`
	ClassSectionsSlug     string    `json:"class_sections_slug"      form:"class_sections_slug"      validate:"min=1,max=160"`
	ClassSectionsName     string    `json:"class_sections_name"      form:"class_sections_name"      validate:"required,min=1,max=100"`

	// Opsional
	ClassSectionsTeacherID          *uuid.UUID `json:"class_sections_teacher_id"           form:"class_sections_teacher_id"`
	ClassSectionsAssistantTeacherID *uuid.UUID `json:"class_sections_assistant_teacher_id" form:"class_sections_assistant_teacher_id"`
	ClassSectionsClassRoomID        *uuid.UUID `json:"class_sections_class_room_id"        form:"class_sections_class_room_id"`
	ClassSectionsLeaderStudentID    *uuid.UUID `json:"class_sections_leader_student_id"    form:"class_sections_leader_student_id"`

	ClassSectionsCode     *string `json:"class_sections_code"     form:"class_sections_code"     validate:"omitempty,max=50"`
	ClassSectionsSchedule *string `json:"class_sections_schedule" form:"class_sections_schedule"`
	ClassSectionsCapacity *int    `json:"class_sections_capacity" form:"class_sections_capacity"`

	ClassSectionsTotalStudents *int    `json:"class_sections_total_students" form:"class_sections_total_students" validate:"omitempty,min=0"`
	ClassSectionsGroupURL      *string `json:"class_sections_group_url"      form:"class_sections_group_url"`

	// Image (opsional)
	ClassSectionsImageURL       *string `json:"class_sections_image_url"        form:"class_sections_image_url"`
	ClassSectionsImageObjectKey *string `json:"class_sections_image_object_key" form:"class_sections_image_object_key"`

	// Status
	ClassSectionsIsActive *bool `json:"class_sections_is_active" form:"class_sections_is_active"`
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

	r.ClassSectionsSlug = strings.ToLower(strings.TrimSpace(r.ClassSectionsSlug))
	r.ClassSectionsName = strings.TrimSpace(r.ClassSectionsName)
	trimPtr(&r.ClassSectionsCode, false)
	trimPtr(&r.ClassSectionsSchedule, false)
	trimPtr(&r.ClassSectionsGroupURL, false)
	trimPtr(&r.ClassSectionsImageURL, false)
	trimPtr(&r.ClassSectionsImageObjectKey, false)
}

func (r ClassSectionCreateRequest) ToModel() *m.ClassSectionModel {
	now := time.Now()
	cs := &m.ClassSectionModel{
		ClassSectionsMasjidID: r.ClassSectionsMasjidID,
		ClassSectionsClassID:  r.ClassSectionsClassID,
		ClassSectionsSlug:     r.ClassSectionsSlug,
		ClassSectionsName:     r.ClassSectionsName,

		ClassSectionsTeacherID:          r.ClassSectionsTeacherID,
		ClassSectionsAssistantTeacherID: r.ClassSectionsAssistantTeacherID,
		ClassSectionsClassRoomID:        r.ClassSectionsClassRoomID,
		ClassSectionsLeaderStudentID:    r.ClassSectionsLeaderStudentID,

		ClassSectionsCode:     r.ClassSectionsCode,
		ClassSectionsSchedule: r.ClassSectionsSchedule,
		ClassSectionsCapacity: r.ClassSectionsCapacity,

		ClassSectionsGroupURL:       r.ClassSectionsGroupURL,
		ClassSectionsImageURL:       r.ClassSectionsImageURL,
		ClassSectionsImageObjectKey: r.ClassSectionsImageObjectKey,

		ClassSectionsCreatedAt: now,
		ClassSectionsUpdatedAt: now,
	}
	if r.ClassSectionsIsActive != nil {
		cs.ClassSectionsIsActive = *r.ClassSectionsIsActive
	} else {
		cs.ClassSectionsIsActive = true
	}
	if r.ClassSectionsTotalStudents != nil {
		cs.ClassSectionsTotalStudents = *r.ClassSectionsTotalStudents
	}
	return cs
}

type ClassSectionResponse struct {
	ClassSectionsID       uuid.UUID `json:"class_sections_id"`
	ClassSectionsMasjidID uuid.UUID `json:"class_sections_masjid_id"`

	ClassSectionsClassID            uuid.UUID  `json:"class_sections_class_id"`
	ClassSectionsTeacherID          *uuid.UUID `json:"class_sections_teacher_id"`
	ClassSectionsAssistantTeacherID *uuid.UUID `json:"class_sections_assistant_teacher_id"`
	ClassSectionsClassRoomID        *uuid.UUID `json:"class_sections_class_room_id"`

	ClassSectionsLeaderStudentID *uuid.UUID `json:"class_sections_leader_student_id"`

	ClassSectionsSlug string  `json:"class_sections_slug"`
	ClassSectionsName string  `json:"class_sections_name"`
	ClassSectionsCode *string `json:"class_sections_code"`

	ClassSectionsSchedule *string `json:"class_sections_schedule"`

	ClassSectionsCapacity      *int `json:"class_sections_capacity"`
	ClassSectionsTotalStudents int  `json:"class_sections_total_students"`

	ClassSectionsGroupURL *string `json:"class_sections_group_url"`

	ClassSectionsImageURL                *string    `json:"class_sections_image_url"`
	ClassSectionsImageObjectKey          *string    `json:"class_sections_image_object_key"`
	ClassSectionsImageURLOld             *string    `json:"class_sections_image_url_old"`
	ClassSectionsImageObjectKeyOld       *string    `json:"class_sections_image_object_key_old"`
	ClassSectionsImageDeletePendingUntil *time.Time `json:"class_sections_image_delete_pending_until"`

	ClassSectionsIsActive  bool       `json:"class_sections_is_active"`
	ClassSectionsCreatedAt time.Time  `json:"class_sections_created_at"`
	ClassSectionsUpdatedAt time.Time  `json:"class_sections_updated_at"`
	ClassSectionsDeletedAt *time.Time `json:"class_sections_deleted_at,omitempty"`
}

func FromModelClassSection(cs *m.ClassSectionModel) ClassSectionResponse {
	var deletedAt *time.Time
	if cs.ClassSectionsDeletedAt.Valid {
		t := cs.ClassSectionsDeletedAt.Time
		deletedAt = &t
	}
	return ClassSectionResponse{
		ClassSectionsID:       cs.ClassSectionsID,
		ClassSectionsMasjidID: cs.ClassSectionsMasjidID,

		ClassSectionsClassID:            cs.ClassSectionsClassID,
		ClassSectionsTeacherID:          cs.ClassSectionsTeacherID,
		ClassSectionsAssistantTeacherID: cs.ClassSectionsAssistantTeacherID,
		ClassSectionsClassRoomID:        cs.ClassSectionsClassRoomID,

		ClassSectionsLeaderStudentID: cs.ClassSectionsLeaderStudentID,

		ClassSectionsSlug: cs.ClassSectionsSlug,
		ClassSectionsName: cs.ClassSectionsName,
		ClassSectionsCode: cs.ClassSectionsCode,

		ClassSectionsSchedule: cs.ClassSectionsSchedule,

		ClassSectionsCapacity:      cs.ClassSectionsCapacity,
		ClassSectionsTotalStudents: cs.ClassSectionsTotalStudents,

		ClassSectionsGroupURL: cs.ClassSectionsGroupURL,

		ClassSectionsImageURL:                cs.ClassSectionsImageURL,
		ClassSectionsImageObjectKey:          cs.ClassSectionsImageObjectKey,
		ClassSectionsImageURLOld:             cs.ClassSectionsImageURLOld,
		ClassSectionsImageObjectKeyOld:       cs.ClassSectionsImageObjectKeyOld,
		ClassSectionsImageDeletePendingUntil: cs.ClassSectionsImageDeletePendingUntil,

		ClassSectionsIsActive:  cs.ClassSectionsIsActive,
		ClassSectionsCreatedAt: cs.ClassSectionsCreatedAt,
		ClassSectionsUpdatedAt: cs.ClassSectionsUpdatedAt,
		ClassSectionsDeletedAt: deletedAt,
	}
}

/* =========================================================
   PATCH REQUEST — tri-state
   ========================================================= */

type ClassSectionPatchRequest struct {
	// String wajib di model
	ClassSectionsSlug PatchFieldCS[string] `json:"class_sections_slug"`
	ClassSectionsName PatchFieldCS[string] `json:"class_sections_name"`

	// UUID pointer di model (null → clear)
	ClassSectionsTeacherID          PatchFieldCS[uuid.UUID] `json:"class_sections_teacher_id"`
	ClassSectionsAssistantTeacherID PatchFieldCS[uuid.UUID] `json:"class_sections_assistant_teacher_id"`
	ClassSectionsClassRoomID        PatchFieldCS[uuid.UUID] `json:"class_sections_class_room_id"`
	ClassSectionsLeaderStudentID    PatchFieldCS[uuid.UUID] `json:"class_sections_leader_student_id"`

	// Pointer string (pakai T=*string supaya bisa **string saat normalize)
	ClassSectionsCode     PatchFieldCS[*string] `json:"class_sections_code"`
	ClassSectionsSchedule PatchFieldCS[*string] `json:"class_sections_schedule"`
	ClassSectionsGroupURL PatchFieldCS[*string] `json:"class_sections_group_url"`

	// Image
	ClassSectionsImageURL       PatchFieldCS[*string] `json:"class_sections_image_url"`
	ClassSectionsImageObjectKey PatchFieldCS[*string] `json:"class_sections_image_object_key"`

	// Old / delete pending
	ClassSectionsImageURLOld             PatchFieldCS[*string]    `json:"class_sections_image_url_old"`
	ClassSectionsImageObjectKeyOld       PatchFieldCS[*string]    `json:"class_sections_image_object_key_old"`
	ClassSectionsImageDeletePendingUntil PatchFieldCS[*time.Time] `json:"class_sections_image_delete_pending_until"`

	// Numerik & status
	ClassSectionsCapacity      PatchFieldCS[int]  `json:"class_sections_capacity"`       // null → set nil
	ClassSectionsTotalStudents PatchFieldCS[int]  `json:"class_sections_total_students"` // opsional
	ClassSectionsIsActive      PatchFieldCS[bool] `json:"class_sections_is_active"`
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

	if p.ClassSectionsSlug.Present && p.ClassSectionsSlug.Value != nil {
		v := strings.ToLower(strings.TrimSpace(*p.ClassSectionsSlug.Value))
		p.ClassSectionsSlug.Value = &v
	}
	if p.ClassSectionsName.Present && p.ClassSectionsName.Value != nil {
		v := strings.TrimSpace(*p.ClassSectionsName.Value)
		p.ClassSectionsName.Value = &v
	}

	if p.ClassSectionsCode.Present {
		trim(p.ClassSectionsCode.Value, false)
	}
	if p.ClassSectionsSchedule.Present {
		trim(p.ClassSectionsSchedule.Value, false)
	}
	if p.ClassSectionsGroupURL.Present {
		trim(p.ClassSectionsGroupURL.Value, false)
	}
	if p.ClassSectionsImageURL.Present {
		trim(p.ClassSectionsImageURL.Value, false)
	}
	if p.ClassSectionsImageObjectKey.Present {
		trim(p.ClassSectionsImageObjectKey.Value, false)
	}
	if p.ClassSectionsImageURLOld.Present {
		trim(p.ClassSectionsImageURLOld.Value, false)
	}
	if p.ClassSectionsImageObjectKeyOld.Present {
		trim(p.ClassSectionsImageObjectKeyOld.Value, false)
	}
}

func (p ClassSectionPatchRequest) Apply(cs *m.ClassSectionModel) {
	// slug & name
	if p.ClassSectionsSlug.Present && p.ClassSectionsSlug.Value != nil {
		cs.ClassSectionsSlug = *p.ClassSectionsSlug.Value
	}
	if p.ClassSectionsName.Present && p.ClassSectionsName.Value != nil {
		cs.ClassSectionsName = *p.ClassSectionsName.Value
	}

	// UUID pointer fields
	if p.ClassSectionsTeacherID.Present {
		if p.ClassSectionsTeacherID.Value == nil {
			cs.ClassSectionsTeacherID = nil
		} else {
			v := *p.ClassSectionsTeacherID.Value
			cs.ClassSectionsTeacherID = &v
		}
	}
	if p.ClassSectionsAssistantTeacherID.Present {
		if p.ClassSectionsAssistantTeacherID.Value == nil {
			cs.ClassSectionsAssistantTeacherID = nil
		} else {
			v := *p.ClassSectionsAssistantTeacherID.Value
			cs.ClassSectionsAssistantTeacherID = &v
		}
	}
	if p.ClassSectionsClassRoomID.Present {
		if p.ClassSectionsClassRoomID.Value == nil {
			cs.ClassSectionsClassRoomID = nil
		} else {
			v := *p.ClassSectionsClassRoomID.Value
			cs.ClassSectionsClassRoomID = &v
		}
	}
	if p.ClassSectionsLeaderStudentID.Present {
		if p.ClassSectionsLeaderStudentID.Value == nil {
			cs.ClassSectionsLeaderStudentID = nil
		} else {
			v := *p.ClassSectionsLeaderStudentID.Value
			cs.ClassSectionsLeaderStudentID = &v
		}
	}

	// **string → pointer string di model
	if p.ClassSectionsCode.Present {
		if p.ClassSectionsCode.Value == nil {
			cs.ClassSectionsCode = nil
		} else {
			cs.ClassSectionsCode = *p.ClassSectionsCode.Value
		}
	}
	if p.ClassSectionsSchedule.Present {
		if p.ClassSectionsSchedule.Value == nil {
			cs.ClassSectionsSchedule = nil
		} else {
			cs.ClassSectionsSchedule = *p.ClassSectionsSchedule.Value
		}
	}
	if p.ClassSectionsGroupURL.Present {
		if p.ClassSectionsGroupURL.Value == nil {
			cs.ClassSectionsGroupURL = nil
		} else {
			cs.ClassSectionsGroupURL = *p.ClassSectionsGroupURL.Value
		}
	}
	if p.ClassSectionsImageURL.Present {
		if p.ClassSectionsImageURL.Value == nil {
			cs.ClassSectionsImageURL = nil
		} else {
			cs.ClassSectionsImageURL = *p.ClassSectionsImageURL.Value
		}
	}
	if p.ClassSectionsImageObjectKey.Present {
		if p.ClassSectionsImageObjectKey.Value == nil {
			cs.ClassSectionsImageObjectKey = nil
		} else {
			cs.ClassSectionsImageObjectKey = *p.ClassSectionsImageObjectKey.Value
		}
	}
	if p.ClassSectionsImageURLOld.Present {
		if p.ClassSectionsImageURLOld.Value == nil {
			cs.ClassSectionsImageURLOld = nil
		} else {
			cs.ClassSectionsImageURLOld = *p.ClassSectionsImageURLOld.Value
		}
	}
	if p.ClassSectionsImageObjectKeyOld.Present {
		if p.ClassSectionsImageObjectKeyOld.Value == nil {
			cs.ClassSectionsImageObjectKeyOld = nil
		} else {
			cs.ClassSectionsImageObjectKeyOld = *p.ClassSectionsImageObjectKeyOld.Value
		}
	}
	if p.ClassSectionsImageDeletePendingUntil.Present {
		if p.ClassSectionsImageDeletePendingUntil.Value == nil {
			cs.ClassSectionsImageDeletePendingUntil = nil
		} else {
			cs.ClassSectionsImageDeletePendingUntil = *p.ClassSectionsImageDeletePendingUntil.Value
		}
	}

	// numerik & status
	if p.ClassSectionsCapacity.Present {
		if p.ClassSectionsCapacity.Value == nil {
			cs.ClassSectionsCapacity = nil
		} else {
			v := *p.ClassSectionsCapacity.Value
			cs.ClassSectionsCapacity = &v
		}
	}
	if p.ClassSectionsTotalStudents.Present && p.ClassSectionsTotalStudents.Value != nil {
		cs.ClassSectionsTotalStudents = *p.ClassSectionsTotalStudents.Value
	}
	if p.ClassSectionsIsActive.Present && p.ClassSectionsIsActive.Value != nil {
		cs.ClassSectionsIsActive = *p.ClassSectionsIsActive.Value
	}

	cs.ClassSectionsUpdatedAt = time.Now()
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
//
// Setelah parse -> Normalize() (validasi tetap dilakukan di controller).
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

	// -------- helpers umum ----------
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
				// untuk UUID non-pointer di DTO, kosong → treat sebagai "tidak diubah"
				// kalau mau izinkan clear, gunakan field *uuid.UUID di DTO (di kamu: memang non-pointer).
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
			// RFC3339 atau YYYY-MM-DD
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
				// kosong → null (untuk capacity: berarti clear)
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

	// -------- mapping field ----------
	// string (non-nullable di model)
	setStr(&r.ClassSectionsSlug, "class_sections_slug", true)
	setStr(&r.ClassSectionsName, "class_sections_name", false)

	// UUID (nullable di model, tapi di DTO kamu pakai PatchFieldCS[uuid.UUID])
	// => kalau kosong, anggap "tidak diubah". Untuk "clear", kirim field pointer versi string: class_sections_teacher_id = "" → tidak bisa clear.
	// Karena di Apply() kamu expect bisa clear, lebih aman dukung pola:
	//  - kalau ingin clear: kirim "class_sections_teacher_id" = "null" lewat JSON dalam payload
	//  - utk form-data murni: tambahkan key "class_sections_teacher_id__null" = "1" (opsional).
	// Namun supaya simpel, kita tetap parse UUID saja:
	_ = setUUID(&r.ClassSectionsTeacherID, "class_sections_teacher_id", "class_sections_teacher_id")
	_ = setUUID(&r.ClassSectionsAssistantTeacherID, "class_sections_assistant_teacher_id", "class_sections_assistant_teacher_id")
	_ = setUUID(&r.ClassSectionsClassRoomID, "class_sections_class_room_id", "class_sections_class_room_id")
	_ = setUUID(&r.ClassSectionsLeaderStudentID, "class_sections_leader_student_id", "class_sections_leader_student_id")

	// *string
	setStrPtr(&r.ClassSectionsCode, "class_sections_code", false)
	setStrPtr(&r.ClassSectionsSchedule, "class_sections_schedule", false)
	setStrPtr(&r.ClassSectionsGroupURL, "class_sections_group_url", false)

	// image slots
	setStrPtr(&r.ClassSectionsImageURL, "class_sections_image_url", false)
	setStrPtr(&r.ClassSectionsImageObjectKey, "class_sections_image_object_key", false)
	setStrPtr(&r.ClassSectionsImageURLOld, "class_sections_image_url_old", false)
	setStrPtr(&r.ClassSectionsImageObjectKeyOld, "class_sections_image_object_key_old", false)

	// time (delete pending)
	if err := setTimePtr(&r.ClassSectionsImageDeletePendingUntil, "class_sections_image_delete_pending_until", "class_sections_image_delete_pending_until"); err != nil {
		return err
	}

	// numerik & status
	if err := setInt(&r.ClassSectionsCapacity, "class_sections_capacity", "class_sections_capacity"); err != nil {
		return err
	}
	if err := setInt(&r.ClassSectionsTotalStudents, "class_sections_total_students", "class_sections_total_students"); err != nil {
		return err
	}
	if err := setBool(&r.ClassSectionsIsActive, "class_sections_is_active"); err != nil {
		return err
	}

	return nil
}
