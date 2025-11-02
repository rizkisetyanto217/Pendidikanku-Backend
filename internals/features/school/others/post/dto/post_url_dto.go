// file: internals/features/social/posts/dto/post_url_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

/* =========================================================
   Helpers
========================================================= */

func (f UpdateField[T]) Value() T { return f.value }

/* =========================================================
   DTO: Create
========================================================= */

type PostURLCreateDTO struct {
	PostURLKind      string  `json:"post_url_kind" validate:"required,max=24"`
	PostURLHref      *string `json:"post_url_href,omitempty"`
	PostURLObjectKey *string `json:"post_url_object_key,omitempty"`
	PostURLLabel     *string `json:"post_url_label,omitempty" validate:"omitempty,max=160"`
	PostURLOrder     *int    `json:"post_url_order,omitempty"`
	PostURLIsPrimary *bool   `json:"post_url_is_primary,omitempty"`
}

/* =========================================================
   DTO: Update (PATCH)
========================================================= */

type PostURLUpdateDTO struct {
	PostURLKind      UpdateField[string]  `json:"post_url_kind,omitempty"`
	PostURLHref      UpdateField[*string] `json:"post_url_href,omitempty"`
	PostURLObjectKey UpdateField[*string] `json:"post_url_object_key,omitempty"`
	PostURLLabel     UpdateField[*string] `json:"post_url_label,omitempty"`
	PostURLOrder     UpdateField[int]     `json:"post_url_order,omitempty"`
	PostURLIsPrimary UpdateField[bool]    `json:"post_url_is_primary,omitempty"`
}

/* =========================================================
   DTO: Response
========================================================= */

type PostURLResponseDTO struct {
	PostURLID              uuid.UUID  `json:"post_url_id"`
	PostURLSchoolID        uuid.UUID  `json:"post_url_school_id"`
	PostURLPostID          uuid.UUID  `json:"post_url_post_id"`
	PostURLKind            string     `json:"post_url_kind"`
	PostURLHref            *string    `json:"post_url_href,omitempty"`
	PostURLObjectKey       *string    `json:"post_url_object_key,omitempty"`
	PostURLObjectKeyOld    *string    `json:"post_url_object_key_old,omitempty"`
	PostURLLabel           *string    `json:"post_url_label,omitempty"`
	PostURLOrder           int        `json:"post_url_order"`
	PostURLIsPrimary       bool       `json:"post_url_is_primary"`
	PostURLCreatedAt       time.Time  `json:"post_url_created_at"`
	PostURLUpdatedAt       time.Time  `json:"post_url_updated_at"`
	PostURLDeletedAt       *time.Time `json:"post_url_deleted_at,omitempty"`
	PostURLDeletePendingAt *time.Time `json:"post_url_delete_pending_until,omitempty"`
}
