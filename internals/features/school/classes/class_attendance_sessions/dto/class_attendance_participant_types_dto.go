// file: internals/features/attendance/dto/class_attendance_session_participant_type_dto.go
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

type ClassAttendanceSessionParticipantTypeCreateDTO struct {
	// tenant
	ClassAttendanceSessionParticipantTypeSchoolID uuid.UUID `json:"class_attendance_session_participant_type_school_id" validate:"required"`

	// data
	ClassAttendanceSessionParticipantTypeCode     string  `json:"class_attendance_session_participant_type_code" validate:"required,max=32"`
	ClassAttendanceSessionParticipantTypeLabel    *string `json:"class_attendance_session_participant_type_label" validate:"omitempty,max=80"`
	ClassAttendanceSessionParticipantTypeSlug     *string `json:"class_attendance_session_participant_type_slug"  validate:"omitempty,max=120"`
	ClassAttendanceSessionParticipantTypeColor    *string `json:"class_attendance_session_participant_type_color" validate:"omitempty,max=20"`
	ClassAttendanceSessionParticipantTypeDesc     *string `json:"class_attendance_session_participant_type_desc"  validate:"omitempty"`
	ClassAttendanceSessionParticipantTypeIsActive *bool   `json:"class_attendance_session_participant_type_is_active" validate:"omitempty"`
}

// ToModel: konversi DTO → Model (persiapan Create)
func (in *ClassAttendanceSessionParticipantTypeCreateDTO) ToModel() *model.ClassAttendanceSessionParticipantTypeModel {
	now := time.Now()
	m := &model.ClassAttendanceSessionParticipantTypeModel{
		ClassAttendanceSessionParticipantTypeSchoolID: in.ClassAttendanceSessionParticipantTypeSchoolID,
		ClassAttendanceSessionParticipantTypeCode:     strings.TrimSpace(in.ClassAttendanceSessionParticipantTypeCode),
		ClassAttendanceSessionParticipantTypeLabel:    trimPtr(in.ClassAttendanceSessionParticipantTypeLabel),
		ClassAttendanceSessionParticipantTypeSlug:     trimPtr(in.ClassAttendanceSessionParticipantTypeSlug),
		ClassAttendanceSessionParticipantTypeColor:    trimPtr(in.ClassAttendanceSessionParticipantTypeColor),
		ClassAttendanceSessionParticipantTypeDesc:     trimPtr(in.ClassAttendanceSessionParticipantTypeDesc),
		ClassAttendanceSessionParticipantTypeIsActive: true, // default
		// DB juga sudah default NOW(), tapi set lokal nggak masalah
		ClassAttendanceSessionParticipantTypeCreatedAt: now,
		ClassAttendanceSessionParticipantTypeUpdatedAt: now,
	}
	if in.ClassAttendanceSessionParticipantTypeIsActive != nil {
		m.ClassAttendanceSessionParticipantTypeIsActive = *in.ClassAttendanceSessionParticipantTypeIsActive
	}
	return m
}

/* =========================================================
   PATCH DTO (partial update)
   ========================================================= */

type ClassAttendanceSessionParticipantTypePatchDTO struct {
	// kunci
	ClassAttendanceSessionParticipantTypeID       uuid.UUID `json:"class_attendance_session_participant_type_id" validate:"required"`
	ClassAttendanceSessionParticipantTypeSchoolID uuid.UUID `json:"class_attendance_session_participant_type_school_id" validate:"required"`

	// field yang bisa diubah (opsional via PatchField)
	ClassAttendanceSessionParticipantTypeCode     PatchField[string]  `json:"class_attendance_session_participant_type_code"`
	ClassAttendanceSessionParticipantTypeLabel    PatchField[*string] `json:"class_attendance_session_participant_type_label"`
	ClassAttendanceSessionParticipantTypeSlug     PatchField[*string] `json:"class_attendance_session_participant_type_slug"`
	ClassAttendanceSessionParticipantTypeColor    PatchField[*string] `json:"class_attendance_session_participant_type_color"`
	ClassAttendanceSessionParticipantTypeDesc     PatchField[*string] `json:"class_attendance_session_participant_type_desc"`
	ClassAttendanceSessionParticipantTypeIsActive PatchField[bool]    `json:"class_attendance_session_participant_type_is_active"`
}

// ApplyPatch: terapkan perubahan ke model existing (in-place)
func (p *ClassAttendanceSessionParticipantTypePatchDTO) ApplyPatch(m *model.ClassAttendanceSessionParticipantTypeModel) {
	if p.ClassAttendanceSessionParticipantTypeCode.Set {
		m.ClassAttendanceSessionParticipantTypeCode = strings.TrimSpace(p.ClassAttendanceSessionParticipantTypeCode.Value)
	}
	if p.ClassAttendanceSessionParticipantTypeLabel.Set {
		m.ClassAttendanceSessionParticipantTypeLabel = trimPtr(p.ClassAttendanceSessionParticipantTypeLabel.Value)
	}
	if p.ClassAttendanceSessionParticipantTypeSlug.Set {
		m.ClassAttendanceSessionParticipantTypeSlug = trimPtr(p.ClassAttendanceSessionParticipantTypeSlug.Value)
	}
	if p.ClassAttendanceSessionParticipantTypeColor.Set {
		m.ClassAttendanceSessionParticipantTypeColor = trimPtr(p.ClassAttendanceSessionParticipantTypeColor.Value)
	}
	if p.ClassAttendanceSessionParticipantTypeDesc.Set {
		m.ClassAttendanceSessionParticipantTypeDesc = trimPtr(p.ClassAttendanceSessionParticipantTypeDesc.Value)
	}
	if p.ClassAttendanceSessionParticipantTypeIsActive.Set {
		m.ClassAttendanceSessionParticipantTypeIsActive = p.ClassAttendanceSessionParticipantTypeIsActive.Value
	}
	m.ClassAttendanceSessionParticipantTypeUpdatedAt = time.Now() // biarkan DB juga update
}

/* =========================================================
   LIST / FILTER DTO + Query Builder
   ========================================================= */

type ClassAttendanceSessionParticipantTypeListQuery struct {
	ClassAttendanceSessionParticipantTypeSchoolID uuid.UUID `json:"class_attendance_session_participant_type_school_id" validate:"required"`

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
func (q *ClassAttendanceSessionParticipantTypeListQuery) BuildQuery(db *gorm.DB) *gorm.DB {
	g := db.Model(&model.ClassAttendanceSessionParticipantTypeModel{}).
		Where("class_attendance_session_participant_type_school_id = ?", q.ClassAttendanceSessionParticipantTypeSchoolID).
		Where("class_attendance_session_participant_type_deleted_at IS NULL") // soft-delete aware default

	if q.CodeEq != nil && strings.TrimSpace(*q.CodeEq) != "" {
		g = g.Where("UPPER(class_attendance_session_participant_type_code) = UPPER(?)", strings.TrimSpace(*q.CodeEq))
	}
	if q.LabelQueryILK != nil && strings.TrimSpace(*q.LabelQueryILK) != "" {
		like := "%" + strings.TrimSpace(*q.LabelQueryILK) + "%"
		g = g.Where("class_attendance_session_participant_type_label ILIKE ?", like)
	}
	if q.SlugEq != nil && strings.TrimSpace(*q.SlugEq) != "" {
		g = g.Where("LOWER(class_attendance_session_participant_type_slug) = LOWER(?)", strings.TrimSpace(*q.SlugEq))
	}
	if q.OnlyActive != nil {
		g = g.Where("class_attendance_session_participant_type_is_active = ?", *q.OnlyActive)
	}

	switch q.Sort {
	case "created_at_asc":
		g = g.Order("class_attendance_session_participant_type_created_at ASC")
	case "code_asc":
		g = g.Order("class_attendance_session_participant_type_code ASC")
	case "code_desc":
		g = g.Order("class_attendance_session_participant_type_code DESC")
	case "label_asc":
		g = g.Order("class_attendance_session_participant_type_label ASC NULLS LAST")
	case "label_desc":
		g = g.Order("class_attendance_session_participant_type_label DESC NULLS LAST")
	default:
		g = g.Order("class_attendance_session_participant_type_created_at DESC")
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

type ClassAttendanceSessionParticipantTypeItem struct {
	ClassAttendanceSessionParticipantTypeID        uuid.UUID `json:"class_attendance_session_participant_type_id"`
	ClassAttendanceSessionParticipantTypeSchoolID  uuid.UUID `json:"class_attendance_session_participant_type_school_id"`
	ClassAttendanceSessionParticipantTypeCode      string    `json:"class_attendance_session_participant_type_code"`
	ClassAttendanceSessionParticipantTypeLabel     *string   `json:"class_attendance_session_participant_type_label,omitempty"`
	ClassAttendanceSessionParticipantTypeSlug      *string   `json:"class_attendance_session_participant_type_slug,omitempty"`
	ClassAttendanceSessionParticipantTypeColor     *string   `json:"class_attendance_session_participant_type_color,omitempty"`
	ClassAttendanceSessionParticipantTypeDesc      *string   `json:"class_attendance_session_participant_type_desc,omitempty"`
	ClassAttendanceSessionParticipantTypeIsActive  bool      `json:"class_attendance_session_participant_type_is_active"`
	ClassAttendanceSessionParticipantTypeCreatedAt time.Time `json:"class_attendance_session_participant_type_created_at"`
	ClassAttendanceSessionParticipantTypeUpdatedAt time.Time `json:"class_attendance_session_participant_type_updated_at"`
}

func FromModel(m *model.ClassAttendanceSessionParticipantTypeModel) ClassAttendanceSessionParticipantTypeItem {
	return ClassAttendanceSessionParticipantTypeItem{
		ClassAttendanceSessionParticipantTypeID:        m.ClassAttendanceSessionParticipantTypeID,
		ClassAttendanceSessionParticipantTypeSchoolID:  m.ClassAttendanceSessionParticipantTypeSchoolID,
		ClassAttendanceSessionParticipantTypeCode:      m.ClassAttendanceSessionParticipantTypeCode,
		ClassAttendanceSessionParticipantTypeLabel:     m.ClassAttendanceSessionParticipantTypeLabel,
		ClassAttendanceSessionParticipantTypeSlug:      m.ClassAttendanceSessionParticipantTypeSlug,
		ClassAttendanceSessionParticipantTypeColor:     m.ClassAttendanceSessionParticipantTypeColor,
		ClassAttendanceSessionParticipantTypeDesc:      m.ClassAttendanceSessionParticipantTypeDesc,
		ClassAttendanceSessionParticipantTypeIsActive:  m.ClassAttendanceSessionParticipantTypeIsActive,
		ClassAttendanceSessionParticipantTypeCreatedAt: m.ClassAttendanceSessionParticipantTypeCreatedAt,
		ClassAttendanceSessionParticipantTypeUpdatedAt: m.ClassAttendanceSessionParticipantTypeUpdatedAt,
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
