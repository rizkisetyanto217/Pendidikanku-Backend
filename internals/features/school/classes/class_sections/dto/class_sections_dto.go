// file: internals/features/school/classes/class_sections/dto/class_section_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	// models
	m "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	"gorm.io/datatypes"

	dbtime "madinahsalam_backend/internals/helpers/dbtime"
)

/* =========================================================
   Helpers (trim)
========================================================= */

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
   ==================  C L A S S   S E C T I O N  ==================
========================================================= */

/* ----------------- CREATE REQUEST ----------------- */

type ClassSectionCreateRequest struct {
	// Wajib
	ClassSectionSchoolID uuid.UUID `json:"class_section_school_id" form:"class_section_school_id" validate:"required"`
	ClassSectionClassID  uuid.UUID `json:"class_section_class_id"  form:"class_section_class_id"  validate:"required"`
	ClassSectionSlug     string    `json:"class_section_slug"      form:"class_section_slug"      validate:"min=1,max=160"`
	ClassSectionName     string    `json:"class_section_name"      form:"class_section_name"      validate:"required,min=1,max=100"`

	// Opsional
	ClassSectionCode     *string `json:"class_section_code"      form:"class_section_code"`
	ClassSectionSchedule *string `json:"class_section_schedule"  form:"class_section_schedule"`
	ClassSectionGroupURL *string `json:"class_section_group_url" form:"class_section_group_url"`

	// Kuota
	ClassSectionQuotaTotal *int `json:"class_section_quota_total" form:"class_section_quota_total" validate:"omitempty,min=0"`
	ClassSectionQuotaTaken *int `json:"class_section_quota_taken" form:"class_section_quota_taken" validate:"omitempty,min=0"`

	// Image (opsional)
	ClassSectionImageURL       *string `json:"class_section_image_url"        form:"class_section_image_url"`
	ClassSectionImageObjectKey *string `json:"class_section_image_object_key" form:"class_section_image_object_key"`

	// Status (enum)
	ClassSectionStatus *string `json:"class_section_status" form:"class_section_status"`

	// ====== RELASI ID (live, sesuai DDL & model) ======
	ClassSectionSchoolTeacherID          *uuid.UUID `json:"class_section_school_teacher_id"            form:"class_section_school_teacher_id"`
	ClassSectionAssistantSchoolTeacherID *uuid.UUID `json:"class_section_assistant_school_teacher_id" form:"class_section_assistant_school_teacher_id"`
	ClassSectionLeaderSchoolStudentID    *uuid.UUID `json:"class_section_leader_school_student_id"    form:"class_section_leader_school_student_id"`
	ClassSectionClassRoomID              *uuid.UUID `json:"class_section_class_room_id"               form:"class_section_class_room_id"`

	// Parent (snapshot ID)
	ClassSectionClassParentID *uuid.UUID `json:"class_section_class_parent_id" form:"class_section_class_parent_id"`

	// TERM (opsional; kolom live untuk FK)
	ClassSectionAcademicTermID *uuid.UUID `json:"class_section_academic_term_id" form:"class_section_academic_term_id"`

	// ========== Pengaturan SUBJECT-TEACHERS ==========
	// enum string: "self_select" | "assigned" | "hybrid"
	ClassSectionSubjectTeachersEnrollmentMode             *string `json:"class_section_subject_teachers_enrollment_mode" form:"class_section_subject_teachers_enrollment_mode"`
	ClassSectionSubjectTeachersSelfSelectRequiresApproval *bool   `json:"class_section_subject_teachers_self_select_requires_approval" form:"class_section_subject_teachers_self_select_requires_approval"`
	ClassSectionSubjectTeachersMaxSubjectsPerStudent      *int    `json:"class_section_subject_teachers_max_subjects_per_student" form:"class_section_subject_teachers_max_subjects_per_student"`

	// 🔥 NEW: kalau true, room dari section dipush ke semua CSST terkait
	ClassSectionPropagateRoomToCSST *bool `json:"class_section_propagate_room_to_csst" form:"class_section_propagate_room_to_csst"`
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

	if r.ClassSectionSubjectTeachersEnrollmentMode != nil {
		v := strings.ToLower(strings.TrimSpace(*r.ClassSectionSubjectTeachersEnrollmentMode))
		r.ClassSectionSubjectTeachersEnrollmentMode = &v
	}
	if r.ClassSectionStatus != nil {
		v := strings.ToLower(strings.TrimSpace(*r.ClassSectionStatus))
		r.ClassSectionStatus = &v
	}
}

func (r ClassSectionCreateRequest) ToModel() *m.ClassSectionModel {
	now := time.Now()
	cs := &m.ClassSectionModel{
		ClassSectionSchoolID: r.ClassSectionSchoolID,
		ClassSectionClassID:  &r.ClassSectionClassID,
		ClassSectionSlug:     r.ClassSectionSlug,
		ClassSectionName:     r.ClassSectionName,

		ClassSectionCode:     r.ClassSectionCode,
		ClassSectionSchedule: r.ClassSectionSchedule,
		ClassSectionGroupURL: r.ClassSectionGroupURL,

		ClassSectionImageURL:       r.ClassSectionImageURL,
		ClassSectionImageObjectKey: r.ClassSectionImageObjectKey,

		// parent snapshot id (opsional)
		ClassSectionClassParentID: r.ClassSectionClassParentID,

		// term
		ClassSectionAcademicTermID: r.ClassSectionAcademicTermID,

		ClassSectionCreatedAt: now,
		ClassSectionUpdatedAt: now,
	}

	// Kuota
	if r.ClassSectionQuotaTotal != nil {
		cs.ClassSectionQuotaTotal = r.ClassSectionQuotaTotal
	}
	if r.ClassSectionQuotaTaken != nil {
		cs.ClassSectionQuotaTaken = *r.ClassSectionQuotaTaken
	}

	// ================= Status (enum) =================
	// 1) pakai class_section_status kalau ada
	// 2) else default "active"
	status := m.ClassStatusActive
	if r.ClassSectionStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassSectionStatus)) {
		case "active":
			status = m.ClassStatusActive
		case "inactive":
			status = m.ClassStatusInactive
		case "completed":
			status = m.ClassStatusCompleted
		}
	}
	cs.ClassSectionStatus = status

	// Relasi IDs (opsional)
	cs.ClassSectionSchoolTeacherID = r.ClassSectionSchoolTeacherID
	cs.ClassSectionAssistantSchoolTeacherID = r.ClassSectionAssistantSchoolTeacherID
	cs.ClassSectionLeaderSchoolStudentID = r.ClassSectionLeaderSchoolStudentID
	cs.ClassSectionClassRoomID = r.ClassSectionClassRoomID

	// Pengaturan Subject-Teachers
	if r.ClassSectionSubjectTeachersEnrollmentMode != nil {
		switch strings.ToLower(*r.ClassSectionSubjectTeachersEnrollmentMode) {
		case "self_select":
			cs.ClassSectionSubjectTeachersEnrollmentMode = m.EnrollSelfSelect
		case "assigned":
			cs.ClassSectionSubjectTeachersEnrollmentMode = m.EnrollAssigned
		case "hybrid":
			cs.ClassSectionSubjectTeachersEnrollmentMode = m.EnrollHybrid
		}
	}
	if r.ClassSectionSubjectTeachersSelfSelectRequiresApproval != nil {
		cs.ClassSectionSubjectTeachersSelfSelectRequiresApproval = *r.ClassSectionSubjectTeachersSelfSelectRequiresApproval
	}
	if r.ClassSectionSubjectTeachersMaxSubjectsPerStudent != nil {
		cs.ClassSectionSubjectTeachersMaxSubjectsPerStudent = r.ClassSectionSubjectTeachersMaxSubjectsPerStudent
	}

	return cs
}

/* ----------------- RESPONSE ----------------- */

type ClassSectionResponse struct {
	// Identitas & relasi dasar
	ClassSectionID       uuid.UUID `json:"class_section_id"`
	ClassSectionSchoolID uuid.UUID `json:"class_section_school_id"`

	ClassSectionClassID *uuid.UUID `json:"class_section_class_id,omitempty"`

	// Properti editable
	ClassSectionSlug string  `json:"class_section_slug"`
	ClassSectionName string  `json:"class_section_name"`
	ClassSectionCode *string `json:"class_section_code"`

	ClassSectionSchedule *string `json:"class_section_schedule"`

	// Kuota (TOTAL & TAKEN)
	ClassSectionQuotaTotal *int `json:"class_section_quota_total,omitempty"`
	ClassSectionQuotaTaken int  `json:"class_section_quota_taken"`

	// STATS (ALL & ACTIVE)
	ClassSectionTotalStudentsActive       int             `json:"class_section_total_students_active"`
	ClassSectionTotalStudentsMale         int             `json:"class_section_total_students_male"`
	ClassSectionTotalStudentsFemale       int             `json:"class_section_total_students_female"`
	ClassSectionTotalStudentsMaleActive   int             `json:"class_section_total_students_male_active"`
	ClassSectionTotalStudentsFemaleActive int             `json:"class_section_total_students_female_active"`
	ClassSectionStats                     json.RawMessage `json:"class_section_stats,omitempty"`

	ClassSectionGroupURL *string `json:"class_section_group_url"`

	// Image
	ClassSectionImageURL                *string    `json:"class_section_image_url"`
	ClassSectionImageObjectKey          *string    `json:"class_section_image_object_key"`
	ClassSectionImageURLOld             *string    `json:"class_section_image_url_old"`
	ClassSectionImageObjectKeyOld       *string    `json:"class_section_image_object_key_old"`
	ClassSectionImageDeletePendingUntil *time.Time `json:"class_section_image_delete_pending_until"`

	// Status & audit
	ClassSectionStatus      string     `json:"class_section_status"`       // "active" | "inactive" | "completed"
	ClassSectionCompletedAt *time.Time `json:"class_section_completed_at"` // nullable
	ClassSectionCreatedAt   time.Time  `json:"class_section_created_at"`
	ClassSectionUpdatedAt   time.Time  `json:"class_section_updated_at"`
	ClassSectionDeletedAt   *time.Time `json:"class_section_deleted_at,omitempty"`

	// ================== CACHE & RELASI ==================
	// Class (labels)
	ClassSectionClassNameCache *string `json:"class_section_class_name_cache,omitempty"`
	ClassSectionClassSlugCache *string `json:"class_section_class_slug_cache,omitempty"`

	// Parent (id + labels)
	ClassSectionClassParentID         *uuid.UUID `json:"class_section_class_parent_id,omitempty"`
	ClassSectionClassParentNameCache  *string    `json:"class_section_class_parent_name_cache,omitempty"`
	ClassSectionClassParentSlugCache  *string    `json:"class_section_class_parent_slug_cache,omitempty"`
	ClassSectionClassParentLevelCache *int16     `json:"class_section_class_parent_level_cache,omitempty"`

	// People: ID + SLUG + RAW JSON (cache)
	ClassSectionSchoolTeacherID        *uuid.UUID      `json:"class_section_school_teacher_id,omitempty"`
	ClassSectionSchoolTeacherSlugCache *string         `json:"class_section_school_teacher_slug_cache,omitempty"`
	ClassSectionSchoolTeacherCache     json.RawMessage `json:"class_section_school_teacher_cache,omitempty"`

	ClassSectionAssistantSchoolTeacherID        *uuid.UUID      `json:"class_section_assistant_school_teacher_id,omitempty"`
	ClassSectionAssistantSchoolTeacherSlugCache *string         `json:"class_section_assistant_school_teacher_slug_cache,omitempty"`
	ClassSectionAssistantSchoolTeacherCache     json.RawMessage `json:"class_section_assistant_school_teacher_cache,omitempty"`

	ClassSectionLeaderSchoolStudentID        *uuid.UUID      `json:"class_section_leader_school_student_id,omitempty"`
	ClassSectionLeaderSchoolStudentSlugCache *string         `json:"class_section_leader_school_student_slug_cache,omitempty"`
	ClassSectionLeaderSchoolStudentCache     json.RawMessage `json:"class_section_leader_school_student_cache,omitempty"`

	// Room: ID + SLUG + NAME + LOCATION + RAW JSON
	ClassSectionClassRoomID            *uuid.UUID      `json:"class_section_class_room_id,omitempty"`
	ClassSectionClassRoomSlugCache     *string         `json:"class_section_class_room_slug_cache,omitempty"`
	ClassSectionClassRoomNameCache     *string         `json:"class_section_class_room_name_cache,omitempty"`
	ClassSectionClassRoomSlugCacheGen  *string         `json:"class_section_class_room_slug_cache_gen,omitempty"`
	ClassSectionClassRoomLocationCache *string         `json:"class_section_class_room_location_cache,omitempty"`
	ClassSectionClassRoomCache         json.RawMessage `json:"class_section_class_room_cache,omitempty"`

	// TERM (bukan JSON, sesuai SQL)
	ClassSectionAcademicTermID                *uuid.UUID `json:"class_section_academic_term_id,omitempty"`
	ClassSectionAcademicTermNameCache         *string    `json:"class_section_academic_term_name_cache,omitempty"`
	ClassSectionAcademicTermSlugCache         *string    `json:"class_section_academic_term_slug_cache,omitempty"`
	ClassSectionAcademicTermAcademicYearCache *string    `json:"class_section_academic_term_academic_year_cache,omitempty"`
	ClassSectionAcademicTermAngkatanCache     *int       `json:"class_section_academic_term_angkatan_cache,omitempty"`

	// ============== SUBJECT-TEACHERS SETTINGS ==========
	ClassSectionSubjectTeachersEnrollmentMode             string `json:"class_section_subject_teachers_enrollment_mode"`
	ClassSectionSubjectTeachersSelfSelectRequiresApproval bool   `json:"class_section_subject_teachers_self_select_requires_approval"`
	ClassSectionSubjectTeachersMaxSubjectsPerStudent      *int   `json:"class_section_subject_teachers_max_subjects_per_student,omitempty"`

	// TOTAL CSST (ALL + ACTIVE)
	ClassSectionTotalClassClassSectionSubjectTeachers       int `json:"class_section_total_class_class_section_subject_teachers"`
	ClassSectionTotalClassClassSectionSubjectTeachersActive int `json:"class_section_total_class_class_section_subject_teachers_active"`
}

// =================== TZ Helpers: FULL ===================

// Konversi semua field time ke timezone sekolah (dari token/middleware)
func (r ClassSectionResponse) WithSchoolTime(c *fiber.Ctx) ClassSectionResponse {
	out := r

	out.ClassSectionCreatedAt = dbtime.ToSchoolTime(c, r.ClassSectionCreatedAt)
	out.ClassSectionUpdatedAt = dbtime.ToSchoolTime(c, r.ClassSectionUpdatedAt)
	out.ClassSectionCompletedAt = dbtime.ToSchoolTimePtr(c, r.ClassSectionCompletedAt)
	out.ClassSectionDeletedAt = dbtime.ToSchoolTimePtr(c, r.ClassSectionDeletedAt)

	// Image soft-delete schedule ikut timezone sekolah juga
	out.ClassSectionImageDeletePendingUntil = dbtime.ToSchoolTimePtr(c, r.ClassSectionImageDeletePendingUntil)

	return out
}

func FromModelClassSection(cs *m.ClassSectionModel) ClassSectionResponse {
	var deletedAt *time.Time
	if cs.ClassSectionDeletedAt.Valid {
		t := cs.ClassSectionDeletedAt.Time
		deletedAt = &t
	}

	// datatypes.JSON -> RawMessage (langsung []byte)
	toRawJSON := func(j datatypes.JSON) json.RawMessage {
		if len(j) == 0 {
			return nil
		}
		return json.RawMessage(j)
	}

	// datatypes.JSONMap (map[string]any) -> RawMessage (marshal dulu)
	toRawJSONMap := func(m datatypes.JSONMap) json.RawMessage {
		if m == nil {
			return nil
		}
		b, err := json.Marshal(m)
		if err != nil {
			return nil
		}
		return json.RawMessage(b)
	}

	statusStr := cs.ClassSectionStatus.String()
	if statusStr == "" {
		statusStr = "active"
	}

	return ClassSectionResponse{
		// identitas
		ClassSectionID:       cs.ClassSectionID,
		ClassSectionSchoolID: cs.ClassSectionSchoolID,

		ClassSectionClassID: cs.ClassSectionClassID,

		// editable
		ClassSectionSlug: cs.ClassSectionSlug,
		ClassSectionName: cs.ClassSectionName,
		ClassSectionCode: cs.ClassSectionCode,

		ClassSectionSchedule: cs.ClassSectionSchedule,

		// Kuota
		ClassSectionQuotaTotal: cs.ClassSectionQuotaTotal,
		ClassSectionQuotaTaken: cs.ClassSectionQuotaTaken,

		// STATS
		ClassSectionTotalStudentsActive:       cs.ClassSectionTotalStudentsActive,
		ClassSectionTotalStudentsMale:         cs.ClassSectionTotalStudentsMale,
		ClassSectionTotalStudentsFemale:       cs.ClassSectionTotalStudentsFemale,
		ClassSectionTotalStudentsMaleActive:   cs.ClassSectionTotalStudentsMaleActive,
		ClassSectionTotalStudentsFemaleActive: cs.ClassSectionTotalStudentsFemaleActive,
		ClassSectionStats:                     toRawJSON(cs.ClassSectionStats),

		ClassSectionGroupURL: cs.ClassSectionGroupURL,

		ClassSectionImageURL:                cs.ClassSectionImageURL,
		ClassSectionImageObjectKey:          cs.ClassSectionImageObjectKey,
		ClassSectionImageURLOld:             cs.ClassSectionImageURLOld,
		ClassSectionImageObjectKeyOld:       cs.ClassSectionImageObjectKeyOld,
		ClassSectionImageDeletePendingUntil: cs.ClassSectionImageDeletePendingUntil,

		ClassSectionStatus:      statusStr,
		ClassSectionCompletedAt: cs.ClassSectionCompletedAt,
		ClassSectionCreatedAt:   cs.ClassSectionCreatedAt,
		ClassSectionUpdatedAt:   cs.ClassSectionUpdatedAt,
		ClassSectionDeletedAt:   deletedAt,

		// cache (read-only)
		ClassSectionClassNameCache: cs.ClassSectionClassNameCache,
		ClassSectionClassSlugCache: cs.ClassSectionClassSlugCache,

		ClassSectionClassParentID:         cs.ClassSectionClassParentID,
		ClassSectionClassParentNameCache:  cs.ClassSectionClassParentNameCache,
		ClassSectionClassParentSlugCache:  cs.ClassSectionClassParentSlugCache,
		ClassSectionClassParentLevelCache: cs.ClassSectionClassParentLevelCache,

		// People (IDs + slugs + JSON cache)
		ClassSectionSchoolTeacherID:        cs.ClassSectionSchoolTeacherID,
		ClassSectionSchoolTeacherSlugCache: cs.ClassSectionSchoolTeacherSlugCache,
		ClassSectionSchoolTeacherCache:     toRawJSON(cs.ClassSectionSchoolTeacherCache),

		ClassSectionAssistantSchoolTeacherID:        cs.ClassSectionAssistantSchoolTeacherID,
		ClassSectionAssistantSchoolTeacherSlugCache: cs.ClassSectionAssistantSchoolTeacherSlugCache,
		ClassSectionAssistantSchoolTeacherCache:     toRawJSON(cs.ClassSectionAssistantSchoolTeacherCache),

		ClassSectionLeaderSchoolStudentID:        cs.ClassSectionLeaderSchoolStudentID,
		ClassSectionLeaderSchoolStudentSlugCache: cs.ClassSectionLeaderSchoolStudentSlugCache,
		ClassSectionLeaderSchoolStudentCache:     toRawJSON(cs.ClassSectionLeaderSchoolStudentCache),

		// Room (ID + slug + name + location + JSON cache)
		ClassSectionClassRoomID:            cs.ClassSectionClassRoomID,
		ClassSectionClassRoomSlugCache:     cs.ClassSectionClassRoomSlugCache,
		ClassSectionClassRoomNameCache:     cs.ClassSectionClassRoomNameCache,
		ClassSectionClassRoomSlugCacheGen:  cs.ClassSectionClassRoomSlugCacheGen,
		ClassSectionClassRoomLocationCache: cs.ClassSectionClassRoomLocationCache,
		ClassSectionClassRoomCache:         toRawJSONMap(cs.ClassSectionClassRoomCache),

		// term
		ClassSectionAcademicTermID:                cs.ClassSectionAcademicTermID,
		ClassSectionAcademicTermNameCache:         cs.ClassSectionAcademicTermNameCache,
		ClassSectionAcademicTermSlugCache:         cs.ClassSectionAcademicTermSlugCache,
		ClassSectionAcademicTermAcademicYearCache: cs.ClassSectionAcademicTermAcademicYearCache,
		ClassSectionAcademicTermAngkatanCache:     cs.ClassSectionAcademicTermAngkatanCache,

		// Subject-Teachers settings
		ClassSectionSubjectTeachersEnrollmentMode:             cs.ClassSectionSubjectTeachersEnrollmentMode.String(),
		ClassSectionSubjectTeachersSelfSelectRequiresApproval: cs.ClassSectionSubjectTeachersSelfSelectRequiresApproval,
		ClassSectionSubjectTeachersMaxSubjectsPerStudent:      cs.ClassSectionSubjectTeachersMaxSubjectsPerStudent,

		// CSST totals
		ClassSectionTotalClassClassSectionSubjectTeachers:       cs.ClassSectionTotalClassClassSectionSubjectTeachers,
		ClassSectionTotalClassClassSectionSubjectTeachersActive: cs.ClassSectionTotalClassClassSectionSubjectTeachersActive,
	}
}

// Versi langsung TZ-aware dari single model
func FromModelClassSectionWithSchoolTime(c *fiber.Ctx, cs *m.ClassSectionModel) ClassSectionResponse {
	return FromModelClassSection(cs).WithSchoolTime(c)
}

/* ----------------- PATCH REQUEST ----------------- */

// contoh OptionalBool, dipakai untuk propagate_room_to_csst
type OptionalBool struct {
	Present bool
	Value   *bool
}

func (o *OptionalBool) UnmarshalJSON(b []byte) error {
	o.Present = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var v bool
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

type ClassSectionPatchRequest struct {
	// Relasi IDs (live)
	ClassSectionSchoolTeacherID          PatchFieldCS[uuid.UUID] `json:"class_section_school_teacher_id"`
	ClassSectionAssistantSchoolTeacherID PatchFieldCS[uuid.UUID] `json:"class_section_assistant_school_teacher_id"`
	ClassSectionLeaderSchoolStudentID    PatchFieldCS[uuid.UUID] `json:"class_section_leader_school_student_id"`
	ClassSectionClassRoomID              PatchFieldCS[uuid.UUID] `json:"class_section_class_room_id"`

	// PARENT ID
	ClassSectionClassParentID PatchFieldCS[uuid.UUID] `json:"class_section_class_parent_id"`

	// TERM (live)
	ClassSectionAcademicTermID PatchFieldCS[uuid.UUID] `json:"class_section_academic_term_id"`

	// Properti editable
	ClassSectionSlug     PatchFieldCS[string] `json:"class_section_slug"`
	ClassSectionName     PatchFieldCS[string] `json:"class_section_name"`
	ClassSectionCode     PatchFieldCS[string] `json:"class_section_code"`
	ClassSectionSchedule PatchFieldCS[string] `json:"class_section_schedule"`

	// Kuota
	ClassSectionQuotaTotal PatchFieldCS[int] `json:"class_section_quota_total"`
	ClassSectionQuotaTaken PatchFieldCS[int] `json:"class_section_quota_taken"`

	ClassSectionGroupURL PatchFieldCS[string] `json:"class_section_group_url"`

	// Image meta
	ClassSectionImageURL       PatchFieldCS[string] `json:"class_section_image_url"`
	ClassSectionImageObjectKey PatchFieldCS[string] `json:"class_section_image_object_key"`

	// Status (enum)
	ClassSectionStatus PatchFieldCS[string] `json:"class_section_status"`

	// ====== SUBJECT-TEACHERS settings ======
	ClassSectionSubjectTeachersEnrollmentMode             PatchFieldCS[string] `json:"class_section_subject_teachers_enrollment_mode"`
	ClassSectionSubjectTeachersSelfSelectRequiresApproval PatchFieldCS[bool]   `json:"class_section_subject_teachers_self_select_requires_approval"`
	ClassSectionSubjectTeachersMaxSubjectsPerStudent      PatchFieldCS[int]    `json:"class_section_subject_teachers_max_subjects_per_student"`

	// 🔥 NEW: flag propagate room → CSST (kontrol di controller)
	ClassSectionPropagateRoomToCSST OptionalBool `json:"class_section_propagate_room_to_csst" form:"class_section_propagate_room_to_csst"`
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

	// Relasi IDs
	setUUIDPtr(r.ClassSectionSchoolTeacherID, &cs.ClassSectionSchoolTeacherID)
	setUUIDPtr(r.ClassSectionAssistantSchoolTeacherID, &cs.ClassSectionAssistantSchoolTeacherID)
	setUUIDPtr(r.ClassSectionLeaderSchoolStudentID, &cs.ClassSectionLeaderSchoolStudentID)
	setUUIDPtr(r.ClassSectionClassRoomID, &cs.ClassSectionClassRoomID)

	// Parent
	setUUIDPtr(r.ClassSectionClassParentID, &cs.ClassSectionClassParentID)

	// TERM
	setUUIDPtr(r.ClassSectionAcademicTermID, &cs.ClassSectionAcademicTermID)

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

	// Kuota
	setIntPtr(r.ClassSectionQuotaTotal, &cs.ClassSectionQuotaTotal)
	if r.ClassSectionQuotaTaken.Present && r.ClassSectionQuotaTaken.Value != nil {
		cs.ClassSectionQuotaTaken = *r.ClassSectionQuotaTaken.Value
	}

	// Image meta
	setStrPtr(r.ClassSectionImageURL, &cs.ClassSectionImageURL, false)
	setStrPtr(r.ClassSectionImageObjectKey, &cs.ClassSectionImageObjectKey, false)

	// ====== Status (enum) ======
	if r.ClassSectionStatus.Present && r.ClassSectionStatus.Value != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassSectionStatus.Value)) {
		case "active":
			cs.ClassSectionStatus = m.ClassStatusActive
		case "inactive":
			cs.ClassSectionStatus = m.ClassStatusInactive
		case "completed":
			cs.ClassSectionStatus = m.ClassStatusCompleted
		}
	}

	// ====== Subject-Teachers settings ======
	if r.ClassSectionSubjectTeachersEnrollmentMode.Present {
		if r.ClassSectionSubjectTeachersEnrollmentMode.Value != nil {
			switch strings.ToLower(strings.TrimSpace(*r.ClassSectionSubjectTeachersEnrollmentMode.Value)) {
			case "self_select":
				cs.ClassSectionSubjectTeachersEnrollmentMode = m.EnrollSelfSelect
			case "assigned":
				cs.ClassSectionSubjectTeachersEnrollmentMode = m.EnrollAssigned
			case "hybrid":
				cs.ClassSectionSubjectTeachersEnrollmentMode = m.EnrollHybrid
			}
		}
	}
	if r.ClassSectionSubjectTeachersSelfSelectRequiresApproval.Present &&
		r.ClassSectionSubjectTeachersSelfSelectRequiresApproval.Value != nil {
		cs.ClassSectionSubjectTeachersSelfSelectRequiresApproval =
			*r.ClassSectionSubjectTeachersSelfSelectRequiresApproval.Value
	}
	if r.ClassSectionSubjectTeachersMaxSubjectsPerStudent.Present {
		setIntPtr(r.ClassSectionSubjectTeachersMaxSubjectsPerStudent,
			&cs.ClassSectionSubjectTeachersMaxSubjectsPerStudent)
	}

	// NOTE: flag ClassSectionPropagateRoomToCSST
	// nggak di-apply ke model; dipakai di controller sebagai kontrol behaviour.
}

/* ----------------- Decoder PATCH ----------------- */

// === Aliases yang diterima dari FE (snake_case, camelCase, short) ===
var (
	aliasTeacherID = []string{
		"class_section_school_teacher_id", "school_teacher_id", "teacher_id",
		"classSectionSchoolTeacherId", "schoolTeacherId", "teacherId",
	}
	aliasAsstTeacherID = []string{
		"class_section_assistant_school_teacher_id", "assistant_school_teacher_id", "assistant_teacher_id",
		"classSectionAssistantSchoolTeacherId", "assistantSchoolTeacherId", "assistantTeacherId",
	}
	aliasLeaderStudentID = []string{
		"class_section_leader_school_student_id", "leader_school_student_id", "leader_student_id",
		"classSectionLeaderSchoolStudentId", "leaderSchoolStudentId", "leaderStudentId",
	}
	aliasRoomID = []string{
		"class_section_class_room_id", "class_room_id", "room_id",
		"classSectionClassRoomId", "classRoomId", "roomId",
	}
	aliasTermID = []string{
		"class_section_academic_term_id", "academic_term_id", "term_id",
		"classSectionAcademicTermId", "academicTermId", "termId",
	}
	// parent id
	aliasClassParentID = []string{
		"class_section_class_parent_id", "class_parent_id", "classParentId",
	}

	aliasSlug = []string{
		"class_section_slug", "slug", "classSectionSlug",
	}
	aliasName = []string{
		"class_section_name", "name", "classSectionName",
	}
	aliasCode = []string{
		"class_section_code", "code", "classSectionCode",
	}
	aliasSchedule = []string{
		"class_section_schedule", "schedule", "classSectionSchedule",
	}
	// Kuota
	aliasQuotaTotal = []string{
		"class_section_quota_total", "quota_total", "classSectionQuotaTotal",
		"class_section_quota_total", "capacity", "classSectionCapacity",
	}
	aliasQuotaTaken = []string{
		"class_section_quota_taken", "quota_taken", "classSectionQuotaTaken",
		"class_section_total_students_active", "total_students", "classSectionTotalStudents",
	}
	aliasGroupURL = []string{
		"class_section_group_url", "group_url", "classSectionGroupUrl",
	}
	aliasImageURL = []string{
		"class_section_image_url", "image_url", "classSectionImageUrl",
	}
	aliasImageKey = []string{
		"class_section_image_object_key", "image_object_key", "classSectionImageObjectKey",
	}
	aliasStatus = []string{
		"class_section_status", "status", "classSectionStatus",
	}
	// subject-teachers
	aliasSTMode = []string{
		"class_section_subject_teachers_enrollment_mode", "subject_teachers_enrollment_mode", "classSectionSubjectTeachersEnrollmentMode",
	}
	aliasSTReqApproval = []string{
		"class_section_subject_teachers_self_select_requires_approval", "subject_teachers_self_select_requires_approval", "classSectionSubjectTeachersSelfSelectRequiresApproval",
	}
	aliasSTMaxSubj = []string{
		"class_section_subject_teachers_max_subjects_per_student", "subject_teachers_max_subjects_per_student", "classSectionSubjectTeachersMaxSubjectsPerStudent",
	}

	// 🔥 NEW: alias propagate room → CSST
	aliasPropagateRoomToCSST = []string{
		"class_section_propagate_room_to_csst", "propagate_room_to_csst", "classSectionPropagateRoomToCsst",
	}
)

// helper: jika canonical belum ada, isi dari alias pertama yang ditemukan
func setCanon(raw map[string]any, canon string, aliases []string) {
	if _, ok := raw[canon]; ok {
		return
	}
	for _, k := range aliases {
		if v, ok := raw[k]; ok {
			raw[canon] = v
			return
		}
		lk := strings.ToLower(k)
		if v, ok := raw[lk]; ok {
			raw[canon] = v
			return
		}
	}
}

func DecodePatchClassSectionFromRequest(c *fiber.Ctx, dst *ClassSectionPatchRequest) error {
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	switch {
	case strings.HasPrefix(ct, "application/json"):
		var raw map[string]any
		if err := json.Unmarshal(c.Body(), &raw); err != nil {
			return errors.New("payload JSON tidak valid")
		}
		if raw == nil {
			raw = map[string]any{}
		}

		// Relasi IDs + TERM + PARENT
		setCanon(raw, "class_section_school_teacher_id", aliasTeacherID)
		setCanon(raw, "class_section_assistant_school_teacher_id", aliasAsstTeacherID)
		setCanon(raw, "class_section_leader_school_student_id", aliasLeaderStudentID)
		setCanon(raw, "class_section_class_room_id", aliasRoomID)
		setCanon(raw, "class_section_academic_term_id", aliasTermID)
		setCanon(raw, "class_section_class_parent_id", aliasClassParentID)

		// Editable props
		setCanon(raw, "class_section_slug", aliasSlug)
		setCanon(raw, "class_section_name", aliasName)
		setCanon(raw, "class_section_code", aliasCode)
		setCanon(raw, "class_section_schedule", aliasSchedule)
		setCanon(raw, "class_section_group_url", aliasGroupURL)
		setCanon(raw, "class_section_image_url", aliasImageURL)
		setCanon(raw, "class_section_image_object_key", aliasImageKey)

		// Kuota
		setCanon(raw, "class_section_quota_total", aliasQuotaTotal)
		setCanon(raw, "class_section_quota_taken", aliasQuotaTaken)

		// Status
		setCanon(raw, "class_section_status", aliasStatus)

		// subject-teachers settings
		setCanon(raw, "class_section_subject_teachers_enrollment_mode", aliasSTMode)
		setCanon(raw, "class_section_subject_teachers_self_select_requires_approval", aliasSTReqApproval)
		setCanon(raw, "class_section_subject_teachers_max_subjects_per_student", aliasSTMaxSubj)

		// 🔥 NEW: propagate room → CSST
		setCanon(raw, "class_section_propagate_room_to_csst", aliasPropagateRoomToCSST)

		buf, _ := json.Marshal(raw)
		if err := json.Unmarshal(buf, dst); err != nil {
			return errors.New("payload JSON tidak valid (canon)")
		}
		return nil

	case strings.HasPrefix(ct, "multipart/form-data"):
		// helpers
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
		markUUIDAliases := func(keys []string, pf *PatchFieldCS[uuid.UUID]) {
			for _, key := range keys {
				if v := strings.TrimSpace(c.FormValue(key)); v != "" || formHasKey(c, key) {
					markUUID(key, pf)
					return
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
		markStrAliases := func(keys []string, pf *PatchFieldCS[string]) {
			for _, key := range keys {
				if v := c.FormValue(key); v != "" || formHasKey(c, key) {
					markStr(key, pf)
					return
				}
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
		markIntAliases := func(keys []string, pf *PatchFieldCS[int]) {
			for _, key := range keys {
				if v := c.FormValue(key); v != "" || formHasKey(c, key) {
					markInt(key, pf)
					return
				}
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
		markBoolAliases := func(keys []string, pf *PatchFieldCS[bool]) {
			for _, key := range keys {
				if v := c.FormValue(key); v != "" || formHasKey(c, key) {
					markBool(key, pf)
					return
				}
			}
		}

		// OptionalBool untuk propagate_room_to_csst
		markOptionalBoolAliases := func(keys []string, ob *OptionalBool) {
			for _, key := range keys {
				if v := c.FormValue(key); v != "" || formHasKey(c, key) {
					ob.Present = true
					if strings.EqualFold(strings.TrimSpace(v), "null") || strings.TrimSpace(v) == "" {
						ob.Value = nil
						return
					}
					lv := strings.ToLower(strings.TrimSpace(v))
					b := lv == "1" || lv == "true" || lv == "on" || lv == "yes" || lv == "y"
					ob.Value = &b
					return
				}
			}
		}

		// Map form fields (pakai alias)
		markUUIDAliases(aliasTeacherID, &dst.ClassSectionSchoolTeacherID)
		markUUIDAliases(aliasAsstTeacherID, &dst.ClassSectionAssistantSchoolTeacherID)
		markUUIDAliases(aliasLeaderStudentID, &dst.ClassSectionLeaderSchoolStudentID)
		markUUIDAliases(aliasRoomID, &dst.ClassSectionClassRoomID)
		markUUIDAliases(aliasTermID, &dst.ClassSectionAcademicTermID)
		markUUIDAliases(aliasClassParentID, &dst.ClassSectionClassParentID)

		markStrAliases(aliasSlug, &dst.ClassSectionSlug)
		markStrAliases(aliasName, &dst.ClassSectionName)
		markStrAliases(aliasCode, &dst.ClassSectionCode)
		markStrAliases(aliasSchedule, &dst.ClassSectionSchedule)
		markStrAliases(aliasGroupURL, &dst.ClassSectionGroupURL)

		markIntAliases(aliasQuotaTotal, &dst.ClassSectionQuotaTotal)
		markIntAliases(aliasQuotaTaken, &dst.ClassSectionQuotaTaken)

		markStrAliases(aliasImageURL, &dst.ClassSectionImageURL)
		markStrAliases(aliasImageKey, &dst.ClassSectionImageObjectKey)

		// Status
		markStrAliases(aliasStatus, &dst.ClassSectionStatus)

		// Subject-Teachers settings
		markStrAliases(aliasSTMode, &dst.ClassSectionSubjectTeachersEnrollmentMode)
		markBoolAliases(aliasSTReqApproval, &dst.ClassSectionSubjectTeachersSelfSelectRequiresApproval)
		markIntAliases(aliasSTMaxSubj, &dst.ClassSectionSubjectTeachersMaxSubjectsPerStudent)

		// propagate room → CSST
		markOptionalBoolAliases(aliasPropagateRoomToCSST, &dst.ClassSectionPropagateRoomToCSST)

		return nil

	default:
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
	UserClassSection *StudentClassSectionResp `json:"student_class_section,omitempty"`
	ClassSectionID   string                   `json:"class_section_id"`
}

// ===== TEACHER LITE (untuk homeroom & assistant) =====

type TeacherPersonLite struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	TitlePrefix   *string `json:"title_prefix,omitempty"`
	TitleSuffix   *string `json:"title_suffix,omitempty"`
	WhatsappURL   *string `json:"whatsapp_url,omitempty"`
	Gender        *string `json:"gender,omitempty"`
	TeacherNumber *string `json:"teacher_code,omitempty"` // nomor induk / kode guru di sekolah
}

func teacherLiteFromJSON(raw datatypes.JSON) *TeacherPersonLite {
	if len(raw) == 0 {
		return nil
	}
	var t TeacherPersonLite
	if err := json.Unmarshal(raw, &t); err != nil {
		// data lama / beda struktur → abaikan
		return nil
	}
	if strings.TrimSpace(t.ID) == "" {
		return nil
	}

	t.Name = strings.TrimSpace(t.Name)

	// Normalisasi gender & teacher_code kalau ada
	if t.TeacherNumber != nil {
		v := strings.TrimSpace(*t.TeacherNumber)
		if v == "" {
			t.TeacherNumber = nil
		} else {
			t.TeacherNumber = &v
		}
	}
	if t.Gender != nil {
		v := strings.TrimSpace(*t.Gender)
		if v == "" {
			t.Gender = nil
		} else {
			t.Gender = &v
		}
	}

	return &t
}

// ----------------- COMPACT RESPONSE (untuk list, dropdown, dsb) -----------------

type ClassSectionCompactResponse struct {
	// Identitas & relasi utama
	ClassSectionID uuid.UUID `json:"class_section_id"`

	ClassSectionAcademicTermID *uuid.UUID `json:"class_section_academic_term_id,omitempty"`

	// Basic info
	ClassSectionSlug string  `json:"class_section_slug"`
	ClassSectionName string  `json:"class_section_name"`
	ClassSectionCode *string `json:"class_section_code,omitempty"`

	ClassSectionImageURL *string `json:"class_section_image_url,omitempty"`

	// Kuota & status singkat
	ClassSectionQuotaTotal *int `json:"class_section_quota_total,omitempty"`
	ClassSectionQuotaTaken int  `json:"class_section_quota_taken"`

	ClassSectionStatus      string     `json:"class_section_status"`
	ClassSectionCompletedAt *time.Time `json:"class_section_completed_at,omitempty"`

	// Cache: term
	ClassSectionAcademicTermNameCache *string `json:"class_section_academic_term_name_cache,omitempty"`
	ClassSectionAcademicTermSlugCache *string `json:"class_section_academic_term_slug_cache,omitempty"`

	// People: ID + SLUG + RAW JSON (cache)
	ClassSectionSchoolTeacherID        *uuid.UUID      `json:"class_section_school_teacher_id,omitempty"`
	ClassSectionSchoolTeacherSlugCache *string         `json:"class_section_school_teacher_slug_cache,omitempty"`
	ClassSectionSchoolTeacherCache     json.RawMessage `json:"class_section_school_teacher_cache,omitempty"`
}

// =================== TZ Helpers: COMPACT ===================

func (r ClassSectionCompactResponse) WithSchoolTime(c *fiber.Ctx) ClassSectionCompactResponse {
	out := r
	out.ClassSectionCompletedAt = dbtime.ToSchoolTimePtr(c, r.ClassSectionCompletedAt)
	return out
}

/* ----------------- MAPPERS: FULL & COMPACT UNTUK LIST ----------------- */

// FULL: tetap seperti sebelumnya
func FromSectionModels(list []m.ClassSectionModel) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelClassSection(&list[i]))
	}
	return out
}

func FromSectionModelPtrs(list []*m.ClassSectionModel) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(list))
	for _, cs := range list {
		if cs == nil {
			continue
		}
		out = append(out, FromModelClassSection(cs))
	}
	return out
}

// FULL + TZ-aware
func FromSectionModelsWithSchoolTime(c *fiber.Ctx, list []m.ClassSectionModel) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelClassSection(&list[i]).WithSchoolTime(c))
	}
	return out
}

func FromSectionModelPtrsWithSchoolTime(c *fiber.Ctx, list []*m.ClassSectionModel) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(list))
	for _, cs := range list {
		if cs == nil {
			continue
		}
		out = append(out, FromModelClassSection(cs).WithSchoolTime(c))
	}
	return out
}

// COMPACT: single
func FromModelClassSectionToCompact(cs *m.ClassSectionModel) ClassSectionCompactResponse {

	statusStr := cs.ClassSectionStatus.String()
	if statusStr == "" {
		statusStr = "active"
	}

	return ClassSectionCompactResponse{
		ClassSectionID: cs.ClassSectionID,

		ClassSectionAcademicTermID: cs.ClassSectionAcademicTermID,

		ClassSectionSlug: cs.ClassSectionSlug,
		ClassSectionName: cs.ClassSectionName,
		ClassSectionCode: cs.ClassSectionCode,

		ClassSectionImageURL: cs.ClassSectionImageURL,

		ClassSectionQuotaTotal: cs.ClassSectionQuotaTotal,
		ClassSectionQuotaTaken: cs.ClassSectionQuotaTaken,

		ClassSectionStatus:      statusStr,
		ClassSectionCompletedAt: cs.ClassSectionCompletedAt,

		// caches: term
		ClassSectionAcademicTermNameCache: cs.ClassSectionAcademicTermNameCache,
		ClassSectionAcademicTermSlugCache: cs.ClassSectionAcademicTermSlugCache,

		// teachers (lite)
		ClassSectionSchoolTeacherID:        cs.ClassSectionSchoolTeacherID,
		ClassSectionSchoolTeacherSlugCache: cs.ClassSectionSchoolTeacherSlugCache,
		ClassSectionSchoolTeacherCache:     json.RawMessage(cs.ClassSectionSchoolTeacherCache),
	}
}

func FromModelClassSectionToCompactWithSchoolTime(c *fiber.Ctx, cs *m.ClassSectionModel) ClassSectionCompactResponse {
	return FromModelClassSectionToCompact(cs).WithSchoolTime(c)
}

// COMPACT: batch by-value
func FromSectionModelsToCompact(list []m.ClassSectionModel) []ClassSectionCompactResponse {
	out := make([]ClassSectionCompactResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelClassSectionToCompact(&list[i]))
	}
	return out
}

// COMPACT: TZ-aware batch by-value
func FromSectionModelsToCompactWithSchoolTime(c *fiber.Ctx, list []m.ClassSectionModel) []ClassSectionCompactResponse {
	out := make([]ClassSectionCompactResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelClassSectionToCompact(&list[i]).WithSchoolTime(c))
	}
	return out
}

// COMPACT: batch []*Model (kalau suatu saat dipakai)
func FromSectionModelPtrsToCompact(list []*m.ClassSectionModel) []ClassSectionCompactResponse {
	out := make([]ClassSectionCompactResponse, 0, len(list))
	for _, cs := range list {
		if cs == nil {
			continue
		}
		out = append(out, FromModelClassSectionToCompact(cs))
	}
	return out
}

// COMPACT: TZ-aware batch []*Model
func FromSectionModelPtrsToCompactWithSchoolTime(c *fiber.Ctx, list []*m.ClassSectionModel) []ClassSectionCompactResponse {
	out := make([]ClassSectionCompactResponse, 0, len(list))
	for _, cs := range list {
		if cs == nil {
			continue
		}
		out = append(out, FromModelClassSectionToCompact(cs).WithSchoolTime(c))
	}
	return out
}
