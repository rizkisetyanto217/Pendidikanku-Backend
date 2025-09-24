// file: internals/features/attendance/user_class_session_attendance_urls/dto/user_class_session_attendance_url_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"
)

/* =========================================================
   Validator singleton
========================================================= */

var validate = validator.New()

/* =========================================================
   Kinds
========================================================= */

const (
	UCSAURLKindImage      = "image"
	UCSAURLKindVideo      = "video"
	UCSAURLKindAttachment = "attachment"
	UCSAURLKindLink       = "link"
	UCSAURLKindAudio      = "audio"
)

/* =========================================================
   Create
   - Minimal: masjid_id, attendance_id, kind
   - Salah satu dari: href atau object_key (boleh dua-duanya)
========================================================= */

type CreateUserClassSessionAttendanceURLRequest struct {
	// Tenant & owner (biasanya diisi dari resolver, bukan dari body)
	UserClassSessionAttendanceURLMasjidID     uuid.UUID `json:"masjid_id"      validate:"required"`
	UserClassSessionAttendanceURLAttendanceID uuid.UUID `json:"attendance_id"  validate:"required"`

	// Optional lookup type
	UserClassSessionAttendanceTypeID *uuid.UUID `json:"type_id"`

	// Jenis/peran aset
	UserClassSessionAttendanceURLKind string `json:"kind" validate:"required,max=24"`

	// Lokasi file/link
	UserClassSessionAttendanceURLHref      *string `json:"href"        validate:"omitempty,max=4000"`
	UserClassSessionAttendanceURLObjectKey *string `json:"object_key"  validate:"omitempty,max=2000"`

	// Metadata tampilan
	UserClassSessionAttendanceURLLabel     *string `json:"label"      validate:"omitempty,max=160"`
	UserClassSessionAttendanceURLOrder     *int    `json:"order"      validate:"omitempty"`
	UserClassSessionAttendanceURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping (opsional)
	UserClassSessionAttendanceURLTrashURL           *string    `json:"trash_url"          validate:"omitempty,max=4000"`
	UserClassSessionAttendanceURLDeletePendingUntil *time.Time `json:"delete_pending_until"`

	// Uploader (opsional)
	UserClassSessionAttendanceURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id"`
	UserClassSessionAttendanceURLUploaderStudentID *uuid.UUID `json:"uploader_student_id"`
}

func (r *CreateUserClassSessionAttendanceURLRequest) Normalize() {
	r.UserClassSessionAttendanceURLKind = strings.TrimSpace(strings.ToLower(r.UserClassSessionAttendanceURLKind))
	if r.UserClassSessionAttendanceURLLabel != nil {
		lbl := strings.TrimSpace(*r.UserClassSessionAttendanceURLLabel)
		r.UserClassSessionAttendanceURLLabel = &lbl
	}
	if r.UserClassSessionAttendanceURLHref != nil {
		h := strings.TrimSpace(*r.UserClassSessionAttendanceURLHref)
		if h == "" {
			r.UserClassSessionAttendanceURLHref = nil
		} else {
			r.UserClassSessionAttendanceURLHref = &h
		}
	}
	if r.UserClassSessionAttendanceURLObjectKey != nil {
		ok := strings.TrimSpace(*r.UserClassSessionAttendanceURLObjectKey)
		if ok == "" {
			r.UserClassSessionAttendanceURLObjectKey = nil
		} else {
			r.UserClassSessionAttendanceURLObjectKey = &ok
		}
	}
}

func (r *CreateUserClassSessionAttendanceURLRequest) Validate() error {
	// minimal: butuh salah satu dari href/object_key
	if (r.UserClassSessionAttendanceURLHref == nil || strings.TrimSpace(*r.UserClassSessionAttendanceURLHref) == "") &&
		(r.UserClassSessionAttendanceURLObjectKey == nil || strings.TrimSpace(*r.UserClassSessionAttendanceURLObjectKey) == "") {
		return errors.New("either href or object_key must be provided")
	}
	return validate.Struct(r)
}

/* =========================================================
   Update (PATCH)
   - Semua field pointer agar optional (partial update)
   - ObjectKeyOld tidak diekspos di request; diisi di service saat replace
========================================================= */

type UpdateUserClassSessionAttendanceURLRequest struct {
	// ID baris yang mau diupdate (biasanya dari path)
	ID uuid.UUID `json:"-" validate:"required"`

	// Optional lookup type
	UserClassSessionAttendanceTypeID *uuid.UUID `json:"type_id" validate:"omitempty"`

	// Jenis/peran aset
	UserClassSessionAttendanceURLKind *string `json:"kind" validate:"omitempty,max=24"`

	// Lokasi file/link
	UserClassSessionAttendanceURLHref      *string `json:"href"        validate:"omitempty,max=4000"`
	UserClassSessionAttendanceURLObjectKey *string `json:"object_key"  validate:"omitempty,max=2000"`

	// Metadata tampilan
	UserClassSessionAttendanceURLLabel     *string `json:"label"      validate:"omitempty,max=160"`
	UserClassSessionAttendanceURLOrder     *int    `json:"order"      validate:"omitempty"`
	UserClassSessionAttendanceURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping
	UserClassSessionAttendanceURLTrashURL           *string    `json:"trash_url"          validate:"omitempty,max=4000"`
	UserClassSessionAttendanceURLDeletePendingUntil *time.Time `json:"delete_pending_until" validate:"omitempty"`

	// Uploader
	UserClassSessionAttendanceURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id" validate:"omitempty"`
	UserClassSessionAttendanceURLUploaderStudentID *uuid.UUID `json:"uploader_student_id" validate:"omitempty"`
}

func (r *UpdateUserClassSessionAttendanceURLRequest) Normalize() {
	if r.UserClassSessionAttendanceURLKind != nil {
		k := strings.TrimSpace(strings.ToLower(*r.UserClassSessionAttendanceURLKind))
		r.UserClassSessionAttendanceURLKind = &k
	}
	if r.UserClassSessionAttendanceURLLabel != nil {
		lbl := strings.TrimSpace(*r.UserClassSessionAttendanceURLLabel)
		r.UserClassSessionAttendanceURLLabel = &lbl
	}
	if r.UserClassSessionAttendanceURLHref != nil {
		h := strings.TrimSpace(*r.UserClassSessionAttendanceURLHref)
		if h == "" {
			r.UserClassSessionAttendanceURLHref = nil
		} else {
			r.UserClassSessionAttendanceURLHref = &h
		}
	}
	if r.UserClassSessionAttendanceURLObjectKey != nil {
		ok := strings.TrimSpace(*r.UserClassSessionAttendanceURLObjectKey)
		if ok == "" {
			r.UserClassSessionAttendanceURLObjectKey = nil
		} else {
			r.UserClassSessionAttendanceURLObjectKey = &ok
		}
	}
}

func (r *UpdateUserClassSessionAttendanceURLRequest) Validate() error {
	return validate.Struct(r)
}

/* =========================================================
   List (Query Params)
   - filter masjid_id, attendance_id, kind, is_primary, q(label contains)
   - paging & ordering
========================================================= */

type ListUserClassSessionAttendanceURLRequest struct {
	// Filter
	MasjidID     *uuid.UUID `query:"masjid_id"`
	AttendanceID *uuid.UUID `query:"attendance_id"`
	Kind         *string    `query:"kind"`
	IsPrimary    *bool      `query:"is_primary"`
	Q            *string    `query:"q"` // cari di label (ILIKE %q%)

	// Paging
	Limit  int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int `query:"offset" validate:"omitempty,min=0"`

	// Ordering: default "is_primary desc, order asc, created_at asc"
	// Field yang didukung: is_primary, order, created_at
	OrderBy *string `query:"order_by"` // contoh: "is_primary desc, order asc"
}

func (r *ListUserClassSessionAttendanceURLRequest) Normalize() {
	if r.Kind != nil {
		k := strings.TrimSpace(strings.ToLower(*r.Kind))
		r.Kind = &k
	}
	if r.Q != nil {
		q := strings.TrimSpace(*r.Q)
		if q == "" {
			r.Q = nil
		} else {
			r.Q = &q
		}
	}
	if r.Limit == 0 {
		r.Limit = 20
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.OrderBy != nil {
		ob := strings.TrimSpace(*r.OrderBy)
		if ob == "" {
			r.OrderBy = nil
		} else {
			// biarkan raw; whitelist kolom di controller
			r.OrderBy = &ob
		}
	}
}

func (r *ListUserClassSessionAttendanceURLRequest) Validate() error {
	return validate.Struct(r)
}

/* =========================================================
   Response Item & List
========================================================= */

type UserClassSessionAttendanceURLItem struct {
	ID                 uuid.UUID  `json:"id"`
	MasjidID           uuid.UUID  `json:"masjid_id"`
	AttendanceID       uuid.UUID  `json:"attendance_id"`
	TypeID             *uuid.UUID `json:"type_id,omitempty"`
	Kind               string     `json:"kind"`
	Href               *string    `json:"href,omitempty"`
	ObjectKey          *string    `json:"object_key,omitempty"`
	ObjectKeyOld       *string    `json:"object_key_old,omitempty"`
	Label              *string    `json:"label,omitempty"`
	Order              int        `json:"order"`
	IsPrimary          bool       `json:"is_primary"`
	TrashURL           *string    `json:"trash_url,omitempty"`
	DeletePendingUntil *time.Time `json:"delete_pending_until,omitempty"`
	UploaderTeacherID  *uuid.UUID `json:"uploader_teacher_id,omitempty"`
	UploaderStudentID  *uuid.UUID `json:"uploader_student_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type ListUserClassSessionAttendanceURLResponse struct {
	Items []UserClassSessionAttendanceURLItem `json:"items"`
	Meta  ListMeta                            `json:"meta"`
}

type ListMetaUserClassSessionAttendanceURL struct {
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	TotalItems int64 `json:"total_items"`
}

/* =========================================================
   Mapper Model → DTO
========================================================= */

func FromModelUserClassSessionAttendanceURL(m model.UserClassSessionAttendanceURLModel) UserClassSessionAttendanceURLItem {
	var deletedAt *time.Time
	if m.UserClassSessionAttendanceURLDeletedAt.Valid {
		t := m.UserClassSessionAttendanceURLDeletedAt.Time
		deletedAt = &t
	}
	return UserClassSessionAttendanceURLItem{
		ID:                 m.UserClassSessionAttendanceURLID,
		MasjidID:           m.UserClassSessionAttendanceURLMasjidID,
		AttendanceID:       m.UserClassSessionAttendanceURLAttendanceID,
		TypeID:             m.UserClassSessionAttendanceTypeID,
		Kind:               m.UserClassSessionAttendanceURLKind,
		Href:               m.UserClassSessionAttendanceURLHref,
		ObjectKey:          m.UserClassSessionAttendanceURLObjectKey,
		ObjectKeyOld:       m.UserClassSessionAttendanceURLObjectKeyOld,
		Label:              m.UserClassSessionAttendanceURLLabel,
		Order:              m.UserClassSessionAttendanceURLOrder,
		IsPrimary:          m.UserClassSessionAttendanceURLIsPrimary,
		TrashURL:           m.UserClassSessionAttendanceURLTrashURL,
		DeletePendingUntil: m.UserClassSessionAttendanceURLDeletePendingUntil,
		UploaderTeacherID:  m.UserClassSessionAttendanceURLUploaderTeacherID,
		UploaderStudentID:  m.UserClassSessionAttendanceURLUploaderStudentID,
		CreatedAt:          m.UserClassSessionAttendanceURLCreatedAt,
		UpdatedAt:          m.UserClassSessionAttendanceURLUpdatedAt,
		DeletedAt:          deletedAt,
	}
}

/* =========================================================
   Upsert helper sederhana (opsional)
========================================================= */

type UCSAURLUpsert struct {
	Kind      string  `json:"kind"` // default "attachment"
	Label     *string `json:"label"`
	Href      *string `json:"href"`
	ObjectKey *string `json:"object_key"`
	Order     *int    `json:"order"`
	IsPrimary *bool   `json:"is_primary"`
	// Uploader (opsional)
	UploaderTeacherID *uuid.UUID `json:"uploader_teacher_id"`
	UploaderStudentID *uuid.UUID `json:"uploader_student_id"`
}

func (u *UCSAURLUpsert) Normalize() {
	u.Kind = strings.TrimSpace(strings.ToLower(u.Kind))
	if u.Kind == "" {
		u.Kind = UCSAURLKindAttachment
	}
	if u.Label != nil {
		lbl := strings.TrimSpace(*u.Label)
		u.Label = &lbl
	}
	if u.Href != nil {
		h := strings.TrimSpace(*u.Href)
		if h == "" {
			u.Href = nil
		} else {
			u.Href = &h
		}
	}
	if u.ObjectKey != nil {
		ok := strings.TrimSpace(*u.ObjectKey)
		if ok == "" {
			u.ObjectKey = nil
		} else {
			u.ObjectKey = &ok
		}
	}
}
