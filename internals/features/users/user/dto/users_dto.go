package dto

import (
	"strings"
	"time"

	uModel "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
)

/* =======================================================
   REQUEST DTOs
   ======================================================= */

// CreateUserRequest — untuk register / create by admin
// (Tanpa field Role — role dikelola via user_roles)
type CreateUserRequest struct {
	UserName         string  `json:"user_name" validate:"required,min=3,max=50"`
	FullName         string  `json:"full_name" validate:"required,min=3,max=100"`
	Email            string  `json:"email" validate:"required,email,max=255"`
	Password         string  `json:"password" validate:"required,min=8"`
	GoogleID         *string `json:"google_id,omitempty"`
	SecurityQuestion string  `json:"security_question" validate:"required"`
	SecurityAnswer   string  `json:"security_answer" validate:"required,min=3,max=255"`
	IsActive         *bool   `json:"is_active,omitempty"`
}

// Normalize — trim & normalisasi dasar
func (r *CreateUserRequest) Normalize() {
	r.UserName = strings.TrimSpace(r.UserName)
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email)) // citext di DB, tetap normalize
	r.SecurityQuestion = strings.TrimSpace(r.SecurityQuestion)
	r.SecurityAnswer = strings.TrimSpace(r.SecurityAnswer)
}

// ToModel — konversi ke model (hash password di service/controller!)
func (r *CreateUserRequest) ToModel() *uModel.UserModel {
	m := &uModel.UserModel{
		UserName:         r.UserName,
		FullName:         &r.FullName, // model pakai *string
		Email:            r.Email,
		Password:         r.Password, // hash di controller
		GoogleID:         r.GoogleID,
		SecurityQuestion: r.SecurityQuestion,
		SecurityAnswer:   r.SecurityAnswer,
	}
	if r.IsActive != nil {
		m.IsActive = *r.IsActive
	}
	return m
}

// UpdateUserRequest — partial update (pakai pointer agar bisa bedakan omit vs null)
type UpdateUserRequest struct {
	UserName         *string `json:"user_name,omitempty" validate:"omitempty,min=3,max=50"`
	FullName         *string `json:"full_name,omitempty" validate:"omitempty,min=3,max=100"`
	Email            *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Password         *string `json:"password,omitempty" validate:"omitempty,min=8"`
	GoogleID         *string `json:"google_id,omitempty"`
	SecurityQuestion *string `json:"security_question,omitempty" validate:"omitempty"`
	SecurityAnswer   *string `json:"security_answer,omitempty" validate:"omitempty,min=3,max=255"`
	IsActive         *bool   `json:"is_active,omitempty"`
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
	if r.SecurityQuestion != nil {
		v := strings.TrimSpace(*r.SecurityQuestion)
		r.SecurityQuestion = &v
	}
	if r.SecurityAnswer != nil {
		v := strings.TrimSpace(*r.SecurityAnswer)
		r.SecurityAnswer = &v
	}
}

// ApplyToModel — terapkan perubahan parsial ke model existing
func (r *UpdateUserRequest) ApplyToModel(m *uModel.UserModel) {
	if r.UserName != nil {
		m.UserName = *r.UserName
	}
	if r.FullName != nil {
		// model: *string
		m.FullName = r.FullName
	}
	if r.Email != nil {
		m.Email = *r.Email
	}
	if r.Password != nil {
		m.Password = *r.Password // hash di controller sebelum Save
	}
	if r.GoogleID != nil {
		m.GoogleID = r.GoogleID // bisa di-nil-kan
	}
	if r.SecurityQuestion != nil {
		m.SecurityQuestion = *r.SecurityQuestion
	}
	if r.SecurityAnswer != nil {
		m.SecurityAnswer = *r.SecurityAnswer
	}
	if r.IsActive != nil {
		m.IsActive = *r.IsActive
	}
}

/* =======================================================
   ROLE MANAGEMENT DTOs (via user_roles)
   ======================================================= */

// GrantRoleRequest — endpoint: POST /api/a/user-roles/grant
type GrantRoleRequest struct {
	UserID    uuid.UUID  `json:"user_id" validate:"required"`
	RoleName  string     `json:"role_name" validate:"required"`
	MasjidID  *uuid.UUID `json:"masjid_id,omitempty"` // null = global
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
}

// RevokeRoleRequest — endpoint: POST /api/a/user-roles/revoke
type RevokeRoleRequest struct {
	UserID   uuid.UUID  `json:"user_id" validate:"required"`
	RoleName string     `json:"role_name" validate:"required"`
	MasjidID *uuid.UUID `json:"masjid_id,omitempty"` // null = global
}

/* =======================================================
   RESPONSE DTOs
   ======================================================= */

// Default user response (tanpa deleted_at, tanpa role)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	UserName  string    `json:"user_name"`
	FullName  *string   `json:"full_name,omitempty"`
	Email     string    `json:"email"`
	GoogleID  *string   `json:"google_id,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserResponseWithDeletedAt — admin-only (expose deleted_at)
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

// UserWithRolesResponse — gabungan user + roles claim
type UserWithRolesResponse struct {
	User  UserResponse `json:"user"`
	Roles RolesClaim   `json:"roles"`
}

// FromModel — map model ke UserResponse
func FromModel(m *uModel.UserModel) *UserResponse {
	if m == nil {
		return nil
	}
	return &UserResponse{
		ID:        m.ID,
		UserName:  m.UserName,
		FullName:  m.FullName,
		Email:     m.Email,
		GoogleID:  m.GoogleID,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromModelList(list []uModel.UserModel) []UserResponse {
	out := make([]UserResponse, 0, len(list))
	for i := range list {
		out = append(out, *FromModel(&list[i]))
	}
	return out
}

// FromModelWithDeletedAt — gunakan bila butuh expose deleted_at (admin tools)
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
