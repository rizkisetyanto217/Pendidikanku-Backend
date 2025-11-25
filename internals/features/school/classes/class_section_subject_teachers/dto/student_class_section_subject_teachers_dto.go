// file: internals/features/school/sectionsubjectteachers/dto/student_class_section_subject_teacher_dto.go
package dto

import (
	"time"

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
   MAIN ITEM — FULL MIRROR OF MODEL
========================================================= */

type StudentCSSTItem struct {
	StudentClassSectionSubjectTeacherID uuid.UUID `json:"student_class_section_subject_teacher_id"`

	StudentClassSectionSubjectTeacherSchoolID uuid.UUID `json:"student_class_section_subject_teacher_school_id"`

	StudentClassSectionSubjectTeacherStudentID uuid.UUID `json:"student_class_section_subject_teacher_student_id"`
	StudentClassSectionSubjectTeacherCSSTID    uuid.UUID `json:"student_class_section_subject_teacher_csst_id"`

	StudentClassSectionSubjectTeacherIsActive bool       `json:"student_class_section_subject_teacher_is_active"`
	StudentClassSectionSubjectTeacherFrom     *time.Time `json:"student_class_section_subject_teacher_from,omitempty"`
	StudentClassSectionSubjectTeacherTo       *time.Time `json:"student_class_section_subject_teacher_to,omitempty"`

	StudentClassSectionSubjectTeacherScoreTotal    *float64 `json:"student_class_section_subject_teacher_score_total,omitempty"`
	StudentClassSectionSubjectTeacherScoreMaxTotal *float64 `json:"student_class_section_subject_teacher_score_max_total,omitempty"`
	StudentClassSectionSubjectTeacherScorePercent  *float64 `json:"student_class_section_subject_teacher_score_percent,omitempty"`
	StudentClassSectionSubjectTeacherGradeLetter   *string  `json:"student_class_section_subject_teacher_grade_letter,omitempty"`
	StudentClassSectionSubjectTeacherGradePoint    *float64 `json:"student_class_section_subject_teacher_grade_point,omitempty"`
	StudentClassSectionSubjectTeacherIsPassed      *bool    `json:"student_class_section_subject_teacher_is_passed,omitempty"`

	StudentClassSectionSubjectTeacherUserProfileNameSnapshot              *string `json:"student_class_section_subject_teacher_user_profile_name_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileAvatarURLSnapshot         *string `json:"student_class_section_subject_teacher_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileWhatsappURLSnapshot       *string `json:"student_class_section_subject_teacher_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileParentNameSnapshot        *string `json:"student_class_section_subject_teacher_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileParentWhatsappURLSnapshot *string `json:"student_class_section_subject_teacher_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileGenderSnapshot            *string `json:"student_class_section_subject_teacher_user_profile_gender_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherStudentCodeSnapshot                  *string `json:"student_class_section_subject_teacher_student_code_snapshot,omitempty"`

	StudentClassSectionSubjectTeacherEditsHistory datatypes.JSON `json:"student_class_section_subject_teacher_edits_history"`

	// NOTES (mirror model)
	StudentClassSectionSubjectTeacherStudentNotes                 *string    `json:"student_class_section_subject_teacher_student_notes,omitempty"`
	StudentClassSectionSubjectTeacherStudentNotesUpdatedAt        *time.Time `json:"student_class_section_subject_teacher_student_notes_updated_at,omitempty"`
	StudentClassSectionSubjectTeacherHomeroomNotes                *string    `json:"student_class_section_subject_teacher_homeroom_notes,omitempty"`
	StudentClassSectionSubjectTeacherHomeroomNotesUpdatedAt       *time.Time `json:"student_class_section_subject_teacher_homeroom_notes_updated_at,omitempty"`
	StudentClassSectionSubjectTeacherSubjectTeacherNotes          *string    `json:"student_class_section_subject_teacher_subject_teacher_notes,omitempty"`
	StudentClassSectionSubjectTeacherSubjectTeacherNotesUpdatedAt *time.Time `json:"student_class_section_subject_teacher_subject_teacher_notes_updated_at,omitempty"`

	StudentClassSectionSubjectTeacherSlug *string        `json:"student_class_section_subject_teacher_slug,omitempty"`
	StudentClassSectionSubjectTeacherMeta datatypes.JSON `json:"student_class_section_subject_teacher_meta"`

	StudentClassSectionSubjectTeacherCreatedAt time.Time  `json:"student_class_section_subject_teacher_created_at"`
	StudentClassSectionSubjectTeacherUpdatedAt time.Time  `json:"student_class_section_subject_teacher_updated_at"`
	StudentClassSectionSubjectTeacherDeletedAt *time.Time `json:"student_class_section_subject_teacher_deleted_at,omitempty"`

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
