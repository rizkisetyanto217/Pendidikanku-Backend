// file: internals/features/school/sectionsubjectteachers/dto/student_class_section_subject_teacher_dto.go
package dto

import (
	"time"

	model "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	dbtime "madinahsalam_backend/internals/helpers/dbtime"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =========================================================
   UTIL & COMMON
========================================================= */

// PatchFieldStudentCSST → tri-state update
type PatchFieldStudentCSST[T any] struct {
	Set    bool `json:"-"`
	IsNull bool `json:"-"`
	Value  T    `json:"-"`
}

func PF[T any](v T) PatchFieldStudentCSST[T] { return PatchFieldStudentCSST[T]{Set: true, Value: v} }
func PFNull[T any]() PatchFieldStudentCSST[T] {
	return PatchFieldStudentCSST[T]{Set: true, IsNull: true}
}
func PFUnset[T any]() PatchFieldStudentCSST[T] { return PatchFieldStudentCSST[T]{Set: false} }

/* =========================================================
   REQUEST: PATH / BODY HELPERS
========================================================= */

type IDParam struct {
	ID uuid.UUID `json:"id" params:"id" validate:"required"`
}

type BulkIDsRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,dive,required"`
}

/* =========================================================
   CREATE / UPSERT
========================================================= */

type StudentCSSTCreateRequest struct {
	CSSTID    uuid.UUID  `json:"csst_id" validate:"required"`
	StudentID uuid.UUID  `json:"student_id" validate:"required"`
	IsActive  *bool      `json:"is_active,omitempty"`
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`

	// optional notes on create
	StudentNotes        *string `json:"student_notes,omitempty"`
	HomeroomNotes       *string `json:"homeroom_notes,omitempty"`
	SubjectTeacherNotes *string `json:"subject_teacher_notes,omitempty"`

	IdempotencyKey *string `json:"idempotency_key,omitempty" validate:"omitempty,max=120"`
}

type StudentCSSTBulkCreateItem struct {
	CSSTID    uuid.UUID  `json:"csst_id" validate:"required"`
	StudentID uuid.UUID  `json:"student_id" validate:"required"`
	IsActive  *bool      `json:"is_active,omitempty"`
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`

	// optional notes for bulk create
	StudentNotes        *string `json:"student_notes,omitempty"`
	HomeroomNotes       *string `json:"homeroom_notes,omitempty"`
	SubjectTeacherNotes *string `json:"subject_teacher_notes,omitempty"`

	ClientRef *string `json:"client_ref,omitempty" validate:"omitempty,max=120"`
}

type StudentCSSTBulkCreateRequest struct {
	Items          []StudentCSSTBulkCreateItem `json:"items" validate:"required,min=1,dive"`
	IdempotencyKey *string                     `json:"idempotency_key,omitempty" validate:"omitempty,max=120"`
	SkipDuplicates bool                        `json:"skip_duplicates,omitempty"`
	ReturnExisting bool                        `json:"return_existing,omitempty"`
}

type StudentCSSTUpsertRequest struct {
	CSSTID    uuid.UUID  `json:"csst_id" validate:"required"`
	StudentID uuid.UUID  `json:"student_id" validate:"required"`
	IsActive  *bool      `json:"is_active,omitempty"`
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`

	// optional notes on upsert
	StudentNotes        *string `json:"student_notes,omitempty"`
	HomeroomNotes       *string `json:"homeroom_notes,omitempty"`
	SubjectTeacherNotes *string `json:"subject_teacher_notes,omitempty"`
}

/* =========================================================
   UPDATE / PATCH
========================================================= */

type StudentCSSTPatchRequest struct {
	CSSTID    PatchFieldStudentCSST[uuid.UUID]  `json:"csst_id,omitempty"`
	StudentID PatchFieldStudentCSST[uuid.UUID]  `json:"student_id,omitempty"`
	IsActive  PatchFieldStudentCSST[bool]       `json:"is_active,omitempty"`
	From      PatchFieldStudentCSST[*time.Time] `json:"from,omitempty"`
	To        PatchFieldStudentCSST[*time.Time] `json:"to,omitempty"`

	// notes patch (tri-state)
	StudentNotes        PatchFieldStudentCSST[*string] `json:"student_notes,omitempty"`
	HomeroomNotes       PatchFieldStudentCSST[*string] `json:"homeroom_notes,omitempty"`
	SubjectTeacherNotes PatchFieldStudentCSST[*string] `json:"subject_teacher_notes,omitempty"`
}

type StudentCSSTBulkPatchRequest struct {
	IDs   []uuid.UUID             `json:"ids" validate:"required,min=1,dive,required"`
	Patch StudentCSSTPatchRequest `json:"patch" validate:"required"`
}

type StudentCSSTToggleActiveRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}

type StudentCSSTBulkToggleActiveRequest struct {
	IDs      []uuid.UUID `json:"ids" validate:"required,min=1,dive,required"`
	IsActive bool        `json:"is_active" validate:"required"`
}

/* =========================================================
   DELETE / RESTORE
========================================================= */

type StudentCSSTDeleteRequest struct {
	Force bool `json:"force,omitempty"`
}

type StudentCSSTRestoreRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,dive,required"`
}

/* =========================================================
   LIST / QUERY PARAMS
========================================================= */

const (
	StudentCSSTSortCreatedAt = "created_at"
	StudentCSSTSortUpdatedAt = "updated_at"
	StudentCSSTSortStudent   = "student_id"
	StudentCSSTSortCSST      = "csst_id"
)

type StudentCSSTListQuery struct {
	Page     int `query:"page" validate:"omitempty,min=1"`
	PageSize int `query:"page_size" validate:"omitempty,min=1,max=200"`

	StudentID      *uuid.UUID `query:"student_id"`
	CSSTID         *uuid.UUID `query:"csst_id"`
	IsActive       *bool      `query:"is_active"`
	IncludeDeleted bool       `query:"include_deleted"`

	Q *string `query:"q"`

	SortBy string `query:"sort_by"`
	Order  string `query:"order"`

	Mode string `query:"mode"` // "full" | "compact" (default: full)

	IncludeSection      bool `query:"include_section"`
	IncludeClassSubject bool `query:"include_class_subject"`
	IncludeTeacher      bool `query:"include_teacher"`
}

/* =========================================================
   RESPONSE MODELS (EXPANDED RELATIONS)
========================================================= */

type StudentBrief struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id,omitempty"`
	FullName string    `json:"full_name,omitempty"`
	Avatar   string    `json:"avatar,omitempty"`
}

type SectionBrief struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name,omitempty"`
}

type ClassSubjectBrief struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code,omitempty"`
	Name string    `json:"name,omitempty"`
}

type TeacherBrief struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id,omitempty"`
	FullName string    `json:"full_name,omitempty"`
	Email    string    `json:"email,omitempty"`
	Phone    string    `json:"phone,omitempty"`
}

/* =========================================================
   MAIN ITEM — mirror student_csst_* model
========================================================= */

type StudentCSSTItem struct {
	StudentCSSTID       uuid.UUID `json:"student_csst_id"`
	StudentCSSTSchoolID uuid.UUID `json:"student_csst_school_id"`

	StudentCSSTStudentID uuid.UUID `json:"student_csst_student_id"`
	StudentCSSTCSSTID    uuid.UUID `json:"student_csst_csst_id"`

	StudentCSSTIsActive bool       `json:"student_csst_is_active"`
	StudentCSSTFrom     *time.Time `json:"student_csst_from,omitempty"`
	StudentCSSTTo       *time.Time `json:"student_csst_to,omitempty"`

	StudentCSSTScoreTotal    *float64 `json:"student_csst_score_total,omitempty"`
	StudentCSSTScoreMaxTotal *float64 `json:"student_csst_score_max_total,omitempty"`
	StudentCSSTScorePercent  *float64 `json:"student_csst_score_percent,omitempty"`
	StudentCSSTGradeLetter   *string  `json:"student_csst_grade_letter,omitempty"`
	StudentCSSTGradePoint    *float64 `json:"student_csst_grade_point,omitempty"`
	StudentCSSTIsPassed      *bool    `json:"student_csst_is_passed,omitempty"`

	// diselaraskan dengan kolom di migration + model
	StudentCSSTUserProfileNameCache         *string        `json:"student_csst_user_profile_name_cache,omitempty"`
	StudentCSSTUserProfileAvatarURLCache    *string        `json:"student_csst_user_profile_avatar_url_cache,omitempty"`
	StudentCSSTUserProfileWhatsappURLCache  *string        `json:"student_csst_user_profile_wa_url_cache,omitempty"` // json tetap pakai _wa_ biar backwards compatible
	StudentCSSTUserProfileParentNameCache   *string        `json:"student_csst_user_profile_parent_name_cache,omitempty"`
	StudentCSSTUserProfileParentWAURLCache  *string        `json:"student_csst_user_profile_parent_wa_url_cache,omitempty"`
	StudentCSSTUserProfileGenderCache       *string        `json:"student_csst_user_profile_gender_cache,omitempty"`
	StudentCSSTSchoolStudentCodeCache       *string        `json:"student_csst_school_student_code_cache,omitempty"`
	StudentCSSTEditsHistory                 datatypes.JSON `json:"student_csst_edits_history"`
	StudentCSSTStudentNotes                 *string        `json:"student_csst_student_notes,omitempty"`
	StudentCSSTStudentNotesUpdatedAt        *time.Time     `json:"student_csst_student_notes_updated_at,omitempty"`
	StudentCSSTHomeroomNotes                *string        `json:"student_csst_homeroom_notes,omitempty"`
	StudentCSSTHomeroomNotesUpdatedAt       *time.Time     `json:"student_csst_homeroom_notes_updated_at,omitempty"`
	StudentCSSTSubjectTeacherNotes          *string        `json:"student_csst_subject_teacher_notes,omitempty"`
	StudentCSSTSubjectTeacherNotesUpdatedAt *time.Time     `json:"student_csst_subject_teacher_notes_updated_at,omitempty"`
	StudentCSSTSlug                         *string        `json:"student_csst_slug,omitempty"`
	StudentCSSTMeta                         datatypes.JSON `json:"student_csst_meta"`
	StudentCSSTCreatedAt                    time.Time      `json:"student_csst_created_at"`
	StudentCSSTUpdatedAt                    time.Time      `json:"student_csst_updated_at"`
	StudentCSSTDeletedAt                    *time.Time     `json:"student_csst_deleted_at,omitempty"`

	// Expanded relations
	Student      *StudentBrief      `json:"student,omitempty"`
	Section      *SectionBrief      `json:"section,omitempty"`
	ClassSubject *ClassSubjectBrief `json:"class_subject,omitempty"`
	Teacher      *TeacherBrief      `json:"teacher,omitempty"`
}

/* =========================================================
   RESPONSE WRAPPERS
========================================================= */

type PageMeta struct {
	Total       int64 `json:"total"`
	Page        int   `json:"page"`
	PageSize    int   `json:"page_size"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

type StudentCSSTDetailResponse struct {
	Data StudentCSSTItem `json:"data"`
}

type StudentCSSTListResponse struct {
	Data []StudentCSSTItem `json:"data"`
	Meta PageMeta          `json:"meta"`
}

type StudentCSSTCreateResponse struct {
	Data StudentCSSTItem `json:"data"`
}

type StudentCSSTBulkCreateResult struct {
	Item      StudentCSSTItem `json:"item"`
	ClientRef *string         `json:"client_ref,omitempty"`
	Duplicate bool            `json:"duplicate,omitempty"`
}

type StudentCSSTBulkCreateResponse struct {
	Results []StudentCSSTBulkCreateResult `json:"results"`
	Meta    struct {
		Inserted int `json:"inserted"`
		Skipped  int `json:"skipped"`
		Existing int `json:"existing"`
	} `json:"meta"`
}

type AffectedResponse struct {
	Affected int `json:"affected"`
}

/* =========================================================
   ERROR
========================================================= */

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error       string       `json:"error"`
	Description string       `json:"description,omitempty"`
	Fields      []FieldError `json:"fields,omitempty"`
}

/* =========================================================
   UPDATE NOTES REQUEST
========================================================= */

type StudentCSSTUpdateNotesRequest struct {
	// Notes:
	// - Jika diisi string -> set isi notes
	// - Jika dikirim null -> clear (set NULL di DB)
	// - Jika field tidak dikirim -> anggap invalid (wajib ada key-nya)
	Notes *string `json:"notes"` // optional: string atau null
}

/* =========================================================
   MAPPERS
========================================================= */

// Versi lama: mapper tanpa konversi timezone (pakai DB time apa adanya)
func FromStudentCSSTModel(m *model.StudentClassSectionSubjectTeacherModel) StudentCSSTItem {
	return fromStudentCSSTModelInternal(nil, m)
}

// Optional helper kalau mau dipakai di tempat lain (DB time)
func FromStudentCSSTModels(rows []model.StudentClassSectionSubjectTeacherModel) []StudentCSSTItem {
	out := make([]StudentCSSTItem, 0, len(rows))
	for i := range rows {
		out = append(out, FromStudentCSSTModel(&rows[i]))
	}
	return out
}

// Versi baru: mapper dengan konversi ke timezone sekolah (pakai dbtime + context)
func FromStudentCSSTModelWithSchoolTime(
	c *fiber.Ctx,
	m *model.StudentClassSectionSubjectTeacherModel,
) StudentCSSTItem {
	return fromStudentCSSTModelInternal(c, m)
}

func FromStudentCSSTModelsWithSchoolTime(
	c *fiber.Ctx,
	rows []model.StudentClassSectionSubjectTeacherModel,
) []StudentCSSTItem {
	out := make([]StudentCSSTItem, 0, len(rows))
	for i := range rows {
		out = append(out, FromStudentCSSTModelWithSchoolTime(c, &rows[i]))
	}
	return out
}

/* =========================================================
   INTERNAL CORE MAPPER (pakai dbtime)
========================================================= */

func fromStudentCSSTModelInternal(
	c *fiber.Ctx,
	m *model.StudentClassSectionSubjectTeacherModel,
) StudentCSSTItem {
	if m == nil {
		return StudentCSSTItem{}
	}

	var deletedAt *time.Time
	if m.StudentCSSTDeletedAt.Valid {
		t := dbtime.ToSchoolTime(c, m.StudentCSSTDeletedAt.Time)
		deletedAt = &t
	}

	return StudentCSSTItem{
		StudentCSSTID:       m.StudentCSSTID,
		StudentCSSTSchoolID: m.StudentCSSTSchoolID,

		StudentCSSTStudentID: m.StudentCSSTStudentID,
		StudentCSSTCSSTID:    m.StudentCSSTCSSTID,

		StudentCSSTIsActive: m.StudentCSSTIsActive,
		StudentCSSTFrom:     dbtime.ToSchoolTimePtr(c, m.StudentCSSTFrom),
		StudentCSSTTo:       dbtime.ToSchoolTimePtr(c, m.StudentCSSTTo),

		StudentCSSTScoreTotal:    m.StudentCSSTScoreTotal,
		StudentCSSTScoreMaxTotal: m.StudentCSSTScoreMaxTotal,
		StudentCSSTScorePercent:  m.StudentCSSTScorePercent,
		StudentCSSTGradeLetter:   m.StudentCSSTGradeLetter,
		StudentCSSTGradePoint:    m.StudentCSSTGradePoint,
		StudentCSSTIsPassed:      m.StudentCSSTIsPassed,

		StudentCSSTUserProfileNameCache:         m.StudentCSSTUserProfileNameCache,
		StudentCSSTUserProfileAvatarURLCache:    m.StudentCSSTUserProfileAvatarURLCache,
		StudentCSSTUserProfileWhatsappURLCache:  m.StudentCSSTUserProfileWhatsappURLCache,
		StudentCSSTUserProfileParentNameCache:   m.StudentCSSTUserProfileParentNameCache,
		StudentCSSTUserProfileParentWAURLCache:  m.StudentCSSTUserProfileParentWAURLCache,
		StudentCSSTUserProfileGenderCache:       m.StudentCSSTUserProfileGenderCache,
		StudentCSSTSchoolStudentCodeCache:       m.StudentCSSTSchoolStudentCodeCache,
		StudentCSSTEditsHistory:                 m.StudentCSSTEditsHistory,
		StudentCSSTStudentNotes:                 m.StudentCSSTStudentNotes,
		StudentCSSTStudentNotesUpdatedAt:        dbtime.ToSchoolTimePtr(c, m.StudentCSSTStudentNotesUpdatedAt),
		StudentCSSTHomeroomNotes:                m.StudentCSSTHomeroomNotes,
		StudentCSSTHomeroomNotesUpdatedAt:       dbtime.ToSchoolTimePtr(c, m.StudentCSSTHomeroomNotesUpdatedAt),
		StudentCSSTSubjectTeacherNotes:          m.StudentCSSTSubjectTeacherNotes,
		StudentCSSTSubjectTeacherNotesUpdatedAt: dbtime.ToSchoolTimePtr(c, m.StudentCSSTSubjectTeacherNotesUpdatedAt),
		StudentCSSTSlug:                         m.StudentCSSTSlug,
		StudentCSSTMeta:                         m.StudentCSSTMeta,
		StudentCSSTCreatedAt:                    dbtime.ToSchoolTime(c, m.StudentCSSTCreatedAt),
		StudentCSSTUpdatedAt:                    dbtime.ToSchoolTime(c, m.StudentCSSTUpdatedAt),
		StudentCSSTDeletedAt:                    deletedAt,

		Student:      nil,
		Section:      nil,
		ClassSubject: nil,
		Teacher:      nil,
	}
}

/* =========================================================
   COMPACT ITEM — ringan untuk list / nested include
========================================================= */

type StudentCSSTCompactItem struct {
	StudentCSSTID uuid.UUID `json:"student_csst_id"`

	StudentCSSTStudentID uuid.UUID `json:"student_csst_student_id"`
	StudentCSSTCSSTID    uuid.UUID `json:"student_csst_csst_id"`

	StudentCSSTIsActive bool       `json:"student_csst_is_active"`
	StudentCSSTFrom     *time.Time `json:"student_csst_from,omitempty"`
	StudentCSSTTo       *time.Time `json:"student_csst_to,omitempty"`

	// cache yang sering kepake buat UI list
	StudentCSSTUserProfileNameCache      *string `json:"student_csst_user_profile_name_cache,omitempty"`
	StudentCSSTUserProfileAvatarURLCache *string `json:"student_csst_user_profile_avatar_url_cache,omitempty"`
	StudentCSSTSchoolStudentCodeCache    *string `json:"student_csst_school_student_code_cache,omitempty"`
	StudentCSSTUserProfileGenderCache    *string `json:"student_csst_user_profile_gender_cache,omitempty"`

	StudentCSSTSlug *string `json:"student_csst_slug,omitempty"`

	StudentCSSTCreatedAt time.Time  `json:"student_csst_created_at"`
	StudentCSSTUpdatedAt time.Time  `json:"student_csst_updated_at"`
	StudentCSSTDeletedAt *time.Time `json:"student_csst_deleted_at,omitempty"`
}

/* =========================================================
   MAPPERS — COMPACT (DB time vs School time)
========================================================= */

func FromStudentCSSTModelCompact(m *model.StudentClassSectionSubjectTeacherModel) StudentCSSTCompactItem {
	return fromStudentCSSTModelCompactInternal(nil, m)
}

func FromStudentCSSTModelsCompact(rows []model.StudentClassSectionSubjectTeacherModel) []StudentCSSTCompactItem {
	out := make([]StudentCSSTCompactItem, 0, len(rows))
	for i := range rows {
		out = append(out, FromStudentCSSTModelCompact(&rows[i]))
	}
	return out
}

func FromStudentCSSTModelCompactWithSchoolTime(
	c *fiber.Ctx,
	m *model.StudentClassSectionSubjectTeacherModel,
) StudentCSSTCompactItem {
	return fromStudentCSSTModelCompactInternal(c, m)
}

func FromStudentCSSTModelsCompactWithSchoolTime(
	c *fiber.Ctx,
	rows []model.StudentClassSectionSubjectTeacherModel,
) []StudentCSSTCompactItem {
	out := make([]StudentCSSTCompactItem, 0, len(rows))
	for i := range rows {
		out = append(out, FromStudentCSSTModelCompactWithSchoolTime(c, &rows[i]))
	}
	return out
}

/* =========================================================
   INTERNAL CORE
========================================================= */

func fromStudentCSSTModelCompactInternal(
	c *fiber.Ctx,
	m *model.StudentClassSectionSubjectTeacherModel,
) StudentCSSTCompactItem {
	if m == nil {
		return StudentCSSTCompactItem{}
	}

	var deletedAt *time.Time
	if m.StudentCSSTDeletedAt.Valid {
		t := dbtime.ToSchoolTime(c, m.StudentCSSTDeletedAt.Time)
		deletedAt = &t
	}

	return StudentCSSTCompactItem{
		StudentCSSTID: m.StudentCSSTID,

		StudentCSSTStudentID: m.StudentCSSTStudentID,
		StudentCSSTCSSTID:    m.StudentCSSTCSSTID,

		StudentCSSTIsActive: m.StudentCSSTIsActive,
		StudentCSSTFrom:     dbtime.ToSchoolTimePtr(c, m.StudentCSSTFrom),
		StudentCSSTTo:       dbtime.ToSchoolTimePtr(c, m.StudentCSSTTo),

		StudentCSSTUserProfileNameCache:      m.StudentCSSTUserProfileNameCache,
		StudentCSSTUserProfileAvatarURLCache: m.StudentCSSTUserProfileAvatarURLCache,
		StudentCSSTSchoolStudentCodeCache:    m.StudentCSSTSchoolStudentCodeCache,
		StudentCSSTUserProfileGenderCache:    m.StudentCSSTUserProfileGenderCache,

		StudentCSSTSlug: m.StudentCSSTSlug,

		StudentCSSTCreatedAt: dbtime.ToSchoolTime(c, m.StudentCSSTCreatedAt),
		StudentCSSTUpdatedAt: dbtime.ToSchoolTime(c, m.StudentCSSTUpdatedAt),
		StudentCSSTDeletedAt: deletedAt,
	}
}
