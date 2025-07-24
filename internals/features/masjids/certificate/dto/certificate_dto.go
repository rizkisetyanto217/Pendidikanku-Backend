package dto

import (
	"time"

	"github.com/google/uuid"
)

// Digunakan saat membuat sertifikat baru
type CreateCertificateDTO struct {
	CertificateTitle       string    `json:"certificate_title" validate:"required"`
	CertificateDescription string    `json:"certificate_description,omitempty"`
	CertificateLectureID   uuid.UUID `json:"certificate_lecture_id" validate:"required"`
	CertificateTemplateURL string    `json:"certificate_template_url,omitempty"`
}

// Digunakan untuk update sebagian data sertifikat
type UpdateCertificateDTO struct {
	CertificateTitle       *string    `json:"certificate_title,omitempty"`
	CertificateDescription *string    `json:"certificate_description,omitempty"`
	CertificateLectureID   *uuid.UUID `json:"certificate_lecture_id,omitempty"`
	CertificateTemplateURL *string    `json:"certificate_template_url,omitempty"`
}

// Digunakan untuk merespons ke frontend
type CertificateResponseDTO struct {
	CertificateID          uuid.UUID `json:"certificate_id"`
	CertificateTitle       string    `json:"certificate_title"`
	CertificateDescription string    `json:"certificate_description"`
	CertificateLectureID   uuid.UUID `json:"certificate_lecture_id"`
	CertificateTemplateURL string    `json:"certificate_template_url"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
