package dto

import (
	"encoding/json"
	"strings"
	"time"

	teacherSnap "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/service"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =========================================================
   OPTIONS & NESTED TYPES
========================================================= */

// Options untuk mapping CSST → response
type FromCSSTOptions struct {
	// kalau true, isi field nested AcademicTerm di response
	IncludeAcademicTerm bool
}

// Nested academic term (dipakai kalau include=academic_term)
type AcademicTermLite struct {
	ID       *uuid.UUID `json:"id,omitempty"`
	Name     *string    `json:"name,omitempty"`
	Slug     *string    `json:"slug,omitempty"`
	Year     *string    `json:"year,omitempty"`
	Angkatan *int       `json:"angkatan,omitempty"`
}

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

// decode JSONB → *TeacherCache (dipakai di response)
func TeacherCacheFromJSON(j *datatypes.JSON) *teacherSnap.TeacherCache {
	if j == nil {
		return nil
	}
	raw := []byte(*j)
	if len(raw) == 0 {
		return nil
	}
	// handle literal "null"
	if strings.TrimSpace(string(raw)) == "null" {
		return nil
	}

	var ts teacherSnap.TeacherCache
	if err := json.Unmarshal(raw, &ts); err != nil {
		// kalau gagal parse, jangan panik – cukup kembalikan nil
		return nil
	}
	// opsional: trim string di dalam cache supaya bersih
	trim := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	ts.Name = trim(ts.Name)
	ts.AvatarURL = trim(ts.AvatarURL)
	ts.WhatsappURL = trim(ts.WhatsappURL)
	ts.TitlePrefix = trim(ts.TitlePrefix)
	ts.TitleSuffix = trim(ts.TitleSuffix)
	ts.Gender = trim(ts.Gender)
	ts.TeacherCode = trim(ts.TeacherCode)

	ts.ID = strings.TrimSpace(ts.ID)

	if ts.ID == "" &&
		ts.Name == nil &&
		ts.AvatarURL == nil &&
		ts.WhatsappURL == nil &&
		ts.TitlePrefix == nil &&
		ts.TitleSuffix == nil &&
		ts.Gender == nil &&
		ts.TeacherCode == nil {
		return nil
	}

	return &ts
}

/* =========================================================
   1) REQUEST DTO (FOLLOW SQL/MODEL TERBARU)
========================================================= */

// Create
type CreateClassSectionSubjectTeacherRequest struct {
	// Biasanya diisi dari context auth pada controller
	ClassSectionSubjectTeacherSchoolID *uuid.UUID `json:"class_section_subject_teacher_school_id"  validate:"omitempty,uuid"`

	// Relasi utama
	ClassSectionSubjectTeacherClassSectionID uuid.UUID `json:"class_section_subject_teacher_class_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"required,uuid"`

	// pakai school_teachers.school_teacher_id (wajib untuk manual create)
	ClassSectionSubjectTeacherSchoolTeacherID uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"required,uuid"`

	// ➕ Asisten (opsional)
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	// Opsional
	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherClassRoomID *uuid.UUID `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`

	// quota_total (capacity baru) — >=0 divalidasi di DB (CHECK)
	ClassSectionSubjectTeacherQuotaTotal *int `json:"class_section_subject_teacher_quota_total" validate:"omitempty"`

	// enum: offline|online|hybrid
	ClassSectionSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	// Attendance entry mode per CSST (boleh kosong, nanti fallback ke default school di service)
	// enum: teacher_only | student_only | both
	ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache *csstModel.AttendanceEntryMode `json:"class_section_subject_teacher_school_attendance_entry_mode_cache" validate:"omitempty,oneof=teacher_only student_only both"`

	// Target pertemuan & KKM spesifik CSST (opsional)
	ClassSectionSubjectTeacherTotalMeetingsTarget *int `json:"class_section_subject_teacher_total_meetings_target" validate:"omitempty"`
	ClassSectionSubjectTeacherMinPassingScore     *int `json:"class_section_subject_teacher_min_passing_score" validate:"omitempty,gte=0"`

	// Status (enum baru) + kompat lama (bool)
	// enum: active | inactive | completed
	ClassSectionSubjectTeacherStatus *string `json:"class_section_subject_teacher_status" validate:"omitempty,oneof=active inactive completed"`
	// kompat: FE lama masih kirim is_active → kita map ke status
	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherSchoolID        *uuid.UUID `json:"class_section_subject_teacher_school_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSectionID  *uuid.UUID `json:"class_section_subject_teacher_class_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSubjectID  *uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"omitempty,uuid"`

	// ➕ Asisten (opsional)
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSlug         *string                      `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription  *string                      `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherClassRoomID  *uuid.UUID                   `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL     *string                      `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherQuotaTotal   *int                         `json:"class_section_subject_teacher_quota_total" validate:"omitempty"`
	ClassSectionSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	// Bisa update custom attendance mode juga
	ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache *csstModel.AttendanceEntryMode `json:"class_section_subject_teacher_school_attendance_entry_mode_cache" validate:"omitempty,oneof=teacher_only student_only both"`

	ClassSectionSubjectTeacherTotalMeetingsTarget *int `json:"class_section_subject_teacher_total_meetings_target" validate:"omitempty"`
	ClassSectionSubjectTeacherMinPassingScore     *int `json:"class_section_subject_teacher_min_passing_score" validate:"omitempty,gte=0"`

	// Status (enum) + kompat lama (bool)
	ClassSectionSubjectTeacherStatus   *string `json:"class_section_subject_teacher_status" validate:"omitempty,oneof=active inactive completed"`
	ClassSectionSubjectTeacherIsActive *bool   `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

/*
=========================================================
 2. RESPONSE DTO — sinkron SQL/model terbaru
=========================================================
*/

type ClassSectionSubjectTeacherResponse struct {
	/* ===== IDs & Relations ===== */
	ClassSectionSubjectTeacherID                       uuid.UUID  `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID                 uuid.UUID  `json:"class_section_subject_teacher_school_id"`
	ClassSectionSubjectTeacherClassSectionID           uuid.UUID  `json:"class_section_subject_teacher_class_section_id"`
	ClassSectionSubjectTeacherClassSubjectID           uuid.UUID  `json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherSchoolTeacherID          *uuid.UUID `json:"class_section_subject_teacher_school_teacher_id,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id,omitempty"`
	ClassSectionSubjectTeacherClassRoomID              *uuid.UUID `json:"class_section_subject_teacher_class_room_id,omitempty"`

	/* ===== Identitas & Fasilitas ===== */
	ClassSectionSubjectTeacherSlug        *string `json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string `json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string `json:"class_section_subject_teacher_group_url,omitempty"`

	/* ===== Agregat & kapasitas (quota_total / quota_taken) ===== */
	ClassSectionSubjectTeacherTotalAttendance          int    `json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherTotalMeetingsTarget      *int   `json:"class_section_subject_teacher_total_meetings_target,omitempty"`
	ClassSectionSubjectTeacherQuotaTotal               *int   `json:"class_section_subject_teacher_quota_total,omitempty"`
	ClassSectionSubjectTeacherQuotaTaken               int    `json:"class_section_subject_teacher_quota_taken"`
	ClassSectionSubjectTeacherTotalAssessments         int    `json:"class_section_subject_teacher_total_assessments"`
	ClassSectionSubjectTeacherTotalAssessmentsGraded   int    `json:"class_section_subject_teacher_total_assessments_graded"`
	ClassSectionSubjectTeacherTotalAssessmentsUngraded int    `json:"class_section_subject_teacher_total_assessments_ungraded"`
	ClassSectionSubjectTeacherTotalStudentsPassed      int    `json:"class_section_subject_teacher_total_students_passed"`
	ClassSectionSubjectTeacherDeliveryMode             string `json:"class_section_subject_teacher_delivery_mode"`

	// total buku terkait CSST (cache)
	ClassSectionSubjectTeacherTotalBooks int `json:"class_section_subject_teacher_total_books"`

	// Attendance mode efektif yang dipakai di CSST (hasil cache)
	ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache *string `json:"class_section_subject_teacher_school_attendance_entry_mode_cache,omitempty"`

	/* ===== SECTION caches (varchar/text) ===== */
	ClassSectionSubjectTeacherClassSectionSlugCache *string `json:"class_section_subject_teacher_class_section_slug_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionNameCache *string `json:"class_section_subject_teacher_class_section_name_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionCodeCache *string `json:"class_section_subject_teacher_class_section_code_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionURLCache  *string `json:"class_section_subject_teacher_class_section_url_cache,omitempty"`

	/* ===== ROOM cache ===== */
	ClassSectionSubjectTeacherClassRoomSlugCache *string         `json:"class_section_subject_teacher_class_room_slug_cache,omitempty"`
	ClassSectionSubjectTeacherClassRoomCache     *datatypes.JSON `json:"class_section_subject_teacher_class_room_cache,omitempty"`
	// generated
	ClassSectionSubjectTeacherClassRoomNameCache     *string `json:"class_section_subject_teacher_class_room_name_cache,omitempty"`
	ClassSectionSubjectTeacherClassRoomSlugCacheGen  *string `json:"class_section_subject_teacher_class_room_slug_cache_gen,omitempty"`
	ClassSectionSubjectTeacherClassRoomLocationCache *string `json:"class_section_subject_teacher_class_room_location_cache,omitempty"`

	/* ===== PEOPLE caches ===== */
	ClassSectionSubjectTeacherSchoolTeacherSlugCache          *string                   `json:"class_section_subject_teacher_school_teacher_slug_cache,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherCache              *teacherSnap.TeacherCache `json:"class_section_subject_teacher_school_teacher_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSlugCache *string                   `json:"class_section_subject_teacher_assistant_school_teacher_slug_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherCache     *teacherSnap.TeacherCache `json:"class_section_subject_teacher_assistant_school_teacher_cache,omitempty"`
	// generated names
	ClassSectionSubjectTeacherSchoolTeacherNameCache          *string `json:"class_section_subject_teacher_school_teacher_name_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache *string `json:"class_section_subject_teacher_assistant_school_teacher_name_cache,omitempty"`

	/* ===== SUBJECT (via CLASS_SUBJECT) cache ===== */
	ClassSectionSubjectTeacherSubjectID        *uuid.UUID `json:"class_section_subject_teacher_subject_id,omitempty"`
	ClassSectionSubjectTeacherSubjectNameCache *string    `json:"class_section_subject_teacher_subject_name_cache,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeCache *string    `json:"class_section_subject_teacher_subject_code_cache,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugCache *string    `json:"class_section_subject_teacher_subject_slug_cache,omitempty"`

	/* ===== ACADEMIC_TERM cache ===== */
	ClassSectionSubjectTeacherAcademicTermID            *uuid.UUID `json:"class_section_subject_teacher_academic_term_id,omitempty"`
	ClassSectionSubjectTeacherAcademicTermNameCache     *string    `json:"class_section_subject_teacher_academic_term_name_cache,omitempty"`
	ClassSectionSubjectTeacherAcademicTermSlugCache     *string    `json:"class_section_subject_teacher_academic_term_slug_cache,omitempty"`
	ClassSectionSubjectTeacherAcademicYearCache         *string    `json:"class_section_subject_teacher_academic_year_cache,omitempty"`
	ClassSectionSubjectTeacherAcademicTermAngkatanCache *int       `json:"class_section_subject_teacher_academic_term_angkatan_cache,omitempty"`

	/* ===== KKM SNAPSHOT (cache + override per CSST) ===== */
	ClassSectionSubjectTeacherMinPassingScoreClassSubjectCache *int `json:"class_section_subject_teacher_min_passing_score_class_subject_cache,omitempty"`
	ClassSectionSubjectTeacherMinPassingScore                  *int `json:"class_section_subject_teacher_min_passing_score,omitempty"`

	/* ===== Status & audit ===== */
	ClassSectionSubjectTeacherStatus      string     `json:"class_section_subject_teacher_status"`
	ClassSectionSubjectTeacherIsActive    bool       `json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCompletedAt *time.Time `json:"class_section_subject_teacher_completed_at,omitempty"`
	ClassSectionSubjectTeacherCreatedAt   time.Time  `json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt   time.Time  `json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt   *time.Time `json:"class_section_subject_teacher_deleted_at,omitempty"`

	// nested academic_term (optional, pakai include)
	AcademicTerm *AcademicTermLite `json:"academic_term,omitempty"`
}

/* =========================================================
   3) MAPPERS
========================================================= */

func (r CreateClassSectionSubjectTeacherRequest) ToModel() csstModel.ClassSectionSubjectTeacherModel {
	// bungkus teacherID (required) ke pointer supaya cocok dengan model
	teacherID := r.ClassSectionSubjectTeacherSchoolTeacherID

	m := csstModel.ClassSectionSubjectTeacherModel{
		// Wajib
		ClassSectionSubjectTeacherClassSectionID: r.ClassSectionSubjectTeacherClassSectionID,
		ClassSectionSubjectTeacherClassSubjectID: r.ClassSectionSubjectTeacherClassSubjectID,

		// teacher wajib saat manual create
		ClassSectionSubjectTeacherSchoolTeacherID: &teacherID,

		// Opsional basic
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
	if r.ClassSectionSubjectTeacherQuotaTotal != nil {
		m.ClassSectionSubjectTeacherQuotaTotal = r.ClassSectionSubjectTeacherQuotaTotal
	}
	if r.ClassSectionSubjectTeacherDeliveryMode != nil {
		m.ClassSectionSubjectTeacherDeliveryMode = *r.ClassSectionSubjectTeacherDeliveryMode
	}
	if r.ClassSectionSubjectTeacherTotalMeetingsTarget != nil {
		m.ClassSectionSubjectTeacherTotalMeetingsTarget = r.ClassSectionSubjectTeacherTotalMeetingsTarget
	}
	if r.ClassSectionSubjectTeacherMinPassingScore != nil {
		m.ClassSectionSubjectTeacherMinPassingScore = r.ClassSectionSubjectTeacherMinPassingScore
	}

	// kalau create langsung mau set custom attendance, boleh diisi di sini
	if r.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache != nil {
		m.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache =
			r.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache
	}

	// ====== Status (enum) + fallback is_active ======
	status := csstModel.ClassStatusActive

	if r.ClassSectionSubjectTeacherStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassSectionSubjectTeacherStatus)) {
		case "active":
			status = csstModel.ClassStatusActive
		case "inactive":
			status = csstModel.ClassStatusInactive
		case "completed":
			status = csstModel.ClassStatusCompleted
		}
	} else if r.ClassSectionSubjectTeacherIsActive != nil {
		if *r.ClassSectionSubjectTeacherIsActive {
			status = csstModel.ClassStatusActive
		} else {
			status = csstModel.ClassStatusInactive
		}
	}
	m.ClassSectionSubjectTeacherStatus = status

	return m
}

func (r UpdateClassSectionSubjectTeacherRequest) Apply(m *csstModel.ClassSectionSubjectTeacherModel) {
	if r.ClassSectionSubjectTeacherSchoolID != nil {
		m.ClassSectionSubjectTeacherSchoolID = *r.ClassSectionSubjectTeacherSchoolID
	}
	if r.ClassSectionSubjectTeacherClassSectionID != nil {
		m.ClassSectionSubjectTeacherClassSectionID = *r.ClassSectionSubjectTeacherClassSectionID
	}
	if r.ClassSectionSubjectTeacherClassSubjectID != nil {
		m.ClassSectionSubjectTeacherClassSubjectID = *r.ClassSectionSubjectTeacherClassSubjectID
	}
	if r.ClassSectionSubjectTeacherSchoolTeacherID != nil {
		// sekarang model pakai pointer
		m.ClassSectionSubjectTeacherSchoolTeacherID = r.ClassSectionSubjectTeacherSchoolTeacherID
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
	if r.ClassSectionSubjectTeacherQuotaTotal != nil {
		m.ClassSectionSubjectTeacherQuotaTotal = r.ClassSectionSubjectTeacherQuotaTotal
	}
	if r.ClassSectionSubjectTeacherDeliveryMode != nil {
		m.ClassSectionSubjectTeacherDeliveryMode = *r.ClassSectionSubjectTeacherDeliveryMode
	}
	if r.ClassSectionSubjectTeacherTotalMeetingsTarget != nil {
		m.ClassSectionSubjectTeacherTotalMeetingsTarget = r.ClassSectionSubjectTeacherTotalMeetingsTarget
	}
	if r.ClassSectionSubjectTeacherMinPassingScore != nil {
		m.ClassSectionSubjectTeacherMinPassingScore = r.ClassSectionSubjectTeacherMinPassingScore
	}

	// update custom attendance mode
	if r.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache != nil {
		m.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache =
			r.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache
	}

	// ====== Status (enum) + fallback is_active ======
	if r.ClassSectionSubjectTeacherStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassSectionSubjectTeacherStatus)) {
		case "active":
			m.ClassSectionSubjectTeacherStatus = csstModel.ClassStatusActive
		case "inactive":
			m.ClassSectionSubjectTeacherStatus = csstModel.ClassStatusInactive
		case "completed":
			m.ClassSectionSubjectTeacherStatus = csstModel.ClassStatusCompleted
		}
	} else if r.ClassSectionSubjectTeacherIsActive != nil {
		if *r.ClassSectionSubjectTeacherIsActive {
			m.ClassSectionSubjectTeacherStatus = csstModel.ClassStatusActive
		} else {
			m.ClassSectionSubjectTeacherStatus = csstModel.ClassStatusInactive
		}
	}
}

// internal: mapper dengan options (dipakai semua)
func fromClassSectionSubjectTeacherModelWithOptions(
	m csstModel.ClassSectionSubjectTeacherModel,
	opt FromCSSTOptions,
) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if m.ClassSectionSubjectTeacherDeletedAt.Valid {
		t := m.ClassSectionSubjectTeacherDeletedAt.Time
		deletedAt = &t
	}

	// convert enum pointer → *string untuk JSON
	var attendanceCache *string
	if m.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache != nil {
		v := string(*m.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache)
		attendanceCache = &v
	}

	// status & is_active (derived)
	statusStr := string(m.ClassSectionSubjectTeacherStatus)
	if statusStr == "" {
		statusStr = "active"
	}
	isActive := m.ClassSectionSubjectTeacherStatus == csstModel.ClassStatusActive

	resp := ClassSectionSubjectTeacherResponse{
		// IDs & Relations
		ClassSectionSubjectTeacherID:                       m.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherSchoolID:                 m.ClassSectionSubjectTeacherSchoolID,
		ClassSectionSubjectTeacherClassSectionID:           m.ClassSectionSubjectTeacherClassSectionID,
		ClassSectionSubjectTeacherClassSubjectID:           m.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherSchoolTeacherID:          m.ClassSectionSubjectTeacherSchoolTeacherID,
		ClassSectionSubjectTeacherAssistantSchoolTeacherID: m.ClassSectionSubjectTeacherAssistantSchoolTeacherID,
		ClassSectionSubjectTeacherClassRoomID:              m.ClassSectionSubjectTeacherClassRoomID,

		// Identitas / fasilitas
		ClassSectionSubjectTeacherSlug:        m.ClassSectionSubjectTeacherSlug,
		ClassSectionSubjectTeacherDescription: m.ClassSectionSubjectTeacherDescription,
		ClassSectionSubjectTeacherGroupURL:    m.ClassSectionSubjectTeacherGroupURL,

		// Agregat & kapasitas
		ClassSectionSubjectTeacherTotalAttendance:          m.ClassSectionSubjectTeacherTotalAttendance,
		ClassSectionSubjectTeacherTotalMeetingsTarget:      m.ClassSectionSubjectTeacherTotalMeetingsTarget,
		ClassSectionSubjectTeacherQuotaTotal:               m.ClassSectionSubjectTeacherQuotaTotal,
		ClassSectionSubjectTeacherQuotaTaken:               m.ClassSectionSubjectTeacherQuotaTaken,
		ClassSectionSubjectTeacherTotalAssessments:         m.ClassSectionSubjectTeacherTotalAssessments,
		ClassSectionSubjectTeacherTotalAssessmentsGraded:   m.ClassSectionSubjectTeacherTotalAssessmentsGraded,
		ClassSectionSubjectTeacherTotalAssessmentsUngraded: m.ClassSectionSubjectTeacherTotalAssessmentsUngraded,
		ClassSectionSubjectTeacherTotalStudentsPassed:      m.ClassSectionSubjectTeacherTotalStudentsPassed,
		ClassSectionSubjectTeacherDeliveryMode:             string(m.ClassSectionSubjectTeacherDeliveryMode),
		ClassSectionSubjectTeacherTotalBooks:               m.ClassSectionSubjectTeacherTotalBooks,

		// attendance cache
		ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache: attendanceCache,

		// SECTION caches
		ClassSectionSubjectTeacherClassSectionSlugCache: m.ClassSectionSubjectTeacherClassSectionSlugCache,
		ClassSectionSubjectTeacherClassSectionNameCache: m.ClassSectionSubjectTeacherClassSectionNameCache,
		ClassSectionSubjectTeacherClassSectionCodeCache: m.ClassSectionSubjectTeacherClassSectionCodeCache,
		ClassSectionSubjectTeacherClassSectionURLCache:  m.ClassSectionSubjectTeacherClassSectionURLCache,

		// ROOM cache + generated
		ClassSectionSubjectTeacherClassRoomSlugCache:     m.ClassSectionSubjectTeacherClassRoomSlugCache,
		ClassSectionSubjectTeacherClassRoomCache:         m.ClassSectionSubjectTeacherClassRoomCache,
		ClassSectionSubjectTeacherClassRoomNameCache:     m.ClassSectionSubjectTeacherClassRoomNameCache,
		ClassSectionSubjectTeacherClassRoomSlugCacheGen:  m.ClassSectionSubjectTeacherClassRoomSlugCacheGen,
		ClassSectionSubjectTeacherClassRoomLocationCache: m.ClassSectionSubjectTeacherClassRoomLocationCache,

		// PEOPLE caches + generated names
		ClassSectionSubjectTeacherSchoolTeacherSlugCache:          m.ClassSectionSubjectTeacherSchoolTeacherSlugCache,
		ClassSectionSubjectTeacherSchoolTeacherCache:              TeacherCacheFromJSON(m.ClassSectionSubjectTeacherSchoolTeacherCache),
		ClassSectionSubjectTeacherAssistantSchoolTeacherSlugCache: m.ClassSectionSubjectTeacherAssistantSchoolTeacherSlugCache,
		ClassSectionSubjectTeacherAssistantSchoolTeacherCache:     TeacherCacheFromJSON(m.ClassSectionSubjectTeacherAssistantSchoolTeacherCache),
		ClassSectionSubjectTeacherSchoolTeacherNameCache:          m.ClassSectionSubjectTeacherSchoolTeacherNameCache,
		ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache: m.ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache,

		// SUBJECT cache
		ClassSectionSubjectTeacherSubjectID:        m.ClassSectionSubjectTeacherSubjectID,
		ClassSectionSubjectTeacherSubjectNameCache: m.ClassSectionSubjectTeacherSubjectNameCache,
		ClassSectionSubjectTeacherSubjectCodeCache: m.ClassSectionSubjectTeacherSubjectCodeCache,
		ClassSectionSubjectTeacherSubjectSlugCache: m.ClassSectionSubjectTeacherSubjectSlugCache,

		// ACADEMIC_TERM cache
		ClassSectionSubjectTeacherAcademicTermID:            m.ClassSectionSubjectTeacherAcademicTermID,
		ClassSectionSubjectTeacherAcademicTermNameCache:     m.ClassSectionSubjectTeacherAcademicTermNameCache,
		ClassSectionSubjectTeacherAcademicTermSlugCache:     m.ClassSectionSubjectTeacherAcademicTermSlugCache,
		ClassSectionSubjectTeacherAcademicYearCache:         m.ClassSectionSubjectTeacherAcademicYearCache,
		ClassSectionSubjectTeacherAcademicTermAngkatanCache: m.ClassSectionSubjectTeacherAcademicTermAngkatanCache,

		// KKM
		ClassSectionSubjectTeacherMinPassingScoreClassSubjectCache: m.ClassSectionSubjectTeacherMinPassingScoreClassSubjectCache,
		ClassSectionSubjectTeacherMinPassingScore:                  m.ClassSectionSubjectTeacherMinPassingScore,

		// Status & audit
		ClassSectionSubjectTeacherStatus:      statusStr,
		ClassSectionSubjectTeacherIsActive:    isActive,
		ClassSectionSubjectTeacherCompletedAt: m.ClassSectionSubjectTeacherCompletedAt,
		ClassSectionSubjectTeacherCreatedAt:   m.ClassSectionSubjectTeacherCreatedAt,
		ClassSectionSubjectTeacherUpdatedAt:   m.ClassSectionSubjectTeacherUpdatedAt,
		ClassSectionSubjectTeacherDeletedAt:   deletedAt,
	}

	// isi nested AcademicTerm kalau diminta
	if opt.IncludeAcademicTerm && m.ClassSectionSubjectTeacherAcademicTermID != nil {
		resp.AcademicTerm = &AcademicTermLite{
			ID:       m.ClassSectionSubjectTeacherAcademicTermID,
			Name:     m.ClassSectionSubjectTeacherAcademicTermNameCache,
			Slug:     m.ClassSectionSubjectTeacherAcademicTermSlugCache,
			Year:     m.ClassSectionSubjectTeacherAcademicYearCache,
			Angkatan: m.ClassSectionSubjectTeacherAcademicTermAngkatanCache,
		}
	}

	return resp
}

// API lama: single model → response (tanpa include extra)
func FromClassSectionSubjectTeacherModel(m csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	return fromClassSectionSubjectTeacherModelWithOptions(m, FromCSSTOptions{})
}

// Baru: versi dengan options (dipakai controller untuk include=...)
func FromClassSectionSubjectTeacherModelsWithOptions(
	rows []csstModel.ClassSectionSubjectTeacherModel,
	opt FromCSSTOptions,
) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, fromClassSectionSubjectTeacherModelWithOptions(r, opt))
	}
	return out
}

// Lama: masih ada, tapi delegasi ke WithOptions tanpa include extra
func FromClassSectionSubjectTeacherModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	return FromClassSectionSubjectTeacherModelsWithOptions(rows, FromCSSTOptions{})
}

// Alias helper supaya konsisten dengan pemanggilan di controller:
// csstDto.FromCSSTModels(createdCSSTs)
func FromCSSTModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	return FromClassSectionSubjectTeacherModels(rows)
}

/*
	=========================================================
	  4) COMPACT / LITE DTO (digabung)

	- Compact jadi bentuk utama
	- CSSTItemLite dijadikan alias ke compact
=========================================================
*/

// Bentuk compact utama (dipakai di semua tempat: list, nested csst, dll)
type ClassSectionSubjectTeacherCompactResponse struct {
	// IDs & relations
	ClassSectionSubjectTeacherID              uuid.UUID  `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID        uuid.UUID  `json:"class_section_subject_teacher_school_id"`
	ClassSectionSubjectTeacherClassSectionID  uuid.UUID  `json:"class_section_subject_teacher_class_section_id"`
	ClassSectionSubjectTeacherClassSubjectID  uuid.UUID  `json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_school_teacher_id,omitempty"`
	// asisten
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id,omitempty"`

	// slug (selalu string, aman buat FE)
	ClassSectionSubjectTeacherSlug string `json:"class_section_subject_teacher_slug"`

	// delivery mode (enum string)
	ClassSectionSubjectTeacherDeliveryMode csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode"`

	// Agregat & kapasitas (diambil dari model)
	ClassSectionSubjectTeacherTotalAttendance          int  `json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherTotalMeetingsTarget      *int `json:"class_section_subject_teacher_total_meetings_target,omitempty"`
	ClassSectionSubjectTeacherTotalAssessments         int  `json:"class_section_subject_teacher_total_assessments"`
	ClassSectionSubjectTeacherTotalAssessmentsGraded   int  `json:"class_section_subject_teacher_total_assessments_graded"`
	ClassSectionSubjectTeacherTotalAssessmentsUngraded int  `json:"class_section_subject_teacher_total_assessments_ungraded"`
	ClassSectionSubjectTeacherTotalStudentsPassed      int  `json:"class_section_subject_teacher_total_students_passed"`

	// SECTION cache
	ClassSectionSubjectTeacherClassSectionSlugCache *string `json:"class_section_subject_teacher_class_section_slug_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionNameCache *string `json:"class_section_subject_teacher_class_section_name_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionCodeCache *string `json:"class_section_subject_teacher_class_section_code_cache,omitempty"`

	// TEACHER cache (JSONB – tetap datatypes.JSON, sesuai permintaan JSONB)
	ClassSectionSubjectTeacherSchoolTeacherSlugCache *string         `json:"class_section_subject_teacher_school_teacher_slug_cache,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherCache     *datatypes.JSON `json:"class_section_subject_teacher_school_teacher_cache,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherNameCache *string         `json:"class_section_subject_teacher_school_teacher_name_cache,omitempty"`

	// Assistant teacher cache
	ClassSectionSubjectTeacherAssistantSchoolTeacherSlugCache *string         `json:"class_section_subject_teacher_assistant_school_teacher_slug_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherCache     *datatypes.JSON `json:"class_section_subject_teacher_assistant_school_teacher_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache *string         `json:"class_section_subject_teacher_assistant_school_teacher_name_cache,omitempty"`

	// SUBJECT cache
	ClassSectionSubjectTeacherSubjectID        *uuid.UUID `json:"class_section_subject_teacher_subject_id,omitempty"`
	ClassSectionSubjectTeacherSubjectNameCache *string    `json:"class_section_subject_teacher_subject_name_cache,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeCache *string    `json:"class_section_subject_teacher_subject_code_cache,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugCache *string    `json:"class_section_subject_teacher_subject_slug_cache,omitempty"`

	// Status & audit
	ClassSectionSubjectTeacherStatus      string     `json:"class_section_subject_teacher_status"`
	ClassSectionSubjectTeacherIsActive    bool       `json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCompletedAt *time.Time `json:"class_section_subject_teacher_completed_at,omitempty"`
	ClassSectionSubjectTeacherCreatedAt   time.Time  `json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt   time.Time  `json:"class_section_subject_teacher_updated_at"`
}

// mapping single → compact
func FromClassSectionSubjectTeacherModelCompact(mo csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherCompactResponse {
	// manual safe-string (ganti helper.SafeStrPtr)
	slug := ""
	if mo.ClassSectionSubjectTeacherSlug != nil {
		slug = *mo.ClassSectionSubjectTeacherSlug
	}

	// status & aktif (derived)
	statusStr := string(mo.ClassSectionSubjectTeacherStatus)
	if statusStr == "" {
		statusStr = "active"
	}
	isActive := mo.ClassSectionSubjectTeacherStatus == csstModel.ClassStatusActive

	return ClassSectionSubjectTeacherCompactResponse{
		ClassSectionSubjectTeacherID:                       mo.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherSchoolID:                 mo.ClassSectionSubjectTeacherSchoolID,
		ClassSectionSubjectTeacherClassSectionID:           mo.ClassSectionSubjectTeacherClassSectionID,
		ClassSectionSubjectTeacherClassSubjectID:           mo.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherSchoolTeacherID:          mo.ClassSectionSubjectTeacherSchoolTeacherID,
		ClassSectionSubjectTeacherAssistantSchoolTeacherID: mo.ClassSectionSubjectTeacherAssistantSchoolTeacherID,

		ClassSectionSubjectTeacherSlug:         slug,
		ClassSectionSubjectTeacherDeliveryMode: mo.ClassSectionSubjectTeacherDeliveryMode,

		ClassSectionSubjectTeacherTotalAttendance:          mo.ClassSectionSubjectTeacherTotalAttendance,
		ClassSectionSubjectTeacherTotalMeetingsTarget:      mo.ClassSectionSubjectTeacherTotalMeetingsTarget,
		ClassSectionSubjectTeacherTotalAssessments:         mo.ClassSectionSubjectTeacherTotalAssessments,
		ClassSectionSubjectTeacherTotalAssessmentsGraded:   mo.ClassSectionSubjectTeacherTotalAssessmentsGraded,
		ClassSectionSubjectTeacherTotalAssessmentsUngraded: mo.ClassSectionSubjectTeacherTotalAssessmentsUngraded,
		ClassSectionSubjectTeacherTotalStudentsPassed:      mo.ClassSectionSubjectTeacherTotalStudentsPassed,

		ClassSectionSubjectTeacherClassSectionSlugCache: mo.ClassSectionSubjectTeacherClassSectionSlugCache,
		ClassSectionSubjectTeacherClassSectionNameCache: mo.ClassSectionSubjectTeacherClassSectionNameCache,
		ClassSectionSubjectTeacherClassSectionCodeCache: mo.ClassSectionSubjectTeacherClassSectionCodeCache,

		ClassSectionSubjectTeacherSchoolTeacherSlugCache: mo.ClassSectionSubjectTeacherSchoolTeacherSlugCache,
		ClassSectionSubjectTeacherSchoolTeacherCache:     mo.ClassSectionSubjectTeacherSchoolTeacherCache,
		ClassSectionSubjectTeacherSchoolTeacherNameCache: mo.ClassSectionSubjectTeacherSchoolTeacherNameCache,

		ClassSectionSubjectTeacherAssistantSchoolTeacherSlugCache: mo.ClassSectionSubjectTeacherAssistantSchoolTeacherSlugCache,
		ClassSectionSubjectTeacherAssistantSchoolTeacherCache:     mo.ClassSectionSubjectTeacherAssistantSchoolTeacherCache,
		ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache: mo.ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache,

		ClassSectionSubjectTeacherSubjectID:        mo.ClassSectionSubjectTeacherSubjectID,
		ClassSectionSubjectTeacherSubjectNameCache: mo.ClassSectionSubjectTeacherSubjectNameCache,
		ClassSectionSubjectTeacherSubjectCodeCache: mo.ClassSectionSubjectTeacherSubjectCodeCache,
		ClassSectionSubjectTeacherSubjectSlugCache: mo.ClassSectionSubjectTeacherSubjectSlugCache,

		ClassSectionSubjectTeacherStatus:      statusStr,
		ClassSectionSubjectTeacherIsActive:    isActive,
		ClassSectionSubjectTeacherCompletedAt: mo.ClassSectionSubjectTeacherCompletedAt,
		ClassSectionSubjectTeacherCreatedAt:   mo.ClassSectionSubjectTeacherCreatedAt,
		ClassSectionSubjectTeacherUpdatedAt:   mo.ClassSectionSubjectTeacherUpdatedAt,
	}
}

// mapping list → compact
func FromClassSectionSubjectTeacherModelsCompact(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherCompactResponse {
	out := make([]ClassSectionSubjectTeacherCompactResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromClassSectionSubjectTeacherModelCompact(rows[i]))
	}
	return out
}

// ===============================
// Alias: "lite" = compact
// ===============================

// Semua pemanggilan lama yang pakai CSSTItemLite akan mendapatkan bentuk compact ini
type CSSTItemLite = ClassSectionSubjectTeacherCompactResponse

func CSSTLiteFromModel(m *csstModel.ClassSectionSubjectTeacherModel) CSSTItemLite {
	if m == nil {
		return CSSTItemLite{}
	}
	return FromClassSectionSubjectTeacherModelCompact(*m)
}

func CSSTLiteSliceFromModels(list []csstModel.ClassSectionSubjectTeacherModel) []CSSTItemLite {
	out := make([]CSSTItemLite, 0, len(list))
	for i := range list {
		out = append(out, FromClassSectionSubjectTeacherModelCompact(list[i]))
	}
	return out
}
