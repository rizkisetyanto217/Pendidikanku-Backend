package dto

import (
	"masjidku_backend/internals/features/home/posts/model"
	"time"
)

// ============================
// Response DTO
// ============================
type PostLikeDTO struct {
	PostLikeID      string    `json:"post_like_id"`
	PostLikeIsLiked bool      `json:"post_like_is_liked"`
	PostLikePostID  string    `json:"post_like_post_id"`
	PostLikeUserID  string    `json:"post_like_user_id"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ============================
// Create or Toggle Request DTO
// ============================
type ToggleLikeRequest struct {
	PostID string `json:"post_id" validate:"required,uuid"`
}

// ============================
// Converter
// ============================
func ToPostLikeDTO(m model.PostLikeModel) PostLikeDTO {
	return PostLikeDTO{
		PostLikeID:      m.PostLikeID,
		PostLikeIsLiked: m.PostLikeIsLiked,
		PostLikePostID:  m.PostLikePostID,
		PostLikeUserID:  m.PostLikeUserID,
		UpdatedAt:       m.PostLikeUpdatedAt,
	}
}
