package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolProfileModel struct {
	// PK & Relasi
	SchoolProfileID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:school_profile_id"        json:"school_profile_id"`
	SchoolProfileSchoolID uuid.UUID `gorm:"type:uuid;unique;not null;column:school_profile_school_id"                       json:"school_profile_school_id"`

	// Deskripsi & sejarah
	SchoolProfileDescription *string `gorm:"type:text;column:school_profile_description"                                      json:"school_profile_description,omitempty"`
	SchoolProfileFoundedYear *int    `gorm:"type:int;column:school_profile_founded_year"                                      json:"school_profile_founded_year,omitempty"`

	// Alamat & kontak publik
	SchoolProfileAddress      *string `gorm:"type:text;column:school_profile_address"                                         json:"school_profile_address,omitempty"`
	SchoolProfileContactPhone *string `gorm:"type:varchar(30);column:school_profile_contact_phone"                            json:"school_profile_contact_phone,omitempty"`
	SchoolProfileContactEmail *string `gorm:"type:varchar(120);column:school_profile_contact_email"                           json:"school_profile_contact_email,omitempty"`

	// Sosial/link publik (incl. maps)
	SchoolProfileGoogleMapsURL          *string `gorm:"type:text;column:school_profile_google_maps_url"                      json:"school_profile_google_maps_url,omitempty"`
	SchoolProfileInstagramURL           *string `gorm:"type:text;column:school_profile_instagram_url"                        json:"school_profile_instagram_url,omitempty"`
	SchoolProfileWhatsappURL            *string `gorm:"type:text;column:school_profile_whatsapp_url"                         json:"school_profile_whatsapp_url,omitempty"`
	SchoolProfileYoutubeURL             *string `gorm:"type:text;column:school_profile_youtube_url"                          json:"school_profile_youtube_url,omitempty"`
	SchoolProfileFacebookURL            *string `gorm:"type:text;column:school_profile_facebook_url"                         json:"school_profile_facebook_url,omitempty"`
	SchoolProfileTiktokURL              *string `gorm:"type:text;column:school_profile_tiktok_url"                           json:"school_profile_tiktok_url,omitempty"`
	SchoolProfileWhatsappGroupIkhwanURL *string `gorm:"type:text;column:school_profile_whatsapp_group_ikhwan_url"            json:"school_profile_whatsapp_group_ikhwan_url,omitempty"`
	SchoolProfileWhatsappGroupAkhwatURL *string `gorm:"type:text;column:school_profile_whatsapp_group_akhwat_url"            json:"school_profile_whatsapp_group_akhwat_url,omitempty"`
	SchoolProfileWebsiteURL             *string `gorm:"type:text;column:school_profile_website_url"                          json:"school_profile_website_url,omitempty"`

	// Koordinat (DOUBLE PRECISION di SQL)
	SchoolProfileLatitude  *float64 `gorm:"type:double precision;column:school_profile_latitude"                             json:"school_profile_latitude,omitempty"`
	SchoolProfileLongitude *float64 `gorm:"type:double precision;column:school_profile_longitude"                            json:"school_profile_longitude,omitempty"`

	// Profil sekolah (opsional) — TANPA phone (sesuai SQL)
	SchoolProfileSchoolNPSN            *string    `gorm:"type:varchar(20);column:school_profile_school_npsn;uniqueIndex:ux_mpp_npsn" json:"school_profile_school_npsn,omitempty"`
	SchoolProfileSchoolNSS             *string    `gorm:"type:varchar(20);column:school_profile_school_nss;uniqueIndex:ux_mpp_nss"   json:"school_profile_school_nss,omitempty"`
	SchoolProfileSchoolAccreditation   *string    `gorm:"type:varchar(10);column:school_profile_school_accreditation"                json:"school_profile_school_accreditation,omitempty"`
	SchoolProfileSchoolPrincipalUserID *uuid.UUID `gorm:"type:uuid;column:school_profile_school_principal_user_id"                  json:"school_profile_school_principal_user_id,omitempty"`
	SchoolProfileSchoolEmail           *string    `gorm:"type:varchar(120);column:school_profile_school_email"                       json:"school_profile_school_email,omitempty"`
	SchoolProfileSchoolAddress         *string    `gorm:"type:text;column:school_profile_school_address"                             json:"school_profile_school_address,omitempty"`
	SchoolProfileSchoolStudentCapacity *int       `gorm:"type:int;column:school_profile_school_student_capacity"                     json:"school_profile_school_student_capacity,omitempty"`
	SchoolProfileSchoolIsBoarding      bool       `gorm:"type:boolean;not null;default:false;column:school_profile_school_is_boarding" json:"school_profile_school_is_boarding"`

	// (SQL terbaru tidak punya kolom FTS, jadi field ini dihapus)
	// SchoolProfileSearch string `gorm:"type:tsvector;column:school_profile_search;<-:false" json:"school_profile_search"`

	// Audit
	SchoolProfileCreatedAt time.Time      `gorm:"autoCreateTime;column:school_profile_created_at" json:"school_profile_created_at"`
	SchoolProfileUpdatedAt time.Time      `gorm:"autoUpdateTime;column:school_profile_updated_at" json:"school_profile_updated_at"`
	SchoolProfileDeletedAt gorm.DeletedAt `gorm:"column:school_profile_deleted_at"                json:"school_profile_deleted_at" swaggertype:"string"`
}

func (SchoolProfileModel) TableName() string {
	return "school_profiles"
}
