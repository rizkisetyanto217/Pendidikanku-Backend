package dto

import (
	"errors"
	"strings"
	"time"

	uModel "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
)

/* =======================================================
   REQUEST DTOs
   ======================================================= */

// CreateUserRequest — untuk register / create by admin
// Catatan: minimal salah satu dari Password atau GoogleID harus terisi.
type CreateUserRequest struct {
	UserName string  `json:"user_name" validate:"required,min=3,max=50"`
	FullName string  `json:"full_name" validate:"required,min=3,max=100"`
	Email    string  `json:"email" validate:"required,email,max=255"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=8"`
	GoogleID *string `json:"google_id,omitempty" validate:"omitempty,max=255"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// Normalize — trim & normalisasi dasar
func (r *CreateUserRequest) Normalize() {
	r.UserName = strings.TrimSpace(r.UserName)
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Password != nil {
		v := strings.TrimSpace(*r.Password)
		r.Password = &v
	}
	if r.GoogleID != nil {
		v := strings.TrimSpace(*r.GoogleID)
		r.GoogleID = &v
	}
}

// Validate business rule khusus
func (r *CreateUserRequest) Validate() error {
	// Minimal salah satu: password atau google_id
	if (r.Password == nil || *r.Password == "") && (r.GoogleID == nil || *r.GoogleID == "") {
		return errors.New("password atau google_id wajib diisi salah satu")
	}
	return nil
}

// ToModel — konversi ke model (hash password di service/controller!)
func (r *CreateUserRequest) ToModel() *uModel.UserModel {
	m := &uModel.UserModel{
		UserName: r.UserName,
		FullName: &r.FullName,
		Email:    r.Email,
		Password: r.Password, // hash di controller jika tidak nil
		GoogleID: r.GoogleID,
		// IsActive default true di DB; override jika diberikan
	}
	if r.IsActive != nil {
		m.IsActive = *r.IsActive
	}
	return m
}

// UpdateUserRequest — partial update (bedakan omit vs null)
type UpdateUserRequest struct {
	UserName        *string    `json:"user_name,omitempty" validate:"omitempty,min=3,max=50"`
	FullName        *string    `json:"full_name,omitempty" validate:"omitempty,min=3,max=100"`
	Email           *string    `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Password        *string    `json:"password,omitempty" validate:"omitempty,min=8"`
	GoogleID        *string    `json:"google_id,omitempty" validate:"omitempty,max=255"`
	IsActive        *bool      `json:"is_active,omitempty"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"` // opsional; biasanya dikelola oleh flow verifikasi
}

// Normalize — trims if present
func (r *UpdateUserRequest) Normalize() {
	if r.UserName != nil {
		v := strings.TrimSpace(*r.UserName)
		r.UserName = &v
	}
	if r.FullName != nil {
		v := strings.TrimSpace(*r.FullName)
		r.FullName = &v
	}
	if r.Email != nil {
		v := strings.TrimSpace(strings.ToLower(*r.Email))
		r.Email = &v
	}
	if r.Password != nil {
		v := strings.TrimSpace(*r.Password)
		r.Password = &v
	}
	if r.GoogleID != nil {
		v := strings.TrimSpace(*r.GoogleID)
		r.GoogleID = &v
	}
}

// ApplyToModel — terapkan perubahan parsial ke model existing
func (r *UpdateUserRequest) ApplyToModel(m *uModel.UserModel) {
	if r.UserName != nil {
		m.UserName = *r.UserName
	}
	if r.FullName != nil {
		m.FullName = r.FullName // model pakai *string
	}
	if r.Email != nil {
		m.Email = *r.Email
	}
	if r.Password != nil {
		// hash di controller sebelum Save (jika tidak empty)
		m.Password = r.Password
	}
	if r.GoogleID != nil {
		m.GoogleID = r.GoogleID // boleh nil-kan untuk cabut link SSO
	}
	if r.IsActive != nil {
		m.IsActive = *r.IsActive
	}
	if r.EmailVerifiedAt != nil {
		m.EmailVerifiedAt = r.EmailVerifiedAt
	}
}

/* =======================================================
   ROLE MANAGEMENT DTOs (via user_roles)
   ======================================================= */

type GrantRoleRequest struct {
	UserID     uuid.UUID  `json:"user_id" validate:"required"`
	RoleName   string     `json:"role_name" validate:"required"`
	MasjidID   *uuid.UUID `json:"masjid_id,omitempty"` // null = global
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
}

type RevokeRoleRequest struct {
	UserID   uuid.UUID  `json:"user_id" validate:"required"`
	RoleName string     `json:"role_name" validate:"required"`
	MasjidID *uuid.UUID `json:"masjid_id,omitempty"` // null = global
}

/* =======================================================
   RESPONSE DTOs
   ======================================================= */

type UserResponse struct {
	ID              uuid.UUID  `json:"id"`
	UserName        string     `json:"user_name"`
	FullName        *string    `json:"full_name,omitempty"`
	Email           string     `json:"email"`
	GoogleID        *string    `json:"google_id,omitempty"`
	IsActive        bool       `json:"is_active"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UserResponseWithDeletedAt struct {
	UserResponse
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// RolesClaim (cermin dari fn_user_roles_claim)
type MasjidRole struct {
	MasjidID uuid.UUID `json:"masjid_id"`
	Roles    []string  `json:"roles"`
}
type RolesClaim struct {
	RoleGlobal  []string     `json:"role_global"`
	MasjidRoles []MasjidRole `json:"masjid_roles"`
}

type UserWithRolesResponse struct {
	User  UserResponse `json:"user"`
	Roles RolesClaim   `json:"roles"`
}

/* =======================================================
   Mappers
   ======================================================= */

func FromModel(m *uModel.UserModel) *UserResponse {
	if m == nil {
		return nil
	}
	return &UserResponse{
		ID:              m.ID,
		UserName:        m.UserName,
		FullName:        m.FullName,
		Email:           m.Email,
		GoogleID:        m.GoogleID,
		IsActive:        m.IsActive,
		EmailVerifiedAt: m.EmailVerifiedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func FromModelList(list []uModel.UserModel) []UserResponse {
	out := make([]UserResponse, 0, len(list))
	for i := range list {
		out = append(out, *FromModel(&list[i]))
	}
	return out
}

func FromModelWithDeletedAt(m *uModel.UserModel) *UserResponseWithDeletedAt {
	if m == nil {
		return nil
	}
	var del *time.Time
	if !m.DeletedAt.Time.IsZero() {
		t := m.DeletedAt.Time
		del = &t
	}
	return &UserResponseWithDeletedAt{
		UserResponse: *FromModel(m),
		DeletedAt:    del,
	}
}
