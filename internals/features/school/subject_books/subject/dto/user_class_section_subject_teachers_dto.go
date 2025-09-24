// file: internals/features/school/sectionsubjectteachers/dto/user_class_section_subject_teacher_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

/* =========================================================
   UTIL & COMMON
   ========================================================= */

// PatchFieldUserCSST mewakili tri-state untuk PATCH:
// - Nil:   field tidak diubah
// - Set==true dengan Value: set nilai
// - Set==true dengan IsNull==true: set NULL (untuk kolom nullable)
// Catatan: untuk kolom non-nullable, abaikan IsNull=true di layer service/validator.
type PatchFieldUserCSST[T any] struct {
	Set    bool `json:"-"`
	IsNull bool `json:"-"`
	Value  T    `json:"-"`
}

// Helper pembuat PatchFieldUserCSST
func PF[T any](v T) PatchFieldUserCSST[T]   { return PatchFieldUserCSST[T]{Set: true, Value: v} }
func PFNull[T any]() PatchFieldUserCSST[T]  { return PatchFieldUserCSST[T]{Set: true, IsNull: true} }
func PFUnset[T any]() PatchFieldUserCSST[T] { return PatchFieldUserCSST[T]{Set: false} }

// Include options untuk expand relasi di response list/detail
type IncludeOptions struct {
	Section      bool `json:"section,omitempty" query:"include_section"`
	ClassSubject bool `json:"class_subject,omitempty" query:"include_class_subject"`
	Teacher      bool `json:"teacher,omitempty" query:"include_teacher"`
}

/* =========================================================
   REQUEST: PATH / BODY HELPERS
   ========================================================= */

// Path params standar (/:id)
type IDParam struct {
	ID uuid.UUID `json:"id" params:"id" validate:"required"`
}

// Bulk IDs (delete/restore/toggle/batch patch)
type BulkIDsRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,dive,required"`
}

/* =========================================================
   CREATE / UPSERT
   ========================================================= */

// Create satu record
type UserCSSTCreateRequest struct {
	SectionID      uuid.UUID `json:"section_id" validate:"required"`
	ClassSubjectID uuid.UUID `json:"class_subject_id" validate:"required"`
	TeacherID      uuid.UUID `json:"teacher_id" validate:"required"`
	IsActive       *bool     `json:"is_active,omitempty"` // default: true
	// Optional client-provided idempotency key (untuk cegah duplikasi pada retry)
	IdempotencyKey *string `json:"idempotency_key,omitempty" validate:"omitempty,max=120"`
}

// Create bulk
type UserCSSTBulkCreateItem struct {
	SectionID      uuid.UUID `json:"section_id" validate:"required"`
	ClassSubjectID uuid.UUID `json:"class_subject_id" validate:"required"`
	TeacherID      uuid.UUID `json:"teacher_id" validate:"required"`
	IsActive       *bool     `json:"is_active,omitempty"`
	// Optional reference untuk mapping hasil (misal baris CSV)
	ClientRef *string `json:"client_ref,omitempty" validate:"omitempty,max=120"`
}
type UserCSSTBulkCreateRequest struct {
	Items          []UserCSSTBulkCreateItem `json:"items" validate:"required,min=1,dive"`
	IdempotencyKey *string                  `json:"idempotency_key,omitempty" validate:"omitempty,max=120"`
	SkipDuplicates bool                     `json:"skip_duplicates,omitempty"` // true: lewati duplikat (by unique key)
	ReturnExisting bool                     `json:"return_existing,omitempty"` // true: kalau duplikat, ikut kembalikan data existing
}

// Upsert berdasarkan natural-unique (masjid_id, section_id, class_subject_id, teacher_id)
type UserCSSTUpsertRequest struct {
	SectionID      uuid.UUID `json:"section_id" validate:"required"`
	ClassSubjectID uuid.UUID `json:"class_subject_id" validate:"required"`
	TeacherID      uuid.UUID `json:"teacher_id" validate:"required"`
	IsActive       *bool     `json:"is_active,omitempty"`
}

/* =========================================================
   UPDATE / PATCH
   ========================================================= */

// PATCH satu record (tri-state)
type UserCSSTPatchRequest struct {
	SectionID      PatchFieldUserCSST[uuid.UUID] `json:"section_id,omitempty"`
	ClassSubjectID PatchFieldUserCSST[uuid.UUID] `json:"class_subject_id,omitempty"`
	TeacherID      PatchFieldUserCSST[uuid.UUID] `json:"teacher_id,omitempty"`
	IsActive       PatchFieldUserCSST[bool]      `json:"is_active,omitempty"`
}

// PATCH bulk by IDs
type UserCSSTBulkPatchRequest struct {
	IDs   []uuid.UUID          `json:"ids" validate:"required,min=1,dive,required"`
	Patch UserCSSTPatchRequest `json:"patch" validate:"required"`
}

// Toggle is_active (single & bulk)
type UserCSSTToggleActiveRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}
type UserCSSTBulkToggleActiveRequest struct {
	IDs      []uuid.UUID `json:"ids" validate:"required,min=1,dive,required"`
	IsActive bool        `json:"is_active" validate:"required"`
}

/* =========================================================
   DELETE / RESTORE
   ========================================================= */

// Soft delete
type UserCSSTDeleteRequest struct {
	// Force: hard delete; default soft delete
	Force bool `json:"force,omitempty"`
}

// Restore soft-deleted
type UserCSSTRestoreRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,dive,required"`
}

/* =========================================================
   LIST / QUERY PARAMS
   ========================================================= */

// Sort fields yang diizinkan
const (
	UserCSSTSortCreatedAt = "created_at"
	UserCSSTSortUpdatedAt = "updated_at"
	UserCSSTSortTeacher   = "teacher_id"
	UserCSSTSortSection   = "section_id"
	UserCSSTSortSubject   = "class_subject_id"
)

// Query untuk /list (querystring)
type UserCSSTListQuery struct {
	// Paging
	Page     int `query:"page" validate:"omitempty,min=1"`              // default 1
	PageSize int `query:"page_size" validate:"omitempty,min=1,max=200"` // default 20/50 sesuai policy

	// Filter
	SectionID      *uuid.UUID `query:"section_id"`
	ClassSubjectID *uuid.UUID `query:"class_subject_id"`
	TeacherID      *uuid.UUID `query:"teacher_id"`
	IsActive       *bool      `query:"is_active"`
	IncludeDeleted bool       `query:"include_deleted"`

	// Pencarian ringan (opsional)
	Q *string `query:"q"` // bebas: bisa cari by nama guru (join), dsb — implementasi di repo

	// Sorting
	SortBy string `query:"sort_by"` // created_at|updated_at|teacher_id|section_id|class_subject_id
	Order  string `query:"order"`   // asc|desc

	// Include relasi
	IncludeSection      bool `query:"include_section"`
	IncludeClassSubject bool `query:"include_class_subject"`
	IncludeTeacher      bool `query:"include_teacher"`
}

/* =========================================================
   RESPONSE MODELS
   ========================================================= */

// Bentuk ringkas untuk relasi (optional expand)
type SectionBrief struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name,omitempty"`
	// tambahkan field lain bila perlu (grade/rombel, dsb)
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

// Item utama
type UserCSSTItem struct {
	ID          uuid.UUID  `json:"id"`
	MasjidID    uuid.UUID  `json:"masjid_id"`
	SectionID   uuid.UUID  `json:"section_id"`
	ClassSubjID uuid.UUID  `json:"class_subject_id"`
	TeacherID   uuid.UUID  `json:"teacher_id"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`

	// Expanded (optional)
	Section      *SectionBrief      `json:"section,omitempty"`
	ClassSubject *ClassSubjectBrief `json:"class_subject,omitempty"`
	Teacher      *TeacherBrief      `json:"teacher,omitempty"`
}

// Envelope meta paging
type PageMeta struct {
	Total       int64 `json:"total"`
	Page        int   `json:"page"`
	PageSize    int   `json:"page_size"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

// Response: detail
type UserCSSTDetailResponse struct {
	Data UserCSSTItem `json:"data"`
}

// Response: list
type UserCSSTListResponse struct {
	Data []UserCSSTItem `json:"data"`
	Meta PageMeta       `json:"meta"`
}

// Response: create/upsert
type UserCSSTCreateResponse struct {
	Data UserCSSTItem `json:"data"`
}

// Response: bulk create
type UserCSSTBulkCreateResult struct {
	Item UserCSSTItem `json:"item"`
	// optional client ref bila ada
	ClientRef *string `json:"client_ref,omitempty"`
	// Duplicate true bila item dilewati/diambil dari existing tergantung flags
	Duplicate bool `json:"duplicate,omitempty"`
}
type UserCSSTBulkCreateResponse struct {
	Results []UserCSSTBulkCreateResult `json:"results"`
	Meta    struct {
		Inserted int `json:"inserted"`
		Skipped  int `json:"skipped"`
		Existing int `json:"existing"`
	} `json:"meta"`
}

// Response: generic operation
type AffectedResponse struct {
	Affected int `json:"affected"`
}

/* =========================================================
   ERROR ENVELOPE (opsional, kalau kamu pakai pola ini)
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
