// file: internals/features/school/classes/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	teacherSnap "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/service"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	// TZ-aware helpers
	"madinahsalam_backend/internals/helpers/dbtime"
)

/* =========================================================
   OPTIONS & NESTED TYPES
========================================================= */

type FromCSSTOptions struct {
	IncludeAcademicTerm bool
}

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

func TeacherCacheFromJSON(j *datatypes.JSON) *teacherSnap.TeacherCache {
	if j == nil {
		return nil
	}
	raw := []byte(*j)
	if len(raw) == 0 {
		return nil
	}
	if strings.TrimSpace(string(raw)) == "null" {
		return nil
	}

	var ts teacherSnap.TeacherCache
	if err := json.Unmarshal(raw, &ts); err != nil {
		return nil
	}

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
   1) REQUEST DTO (FOLLOW SQL/MODEL TERBARU: csst_*)
========================================================= */

// Create
type CreateCSSTRequest struct {
	// biasanya diisi dari auth context
	CSSTSchoolID *uuid.UUID `json:"csst_school_id" validate:"omitempty,uuid"`

	// relasi utama
	CSSTClassSectionID uuid.UUID `json:"csst_class_section_id" validate:"required,uuid"`
	CSSTClassSubjectID uuid.UUID `json:"csst_class_subject_id" validate:"required,uuid"`

	// teacher wajib (manual create)
	CSSTSchoolTeacherID uuid.UUID `json:"csst_school_teacher_id" validate:"required,uuid"`

	// asisten opsional
	CSSTAssistantSchoolTeacherID *uuid.UUID `json:"csst_assistant_school_teacher_id" validate:"omitempty,uuid"`

	// opsional basic
	CSSTSlug        *string    `json:"csst_slug" validate:"omitempty,max=160"`
	CSSTDescription *string    `json:"csst_description" validate:"omitempty"`
	CSSTClassRoomID *uuid.UUID `json:"csst_class_room_id" validate:"omitempty,uuid"`
	CSSTGroupURL    *string    `json:"csst_group_url" validate:"omitempty,max=2000"`

	// quota_total — >=0 divalidasi DB
	CSSTQuotaTotal *int `json:"csst_quota_total" validate:"omitempty"`

	// enum: offline|online|hybrid
	CSSTDeliveryMode *csstModel.ClassDeliveryMode `json:"csst_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	// enum: teacher_only | student_only | both
	CSSTSchoolAttendanceEntryModeCache *csstModel.AttendanceEntryMode `json:"csst_school_attendance_entry_mode_cache" validate:"omitempty,oneof=teacher_only student_only both"`

	// target pertemuan
	CSSTTotalMeetingsTarget *int `json:"csst_total_meetings_target" validate:"omitempty"`

	// status enum
	CSSTStatus *csstModel.ClassStatus `json:"csst_status" validate:"omitempty,oneof=active inactive completed"`
}

// Update (partial)
type UpdateCSSTRequest struct {
	CSSTSchoolID        *uuid.UUID `json:"csst_school_id" validate:"omitempty,uuid"`
	CSSTClassSectionID  *uuid.UUID `json:"csst_class_section_id" validate:"omitempty,uuid"`
	CSSTClassSubjectID  *uuid.UUID `json:"csst_class_subject_id" validate:"omitempty,uuid"`
	CSSTSchoolTeacherID *uuid.UUID `json:"csst_school_teacher_id" validate:"omitempty,uuid"`

	CSSTAssistantSchoolTeacherID *uuid.UUID `json:"csst_assistant_school_teacher_id" validate:"omitempty,uuid"`

	CSSTSlug        *string    `json:"csst_slug" validate:"omitempty,max=160"`
	CSSTDescription *string    `json:"csst_description" validate:"omitempty"`
	CSSTClassRoomID *uuid.UUID `json:"csst_class_room_id" validate:"omitempty,uuid"`
	CSSTGroupURL    *string    `json:"csst_group_url" validate:"omitempty,max=2000"`

	CSSTQuotaTotal   *int                         `json:"csst_quota_total" validate:"omitempty"`
	CSSTDeliveryMode *csstModel.ClassDeliveryMode `json:"csst_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	CSSTSchoolAttendanceEntryModeCache *csstModel.AttendanceEntryMode `json:"csst_school_attendance_entry_mode_cache" validate:"omitempty,oneof=teacher_only student_only both"`

	CSSTTotalMeetingsTarget *int `json:"csst_total_meetings_target" validate:"omitempty"`

	CSSTStatus *csstModel.ClassStatus `json:"csst_status" validate:"omitempty,oneof=active inactive completed"`
}

/* =========================================================
   2) RESPONSE DTO — sinkron SQL/model terbaru (csst_*)
========================================================= */

type CSSTResponse struct {
	/* ===== IDs & Relations ===== */
	CSSTID       uuid.UUID `json:"csst_id"`
	CSSTSchoolID uuid.UUID `json:"csst_school_id"`

	CSSTClassSectionID           uuid.UUID  `json:"csst_class_section_id"`
	CSSTClassSubjectID           uuid.UUID  `json:"csst_class_subject_id"`
	CSSTSchoolTeacherID          *uuid.UUID `json:"csst_school_teacher_id,omitempty"`
	CSSTAssistantSchoolTeacherID *uuid.UUID `json:"csst_assistant_school_teacher_id,omitempty"`
	CSSTClassRoomID              *uuid.UUID `json:"csst_class_room_id,omitempty"`

	/* ===== Identitas & Fasilitas ===== */
	CSSTSlug        *string `json:"csst_slug,omitempty"`
	CSSTDescription *string `json:"csst_description,omitempty"`
	CSSTGroupURL    *string `json:"csst_group_url,omitempty"`

	/* ===== Agregat & quota ===== */
	CSSTTotalAttendance     int  `json:"csst_total_attendance"`
	CSSTTotalMeetingsTarget *int `json:"csst_total_meetings_target,omitempty"`
	CSSTQuotaTotal          *int `json:"csst_quota_total,omitempty"`
	CSSTQuotaTaken          int  `json:"csst_quota_taken"`

	CSSTTotalAssessments          int `json:"csst_total_assessments"`
	CSSTTotalAssessmentsTraining  int `json:"csst_total_assessments_training"`
	CSSTTotalAssessmentsDailyExam int `json:"csst_total_assessments_daily_exam"`
	CSSTTotalAssessmentsExam      int `json:"csst_total_assessments_exam"`
	CSSTTotalStudentsPassed       int `json:"csst_total_students_passed"`

	CSSTDeliveryMode string `json:"csst_delivery_mode"`

	/* ===== Attendance mode cache ===== */
	CSSTSchoolAttendanceEntryModeCache *string `json:"csst_school_attendance_entry_mode_cache,omitempty"`

	/* ===== SECTION caches ===== */
	CSSTClassSectionSlugCache *string `json:"csst_class_section_slug_cache,omitempty"`
	CSSTClassSectionNameCache *string `json:"csst_class_section_name_cache,omitempty"`
	CSSTClassSectionCodeCache *string `json:"csst_class_section_code_cache,omitempty"`
	CSSTClassSectionURLCache  *string `json:"csst_class_section_url_cache,omitempty"`

	/* ===== ROOM cache ===== */
	CSSTClassRoomSlugCache *string         `json:"csst_class_room_slug_cache,omitempty"`
	CSSTClassRoomCache     *datatypes.JSON `json:"csst_class_room_cache,omitempty"`
	// generated
	CSSTClassRoomNameCache     *string `json:"csst_class_room_name_cache,omitempty"`
	CSSTClassRoomSlugCacheGen  *string `json:"csst_class_room_slug_cache_gen,omitempty"`
	CSSTClassRoomLocationCache *string `json:"csst_class_room_location_cache,omitempty"`

	/* ===== PEOPLE caches ===== */
	CSSTSchoolTeacherSlugCache          *string                   `json:"csst_school_teacher_slug_cache,omitempty"`
	CSSTSchoolTeacherCache              *teacherSnap.TeacherCache `json:"csst_school_teacher_cache,omitempty"`
	CSSTAssistantSchoolTeacherSlugCache *string                   `json:"csst_assistant_school_teacher_slug_cache,omitempty"`
	CSSTAssistantSchoolTeacherCache     *teacherSnap.TeacherCache `json:"csst_assistant_school_teacher_cache,omitempty"`
	// generated
	CSSTSchoolTeacherNameCache          *string `json:"csst_school_teacher_name_cache,omitempty"`
	CSSTAssistantSchoolTeacherNameCache *string `json:"csst_assistant_school_teacher_name_cache,omitempty"`

	/* ===== SUBJECT cache ===== */
	CSSTTotalBooks       int        `json:"csst_total_books"`
	CSSTSubjectID        *uuid.UUID `json:"csst_subject_id,omitempty"`
	CSSTSubjectNameCache *string    `json:"csst_subject_name_cache,omitempty"`
	CSSTSubjectCodeCache *string    `json:"csst_subject_code_cache,omitempty"`
	CSSTSubjectSlugCache *string    `json:"csst_subject_slug_cache,omitempty"`

	/* ===== ACADEMIC_TERM cache ===== */
	CSSTAcademicTermID            *uuid.UUID `json:"csst_academic_term_id,omitempty"`
	CSSTAcademicTermNameCache     *string    `json:"csst_academic_term_name_cache,omitempty"`
	CSSTAcademicTermSlugCache     *string    `json:"csst_academic_term_slug_cache,omitempty"`
	CSSTAcademicYearCache         *string    `json:"csst_academic_year_cache,omitempty"`
	CSSTAcademicTermAngkatanCache *int       `json:"csst_academic_term_angkatan_cache,omitempty"`

	/* ===== KKM cache ===== */
	CSSTMinPassingScoreClassSubjectCache *int `json:"csst_min_passing_score_class_subject_cache,omitempty"`

	/* ===== Status & audit ===== */
	CSSTStatus      csstModel.ClassStatus `json:"csst_status"`
	CSSTCompletedAt *time.Time            `json:"csst_completed_at,omitempty"`
	CSSTCreatedAt   time.Time             `json:"csst_created_at"`
	CSSTUpdatedAt   time.Time             `json:"csst_updated_at"`
	CSSTDeletedAt   *time.Time            `json:"csst_deleted_at,omitempty"`

	AcademicTerm *AcademicTermLite `json:"academic_term,omitempty"`
}

/* =================== TZ Helpers =================== */

func (r CSSTResponse) WithSchoolTime(c *fiber.Ctx) CSSTResponse {
	out := r
	out.CSSTCreatedAt = dbtime.ToSchoolTime(c, r.CSSTCreatedAt)
	out.CSSTUpdatedAt = dbtime.ToSchoolTime(c, r.CSSTUpdatedAt)
	out.CSSTCompletedAt = dbtime.ToSchoolTimePtr(c, r.CSSTCompletedAt)
	out.CSSTDeletedAt = dbtime.ToSchoolTimePtr(c, r.CSSTDeletedAt)
	return out
}

func FromCSSTModelWithSchoolTime(
	c *fiber.Ctx,
	m csstModel.ClassSectionSubjectTeacherModel,
) CSSTResponse {
	return FromCSSTModel(m).WithSchoolTime(c)
}

/* =========================================================
   3) MAPPERS
========================================================= */

func (r CreateCSSTRequest) ToModel() csstModel.ClassSectionSubjectTeacherModel {
	teacherID := r.CSSTSchoolTeacherID

	m := csstModel.ClassSectionSubjectTeacherModel{
		CSSTClassSectionID: r.CSSTClassSectionID,
		CSSTClassSubjectID: r.CSSTClassSubjectID,

		// teacher wajib saat manual create
		CSSTSchoolTeacherID: &teacherID,

		CSSTSlug:        trimLowerPtr(r.CSSTSlug),
		CSSTDescription: trimPtr(r.CSSTDescription),
		CSSTClassRoomID: r.CSSTClassRoomID,
		CSSTGroupURL:    trimPtr(r.CSSTGroupURL),
	}

	if r.CSSTAssistantSchoolTeacherID != nil {
		m.CSSTAssistantSchoolTeacherID = r.CSSTAssistantSchoolTeacherID
	}
	if r.CSSTSchoolID != nil {
		m.CSSTSchoolID = *r.CSSTSchoolID
	}
	if r.CSSTQuotaTotal != nil {
		m.CSSTQuotaTotal = r.CSSTQuotaTotal
	}
	if r.CSSTDeliveryMode != nil {
		m.CSSTDeliveryMode = *r.CSSTDeliveryMode
	}
	if r.CSSTTotalMeetingsTarget != nil {
		m.CSSTTotalMeetingsTarget = r.CSSTTotalMeetingsTarget
	}
	if r.CSSTSchoolAttendanceEntryModeCache != nil {
		m.CSSTSchoolAttendanceEntryModeCache = r.CSSTSchoolAttendanceEntryModeCache
	}

	status := csstModel.ClassStatusActive
	if r.CSSTStatus != nil {
		status = *r.CSSTStatus
	}
	m.CSSTStatus = status

	return m
}

func (r UpdateCSSTRequest) Apply(m *csstModel.ClassSectionSubjectTeacherModel) {
	if r.CSSTSchoolID != nil {
		m.CSSTSchoolID = *r.CSSTSchoolID
	}
	if r.CSSTClassSectionID != nil {
		m.CSSTClassSectionID = *r.CSSTClassSectionID
	}
	if r.CSSTClassSubjectID != nil {
		m.CSSTClassSubjectID = *r.CSSTClassSubjectID
	}
	if r.CSSTSchoolTeacherID != nil {
		m.CSSTSchoolTeacherID = r.CSSTSchoolTeacherID
	}
	if r.CSSTAssistantSchoolTeacherID != nil {
		m.CSSTAssistantSchoolTeacherID = r.CSSTAssistantSchoolTeacherID
	}
	if r.CSSTSlug != nil {
		m.CSSTSlug = trimLowerPtr(r.CSSTSlug)
	}
	if r.CSSTDescription != nil {
		m.CSSTDescription = trimPtr(r.CSSTDescription)
	}
	if r.CSSTClassRoomID != nil {
		m.CSSTClassRoomID = r.CSSTClassRoomID
	}
	if r.CSSTGroupURL != nil {
		m.CSSTGroupURL = trimPtr(r.CSSTGroupURL)
	}
	if r.CSSTQuotaTotal != nil {
		m.CSSTQuotaTotal = r.CSSTQuotaTotal
	}
	if r.CSSTDeliveryMode != nil {
		m.CSSTDeliveryMode = *r.CSSTDeliveryMode
	}
	if r.CSSTTotalMeetingsTarget != nil {
		m.CSSTTotalMeetingsTarget = r.CSSTTotalMeetingsTarget
	}
	if r.CSSTSchoolAttendanceEntryModeCache != nil {
		m.CSSTSchoolAttendanceEntryModeCache = r.CSSTSchoolAttendanceEntryModeCache
	}
	if r.CSSTStatus != nil {
		m.CSSTStatus = *r.CSSTStatus
	}
}

func fromCSSTModelWithOptions(
	m csstModel.ClassSectionSubjectTeacherModel,
	opt FromCSSTOptions,
) CSSTResponse {
	var deletedAt *time.Time
	if m.CSSTDeletedAt.Valid {
		t := m.CSSTDeletedAt.Time
		deletedAt = &t
	}

	var attendanceCache *string
	if m.CSSTSchoolAttendanceEntryModeCache != nil {
		v := string(*m.CSSTSchoolAttendanceEntryModeCache)
		attendanceCache = &v
	}

	status := m.CSSTStatus
	if status == "" {
		status = csstModel.ClassStatusActive
	}

	resp := CSSTResponse{
		CSSTID:       m.CSSTID,
		CSSTSchoolID: m.CSSTSchoolID,

		CSSTClassSectionID:           m.CSSTClassSectionID,
		CSSTClassSubjectID:           m.CSSTClassSubjectID,
		CSSTSchoolTeacherID:          m.CSSTSchoolTeacherID,
		CSSTAssistantSchoolTeacherID: m.CSSTAssistantSchoolTeacherID,
		CSSTClassRoomID:              m.CSSTClassRoomID,

		CSSTSlug:        m.CSSTSlug,
		CSSTDescription: m.CSSTDescription,
		CSSTGroupURL:    m.CSSTGroupURL,

		CSSTTotalAttendance:     m.CSSTTotalAttendance,
		CSSTTotalMeetingsTarget: m.CSSTTotalMeetingsTarget,
		CSSTQuotaTotal:          m.CSSTQuotaTotal,
		CSSTQuotaTaken:          m.CSSTQuotaTaken,

		CSSTTotalAssessments:          m.CSSTTotalAssessments,
		CSSTTotalAssessmentsTraining:  m.CSSTTotalAssessmentsTrain,
		CSSTTotalAssessmentsDailyExam: m.CSSTTotalAssessmentsDaily,
		CSSTTotalAssessmentsExam:      m.CSSTTotalAssessmentsExam,
		CSSTTotalStudentsPassed:       m.CSSTTotalStudentsPassed,

		CSSTDeliveryMode: string(m.CSSTDeliveryMode),

		CSSTSchoolAttendanceEntryModeCache: attendanceCache,

		CSSTClassSectionSlugCache: m.CSSTClassSectionSlugCache,
		CSSTClassSectionNameCache: m.CSSTClassSectionNameCache,
		CSSTClassSectionCodeCache: m.CSSTClassSectionCodeCache,
		CSSTClassSectionURLCache:  m.CSSTClassSectionURLCache,

		CSSTClassRoomSlugCache:     m.CSSTClassRoomSlugCache,
		CSSTClassRoomCache:         m.CSSTClassRoomCache,
		CSSTClassRoomNameCache:     m.CSSTClassRoomNameCache,
		CSSTClassRoomSlugCacheGen:  m.CSSTClassRoomSlugCacheGen,
		CSSTClassRoomLocationCache: m.CSSTClassRoomLocationCache,

		CSSTSchoolTeacherSlugCache:          m.CSSTSchoolTeacherSlugCache,
		CSSTSchoolTeacherCache:              TeacherCacheFromJSON(m.CSSTSchoolTeacherCache),
		CSSTAssistantSchoolTeacherSlugCache: m.CSSTAssistantSchoolTeacherSlugCache,
		CSSTAssistantSchoolTeacherCache:     TeacherCacheFromJSON(m.CSSTAssistantSchoolTeacherCache),
		CSSTSchoolTeacherNameCache:          m.CSSTSchoolTeacherNameCache,
		CSSTAssistantSchoolTeacherNameCache: m.CSSTAssistantSchoolTeacherNameCache,

		CSSTTotalBooks:       m.CSSTTotalBooks,
		CSSTSubjectID:        m.CSSTSubjectID,
		CSSTSubjectNameCache: m.CSSTSubjectNameCache,
		CSSTSubjectCodeCache: m.CSSTSubjectCodeCache,
		CSSTSubjectSlugCache: m.CSSTSubjectSlugCache,

		CSSTAcademicTermID:            m.CSSTAcademicTermID,
		CSSTAcademicTermNameCache:     m.CSSTAcademicTermNameCache,
		CSSTAcademicTermSlugCache:     m.CSSTAcademicTermSlugCache,
		CSSTAcademicYearCache:         m.CSSTAcademicYearCache,
		CSSTAcademicTermAngkatanCache: m.CSSTAcademicTermAngkatanCache,

		CSSTMinPassingScoreClassSubjectCache: m.CSSTMinPassingScoreClassSubjectCache,

		CSSTStatus:      status,
		CSSTCompletedAt: m.CSSTCompletedAt,
		CSSTCreatedAt:   m.CSSTCreatedAt,
		CSSTUpdatedAt:   m.CSSTUpdatedAt,
		CSSTDeletedAt:   deletedAt,
	}

	if opt.IncludeAcademicTerm && m.CSSTAcademicTermID != nil {
		resp.AcademicTerm = &AcademicTermLite{
			ID:       m.CSSTAcademicTermID,
			Name:     m.CSSTAcademicTermNameCache,
			Slug:     m.CSSTAcademicTermSlugCache,
			Year:     m.CSSTAcademicYearCache,
			Angkatan: m.CSSTAcademicTermAngkatanCache,
		}
	}

	return resp
}

func FromCSSTModel(m csstModel.ClassSectionSubjectTeacherModel) CSSTResponse {
	return fromCSSTModelWithOptions(m, FromCSSTOptions{})
}

func FromCSSTModelsWithOptions(
	rows []csstModel.ClassSectionSubjectTeacherModel,
	opt FromCSSTOptions,
) []CSSTResponse {
	out := make([]CSSTResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, fromCSSTModelWithOptions(r, opt))
	}
	return out
}

func FromCSSTModels(rows []csstModel.ClassSectionSubjectTeacherModel) []CSSTResponse {
	return FromCSSTModelsWithOptions(rows, FromCSSTOptions{})
}

/*
	=========================================================
	  4) COMPACT / LITE DTO (csst_*)

=========================================================
*/
type CSSTCompactResponse struct {
	CSSTID uuid.UUID `json:"csst_id"`

	CSSTClassSubjectID  uuid.UUID  `json:"csst_class_subject_id"`
	CSSTSchoolTeacherID *uuid.UUID `json:"csst_school_teacher_id,omitempty"`

	CSSTDeliveryMode csstModel.ClassDeliveryMode `json:"csst_delivery_mode"`

	CSSTClassSectionSlugCache *string `json:"csst_class_section_slug_cache,omitempty"`
	CSSTClassSectionNameCache *string `json:"csst_class_section_name_cache,omitempty"`
	CSSTClassSectionCodeCache *string `json:"csst_class_section_code_cache,omitempty"`

	CSSTSchoolTeacherSlugCache *string                   `json:"csst_school_teacher_slug_cache,omitempty"`
	CSSTSchoolTeacherCache     *teacherSnap.TeacherCache `json:"csst_school_teacher_cache,omitempty"`

	CSSTSubjectID        *uuid.UUID `json:"csst_subject_id,omitempty"`
	CSSTSubjectNameCache *string    `json:"csst_subject_name_cache,omitempty"`
	CSSTSubjectCodeCache *string    `json:"csst_subject_code_cache,omitempty"`
	CSSTSubjectSlugCache *string    `json:"csst_subject_slug_cache,omitempty"`

	CSSTStatus      csstModel.ClassStatus `json:"csst_status"`
	CSSTCompletedAt *time.Time            `json:"csst_completed_at,omitempty"`
	CSSTCreatedAt   time.Time             `json:"csst_created_at"`
	CSSTUpdatedAt   time.Time             `json:"csst_updated_at"`
}

func (r CSSTCompactResponse) WithSchoolTime(c *fiber.Ctx) CSSTCompactResponse {
	out := r
	out.CSSTCreatedAt = dbtime.ToSchoolTime(c, r.CSSTCreatedAt)
	out.CSSTUpdatedAt = dbtime.ToSchoolTime(c, r.CSSTUpdatedAt)
	out.CSSTCompletedAt = dbtime.ToSchoolTimePtr(c, r.CSSTCompletedAt)
	return out
}

func FromCSSTModelCompact(mo csstModel.ClassSectionSubjectTeacherModel) CSSTCompactResponse {
	status := mo.CSSTStatus
	if status == "" {
		status = csstModel.ClassStatusActive
	}

	return CSSTCompactResponse{
		CSSTID: mo.CSSTID,

		CSSTClassSubjectID:  mo.CSSTClassSubjectID,
		CSSTSchoolTeacherID: mo.CSSTSchoolTeacherID,

		CSSTDeliveryMode: mo.CSSTDeliveryMode,

		CSSTClassSectionSlugCache: mo.CSSTClassSectionSlugCache,
		CSSTClassSectionNameCache: mo.CSSTClassSectionNameCache,
		CSSTClassSectionCodeCache: mo.CSSTClassSectionCodeCache,

		CSSTSchoolTeacherSlugCache: mo.CSSTSchoolTeacherSlugCache,
		CSSTSchoolTeacherCache:     TeacherCacheFromJSON(mo.CSSTSchoolTeacherCache),

		CSSTSubjectID:        mo.CSSTSubjectID,
		CSSTSubjectNameCache: mo.CSSTSubjectNameCache,
		CSSTSubjectCodeCache: mo.CSSTSubjectCodeCache,
		CSSTSubjectSlugCache: mo.CSSTSubjectSlugCache,

		CSSTStatus:      status,
		CSSTCompletedAt: mo.CSSTCompletedAt,
		CSSTCreatedAt:   mo.CSSTCreatedAt,
		CSSTUpdatedAt:   mo.CSSTUpdatedAt,
	}
}

func FromCSSTModelsCompact(rows []csstModel.ClassSectionSubjectTeacherModel) []CSSTCompactResponse {
	out := make([]CSSTCompactResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromCSSTModelCompact(rows[i]))
	}
	return out
}

func FromCSSTModelCompactWithSchoolTime(c *fiber.Ctx, m csstModel.ClassSectionSubjectTeacherModel) CSSTCompactResponse {
	return FromCSSTModelCompact(m).WithSchoolTime(c)
}

func FromCSSTModelsCompactWithSchoolTime(
	c *fiber.Ctx,
	rows []csstModel.ClassSectionSubjectTeacherModel,
) []CSSTCompactResponse {
	out := make([]CSSTCompactResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromCSSTModelCompact(rows[i]).WithSchoolTime(c))
	}
	return out
}

// alias lite = compact
type CSSTItemLite = CSSTCompactResponse

func CSSTLiteFromModel(m *csstModel.ClassSectionSubjectTeacherModel) CSSTItemLite {
	if m == nil {
		return CSSTItemLite{}
	}
	return FromCSSTModelCompact(*m)
}

func CSSTLiteSliceFromModels(list []csstModel.ClassSectionSubjectTeacherModel) []CSSTItemLite {
	out := make([]CSSTItemLite, 0, len(list))
	for i := range list {
		out = append(out, FromCSSTModelCompact(list[i]))
	}
	return out
}
