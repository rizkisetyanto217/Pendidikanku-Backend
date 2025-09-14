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

type UsersProfileModel struct {
	// PK
	UsersProfileID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid();column:users_profile_id" json:"users_profile_id"`

	// FK & Unique
	UsersProfileUserID uuid.UUID `gorm:"type:uuid;not null;column:users_profile_user_id;uniqueIndex:uq_users_profile_user_id" json:"users_profile_user_id"`

	// Identitas dasar
	UsersProfileDonationName string     `gorm:"size:50;column:users_profile_donation_name" json:"users_profile_donation_name"`
	UsersProfileDateOfBirth  *time.Time `gorm:"type:date;column:users_profile_date_of_birth" json:"users_profile_date_of_birth,omitempty"`
	UsersProfileGender       *Gender    `gorm:"type:varchar(10);column:users_profile_gender" json:"users_profile_gender,omitempty"`
	UsersProfileLocation     *string    `gorm:"size:100;column:users_profile_location" json:"users_profile_location,omitempty"`
	UsersProfilePhoneNumber  *string    `gorm:"size:20;column:users_profile_phone_number" json:"users_profile_phone_number,omitempty"`
	UsersProfileBio          *string    `gorm:"size:300;column:users_profile_bio" json:"users_profile_bio,omitempty"`

	// Konten panjang & riwayat
	UsersProfileBiographyLong *string `gorm:"type:text;column:users_profile_biography_long" json:"users_profile_biography_long,omitempty"`
	UsersProfileExperience    *string `gorm:"type:text;column:users_profile_experience" json:"users_profile_experience,omitempty"`
	UsersProfileCertifications *string `gorm:"type:text;column:users_profile_certifications" json:"users_profile_certifications,omitempty"`

	// Sosial media utama
	UsersProfileInstagramURL *string `gorm:"type:text;column:users_profile_instagram_url" json:"users_profile_instagram_url,omitempty"`
	UsersProfileWhatsappURL  *string `gorm:"type:text;column:users_profile_whatsapp_url" json:"users_profile_whatsapp_url,omitempty"`
	UsersProfileYoutubeURL   *string `gorm:"type:text;column:users_profile_youtube_url" json:"users_profile_youtube_url,omitempty"`
	UsersProfileFacebookURL  *string `gorm:"type:text;column:users_profile_facebook_url" json:"users_profile_facebook_url,omitempty"`
	UsersProfileTiktokURL    *string `gorm:"type:text;column:users_profile_tiktok_url" json:"users_profile_tiktok_url,omitempty"`

	// Sosial media tambahan
	UsersProfileTelegramUsername *string `gorm:"size:50;column:users_profile_telegram_username" json:"users_profile_telegram_username,omitempty"`
	UsersProfileLinkedinURL      *string `gorm:"type:text;column:users_profile_linkedin_url" json:"users_profile_linkedin_url,omitempty"`
	UsersProfileTwitterURL       *string `gorm:"type:text;column:users_profile_twitter_url" json:"users_profile_twitter_url,omitempty"`
	UsersProfileGithubURL        *string `gorm:"type:text;column:users_profile_github_url" json:"users_profile_github_url,omitempty"`

	// Privasi
	UsersProfileIsPublicProfile bool `gorm:"not null;default:true;column:users_profile_is_public_profile" json:"users_profile_is_public_profile"`

	// Verifikasi
	UsersProfileIsVerified bool        `gorm:"not null;default:false;column:users_profile_is_verified" json:"users_profile_is_verified"`
	UsersProfileVerifiedAt *time.Time  `gorm:"column:users_profile_verified_at" json:"users_profile_verified_at,omitempty"`
	UsersProfileVerifiedBy *uuid.UUID  `gorm:"type:uuid;column:users_profile_verified_by" json:"users_profile_verified_by,omitempty"`

	// Pendidikan & pekerjaan
	UsersProfileEducation *string `gorm:"type:text;column:users_profile_education" json:"users_profile_education,omitempty"`
	UsersProfileCompany   *string `gorm:"type:text;column:users_profile_company" json:"users_profile_company,omitempty"`
	UsersProfilePosition  *string `gorm:"type:text;column:users_profile_position" json:"users_profile_position,omitempty"`

	// Interests & skills (text[])
	UsersProfileInterests pq.StringArray `gorm:"type:text[];column:users_profile_interests" json:"users_profile_interests"`
	UsersProfileSkills    pq.StringArray `gorm:"type:text[];column:users_profile_skills" json:"users_profile_skills"`


	// Audit
	UsersProfileCreatedAt time.Time      `gorm:"autoCreateTime;column:users_profile_created_at" json:"users_profile_created_at"`
	UsersProfileUpdatedAt time.Time      `gorm:"column:users_profile_updated_at" json:"users_profile_updated_at"`
	UsersProfileDeletedAt gorm.DeletedAt `gorm:"column:users_profile_deleted_at;index" json:"users_profile_deleted_at,omitempty"`
}

func (UsersProfileModel) TableName() string { return "users_profile" }
