// file: internals/features/social/posts/dto/post_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	smodel "schoolku_backend/internals/features/school/others/post/model"
)

/* ==============================
   CREATE (POST /posts)
============================== */

type CreatePostRequest struct {
	// Tenant
	PostSchoolID uuid.UUID `json:"post_school_id" validate:"required"`

	// Jenis
	PostKind smodel.PostKind `json:"post_kind" validate:"required,oneof=announcement material post other"`

	// Pengirim & relasi
	IsDKMSender            *bool      `json:"is_dkm_sender" validate:"omitempty"`
	PostCreatedByTeacherID *uuid.UUID `json:"post_created_by_teacher_id" validate:"omitempty"`
	PostClassSectionID     *uuid.UUID `json:"post_class_section_id" validate:"omitempty"`
	PostThemeID            *uuid.UUID `json:"post_theme_id" validate:"omitempty"`

	// Identitas & isi
	PostSlug    *string   `json:"post_slug" validate:"omitempty,max=160"`
	PostTitle   string    `json:"post_title" validate:"required,max=200"`
	PostDate    time.Time `json:"post_date" validate:"required"`
	PostContent string    `json:"post_content" validate:"required"`

	PostExcerpt *string          `json:"post_excerpt" validate:"omitempty"`
	PostMeta    *json.RawMessage `json:"post_meta" validate:"omitempty"`

	// Status & publish
	PostIsActive    *bool      `json:"post_is_active" validate:"omitempty"`
	PostIsPublished *bool      `json:"post_is_published" validate:"omitempty"`
	PostPublishedAt *time.Time `json:"post_published_at" validate:"omitempty"`

	// Snapshot audiens saat publish
	PostAudienceSnapshot *json.RawMessage `json:"post_audience_snapshot" validate:"omitempty"`

	// Target section (khusus announcement)
	PostSectionIDs []uuid.UUID `json:"post_section_ids" validate:"omitempty,dive,uuid"`
}

func (r *CreatePostRequest) ToModel() *smodel.Post {
	isActive := true
	if r.PostIsActive != nil {
		isActive = *r.PostIsActive
	}
	isPublished := false
	if r.PostIsPublished != nil {
		isPublished = *r.PostIsPublished
	}

	var meta datatypes.JSON
	if r.PostMeta != nil && len(*r.PostMeta) > 0 {
		meta = datatypes.JSON(*r.PostMeta)
	}
	var aud datatypes.JSON
	if r.PostAudienceSnapshot != nil && len(*r.PostAudienceSnapshot) > 0 {
		aud = datatypes.JSON(*r.PostAudienceSnapshot)
	}

	m := &smodel.Post{
		PostSchoolID: r.PostSchoolID,
		PostKind:     r.PostKind,

		IsDKMSender:            r.IsDKMSender != nil && *r.IsDKMSender,
		PostCreatedByTeacherID: r.PostCreatedByTeacherID,
		PostClassSectionID:     r.PostClassSectionID,
		PostThemeID:            r.PostThemeID,

		PostSlug:    trimPtr(r.PostSlug),
		PostTitle:   strings.TrimSpace(r.PostTitle),
		PostDate:    r.PostDate,
		PostContent: strings.TrimSpace(r.PostContent),

		PostExcerpt: trimPtr(r.PostExcerpt),
		PostMeta:    meta,

		PostIsActive:    isActive,
		PostIsPublished: isPublished,
		PostPublishedAt: r.PostPublishedAt,

		PostAudienceSnapshot: aud,

		PostSectionIDs: r.PostSectionIDs,
	}
	return m
}

/* ==============================
   PATCH (PATCH /posts/:id)
============================== */

type PatchPostRequest struct {
	PostKind UpdateField[smodel.PostKind] `json:"post_kind"`

	IsDKMSender            UpdateField[bool]      `json:"is_dkm_sender"`
	PostCreatedByTeacherID UpdateField[uuid.UUID] `json:"post_created_by_teacher_id"`
	PostClassSectionID     UpdateField[uuid.UUID] `json:"post_class_section_id"`
	PostThemeID            UpdateField[uuid.UUID] `json:"post_theme_id"`

	PostSlug    UpdateField[string]    `json:"post_slug"`
	PostTitle   UpdateField[string]    `json:"post_title"`
	PostDate    UpdateField[time.Time] `json:"post_date"`
	PostContent UpdateField[string]    `json:"post_content"`

	PostExcerpt UpdateField[string]          `json:"post_excerpt"`
	PostMeta    UpdateField[json.RawMessage] `json:"post_meta"`

	PostIsActive    UpdateField[bool]      `json:"post_is_active"`
	PostIsPublished UpdateField[bool]      `json:"post_is_published"`
	PostPublishedAt UpdateField[time.Time] `json:"post_published_at"`

	PostAudienceSnapshot UpdateField[json.RawMessage] `json:"post_audience_snapshot"`

	PostSectionIDs UpdateField[[]uuid.UUID] `json:"post_section_ids"`
}

// ToUpdates mengubah payload PATCH menjadi map untuk GORM .Updates(...)
func (p *PatchPostRequest) ToUpdates() map[string]any {
	u := make(map[string]any, 16)

	// Jenis
	if p.PostKind.ShouldUpdate() && !p.PostKind.IsNull() {
		u["post_kind"] = p.PostKind.Val()
	}

	// Pengirim & relasi
	if p.IsDKMSender.ShouldUpdate() && !p.IsDKMSender.IsNull() {
		u["is_dkm_sender"] = p.IsDKMSender.Val()
	}
	if p.PostCreatedByTeacherID.ShouldUpdate() {
		if p.PostCreatedByTeacherID.IsNull() {
			u["post_created_by_teacher_id"] = nil
		} else {
			u["post_created_by_teacher_id"] = p.PostCreatedByTeacherID.Val()
		}
	}
	if p.PostClassSectionID.ShouldUpdate() {
		if p.PostClassSectionID.IsNull() {
			u["post_class_section_id"] = nil
		} else {
			u["post_class_section_id"] = p.PostClassSectionID.Val()
		}
	}
	if p.PostThemeID.ShouldUpdate() {
		if p.PostThemeID.IsNull() {
			u["post_theme_id"] = nil
		} else {
			u["post_theme_id"] = p.PostThemeID.Val()
		}
	}

	// Identitas & isi
	if p.PostSlug.ShouldUpdate() {
		if p.PostSlug.IsNull() {
			u["post_slug"] = nil
		} else {
			s := strings.TrimSpace(p.PostSlug.Val())
			if s == "" {
				u["post_slug"] = nil
			} else {
				u["post_slug"] = s
			}
		}
	}
	if p.PostTitle.ShouldUpdate() && !p.PostTitle.IsNull() {
		if t := strings.TrimSpace(p.PostTitle.Val()); t != "" {
			u["post_title"] = t
		}
	}
	if p.PostDate.ShouldUpdate() && !p.PostDate.IsNull() {
		u["post_date"] = p.PostDate.Val()
	}
	if p.PostContent.ShouldUpdate() && !p.PostContent.IsNull() {
		if c := strings.TrimSpace(p.PostContent.Val()); c != "" {
			u["post_content"] = c
		}
	}

	if p.PostExcerpt.ShouldUpdate() {
		if p.PostExcerpt.IsNull() {
			u["post_excerpt"] = nil
		} else {
			if e := strings.TrimSpace(p.PostExcerpt.Val()); e == "" {
				u["post_excerpt"] = nil
			} else {
				u["post_excerpt"] = e
			}
		}
	}
	if p.PostMeta.ShouldUpdate() {
		if p.PostMeta.IsNull() {
			u["post_meta"] = nil
		} else {
			raw := p.PostMeta.Val()
			if len(raw) == 0 {
				u["post_meta"] = nil
			} else {
				u["post_meta"] = datatypes.JSON(raw)
			}
		}
	}

	// Status & publish
	if p.PostIsActive.ShouldUpdate() && !p.PostIsActive.IsNull() {
		u["post_is_active"] = p.PostIsActive.Val()
	}
	if p.PostIsPublished.ShouldUpdate() && !p.PostIsPublished.IsNull() {
		u["post_is_published"] = p.PostIsPublished.Val()
	}
	if p.PostPublishedAt.ShouldUpdate() {
		if p.PostPublishedAt.IsNull() {
			u["post_published_at"] = nil
		} else {
			u["post_published_at"] = p.PostPublishedAt.Val()
		}
	}

	// Snapshot audiens
	if p.PostAudienceSnapshot.ShouldUpdate() {
		if p.PostAudienceSnapshot.IsNull() {
			u["post_audience_snapshot"] = nil
		} else {
			raw := p.PostAudienceSnapshot.Val()
			if len(raw) == 0 {
				u["post_audience_snapshot"] = nil
			} else {
				u["post_audience_snapshot"] = datatypes.JSON(raw)
			}
		}
	}

	// Target section
	if p.PostSectionIDs.ShouldUpdate() {
		if p.PostSectionIDs.IsNull() {
			u["post_section_ids"] = nil
		} else {
			u["post_section_ids"] = p.PostSectionIDs.Val()
		}
	}

	return u
}

/* ==============================
   RESPONSES
============================== */

type PostResponse struct {
	PostID       uuid.UUID `json:"post_id"`
	PostSchoolID uuid.UUID `json:"post_school_id"`

	PostKind smodel.PostKind `json:"post_kind"`

	IsDKMSender            bool       `json:"is_dkm_sender"`
	PostCreatedByTeacherID *uuid.UUID `json:"post_created_by_teacher_id,omitempty"`
	PostClassSectionID     *uuid.UUID `json:"post_class_section_id,omitempty"`
	PostThemeID            *uuid.UUID `json:"post_theme_id,omitempty"`

	PostSlug    *string   `json:"post_slug,omitempty"`
	PostTitle   string    `json:"post_title"`
	PostDate    time.Time `json:"post_date"`
	PostContent string    `json:"post_content"`

	PostExcerpt *string          `json:"post_excerpt,omitempty"`
	PostMeta    *json.RawMessage `json:"post_meta,omitempty"`

	PostIsActive    bool       `json:"post_is_active"`
	PostCreatedAt   time.Time  `json:"post_created_at"`
	PostUpdatedAt   time.Time  `json:"post_updated_at"`
	PostDeletedAt   *time.Time `json:"post_deleted_at,omitempty"`
	PostIsPublished bool       `json:"post_is_published"`
	PostPublishedAt *time.Time `json:"post_published_at,omitempty"`

	PostAudienceSnapshot *json.RawMessage `json:"post_audience_snapshot,omitempty"`

	PostSearch *string `json:"post_search,omitempty"`

	PostSectionIDs []uuid.UUID `json:"post_section_ids"`
}

type ListPostResponse struct {
	Data   []PostResponse `json:"data"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

/* ==============================
   MAPPERS
============================== */

func FromModelPost(m *smodel.Post) PostResponse {
	var deletedAt *time.Time
	if m.PostDeletedAt.Valid {
		t := m.PostDeletedAt.Time
		deletedAt = &t
	}
	var meta *json.RawMessage
	if len(m.PostMeta) > 0 {
		tmp := json.RawMessage(m.PostMeta)
		meta = &tmp
	}
	var aud *json.RawMessage
	if len(m.PostAudienceSnapshot) > 0 {
		tmp := json.RawMessage(m.PostAudienceSnapshot)
		aud = &tmp
	}
	var search *string
	if s := strings.TrimSpace(m.PostSearch); s != "" {
		search = &s
	}

	return PostResponse{
		PostID:       m.PostID,
		PostSchoolID: m.PostSchoolID,

		PostKind: m.PostKind,

		IsDKMSender:            m.IsDKMSender,
		PostCreatedByTeacherID: m.PostCreatedByTeacherID,
		PostClassSectionID:     m.PostClassSectionID,
		PostThemeID:            m.PostThemeID,

		PostSlug:    m.PostSlug,
		PostTitle:   m.PostTitle,
		PostDate:    m.PostDate,
		PostContent: m.PostContent,

		PostExcerpt:          m.PostExcerpt,
		PostMeta:             meta,
		PostIsActive:         m.PostIsActive,
		PostCreatedAt:        m.PostCreatedAt,
		PostUpdatedAt:        m.PostUpdatedAt,
		PostDeletedAt:        deletedAt,
		PostIsPublished:      m.PostIsPublished,
		PostPublishedAt:      m.PostPublishedAt,
		PostAudienceSnapshot: aud,
		PostSearch:           search,

		PostSectionIDs: m.PostSectionIDs,
	}
}

func FromModelsPost(items []smodel.Post) []PostResponse {
	out := make([]PostResponse, 0, len(items))
	for i := range items {
		out = append(out, FromModelPost(&items[i]))
	}
	return out
}
