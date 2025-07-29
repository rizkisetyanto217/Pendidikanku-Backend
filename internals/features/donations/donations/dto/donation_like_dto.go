// dtos/donation_like_dto.go

package dto

import (
	"time"

	"github.com/google/uuid"
)

// ✅ Request DTO: Digunakan saat user klik like/unlike
type CreateOrToggleDonationLikeDTO struct {
	DonationLikeDonationID uuid.UUID `json:"donation_like_donation_id" validate:"required"`
}

// ✅ Response DTO: Digunakan saat mengembalikan status like
type DonationLikeResponseDTO struct {
	DonationLikeID          uuid.UUID  `json:"donation_like_id"`
	DonationLikeIsLiked     bool       `json:"donation_like_is_liked"`
	DonationLikeDonationID  uuid.UUID  `json:"donation_like_donation_id"`
	DonationLikeUserID      uuid.UUID  `json:"donation_like_user_id"`
	DonationLikeMasjidID    *uuid.UUID `json:"donation_like_masjid_id,omitempty"`
	DonationLikeUpdatedAt   time.Time  `json:"donation_like_updated_at"`
}
