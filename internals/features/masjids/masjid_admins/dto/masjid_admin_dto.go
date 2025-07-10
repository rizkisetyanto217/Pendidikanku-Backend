package dto

import (
	"masjidku_backend/internals/features/masjids/masjid_admins/model"
	"time"

	"github.com/google/uuid"
)

type MasjidAdminRequest struct {
	MasjidAdminMasjidID uuid.UUID `json:"masjid_admins_masjid_id"`
	MasjidAdminsUserID  uuid.UUID `json:"masjid_admins_user_id"`
}

type MasjidAdminResponse struct {
	MasjidAdminID uuid.UUID `json:"masjid_admin_id"`
	MasjidID      uuid.UUID `json:"masjid_id"`
	UserID        uuid.UUID `json:"user_id"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

func ToMasjidAdminResponse(data model.MasjidAdminModel) MasjidAdminResponse {
	return MasjidAdminResponse{
		MasjidAdminID: data.MasjidAdminID,
		MasjidID:      data.MasjidID,
		UserID:        data.UserID,
		IsActive:      data.IsActive,
		CreatedAt:     data.CreatedAt,
	}
}
