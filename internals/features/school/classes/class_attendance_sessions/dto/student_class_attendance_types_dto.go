// file: internals/features/attendance/dto/student_class_session_attendance_type_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	model "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
)

/* =========================================================
   Generic PatchField untuk PATCH partial
   ========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"set"`   // true → field ini akan diubah
	Value T    `json:"value"` // nilai baru (boleh nol / nil tergantung T)
}

/* =========================================================
   CREATE DTO
   ========================================================= */

type StudentClassSessionAttendanceTypeCreateDTO struct {
	// tenant
	StudentClassSessionAttendanceTypeSchoolID uuid.UUID `json:"student_class_session_attendance_type_school_id" validate:"required"`

	// data
	StudentClassSessionAttendanceTypeCode     string  `json:"student_class_session_attendance_type_code" validate:"required,max=32"`
	StudentClassSessionAttendanceTypeLabel    *string `json:"student_class_session_attendance_type_label" validate:"omitempty,max=80"`
	StudentClassSessionAttendanceTypeSlug     *string `json:"student_class_session_attendance_type_slug"  validate:"omitempty,max=120"`
	StudentClassSessionAttendanceTypeColor    *string `json:"student_class_session_attendance_type_color" validate:"omitempty,max=20"`
	StudentClassSessionAttendanceTypeDesc     *string `json:"student_class_session_attendance_type_desc"  validate:"omitempty"`
	StudentClassSessionAttendanceTypeIsActive *bool   `json:"student_class_session_attendance_type_is_active" validate:"omitempty"`
}

// ToModel: konversi DTO → Model (persiapan Create)
func (in *StudentClassSessionAttendanceTypeCreateDTO) ToModel() *model.StudentClassSessionAttendanceTypeModel {
	now := time.Now()
	m := &model.StudentClassSessionAttendanceTypeModel{
		StudentClassSessionAttendanceTypeSchoolID: in.StudentClassSessionAttendanceTypeSchoolID,
		StudentClassSessionAttendanceTypeCode:     strings.TrimSpace(in.StudentClassSessionAttendanceTypeCode),
		StudentClassSessionAttendanceTypeLabel:    trimPtr(in.StudentClassSessionAttendanceTypeLabel),
		StudentClassSessionAttendanceTypeSlug:     trimPtr(in.StudentClassSessionAttendanceTypeSlug),
		StudentClassSessionAttendanceTypeColor:    trimPtr(in.StudentClassSessionAttendanceTypeColor),
		StudentClassSessionAttendanceTypeDesc:     trimPtr(in.StudentClassSessionAttendanceTypeDesc),
		StudentClassSessionAttendanceTypeIsActive: true, // default
		// biarkan DB auto sekarang juga set, tapi set lokal tak masalah
		StudentClassSessionAttendanceTypeCreatedAt: now,
		StudentClassSessionAttendanceTypeUpdatedAt: now,
	}
	if in.StudentClassSessionAttendanceTypeIsActive != nil {
		m.StudentClassSessionAttendanceTypeIsActive = *in.StudentClassSessionAttendanceTypeIsActive
	}
	return m
}

/* =========================================================
   PATCH DTO (partial update)
   ========================================================= */

type StudentClassSessionAttendanceTypePatchDTO struct {
	// kunci
	StudentClassSessionAttendanceTypeID       uuid.UUID `json:"student_class_session_attendance_type_id" validate:"required"`
	StudentClassSessionAttendanceTypeSchoolID uuid.UUID `json:"student_class_session_attendance_type_school_id" validate:"required"`

	// field yang bisa diubah (opsional via PatchField)
	StudentClassSessionAttendanceTypeCode     PatchField[string]  `json:"student_class_session_attendance_type_code"`
	StudentClassSessionAttendanceTypeLabel    PatchField[*string] `json:"student_class_session_attendance_type_label"`
	StudentClassSessionAttendanceTypeSlug     PatchField[*string] `json:"student_class_session_attendance_type_slug"`
	StudentClassSessionAttendanceTypeColor    PatchField[*string] `json:"student_class_session_attendance_type_color"`
	StudentClassSessionAttendanceTypeDesc     PatchField[*string] `json:"student_class_session_attendance_type_desc"`
	StudentClassSessionAttendanceTypeIsActive PatchField[bool]    `json:"student_class_session_attendance_type_is_active"`
}

// ApplyPatch: terapkan perubahan ke model existing (in-place)
func (p *StudentClassSessionAttendanceTypePatchDTO) ApplyPatch(m *model.StudentClassSessionAttendanceTypeModel) {
	if p.StudentClassSessionAttendanceTypeCode.Set {
		m.StudentClassSessionAttendanceTypeCode = strings.TrimSpace(p.StudentClassSessionAttendanceTypeCode.Value)
	}
	if p.StudentClassSessionAttendanceTypeLabel.Set {
		m.StudentClassSessionAttendanceTypeLabel = trimPtr(p.StudentClassSessionAttendanceTypeLabel.Value)
	}
	if p.StudentClassSessionAttendanceTypeSlug.Set {
		m.StudentClassSessionAttendanceTypeSlug = trimPtr(p.StudentClassSessionAttendanceTypeSlug.Value)
	}
	if p.StudentClassSessionAttendanceTypeColor.Set {
		m.StudentClassSessionAttendanceTypeColor = trimPtr(p.StudentClassSessionAttendanceTypeColor.Value)
	}
	if p.StudentClassSessionAttendanceTypeDesc.Set {
		m.StudentClassSessionAttendanceTypeDesc = trimPtr(p.StudentClassSessionAttendanceTypeDesc.Value)
	}
	if p.StudentClassSessionAttendanceTypeIsActive.Set {
		m.StudentClassSessionAttendanceTypeIsActive = p.StudentClassSessionAttendanceTypeIsActive.Value
	}
	m.StudentClassSessionAttendanceTypeUpdatedAt = time.Now() // biarkan DB juga update
}

/* =========================================================
   LIST / FILTER DTO + Query Builder
   ========================================================= */

type StudentClassSessionAttendanceTypeListQuery struct {
	StudentClassSessionAttendanceTypeSchoolID uuid.UUID `json:"student_class_session_attendance_type_school_id" validate:"required"`

	// filter opsional
	CodeEq        *string `json:"code_eq"        validate:"omitempty,max=32"`  // match code (case-insensitive)
	LabelQueryILK *string `json:"label_query"    validate:"omitempty,max=120"` // ILIKE %q% (dibantu trigram)
	SlugEq        *string `json:"slug_eq"        validate:"omitempty,max=120"` // match slug (case-insensitive)
	OnlyActive    *bool   `json:"only_active"`                                 // true=aktif; false=non-aktif; nil=semua (kec. soft-deleted)

	// paging & urutan
	Limit  int    `json:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int    `json:"offset" validate:"omitempty,min=0"`
	Sort   string `json:"sort"   validate:"omitempty,oneof=created_at_desc created_at_asc code_asc code_desc label_asc label_desc"`
}

// BuildQuery: terapkan filter ke *gorm.DB (tidak memanggil Find/Count)
func (q *StudentClassSessionAttendanceTypeListQuery) BuildQuery(db *gorm.DB) *gorm.DB {
	g := db.Model(&model.StudentClassSessionAttendanceTypeModel{}).
		Where("student_class_session_attendance_type_school_id = ?", q.StudentClassSessionAttendanceTypeSchoolID).
		Where("student_class_session_attendance_type_deleted_at IS NULL") // soft-delete aware default

	if q.CodeEq != nil && strings.TrimSpace(*q.CodeEq) != "" {
		g = g.Where("UPPER(student_class_session_attendance_type_code) = UPPER(?)", strings.TrimSpace(*q.CodeEq))
	}
	if q.LabelQueryILK != nil && strings.TrimSpace(*q.LabelQueryILK) != "" {
		like := "%" + strings.TrimSpace(*q.LabelQueryILK) + "%"
		g = g.Where("student_class_session_attendance_type_label ILIKE ?", like)
	}
	if q.SlugEq != nil && strings.TrimSpace(*q.SlugEq) != "" {
		g = g.Where("LOWER(student_class_session_attendance_type_slug) = LOWER(?)", strings.TrimSpace(*q.SlugEq))
	}
	if q.OnlyActive != nil {
		g = g.Where("student_class_session_attendance_type_is_active = ?", *q.OnlyActive)
	}

	switch q.Sort {
	case "created_at_asc":
		g = g.Order("student_class_session_attendance_type_created_at ASC")
	case "code_asc":
		g = g.Order("student_class_session_attendance_type_code ASC")
	case "code_desc":
		g = g.Order("student_class_session_attendance_type_code DESC")
	case "label_asc":
		g = g.Order("student_class_session_attendance_type_label ASC NULLS LAST")
	case "label_desc":
		g = g.Order("student_class_session_attendance_type_label DESC NULLS LAST")
	default:
		g = g.Order("student_class_session_attendance_type_created_at DESC")
	}

	lim := q.Limit
	if lim <= 0 {
		lim = 20
	}
	if lim > 200 {
		lim = 200
	}
	off := q.Offset
	if off < 0 {
		off = 0
	}
	return g.Limit(lim).Offset(off)
}

/* =========================================================
   RESPONSE DTO (item & paging)
   ========================================================= */

type StudentClassSessionAttendanceTypeItem struct {
	StudentClassSessionAttendanceTypeID        uuid.UUID `json:"student_class_session_attendance_type_id"`
	StudentClassSessionAttendanceTypeSchoolID  uuid.UUID `json:"student_class_session_attendance_type_school_id"`
	StudentClassSessionAttendanceTypeCode      string    `json:"student_class_session_attendance_type_code"`
	StudentClassSessionAttendanceTypeLabel     *string   `json:"student_class_session_attendance_type_label,omitempty"`
	StudentClassSessionAttendanceTypeSlug      *string   `json:"student_class_session_attendance_type_slug,omitempty"`
	StudentClassSessionAttendanceTypeColor     *string   `json:"student_class_session_attendance_type_color,omitempty"`
	StudentClassSessionAttendanceTypeDesc      *string   `json:"student_class_session_attendance_type_desc,omitempty"`
	StudentClassSessionAttendanceTypeIsActive  bool      `json:"student_class_session_attendance_type_is_active"`
	StudentClassSessionAttendanceTypeCreatedAt time.Time `json:"student_class_session_attendance_type_created_at"`
	StudentClassSessionAttendanceTypeUpdatedAt time.Time `json:"student_class_session_attendance_type_updated_at"`
}

func FromModel(m *model.StudentClassSessionAttendanceTypeModel) StudentClassSessionAttendanceTypeItem {
	return StudentClassSessionAttendanceTypeItem{
		StudentClassSessionAttendanceTypeID:        m.StudentClassSessionAttendanceTypeID,
		StudentClassSessionAttendanceTypeSchoolID:  m.StudentClassSessionAttendanceTypeSchoolID,
		StudentClassSessionAttendanceTypeCode:      m.StudentClassSessionAttendanceTypeCode,
		StudentClassSessionAttendanceTypeLabel:     m.StudentClassSessionAttendanceTypeLabel,
		StudentClassSessionAttendanceTypeSlug:      m.StudentClassSessionAttendanceTypeSlug,
		StudentClassSessionAttendanceTypeColor:     m.StudentClassSessionAttendanceTypeColor,
		StudentClassSessionAttendanceTypeDesc:      m.StudentClassSessionAttendanceTypeDesc,
		StudentClassSessionAttendanceTypeIsActive:  m.StudentClassSessionAttendanceTypeIsActive,
		StudentClassSessionAttendanceTypeCreatedAt: m.StudentClassSessionAttendanceTypeCreatedAt,
		StudentClassSessionAttendanceTypeUpdatedAt: m.StudentClassSessionAttendanceTypeUpdatedAt,
	}
}

type Page[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	NextOffset int   `json:"next_offset"`
	HasMore    bool  `json:"has_more"`
}

func NewPage[T any](items []T, total int64, limit, offset int) Page[T] {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	next := offset + limit
	return Page[T]{
		Items:      items,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		NextOffset: next,
		HasMore:    int64(next) < total,
	}
}

/* =========================================================
   Validator helper (opsional di controller)
   ========================================================= */

func ValidateStruct(v *validator.Validate, s any) error {
	if v == nil {
		return nil
	}
	return v.Struct(s)
}

/* =========================================================
   Utils lokal kecil
   ========================================================= */

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
