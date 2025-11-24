// file: internals/features/attendance/student_class_session_attendance_urls/dto/student_class_session_attendance_url_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	model "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/model"
)

/* =========================================================
   Validator singleton
========================================================= */

var validate = validator.New()

/* =========================================================
   Kinds
========================================================= */

const (
	CASPURLKindImage      = "image"
	CASPURLKindVideo      = "video"
	CASPURLKindAttachment = "attachment"
	CASPURLKindLink       = "link"
	CASPURLKindAudio      = "audio"
)

/* =========================================================
   Create
   - Minimal: school_id, participant_id, kind
   - Salah satu dari: href/url atau object_key (boleh dua-duanya)
========================================================= */

type ClassAttendanceSessionParticipantURLCreateRequest struct {
	// Tenant & owner (biasanya diisi dari resolver, bukan dari body)
	ClassAttendanceSessionParticipantURLSchoolID      uuid.UUID `json:"school_id"       validate:"required"`
	ClassAttendanceSessionParticipantURLParticipantID uuid.UUID `json:"participant_id" validate:"required"`

	// Optional lookup type
	ClassAttendanceSessionParticipantTypeID *uuid.UUID `json:"type_id"`

	// Jenis/peran aset
	ClassAttendanceSessionParticipantURLKind string `json:"kind" validate:"required,max=24"`

	// Lokasi file/link
	// secara JSON kita tetap pakai "href" untuk URL utama
	ClassAttendanceSessionParticipantURLHref      *string `json:"href"       validate:"omitempty,max=4000"`
	ClassAttendanceSessionParticipantURLObjectKey *string `json:"object_key" validate:"omitempty,max=2000"`

	// Metadata tampilan
	ClassAttendanceSessionParticipantURLLabel     *string `json:"label"      validate:"omitempty,max=160"`
	ClassAttendanceSessionParticipantURLOrder     *int    `json:"order"      validate:"omitempty"`
	ClassAttendanceSessionParticipantURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping (opsional)
	ClassAttendanceSessionParticipantURLDeletePendingUntil *time.Time `json:"delete_pending_until"`

	// Uploader (opsional)
	ClassAttendanceSessionParticipantURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id"`
	ClassAttendanceSessionParticipantURLUploaderStudentID *uuid.UUID `json:"uploader_student_id"`
}

func (r *ClassAttendanceSessionParticipantURLCreateRequest) Normalize() {
	r.ClassAttendanceSessionParticipantURLKind = strings.TrimSpace(strings.ToLower(r.ClassAttendanceSessionParticipantURLKind))
	if r.ClassAttendanceSessionParticipantURLLabel != nil {
		lbl := strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLLabel)
		r.ClassAttendanceSessionParticipantURLLabel = &lbl
	}
	if r.ClassAttendanceSessionParticipantURLHref != nil {
		h := strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLHref)
		if h == "" {
			r.ClassAttendanceSessionParticipantURLHref = nil
		} else {
			r.ClassAttendanceSessionParticipantURLHref = &h
		}
	}
	if r.ClassAttendanceSessionParticipantURLObjectKey != nil {
		ok := strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLObjectKey)
		if ok == "" {
			r.ClassAttendanceSessionParticipantURLObjectKey = nil
		} else {
			r.ClassAttendanceSessionParticipantURLObjectKey = &ok
		}
	}
}

func (r *ClassAttendanceSessionParticipantURLCreateRequest) Validate() error {
	// minimal: butuh salah satu dari href/object_key
	if (r.ClassAttendanceSessionParticipantURLHref == nil || strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLHref) == "") &&
		(r.ClassAttendanceSessionParticipantURLObjectKey == nil || strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLObjectKey) == "") {
		return errors.New("either href or object_key must be provided")
	}
	return validate.Struct(r)
}

/* =========================================================
   Update (PATCH)
   - Semua field pointer agar optional (partial update)
   - *_old & retensi diisi di service saat replace
========================================================= */

type ClassAttendanceSessionParticipantURLUpdateRequest struct {
	// ID baris yang mau diupdate (biasanya dari path)
	ID uuid.UUID `json:"-" validate:"required"`

	// Optional lookup type
	ClassAttendanceSessionParticipantTypeID *uuid.UUID `json:"type_id" validate:"omitempty"`

	// Jenis/peran aset
	ClassAttendanceSessionParticipantURLKind *string `json:"kind" validate:"omitempty,max=24"`

	// Lokasi file/link
	ClassAttendanceSessionParticipantURLHref      *string `json:"href"       validate:"omitempty,max=4000"`
	ClassAttendanceSessionParticipantURLObjectKey *string `json:"object_key" validate:"omitempty,max=2000"`

	// Metadata tampilan
	ClassAttendanceSessionParticipantURLLabel     *string `json:"label"      validate:"omitempty,max=160"`
	ClassAttendanceSessionParticipantURLOrder     *int    `json:"order"      validate:"omitempty"`
	ClassAttendanceSessionParticipantURLIsPrimary *bool   `json:"is_primary" validate:"omitempty"`

	// Housekeeping
	ClassAttendanceSessionParticipantURLDeletePendingUntil *time.Time `json:"delete_pending_until" validate:"omitempty"`

	// Uploader
	ClassAttendanceSessionParticipantURLUploaderTeacherID *uuid.UUID `json:"uploader_teacher_id" validate:"omitempty"`
	ClassAttendanceSessionParticipantURLUploaderStudentID *uuid.UUID `json:"uploader_student_id" validate:"omitempty"`
}

func (r *ClassAttendanceSessionParticipantURLUpdateRequest) Normalize() {
	if r.ClassAttendanceSessionParticipantURLKind != nil {
		k := strings.TrimSpace(strings.ToLower(*r.ClassAttendanceSessionParticipantURLKind))
		r.ClassAttendanceSessionParticipantURLKind = &k
	}
	if r.ClassAttendanceSessionParticipantURLLabel != nil {
		lbl := strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLLabel)
		r.ClassAttendanceSessionParticipantURLLabel = &lbl
	}
	if r.ClassAttendanceSessionParticipantURLHref != nil {
		h := strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLHref)
		if h == "" {
			r.ClassAttendanceSessionParticipantURLHref = nil
		} else {
			r.ClassAttendanceSessionParticipantURLHref = &h
		}
	}
	if r.ClassAttendanceSessionParticipantURLObjectKey != nil {
		ok := strings.TrimSpace(*r.ClassAttendanceSessionParticipantURLObjectKey)
		if ok == "" {
			r.ClassAttendanceSessionParticipantURLObjectKey = nil
		} else {
			r.ClassAttendanceSessionParticipantURLObjectKey = &ok
		}
	}
}

func (r *ClassAttendanceSessionParticipantURLUpdateRequest) Validate() error {
	return validate.Struct(r)
}

/* =========================================================
   List (Query Params)
   - filter school_id, participant_id, kind, is_primary, q(label contains)
   - paging & ordering
========================================================= */

type ClassAttendanceSessionParticipantURLListRequest struct {
	// Filter
	SchoolID      *uuid.UUID `query:"school_id"`
	ParticipantID *uuid.UUID `query:"participant_id"`
	Kind          *string    `query:"kind"`
	IsPrimary     *bool      `query:"is_primary"`
	Q             *string    `query:"q"` // cari di label (ILIKE %q%)

	// Paging
	Limit  int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int `query:"offset" validate:"omitempty,min=0"`

	// Ordering: default "is_primary desc, order asc, created_at asc"
	// Field yang didukung: is_primary, order, created_at
	OrderBy *string `query:"order_by"` // contoh: "is_primary desc, order asc"
}

func (r *ClassAttendanceSessionParticipantURLListRequest) Normalize() {
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

func (r *ClassAttendanceSessionParticipantURLListRequest) Validate() error {
	return validate.Struct(r)
}

/* =========================================================
   Response Item & List
========================================================= */

type ClassAttendanceSessionParticipantURLItem struct {
	ID            uuid.UUID  `json:"id"`
	SchoolID      uuid.UUID  `json:"school_id"`
	ParticipantID uuid.UUID  `json:"participant_id"`
	TypeID        *uuid.UUID `json:"type_id,omitempty"`
	Kind          string     `json:"kind"`
	Href          *string    `json:"href,omitempty"`       // map dari *_url
	ObjectKey     *string    `json:"object_key,omitempty"` // map dari *_object_key
	URLOld        *string    `json:"url_old,omitempty"`    // map dari *_url_old
	ObjectKeyOld  *string    `json:"object_key_old,omitempty"`

	Label              *string    `json:"label,omitempty"`
	Order              int        `json:"order"`
	IsPrimary          bool       `json:"is_primary"`
	DeletePendingUntil *time.Time `json:"delete_pending_until,omitempty"`
	UploaderTeacherID  *uuid.UUID `json:"uploader_teacher_id,omitempty"`
	UploaderStudentID  *uuid.UUID `json:"uploader_student_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type ClassAttendanceSessionParticipantURLListResponse struct {
	Items []ClassAttendanceSessionParticipantURLItem `json:"items"`
	Meta  ListMeta                                   `json:"meta"`
}

// Kalau kamu pakai meta custom, boleh dipakai ini;
// kalau nggak, bisa dihapus & pakai ListMeta global dari helpers.
type ListMetaClassAttendanceSessionParticipantURL struct {
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	TotalItems int64 `json:"total_items"`
}

/* =========================================================
   Mapper Model → DTO
========================================================= */

func FromModelClassAttendanceSessionParticipantURL(m model.ClassAttendanceSessionParticipantURLModel) ClassAttendanceSessionParticipantURLItem {
	var deletedAt *time.Time
	if m.ClassAttendanceSessionParticipantURLDeletedAt.Valid {
		t := m.ClassAttendanceSessionParticipantURLDeletedAt.Time
		deletedAt = &t
	}

	return ClassAttendanceSessionParticipantURLItem{
		ID:            m.ClassAttendanceSessionParticipantURLID,
		SchoolID:      m.ClassAttendanceSessionParticipantURLSchoolID,
		ParticipantID: m.ClassAttendanceSessionParticipantURLParticipantID,
		TypeID:        m.ClassAttendanceSessionParticipantTypeID,
		Kind:          m.ClassAttendanceSessionParticipantURLKind,

		// mapping kolom-kolom URL
		Href:         m.ClassAttendanceSessionParticipantURL,
		ObjectKey:    m.ClassAttendanceSessionParticipantURLObjectKey,
		URLOld:       m.ClassAttendanceSessionParticipantURLOld,
		ObjectKeyOld: m.ClassAttendanceSessionParticipantURLObjectKeyOld,

		Label:              m.ClassAttendanceSessionParticipantURLLabel,
		Order:              m.ClassAttendanceSessionParticipantURLOrder,
		IsPrimary:          m.ClassAttendanceSessionParticipantURLIsPrimary,
		DeletePendingUntil: m.ClassAttendanceSessionParticipantURLDeletePendingUntil,
		UploaderTeacherID:  m.ClassAttendanceSessionParticipantURLUploaderTeacherID,
		UploaderStudentID:  m.ClassAttendanceSessionParticipantURLUploaderStudentID,
		CreatedAt:          m.ClassAttendanceSessionParticipantURLCreatedAt,
		UpdatedAt:          m.ClassAttendanceSessionParticipantURLUpdatedAt,
		DeletedAt:          deletedAt,
	}
}

/* =========================================================
   Upsert helper sederhana (opsional)
========================================================= */

type ClassAttendanceSessionParticipantURLUpsert struct {
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

func (u *ClassAttendanceSessionParticipantURLUpsert) Normalize() {
	u.Kind = strings.TrimSpace(strings.ToLower(u.Kind))
	if u.Kind == "" {
		u.Kind = CASPURLKindAttachment
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
