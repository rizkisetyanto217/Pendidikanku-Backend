// file: internals/features/authz/dto/user_role_dto.go
package dto

import (
	"time"

	"madinahsalam_backend/internals/features/users/users/model"

	"github.com/google/uuid"
)

/* =========
   Response
   ========= */

type UserRoleResponse struct {
	UserRoleID uuid.UUID  `json:"user_role_id"`
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     uuid.UUID  `json:"role_id"`
	SchoolID   *uuid.UUID `json:"school_id,omitempty"`
	AssignedAt *time.Time `json:"assigned_at,omitempty"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func FromModelUserRole(m model.UserRole) UserRoleResponse {
	return UserRoleResponse{
		UserRoleID: m.UserRoleID,
		UserID:     m.UserID,
		RoleID:     m.RoleID,
		SchoolID:   m.SchoolID,
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
	SchoolID   *uuid.UUID `json:"school_id,omitempty"`   // null = global
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"` // opsional
}

func (r CreateUserRoleRequest) ToModel() model.UserRole {
	return model.UserRole{
		UserID:     r.UserID,
		RoleID:     r.RoleID,
		SchoolID:   r.SchoolID,
		AssignedBy: r.AssignedBy,
		// AssignedAt biarkan diisi default DB (NOW())
	}
}

/* =========
   Update (partial)
   ========= */

type UpdateUserRoleRequest struct {
	// Hanya izinkan ubah scope & assigned_by (user_id & role_id dianggap immutable)
	SchoolID      *uuid.UUID `json:"school_id,omitempty"`       // kirim untuk ubah nilai
	ClearSchoolID *bool      `json:"clear_school_id,omitempty"` // true => set NULL (global)
	AssignedBy    *uuid.UUID `json:"assigned_by,omitempty"`
}

// Apply menerapkan perubahan partial ke model.
//
// Aturan SchoolID:
// - Jika ClearSchoolID == true -> set m.SchoolID = nil
// - Else jika SchoolID != nil   -> set m.SchoolID = SchoolID
// - Else                        -> tidak diubah
func (r UpdateUserRoleRequest) Apply(m *model.UserRole) {
	if r.ClearSchoolID != nil && *r.ClearSchoolID {
		m.SchoolID = nil
	} else if r.SchoolID != nil {
		m.SchoolID = r.SchoolID
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
	SchoolID  *uuid.UUID `query:"school_id"`  // null = global, kosong = semua
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
