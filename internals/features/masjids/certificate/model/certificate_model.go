package model

import (
	"time"

	"github.com/google/uuid"
)

type CertificateModel struct {
    CertificateID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"certificate_id"`
    CertificateTitle       string    `gorm:"type:text;not null" json:"certificate_title"`
    CertificateDescription string    `gorm:"type:text" json:"certificate_description"`
    
    CertificateLectureID   uuid.UUID `gorm:"type:uuid;not null" json:"certificate_lecture_id"`
    CertificateTemplateURL string    `gorm:"type:text" json:"certificate_template_url"`

    CreatedAt              time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt              time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CertificateModel) TableName() string {
	return "certificates"
}
