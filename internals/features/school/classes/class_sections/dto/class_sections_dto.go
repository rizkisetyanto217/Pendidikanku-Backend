package dto

import (
	"encoding/json"
	"errors"
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
   CSST LITE PAYLOAD (untuk cache & listing)
   ========================================================= */

type CSSTItemLite struct {
	ID       string `json:"id"`
	IsActive bool   `json:"is_active"`

	Teacher struct {
		ID string `json:"id"`
	} `json:"teacher"`

	ClassSubject struct {
		ID      string `json:"id"`
		Subject struct {
			ID   string  `json:"id"`
			Name *string `json:"name,omitempty"`
		} `json:"subject"`
	} `json:"class_subject"`

	Room *struct {
		ID string `json:"id"`
	} `json:"room,omitempty"`

	GroupURL *string `json:"group_url,omitempty"`

	Stats *struct {
		TotalAttendance *int32 `json:"total_attendance,omitempty"`
	} `json:"stats,omitempty"`
}

/* =========================================================
   CREATE REQUEST (tanpa snapshot)
   ========================================================= */

type ClassSectionCreateRequest struct {
	// Wajib
	ClassSectionMasjidID uuid.UUID `json:"class_section_masjid_id" form:"class_section_masjid_id" validate:"required"`
	ClassSectionClassID  uuid.UUID `json:"class_section_class_id"  form:"class_section_class_id"  validate:"required"`
	ClassSectionSlug     string    `json:"class_section_slug"      form:"class_section_slug"      validate:"min=1,max=160"`
	ClassSectionName     string    `json:"class_section_name"      form:"class_section_name"      validate:"required,min=1,max=100"`

	// Opsional (bukan snapshot)
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

/* =========================================================
   RESPONSE (read-only snapshots + CSST LITE)
   ========================================================= */

type ClassSectionResponse struct {
	// Identitas & relasi dasar
	ClassSectionID       uuid.UUID `json:"class_section_id"`
	ClassSectionMasjidID uuid.UUID `json:"class_section_masjid_id"`

	ClassSectionClassID            uuid.UUID  `json:"class_section_class_id"`
	ClassSectionTeacherID          *uuid.UUID `json:"class_section_teacher_id"`
	ClassSectionAssistantTeacherID *uuid.UUID `json:"class_section_assistant_teacher_id"`
	ClassSectionClassRoomID        *uuid.UUID `json:"class_section_class_room_id"`
	ClassSectionLeaderStudentID    *uuid.UUID `json:"class_section_leader_student_id"`

	// Properti editable
	ClassSectionSlug string  `json:"class_section_slug"`
	ClassSectionName string  `json:"class_section_name"`
	ClassSectionCode *string `json:"class_section_code"`

	ClassSectionSchedule *string `json:"class_section_schedule"`

	ClassSectionCapacity      *int `json:"class_section_capacity"`
	ClassSectionTotalStudents int  `json:"class_section_total_students"`

	ClassSectionGroupURL *string `json:"class_section_group_url"`

	// Image (editable)
	ClassSectionImageURL                *string    `json:"class_section_image_url"`
	ClassSectionImageObjectKey          *string    `json:"class_section_image_object_key"`
	ClassSectionImageURLOld             *string    `json:"class_section_image_url_old"`
	ClassSectionImageObjectKeyOld       *string    `json:"class_section_image_object_key_old"`
	ClassSectionImageDeletePendingUntil *time.Time `json:"class_section_image_delete_pending_until"`

	// Status & audit
	ClassSectionIsActive  bool       `json:"class_section_is_active"`
	ClassSectionCreatedAt time.Time  `json:"class_section_created_at"`
	ClassSectionUpdatedAt time.Time  `json:"class_section_updated_at"`
	ClassSectionDeletedAt *time.Time `json:"class_section_deleted_at,omitempty"`

	// ================== SNAPSHOTS (READ-ONLY) ==================

	// class & parent/teacher/leader
	ClassSectionClassSlugSnapshot *string `json:"class_section_class_slug_snapshot,omitempty"`

	// Parent snapshots (lengkap)
	ClassSectionParentNameSnapshot  *string `json:"class_section_parent_name_snapshot,omitempty"`
	ClassSectionParentCodeSnapshot  *string `json:"class_section_parent_code_snapshot,omitempty"`
	ClassSectionParentSlugSnapshot  *string `json:"class_section_parent_slug_snapshot,omitempty"`
	ClassSectionParentLevelSnapshot *string `json:"class_section_parent_level_snapshot,omitempty"`
	ClassSectionParentURLSnapshot   *string `json:"class_section_parent_url_snapshot,omitempty"`

	ClassSectionTeacherNameSnapshot          *string `json:"class_section_teacher_name_snapshot,omitempty"`
	ClassSectionAssistantTeacherNameSnapshot *string `json:"class_section_assistant_teacher_name_snapshot,omitempty"`
	ClassSectionLeaderStudentNameSnapshot    *string `json:"class_section_leader_student_name_snapshot,omitempty"`

	// kontak (snapshot)
	ClassSectionTeacherContactPhoneSnapshot          *string `json:"class_section_teacher_contact_phone_snapshot,omitempty"`
	ClassSectionAssistantTeacherContactPhoneSnapshot *string `json:"class_section_assistant_teacher_contact_phone_snapshot,omitempty"`
	ClassSectionLeaderStudentContactPhoneSnapshot    *string `json:"class_section_leader_student_contact_phone_snapshot,omitempty"`

	// avatar (snapshot baru)
	ClassSectionTeacherAvatarURLSnapshot          *string `json:"class_section_teacher_avatar_url_snapshot,omitempty"`
	ClassSectionAssistantTeacherAvatarURLSnapshot *string `json:"class_section_assistant_teacher_avatar_url_snapshot,omitempty"`

	// ROOM snapshots
	ClassSectionRoomNameSnapshot     *string `json:"class_section_room_name_snapshot,omitempty"`
	ClassSectionRoomSlugSnapshot     *string `json:"class_section_room_slug_snapshot,omitempty"`
	ClassSectionRoomLocationSnapshot *string `json:"class_section_room_location_snapshot,omitempty"`

	// housekeeping snapshot
	ClassSectionSnapshotUpdatedAt *time.Time `json:"class_section_snapshot_updated_at,omitempty"`

	// TERM (lean snapshots)
	ClassSectionTermID                *uuid.UUID `json:"class_section_term_id,omitempty"`
	ClassSectionTermNameSnapshot      *string    `json:"class_section_term_name_snapshot,omitempty"`
	ClassSectionTermSlugSnapshot      *string    `json:"class_section_term_slug_snapshot,omitempty"`
	ClassSectionTermYearLabelSnapshot *string    `json:"class_section_term_year_label_snapshot,omitempty"`

	// ============== CSST LITE (cache untuk listing) ============
	ClassSectionsCSST []CSSTItemLite `json:"class_sections_csst"`
}

func FromModelClassSection(cs *m.ClassSectionModel) ClassSectionResponse {
	var deletedAt *time.Time
	// Catatan: model kamu menggunakan sql.NullTime; sesuaikan aksesnya:
	if cs.ClassSectionDeletedAt.Valid {
		t := cs.ClassSectionDeletedAt.Time
		deletedAt = &t
	}

	// Unmarshal CSST lite dari JSONB; jika gagal, fallback ke [] (bukan error)
	var csstLite []CSSTItemLite
	if len(cs.ClassSectionsCSST) > 0 {
		_ = json.Unmarshal(cs.ClassSectionsCSST, &csstLite)
	}

	return ClassSectionResponse{
		// identitas
		ClassSectionID:       cs.ClassSectionID,
		ClassSectionMasjidID: cs.ClassSectionMasjidID,

		ClassSectionClassID:            cs.ClassSectionClassID,
		ClassSectionTeacherID:          cs.ClassSectionTeacherID,
		ClassSectionAssistantTeacherID: cs.ClassSectionAssistantTeacherID,
		ClassSectionClassRoomID:        cs.ClassSectionClassRoomID,
		ClassSectionLeaderStudentID:    cs.ClassSectionLeaderStudentID,

		// editable
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

		// snapshots (read-only)
		ClassSectionClassSlugSnapshot: cs.ClassSectionClassSlugSnapshot,

		// parent (lengkap)
		ClassSectionParentNameSnapshot:  cs.ClassSectionParentNameSnapshot,
		ClassSectionParentCodeSnapshot:  cs.ClassSectionParentCodeSnapshot,
		ClassSectionParentSlugSnapshot:  cs.ClassSectionParentSlugSnapshot,
		ClassSectionParentLevelSnapshot: cs.ClassSectionParentLevelSnapshot,
		ClassSectionParentURLSnapshot:   cs.ClassSectionParentURLSnapshot,

		// nama orang
		ClassSectionTeacherNameSnapshot:          cs.ClassSectionTeacherNameSnapshot,
		ClassSectionAssistantTeacherNameSnapshot: cs.ClassSectionAssistantTeacherNameSnapshot,
		ClassSectionLeaderStudentNameSnapshot:    cs.ClassSectionLeaderStudentNameSnapshot,

		// avatar
		ClassSectionTeacherAvatarURLSnapshot:          cs.ClassSectionTeacherAvatarURLSnapshot,
		ClassSectionAssistantTeacherAvatarURLSnapshot: cs.ClassSectionAssistantTeacherAvatarURLSnapshot,

		// kontak
		ClassSectionTeacherContactPhoneSnapshot:          cs.ClassSectionTeacherContactPhoneSnapshot,
		ClassSectionAssistantTeacherContactPhoneSnapshot: cs.ClassSectionAssistantTeacherContactPhoneSnapshot,
		ClassSectionLeaderStudentContactPhoneSnapshot:    cs.ClassSectionLeaderStudentContactPhoneSnapshot,

		// room
		ClassSectionRoomNameSnapshot:     cs.ClassSectionRoomNameSnapshot,
		ClassSectionRoomSlugSnapshot:     cs.ClassSectionRoomSlugSnapshot,
		ClassSectionRoomLocationSnapshot: cs.ClassSectionRoomLocationSnapshot,

		// housekeeping
		ClassSectionSnapshotUpdatedAt: cs.ClassSectionSnapshotUpdatedAt,

		// term
		ClassSectionTermID:                cs.ClassSectionTermID,
		ClassSectionTermNameSnapshot:      cs.ClassSectionTermNameSnapshot,
		ClassSectionTermSlugSnapshot:      cs.ClassSectionTermSlugSnapshot,
		ClassSectionTermYearLabelSnapshot: cs.ClassSectionTermYearLabelSnapshot,

		// csst lite
		ClassSectionsCSST: csstLite,
	}
}

/* =========================================================
   PATCH REQUEST (tri-state via PatchFieldCS[T])
   - Present=true, Value=nil  -> set NULL (untuk kolom pointer)
   - Present=true, Value=val  -> set val
   - Present=false            -> skip
========================================================= */

type ClassSectionPatchRequest struct {
	// Relasi opsional
	ClassSectionTeacherID          PatchFieldCS[uuid.UUID] `json:"class_section_teacher_id"`
	ClassSectionAssistantTeacherID PatchFieldCS[uuid.UUID] `json:"class_section_assistant_teacher_id"`
	ClassSectionClassRoomID        PatchFieldCS[uuid.UUID] `json:"class_section_class_room_id"`
	ClassSectionLeaderStudentID    PatchFieldCS[uuid.UUID] `json:"class_section_leader_student_id"`

	// Properti editable
	ClassSectionSlug     PatchFieldCS[string] `json:"class_section_slug"`
	ClassSectionName     PatchFieldCS[string] `json:"class_section_name"`
	ClassSectionCode     PatchFieldCS[string] `json:"class_section_code"`
	ClassSectionSchedule PatchFieldCS[string] `json:"class_section_schedule"`

	ClassSectionCapacity      PatchFieldCS[int] `json:"class_section_capacity"`
	ClassSectionTotalStudents PatchFieldCS[int] `json:"class_section_total_students"`

	ClassSectionGroupURL PatchFieldCS[string] `json:"class_section_group_url"`

	// Image meta (biasanya di-set via upload; tetap disediakan untuk konsistensi)
	ClassSectionImageURL       PatchFieldCS[string] `json:"class_section_image_url"`
	ClassSectionImageObjectKey PatchFieldCS[string] `json:"class_section_image_object_key"`

	// Status
	ClassSectionIsActive PatchFieldCS[bool] `json:"class_section_is_active"`
}

/* =========================================================
   Apply ke model (mutasi in-place)
========================================================= */

func (r *ClassSectionPatchRequest) Apply(cs *m.ClassSectionModel) {
	// Helper setter untuk pointer kolom
	setUUIDPtr := func(f PatchFieldCS[uuid.UUID], dst **uuid.UUID) {
		if !f.Present {
			return
		}
		if f.Value == nil {
			*dst = nil
			return
		}
		v := *f.Value
		*dst = &v
	}
	setStrPtr := func(f PatchFieldCS[string], dst **string, doLower bool) {
		if !f.Present {
			return
		}
		if f.Value == nil {
			*dst = nil
			return
		}
		v := strings.TrimSpace(*f.Value)
		if v == "" {
			*dst = nil
			return
		}
		if doLower {
			v = strings.ToLower(v)
		}
		*dst = &v
	}
	setIntPtr := func(f PatchFieldCS[int], dst **int) {
		if !f.Present {
			return
		}
		if f.Value == nil {
			*dst = nil
			return
		}
		v := *f.Value
		*dst = &v
	}

	// Relasi
	setUUIDPtr(r.ClassSectionTeacherID, &cs.ClassSectionTeacherID)
	setUUIDPtr(r.ClassSectionAssistantTeacherID, &cs.ClassSectionAssistantTeacherID)
	setUUIDPtr(r.ClassSectionClassRoomID, &cs.ClassSectionClassRoomID)
	setUUIDPtr(r.ClassSectionLeaderStudentID, &cs.ClassSectionLeaderStudentID)

	// String non-pointer (slug, name)
	if r.ClassSectionSlug.Present && r.ClassSectionSlug.Value != nil {
		cs.ClassSectionSlug = strings.ToLower(strings.TrimSpace(*r.ClassSectionSlug.Value))
	}
	if r.ClassSectionName.Present && r.ClassSectionName.Value != nil {
		cs.ClassSectionName = strings.TrimSpace(*r.ClassSectionName.Value)
	}

	// String pointer
	setStrPtr(r.ClassSectionCode, &cs.ClassSectionCode, false)
	setStrPtr(r.ClassSectionSchedule, &cs.ClassSectionSchedule, false)
	setStrPtr(r.ClassSectionGroupURL, &cs.ClassSectionGroupURL, false)

	// Kapasitas & total students (pointer & non-pointer)
	setIntPtr(r.ClassSectionCapacity, &cs.ClassSectionCapacity)
	if r.ClassSectionTotalStudents.Present && r.ClassSectionTotalStudents.Value != nil {
		cs.ClassSectionTotalStudents = *r.ClassSectionTotalStudents.Value
	}

	// Image meta optional
	setStrPtr(r.ClassSectionImageURL, &cs.ClassSectionImageURL, false)
	setStrPtr(r.ClassSectionImageObjectKey, &cs.ClassSectionImageObjectKey, false)

	// Status
	if r.ClassSectionIsActive.Present && r.ClassSectionIsActive.Value != nil {
		cs.ClassSectionIsActive = *r.ClassSectionIsActive.Value
	}
}

/* =========================================================
   Decoder: JSON or multipart/form-data
   - multipart hanya mengisi field yang ada di form
   - JSON pakai BodyParser standar (PatchFieldCS via UnmarshalJSON)
========================================================= */

func DecodePatchClassSectionFromRequest(c *fiber.Ctx, dst *ClassSectionPatchRequest) error {
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	switch {
	case strings.HasPrefix(ct, "application/json"):
		if err := c.BodyParser(dst); err != nil {
			return errors.New("payload JSON tidak valid")
		}
		return nil

	case strings.HasPrefix(ct, "multipart/form-data"):
		// Helper untuk tandai present + isi value
		markUUID := func(key string, pf *PatchFieldCS[uuid.UUID]) {
			if v := strings.TrimSpace(c.FormValue(key)); v != "" {
				pf.Present = true
				if strings.EqualFold(v, "null") {
					pf.Value = nil
					return
				}
				if id, err := uuid.Parse(v); err == nil {
					val := id
					pf.Value = &val
				} else {
					pf.Value = nil // biar ketangkap validasi di controller bila perlu
				}
			}
		}
		markStr := func(key string, pf *PatchFieldCS[string]) {
			if v := c.FormValue(key); v != "" || formHasKey(c, key) {
				pf.Present = true
				if strings.EqualFold(strings.TrimSpace(v), "null") {
					pf.Value = nil
					return
				}
				val := v
				pf.Value = &val
			}
		}
		markInt := func(key string, pf *PatchFieldCS[int]) {
			if v := c.FormValue(key); v != "" || formHasKey(c, key) {
				pf.Present = true
				if strings.EqualFold(strings.TrimSpace(v), "null") {
					pf.Value = nil
					return
				}
				iv, err := strconv.Atoi(strings.TrimSpace(v))
				if err != nil {
					// tetap tandai present, biar controller bisa balas 400 jika mau
					pf.Value = nil
					return
				}
				pf.Value = &iv
			}
		}
		markBool := func(key string, pf *PatchFieldCS[bool]) {
			if v := c.FormValue(key); v != "" || formHasKey(c, key) {
				pf.Present = true
				if strings.EqualFold(strings.TrimSpace(v), "null") {
					pf.Value = nil
					return
				}
				lv := strings.ToLower(strings.TrimSpace(v))
				b := lv == "1" || lv == "true" || lv == "ya" || lv == "yes"
				pf.Value = &b
			}
		}

		// Map form fields -> patch fields (pakai nama JSON yang sama)
		markUUID("class_section_teacher_id", &dst.ClassSectionTeacherID)
		markUUID("class_section_assistant_teacher_id", &dst.ClassSectionAssistantTeacherID)
		markUUID("class_section_class_room_id", &dst.ClassSectionClassRoomID)
		markUUID("class_section_leader_student_id", &dst.ClassSectionLeaderStudentID)

		markStr("class_section_slug", &dst.ClassSectionSlug)
		markStr("class_section_name", &dst.ClassSectionName)
		markStr("class_section_code", &dst.ClassSectionCode)
		markStr("class_section_schedule", &dst.ClassSectionSchedule)
		markStr("class_section_group_url", &dst.ClassSectionGroupURL)

		markInt("class_section_capacity", &dst.ClassSectionCapacity)
		markInt("class_section_total_students", &dst.ClassSectionTotalStudents)

		markStr("class_section_image_url", &dst.ClassSectionImageURL)
		markStr("class_section_image_object_key", &dst.ClassSectionImageObjectKey)

		markBool("class_section_is_active", &dst.ClassSectionIsActive)

		return nil

	default:
		// fallback: coba JSON
		if err := c.BodyParser(dst); err != nil {
			return errors.New("Content-Type tidak didukung; gunakan application/json atau multipart/form-data")
		}
		return nil
	}
}

// formHasKey mengecek eksistensi key pada multipart form (meski kosong)
func formHasKey(c *fiber.Ctx, key string) bool {
	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return false
	}
	_, ok := form.Value[key]
	return ok
}
