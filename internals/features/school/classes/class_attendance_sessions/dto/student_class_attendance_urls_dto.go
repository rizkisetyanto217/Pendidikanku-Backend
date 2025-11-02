// file: internals/features/attendance/student_class_session_attendance_urls/dto/student_class_session_attendance_url_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	model "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
)

/* =========================================================
   Validator singleton
========================================================= */

var validate = validator.New()

/* =========================================================
   Kinds
========================================================= */

const (
	SCSAURLKindImage      = "image"
	SCSAURLKindVideo      = "video"
	SCSAURLKindAttachment = "attachment"
	SCSAURLKindLink       = "link"
	SCSAURLKindAudio      = "audio"
)

/* =========================================================
   Create
   - Minimal: school_id, attendance_id, kind
   - Salah satu dari: href atau object_key (boleh dua-duanya)
========================================================= */

type StudentClassSessionAttendanceURLCreateRequest struct {
	// Tenant & owner (biasanya diisi dari resolver, bukan dari body)
	StudentClassSessionAttendanceURLSchoolID     uuid.UUID `json:"school_id"      validate:"required"`
	StudentClassSessionAttendanceURLAttendanceID uuid.UUID `json:"attendance_id"  validate:"required"`

	// Optional lookup type
	StudentClassSessionAttendanceTypeID *uuid.UUID `json:"type_id"`

	// Jenis/peran aset
	StudentClassSessionAttendanceURLKind string `json:"kind" validate:"required,max=24"`

	// Lokasi file/link
	StudentClassSessionAttendanceURLHref      *string `json:"href"        validate:"omitempty,max=4000"`
	StudentClassSessionAttendanceURLObjectKey *string `json:"object_key"  validate:"omitempty,max=2000"`

	// Metadata tampilan
	StudentClassSessionAttendanceURLLabel     *string `json:"label"      validate:"omitempty,max=160"`
	StudentClassSessionAttendanceURLOrder     *int    `json:"order"      validate:"omitempty"`
	StudentClassSessionAttendanceURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping (opsional)
	StudentClassSessionAttendanceURLTrashURL           *string    `json:"trash_url"          validate:"omitempty,max=4000"`
	StudentClassSessionAttendanceURLDeletePendingUntil *time.Time `json:"delete_pending_until"`

	// Uploader (opsional)
	StudentClassSessionAttendanceURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id"`
	StudentClassSessionAttendanceURLUploaderStudentID *uuid.UUID `json:"uploader_student_id"`
}

func (r *StudentClassSessionAttendanceURLCreateRequest) Normalize() {
	r.StudentClassSessionAttendanceURLKind = strings.TrimSpace(strings.ToLower(r.StudentClassSessionAttendanceURLKind))
	if r.StudentClassSessionAttendanceURLLabel != nil {
		lbl := strings.TrimSpace(*r.StudentClassSessionAttendanceURLLabel)
		r.StudentClassSessionAttendanceURLLabel = &lbl
	}
	if r.StudentClassSessionAttendanceURLHref != nil {
		h := strings.TrimSpace(*r.StudentClassSessionAttendanceURLHref)
		if h == "" {
			r.StudentClassSessionAttendanceURLHref = nil
		} else {
			r.StudentClassSessionAttendanceURLHref = &h
		}
	}
	if r.StudentClassSessionAttendanceURLObjectKey != nil {
		ok := strings.TrimSpace(*r.StudentClassSessionAttendanceURLObjectKey)
		if ok == "" {
			r.StudentClassSessionAttendanceURLObjectKey = nil
		} else {
			r.StudentClassSessionAttendanceURLObjectKey = &ok
		}
	}
}

func (r *StudentClassSessionAttendanceURLCreateRequest) Validate() error {
	// minimal: butuh salah satu dari href/object_key
	if (r.StudentClassSessionAttendanceURLHref == nil || strings.TrimSpace(*r.StudentClassSessionAttendanceURLHref) == "") &&
		(r.StudentClassSessionAttendanceURLObjectKey == nil || strings.TrimSpace(*r.StudentClassSessionAttendanceURLObjectKey) == "") {
		return errors.New("either href or object_key must be provided")
	}
	return validate.Struct(r)
}

/* =========================================================
   Update (PATCH)
   - Semua field pointer agar optional (partial update)
   - ObjectKeyOld tidak diekspos di request; diisi di service saat replace
========================================================= */

type StudentClassSessionAttendanceURLUpdateRequest struct {
	// ID baris yang mau diupdate (biasanya dari path)
	ID uuid.UUID `json:"-" validate:"required"`

	// Optional lookup type
	StudentClassSessionAttendanceTypeID *uuid.UUID `json:"type_id" validate:"omitempty"`

	// Jenis/peran aset
	StudentClassSessionAttendanceURLKind *string `json:"kind" validate:"omitempty,max=24"`

	// Lokasi file/link
	StudentClassSessionAttendanceURLHref      *string `json:"href"        validate:"omitempty,max=4000"`
	StudentClassSessionAttendanceURLObjectKey *string `json:"object_key"  validate:"omitempty,max=2000"`

	// Metadata tampilan
	StudentClassSessionAttendanceURLLabel     *string `json:"label"      validate:"omitempty,max=160"`
	StudentClassSessionAttendanceURLOrder     *int    `json:"order"      validate:"omitempty"`
	StudentClassSessionAttendanceURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping
	StudentClassSessionAttendanceURLTrashURL           *string    `json:"trash_url"          validate:"omitempty,max=4000"`
	StudentClassSessionAttendanceURLDeletePendingUntil *time.Time `json:"delete_pending_until" validate:"omitempty"`

	// Uploader
	StudentClassSessionAttendanceURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id" validate:"omitempty"`
	StudentClassSessionAttendanceURLUploaderStudentID *uuid.UUID `json:"uploader_student_id" validate:"omitempty"`
}

func (r *StudentClassSessionAttendanceURLUpdateRequest) Normalize() {
	if r.StudentClassSessionAttendanceURLKind != nil {
		k := strings.TrimSpace(strings.ToLower(*r.StudentClassSessionAttendanceURLKind))
		r.StudentClassSessionAttendanceURLKind = &k
	}
	if r.StudentClassSessionAttendanceURLLabel != nil {
		lbl := strings.TrimSpace(*r.StudentClassSessionAttendanceURLLabel)
		r.StudentClassSessionAttendanceURLLabel = &lbl
	}
	if r.StudentClassSessionAttendanceURLHref != nil {
		h := strings.TrimSpace(*r.StudentClassSessionAttendanceURLHref)
		if h == "" {
			r.StudentClassSessionAttendanceURLHref = nil
		} else {
			r.StudentClassSessionAttendanceURLHref = &h
		}
	}
	if r.StudentClassSessionAttendanceURLObjectKey != nil {
		ok := strings.TrimSpace(*r.StudentClassSessionAttendanceURLObjectKey)
		if ok == "" {
			r.StudentClassSessionAttendanceURLObjectKey = nil
		} else {
			r.StudentClassSessionAttendanceURLObjectKey = &ok
		}
	}
}

func (r *StudentClassSessionAttendanceURLUpdateRequest) Validate() error {
	return validate.Struct(r)
}

/* =========================================================
   List (Query Params)
   - filter school_id, attendance_id, kind, is_primary, q(label contains)
   - paging & ordering
========================================================= */

type StudentClassSessionAttendanceURLListRequest struct {
	// Filter
	SchoolID     *uuid.UUID `query:"school_id"`
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

func (r *StudentClassSessionAttendanceURLListRequest) Normalize() {
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

func (r *StudentClassSessionAttendanceURLListRequest) Validate() error {
	return validate.Struct(r)
}

/* =========================================================
   Response Item & List
========================================================= */

type StudentClassSessionAttendanceURLItem struct {
	ID                 uuid.UUID  `json:"id"`
	SchoolID           uuid.UUID  `json:"school_id"`
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

type StudentClassSessionAttendanceURLListResponse struct {
	Items []StudentClassSessionAttendanceURLItem `json:"items"`
	Meta  ListMeta                               `json:"meta"`
}

type ListMetaStudentClassSessionAttendanceURL struct {
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	TotalItems int64 `json:"total_items"`
}

/* =========================================================
   Mapper Model → DTO
========================================================= */

func FromModelStudentClassSessionAttendanceURL(m model.StudentClassSessionAttendanceURLModel) StudentClassSessionAttendanceURLItem {
	var deletedAt *time.Time
	if m.StudentClassSessionAttendanceURLDeletedAt.Valid {
		t := m.StudentClassSessionAttendanceURLDeletedAt.Time
		deletedAt = &t
	}
	return StudentClassSessionAttendanceURLItem{
		ID:                 m.StudentClassSessionAttendanceURLID,
		SchoolID:           m.StudentClassSessionAttendanceURLSchoolID,
		AttendanceID:       m.StudentClassSessionAttendanceURLAttendanceID,
		TypeID:             m.StudentClassSessionAttendanceTypeID,
		Kind:               m.StudentClassSessionAttendanceURLKind,
		Href:               m.StudentClassSessionAttendanceURLHref,
		ObjectKey:          m.StudentClassSessionAttendanceURLObjectKey,
		ObjectKeyOld:       m.StudentClassSessionAttendanceURLObjectKeyOld,
		Label:              m.StudentClassSessionAttendanceURLLabel,
		Order:              m.StudentClassSessionAttendanceURLOrder,
		IsPrimary:          m.StudentClassSessionAttendanceURLIsPrimary,
		TrashURL:           m.StudentClassSessionAttendanceURLTrashURL,
		DeletePendingUntil: m.StudentClassSessionAttendanceURLDeletePendingUntil,
		UploaderTeacherID:  m.StudentClassSessionAttendanceURLUploaderTeacherID,
		UploaderStudentID:  m.StudentClassSessionAttendanceURLUploaderStudentID,
		CreatedAt:          m.StudentClassSessionAttendanceURLCreatedAt,
		UpdatedAt:          m.StudentClassSessionAttendanceURLUpdatedAt,
		DeletedAt:          deletedAt,
	}
}

/* =========================================================
   Upsert helper sederhana (opsional)
========================================================= */

type SCSAURLUpsert struct {
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

func (u *SCSAURLUpsert) Normalize() {
	u.Kind = strings.TrimSpace(strings.ToLower(u.Kind))
	if u.Kind == "" {
		u.Kind = SCSAURLKindAttachment
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
