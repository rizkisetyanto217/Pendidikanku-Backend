// file: internals/features/school/classes/class_sections/dto/class_section_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	// models
	m "masjidku_backend/internals/features/school/classes/class_sections/model"
	csstModel "masjidku_backend/internals/features/school/academics/subject/model"
)

/* =========================================================
   Helpers (trim)
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
func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

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
   ==================  C L A S S   S E C T I O N  ==================
========================================================= */
/* ----------------- CREATE REQUEST ----------------- */

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

	// ========== Konfigurasi CSST (baru) ==========
	// enum string: "self_select" | "assigned" | "hybrid"
	ClassSectionCSSTEnrollmentMode        *string    `json:"class_section_csst_enrollment_mode" form:"class_section_csst_enrollment_mode"`
	ClassSectionCSSTRequiresApproval      *bool      `json:"class_section_csst_self_select_requires_approval" form:"class_section_csst_self_select_requires_approval"`
	ClassSectionCSSTMaxSubjectsPerStudent *int       `json:"class_section_csst_max_subjects_per_student" form:"class_section_csst_max_subjects_per_student"`
	ClassSectionCSSTSwitchDeadline        *time.Time `json:"class_section_csst_switch_deadline" form:"class_section_csst_switch_deadline"`

	// JSON object, contoh: {"allow_chat":true}
	ClassSectionFeatures *json.RawMessage `json:"class_section_features" form:"class_section_features"`
}

func (r *ClassSectionCreateRequest) Normalize() {
	trimPP := func(pp **string, lower bool) {
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
	trimPP(&r.ClassSectionCode, false)
	trimPP(&r.ClassSectionSchedule, false)
	trimPP(&r.ClassSectionGroupURL, false)
	trimPP(&r.ClassSectionImageURL, false)
	trimPP(&r.ClassSectionImageObjectKey, false)

	if r.ClassSectionCSSTEnrollmentMode != nil {
		v := strings.ToLower(strings.TrimSpace(*r.ClassSectionCSSTEnrollmentMode))
		r.ClassSectionCSSTEnrollmentMode = &v
	}
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

	// Konfigurasi CSST
	if r.ClassSectionCSSTEnrollmentMode != nil {
		switch strings.ToLower(*r.ClassSectionCSSTEnrollmentMode) {
		case "self_select":
			cs.ClassSectionCSSTEnrollmentMode = m.CSSTEnrollSelfSelect
		case "assigned":
			cs.ClassSectionCSSTEnrollmentMode = m.CSSTEnrollAssigned
		case "hybrid":
			cs.ClassSectionCSSTEnrollmentMode = m.CSSTEnrollHybrid
		}
	}
	if r.ClassSectionCSSTRequiresApproval != nil {
		cs.ClassSectionCSSTSelfSelectRequiresApproval = *r.ClassSectionCSSTRequiresApproval
	}
	if r.ClassSectionCSSTMaxSubjectsPerStudent != nil {
		cs.ClassSectionCSSTMaxSubjectsPerStudent = r.ClassSectionCSSTMaxSubjectsPerStudent
	}
	if r.ClassSectionCSSTSwitchDeadline != nil {
		cs.ClassSectionCSSTSwitchDeadline = r.ClassSectionCSSTSwitchDeadline
	}
	if r.ClassSectionFeatures != nil && len(*r.ClassSectionFeatures) > 0 {
		var tmp any
		if err := json.Unmarshal(*r.ClassSectionFeatures, &tmp); err == nil {
			cs.ClassSectionFeatures = datatypes.JSON(*r.ClassSectionFeatures) // <-- konversi
		}
	}
	return cs
}

/* ----------------- RESPONSE ----------------- */

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

	// Image
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

	// ================== SNAPSHOTS (READ-ONLY via *_snap) ==================
	ClassSectionClassSlugSnap *string `json:"class_section_class_slug_snap,omitempty"`

	// Parent
	ClassSectionParentNameSnap  *string `json:"class_section_parent_name_snap,omitempty"`
	ClassSectionParentCodeSnap  *string `json:"class_section_parent_code_snap,omitempty"`
	ClassSectionParentSlugSnap  *string `json:"class_section_parent_slug_snap,omitempty"`
	ClassSectionParentLevelSnap *string `json:"class_section_parent_level_snap,omitempty"`

	// People (derived names)
	ClassSectionTeacherNameSnap          *string `json:"class_section_teacher_name_snap,omitempty"`
	ClassSectionAssistantTeacherNameSnap *string `json:"class_section_assistant_teacher_name_snap,omitempty"`
	ClassSectionLeaderStudentNameSnap    *string `json:"class_section_leader_student_name_snap,omitempty"`

	// Room
	ClassSectionRoomNameSnap     *string `json:"class_section_room_name_snap,omitempty"`
	ClassSectionRoomSlugSnap     *string `json:"class_section_room_slug_snap,omitempty"`
	ClassSectionRoomLocationSnap *string `json:"class_section_room_location_snap,omitempty"`

	// TERM
	ClassSectionTermID        *uuid.UUID `json:"class_section_term_id,omitempty"`
	ClassSectionTermNameSnap  *string    `json:"class_section_term_name_snap,omitempty"`
	ClassSectionTermSlugSnap  *string    `json:"class_section_term_slug_snap,omitempty"`
	ClassSectionTermYearLabel *string    `json:"class_section_term_year_label_snap,omitempty"`

	// housekeeping snapshot
	ClassSectionSnapshotUpdatedAt *time.Time `json:"class_section_snapshot_updated_at,omitempty"`

	// ================== RAW SNAPSHOTS (JSONB asli) — NEW ==================
	ClassSectionClassSnapshot            json.RawMessage `json:"class_section_class_snapshot,omitempty"`
	ClassSectionParentSnapshot           json.RawMessage `json:"class_section_parent_snapshot,omitempty"`
	ClassSectionTermSnapshot             json.RawMessage `json:"class_section_term_snapshot,omitempty"`
	ClassSectionTeacherSnapshot          json.RawMessage `json:"class_section_teacher_snapshot,omitempty"`           // <— penting
	ClassSectionAssistantTeacherSnapshot json.RawMessage `json:"class_section_assistant_teacher_snapshot,omitempty"` // <— penting
	ClassSectionLeaderStudentSnapshot    json.RawMessage `json:"class_section_leader_student_snapshot,omitempty"`
	ClassSectionRoomSnapshot             json.RawMessage `json:"class_section_room_snapshot,omitempty"`

	// ============== CSST LITE & COUNTS ============
	ClassSectionsCSST            []CSSTItemLite `json:"class_sections_csst"`
	ClassSectionsCSSTCount       int            `json:"class_sections_csst_count"`
	ClassSectionsCSSTActiveCount int            `json:"class_sections_csst_active_count"`

	// ============== CSST SETTINGS (echo) ==========
	ClassSectionCSSTEnrollmentMode             string     `json:"class_section_csst_enrollment_mode"`
	ClassSectionCSSTSelfSelectRequiresApproval bool       `json:"class_section_csst_self_select_requires_approval"`
	ClassSectionCSSTMaxSubjectsPerStudent      *int       `json:"class_section_csst_max_subjects_per_student,omitempty"`
	ClassSectionCSSTSwitchDeadline             *time.Time `json:"class_section_csst_switch_deadline,omitempty"`

	// features (JSON object)
	ClassSectionFeatures json.RawMessage `json:"class_section_features"`
}

func FromModelClassSection(cs *m.ClassSectionModel) ClassSectionResponse {
	var deletedAt *time.Time
	if cs.ClassSectionDeletedAt.Valid {
		t := cs.ClassSectionDeletedAt.Time
		deletedAt = &t
	}

	// Unmarshal CSST lite dari JSONB (ignore error, fallback [])
	var csstLite []CSSTItemLite
	if len(cs.ClassSectionsCSST) > 0 {
		_ = json.Unmarshal(cs.ClassSectionsCSST, &csstLite)
	}

	// features JSON
	var features json.RawMessage
	if len(cs.ClassSectionFeatures) > 0 {
		features = json.RawMessage(cs.ClassSectionFeatures)
	} else {
		features = json.RawMessage(`{}`)
	}

	// helper: konversi datatypes.JSON → json.RawMessage (kosong → nil)
	toRaw := func(j datatypes.JSON) json.RawMessage {
		if len(j) == 0 {
			return nil
		}
		return json.RawMessage(j)
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

		// snapshots (derived, read-only)
		ClassSectionClassSlugSnap: cs.ClassSectionClassSlugSnap,

		ClassSectionParentNameSnap:  cs.ClassSectionParentNameSnap,
		ClassSectionParentCodeSnap:  cs.ClassSectionParentCodeSnap,
		ClassSectionParentSlugSnap:  cs.ClassSectionParentSlugSnap,
		ClassSectionParentLevelSnap: cs.ClassSectionParentLevelSnap,

		ClassSectionTeacherNameSnap:          cs.ClassSectionTeacherNameSnap,
		ClassSectionAssistantTeacherNameSnap: cs.ClassSectionAssistantTeacherNameSnap,
		ClassSectionLeaderStudentNameSnap:    cs.ClassSectionLeaderStudentNameSnap,

		ClassSectionRoomNameSnap:     cs.ClassSectionRoomNameSnap,
		ClassSectionRoomSlugSnap:     cs.ClassSectionRoomSlugSnap,
		ClassSectionRoomLocationSnap: cs.ClassSectionRoomLocationSnap,

		ClassSectionTermID:        cs.ClassSectionTermID,
		ClassSectionTermNameSnap:  cs.ClassSectionTermNameSnap,
		ClassSectionTermSlugSnap:  cs.ClassSectionTermSlugSnap,
		ClassSectionTermYearLabel: cs.ClassSectionTermYearLabel, // <— konsisten *_snap

		ClassSectionSnapshotUpdatedAt: cs.ClassSectionSnapshotUpdatedAt,

		// RAW snapshots (JSONB asli)
		ClassSectionClassSnapshot:            toRaw(cs.ClassSectionClassSnapshot),
		ClassSectionParentSnapshot:           toRaw(cs.ClassSectionParentSnapshot),
		ClassSectionTermSnapshot:             toRaw(cs.ClassSectionTermSnapshot),
		ClassSectionTeacherSnapshot:          toRaw(cs.ClassSectionTeacherSnapshot),
		ClassSectionAssistantTeacherSnapshot: toRaw(cs.ClassSectionAssistantTeacherSnapshot),
		ClassSectionLeaderStudentSnapshot:    toRaw(cs.ClassSectionLeaderStudentSnapshot),
		ClassSectionRoomSnapshot:             toRaw(cs.ClassSectionRoomSnapshot),

		// CSST lite & counts
		ClassSectionsCSST:            csstLite,
		ClassSectionsCSSTCount:       cs.ClassSectionsCSSTCount,
		ClassSectionsCSSTActiveCount: cs.ClassSectionsCSSTActiveCount,

		// CSST settings
		ClassSectionCSSTEnrollmentMode:             cs.ClassSectionCSSTEnrollmentMode.String(),
		ClassSectionCSSTSelfSelectRequiresApproval: cs.ClassSectionCSSTSelfSelectRequiresApproval,
		ClassSectionCSSTMaxSubjectsPerStudent:      cs.ClassSectionCSSTMaxSubjectsPerStudent,
		ClassSectionCSSTSwitchDeadline:             cs.ClassSectionCSSTSwitchDeadline,

		// features
		ClassSectionFeatures: features,
	}
}

/* ----------------- PATCH REQUEST ----------------- */

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

	// Image meta
	ClassSectionImageURL       PatchFieldCS[string] `json:"class_section_image_url"`
	ClassSectionImageObjectKey PatchFieldCS[string] `json:"class_section_image_object_key"`

	// Status
	ClassSectionIsActive PatchFieldCS[bool] `json:"class_section_is_active"`

	// ====== CSST settings ======
	ClassSectionCSSTEnrollmentMode             PatchFieldCS[string]          `json:"class_section_csst_enrollment_mode"`               // "self_select"|"assigned"|"hybrid"
	ClassSectionCSSTSelfSelectRequiresApproval PatchFieldCS[bool]            `json:"class_section_csst_self_select_requires_approval"` // true/false
	ClassSectionCSSTMaxSubjectsPerStudent      PatchFieldCS[int]             `json:"class_section_csst_max_subjects_per_student"`
	ClassSectionCSSTSwitchDeadline             PatchFieldCS[time.Time]       `json:"class_section_csst_switch_deadline"`
	ClassSectionFeatures                       PatchFieldCS[json.RawMessage] `json:"class_section_features"` // JSON object
}

/* ----------------- Apply PATCH ----------------- */

func (r *ClassSectionPatchRequest) Apply(cs *m.ClassSectionModel) {
	// Helpers
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

	// Kapasitas & total students
	setIntPtr(r.ClassSectionCapacity, &cs.ClassSectionCapacity)
	if r.ClassSectionTotalStudents.Present && r.ClassSectionTotalStudents.Value != nil {
		cs.ClassSectionTotalStudents = *r.ClassSectionTotalStudents.Value
	}

	// Image meta
	setStrPtr(r.ClassSectionImageURL, &cs.ClassSectionImageURL, false)
	setStrPtr(r.ClassSectionImageObjectKey, &cs.ClassSectionImageObjectKey, false)

	// Status
	if r.ClassSectionIsActive.Present && r.ClassSectionIsActive.Value != nil {
		cs.ClassSectionIsActive = *r.ClassSectionIsActive.Value
	}

	// ====== CSST settings ======
	if r.ClassSectionCSSTEnrollmentMode.Present {
		if r.ClassSectionCSSTEnrollmentMode.Value == nil {
			// kosongkan → biarkan nilai DB apa adanya
		} else {
			switch strings.ToLower(strings.TrimSpace(*r.ClassSectionCSSTEnrollmentMode.Value)) {
			case "self_select":
				cs.ClassSectionCSSTEnrollmentMode = m.CSSTEnrollSelfSelect
			case "assigned":
				cs.ClassSectionCSSTEnrollmentMode = m.CSSTEnrollAssigned
			case "hybrid":
				cs.ClassSectionCSSTEnrollmentMode = m.CSSTEnrollHybrid
			}
		}
	}
	if r.ClassSectionCSSTSelfSelectRequiresApproval.Present && r.ClassSectionCSSTSelfSelectRequiresApproval.Value != nil {
		cs.ClassSectionCSSTSelfSelectRequiresApproval = *r.ClassSectionCSSTSelfSelectRequiresApproval.Value
	}
	if r.ClassSectionCSSTMaxSubjectsPerStudent.Present {
		setIntPtr(r.ClassSectionCSSTMaxSubjectsPerStudent, &cs.ClassSectionCSSTMaxSubjectsPerStudent)
	}
	if r.ClassSectionCSSTSwitchDeadline.Present {
		if r.ClassSectionCSSTSwitchDeadline.Value == nil {
			cs.ClassSectionCSSTSwitchDeadline = nil
		} else {
			t := *r.ClassSectionCSSTSwitchDeadline.Value
			cs.ClassSectionCSSTSwitchDeadline = &t
		}
	}
	if r.ClassSectionFeatures.Present {
		if r.ClassSectionFeatures.Value == nil || len(*r.ClassSectionFeatures.Value) == 0 {
			cs.ClassSectionFeatures = datatypes.JSON([]byte(`{}`)) // kosong: object {}
		} else {
			var tmp any
			if err := json.Unmarshal(*r.ClassSectionFeatures.Value, &tmp); err == nil {
				cs.ClassSectionFeatures = datatypes.JSON(*r.ClassSectionFeatures.Value)
			}
		}
	}
}

/* ----------------- Decoder PATCH ----------------- */

func DecodePatchClassSectionFromRequest(c *fiber.Ctx, dst *ClassSectionPatchRequest) error {
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	switch {
	case strings.HasPrefix(ct, "application/json"):
		if err := c.BodyParser(dst); err != nil {
			return errors.New("payload JSON tidak valid")
		}
		return nil

	case strings.HasPrefix(ct, "multipart/form-data"):
		markUUID := func(key string, pf *PatchFieldCS[uuid.UUID]) {
			if v := strings.TrimSpace(c.FormValue(key)); v != "" || formHasKey(c, key) {
				pf.Present = true
				if strings.EqualFold(v, "null") || v == "" {
					pf.Value = nil
					return
				}
				if id, err := uuid.Parse(v); err == nil {
					val := id
					pf.Value = &val
				} else {
					pf.Value = nil
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
				if strings.EqualFold(strings.TrimSpace(v), "null") || strings.TrimSpace(v) == "" {
					pf.Value = nil
					return
				}
				iv, err := strconv.Atoi(strings.TrimSpace(v))
				if err != nil {
					pf.Value = nil
					return
				}
				pf.Value = &iv
			}
		}
		markBool := func(key string, pf *PatchFieldCS[bool]) {
			if v := c.FormValue(key); v != "" || formHasKey(c, key) {
				pf.Present = true
				if strings.EqualFold(strings.TrimSpace(v), "null") || strings.TrimSpace(v) == "" {
					pf.Value = nil
					return
				}
				lv := strings.ToLower(strings.TrimSpace(v))
				b := lv == "1" || lv == "true" || lv == "on" || lv == "yes" || lv == "y"
				pf.Value = &b
			}
		}
		markTime := func(key string, pf *PatchFieldCS[time.Time]) error {
			if v := c.FormValue(key); v != "" || formHasKey(c, key) {
				pf.Present = true
				s := strings.TrimSpace(v)
				if s == "" || strings.EqualFold(s, "null") {
					pf.Value = nil
					return nil
				}
				// RFC3339 atau YYYY-MM-DD
				if t, e := time.Parse(time.RFC3339, s); e == nil {
					pf.Value = &t
					return nil
				}
				if t, e := time.Parse("2006-01-02", s); e == nil {
					pf.Value = &t
					return nil
				}
				return fiber.NewError(fiber.StatusBadRequest, key+" format invalid (pakai RFC3339 atau YYYY-MM-DD)")
			}
			return nil
		}
		markJSON := func(key string, pf *PatchFieldCS[json.RawMessage]) error {
			if v := c.FormValue(key); v != "" || formHasKey(c, key) {
				pf.Present = true
				s := strings.TrimSpace(v)
				if s == "" || strings.EqualFold(s, "null") {
					pf.Value = nil
					return nil
				}
				// validasi json object/array apapun
				var tmp any
				if err := json.Unmarshal([]byte(s), &tmp); err != nil {
					return fiber.NewError(fiber.StatusBadRequest, key+" harus JSON valid")
				}
				raw := json.RawMessage(s)
				pf.Value = &raw
			}
			return nil
		}

		// Map form fields -> patch fields
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

		// CSST settings
		markStr("class_section_csst_enrollment_mode", &dst.ClassSectionCSSTEnrollmentMode)
		markBool("class_section_csst_self_select_requires_approval", &dst.ClassSectionCSSTSelfSelectRequiresApproval)
		markInt("class_section_csst_max_subjects_per_student", &dst.ClassSectionCSSTMaxSubjectsPerStudent)
		if err := markTime("class_section_csst_switch_deadline", &dst.ClassSectionCSSTSwitchDeadline); err != nil {
			return err
		}
		if err := markJSON("class_section_features", &dst.ClassSectionFeatures); err != nil {
			return err
		}

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

/*
Dipakai oleh controller:
- ClassSectionJoinRequest (Normalize, Validate)
- JoinRole + konstanta JoinRoleStudent / JoinRoleTeacher
- ClassSectionJoinResponse (memuat UserClassSectionResp dari DTO lain)
*/

// Peran saat join
type JoinRole string

const (
	JoinRoleStudent JoinRole = "student"
	JoinRoleTeacher JoinRole = "teacher"
)

/* ----------------- REQUEST: JOIN (student only) ----------------- */

type ClassSectionJoinRequest struct {
	Code           string    `json:"code"`             // kode join input siswa (case-sensitive)
	ClassSectionID uuid.UUID `json:"class_section_id"` // section target
}

func (r *ClassSectionJoinRequest) Normalize() {
	r.Code = strings.TrimSpace(r.Code) // JANGAN lower(); bcrypt case-sensitive
}

func (r *ClassSectionJoinRequest) Validate() error {
	if r.Code == "" {
		return errors.New("code wajib diisi")
	}
	if r.ClassSectionID == uuid.Nil {
		return errors.New("class_section_id wajib diisi")
	}
	return nil
}

type ClassSectionJoinResponse struct {
	UserClassSection *UserClassSectionResp `json:"user_class_section,omitempty"`
	ClassSectionID   string                `json:"class_section_id"`
}

/* =========================================================
   ==================  C S S T  (Section × Subject × Teacher)  ==================
========================================================= */

// Create
type CreateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherMasjidID       *uuid.UUID `json:"class_section_subject_teacher_masjid_id"  validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSectionID      uuid.UUID  `json:"class_section_subject_teacher_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID  `json:"class_section_subject_teacher_class_subject_id" validate:"required,uuid"`
	// pakai masjid_teachers.masjid_teacher_id
	ClassSectionSubjectTeacherTeacherID uuid.UUID `json:"class_section_subject_teacher_teacher_id" validate:"required,uuid"`

	// opsional: snapshot asisten (bukan FK kolom)
	ClassSectionSubjectTeacherAssistantTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_teacher_id" validate:"omitempty,uuid"`

	// SLUG (opsional)
	ClassSectionSubjectTeacherSlug *string `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`

	// Deskripsi (opsional)
	ClassSectionSubjectTeacherDescription *string `json:"class_section_subject_teacher_description" validate:"omitempty"`

	// Override ruangan (opsional)
	ClassSectionSubjectTeacherRoomID *uuid.UUID `json:"class_section_subject_teacher_room_id" validate:"omitempty,uuid"`

	// Link grup (opsional)
	ClassSectionSubjectTeacherGroupURL *string `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`

	// Status aktif (opsional, default: true)
	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherMasjidID       *uuid.UUID `json:"class_section_subject_teacher_masjid_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSectionID      *uuid.UUID `json:"class_section_subject_teacher_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSubjectID *uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherTeacherID      *uuid.UUID `json:"class_section_subject_teacher_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherAssistantTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `json:"class_section_subject_teacher_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`

	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

/* ----------------- RESPONSE (CSST) ----------------- */

type ClassSectionSubjectTeacherResponse struct {
	ClassSectionSubjectTeacherID             uuid.UUID  `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherMasjidID       uuid.UUID  `json:"class_section_subject_teacher_masjid_id"`
	ClassSectionSubjectTeacherSectionID      uuid.UUID  `json:"class_section_subject_teacher_section_id"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID  `json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherTeacherID      uuid.UUID  `json:"class_section_subject_teacher_teacher_id"`

	// read-only (generated by DB)
	ClassSectionSubjectTeacherTeacherNameSnap          *string `json:"class_section_subject_teacher_teacher_name_snap,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherNameSnap *string `json:"class_section_subject_teacher_assistant_teacher_name_snap,omitempty"`

	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `json:"class_section_subject_teacher_room_id,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url,omitempty"`

	ClassSectionSubjectTeacherIsActive  bool       `json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time  `json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time  `json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt *time.Time `json:"class_section_subject_teacher_deleted_at,omitempty"`
}

/* ----------------- MAPPERS (CSST) ----------------- */

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
	return m
}

func FromClassSectionSubjectTeacherModel(m csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if m.ClassSectionSubjectTeacherDeletedAt.Valid {
		t := m.ClassSectionSubjectTeacherDeletedAt.Time
		deletedAt = &t
	}
	return ClassSectionSubjectTeacherResponse{
		ClassSectionSubjectTeacherID:             m.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherMasjidID:       m.ClassSectionSubjectTeacherMasjidID,
		ClassSectionSubjectTeacherSectionID:      m.ClassSectionSubjectTeacherSectionID,
		ClassSectionSubjectTeacherClassSubjectID: m.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherTeacherID:      m.ClassSectionSubjectTeacherTeacherID,

		ClassSectionSubjectTeacherTeacherNameSnap:          m.ClassSectionSubjectTeacherTeacherNameSnap,
		ClassSectionSubjectTeacherAssistantTeacherNameSnap: m.ClassSectionSubjectTeacherAssistantTeacherNameSnap,

		ClassSectionSubjectTeacherSlug:        m.ClassSectionSubjectTeacherSlug,
		ClassSectionSubjectTeacherDescription: m.ClassSectionSubjectTeacherDescription,
		ClassSectionSubjectTeacherRoomID:      m.ClassSectionSubjectTeacherRoomID,
		ClassSectionSubjectTeacherGroupURL:    m.ClassSectionSubjectTeacherGroupURL,

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
