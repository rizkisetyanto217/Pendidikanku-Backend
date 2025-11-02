package model

import (
	"time"

	// School "schoolku_backend/internals/features/schools/schools/model"
	// User "schoolku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
)

type UserFollowSchoolModel struct {
	// PK komposit
	UserFollowSchoolUserID   uuid.UUID `gorm:"column:user_follow_school_user_id;type:uuid;not null;primaryKey"   json:"user_follow_school_user_id"`
	UserFollowSchoolSchoolID uuid.UUID `gorm:"column:user_follow_school_school_id;type:uuid;not null;primaryKey" json:"user_follow_school_school_id"`

	// Timestamp
	UserFollowSchoolCreatedAt time.Time `gorm:"column:user_follow_school_created_at;not null;autoCreateTime" json:"user_follow_school_created_at"`

	// Relasi (opsional, kalau mau di-preload)
	// User   User.UserModel    `gorm:"foreignKey:UserFollowSchoolUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"         json:"user,omitempty"`
	// School School.SchoolModel`gorm:"foreignKey:UserFollowSchoolSchoolID;references:SchoolID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"school,omitempty"`
}

func (UserFollowSchoolModel) TableName() string {
	return "user_follow_school"
}
