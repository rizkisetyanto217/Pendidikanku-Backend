package dto

import "github.com/google/uuid"

// 🔹 Create DTO
type CreateUserCertificateDTO struct {
	UserCertUserID        uuid.UUID `json:"user_cert_user_id" validate:"required"`
	UserCertCertificateID uuid.UUID `json:"user_cert_certificate_id" validate:"required"`
	UserCertScore         *int      `json:"user_cert_score"` // Optional
	UserCertSlugURL       string    `json:"user_cert_slug_url" validate:"required"`
	UserCertIsUpToDate    *bool     `json:"user_cert_is_up_to_date"` // Optional, default true
}

// 🔹 Update DTO
type UpdateUserCertificateDTO struct {
	UserCertScore       *int    `json:"user_cert_score"`
	UserCertSlugURL     *string `json:"user_cert_slug_url"`
	UserCertIsUpToDate  *bool   `json:"user_cert_is_up_to_date"`
}
// 🔹 Response DTO
type UserCertificateResponseDTO struct {
	UserCertID            uuid.UUID `json:"user_cert_id"`
	UserCertUserID        uuid.UUID `json:"user_cert_user_id"`
	UserCertCertificateID uuid.UUID `json:"user_cert_certificate_id"`
	UserCertScore         *int      `json:"user_cert_score,omitempty"`
	UserCertSlugURL       string    `json:"user_cert_slug_url"`
	UserCertIsUpToDate    bool      `json:"user_cert_is_up_to_date"`
	UserCertIssuedAt      string    `json:"user_cert_issued_at"`
	CreatedAt             string    `json:"created_at"`
	UpdatedAt             string    `json:"updated_at"`
}
