package dto

import (
	"time"

	"github.com/google/uuid"

	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
)

// =========================
// Request (Create / Upsert)
// =========================

// pakai pointer di is_active biar optional; default = true
type MasjidAdminRequest struct {
	MasjidAdminsMasjidID uuid.UUID `json:"masjid_admins_masjid_id"` // bisa dioverride dari path/token
	MasjidAdminsUserID   uuid.UUID `json:"masjid_admins_user_id"`
	MasjidAdminsIsActive *bool     `json:"masjid_admins_is_active,omitempty"`
}

// =========================
// Response
// =========================

type MasjidAdminResponse struct {
	MasjidAdminsID        uuid.UUID `json:"masjid_admins_id"`
	MasjidAdminsMasjidID  uuid.UUID `json:"masjid_admins_masjid_id"`
	MasjidAdminsUserID    uuid.UUID `json:"masjid_admins_user_id"`
	MasjidAdminsIsActive  bool      `json:"masjid_admins_is_active"`
	MasjidAdminCreatedAt  time.Time `json:"masjid_admin_created_at"`
	MasjidAdminUpdatedAt  time.Time `json:"masjid_admin_updated_at"`
	// kalau mau expose soft delete timestamp, buka ini:
	// MasjidAdminDeletedAt *time.Time `json:"masjid_admin_deleted_at,omitempty"`
}

// =========================
// Converters
// =========================

// DTO -> Model (untuk CREATE)
// Catatan: controller boleh override MasjidAdminsMasjidID dari path/token
func (r *MasjidAdminRequest) ToModelCreate() *model.MasjidAdminModel {
	if r == nil {
		return nil
	}
	active := true
	if r.MasjidAdminsIsActive != nil {
		active = *r.MasjidAdminsIsActive
	}
	return &model.MasjidAdminModel{
		MasjidAdminsMasjidID: r.MasjidAdminsMasjidID,
		MasjidAdminsUserID:   r.MasjidAdminsUserID,
		MasjidAdminsIsActive: active,
	}
}

// Model -> DTO (untuk response GET/POST/PUT)
func ToMasjidAdminResponse(m *model.MasjidAdminModel) MasjidAdminResponse {
	return MasjidAdminResponse{
		MasjidAdminsID:       m.MasjidAdminsID,
		MasjidAdminsMasjidID: m.MasjidAdminsMasjidID,
		MasjidAdminsUserID:   m.MasjidAdminsUserID,
		MasjidAdminsIsActive: m.MasjidAdminsIsActive,

		MasjidAdminCreatedAt: m.MasjidAdminCreatedAt,
		MasjidAdminUpdatedAt: m.MasjidAdminUpdatedAt,
	}
}
