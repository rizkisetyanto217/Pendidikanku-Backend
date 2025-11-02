package dto

import (
	"schoolku_backend/internals/features/home/posts/model"
	"time"
)

// ============================
// Response DTO
// ============================
type PostDTO struct {
	PostID          string        `json:"post_id"`
	PostTitle       string        `json:"post_title"`
	PostContent     string        `json:"post_content"`
	PostImageURL    *string       `json:"post_image_url"`
	PostIsPublished bool          `json:"post_is_published"`
	PostType        string        `json:"post_type"`
	PostThemeID     *string       `json:"post_theme_id"`
	PostSchoolID    *string       `json:"post_school_id"`
	PostUserID      *string       `json:"post_user_id"`
	PostCreatedAt   time.Time     `json:"post_created_at"`
	PostUpdatedAt   time.Time     `json:"post_updated_at"`
	PostDeletedAt   *time.Time    `json:"post_deleted_at"`
	PostTheme       *PostThemeDTO `json:"post_theme,omitempty"`
	LikeCount       int64         `json:"like_count"`
	IsLikedByUser   bool          `json:"is_liked_by_user"`
}

// ============================
// Create Request DTO
// ============================
type CreatePostRequest struct {
	PostTitle       string  `json:"post_title" validate:"required,min=3"`
	PostContent     string  `json:"post_content" validate:"required"`
	PostImageURL    *string `json:"post_image_url"`
	PostIsPublished bool    `json:"post_is_published"`
	PostType        string  `json:"post_type" validate:"omitempty,oneof=text image video"`
	PostThemeID     *string `json:"post_theme_id" validate:"omitempty,uuid"`
	PostSchoolID    *string `json:"post_school_id"`
}

// ============================
// Update Request DTO
// ============================
type UpdatePostRequest struct {
	PostTitle       string  `json:"post_title" validate:"required,min=3"`
	PostContent     string  `json:"post_content" validate:"required"`
	PostImageURL    *string `json:"post_image_url"`
	PostIsPublished bool    `json:"post_is_published"`
	PostType        string  `json:"post_type" validate:"omitempty,oneof=text image video"`
	PostThemeID     *string `json:"post_theme_id" validate:"omitempty,uuid"`
}

// ============================
// Converters
// ============================

// Untuk endpoint publik/admin yang hanya perlu like count (tanpa status like user)
func ToPostDTO(m model.PostModel, likeCount int64) PostDTO {
	return PostDTO{
		PostID:          m.PostID,
		PostTitle:       m.PostTitle,
		PostContent:     m.PostContent,
		PostImageURL:    m.PostImageURL,
		PostIsPublished: m.PostIsPublished,
		PostType:        m.PostType,
		PostThemeID:     m.PostThemeID,
		PostSchoolID:    m.PostSchoolID,
		PostUserID:      m.PostUserID,
		PostCreatedAt:   m.PostCreatedAt,
		PostUpdatedAt:   m.PostUpdatedAt,
		PostDeletedAt:   m.PostDeletedAt,
		LikeCount:       likeCount,
		IsLikedByUser:   false,
	}
}

// Untuk endpoint publik yang butuh tema dan like count
func ToPostDTOWithTheme(m model.PostModel, theme *model.PostThemeModel, likeCount int64) PostDTO {
	return ToPostDTOFull(m, theme, likeCount, false)
}

// ✅ Fungsi utama: Untuk post lengkap + tema + like count + status like user
func ToPostDTOFull(m model.PostModel, theme *model.PostThemeModel, likeCount int64, isLiked bool) PostDTO {
	dto := PostDTO{
		PostID:          m.PostID,
		PostTitle:       m.PostTitle,
		PostContent:     m.PostContent,
		PostImageURL:    m.PostImageURL,
		PostIsPublished: m.PostIsPublished,
		PostType:        m.PostType,
		PostThemeID:     m.PostThemeID,
		PostSchoolID:    m.PostSchoolID,
		PostUserID:      m.PostUserID,
		PostCreatedAt:   m.PostCreatedAt,
		PostUpdatedAt:   m.PostUpdatedAt,
		PostDeletedAt:   m.PostDeletedAt,
		LikeCount:       likeCount,
		IsLikedByUser:   isLiked,
	}

	if theme != nil {
		dto.PostTheme = &PostThemeDTO{
			PostThemeID:          theme.PostThemeID,
			PostThemeName:        theme.PostThemeName,
			PostThemeDescription: theme.PostThemeDescription,
			PostThemeSchoolID:    theme.PostThemeSchoolID,
			PostThemeCreatedAt:   theme.PostThemeCreatedAt,
		}
	}

	return dto
}

// Untuk membuat model baru saat create post
func ToPostModel(req CreatePostRequest, userID *string) model.PostModel {
	return model.PostModel{
		PostTitle:       req.PostTitle,
		PostContent:     req.PostContent,
		PostImageURL:    req.PostImageURL,
		PostIsPublished: req.PostIsPublished,
		PostType:        req.PostType,
		PostThemeID:     req.PostThemeID,
		PostSchoolID:    req.PostSchoolID,
		PostUserID:      userID,
	}
}

// Untuk update post (direct ke model)
func UpdatePostModel(m *model.PostModel, req UpdatePostRequest) {
	m.PostTitle = req.PostTitle
	m.PostContent = req.PostContent
	m.PostImageURL = req.PostImageURL
	m.PostIsPublished = req.PostIsPublished
	m.PostType = req.PostType
	m.PostThemeID = req.PostThemeID
}
