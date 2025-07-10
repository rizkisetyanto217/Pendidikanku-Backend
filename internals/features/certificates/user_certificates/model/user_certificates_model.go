package model

import (
	"time"

	"github.com/google/uuid"
	// subcategoryModel "masjidku_backend/internals/features/lessons/subcategory/model"
)

type UserCertificate struct {
	UserCertID            uint      `json:"user_cert_id" gorm:"column:user_cert_id;primaryKey"`
	UserCertUserID        uuid.UUID `json:"user_cert_user_id" gorm:"column:user_cert_user_id;type:uuid;not null"`
	UserCertSubcategoryID uint      `json:"user_cert_subcategory_id" gorm:"column:user_cert_subcategory_id;not null"`
	UserCertIsUpToDate    bool      `json:"user_cert_is_up_to_date" gorm:"column:user_cert_is_up_to_date;not null;default:true"`
	UserCertSlugURL       string    `json:"user_cert_slug_url" gorm:"column:user_cert_slug_url;unique;not null"`
	UserCertIssuedAt      time.Time `json:"user_cert_issued_at" gorm:"column:user_cert_issued_at;not null"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (UserCertificate) TableName() string {
	return "user_certificates"
}
