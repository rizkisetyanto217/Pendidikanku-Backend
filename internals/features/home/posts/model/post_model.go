package model

import (
	MasjidModel "masjidku_backend/internals/features/masjids/masjids/model"
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type PostModel struct {
	PostID          string     `gorm:"column:post_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	PostTitle       string     `gorm:"column:post_title;type:varchar(255);not null"`
	PostContent     string     `gorm:"column:post_content;type:text;not null"`
	PostImageURL    *string    `gorm:"column:post_image_url;type:text"`
	PostIsPublished bool       `gorm:"column:post_is_published;default:false"`
	PostType        string     `gorm:"column:post_type;type:varchar(50);default:'text'"`
	PostMasjidID    *string    `gorm:"column:post_masjid_id;type:uuid"`
	PostUserID      *string    `gorm:"column:post_user_id;type:uuid"`
	PostCreatedAt   time.Time  `gorm:"column:post_created_at;autoCreateTime"`
	PostUpdatedAt   time.Time  `gorm:"column:post_updated_at;autoUpdateTime"`
	PostDeletedAt   *time.Time `gorm:"column:post_deleted_at"`

	// Relations
	Masjid *MasjidModel.MasjidModel `gorm:"foreignKey:PostMasjidID"`
	User   *UserModel.UserModel     `gorm:"foreignKey:PostUserID"`
	Likes  []PostLikeModel          `gorm:"foreignKey:PostLikePostID"`
}

func (PostModel) TableName() string {
	return "posts"
}
