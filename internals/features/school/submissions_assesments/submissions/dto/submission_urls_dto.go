// file: internals/features/assessments/submission_urls/dto/submission_url_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	model "schoolku_backend/internals/features/school/submissions_assesments/submissions/model"
)

var validate = validator.New()

/*
	=========================================================
	  Constants — kinds (sinkron dg model/policy)

=========================================================
*/
const (
	SubmissionURLKindImage      = "image"
	SubmissionURLKindVideo      = "video"
	SubmissionURLKindAttachment = "attachment"
	SubmissionURLKindLink       = "link"
	SubmissionURLKindAudio      = "audio"
)

var allowedKinds = map[string]struct{}{
	SubmissionURLKindImage:      {},
	SubmissionURLKindVideo:      {},
	SubmissionURLKindAttachment: {},
	SubmissionURLKindLink:       {},
	SubmissionURLKindAudio:      {},
}

func normalizeKind(s string) (string, error) {
	k := strings.ToLower(strings.TrimSpace(s))
	if _, ok := allowedKinds[k]; !ok {
		return "", errors.New("invalid kind")
	}
	return k, nil
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

/* =========================================================
   Create
========================================================= */

type CreateSubmissionURLRequest struct {
	SubmissionURLSchoolID     uuid.UUID `json:"submission_url_school_id" validate:"required"`
	SubmissionURLSubmissionID uuid.UUID `json:"submission_url_submission_id" validate:"required"`

	SubmissionURLKind      string  `json:"submission_url_kind" validate:"required,max=24"`
	SubmissionURLHref      *string `json:"submission_url_href" validate:"omitempty,max=4000"`
	SubmissionURLObjectKey *string `json:"submission_url_object_key" validate:"omitempty,max=2000"`

	SubmissionURLLabel     *string `json:"submission_url_label" validate:"omitempty,max=160"`
	SubmissionURLOrder     *int    `json:"submission_url_order"`
	SubmissionURLIsPrimary *bool   `json:"submission_url_is_primary"`
}

func (r *CreateSubmissionURLRequest) Normalize() error {
	// kind
	k, err := normalizeKind(r.SubmissionURLKind)
	if err != nil {
		return err
	}
	r.SubmissionURLKind = k

	// trim-able fields
	r.SubmissionURLLabel = trimPtr(r.SubmissionURLLabel)
	r.SubmissionURLHref = trimPtr(r.SubmissionURLHref)
	r.SubmissionURLObjectKey = trimPtr(r.SubmissionURLObjectKey)

	return nil
}

func (r *CreateSubmissionURLRequest) Validate() error {
	// minimal: href atau object_key harus ada
	if (r.SubmissionURLHref == nil || strings.TrimSpace(*r.SubmissionURLHref) == "") &&
		(r.SubmissionURLObjectKey == nil || strings.TrimSpace(*r.SubmissionURLObjectKey) == "") {
		return errors.New("either submission_url_href or submission_url_object_key must be provided")
	}
	return validate.Struct(r)
}

func (r *CreateSubmissionURLRequest) ToModel() model.SubmissionURLModel {
	order := 0
	if r.SubmissionURLOrder != nil {
		order = *r.SubmissionURLOrder
	}
	isPrimary := false
	if r.SubmissionURLIsPrimary != nil {
		isPrimary = *r.SubmissionURLIsPrimary
	}
	return model.SubmissionURLModel{
		SubmissionURLSchoolID:     r.SubmissionURLSchoolID,
		SubmissionURLSubmissionID: r.SubmissionURLSubmissionID,
		SubmissionURLKind:         r.SubmissionURLKind,

		SubmissionURLHref:      r.SubmissionURLHref,
		SubmissionURLObjectKey: r.SubmissionURLObjectKey,

		SubmissionURLLabel:     r.SubmissionURLLabel,
		SubmissionURLOrder:     order,
		SubmissionURLIsPrimary: isPrimary,
	}
}

/* =========================================================
   Update / Patch
========================================================= */

type UpdateSubmissionURLRequest struct {
	SubmissionURLID uuid.UUID `json:"submission_url_id" validate:"required"`

	SubmissionURLKind         *string `json:"submission_url_kind" validate:"omitempty,max=24"`
	SubmissionURLHref         *string `json:"submission_url_href" validate:"omitempty,max=4000"`
	SubmissionURLObjectKey    *string `json:"submission_url_object_key" validate:"omitempty,max=2000"`
	SubmissionURLObjectKeyOld *string `json:"submission_url_object_key_old" validate:"omitempty,max=2000"`

	SubmissionURLLabel     *string `json:"submission_url_label" validate:"omitempty,max=160"`
	SubmissionURLOrder     *int    `json:"submission_url_order"`
	SubmissionURLIsPrimary *bool   `json:"submission_url_is_primary"`
}

func (r *UpdateSubmissionURLRequest) Normalize() error {
	if r.SubmissionURLKind != nil {
		k, err := normalizeKind(*r.SubmissionURLKind)
		if err != nil {
			return err
		}
		r.SubmissionURLKind = &k
	}
	r.SubmissionURLLabel = trimPtr(r.SubmissionURLLabel)
	r.SubmissionURLHref = trimPtr(r.SubmissionURLHref)
	r.SubmissionURLObjectKey = trimPtr(r.SubmissionURLObjectKey)
	r.SubmissionURLObjectKeyOld = trimPtr(r.SubmissionURLObjectKeyOld)
	return nil
}

func (r *UpdateSubmissionURLRequest) Validate() error {
	return validate.Struct(r)
}

// ToUpdates → map untuk gorm.Updates()
func (r *UpdateSubmissionURLRequest) ToUpdates() (map[string]any, error) {
	if err := r.Normalize(); err != nil {
		return nil, err
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	upd := map[string]any{}
	if r.SubmissionURLKind != nil {
		upd["submission_url_kind"] = *r.SubmissionURLKind
	}
	if r.SubmissionURLHref != nil { // nil means set NULL
		upd["submission_url_href"] = r.SubmissionURLHref
	}
	if r.SubmissionURLObjectKey != nil {
		upd["submission_url_object_key"] = r.SubmissionURLObjectKey
	}
	if r.SubmissionURLObjectKeyOld != nil {
		upd["submission_url_object_key_old"] = r.SubmissionURLObjectKeyOld
	}
	if r.SubmissionURLLabel != nil {
		upd["submission_url_label"] = r.SubmissionURLLabel
	}
	if r.SubmissionURLOrder != nil {
		upd["submission_url_order"] = *r.SubmissionURLOrder
	}
	if r.SubmissionURLIsPrimary != nil {
		upd["submission_url_is_primary"] = *r.SubmissionURLIsPrimary
	}
	return upd, nil
}

/* =========================================================
   List (query params)
========================================================= */

type ListSubmissionURLRequest struct {
	SubmissionURLSchoolID     *uuid.UUID `query:"submission_url_school_id"`
	SubmissionURLSubmissionID *uuid.UUID `query:"submission_url_submission_id"`
	SubmissionURLKind         *string    `query:"submission_url_kind"`
	SubmissionURLIsPrimary    *bool      `query:"submission_url_is_primary"`
	Q                         *string    `query:"q"`

	Limit   int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset  int     `query:"offset" validate:"omitempty,min=0"`
	OrderBy *string `query:"order_by"` // e.g. created_at|updated_at|order
}

func (r *ListSubmissionURLRequest) Normalize() {
	if r.SubmissionURLKind != nil {
		k := strings.TrimSpace(strings.ToLower(*r.SubmissionURLKind))
		r.SubmissionURLKind = &k
	}
	if r.Q != nil {
		q := strings.TrimSpace(*r.Q)
		if q == "" {
			r.Q = nil
		} else {
			r.Q = &q
		}
	}
	if r.Limit <= 0 {
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

/* =========================================================
   Response
========================================================= */

type SubmissionURLItem struct {
	SubmissionURLID                 uuid.UUID  `json:"submission_url_id"`
	SubmissionURLSchoolID           uuid.UUID  `json:"submission_url_school_id"`
	SubmissionURLSubmissionID       uuid.UUID  `json:"submission_url_submission_id"`
	SubmissionURLKind               string     `json:"submission_url_kind"`
	SubmissionURLHref               *string    `json:"submission_url_href,omitempty"`
	SubmissionURLObjectKey          *string    `json:"submission_url_object_key,omitempty"`
	SubmissionURLObjectKeyOld       *string    `json:"submission_url_object_key_old,omitempty"`
	SubmissionURLLabel              *string    `json:"submission_url_label,omitempty"`
	SubmissionURLOrder              int        `json:"submission_url_order"`
	SubmissionURLIsPrimary          bool       `json:"submission_url_is_primary"`
	SubmissionURLCreatedAt          time.Time  `json:"submission_url_created_at"`
	SubmissionURLUpdatedAt          time.Time  `json:"submission_url_updated_at"`
	SubmissionURLDeletedAt          *time.Time `json:"submission_url_deleted_at,omitempty"`
	SubmissionURLDeletePendingUntil *time.Time `json:"submission_url_delete_pending_until,omitempty"`
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

/* =========================================================
   Mapper dari Model → DTO
========================================================= */

func FromModelsSubmissionURL(m model.SubmissionURLModel) SubmissionURLItem {
	var deletedAt *time.Time
	if m.SubmissionURLDeletedAt.Valid {
		t := m.SubmissionURLDeletedAt.Time
		deletedAt = &t
	}
	return SubmissionURLItem{
		SubmissionURLID:                 m.SubmissionURLID,
		SubmissionURLSchoolID:           m.SubmissionURLSchoolID,
		SubmissionURLSubmissionID:       m.SubmissionURLSubmissionID,
		SubmissionURLKind:               m.SubmissionURLKind,
		SubmissionURLHref:               m.SubmissionURLHref,
		SubmissionURLObjectKey:          m.SubmissionURLObjectKey,
		SubmissionURLObjectKeyOld:       m.SubmissionURLObjectKeyOld,
		SubmissionURLLabel:              m.SubmissionURLLabel,
		SubmissionURLOrder:              m.SubmissionURLOrder,
		SubmissionURLIsPrimary:          m.SubmissionURLIsPrimary,
		SubmissionURLCreatedAt:          m.SubmissionURLCreatedAt,
		SubmissionURLUpdatedAt:          m.SubmissionURLUpdatedAt,
		SubmissionURLDeletedAt:          deletedAt,
		SubmissionURLDeletePendingUntil: m.SubmissionURLDeletePendingUntil,
	}
}

func FromModelsSubmissionsURL(list []model.SubmissionURLModel) []SubmissionURLItem {
	out := make([]SubmissionURLItem, 0, len(list))
	for i := range list {
		out = append(out, FromModelsSubmissionURL(list[i]))
	}
	return out
}
