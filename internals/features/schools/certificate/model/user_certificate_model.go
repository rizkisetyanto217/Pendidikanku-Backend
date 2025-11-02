package model

import (
	"time"

	"github.com/google/uuid"
)

type UserCertificateModel struct {
	UserCertID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid();column:user_cert_id"`
	UserCertUserID          uuid.UUID `gorm:"type:uuid;column:user_cert_user_id"`
	UserCertCertificateID   uuid.UUID `gorm:"type:uuid;not null;column:user_cert_certificate_id"`
	UserCertScore           *int      `gorm:"column:user_cert_score"`
	UserCertSlugURL         string    `gorm:"unique;not null;column:user_cert_slug_url"`
	UserCertIsUpToDate      bool      `gorm:"not null;default:true;column:user_cert_is_up_to_date"`
	UserCertIssuedAt        time.Time `gorm:"not null;default:current_timestamp;column:user_cert_issued_at"`
	CreatedAt               time.Time `gorm:"default:current_timestamp;column:created_at"`
	UpdatedAt               time.Time `gorm:"default:current_timestamp;column:updated_at"`
}

func (UserCertificateModel) TableName() string {
	return "user_certificates"
}