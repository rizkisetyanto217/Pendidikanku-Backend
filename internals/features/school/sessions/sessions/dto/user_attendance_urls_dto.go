// file: internals/features/attendance/user_attendance_urls/dto/user_attendance_url_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

/* =========================================================
   Validator singleton (opsional dipakai di controller)
========================================================= */

var validate = validator.New()

/* =========================================================
   Kinds (sinkron dg model)
========================================================= */

const (
	UAUKindImage      = "image"
	UAUKindVideo      = "video"
	UAUKindAttachment = "attachment"
	UAUKindLink       = "link"
	UAUKindAudio      = "audio"
)

/* =========================================================
   Create
   - Minimal: masjid_id, attendance_id, kind
   - Salah satu dari: href atau object_key boleh diisi (boleh dua-duanya)
========================================================= */

type CreateUserAttendanceURLRequest struct {
	// Tenant & owner (biasanya diisi dari helper resolver, bukan dari body)
	UserAttendanceURLMasjidID   uuid.UUID `json:"masjid_id" validate:"required"`
	UserAttendanceURLAttendance uuid.UUID `json:"attendance_id" validate:"required"`

	// Optional lookup type
	UserAttendanceTypeID *uuid.UUID `json:"type_id"`

	// Jenis/peran aset
	UserAttendanceURLKind string `json:"kind" validate:"required,max=24"`

	// Lokasi file/link
	UserAttendanceURLHref      *string `json:"href" validate:"omitempty,max=4000"`
	UserAttendanceURLObjectKey *string `json:"object_key" validate:"omitempty,max=2000"`

	// Metadata tampilan
	UserAttendanceURLLabel     *string `json:"label" validate:"omitempty,max=160"`
	UserAttendanceURLOrder     *int32  `json:"order" validate:"omitempty"`
	UserAttendanceURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping (opsional)
	UserAttendanceURLTrashURL           *string    `json:"trash_url" validate:"omitempty,max=4000"`
	UserAttendanceURLDeletePendingUntil *time.Time `json:"delete_pending_until"`

	// Uploader (opsional)
	UserAttendanceURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id"`
	UserAttendanceURLUploaderStudentID *uuid.UUID `json:"uploader_student_id"`
}

func (r *CreateUserAttendanceURLRequest) Normalize() {
	r.UserAttendanceURLKind = strings.TrimSpace(strings.ToLower(r.UserAttendanceURLKind))
	if r.UserAttendanceURLLabel != nil {
		lbl := strings.TrimSpace(*r.UserAttendanceURLLabel)
		r.UserAttendanceURLLabel = &lbl
	}
	if r.UserAttendanceURLHref != nil {
		h := strings.TrimSpace(*r.UserAttendanceURLHref)
		if h == "" {
			r.UserAttendanceURLHref = nil
		} else {
			r.UserAttendanceURLHref = &h
		}
	}
	if r.UserAttendanceURLObjectKey != nil {
		ok := strings.TrimSpace(*r.UserAttendanceURLObjectKey)
		if ok == "" {
			r.UserAttendanceURLObjectKey = nil
		} else {
			r.UserAttendanceURLObjectKey = &ok
		}
	}
}

func (r *CreateUserAttendanceURLRequest) Validate() error {
	// minimal rule: butuh salah satu dari href/object_key
	if (r.UserAttendanceURLHref == nil || strings.TrimSpace(*r.UserAttendanceURLHref) == "") &&
		(r.UserAttendanceURLObjectKey == nil || strings.TrimSpace(*r.UserAttendanceURLObjectKey) == "") {
		return errors.New("either href or object_key must be provided")
	}
	return validate.Struct(r)
}

/* =========================================================
   Update (PATCH)
   - Semua field pointer agar optional (partial update)
   - ObjectKeyOld tidak diekspos di request; diisi di service saat replace
========================================================= */

type UpdateUserAttendanceURLRequest struct {
	// ID baris yang mau diupdate (biasanya dari path)
	ID uuid.UUID `json:"-" validate:"required"`

	// Optional lookup type
	UserAttendanceTypeID *uuid.UUID `json:"type_id" validate:"omitempty"`

	// Jenis/peran aset
	UserAttendanceURLKind *string `json:"kind" validate:"omitempty,max=24"`

	// Lokasi file/link
	UserAttendanceURLHref      *string `json:"href" validate:"omitempty,max=4000"`
	UserAttendanceURLObjectKey *string `json:"object_key" validate:"omitempty,max=2000"`

	// Metadata tampilan
	UserAttendanceURLLabel     *string `json:"label" validate:"omitempty,max=160"`
	UserAttendanceURLOrder     *int32  `json:"order" validate:"omitempty"`
	UserAttendanceURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping
	UserAttendanceURLTrashURL           *string    `json:"trash_url" validate:"omitempty,max=4000"`
	UserAttendanceURLDeletePendingUntil *time.Time `json:"delete_pending_until" validate:"omitempty"`

	// Uploader
	UserAttendanceURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id" validate:"omitempty"`
	UserAttendanceURLUploaderStudentID *uuid.UUID `json:"uploader_student_id" validate:"omitempty"`
}

func (r *UpdateUserAttendanceURLRequest) Normalize() {
	if r.UserAttendanceURLKind != nil {
		k := strings.TrimSpace(strings.ToLower(*r.UserAttendanceURLKind))
		r.UserAttendanceURLKind = &k
	}
	if r.UserAttendanceURLLabel != nil {
		lbl := strings.TrimSpace(*r.UserAttendanceURLLabel)
		r.UserAttendanceURLLabel = &lbl
	}
	if r.UserAttendanceURLHref != nil {
		h := strings.TrimSpace(*r.UserAttendanceURLHref)
		if h == "" {
			r.UserAttendanceURLHref = nil
		} else {
			r.UserAttendanceURLHref = &h
		}
	}
	if r.UserAttendanceURLObjectKey != nil {
		ok := strings.TrimSpace(*r.UserAttendanceURLObjectKey)
		if ok == "" {
			r.UserAttendanceURLObjectKey = nil
		} else {
			r.UserAttendanceURLObjectKey = &ok
		}
	}
}

func (r *UpdateUserAttendanceURLRequest) Validate() error {
	// Tidak ada rule wajib selain ID, tapi tetap validasi panjang dll
	return validate.Struct(r)
}

/* =========================================================
   List (Query Params)
   - support filter masjid_id, attendance_id, kind, is_primary, q(label contains)
   - support paging & ordering
========================================================= */

type ListUserAttendanceURLRequest struct {
	// Filter
	MasjidID     *uuid.UUID `query:"masjid_id"`
	AttendanceID *uuid.UUID `query:"attendance_id"`
	Kind         *string    `query:"kind"`
	IsPrimary    *bool      `query:"is_primary"`
	Q            *string    `query:"q"` // cari di label (ILIKE %q%)

	// Paging
	Limit  int `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int `query:"offset" validate:"omitempty,min=0"`

	// Ordering: default "is_primary desc, order asc, created_at asc"
	// Field yang didukung: is_primary, order, created_at
	OrderBy *string `query:"order_by"` // contoh: "is_primary desc, order asc"
}

func (r *ListUserAttendanceURLRequest) Normalize() {
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
			// biarin raw; controller yang akan whitelist kolom
			r.OrderBy = &ob
		}
	}
}

func (r *ListUserAttendanceURLRequest) Validate() error {
	return validate.Struct(r)
}

/* =========================================================
   Response Item & List
========================================================= */

type UserAttendanceURLItem struct {
	ID                 uuid.UUID  `json:"id"`
	MasjidID           uuid.UUID  `json:"masjid_id"`
	AttendanceID       uuid.UUID  `json:"attendance_id"`
	TypeID             *uuid.UUID `json:"type_id,omitempty"`
	Kind               string     `json:"kind"`
	Href               *string    `json:"href,omitempty"`
	ObjectKey          *string    `json:"object_key,omitempty"`
	ObjectKeyOld       *string    `json:"object_key_old,omitempty"`
	Label              *string    `json:"label,omitempty"`
	Order              int32      `json:"order"`
	IsPrimary          bool       `json:"is_primary"`
	TrashURL           *string    `json:"trash_url,omitempty"`
	DeletePendingUntil *time.Time `json:"delete_pending_until,omitempty"`
	UploaderTeacherID  *uuid.UUID `json:"uploader_teacher_id,omitempty"`
	UploaderStudentID  *uuid.UUID `json:"uploader_student_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type ListUserAttendanceURLResponse struct {
	Items []UserAttendanceURLItem `json:"items"`
	Meta  ListMeta                `json:"meta"`
}

type ListMeta2 struct {
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	TotalItems int64 `json:"total_items"`
}

/* =========================================================
   Helpers mapping Model → DTO
   (Panggil dari service/controller saat menyusun response)
========================================================= */

type ModelUserAttendanceURL interface {
	// akses kolom yang diperlukan tanpa meng-import model asli di layer dto
	GetID() uuid.UUID
	GetMasjidID() uuid.UUID
	GetAttendanceID() uuid.UUID
	GetTypeID() *uuid.UUID
	GetKind() string
	GetHref() *string
	GetObjectKey() *string
	GetObjectKeyOld() *string
	GetLabel() *string
	GetOrder() int32
	GetIsPrimary() bool
	GetTrashURL() *string
	GetDeletePendingUntil() *time.Time
	GetUploaderTeacherID() *uuid.UUID
	GetUploaderStudentID() *uuid.UUID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetDeletedAtPtr() *time.Time
}

// Jika kamu tidak mau bikin interface getter di model,
// boleh langsung tulis fungsi mapper spesifik struct model.UserAttendanceURL.

// Example mapper (pseudo, sesuaikan jika tidak pakai interface di model):
/*
func FromModel(m model.UserAttendanceURL) UserAttendanceURLItem {
	var deletedAt *time.Time
	if m.UserAttendanceURLDeletedAt.Valid {
		t := m.UserAttendanceURLDeletedAt.Time
		deletedAt = &t
	}
	return UserAttendanceURLItem{
		ID:                 m.UserAttendanceURLID,
		MasjidID:           m.UserAttendanceURLMasjidID,
		AttendanceID:       m.UserAttendanceURLAttendance,
		TypeID:             m.UserAttendanceTypeID,
		Kind:               m.UserAttendanceURLKind,
		Href:               m.UserAttendanceURLHref,
		ObjectKey:          m.UserAttendanceURLObjectKey,
		ObjectKeyOld:       m.UserAttendanceURLObjectKeyOld,
		Label:              m.UserAttendanceURLLabel,
		Order:              m.UserAttendanceURLOrder,
		IsPrimary:          m.UserAttendanceURLIsPrimary,
		TrashURL:           m.UserAttendanceURLTrashURL,
		DeletePendingUntil: m.UserAttendanceURLDeletePendingUntil,
		UploaderTeacherID:  m.UserAttendanceURLUploaderTeacherID,
		UploaderStudentID:  m.UserAttendanceURLUploaderStudentID,
		CreatedAt:          m.UserAttendanceURLCreatedAt,
		UpdatedAt:          m.UserAttendanceURLUpdatedAt,
		DeletedAt:          deletedAt,
	}
}
*/

// Optional: bentuk sederhana untuk upsert URL dari JSON/multipart
type UAUUpsert struct {
	Kind      string  `json:"kind"` // default "attachment"
	Label     *string `json:"label"`
	Href      *string `json:"href"`
	ObjectKey *string `json:"object_key"`
	Order     *int32  `json:"order"`
	IsPrimary *bool   `json:"is_primary"`
	// Uploader optional:
	UploaderTeacherID *uuid.UUID `json:"uploader_teacher_id"`
	UploaderStudentID *uuid.UUID `json:"uploader_student_id"`
}

func (u *UAUUpsert) Normalize() {
	u.Kind = strings.TrimSpace(strings.ToLower(u.Kind))
	if u.Kind == "" {
		u.Kind = "attachment"
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
