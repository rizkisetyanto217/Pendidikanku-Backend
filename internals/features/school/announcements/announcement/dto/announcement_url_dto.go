package dto

import (
	"time"

	"github.com/google/uuid"
)

/* =========================
   CREATE REQUEST
   ========================= */
type CreateAnnouncementURLRequest struct {
	AnnouncementURLLabel          *string   `json:"announcement_url_label,omitempty" validate:"omitempty,max=120"`
	AnnouncementURLHref           string    `json:"announcement_url_href" validate:"required,url"`
	AnnouncementURLAnnouncementID uuid.UUID `json:"announcement_url_announcement_id" validate:"required,uuid"`
}

/* =========================
   UPDATE REQUEST
   ========================= */
type UpdateAnnouncementURLRequest struct {
	AnnouncementURLLabel            *string    `json:"announcement_url_label,omitempty" validate:"omitempty,max=120"`
	AnnouncementURLHref             *string    `json:"announcement_url_href,omitempty" validate:"omitempty,url"`
	AnnouncementURLTrashURL         *string    `json:"announcement_url_trash_url,omitempty"`
	AnnouncementURLDeletePendingUntil *time.Time `json:"announcement_url_delete_pending_until,omitempty"`
}

/* =========================
   RESPONSE
   ========================= */
type AnnouncementURLResponse struct {
	AnnouncementURLID               uuid.UUID  `json:"announcement_url_id"`
	AnnouncementURLMasjidID         uuid.UUID  `json:"announcement_url_masjid_id"`
	AnnouncementURLAnnouncementID   uuid.UUID  `json:"announcement_url_announcement_id"`

	AnnouncementURLLabel            *string    `json:"announcement_url_label,omitempty"`
	AnnouncementURLHref             string     `json:"announcement_url_href"`
	AnnouncementURLTrashURL         *string    `json:"announcement_url_trash_url,omitempty"`
	AnnouncementURLDeletePendingUntil *time.Time `json:"announcement_url_delete_pending_until,omitempty"`

	AnnouncementURLCreatedAt        time.Time  `json:"announcement_url_created_at"`
	AnnouncementURLUpdatedAt        time.Time  `json:"announcement_url_updated_at"`
	AnnouncementURLDeletedAt        *time.Time `json:"announcement_url_deleted_at,omitempty"`
}
