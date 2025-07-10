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
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	DonationName string         `gorm:"size:50" json:"donation_name"`
	FullName     string         `gorm:"size:50" json:"full_name"`
	DateOfBirth  *time.Time     `json:"date_of_birth" time_format:"2006-01-02"`
	Gender       *Gender         `gorm:"size:10" json:"gender,omitempty"`
	PhoneNumber  string         `gorm:"size:20" json:"phone_number"`
	Bio          string         `gorm:"size:300" json:"bio"`
	Location     string         `gorm:"size:50" json:"location"`
	Occupation   string         `gorm:"size:20" json:"occupation"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Pastikan tabel bernama `users_profile`
func (UsersProfileModel) TableName() string {
	return "users_profile"
}