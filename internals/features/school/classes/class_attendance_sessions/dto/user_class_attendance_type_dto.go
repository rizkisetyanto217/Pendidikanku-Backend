// file: internals/features/attendance/dto/user_class_session_attendance_type_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	model "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"
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

type UserClassSessionAttendanceTypeCreateDTO struct {
	// tenant
	UserClassSessionAttendanceTypeMasjidID uuid.UUID `json:"user_class_session_attendance_type_masjid_id" validate:"required"`

	// data
	UserClassSessionAttendanceTypeCode     string  `json:"user_class_session_attendance_type_code" validate:"required,max=32"`
	UserClassSessionAttendanceTypeLabel    *string `json:"user_class_session_attendance_type_label" validate:"omitempty,max=80"`
	UserClassSessionAttendanceTypeSlug     *string `json:"user_class_session_attendance_type_slug"  validate:"omitempty,max=120"`
	UserClassSessionAttendanceTypeColor    *string `json:"user_class_session_attendance_type_color" validate:"omitempty,max=20"`
	UserClassSessionAttendanceTypeDesc     *string `json:"user_class_session_attendance_type_desc"  validate:"omitempty"`
	UserClassSessionAttendanceTypeIsActive *bool   `json:"user_class_session_attendance_type_is_active" validate:"omitempty"`
}

// ToModel: konversi DTO → Model (persiapan Create)
func (in *UserClassSessionAttendanceTypeCreateDTO) ToModel() *model.UserClassSessionAttendanceTypeModel {
	now := time.Now()
	m := &model.UserClassSessionAttendanceTypeModel{
		UserClassSessionAttendanceTypeMasjidID: in.UserClassSessionAttendanceTypeMasjidID,
		UserClassSessionAttendanceTypeCode:     strings.TrimSpace(in.UserClassSessionAttendanceTypeCode),
		UserClassSessionAttendanceTypeLabel:    trimPtr(in.UserClassSessionAttendanceTypeLabel),
		UserClassSessionAttendanceTypeSlug:     trimPtr(in.UserClassSessionAttendanceTypeSlug),
		UserClassSessionAttendanceTypeColor:    trimPtr(in.UserClassSessionAttendanceTypeColor),
		UserClassSessionAttendanceTypeDesc:     trimPtr(in.UserClassSessionAttendanceTypeDesc),
		UserClassSessionAttendanceTypeIsActive: true, // default
		// biarkan DB auto sekarang juga set, tapi set lokal tak masalah
		UserClassSessionAttendanceTypeCreatedAt: now,
		UserClassSessionAttendanceTypeUpdatedAt: now,
	}
	if in.UserClassSessionAttendanceTypeIsActive != nil {
		m.UserClassSessionAttendanceTypeIsActive = *in.UserClassSessionAttendanceTypeIsActive
	}
	return m
}

/* =========================================================
   PATCH DTO (partial update)
   ========================================================= */

type UserClassSessionAttendanceTypePatchDTO struct {
	// kunci
	UserClassSessionAttendanceTypeID       uuid.UUID `json:"user_class_session_attendance_type_id" validate:"required"`
	UserClassSessionAttendanceTypeMasjidID uuid.UUID `json:"user_class_session_attendance_type_masjid_id" validate:"required"`

	// field yang bisa diubah (opsional via PatchField)
	UserClassSessionAttendanceTypeCode     PatchField[string]  `json:"user_class_session_attendance_type_code"`
	UserClassSessionAttendanceTypeLabel    PatchField[*string] `json:"user_class_session_attendance_type_label"`
	UserClassSessionAttendanceTypeSlug     PatchField[*string] `json:"user_class_session_attendance_type_slug"`
	UserClassSessionAttendanceTypeColor    PatchField[*string] `json:"user_class_session_attendance_type_color"`
	UserClassSessionAttendanceTypeDesc     PatchField[*string] `json:"user_class_session_attendance_type_desc"`
	UserClassSessionAttendanceTypeIsActive PatchField[bool]    `json:"user_class_session_attendance_type_is_active"`
}

// ApplyPatch: terapkan perubahan ke model existing (in-place)
func (p *UserClassSessionAttendanceTypePatchDTO) ApplyPatch(m *model.UserClassSessionAttendanceTypeModel) {
	if p.UserClassSessionAttendanceTypeCode.Set {
		m.UserClassSessionAttendanceTypeCode = strings.TrimSpace(p.UserClassSessionAttendanceTypeCode.Value)
	}
	if p.UserClassSessionAttendanceTypeLabel.Set {
		m.UserClassSessionAttendanceTypeLabel = trimPtr(p.UserClassSessionAttendanceTypeLabel.Value)
	}
	if p.UserClassSessionAttendanceTypeSlug.Set {
		m.UserClassSessionAttendanceTypeSlug = trimPtr(p.UserClassSessionAttendanceTypeSlug.Value)
	}
	if p.UserClassSessionAttendanceTypeColor.Set {
		m.UserClassSessionAttendanceTypeColor = trimPtr(p.UserClassSessionAttendanceTypeColor.Value)
	}
	if p.UserClassSessionAttendanceTypeDesc.Set {
		m.UserClassSessionAttendanceTypeDesc = trimPtr(p.UserClassSessionAttendanceTypeDesc.Value)
	}
	if p.UserClassSessionAttendanceTypeIsActive.Set {
		m.UserClassSessionAttendanceTypeIsActive = p.UserClassSessionAttendanceTypeIsActive.Value
	}
	m.UserClassSessionAttendanceTypeUpdatedAt = time.Now() // biarkan DB juga update
}

/* =========================================================
   LIST / FILTER DTO + Query Builder
   ========================================================= */

type UserClassSessionAttendanceTypeListQuery struct {
	UserClassSessionAttendanceTypeMasjidID uuid.UUID `json:"user_class_session_attendance_type_masjid_id" validate:"required"`

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
func (q *UserClassSessionAttendanceTypeListQuery) BuildQuery(db *gorm.DB) *gorm.DB {
	g := db.Model(&model.UserClassSessionAttendanceTypeModel{}).
		Where("user_class_session_attendance_type_masjid_id = ?", q.UserClassSessionAttendanceTypeMasjidID).
		Where("user_class_session_attendance_type_deleted_at IS NULL") // soft-delete aware default

	if q.CodeEq != nil && strings.TrimSpace(*q.CodeEq) != "" {
		g = g.Where("UPPER(user_class_session_attendance_type_code) = UPPER(?)", strings.TrimSpace(*q.CodeEq))
	}
	if q.LabelQueryILK != nil && strings.TrimSpace(*q.LabelQueryILK) != "" {
		like := "%" + strings.TrimSpace(*q.LabelQueryILK) + "%"
		g = g.Where("user_class_session_attendance_type_label ILIKE ?", like)
	}
	if q.SlugEq != nil && strings.TrimSpace(*q.SlugEq) != "" {
		g = g.Where("LOWER(user_class_session_attendance_type_slug) = LOWER(?)", strings.TrimSpace(*q.SlugEq))
	}
	if q.OnlyActive != nil {
		g = g.Where("user_class_session_attendance_type_is_active = ?", *q.OnlyActive)
	}

	switch q.Sort {
	case "created_at_asc":
		g = g.Order("user_class_session_attendance_type_created_at ASC")
	case "code_asc":
		g = g.Order("user_class_session_attendance_type_code ASC")
	case "code_desc":
		g = g.Order("user_class_session_attendance_type_code DESC")
	case "label_asc":
		g = g.Order("user_class_session_attendance_type_label ASC NULLS LAST")
	case "label_desc":
		g = g.Order("user_class_session_attendance_type_label DESC NULLS LAST")
	default:
		g = g.Order("user_class_session_attendance_type_created_at DESC")
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

type UserClassSessionAttendanceTypeItem struct {
	UserClassSessionAttendanceTypeID        uuid.UUID `json:"user_class_session_attendance_type_id"`
	UserClassSessionAttendanceTypeMasjidID  uuid.UUID `json:"user_class_session_attendance_type_masjid_id"`
	UserClassSessionAttendanceTypeCode      string    `json:"user_class_session_attendance_type_code"`
	UserClassSessionAttendanceTypeLabel     *string   `json:"user_class_session_attendance_type_label,omitempty"`
	UserClassSessionAttendanceTypeSlug      *string   `json:"user_class_session_attendance_type_slug,omitempty"`
	UserClassSessionAttendanceTypeColor     *string   `json:"user_class_session_attendance_type_color,omitempty"`
	UserClassSessionAttendanceTypeDesc      *string   `json:"user_class_session_attendance_type_desc,omitempty"`
	UserClassSessionAttendanceTypeIsActive  bool      `json:"user_class_session_attendance_type_is_active"`
	UserClassSessionAttendanceTypeCreatedAt time.Time `json:"user_class_session_attendance_type_created_at"`
	UserClassSessionAttendanceTypeUpdatedAt time.Time `json:"user_class_session_attendance_type_updated_at"`
}

func FromModel(m *model.UserClassSessionAttendanceTypeModel) UserClassSessionAttendanceTypeItem {
	return UserClassSessionAttendanceTypeItem{
		UserClassSessionAttendanceTypeID:        m.UserClassSessionAttendanceTypeID,
		UserClassSessionAttendanceTypeMasjidID:  m.UserClassSessionAttendanceTypeMasjidID,
		UserClassSessionAttendanceTypeCode:      m.UserClassSessionAttendanceTypeCode,
		UserClassSessionAttendanceTypeLabel:     m.UserClassSessionAttendanceTypeLabel,
		UserClassSessionAttendanceTypeSlug:      m.UserClassSessionAttendanceTypeSlug,
		UserClassSessionAttendanceTypeColor:     m.UserClassSessionAttendanceTypeColor,
		UserClassSessionAttendanceTypeDesc:      m.UserClassSessionAttendanceTypeDesc,
		UserClassSessionAttendanceTypeIsActive:  m.UserClassSessionAttendanceTypeIsActive,
		UserClassSessionAttendanceTypeCreatedAt: m.UserClassSessionAttendanceTypeCreatedAt,
		UserClassSessionAttendanceTypeUpdatedAt: m.UserClassSessionAttendanceTypeUpdatedAt,
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
