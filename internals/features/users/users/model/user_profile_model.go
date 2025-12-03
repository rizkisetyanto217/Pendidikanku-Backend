package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Gender string

const (
	Male   Gender = "male"
	Female Gender = "female"
)

type UserProfileModel struct {
	// PK
	UserProfileID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid();column:user_profile_id" json:"user_profile_id"`

	// FK & Unique per user
	UserProfileUserID uuid.UUID `gorm:"type:uuid;not null;column:user_profile_user_id;uniqueIndex:uq_user_profile_user_id" json:"user_profile_user_id"`

	// Snapshot nama user (dari users)
	UserProfileFullNameCache *string `gorm:"size:100;column:user_profile_full_name_cache" json:"user_profile_full_name_cache,omitempty"`

	// Identitas dasar
	UserProfileSlug         *string    `gorm:"size:80;column:user_profile_slug" json:"user_profile_slug,omitempty"`
	UserProfileDonationName *string    `gorm:"size:50;column:user_profile_donation_name" json:"user_profile_donation_name,omitempty"`
	UserProfileDateOfBirth  *time.Time `gorm:"type:date;column:user_profile_date_of_birth" json:"user_profile_date_of_birth,omitempty"`
	UserProfilePlaceOfBirth *string    `gorm:"size:100;column:user_profile_place_of_birth" json:"user_profile_place_of_birth,omitempty"`
	UserProfileGender       *Gender    `gorm:"type:varchar(10);column:user_profile_gender" json:"user_profile_gender,omitempty"`
	UserProfileLocation     *string    `gorm:"size:100;column:user_profile_location" json:"user_profile_location,omitempty"`
	UserProfileCity         *string    `gorm:"size:100;column:user_profile_city" json:"user_profile_city,omitempty"`
	UserProfileBio          *string    `gorm:"size:300;column:user_profile_bio" json:"user_profile_bio,omitempty"`

	// Konten panjang & riwayat
	UserProfileBiographyLong  *string `gorm:"type:text;column:user_profile_biography_long" json:"user_profile_biography_long,omitempty"`
	UserProfileExperience     *string `gorm:"type:text;column:user_profile_experience" json:"user_profile_experience,omitempty"`
	UserProfileCertifications *string `gorm:"type:text;column:user_profile_certifications" json:"user_profile_certifications,omitempty"`

	// Sosial media
	UserProfileInstagramURL     *string `gorm:"type:text;column:user_profile_instagram_url" json:"user_profile_instagram_url,omitempty"`
	UserProfileWhatsappURL      *string `gorm:"type:text;column:user_profile_whatsapp_url" json:"user_profile_whatsapp_url,omitempty"`
	UserProfileYoutubeURL       *string `gorm:"type:text;column:user_profile_youtube_url" json:"user_profile_youtube_url,omitempty"`
	UserProfileLinkedinURL      *string `gorm:"type:text;column:user_profile_linkedin_url" json:"user_profile_linkedin_url,omitempty"`
	UserProfileGithubURL        *string `gorm:"type:text;column:user_profile_github_url" json:"user_profile_github_url,omitempty"`
	UserProfileTelegramUsername *string `gorm:"size:50;column:user_profile_telegram_username" json:"user_profile_telegram_username,omitempty"`

	// Orang tua / wali
	UserProfileParentName        *string `gorm:"size:100;column:user_profile_parent_name" json:"user_profile_parent_name,omitempty"`
	UserProfileParentWhatsappURL *string `gorm:"type:text;column:user_profile_parent_whatsapp_url" json:"user_profile_parent_whatsapp_url,omitempty"`

	// Avatar (single file, 2-slot + retensi 30 hari)
	UserProfileAvatarURL                *string    `gorm:"type:text;column:user_profile_avatar_url" json:"user_profile_avatar_url,omitempty"`
	UserProfileAvatarObjectKey          *string    `gorm:"type:text;column:user_profile_avatar_object_key" json:"user_profile_avatar_object_key,omitempty"`
	UserProfileAvatarURLOld             *string    `gorm:"type:text;column:user_profile_avatar_url_old" json:"user_profile_avatar_url_old,omitempty"`
	UserProfileAvatarObjectKeyOld       *string    `gorm:"type:text;column:user_profile_avatar_object_key_old" json:"user_profile_avatar_object_key_old,omitempty"`
	UserProfileAvatarDeletePendingUntil *time.Time `gorm:"column:user_profile_avatar_delete_pending_until" json:"user_profile_avatar_delete_pending_until,omitempty"`

	// Privasi & verifikasi (oleh platform Pendidikanku)
	UserProfileIsPublicProfile bool       `gorm:"not null;default:true;column:user_profile_is_public_profile" json:"user_profile_is_public_profile"`
	UserProfileIsVerified      bool       `gorm:"not null;default:false;column:user_profile_is_verified" json:"user_profile_is_verified"`
	UserProfileVerifiedAt      *time.Time `gorm:"column:user_profile_verified_at" json:"user_profile_verified_at,omitempty"`
	UserProfileVerifiedBy      *uuid.UUID `gorm:"type:uuid;column:user_profile_verified_by" json:"user_profile_verified_by,omitempty"`

	// Pendidikan & pekerjaan
	UserProfileEducation *string `gorm:"type:text;column:user_profile_education" json:"user_profile_education,omitempty"`
	UserProfileCompany   *string `gorm:"type:text;column:user_profile_company" json:"user_profile_company,omitempty"`
	UserProfilePosition  *string `gorm:"type:text;column:user_profile_position" json:"user_profile_position,omitempty"`

	// Interests & skills (text[])
	UserProfileInterests pq.StringArray `gorm:"type:text[];column:user_profile_interests" json:"user_profile_interests"`
	UserProfileSkills    pq.StringArray `gorm:"type:text[];column:user_profile_skills" json:"user_profile_skills"`

	// Status kelengkapan profil (wajib gender + whatsapp kalau TRUE)
	UserProfileIsCompleted bool       `gorm:"not null;default:false;column:user_profile_is_completed" json:"user_profile_is_completed"`
	UserProfileCompletedAt *time.Time `gorm:"column:user_profile_completed_at" json:"user_profile_completed_at,omitempty"`

	// Audit
	UserProfileCreatedAt time.Time      `gorm:"autoCreateTime;column:user_profile_created_at" json:"user_profile_created_at"`
	UserProfileUpdatedAt time.Time      `gorm:"autoUpdateTime;column:user_profile_updated_at" json:"user_profile_updated_at"`
	UserProfileDeletedAt gorm.DeletedAt `gorm:"column:user_profile_deleted_at;index" json:"user_profile_deleted_at,omitempty"`
}

func (UserProfileModel) TableName() string { return "user_profiles" }
