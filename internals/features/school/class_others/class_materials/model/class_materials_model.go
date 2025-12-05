// file: internals/features/school/materials/model/class_materials_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

/* =========================================================
   Enums (mirror Postgres ENUM)
========================================================= */

type MaterialType string

const (
	MaterialTypeArticle   MaterialType = "article"
	MaterialTypeDoc       MaterialType = "doc"
	MaterialTypePPT       MaterialType = "ppt"
	MaterialTypePDF       MaterialType = "pdf"
	MaterialTypeImage     MaterialType = "image"
	MaterialTypeYouTube   MaterialType = "youtube"
	MaterialTypeVideoFile MaterialType = "video_file"
	MaterialTypeLink      MaterialType = "link"
	MaterialTypeEmbed     MaterialType = "embed"
)

type MaterialImportance string

const (
	MaterialImportanceImportant  MaterialImportance = "important"
	MaterialImportanceAdditional MaterialImportance = "additional"
	MaterialImportanceOptional   MaterialImportance = "optional"
)

/* =========================================================
   ClassMaterialsModel (tabel: class_materials)
========================================================= */

type ClassMaterialsModel struct {
	// PK + tenant
	ClassMaterialID       uuid.UUID `gorm:"column:class_material_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_material_id"`
	ClassMaterialSchoolID uuid.UUID `gorm:"column:class_material_school_id;type:uuid;not null" json:"class_material_school_id"`
	ClassMaterialCSSTID   uuid.UUID `gorm:"column:class_material_csst_id;type:uuid;not null" json:"class_material_csst_id"`

	// creator
	ClassMaterialCreatedByUserID *uuid.UUID `gorm:"column:class_material_created_by_user_id;type:uuid" json:"class_material_created_by_user_id"`

	// konten utama
	ClassMaterialTitle       string             `gorm:"column:class_material_title;type:text;not null" json:"class_material_title"`
	ClassMaterialDescription *string            `gorm:"column:class_material_description;type:text" json:"class_material_description"`
	ClassMaterialType        MaterialType       `gorm:"column:class_material_type;type:material_type_enum;not null" json:"class_material_type"`
	ClassMaterialContentHTML *string            `gorm:"column:class_material_content_html;type:text" json:"class_material_content_html"`
	ClassMaterialImportance  MaterialImportance `gorm:"column:class_material_importance;type:material_importance_enum;not null;default:important" json:"class_material_importance"`

	// file upload (doc/ppt/pdf/image/video)
	ClassMaterialFileURL       *string `gorm:"column:class_material_file_url;type:text" json:"class_material_file_url"`
	ClassMaterialFileName      *string `gorm:"column:class_material_file_name;type:text" json:"class_material_file_name"`
	ClassMaterialFileMIMEType  *string `gorm:"column:class_material_file_mime_type;type:text" json:"class_material_file_mime_type"`
	ClassMaterialFileSizeBytes *int64  `gorm:"column:class_material_file_size_bytes" json:"class_material_file_size_bytes"`

	// link / embed / YouTube
	ClassMaterialExternalURL *string `gorm:"column:class_material_external_url;type:text" json:"class_material_external_url"`
	ClassMaterialYouTubeID   *string `gorm:"column:class_material_youtube_id;type:text" json:"class_material_youtube_id"`
	ClassMaterialDurationSec *int    `gorm:"column:class_material_duration_sec" json:"class_material_duration_sec"`

	// rules / policy flags
	ClassMaterialIsRequiredForPass bool `gorm:"column:class_material_is_required_for_pass;not null;default:false" json:"class_material_is_required_for_pass"`
	ClassMaterialAffectsScoring    bool `gorm:"column:class_material_affects_scoring;not null;default:false" json:"class_material_affects_scoring"`

	// meeting & session
	ClassMaterialMeetingNumber *int       `gorm:"column:class_material_meeting_number" json:"class_material_meeting_number"`
	ClassMaterialSessionID     *uuid.UUID `gorm:"column:class_material_session_id;type:uuid" json:"class_material_session_id"`

	// source info
	// contoh nilai: "school" | "teacher"
	ClassMaterialSourceKind             *string    `gorm:"column:class_material_source_kind;type:text" json:"class_material_source_kind"`
	ClassMaterialSourceSchoolMaterialID *uuid.UUID `gorm:"column:class_material_source_school_material_id;type:uuid" json:"class_material_source_school_material_id"`

	// order & status
	ClassMaterialOrder       *int       `gorm:"column:class_material_order" json:"class_material_order"`
	ClassMaterialIsActive    bool       `gorm:"column:class_material_is_active;not null;default:true" json:"class_material_is_active"`
	ClassMaterialIsPublished bool       `gorm:"column:class_material_is_published;not null;default:false" json:"class_material_is_published"`
	ClassMaterialPublishedAt *time.Time `gorm:"column:class_material_published_at" json:"class_material_published_at"`
	ClassMaterialDeleted     bool       `gorm:"column:class_material_deleted;not null;default:false" json:"class_material_deleted"`
	ClassMaterialDeletedAt   *time.Time `gorm:"column:class_material_deleted_at" json:"class_material_deleted_at"`
	ClassMaterialCreatedAt   time.Time  `gorm:"column:class_material_created_at;not null;default:now()" json:"class_material_created_at"`
	ClassMaterialUpdatedAt   time.Time  `gorm:"column:class_material_updated_at;not null;default:now()" json:"class_material_updated_at"`
}

/* =========================================================
   TableName override
========================================================= */

func (ClassMaterialsModel) TableName() string {
	return "class_materials"
}
