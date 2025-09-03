// file: internals/features/authz/dto/user_role_dto.go
package dto

import (
	"time"

	"masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
)

/* =========
   Response
   ========= */

type UserRoleResponse struct {
	UserRoleID uuid.UUID  `json:"user_role_id"`
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     uuid.UUID  `json:"role_id"`
	MasjidID   *uuid.UUID `json:"masjid_id,omitempty"`
	AssignedAt *time.Time `json:"assigned_at,omitempty"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func FromModelUserRole(m model.UserRole) UserRoleResponse {
	return UserRoleResponse{
		UserRoleID: m.UserRoleID,
		UserID:     m.UserID,
		RoleID:     m.RoleID,
		MasjidID:   m.MasjidID,
		AssignedAt: m.AssignedAt,
		AssignedBy: m.AssignedBy,
		DeletedAt:  m.DeletedAt,
	}
}

/* =========
   Create
   ========= */

type CreateUserRoleRequest struct {
	UserID     uuid.UUID  `json:"user_id"    validate:"required"`
	RoleID     uuid.UUID  `json:"role_id"    validate:"required"`
	MasjidID   *uuid.UUID `json:"masjid_id,omitempty"`   // null = global
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"` // opsional
}

func (r CreateUserRoleRequest) ToModel() model.UserRole {
	return model.UserRole{
		UserID:     r.UserID,
		RoleID:     r.RoleID,
		MasjidID:   r.MasjidID,
		AssignedBy: r.AssignedBy,
		// AssignedAt biarkan diisi default DB (NOW())
	}
}

/* =========
   Update (partial)
   ========= */

type UpdateUserRoleRequest struct {
	// Hanya izinkan ubah scope & assigned_by (user_id & role_id dianggap immutable)
	MasjidID      *uuid.UUID `json:"masjid_id,omitempty"`       // kirim untuk ubah nilai
	ClearMasjidID *bool      `json:"clear_masjid_id,omitempty"` // true => set NULL (global)
	AssignedBy    *uuid.UUID `json:"assigned_by,omitempty"`
}

// Apply menerapkan perubahan partial ke model.
//
// Aturan MasjidID:
// - Jika ClearMasjidID == true -> set m.MasjidID = nil
// - Else jika MasjidID != nil   -> set m.MasjidID = MasjidID
// - Else                        -> tidak diubah
func (r UpdateUserRoleRequest) Apply(m *model.UserRole) {
	if r.ClearMasjidID != nil && *r.ClearMasjidID {
		m.MasjidID = nil
	} else if r.MasjidID != nil {
		m.MasjidID = r.MasjidID
	}
	if r.AssignedBy != nil {
		m.AssignedBy = r.AssignedBy
	}
}

/* =========
   List Query (filter & paging)
   ========= */

type ListUserRoleQuery struct {
	UserID    *uuid.UUID `query:"user_id"`
	RoleID    *uuid.UUID `query:"role_id"`
	MasjidID  *uuid.UUID `query:"masjid_id"`  // null = global, kosong = semua
	OnlyAlive *bool      `query:"only_alive"` // default: true
	Limit     int        `query:"limit"`      // default: 20
	Offset    int        `query:"offset"`     // default: 0
	OrderBy   string     `query:"order_by"`   // assigned_at|user_id|role_id (default assigned_at)
	Sort      string     `query:"sort"`       // asc|desc (default desc)
}

/* =========
   Common: Pagination meta (ringan)
   ========= */

type Pagination struct {
	Total      int `json:"total"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
	Returned   int `json:"returned"`
	NextOffset int `json:"next_offset"`
	PrevOffset int `json:"prev_offset"`
}
