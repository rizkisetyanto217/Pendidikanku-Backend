// file: internals/features/library/books/dto/book_url_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/academics/books/model"
)

/* =========================================
   REQUEST DTOs
   ========================================= */

type CreateBookURLRequest struct {
	BookURLMasjidID uuid.UUID `json:"book_url_masjid_id" validate:"required"`
	BookURLBookID   uuid.UUID `json:"book_url_book_id"   validate:"required"`

	BookURLKind string `json:"book_url_kind" validate:"required,max=24"`

	BookURLHref         *string `json:"book_url_href,omitempty"        validate:"omitempty,url"`
	BookURLObjectKey    *string `json:"book_url_object_key,omitempty"  validate:"omitempty"`
	BookURLObjectKeyOld *string `json:"book_url_object_key_old,omitempty" validate:"omitempty"`

	BookURLLabel     *string `json:"book_url_label,omitempty"   validate:"omitempty,max=160"`
	BookURLOrder     *int    `json:"book_url_order,omitempty"   validate:"omitempty"`
	BookURLIsPrimary *bool   `json:"book_url_is_primary,omitempty" validate:"omitempty"`
}

type PatchBookURLRequest struct {
	// Semua opsional; dipakai untuk PATCH parsial
	BookURLMasjidID *uuid.UUID `json:"book_url_masjid_id,omitempty" validate:"omitempty"`
	BookURLBookID   *uuid.UUID `json:"book_url_book_id,omitempty"   validate:"omitempty"`

	BookURLKind *string `json:"book_url_kind,omitempty" validate:"omitempty,max=24"`

	BookURLHref         *string `json:"book_url_href,omitempty"         validate:"omitempty,url"`
	BookURLObjectKey    *string `json:"book_url_object_key,omitempty"   validate:"omitempty"`
	BookURLObjectKeyOld *string `json:"book_url_object_key_old,omitempty" validate:"omitempty"`

	BookURLLabel     *string `json:"book_url_label,omitempty"   validate:"omitempty,max=160"`
	BookURLOrder     *int    `json:"book_url_order,omitempty"   validate:"omitempty"`
	BookURLIsPrimary *bool   `json:"book_url_is_primary,omitempty" validate:"omitempty"`
}

/* =========================================
   RESPONSE DTO
   ========================================= */

type BookURLResponse struct {
	BookURLID uuid.UUID `json:"book_url_id"`

	BookURLMasjidID uuid.UUID `json:"book_url_masjid_id"`
	BookURLBookID   uuid.UUID `json:"book_url_book_id"`

	BookURLKind string  `json:"book_url_kind"`
	BookURLHref *string `json:"book_url_href,omitempty"`

	BookURLObjectKey    *string `json:"book_url_object_key,omitempty"`
	BookURLObjectKeyOld *string `json:"book_url_object_key_old,omitempty"`

	BookURLLabel     *string `json:"book_url_label,omitempty"`
	BookURLOrder     int     `json:"book_url_order"`
	BookURLIsPrimary bool    `json:"book_url_is_primary"`

	BookURLCreatedAt          time.Time  `json:"book_url_created_at"`
	BookURLUpdatedAt          time.Time  `json:"book_url_updated_at"`
	BookURLDeletedAt          *time.Time `json:"book_url_deleted_at,omitempty"`
	BookURLDeletePendingUntil *time.Time `json:"book_url_delete_pending_until,omitempty"`
}

/* =========================================
   MAPPERS
   ========================================= */

func (r CreateBookURLRequest) ToModel() *model.BookURLModel {
	m := &model.BookURLModel{
		BookURLMasjidID: r.BookURLMasjidID,
		BookURLBookID:   r.BookURLBookID,

		BookURLKind: strings.TrimSpace(r.BookURLKind),

		BookURLHref:         trimPtr(r.BookURLHref),
		BookURLObjectKey:    trimPtr(r.BookURLObjectKey),
		BookURLObjectKeyOld: trimPtr(r.BookURLObjectKeyOld),

		BookURLLabel: trimPtr(r.BookURLLabel),
	}
	if r.BookURLOrder != nil {
		m.BookURLOrder = *r.BookURLOrder
	}
	if r.BookURLIsPrimary != nil {
		m.BookURLIsPrimary = *r.BookURLIsPrimary
	}
	return m
}

func (p PatchBookURLRequest) Apply(dst *model.BookURLModel) {
	if p.BookURLMasjidID != nil {
		dst.BookURLMasjidID = *p.BookURLMasjidID
	}
	if p.BookURLBookID != nil {
		dst.BookURLBookID = *p.BookURLBookID
	}
	if p.BookURLKind != nil {
		dst.BookURLKind = strings.TrimSpace(*p.BookURLKind)
	}
	if p.BookURLHref != nil {
		dst.BookURLHref = trimPtr(p.BookURLHref)
	}
	if p.BookURLObjectKey != nil {
		dst.BookURLObjectKey = trimPtr(p.BookURLObjectKey)
	}
	if p.BookURLObjectKeyOld != nil {
		dst.BookURLObjectKeyOld = trimPtr(p.BookURLObjectKeyOld)
	}
	if p.BookURLLabel != nil {
		dst.BookURLLabel = trimPtr(p.BookURLLabel)
	}
	if p.BookURLOrder != nil {
		dst.BookURLOrder = *p.BookURLOrder
	}
	if p.BookURLIsPrimary != nil {
		dst.BookURLIsPrimary = *p.BookURLIsPrimary
	}
}

func FromBookURLModel(m *model.BookURLModel) BookURLResponse {
	var deletedAt *time.Time
	if m.BookURLDeletedAt.Valid {
		deletedAt = &m.BookURLDeletedAt.Time
	}
	var purgeAt *time.Time
	if m.BookURLDeletePendingUntil.Valid {
		purgeAt = &m.BookURLDeletePendingUntil.Time
	}

	return BookURLResponse{
		BookURLID: m.BookURLID,

		BookURLMasjidID: m.BookURLMasjidID,
		BookURLBookID:   m.BookURLBookID,

		BookURLKind: m.BookURLKind,
		BookURLHref: m.BookURLHref,

		BookURLObjectKey:    m.BookURLObjectKey,
		BookURLObjectKeyOld: m.BookURLObjectKeyOld,

		BookURLLabel:     m.BookURLLabel,
		BookURLOrder:     m.BookURLOrder,
		BookURLIsPrimary: m.BookURLIsPrimary,

		BookURLCreatedAt:          m.BookURLCreatedAt,
		BookURLUpdatedAt:          m.BookURLUpdatedAt,
		BookURLDeletedAt:          deletedAt,
		BookURLDeletePendingUntil: purgeAt,
	}
}

func FromBookURLModels(list []model.BookURLModel) []BookURLResponse {
	out := make([]BookURLResponse, 0, len(list))
	for i := range list {
		// capture pointer for current element
		m := list[i]
		out = append(out, FromBookURLModel(&m))
	}
	return out
}
