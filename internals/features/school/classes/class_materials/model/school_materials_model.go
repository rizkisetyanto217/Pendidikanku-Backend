package model

import (
	"time"

	"github.com/google/uuid"
)

/* =======================================================
   school_materials
======================================================= */

type SchoolMaterialModel struct {
	SchoolMaterialID uuid.UUID `json:"school_material_id" gorm:"column:school_material_id;type:uuid;default:gen_random_uuid();primaryKey"`

	SchoolMaterialSchoolID       uuid.UUID  `json:"school_material_school_id" gorm:"column:school_material_school_id;type:uuid;not null"`
	SchoolMaterialClassSubjectID *uuid.UUID `json:"school_material_class_subject_id" gorm:"column:school_material_class_subject_id;type:uuid"`

	SchoolMaterialCreatedByUserID *uuid.UUID `json:"school_material_created_by_user_id" gorm:"column:school_material_created_by_user_id;type:uuid"`

	SchoolMaterialTitle       string  `json:"school_material_title" gorm:"column:school_material_title;type:text;not null"`
	SchoolMaterialDescription *string `json:"school_material_description" gorm:"column:school_material_description;type:text"`

	SchoolMaterialType MaterialType `json:"school_material_type" gorm:"column:school_material_type;type:material_type_enum;not null"`

	// artikel (rich text)
	SchoolMaterialContentHTML *string `json:"school_material_content_html" gorm:"column:school_material_content_html;type:text"`

	// file upload
	SchoolMaterialFileURL       *string `json:"school_material_file_url" gorm:"column:school_material_file_url;type:text"`
	SchoolMaterialFileName      *string `json:"school_material_file_name" gorm:"column:school_material_file_name;type:text"`
	SchoolMaterialFileMimeType  *string `json:"school_material_file_mime_type" gorm:"column:school_material_file_mime_type;type:text"`
	SchoolMaterialFileSizeBytes *int64  `json:"school_material_file_size_bytes" gorm:"column:school_material_file_size_bytes"`

	// link / embed / YouTube
	SchoolMaterialExternalURL *string `json:"school_material_external_url" gorm:"column:school_material_external_url;type:text"`
	SchoolMaterialYouTubeID   *string `json:"school_material_youtube_id" gorm:"column:school_material_youtube_id;type:text"`
	SchoolMaterialDurationSec *int32  `json:"school_material_duration_sec" gorm:"column:school_material_duration_sec"`

	SchoolMaterialImportance        MaterialImportance `json:"school_material_importance" gorm:"column:school_material_importance;type:material_importance_enum;not null;default:'important'"`
	SchoolMaterialIsRequiredForPass bool               `json:"school_material_is_required_for_pass" gorm:"column:school_material_is_required_for_pass;not null;default:false"`
	SchoolMaterialAffectsScoring    bool               `json:"school_material_affects_scoring" gorm:"column:school_material_affects_scoring;not null;default:false"`

	SchoolMaterialMeetingNumber *int32 `json:"school_material_meeting_number" gorm:"column:school_material_meeting_number"`
	SchoolMaterialDefaultOrder  *int32 `json:"school_material_default_order" gorm:"column:school_material_default_order"`

	SchoolMaterialScopeTag *string `json:"school_material_scope_tag" gorm:"column:school_material_scope_tag;type:text"`

	SchoolMaterialIsActive    bool       `json:"school_material_is_active" gorm:"column:school_material_is_active;not null;default:true"`
	SchoolMaterialIsPublished bool       `json:"school_material_is_published" gorm:"column:school_material_is_published;not null;default:false"`
	SchoolMaterialPublishedAt *time.Time `json:"school_material_published_at" gorm:"column:school_material_published_at"`

	SchoolMaterialDeleted   bool       `json:"school_material_deleted" gorm:"column:school_material_deleted;not null;default:false"`
	SchoolMaterialDeletedAt *time.Time `json:"school_material_deleted_at" gorm:"column:school_material_deleted_at"`

	SchoolMaterialCreatedAt time.Time `json:"school_material_created_at" gorm:"column:school_material_created_at;not null;default:now()"`
	SchoolMaterialUpdatedAt time.Time `json:"school_material_updated_at" gorm:"column:school_material_updated_at;not null;default:now()"`
}

func (SchoolMaterialModel) TableName() string {
	return "school_materials"
}
