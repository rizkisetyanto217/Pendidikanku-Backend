package dto

import (
	"strings"

	model "masjidku_backend/internals/features/school/sessions/sessions/model"

	"github.com/google/uuid"
)

/*
=========================================
DTO: ClassAttendanceSessionURL
- Upsert (buat/append)
- Patch (partial update)
- Lite (untuk response ringkas)
- Mapper & Normalizer
=========================================
*/

// ---------- Lite (untuk response ringkas di session detail/list) ----------
type ClassAttendanceSessionURLLite struct {
	ID        uuid.UUID `json:"class_attendance_session_url_id"`
	Label     *string   `json:"class_attendance_session_url_label,omitempty"`
	Href      string    `json:"class_attendance_session_url_href"`
	Kind      string    `json:"class_attendance_session_url_kind"`
	IsPrimary bool      `json:"class_attendance_session_url_is_primary"`
	Order     int       `json:"class_attendance_session_url_order"`
}

// Converter model → lite
func ToClassAttendanceSessionURLLite(m *model.ClassAttendanceSessionURLModel) ClassAttendanceSessionURLLite {
	href := ""
	if m.ClassAttendanceSessionURLHref != nil {
		href = *m.ClassAttendanceSessionURLHref
	}
	return ClassAttendanceSessionURLLite{
		ID:        m.ClassAttendanceSessionURLID,
		Label:     m.ClassAttendanceSessionURLLabel,
		Href:      href,
		Kind:      m.ClassAttendanceSessionURLKind,
		IsPrimary: m.ClassAttendanceSessionURLIsPrimary,
		Order:     m.ClassAttendanceSessionURLOrder,
	}
}

// ---------- Upsert (metadata dari FE atau hasil upload) ----------
type ClassAttendanceSessionURLUpsert struct {
	Kind      string  `json:"class_attendance_session_url_kind" validate:"required,min=1,max=24"`
	Label     *string `json:"class_attendance_session_url_label,omitempty" validate:"omitempty,max=160"`
	Href      *string `json:"class_attendance_session_url_href,omitempty" validate:"omitempty,url"`
	ObjectKey *string `json:"class_attendance_session_url_object_key,omitempty" validate:"omitempty"`
	Order     int     `json:"class_attendance_session_url_order"`
	IsPrimary bool    `json:"class_attendance_session_url_is_primary"`
}

// Normalisasi ringan
func (u *ClassAttendanceSessionURLUpsert) Normalize() {
	u.Kind = strings.TrimSpace(u.Kind)
	if u.Kind == "" {
		u.Kind = "attachment"
	}
	if u.Label != nil {
		v := strings.TrimSpace(*u.Label)
		if v == "" {
			u.Label = nil
		} else {
			u.Label = &v
		}
	}
	if u.Href != nil {
		v := strings.TrimSpace(*u.Href)
		if v == "" {
			u.Href = nil
		} else {
			u.Href = &v
		}
	}
	if u.ObjectKey != nil {
		v := strings.TrimSpace(*u.ObjectKey)
		if v == "" {
			u.ObjectKey = nil
		} else {
			u.ObjectKey = &v
		}
	}
}

// ---------- Patch (partial update untuk /book-urls style) ----------
type ClassAttendanceSessionURLPatch struct {
	Label     *string `json:"class_attendance_session_url_label,omitempty" validate:"omitempty,max=160"`
	Order     *int    `json:"class_attendance_session_url_order,omitempty"`
	IsPrimary *bool   `json:"class_attendance_session_url_is_primary,omitempty"`
	Kind      *string `json:"class_attendance_session_url_kind,omitempty" validate:"omitempty,max=24"`
	Href      *string `json:"class_attendance_session_url_href,omitempty" validate:"omitempty,url"`
	ObjectKey *string `json:"class_attendance_session_url_object_key,omitempty" validate:"omitempty"`
}

// Normalisasi untuk patch
func (p *ClassAttendanceSessionURLPatch) Normalize() {
	trim := func(s *string) *string {
		if s == nil {
			return nil
		}
		v := strings.TrimSpace(*s)
		if v == "" {
			return nil
		}
		return &v
	}
	p.Label = trim(p.Label)
	p.Kind = trim(p.Kind)
	p.Href = trim(p.Href)
	p.ObjectKey = trim(p.ObjectKey)
}

// ---------- Request util (opsional, untuk bulk create) ----------
type ClassAttendanceSessionURLCreateRequest struct {
	MasjidID  uuid.UUID                         `json:"class_attendance_session_url_masjid_id" validate:"required"`
	SessionID uuid.UUID                         `json:"class_attendance_session_url_session_id" validate:"required"`
	URLs      []ClassAttendanceSessionURLUpsert `json:"urls" validate:"required,dive"`
}

func (r *ClassAttendanceSessionURLCreateRequest) Normalize() {
	for i := range r.URLs {
		r.URLs[i].Normalize()
	}
}

// Mapper: request → models (untuk tx.Create(&models))
func (r *ClassAttendanceSessionURLCreateRequest) ToModels() []model.ClassAttendanceSessionURLModel {
	out := make([]model.ClassAttendanceSessionURLModel, 0, len(r.URLs))
	for _, u := range r.URLs {
		row := model.ClassAttendanceSessionURLModel{
			ClassAttendanceSessionURLMasjidID:  r.MasjidID,
			ClassAttendanceSessionURLSessionID: r.SessionID,
			ClassAttendanceSessionURLKind:      u.Kind,
			ClassAttendanceSessionURLLabel:     u.Label,
			ClassAttendanceSessionURLHref:      u.Href,
			ClassAttendanceSessionURLObjectKey: u.ObjectKey,
			ClassAttendanceSessionURLOrder:     u.Order,
			ClassAttendanceSessionURLIsPrimary: u.IsPrimary,
		}
		if strings.TrimSpace(row.ClassAttendanceSessionURLKind) == "" {
			row.ClassAttendanceSessionURLKind = "attachment"
		}
		out = append(out, row)
	}
	return out
}
