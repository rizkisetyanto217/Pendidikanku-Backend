package model

import (
	SchoolModel "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
	UserModel "schoolku_backend/internals/features/users/users/model"

	"time"
)

type PostModel struct {
	PostID          string  `gorm:"column:post_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	PostTitle       string  `gorm:"column:post_title;type:varchar(255);not null"`
	PostContent     string  `gorm:"column:post_content;type:text;not null"`
	PostImageURL    *string `gorm:"column:post_image_url;type:text"`
	PostIsPublished bool    `gorm:"column:post_is_published;default:false"`
	PostType        string  `gorm:"column:post_type;type:varchar(50);default:'text'"`

	PostThemeID  *string `gorm:"column:post_theme_id;type:uuid"`
	PostSchoolID *string `gorm:"column:post_school_id;type:uuid"`
	PostUserID   *string `gorm:"column:post_user_id;type:uuid"`

	PostCreatedAt time.Time  `gorm:"column:post_created_at;autoCreateTime"`
	PostUpdatedAt time.Time  `gorm:"column:post_updated_at;autoUpdateTime"`
	PostDeletedAt *time.Time `gorm:"column:post_deleted_at"`

	// Relations
	School *SchoolModel.SchoolModel `gorm:"foreignKey:PostSchoolID"`
	User   *UserModel.UserModel     `gorm:"foreignKey:PostUserID"`
	// Theme  *PostThemeModel.PostThemeModel `gorm:"foreignKey:PostThemeID"`
	Likes []PostLikeModel `gorm:"foreignKey:PostLikePostID"`
}

func (PostModel) TableName() string {
	return "posts"
}
