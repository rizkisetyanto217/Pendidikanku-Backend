package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidProfileModel struct {
	// PK & Relasi
	MasjidProfileID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_profile_id"        json:"masjid_profile_id"`
	MasjidProfileMasjidID uuid.UUID `gorm:"type:uuid;unique;not null;column:masjid_profile_masjid_id"                       json:"masjid_profile_masjid_id"`

	// Deskripsi & sejarah
	MasjidProfileDescription *string `gorm:"type:text;column:masjid_profile_description"                                      json:"masjid_profile_description,omitempty"`
	MasjidProfileFoundedYear *int    `gorm:"type:int;column:masjid_profile_founded_year"                                      json:"masjid_profile_founded_year,omitempty"`

	// Alamat & kontak publik
	MasjidProfileAddress      *string `gorm:"type:text;column:masjid_profile_address"                                         json:"masjid_profile_address,omitempty"`
	MasjidProfileContactPhone *string `gorm:"type:varchar(30);column:masjid_profile_contact_phone"                            json:"masjid_profile_contact_phone,omitempty"`
	MasjidProfileContactEmail *string `gorm:"type:varchar(120);column:masjid_profile_contact_email"                           json:"masjid_profile_contact_email,omitempty"`

	// Sosial/link publik (incl. maps)
	MasjidProfileGoogleMapsURL          *string `gorm:"type:text;column:masjid_profile_google_maps_url"                      json:"masjid_profile_google_maps_url,omitempty"`
	MasjidProfileInstagramURL           *string `gorm:"type:text;column:masjid_profile_instagram_url"                        json:"masjid_profile_instagram_url,omitempty"`
	MasjidProfileWhatsappURL            *string `gorm:"type:text;column:masjid_profile_whatsapp_url"                         json:"masjid_profile_whatsapp_url,omitempty"`
	MasjidProfileYoutubeURL             *string `gorm:"type:text;column:masjid_profile_youtube_url"                          json:"masjid_profile_youtube_url,omitempty"`
	MasjidProfileFacebookURL            *string `gorm:"type:text;column:masjid_profile_facebook_url"                         json:"masjid_profile_facebook_url,omitempty"`
	MasjidProfileTiktokURL              *string `gorm:"type:text;column:masjid_profile_tiktok_url"                           json:"masjid_profile_tiktok_url,omitempty"`
	MasjidProfileWhatsappGroupIkhwanURL *string `gorm:"type:text;column:masjid_profile_whatsapp_group_ikhwan_url"            json:"masjid_profile_whatsapp_group_ikhwan_url,omitempty"`
	MasjidProfileWhatsappGroupAkhwatURL *string `gorm:"type:text;column:masjid_profile_whatsapp_group_akhwat_url"            json:"masjid_profile_whatsapp_group_akhwat_url,omitempty"`
	MasjidProfileWebsiteURL             *string `gorm:"type:text;column:masjid_profile_website_url"                          json:"masjid_profile_website_url,omitempty"`

	// Koordinat (DOUBLE PRECISION di SQL)
	MasjidProfileLatitude  *float64 `gorm:"type:double precision;column:masjid_profile_latitude"                             json:"masjid_profile_latitude,omitempty"`
	MasjidProfileLongitude *float64 `gorm:"type:double precision;column:masjid_profile_longitude"                            json:"masjid_profile_longitude,omitempty"`

	// Profil sekolah (opsional) — TANPA phone (sesuai SQL)
	MasjidProfileSchoolNPSN            *string    `gorm:"type:varchar(20);column:masjid_profile_school_npsn;uniqueIndex:ux_mpp_npsn" json:"masjid_profile_school_npsn,omitempty"`
	MasjidProfileSchoolNSS             *string    `gorm:"type:varchar(20);column:masjid_profile_school_nss;uniqueIndex:ux_mpp_nss"   json:"masjid_profile_school_nss,omitempty"`
	MasjidProfileSchoolAccreditation   *string    `gorm:"type:varchar(10);column:masjid_profile_school_accreditation"                json:"masjid_profile_school_accreditation,omitempty"`
	MasjidProfileSchoolPrincipalUserID *uuid.UUID `gorm:"type:uuid;column:masjid_profile_school_principal_user_id"                  json:"masjid_profile_school_principal_user_id,omitempty"`
	MasjidProfileSchoolEmail           *string    `gorm:"type:varchar(120);column:masjid_profile_school_email"                       json:"masjid_profile_school_email,omitempty"`
	MasjidProfileSchoolAddress         *string    `gorm:"type:text;column:masjid_profile_school_address"                             json:"masjid_profile_school_address,omitempty"`
	MasjidProfileSchoolStudentCapacity *int       `gorm:"type:int;column:masjid_profile_school_student_capacity"                     json:"masjid_profile_school_student_capacity,omitempty"`
	MasjidProfileSchoolIsBoarding      bool       `gorm:"type:boolean;not null;default:false;column:masjid_profile_school_is_boarding" json:"masjid_profile_school_is_boarding"`

	// (SQL terbaru tidak punya kolom FTS, jadi field ini dihapus)
	// MasjidProfileSearch string `gorm:"type:tsvector;column:masjid_profile_search;<-:false" json:"masjid_profile_search"`

	// Audit
	MasjidProfileCreatedAt time.Time      `gorm:"autoCreateTime;column:masjid_profile_created_at" json:"masjid_profile_created_at"`
	MasjidProfileUpdatedAt time.Time      `gorm:"autoUpdateTime;column:masjid_profile_updated_at" json:"masjid_profile_updated_at"`
	MasjidProfileDeletedAt gorm.DeletedAt `gorm:"column:masjid_profile_deleted_at"                json:"masjid_profile_deleted_at" swaggertype:"string"`
}

func (MasjidProfileModel) TableName() string {
	return "masjid_profiles"
}
