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

type CertificateDetailResponse struct {
	CertificateID                 uuid.UUID `json:"certificate_id"`
	CertificateTitle              string    `json:"certificate_title"`
	CertificateDescription        string    `json:"certificate_description"`
	CertificateTemplateURL        string    `json:"certificate_template_url"`
	LectureTitle                  string    `json:"lecture_title"`
	LectureIsCertificateGenerated bool      `json:"lecture_is_certificate_generated"`
	SchoolID                      uuid.UUID `json:"school_id"`
	SchoolName                    string    `json:"school_name"`
	SchoolImageURL                *string   `json:"school_image_url"`
	UserLectureExamUserName       string    `json:"user_lecture_exam_user_name"`
	UserLectureExamGradeResult    *int      `json:"user_lecture_exam_grade_result,omitempty"`
}
