// file: internals/features/attendance/dto/user_attendance_type_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	model "masjidku_backend/internals/features/school/sessions/sessions/model"
)

/* =========================================================
   Generic PatchField untuk PATCH partial
   ========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"set"`   // jika true, artinya "field ini mau diubah"
	Value T    `json:"value"` // nilai baru (boleh nol / nil tergantung T)
}

/* =========================================================
   CREATE DTO
   ========================================================= */

type UserAttendanceTypeCreateDTO struct {
	// tenant
	UserAttendanceTypeMasjidID uuid.UUID `json:"user_attendance_type_masjid_id" validate:"required"`

	// data
	UserAttendanceTypeCode  string  `json:"user_attendance_type_code" validate:"required,max=32"`
	UserAttendanceTypeLabel *string `json:"user_attendance_type_label" validate:"omitempty,max=80"`
	UserAttendanceTypeDesc  *string `json:"user_attendance_type_desc" validate:"omitempty"`
	UserAttendanceTypeIsActive *bool `json:"user_attendance_type_is_active" validate:"omitempty"`
}

// ToModel: konversi DTO → Model (persiapan Create)
func (in *UserAttendanceTypeCreateDTO) ToModel() *model.UserAttendanceTypeModel {
	m := &model.UserAttendanceTypeModel{
		UserAttendanceTypeMasjidID:  in.UserAttendanceTypeMasjidID,
		UserAttendanceTypeCode:      strings.TrimSpace(in.UserAttendanceTypeCode),
		UserAttendanceTypeLabel:     trimPtr(in.UserAttendanceTypeLabel),
		UserAttendanceTypeDesc:      trimPtr(in.UserAttendanceTypeDesc),
		UserAttendanceTypeIsActive:  true, // default
		UserAttendanceTypeCreatedAt: time.Now(),
		UserAttendanceTypeUpdatedAt: time.Now(),
	}
	if in.UserAttendanceTypeIsActive != nil {
		m.UserAttendanceTypeIsActive = *in.UserAttendanceTypeIsActive
	}
	return m
}

/* =========================================================
   PATCH DTO (partial update)
   ========================================================= */

type UserAttendanceTypePatchDTO struct {
	// kunci
	UserAttendanceTypeID       uuid.UUID `json:"user_attendance_type_id" validate:"required"`
	UserAttendanceTypeMasjidID uuid.UUID `json:"user_attendance_type_masjid_id" validate:"required"`

	// field yang bisa diubah (opsional via PatchField)
	UserAttendanceTypeCode     PatchField[string]  `json:"user_attendance_type_code"`
	UserAttendanceTypeLabel    PatchField[*string] `json:"user_attendance_type_label"`
	UserAttendanceTypeDesc     PatchField[*string] `json:"user_attendance_type_desc"`
	UserAttendanceTypeIsActive PatchField[bool]    `json:"user_attendance_type_is_active"`
}

// ApplyPatch: terapkan perubahan ke model existing (in-place)
func (p *UserAttendanceTypePatchDTO) ApplyPatch(m *model.UserAttendanceTypeModel) {
	if p.UserAttendanceTypeCode.Set {
		m.UserAttendanceTypeCode = strings.TrimSpace(p.UserAttendanceTypeCode.Value)
	}
	if p.UserAttendanceTypeLabel.Set {
		m.UserAttendanceTypeLabel = trimPtr(p.UserAttendanceTypeLabel.Value)
	}
	if p.UserAttendanceTypeDesc.Set {
		m.UserAttendanceTypeDesc = trimPtr(p.UserAttendanceTypeDesc.Value)
	}
	if p.UserAttendanceTypeIsActive.Set {
		m.UserAttendanceTypeIsActive = p.UserAttendanceTypeIsActive.Value
	}
	m.UserAttendanceTypeUpdatedAt = time.Now()
}

/* =========================================================
   LIST / FILTER DTO + Query Builder
   ========================================================= */

type UserAttendanceTypeListQuery struct {
	UserAttendanceTypeMasjidID uuid.UUID `json:"user_attendance_type_masjid_id" validate:"required"`

	// filter opsional
	CodeEq        *string `json:"code_eq" validate:"omitempty,max=32"`     // match code (case-insensitive)
	LabelQueryILK *string `json:"label_query" validate:"omitempty,max=120"` // ILIKE %q% (dibantu trigram)
	OnlyActive    *bool   `json:"only_active"`                              // true=aktif saja; false=tidak aktif saja; nil=semua (kecuali soft-deleted)

	// paging & urutan
	Limit  int    `json:"limit" validate:"omitempty,min=1,max=200"`
	Offset int    `json:"offset" validate:"omitempty,min=0"`
	Sort   string `json:"sort" validate:"omitempty,oneof=created_at_desc created_at_asc code_asc code_desc label_asc label_desc"`
}

// BuildQuery: terapkan filter ke *gorm.DB (ingat: tidak memanggil Find/Count)
func (q *UserAttendanceTypeListQuery) BuildQuery(db *gorm.DB) *gorm.DB {
	g := db.Model(&model.UserAttendanceTypeModel{}).
		Where("user_attendance_type_masjid_id = ?", q.UserAttendanceTypeMasjidID).
		Where("user_attendance_type_deleted_at IS NULL") // soft-delete aware default

	if q.CodeEq != nil && strings.TrimSpace(*q.CodeEq) != "" {
		g = g.Where("UPPER(user_attendance_type_code) = UPPER(?)", strings.TrimSpace(*q.CodeEq))
	}
	if q.LabelQueryILK != nil && strings.TrimSpace(*q.LabelQueryILK) != "" {
		like := "%" + strings.TrimSpace(*q.LabelQueryILK) + "%"
		g = g.Where("user_attendance_type_label ILIKE ?", like)
	}
	if q.OnlyActive != nil {
		g = g.Where("user_attendance_type_is_active = ?", *q.OnlyActive)
	}

	switch q.Sort {
	case "created_at_asc":
		g = g.Order("user_attendance_type_created_at ASC")
	case "code_asc":
		g = g.Order("user_attendance_type_code ASC")
	case "code_desc":
		g = g.Order("user_attendance_type_code DESC")
	case "label_asc":
		g = g.Order("user_attendance_type_label ASC NULLS LAST")
	case "label_desc":
		g = g.Order("user_attendance_type_label DESC NULLS LAST")
	default:
		g = g.Order("user_attendance_type_created_at DESC")
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

type UserAttendanceTypeItem struct {
	UserAttendanceTypeID        uuid.UUID  `json:"user_attendance_type_id"`
	UserAttendanceTypeMasjidID  uuid.UUID  `json:"user_attendance_type_masjid_id"`
	UserAttendanceTypeCode      string     `json:"user_attendance_type_code"`
	UserAttendanceTypeLabel     *string    `json:"user_attendance_type_label,omitempty"`
	UserAttendanceTypeDesc      *string    `json:"user_attendance_type_desc,omitempty"`
	UserAttendanceTypeIsActive  bool       `json:"user_attendance_type_is_active"`
	UserAttendanceTypeCreatedAt time.Time  `json:"user_attendance_type_created_at"`
	UserAttendanceTypeUpdatedAt time.Time  `json:"user_attendance_type_updated_at"`
}

func FromModel(m *model.UserAttendanceTypeModel) UserAttendanceTypeItem {
	return UserAttendanceTypeItem{
		UserAttendanceTypeID:        m.UserAttendanceTypeID,
		UserAttendanceTypeMasjidID:  m.UserAttendanceTypeMasjidID,
		UserAttendanceTypeCode:      m.UserAttendanceTypeCode,
		UserAttendanceTypeLabel:     m.UserAttendanceTypeLabel,
		UserAttendanceTypeDesc:      m.UserAttendanceTypeDesc,
		UserAttendanceTypeIsActive:  m.UserAttendanceTypeIsActive,
		UserAttendanceTypeCreatedAt: m.UserAttendanceTypeCreatedAt,
		UserAttendanceTypeUpdatedAt: m.UserAttendanceTypeUpdatedAt,
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
