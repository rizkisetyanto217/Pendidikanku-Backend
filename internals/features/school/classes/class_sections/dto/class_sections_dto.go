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
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	m "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	"gorm.io/datatypes"
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

	// Kuota (utama, mirror ke model quota_total / quota_taken)
	ClassSectionQuotaTotal *int `json:"class_section_quota_total" form:"class_section_quota_total" validate:"omitempty,min=0"`
	ClassSectionQuotaTaken *int `json:"class_section_quota_taken" form:"class_section_quota_taken" validate:"omitempty,min=0"`

	// Image (opsional)
	ClassSectionImageURL       *string `json:"class_section_image_url"        form:"class_section_image_url"`
	ClassSectionImageObjectKey *string `json:"class_section_image_object_key" form:"class_section_image_object_key"`

	// Status
	ClassSectionIsActive *bool `json:"class_section_is_active" form:"class_section_is_active"`

	// ====== RELASI ID (live, sesuai DDL & model) ======
	ClassSectionSchoolTeacherID          *uuid.UUID `json:"class_section_school_teacher_id"            form:"class_section_school_teacher_id"`
	ClassSectionAssistantSchoolTeacherID *uuid.UUID `json:"class_section_assistant_school_teacher_id" form:"class_section_assistant_school_teacher_id"`
	ClassSectionLeaderSchoolStudentID    *uuid.UUID `json:"class_section_leader_school_student_id"    form:"class_section_leader_school_student_id"`
	ClassSectionClassRoomID              *uuid.UUID `json:"class_section_class_room_id"               form:"class_section_class_room_id"`

	// TERM (opsional; kolom live untuk FK)
	ClassSectionAcademicTermID *uuid.UUID `json:"class_section_academic_term_id" form:"class_section_academic_term_id"`

	// ========== Pengaturan SUBJECT-TEACHERS ==========
	// enum string: "self_select" | "assigned" | "hybrid"
	ClassSectionSubjectTeachersEnrollmentMode             *string `json:"class_section_subject_teachers_enrollment_mode" form:"class_section_subject_teachers_enrollment_mode"`
	ClassSectionSubjectTeachersSelfSelectRequiresApproval *bool   `json:"class_section_subject_teachers_self_select_requires_approval" form:"class_section_subject_teachers_self_select_requires_approval"`
	ClassSectionSubjectTeachersMaxSubjectsPerStudent      *int    `json:"class_section_subject_teachers_max_subjects_per_student" form:"class_section_subject_teachers_max_subjects_per_student"`
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

	// Status
	if r.ClassSectionIsActive != nil {
		cs.ClassSectionIsActive = *r.ClassSectionIsActive
	} else {
		cs.ClassSectionIsActive = true
	}

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

	// STATS (ALL & ACTIVE) - per jenis kelamin, dll.
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
	ClassSectionIsActive  bool       `json:"class_section_is_active"`
	ClassSectionCreatedAt time.Time  `json:"class_section_created_at"`
	ClassSectionUpdatedAt time.Time  `json:"class_section_updated_at"`
	ClassSectionDeletedAt *time.Time `json:"class_section_deleted_at,omitempty"`

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

func FromModelClassSection(cs *m.ClassSectionModel) ClassSectionResponse {
	var deletedAt *time.Time
	if cs.ClassSectionDeletedAt.Valid {
		t := cs.ClassSectionDeletedAt.Time
		deletedAt = &t
	}

	// helper: to raw JSON (dari datatypes.JSON)
	toRaw := func(j datatypes.JSON) json.RawMessage {
		if len(j) == 0 {
			return nil
		}
		return json.RawMessage(j)
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
		ClassSectionStats:                     toRaw(cs.ClassSectionStats),

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
		ClassSectionSchoolTeacherCache:     toRaw(cs.ClassSectionSchoolTeacherCache),

		ClassSectionAssistantSchoolTeacherID:        cs.ClassSectionAssistantSchoolTeacherID,
		ClassSectionAssistantSchoolTeacherSlugCache: cs.ClassSectionAssistantSchoolTeacherSlugCache,
		ClassSectionAssistantSchoolTeacherCache:     toRaw(cs.ClassSectionAssistantSchoolTeacherCache),

		ClassSectionLeaderSchoolStudentID:        cs.ClassSectionLeaderSchoolStudentID,
		ClassSectionLeaderSchoolStudentSlugCache: cs.ClassSectionLeaderSchoolStudentSlugCache,
		ClassSectionLeaderSchoolStudentCache:     toRaw(cs.ClassSectionLeaderSchoolStudentCache),

		// Room (ID + slug + name + location + JSON cache)
		ClassSectionClassRoomID:            cs.ClassSectionClassRoomID,
		ClassSectionClassRoomSlugCache:     cs.ClassSectionClassRoomSlugCache,
		ClassSectionClassRoomNameCache:     cs.ClassSectionClassRoomNameCache,
		ClassSectionClassRoomLocationCache: cs.ClassSectionClassRoomLocationCache,
		ClassSectionClassRoomCache:         toRaw(cs.ClassSectionClassRoomCache),

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

/* ----------------- PATCH REQUEST ----------------- */

type ClassSectionPatchRequest struct {
	// Relasi IDs (live)
	ClassSectionSchoolTeacherID          PatchFieldCS[uuid.UUID] `json:"class_section_school_teacher_id"`
	ClassSectionAssistantSchoolTeacherID PatchFieldCS[uuid.UUID] `json:"class_section_assistant_school_teacher_id"`
	ClassSectionLeaderSchoolStudentID    PatchFieldCS[uuid.UUID] `json:"class_section_leader_school_student_id"`
	ClassSectionClassRoomID              PatchFieldCS[uuid.UUID] `json:"class_section_class_room_id"`

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

	// Status
	ClassSectionIsActive PatchFieldCS[bool] `json:"class_section_is_active"`

	// ====== SUBJECT-TEACHERS settings ======
	ClassSectionSubjectTeachersEnrollmentMode             PatchFieldCS[string] `json:"class_section_subject_teachers_enrollment_mode"`               // "self_select"|"assigned"|"hybrid"
	ClassSectionSubjectTeachersSelfSelectRequiresApproval PatchFieldCS[bool]   `json:"class_section_subject_teachers_self_select_requires_approval"` // true/false
	ClassSectionSubjectTeachersMaxSubjectsPerStudent      PatchFieldCS[int]    `json:"class_section_subject_teachers_max_subjects_per_student"`
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

	// Status
	if r.ClassSectionIsActive.Present && r.ClassSectionIsActive.Value != nil {
		cs.ClassSectionIsActive = *r.ClassSectionIsActive.Value
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
	if r.ClassSectionSubjectTeachersSelfSelectRequiresApproval.Present && r.ClassSectionSubjectTeachersSelfSelectRequiresApproval.Value != nil {
		cs.ClassSectionSubjectTeachersSelfSelectRequiresApproval = *r.ClassSectionSubjectTeachersSelfSelectRequiresApproval.Value
	}
	if r.ClassSectionSubjectTeachersMaxSubjectsPerStudent.Present {
		setIntPtr(r.ClassSectionSubjectTeachersMaxSubjectsPerStudent, &cs.ClassSectionSubjectTeachersMaxSubjectsPerStudent)
	}
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
	// Kuota: alias baru + kompat nama lama (capacity / total_students)
	aliasQuotaTotal = []string{
		"class_section_quota_total", "quota_total", "classSectionQuotaTotal",
		"class_section_capacity", "capacity", "classSectionCapacity",
	}
	aliasQuotaTaken = []string{
		"class_section_quota_taken", "quota_taken", "classSectionQuotaTaken",
		"class_section_total_students", "total_students", "classSectionTotalStudents",
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
	aliasIsActive = []string{
		"class_section_is_active", "is_active", "classSectionIsActive",
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

		// Relasi IDs + TERM
		setCanon(raw, "class_section_school_teacher_id", aliasTeacherID)
		setCanon(raw, "class_section_assistant_school_teacher_id", aliasAsstTeacherID)
		setCanon(raw, "class_section_leader_school_student_id", aliasLeaderStudentID)
		setCanon(raw, "class_section_class_room_id", aliasRoomID)
		setCanon(raw, "class_section_academic_term_id", aliasTermID)

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
		setCanon(raw, "class_section_is_active", aliasIsActive)

		// subject-teachers settings
		setCanon(raw, "class_section_subject_teachers_enrollment_mode", aliasSTMode)
		setCanon(raw, "class_section_subject_teachers_self_select_requires_approval", aliasSTReqApproval)
		setCanon(raw, "class_section_subject_teachers_max_subjects_per_student", aliasSTMaxSubj)

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

		// Map form fields (pakai alias)
		markUUIDAliases(aliasTeacherID, &dst.ClassSectionSchoolTeacherID)
		markUUIDAliases(aliasAsstTeacherID, &dst.ClassSectionAssistantSchoolTeacherID)
		markUUIDAliases(aliasLeaderStudentID, &dst.ClassSectionLeaderSchoolStudentID)
		markUUIDAliases(aliasRoomID, &dst.ClassSectionClassRoomID)
		markUUIDAliases(aliasTermID, &dst.ClassSectionAcademicTermID)

		markStrAliases(aliasSlug, &dst.ClassSectionSlug)
		markStrAliases(aliasName, &dst.ClassSectionName)
		markStrAliases(aliasCode, &dst.ClassSectionCode)
		markStrAliases(aliasSchedule, &dst.ClassSectionSchedule)
		markStrAliases(aliasGroupURL, &dst.ClassSectionGroupURL)

		markIntAliases(aliasQuotaTotal, &dst.ClassSectionQuotaTotal)
		markIntAliases(aliasQuotaTaken, &dst.ClassSectionQuotaTaken)

		markStrAliases(aliasImageURL, &dst.ClassSectionImageURL)
		markStrAliases(aliasImageKey, &dst.ClassSectionImageObjectKey)

		markBoolAliases(aliasIsActive, &dst.ClassSectionIsActive)

		// Subject-Teachers settings
		markStrAliases(aliasSTMode, &dst.ClassSectionSubjectTeachersEnrollmentMode)
		markBoolAliases(aliasSTReqApproval, &dst.ClassSectionSubjectTeachersSelfSelectRequiresApproval)
		markIntAliases(aliasSTMaxSubj, &dst.ClassSectionSubjectTeachersMaxSubjectsPerStudent)

		return nil

	default:
		// fallback: coba JSON biasa
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

/* =========================================================
   ==================  C S S T  (Section × Subject × Teacher)  ==================
========================================================= */

// Create
type CreateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherSchoolID       *uuid.UUID `json:"class_section_subject_teacher_school_id"  validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSectionID uuid.UUID  `json:"class_section_subject_teacher_class_section_id" validate:"required,uuid"`
	// NEW: relasi ke CLASS_SUBJECT (bukan lagi class_subject_book)
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"required,uuid"`

	// pakai school_teachers.school_teacher_id
	ClassSectionSubjectTeacherSchoolTeacherID uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"required,uuid"`

	// opsional: asisten
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	// SLUG (opsional)
	ClassSectionSubjectTeacherSlug *string `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`

	// Deskripsi (opsional)
	ClassSectionSubjectTeacherDescription *string `json:"class_section_subject_teacher_description" validate:"omitempty"`

	// Override ruangan (opsional)
	ClassSectionSubjectTeacherClassRoomID *uuid.UUID `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`

	// Link grup (opsional)
	ClassSectionSubjectTeacherGroupURL *string `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`

	// Status aktif (opsional, default: true)
	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherSchoolID       *uuid.UUID `json:"class_section_subject_teacher_school_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSectionID *uuid.UUID `json:"class_section_subject_teacher_class_section_id" validate:"omitempty,uuid"`
	// NEW
	ClassSectionSubjectTeacherClassSubjectID  *uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherClassRoomID *uuid.UUID `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`

	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

/* ----------------- RESPONSE (CSST) ----------------- */

type ClassSectionSubjectTeacherResponse struct {
	ClassSectionSubjectTeacherID             uuid.UUID `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID       uuid.UUID `json:"class_section_subject_teacher_school_id"`
	ClassSectionSubjectTeacherClassSectionID uuid.UUID `json:"class_section_subject_teacher_class_section_id"`

	// NEW: id CLASS_SUBJECT (kolom asli)
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `json:"class_section_subject_teacher_class_subject_id"`

	// Legacy alias untuk FE lama yang masih baca *_class_subject_book_id
	// Isinya sama dengan ClassSectionSubjectTeacherClassSubjectID
	ClassSectionSubjectTeacherClassSubjectBookID uuid.UUID `json:"class_section_subject_teacher_class_subject_book_id"`

	// alias FE lama (teacher_id)
	ClassSectionSubjectTeacherTeacherID                uuid.UUID  `json:"class_section_subject_teacher_teacher_id"`
	ClassSectionSubjectTeacherSchoolTeacherID          uuid.UUID  `json:"class_section_subject_teacher_school_teacher_id"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id,omitempty"`

	// alias FE lama (room_id)
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `json:"class_section_subject_teacher_room_id,omitempty"`
	ClassSectionSubjectTeacherClassRoomID *uuid.UUID `json:"class_section_subject_teacher_class_room_id,omitempty"`

	// read-only (generated by DB)
	ClassSectionSubjectTeacherTeacherNameSnap                 *string `json:"class_section_subject_teacher_teacher_name_snap,omitempty"`           // alias FE lama
	ClassSectionSubjectTeacherAssistantTeacherNameSnap        *string `json:"class_section_subject_teacher_assistant_teacher_name_snap,omitempty"` // alias FE lama
	ClassSectionSubjectTeacherSchoolTeacherNameCache          *string `json:"class_section_subject_teacher_school_teacher_name_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache *string `json:"class_section_subject_teacher_assistant_school_teacher_name_cache,omitempty"`

	ClassSectionSubjectTeacherSlug        *string `json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string `json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string `json:"class_section_subject_teacher_group_url,omitempty"`

	ClassSectionSubjectTeacherIsActive  bool       `json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time  `json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time  `json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt *time.Time `json:"class_section_subject_teacher_deleted_at,omitempty"`
}

/* ----------------- MAPPERS (CSST) ----------------- */

func (r CreateClassSectionSubjectTeacherRequest) ToModel() csstModel.ClassSectionSubjectTeacherModel {
	row := csstModel.ClassSectionSubjectTeacherModel{
		ClassSectionSubjectTeacherClassSectionID: r.ClassSectionSubjectTeacherClassSectionID,
		// NEW: mapping ke field model terbaru
		ClassSectionSubjectTeacherClassSubjectID:  r.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherSchoolTeacherID: r.ClassSectionSubjectTeacherSchoolTeacherID,

		ClassSectionSubjectTeacherSlug:        trimLowerPtr(r.ClassSectionSubjectTeacherSlug),
		ClassSectionSubjectTeacherDescription: trimPtr(r.ClassSectionSubjectTeacherDescription),
		ClassSectionSubjectTeacherClassRoomID: r.ClassSectionSubjectTeacherClassRoomID,
		ClassSectionSubjectTeacherGroupURL:    trimPtr(r.ClassSectionSubjectTeacherGroupURL),
	}

	if r.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
		row.ClassSectionSubjectTeacherAssistantSchoolTeacherID = r.ClassSectionSubjectTeacherAssistantSchoolTeacherID
	}
	if r.ClassSectionSubjectTeacherSchoolID != nil {
		row.ClassSectionSubjectTeacherSchoolID = *r.ClassSectionSubjectTeacherSchoolID
	}
	if r.ClassSectionSubjectTeacherIsActive != nil {
		row.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	} else {
		row.ClassSectionSubjectTeacherIsActive = true
	}
	return row
}

func (r UpdateClassSectionSubjectTeacherRequest) Apply(row *csstModel.ClassSectionSubjectTeacherModel) {
	if r.ClassSectionSubjectTeacherSchoolID != nil {
		row.ClassSectionSubjectTeacherSchoolID = *r.ClassSectionSubjectTeacherSchoolID
	}
	if r.ClassSectionSubjectTeacherClassSectionID != nil {
		row.ClassSectionSubjectTeacherClassSectionID = *r.ClassSectionSubjectTeacherClassSectionID
	}
	if r.ClassSectionSubjectTeacherClassSubjectID != nil {
		row.ClassSectionSubjectTeacherClassSubjectID = *r.ClassSectionSubjectTeacherClassSubjectID
	}
	if r.ClassSectionSubjectTeacherSchoolTeacherID != nil {
		row.ClassSectionSubjectTeacherSchoolTeacherID = *r.ClassSectionSubjectTeacherSchoolTeacherID
	}
	if r.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
		row.ClassSectionSubjectTeacherAssistantSchoolTeacherID = r.ClassSectionSubjectTeacherAssistantSchoolTeacherID
	}
	if r.ClassSectionSubjectTeacherSlug != nil {
		row.ClassSectionSubjectTeacherSlug = trimLowerPtr(r.ClassSectionSubjectTeacherSlug)
	}
	if r.ClassSectionSubjectTeacherDescription != nil {
		row.ClassSectionSubjectTeacherDescription = trimPtr(r.ClassSectionSubjectTeacherDescription)
	}
	if r.ClassSectionSubjectTeacherClassRoomID != nil {
		row.ClassSectionSubjectTeacherClassRoomID = r.ClassSectionSubjectTeacherClassRoomID
	}
	if r.ClassSectionSubjectTeacherGroupURL != nil {
		row.ClassSectionSubjectTeacherGroupURL = trimPtr(r.ClassSectionSubjectTeacherGroupURL)
	}
	if r.ClassSectionSubjectTeacherIsActive != nil {
		row.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	}
}

func FromClassSectionSubjectTeacherModel(row csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if row.ClassSectionSubjectTeacherDeletedAt.Valid {
		t := row.ClassSectionSubjectTeacherDeletedAt.Time
		deletedAt = &t
	}
	resp := ClassSectionSubjectTeacherResponse{
		ClassSectionSubjectTeacherID:             row.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherSchoolID:       row.ClassSectionSubjectTeacherSchoolID,
		ClassSectionSubjectTeacherClassSectionID: row.ClassSectionSubjectTeacherClassSectionID,

		// id CLASS_SUBJECT (baru)
		ClassSectionSubjectTeacherClassSubjectID: row.ClassSectionSubjectTeacherClassSubjectID,
		// alias lama *_class_subject_book_id → isi sama
		ClassSectionSubjectTeacherClassSubjectBookID: row.ClassSectionSubjectTeacherClassSubjectID,

		ClassSectionSubjectTeacherSchoolTeacherID:          row.ClassSectionSubjectTeacherSchoolTeacherID,
		ClassSectionSubjectTeacherAssistantSchoolTeacherID: row.ClassSectionSubjectTeacherAssistantSchoolTeacherID,

		ClassSectionSubjectTeacherClassRoomID: row.ClassSectionSubjectTeacherClassRoomID,

		ClassSectionSubjectTeacherSlug:        row.ClassSectionSubjectTeacherSlug,
		ClassSectionSubjectTeacherDescription: row.ClassSectionSubjectTeacherDescription,
		ClassSectionSubjectTeacherGroupURL:    row.ClassSectionSubjectTeacherGroupURL,

		ClassSectionSubjectTeacherIsActive:  row.ClassSectionSubjectTeacherIsActive,
		ClassSectionSubjectTeacherCreatedAt: row.ClassSectionSubjectTeacherCreatedAt,
		ClassSectionSubjectTeacherUpdatedAt: row.ClassSectionSubjectTeacherUpdatedAt,
		ClassSectionSubjectTeacherDeletedAt: deletedAt,
	}

	// Aliases untuk kompat FE lama
	resp.ClassSectionSubjectTeacherTeacherID = row.ClassSectionSubjectTeacherSchoolTeacherID
	resp.ClassSectionSubjectTeacherRoomID = row.ClassSectionSubjectTeacherClassRoomID

	// Nama cache (lama & baru) — ini masih ikut model CSST yang lama
	resp.ClassSectionSubjectTeacherTeacherNameSnap = row.ClassSectionSubjectTeacherSchoolTeacherNameCache
	resp.ClassSectionSubjectTeacherAssistantTeacherNameSnap = row.ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache
	resp.ClassSectionSubjectTeacherSchoolTeacherNameCache = row.ClassSectionSubjectTeacherSchoolTeacherNameCache
	resp.ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache = row.ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache

	return resp
}

func FromClassSectionSubjectTeacherModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromClassSectionSubjectTeacherModel(r))
	}
	return out
}

// ===== (Opsional) CSST Lite map =====

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

func CSSTLiteFromModel(row csstModel.ClassSectionSubjectTeacherModel) CSSTItemLite {
	out := CSSTItemLite{
		ID:       row.ClassSectionSubjectTeacherID.String(),
		IsActive: row.ClassSectionSubjectTeacherIsActive,
		Teacher: struct {
			ID string `json:"id"`
		}{
			ID: row.ClassSectionSubjectTeacherSchoolTeacherID.String(),
		},
		ClassSubject: struct {
			ID      string `json:"id"`
			Subject struct {
				ID   string  `json:"id"`
				Name *string `json:"name,omitempty"`
			} `json:"subject"`
		}{
			// pakai CLASS_SUBJECT ID (baru)
			ID: row.ClassSectionSubjectTeacherClassSubjectID.String(),
		},
		GroupURL: nil,
		Stats: &struct {
			TotalAttendance *int32 `json:"total_attendance,omitempty"`
		}{
			TotalAttendance: func(v int) *int32 {
				iv := int32(v)
				return &iv
			}(row.ClassSectionSubjectTeacherTotalAttendance),
		},
	}

	if row.ClassSectionSubjectTeacherClassRoomID != nil {
		out.Room = &struct {
			ID string `json:"id"`
		}{
			ID: row.ClassSectionSubjectTeacherClassRoomID.String(),
		}
	}

	if row.ClassSectionSubjectTeacherGroupURL != nil && strings.TrimSpace(*row.ClassSectionSubjectTeacherGroupURL) != "" {
		g := strings.TrimSpace(*row.ClassSectionSubjectTeacherGroupURL)
		out.GroupURL = &g
	}

	out.ClassSubject.Subject.Name = row.ClassSectionSubjectTeacherSubjectNameCache

	return out
}

func CSSTLiteSliceFromModels(rows []csstModel.ClassSectionSubjectTeacherModel) []CSSTItemLite {
	out := make([]CSSTItemLite, 0, len(rows))
	for _, r := range rows {
		out = append(out, CSSTLiteFromModel(r))
	}
	return out
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
	TeacherNumber *string `json:"teacher_number,omitempty"` // nomor induk / kode guru di sekolah
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

	// Normalisasi gender & teacher_number kalau ada
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
