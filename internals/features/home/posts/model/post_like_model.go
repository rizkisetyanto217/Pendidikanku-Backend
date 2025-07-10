package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type PostLikeModel struct {
	PostLikeID        string    `gorm:"column:post_like_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	PostLikeIsLiked   bool      `gorm:"column:post_like_is_liked;default:true"`
	PostLikePostID    string    `gorm:"column:post_like_post_id;type:uuid;not null"`
	PostLikeUserID    string    `gorm:"column:post_like_user_id;type:uuid;not null"`
	PostLikeUpdatedAt time.Time `gorm:"column:post_like_updated_at;autoUpdateTime"`

	// Relations
	User *UserModel.UserModel `gorm:"foreignKey:PostLikeUserID"`
	// Post *PostModel `gorm:"foreignKey:PostLikePostID"` // Uncomment if you want back relation
}

func (PostLikeModel) TableName() string {
	return "post_likes"
}
