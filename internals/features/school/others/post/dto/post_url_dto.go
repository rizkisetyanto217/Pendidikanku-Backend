// file: internals/features/announcements/urls/dto/announcement_url_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/school/others/post/model"

	"github.com/google/uuid"
)

/* =========================
         RESPONSE
========================= */

type AnnouncementURLResponse struct {
	AnnouncementURLId             uuid.UUID `json:"announcement_url_id"`
	AnnouncementURLMasjidId       uuid.UUID `json:"announcement_url_masjid_id"`
	AnnouncementURLAnnouncementId uuid.UUID `json:"announcement_url_announcement_id"`

	AnnouncementURLKind string `json:"announcement_url_kind"`

	AnnouncementURLHref         *string `json:"announcement_url_href,omitempty"`
	AnnouncementURLObjectKey    *string `json:"announcement_url_object_key,omitempty"`
	AnnouncementURLObjectKeyOld *string `json:"announcement_url_object_key_old,omitempty"`

	AnnouncementURLLabel     *string `json:"announcement_url_label,omitempty"`
	AnnouncementURLOrder     int     `json:"announcement_url_order"`
	AnnouncementURLIsPrimary bool    `json:"announcement_url_is_primary"`

	AnnouncementURLCreatedAt          time.Time  `json:"announcement_url_created_at"`
	AnnouncementURLUpdatedAt          time.Time  `json:"announcement_url_updated_at"`
	AnnouncementURLDeletedAt          *time.Time `json:"announcement_url_deleted_at,omitempty"`
	AnnouncementURLDeletePendingUntil *time.Time `json:"announcement_url_delete_pending_until,omitempty"`
}

func FromAnnouncementURLModel(m model.AnnouncementURLModel) AnnouncementURLResponse {
	return AnnouncementURLResponse{
		AnnouncementURLId:                 m.AnnouncementURLId,
		AnnouncementURLMasjidId:           m.AnnouncementURLMasjidId,
		AnnouncementURLAnnouncementId:     m.AnnouncementURLAnnouncementId,
		AnnouncementURLKind:               m.AnnouncementURLKind,
		AnnouncementURLHref:               m.AnnouncementURLHref,
		AnnouncementURLObjectKey:          m.AnnouncementURLObjectKey,
		AnnouncementURLObjectKeyOld:       m.AnnouncementURLObjectKeyOld,
		AnnouncementURLLabel:              m.AnnouncementURLLabel,
		AnnouncementURLOrder:              m.AnnouncementURLOrder,
		AnnouncementURLIsPrimary:          m.AnnouncementURLIsPrimary,
		AnnouncementURLCreatedAt:          m.AnnouncementURLCreatedAt,
		AnnouncementURLUpdatedAt:          m.AnnouncementURLUpdatedAt,
		AnnouncementURLDeletedAt:          m.AnnouncementURLDeletedAt,
		AnnouncementURLDeletePendingUntil: m.AnnouncementURLDeletePendingUntil,
	}
}

func FromAnnouncementURLModels(rows []model.AnnouncementURLModel) []AnnouncementURLResponse {
	out := make([]AnnouncementURLResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, FromAnnouncementURLModel(m))
	}
	return out
}

/* =========================
         CREATE
========================= */

type CreateAnnouncementURLRequest struct {
	AnnouncementURLAnnouncementId uuid.UUID `json:"announcement_url_announcement_id" validate:"required"`
	AnnouncementURLKind           string    `json:"announcement_url_kind" validate:"required,max=24"`

	AnnouncementURLHref      *string `json:"announcement_url_href"`
	AnnouncementURLObjectKey *string `json:"announcement_url_object_key"`

	AnnouncementURLLabel     *string `json:"announcement_url_label" validate:"omitempty,max=160"`
	AnnouncementURLOrder     int     `json:"announcement_url_order"`
	AnnouncementURLIsPrimary bool    `json:"announcement_url_is_primary"`
}

func (r *CreateAnnouncementURLRequest) ToModel(masjidID uuid.UUID) model.AnnouncementURLModel {
	return model.AnnouncementURLModel{
		AnnouncementURLMasjidId:       masjidID,
		AnnouncementURLAnnouncementId: r.AnnouncementURLAnnouncementId,
		AnnouncementURLKind:           r.AnnouncementURLKind,
		AnnouncementURLHref:           r.AnnouncementURLHref,
		AnnouncementURLObjectKey:      r.AnnouncementURLObjectKey,
		AnnouncementURLLabel:          r.AnnouncementURLLabel,
		AnnouncementURLOrder:          r.AnnouncementURLOrder,
		AnnouncementURLIsPrimary:      r.AnnouncementURLIsPrimary,
	}
}

/* =========================
      PATCH / PARTIAL UPDATE
   - Semua pointer → optional
   - Termasuk opsi ganti kind / relink announcement
========================= */

type UpdateAnnouncementURLRequest struct {
	AnnouncementURLAnnouncementId *uuid.UUID `json:"announcement_url_announcement_id"` // optional relink
	AnnouncementURLKind           *string    `json:"announcement_url_kind" validate:"omitempty,max=24"`

	AnnouncementURLHref         *string `json:"announcement_url_href"`
	AnnouncementURLObjectKey    *string `json:"announcement_url_object_key"`     // kalau diisi & ingin retensi, controller bisa set object_key_old + delete_pending_until
	AnnouncementURLObjectKeyOld *string `json:"announcement_url_object_key_old"` // biasanya DISET oleh sistem, bukan input user

	AnnouncementURLLabel     *string `json:"announcement_url_label" validate:"omitempty,max=160"`
	AnnouncementURLOrder     *int    `json:"announcement_url_order"`
	AnnouncementURLIsPrimary *bool   `json:"announcement_url_is_primary"`

	AnnouncementURLDeletePendingUntil *time.Time `json:"announcement_url_delete_pending_until"` // opsional; biasanya sistem yang atur
}

/* =========================
     BULK REORDER (opsi)
========================= */

type ReorderAnnouncementURLsRequest struct {
	Items []struct {
		AnnouncementURLId uuid.UUID `json:"announcement_url_id" validate:"required"`
		Order             int       `json:"order"`
	} `json:"items" validate:"required,dive"`
}

/* =========================
     SET PRIMARY (opsi)
========================= */

type SetPrimaryAnnouncementURLRequest struct {
	AnnouncementURLId uuid.UUID `json:"announcement_url_id" validate:"required"`
	// bisa ditambah guard kind kalau diperlukan:
	// Kind string `json:"kind" validate:"omitempty"`
}

/* =========================
       LIST / FILTER
========================= */

type ListAnnouncementURLQuery struct {
	// Scope wajib di controller: masjid_id dari context
	AnnouncementId *uuid.UUID `query:"announcement_id"`
	Kind           *string    `query:"kind"`
	IsPrimary      *bool      `query:"is_primary"`

	Label     *string `query:"label"`      // exact
	LabelLike *string `query:"label_like"` // ILIKE %...%

	IncludeDeleted *bool `query:"include_deleted"` // default false

	// Sorting & paging → gunakan helper.ParseFiber/ParseWith untuk validasi & fallback
	SortBy *string `query:"sort_by"` // allowed: created_at, order, is_primary
	Sort   *string `query:"sort"`    // asc|desc
	Limit  *int    `query:"limit"`
	Offset *int    `query:"offset"`
}
