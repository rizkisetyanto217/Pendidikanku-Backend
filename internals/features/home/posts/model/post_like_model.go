package model

import (
	SchoolModel "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
	UserModel "schoolku_backend/internals/features/users/users/model"
	"time"
)

type PostLikeModel struct {
	PostLikeID        string    `gorm:"column:post_like_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"post_like_id"`
	PostLikeIsLiked   bool      `gorm:"column:post_like_is_liked;default:true" json:"post_like_is_liked"`
	PostLikePostID    string    `gorm:"column:post_like_post_id;type:uuid;not null" json:"post_like_post_id"`
	PostLikeUserID    string    `gorm:"column:post_like_user_id;type:uuid;not null" json:"post_like_user_id"`
	PostLikeSchoolID  string    `gorm:"column:post_like_school_id;type:uuid;not null" json:"post_like_school_id"` // ✅ baru ditambahkan
	PostLikeUpdatedAt time.Time `gorm:"column:post_like_updated_at;autoUpdateTime" json:"post_like_updated_at"`

	// Relations
	User   *UserModel.UserModel     `gorm:"foreignKey:PostLikeUserID"`
	School *SchoolModel.SchoolModel `gorm:"foreignKey:PostLikeSchoolID"`
	// Post *PostModel              `gorm:"foreignKey:PostLikePostID"` // Uncomment if needed
}

func (PostLikeModel) TableName() string {
	return "post_likes"
}
