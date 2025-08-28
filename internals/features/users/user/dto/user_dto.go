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
type CreateUserRequest struct {
	UserName         string  `json:"user_name" validate:"required,min=3,max=50"`
	FullName         string  `json:"full_name" validate:"required,min=3,max=100"` // ⇐ disamakan dgn model/DDL
	Email            string  `json:"email" validate:"required,email,max=255"`
	Password         string  `json:"password" validate:"required,min=8"`
	GoogleID         *string `json:"google_id,omitempty"`
	Role             string  `json:"role" validate:"omitempty,oneof=owner user teacher treasurer admin dkm author student"`
	SecurityQuestion string  `json:"security_question" validate:"required"`
	SecurityAnswer   string  `json:"security_answer" validate:"required,min=3,max=255"`
	IsActive         *bool   `json:"is_active,omitempty"`
}

// Normalize — trim & normalisasi dasar
func (r *CreateUserRequest) Normalize() {
	r.UserName = strings.TrimSpace(r.UserName)
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email)) // citext di DB, tapi tetap normalize
	r.Role = strings.TrimSpace(r.Role)
	r.SecurityQuestion = strings.TrimSpace(r.SecurityQuestion)
	r.SecurityAnswer = strings.TrimSpace(r.SecurityAnswer)
}

// ToModel — konversi ke model (ingat: hash password di controller!)
func (r *CreateUserRequest) ToModel() *uModel.UserModel {
	m := &uModel.UserModel{
		UserName:         r.UserName,
		FullName:         r.FullName,
		Email:            r.Email,
		Password:         r.Password, // hash di controller
		GoogleID:         r.GoogleID,
		Role:             r.Role,
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
	FullName         *string `json:"full_name,omitempty" validate:"omitempty,min=3,max=100"` // ⇐ update 100
	Email            *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Password         *string `json:"password,omitempty" validate:"omitempty,min=8"`
	GoogleID         *string `json:"google_id,omitempty"`
	Role             *string `json:"role,omitempty" validate:"omitempty,oneof=owner user teacher treasurer admin dkm author student"`
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
	if r.Role != nil {
		v := strings.TrimSpace(*r.Role)
		r.Role = &v
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
		m.FullName = *r.FullName
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
	if r.Role != nil {
		m.Role = *r.Role
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
   RESPONSE DTOs
   ======================================================= */

// Default response (tanpa deleted_at; aman untuk publik)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	UserName  string    `json:"user_name"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	GoogleID  *string   `json:"google_id,omitempty"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// NOTE: Password, SecurityQuestion, SecurityAnswer disembunyikan
}

// Opsi: response yang menyertakan deleted_at (pakai kalau perlu admin-only)
type UserResponseWithDeletedAt struct {
	UserResponse
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
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
		Role:      m.Role,
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

// FromModelWithDeletedAt — gunakan bila kamu butuh expose deleted_at (admin tools)
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
