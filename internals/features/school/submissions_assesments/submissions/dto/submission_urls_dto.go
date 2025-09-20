// file: internals/features/submissions/submission_urls/dto/submission_url_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

/*
	=========================================================
	  Constants — kinds (sinkron dg model)

=========================================================
*/
const (
	SubURLKindImage      = "image"
	SubURLKindVideo      = "video"
	SubURLKindAttachment = "attachment"
	SubURLKindLink       = "link"
	SubURLKindAudio      = "audio"
)

/*
	=========================================================
	  Create

=========================================================
*/
type CreateSubmissionURLRequest struct {
	SubmissionURLMasjidID     uuid.UUID `json:"masjid_id" validate:"required"`
	SubmissionURLSubmissionID uuid.UUID `json:"submission_id" validate:"required"`

	Kind      string  `json:"kind" validate:"required,max=24"`
	Href      *string `json:"href" validate:"omitempty,max=4000"`
	ObjectKey *string `json:"object_key" validate:"omitempty,max=2000"`

	Label     *string `json:"label" validate:"omitempty,max=160"`
	Order     *int32  `json:"order"`
	IsPrimary *bool   `json:"is_primary"`
}

func (r *CreateSubmissionURLRequest) Normalize() {
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

func (r *CreateSubmissionURLRequest) Validate() error {
	if (r.Href == nil || strings.TrimSpace(*r.Href) == "") &&
		(r.ObjectKey == nil || strings.TrimSpace(*r.ObjectKey) == "") {
		return errors.New("either href or object_key must be provided")
	}
	return validate.Struct(r)
}

/*
	=========================================================
	  Update / Patch

=========================================================
*/
type UpdateSubmissionURLRequest struct {
	ID        uuid.UUID `json:"-" validate:"required"`
	Kind      *string   `json:"kind" validate:"omitempty,max=24"`
	Href      *string   `json:"href" validate:"omitempty,max=4000"`
	ObjectKey *string   `json:"object_key" validate:"omitempty,max=2000"`

	Label     *string `json:"label" validate:"omitempty,max=160"`
	Order     *int32  `json:"order"`
	IsPrimary *bool   `json:"is_primary"`
}

func (r *UpdateSubmissionURLRequest) Normalize() {
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

func (r *UpdateSubmissionURLRequest) Validate() error {
	return validate.Struct(r)
}

/*
	=========================================================
	  List (query params)

=========================================================
*/
type ListSubmissionURLRequest struct {
	MasjidID     *uuid.UUID `query:"masjid_id"`
	SubmissionID *uuid.UUID `query:"submission_id"`
	Kind         *string    `query:"kind"`
	IsPrimary    *bool      `query:"is_primary"`
	Q            *string    `query:"q"`

	Limit   int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset  int     `query:"offset" validate:"omitempty,min=0"`
	OrderBy *string `query:"order_by"`
}

func (r *ListSubmissionURLRequest) Normalize() {
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

func (r *ListSubmissionURLRequest) Validate() error {
	return validate.Struct(r)
}

/*
	=========================================================
	  Response

=========================================================
*/
type SubmissionURLItem struct {
	ID           uuid.UUID  `json:"id"`
	MasjidID     uuid.UUID  `json:"masjid_id"`
	SubmissionID uuid.UUID  `json:"submission_id"`
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

type ListSubmissionURLResponse struct {
	Items []SubmissionURLItem `json:"items"`
	Meta  ListMeta            `json:"meta"`
}

type ListMeta struct {
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	TotalItems int64 `json:"total_items"`
}

/*
	=========================================================
	  Mapper dari Model → DTO

=========================================================
*/
type ModelSubmissionURL interface {
	GetID() uuid.UUID
	GetMasjidID() uuid.UUID
	GetSubmissionID() uuid.UUID
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

func FromBookURLModels(m ModelSubmissionURL) SubmissionURLItem {
	return SubmissionURLItem{
		ID:           m.GetID(),
		MasjidID:     m.GetMasjidID(),
		SubmissionID: m.GetSubmissionID(),
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
