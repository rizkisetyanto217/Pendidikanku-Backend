// file: internals/features/assessments/assessment_urls/dto/assessment_url_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

/*
=========================================================

	Validator

=========================================================
*/
var validate = validator.New()

/*
=========================================================

	Constants — kinds (sinkron dg model)

=========================================================
*/
const (
	AssURLKindImage      = "image"
	AssURLKindVideo      = "video"
	AssURLKindAttachment = "attachment"
	AssURLKindLink       = "link"
	AssURLKindAudio      = "audio"
)

/*
=========================================================

	Create

=========================================================
*/
type CreateAssessmentURLRequest struct {
	AssessmentURLSchoolID   uuid.UUID `json:"school_id" validate:"required"`
	AssessmentURLAssessment uuid.UUID `json:"assessment_id" validate:"required"`

	Kind      string  `json:"kind" validate:"required,max=24"`
	Href      *string `json:"href" validate:"omitempty,max=4000"`
	ObjectKey *string `json:"object_key" validate:"omitempty,max=2000"`

	Label     *string `json:"label" validate:"omitempty,max=160"`
	Order     *int32  `json:"order"`
	IsPrimary *bool   `json:"is_primary"`
}

func (r *CreateAssessmentURLRequest) Normalize() {
	r.Kind = strings.TrimSpace(strings.ToLower(r.Kind))
	if r.Label != nil {
		lbl := strings.TrimSpace(*r.Label)
		r.Label = &lbl
	}
	if r.Href != nil {
		h := strings.TrimSpace(*r.Href)
		if h == "" {
			r.Href = nil
		} else {
			r.Href = &h
		}
	}
	if r.ObjectKey != nil {
		ok := strings.TrimSpace(*r.ObjectKey)
		if ok == "" {
			r.ObjectKey = nil
		} else {
			r.ObjectKey = &ok
		}
	}
}

func (r *CreateAssessmentURLRequest) Validate() error {
	// minimal: butuh salah satu dari href/object_key
	if (r.Href == nil || strings.TrimSpace(*r.Href) == "") &&
		(r.ObjectKey == nil || strings.TrimSpace(*r.ObjectKey) == "") {
		return errors.New("either href or object_key must be provided")
	}
	return validate.Struct(r)
}

/*
=========================================================

	Update / Patch (single row)

=========================================================
*/
type UpdateAssessmentURLRequest struct {
	ID        uuid.UUID `json:"-" validate:"required"`
	Kind      *string   `json:"kind" validate:"omitempty,max=24"`
	Href      *string   `json:"href" validate:"omitempty,max=4000"`
	ObjectKey *string   `json:"object_key" validate:"omitempty,max=2000"`

	Label     *string `json:"label" validate:"omitempty,max=160"`
	Order     *int32  `json:"order"`
	IsPrimary *bool   `json:"is_primary"`
}

func (r *UpdateAssessmentURLRequest) Normalize() {
	if r.Kind != nil {
		k := strings.TrimSpace(strings.ToLower(*r.Kind))
		r.Kind = &k
	}
	if r.Label != nil {
		lbl := strings.TrimSpace(*r.Label)
		r.Label = &lbl
	}
	if r.Href != nil {
		h := strings.TrimSpace(*r.Href)
		if h == "" {
			r.Href = nil
		} else {
			r.Href = &h
		}
	}
	if r.ObjectKey != nil {
		ok := strings.TrimSpace(*r.ObjectKey)
		if ok == "" {
			r.ObjectKey = nil
		} else {
			r.ObjectKey = &ok
		}
	}
}

func (r *UpdateAssessmentURLRequest) Validate() error {
	return validate.Struct(r)
}

/*
=========================================================

	Upsert (untuk JSON/multipart batch create/update)
	- ID opsional: jika ada → update, jika kosong → insert
	- ReplaceFile opsional: dipakai controller multipart

=========================================================
*/
type AssessmentURLUpsert struct {
	ID        *uuid.UUID `json:"id"` // optional for update
	Kind      string     `json:"kind" validate:"omitempty,max=24"`
	Label     *string    `json:"label" validate:"omitempty,max=160"`
	Href      *string    `json:"href" validate:"omitempty,max=4000"`
	ObjectKey *string    `json:"object_key" validate:"omitempty,max=2000"`
	Order     *int32     `json:"order"`
	IsPrimary *bool      `json:"is_primary"`
	// optional flag for multipart controller to trigger upload/replace
	ReplaceFile bool `json:"replace_file"`
}

func (u *AssessmentURLUpsert) Normalize() {
	u.Kind = strings.TrimSpace(strings.ToLower(u.Kind))
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

/*
=========================================================

	List (query params)

=========================================================
*/
type ListAssessmentURLRequest struct {
	SchoolID     *uuid.UUID `query:"school_id"`
	AssessmentID *uuid.UUID `query:"assessment_id"`
	Kind         *string    `query:"kind"`
	IsPrimary    *bool      `query:"is_primary"`
	Q            *string    `query:"q"` // search in label (ILIKE %q%)

	Limit   int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset  int     `query:"offset" validate:"omitempty,min=0"`
	OrderBy *string `query:"order_by"` // e.g. "is_primary desc, order asc"
}

func (r *ListAssessmentURLRequest) Normalize() {
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
			r.OrderBy = &ob
		}
	}
}

func (r *ListAssessmentURLRequest) Validate() error {
	return validate.Struct(r)
}

/*
=========================================================

	Response

=========================================================
*/
type AssessmentURLItem struct {
	ID           uuid.UUID  `json:"id"`
	SchoolID     uuid.UUID  `json:"school_id"`
	AssessmentID uuid.UUID  `json:"assessment_id"`
	Kind         string     `json:"kind"`
	Href         *string    `json:"href,omitempty"`
	ObjectKey    *string    `json:"object_key,omitempty"`
	ObjectKeyOld *string    `json:"object_key_old,omitempty"`
	Label        *string    `json:"label,omitempty"`
	Order        int32      `json:"order"`
	IsPrimary    bool       `json:"is_primary"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type ListAssessmentURLResponse struct {
	Items []AssessmentURLItem `json:"items"`
	Meta  ListMeta            `json:"meta"`
}

type ListMeta struct {
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	TotalItems int64 `json:"total_items"`
}

/*
=========================================================

	Mapper Model → DTO (pakai interface agar layer tidak saling import)

=========================================================
*/
type ModelAssessmentURL interface {
	GetID() uuid.UUID
	GetSchoolID() uuid.UUID
	GetAssessmentID() uuid.UUID
	GetKind() string
	GetHref() *string
	GetObjectKey() *string
	GetObjectKeyOld() *string
	GetLabel() *string
	GetOrder() int32
	GetIsPrimary() bool
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetDeletedAtPtr() *time.Time
}

func FromAssessmentURLModel(m ModelAssessmentURL) AssessmentURLItem {
	return AssessmentURLItem{
		ID:           m.GetID(),
		SchoolID:     m.GetSchoolID(),
		AssessmentID: m.GetAssessmentID(),
		Kind:         m.GetKind(),
		Href:         m.GetHref(),
		ObjectKey:    m.GetObjectKey(),
		ObjectKeyOld: m.GetObjectKeyOld(),
		Label:        m.GetLabel(),
		Order:        m.GetOrder(),
		IsPrimary:    m.GetIsPrimary(),
		CreatedAt:    m.GetCreatedAt(),
		UpdatedAt:    m.GetUpdatedAt(),
		DeletedAt:    m.GetDeletedAtPtr(),
	}
}
