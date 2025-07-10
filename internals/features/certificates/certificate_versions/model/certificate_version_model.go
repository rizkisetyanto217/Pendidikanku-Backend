package model

import "time"

type CertificateVersionModel struct {
	CertVersionID            uint       `json:"cert_version_id" gorm:"column:cert_versions_id;primaryKey"`
	CertVersionSubcategoryID uint       `json:"cert_version_subcategory_id" gorm:"column:cert_versions_subcategory_id;not null"`
	CertVersionNumber        int        `json:"cert_version_number" gorm:"column:cert_versions_number;not null"`
	CertVersionTotalThemes          int        `json:"cert_total_themes" gorm:"column:cert_versions_total_themes;not null;default:0"`
	CertVersionNote          string     `json:"cert_version_note" gorm:"column:cert_versions_note"`
	CreatedAt                time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt                *time.Time `json:"updated_at,omitempty" gorm:"column:updated_at"`
}

func (CertificateVersionModel) TableName() string {
	return "certificate_versions"
}