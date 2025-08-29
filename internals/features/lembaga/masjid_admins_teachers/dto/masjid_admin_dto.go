package dto

import (
	"time"

	"github.com/google/uuid"

	"masjidku_backend/internals/features/lembaga/masjid_admins_teachers/model"
)

// =========================
// Request (Create / Upsert)
// =========================

// pakai pointer di is_active biar optional; default = true
type MasjidAdminRequest struct {
	MasjidAdminMasjidID uuid.UUID `json:"masjid_admin_masjid_id"` // bisa dioverride dari path/token
	MasjidAdminUserID   uuid.UUID `json:"masjid_admin_user_id"`
	MasjidAdminIsActive *bool     `json:"masjid_admin_is_active,omitempty"`
}

// =========================
// Response
// =========================

type MasjidAdminResponse struct {
	MasjidAdminID        uuid.UUID `json:"masjid_admin_id"`
	MasjidAdminMasjidID  uuid.UUID `json:"masjid_admin_masjid_id"`
	MasjidAdminUserID    uuid.UUID `json:"masjid_admin_user_id"`
	MasjidAdminIsActive  bool      `json:"masjid_admin_is_active"`
	MasjidAdminCreatedAt time.Time `json:"masjid_admin_created_at"`
	MasjidAdminUpdatedAt time.Time `json:"masjid_admin_updated_at"`
	// kalau mau expose soft delete timestamp, buka ini:
	// MasjidAdminDeletedAt *time.Time `json:"masjid_admin_deleted_at,omitempty"`
}

// =========================
// Converters
// =========================

// DTO -> Model (untuk CREATE)
// Catatan: controller boleh override MasjidAdminMasjidID dari path/token
func (r *MasjidAdminRequest) ToModelCreate() *model.MasjidAdminModel {
	if r == nil {
		return nil
	}
	active := true
	if r.MasjidAdminIsActive != nil {
		active = *r.MasjidAdminIsActive
	}
	return &model.MasjidAdminModel{
		MasjidAdminMasjidID: r.MasjidAdminMasjidID,
		MasjidAdminUserID:   r.MasjidAdminUserID,
		MasjidAdminIsActive: active,
	}
}

// Model -> DTO (untuk response GET/POST/PUT)
func ToMasjidAdminResponse(m *model.MasjidAdminModel) MasjidAdminResponse {
	return MasjidAdminResponse{
		MasjidAdminID:        m.MasjidAdminID,
		MasjidAdminMasjidID:  m.MasjidAdminMasjidID,
		MasjidAdminUserID:    m.MasjidAdminUserID,
		MasjidAdminIsActive:  m.MasjidAdminIsActive,
		MasjidAdminCreatedAt: m.MasjidAdminCreatedAt,
		MasjidAdminUpdatedAt: m.MasjidAdminUpdatedAt,
	}
}

// Opsional: helper untuk partial update (hanya is_active yang optional)
func (r *MasjidAdminRequest) ApplyPartial(m *model.MasjidAdminModel) {
	if r == nil || m == nil {
		return
	}
	// Jangan ubah FK kalau memang tidak ingin mengizinkan update FK di endpoint
	if r.MasjidAdminIsActive != nil {
		m.MasjidAdminIsActive = *r.MasjidAdminIsActive
	}
}
