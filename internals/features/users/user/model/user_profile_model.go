package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Gender string

const (
	Male   Gender = "male"
	Female Gender = "female"
)

type UsersProfileModel struct {
	// PK
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// FK & Unique
	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_users_profile_user_id" json:"user_id"`

	// Columns
	DonationName            string     `gorm:"size:50;column:donation_name" json:"donation_name"`
	PhotoURL                *string    `gorm:"size:255;column:photo_url" json:"photo_url,omitempty"`
	PhotoTrashURL           *string    `gorm:"type:text;column:photo_trash_url" json:"photo_trash_url,omitempty"`
	PhotoDeletePendingUntil *time.Time `gorm:"column:photo_delete_pending_until" json:"photo_delete_pending_until,omitempty"`
	DateOfBirth             *time.Time `gorm:"type:date;column:date_of_birth" json:"date_of_birth,omitempty"`
	Gender                  *Gender    `gorm:"type:varchar(10);column:gender;index:idx_users_profile_gender" json:"gender,omitempty"`
	Location                *string    `gorm:"size:100;column:location" json:"location,omitempty"`
	Occupation              *string    `gorm:"size:50;column:occupation" json:"occupation,omitempty"`
	PhoneNumber             *string    `gorm:"size:20;column:phone_number;index:idx_users_profile_phone" json:"phone_number,omitempty"`
	Bio                     *string    `gorm:"size:300;column:bio" json:"bio,omitempty"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	// Biarkan trigger DB yang mengisi updated_at (tanpa autoUpdateTime agar tidak bentrok)
	UpdatedAt *time.Time     `gorm:"column:updated_at" json:"updated_at,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deleted_at" json:"deleted_at,omitempty"`
}

func (UsersProfileModel) TableName() string { return "users_profile" }
