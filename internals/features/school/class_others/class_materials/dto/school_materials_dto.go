// file: internals/features/school/materials/dto/school_material_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	materialModel "madinahsalam_backend/internals/features/school/class_others/class_materials/model"
)

/* =======================================================
   RESPONSE DTO
   - Dipakai untuk list & detail
======================================================= */

type SchoolMaterialResponse struct {
	SchoolMaterialID uuid.UUID `json:"school_material_id"`

	SchoolMaterialSchoolID       uuid.UUID  `json:"school_material_school_id"`
	SchoolMaterialClassSubjectID *uuid.UUID `json:"school_material_class_subject_id"`

	SchoolMaterialCreatedByUserID *uuid.UUID `json:"school_material_created_by_user_id"`

	SchoolMaterialTitle       string  `json:"school_material_title"`
	SchoolMaterialDescription *string `json:"school_material_description"`

	SchoolMaterialType string `json:"school_material_type"`

	// artikel (rich text)
	SchoolMaterialContentHTML *string `json:"school_material_content_html"`

	// file upload
	SchoolMaterialFileURL       *string `json:"school_material_file_url"`
	SchoolMaterialFileName      *string `json:"school_material_file_name"`
	SchoolMaterialFileMimeType  *string `json:"school_material_file_mime_type"`
	SchoolMaterialFileSizeBytes *int64  `json:"school_material_file_size_bytes"`

	// link / embed / YouTube
	SchoolMaterialExternalURL *string `json:"school_material_external_url"`
	SchoolMaterialYouTubeID   *string `json:"school_material_youtube_id"`
	SchoolMaterialDurationSec *int32  `json:"school_material_duration_sec"`

	SchoolMaterialImportance        string `json:"school_material_importance"`
	SchoolMaterialIsRequiredForPass bool   `json:"school_material_is_required_for_pass"`
	SchoolMaterialAffectsScoring    bool   `json:"school_material_affects_scoring"`

	SchoolMaterialMeetingNumber *int32 `json:"school_material_meeting_number"`
	SchoolMaterialDefaultOrder  *int32 `json:"school_material_default_order"`

	SchoolMaterialScopeTag *string `json:"school_material_scope_tag"`

	SchoolMaterialIsActive    bool       `json:"school_material_is_active"`
	SchoolMaterialIsPublished bool       `json:"school_material_is_published"`
	SchoolMaterialPublishedAt *time.Time `json:"school_material_published_at"`

	SchoolMaterialDeleted   bool       `json:"school_material_deleted"`
	SchoolMaterialDeletedAt *time.Time `json:"school_material_deleted_at"`

	SchoolMaterialCreatedAt time.Time `json:"school_material_created_at"`
	SchoolMaterialUpdatedAt time.Time `json:"school_material_updated_at"`
}

/* =======================================================
   CREATE REQUEST DTO
   - Dipakai di controller saat CREATE
   - school_id & created_by diambil dari context/token
======================================================= */

type SchoolMaterialCreateRequest struct {
	SchoolMaterialClassSubjectID *uuid.UUID `json:"school_material_class_subject_id"`

	SchoolMaterialTitle       string  `json:"school_material_title"`
	SchoolMaterialDescription *string `json:"school_material_description"`

	SchoolMaterialType       string `json:"school_material_type"`       // article/pdf/image/...
	SchoolMaterialImportance string `json:"school_material_importance"` // important/additional/optional

	SchoolMaterialIsRequiredForPass *bool `json:"school_material_is_required_for_pass"`
	SchoolMaterialAffectsScoring    *bool `json:"school_material_affects_scoring"`

	// artikel (rich text)
	SchoolMaterialContentHTML *string `json:"school_material_content_html"`

	// file upload
	SchoolMaterialFileURL       *string `json:"school_material_file_url"`
	SchoolMaterialFileName      *string `json:"school_material_file_name"`
	SchoolMaterialFileMimeType  *string `json:"school_material_file_mime_type"`
	SchoolMaterialFileSizeBytes *int64  `json:"school_material_file_size_bytes"`

	// link / embed / YouTube
	SchoolMaterialExternalURL *string `json:"school_material_external_url"`
	SchoolMaterialYouTubeID   *string `json:"school_material_youtube_id"`
	SchoolMaterialDurationSec *int32  `json:"school_material_duration_sec"`

	// struktur kurikulum
	SchoolMaterialMeetingNumber *int32 `json:"school_material_meeting_number"`
	SchoolMaterialDefaultOrder  *int32 `json:"school_material_default_order"`

	// scope / tagging
	SchoolMaterialScopeTag *string `json:"school_material_scope_tag"`
}

/* =======================================================
   UPDATE REQUEST DTO
   - Dipakai di controller saat UPDATE (PATCH-style)
======================================================= */

type SchoolMaterialUpdateRequest struct {
	SchoolMaterialClassSubjectID *uuid.UUID `json:"school_material_class_subject_id"`

	SchoolMaterialTitle       *string `json:"school_material_title"`
	SchoolMaterialDescription *string `json:"school_material_description"`

	SchoolMaterialType       *string `json:"school_material_type"`
	SchoolMaterialImportance *string `json:"school_material_importance"`

	SchoolMaterialIsRequiredForPass *bool `json:"school_material_is_required_for_pass"`
	SchoolMaterialAffectsScoring    *bool `json:"school_material_affects_scoring"`

	SchoolMaterialContentHTML *string `json:"school_material_content_html"`

	SchoolMaterialFileURL       *string `json:"school_material_file_url"`
	SchoolMaterialFileName      *string `json:"school_material_file_name"`
	SchoolMaterialFileMimeType  *string `json:"school_material_file_mime_type"`
	SchoolMaterialFileSizeBytes *int64  `json:"school_material_file_size_bytes"`

	SchoolMaterialExternalURL *string `json:"school_material_external_url"`
	SchoolMaterialYouTubeID   *string `json:"school_material_youtube_id"`
	SchoolMaterialDurationSec *int32  `json:"school_material_duration_sec"`

	SchoolMaterialMeetingNumber *int32 `json:"school_material_meeting_number"`
	SchoolMaterialDefaultOrder  *int32 `json:"school_material_default_order"`

	SchoolMaterialScopeTag *string `json:"school_material_scope_tag"`

	SchoolMaterialIsActive    *bool `json:"school_material_is_active"`
	SchoolMaterialIsPublished *bool `json:"school_material_is_published"`
}

/* =======================================================
   Helpers: enum parsing
======================================================= */

func parseMaterialType(s string) materialModel.MaterialType {
	switch materialModel.MaterialType(s) {
	case materialModel.MaterialTypeArticle,
		materialModel.MaterialTypeDoc,
		materialModel.MaterialTypePPT,
		materialModel.MaterialTypePDF,
		materialModel.MaterialTypeImage,
		materialModel.MaterialTypeYouTube,
		materialModel.MaterialTypeVideoFile,
		materialModel.MaterialTypeLink,
		materialModel.MaterialTypeEmbed:
		return materialModel.MaterialType(s)
	default:
		// default aman (boleh kamu ganti ke PDF / Article)
		return materialModel.MaterialTypeArticle
	}
}

func parseMaterialImportance(s string) materialModel.MaterialImportance {
	switch materialModel.MaterialImportance(s) {
	case materialModel.MaterialImportanceImportant,
		materialModel.MaterialImportanceAdditional,
		materialModel.MaterialImportanceOptional:
		return materialModel.MaterialImportance(s)
	default:
		return materialModel.MaterialImportanceImportant
	}
}

/* =======================================================
   Mapper: Model -> Response DTO
======================================================= */

func NewSchoolMaterialResponse(m *materialModel.SchoolMaterialModel) *SchoolMaterialResponse {
	if m == nil {
		return nil
	}

	return &SchoolMaterialResponse{
		SchoolMaterialID:                m.SchoolMaterialID,
		SchoolMaterialSchoolID:          m.SchoolMaterialSchoolID,
		SchoolMaterialClassSubjectID:    m.SchoolMaterialClassSubjectID,
		SchoolMaterialCreatedByUserID:   m.SchoolMaterialCreatedByUserID,
		SchoolMaterialTitle:             m.SchoolMaterialTitle,
		SchoolMaterialDescription:       m.SchoolMaterialDescription,
		SchoolMaterialType:              string(m.SchoolMaterialType),
		SchoolMaterialContentHTML:       m.SchoolMaterialContentHTML,
		SchoolMaterialFileURL:           m.SchoolMaterialFileURL,
		SchoolMaterialFileName:          m.SchoolMaterialFileName,
		SchoolMaterialFileMimeType:      m.SchoolMaterialFileMimeType,
		SchoolMaterialFileSizeBytes:     m.SchoolMaterialFileSizeBytes,
		SchoolMaterialExternalURL:       m.SchoolMaterialExternalURL,
		SchoolMaterialYouTubeID:         m.SchoolMaterialYouTubeID,
		SchoolMaterialDurationSec:       m.SchoolMaterialDurationSec,
		SchoolMaterialImportance:        string(m.SchoolMaterialImportance),
		SchoolMaterialIsRequiredForPass: m.SchoolMaterialIsRequiredForPass,
		SchoolMaterialAffectsScoring:    m.SchoolMaterialAffectsScoring,
		SchoolMaterialMeetingNumber:     m.SchoolMaterialMeetingNumber,
		SchoolMaterialDefaultOrder:      m.SchoolMaterialDefaultOrder,
		SchoolMaterialScopeTag:          m.SchoolMaterialScopeTag,
		SchoolMaterialIsActive:          m.SchoolMaterialIsActive,
		SchoolMaterialIsPublished:       m.SchoolMaterialIsPublished,
		SchoolMaterialPublishedAt:       m.SchoolMaterialPublishedAt,
		SchoolMaterialDeleted:           m.SchoolMaterialDeleted,
		SchoolMaterialDeletedAt:         m.SchoolMaterialDeletedAt,
		SchoolMaterialCreatedAt:         m.SchoolMaterialCreatedAt,
		SchoolMaterialUpdatedAt:         m.SchoolMaterialUpdatedAt,
	}
}

func NewSchoolMaterialResponseList(list []*materialModel.SchoolMaterialModel) []*SchoolMaterialResponse {
	out := make([]*SchoolMaterialResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewSchoolMaterialResponse(m))
	}
	return out
}

/* =======================================================
   Mapper: CreateRequest -> Model (untuk INSERT)
   - schoolID & createdByUserID diisi dari context/token
   - now dari dbtime.GetDBTime(c)
======================================================= */

func (req *SchoolMaterialCreateRequest) ToModel(
	schoolID uuid.UUID,
	createdByUserID *uuid.UUID,
	now time.Time, // pakai waktu dari dbtime helper
) *materialModel.SchoolMaterialModel {
	if req == nil {
		return nil
	}

	isRequired := false
	if req.SchoolMaterialIsRequiredForPass != nil {
		isRequired = *req.SchoolMaterialIsRequiredForPass
	}

	affectsScoring := false
	if req.SchoolMaterialAffectsScoring != nil {
		affectsScoring = *req.SchoolMaterialAffectsScoring
	}

	imp := req.SchoolMaterialImportance
	if imp == "" {
		imp = string(materialModel.MaterialImportanceImportant)
	}

	return &materialModel.SchoolMaterialModel{
		SchoolMaterialSchoolID:        schoolID,
		SchoolMaterialClassSubjectID:  req.SchoolMaterialClassSubjectID,
		SchoolMaterialCreatedByUserID: createdByUserID,

		SchoolMaterialTitle:       req.SchoolMaterialTitle,
		SchoolMaterialDescription: req.SchoolMaterialDescription,

		SchoolMaterialType:       parseMaterialType(req.SchoolMaterialType),
		SchoolMaterialImportance: parseMaterialImportance(imp),

		SchoolMaterialIsRequiredForPass: isRequired,
		SchoolMaterialAffectsScoring:    affectsScoring,

		SchoolMaterialContentHTML: req.SchoolMaterialContentHTML,

		SchoolMaterialFileURL:       req.SchoolMaterialFileURL,
		SchoolMaterialFileName:      req.SchoolMaterialFileName,
		SchoolMaterialFileMimeType:  req.SchoolMaterialFileMimeType,
		SchoolMaterialFileSizeBytes: req.SchoolMaterialFileSizeBytes,

		SchoolMaterialExternalURL: req.SchoolMaterialExternalURL,
		SchoolMaterialYouTubeID:   req.SchoolMaterialYouTubeID,
		SchoolMaterialDurationSec: req.SchoolMaterialDurationSec,

		SchoolMaterialMeetingNumber: req.SchoolMaterialMeetingNumber,
		SchoolMaterialDefaultOrder:  req.SchoolMaterialDefaultOrder,

		SchoolMaterialScopeTag: req.SchoolMaterialScopeTag,

		SchoolMaterialIsActive:    true,
		SchoolMaterialIsPublished: false,
		SchoolMaterialDeleted:     false,

		SchoolMaterialCreatedAt: now,
		SchoolMaterialUpdatedAt: now,
	}
}

/* =======================================================
   Mapper: UpdateRequest -> apply ke Model (untuk UPDATE)
   - now dari dbtime.GetDBTime(c)
======================================================= */

func (req *SchoolMaterialUpdateRequest) ApplyToModel(
	m *materialModel.SchoolMaterialModel,
	now time.Time, // pakai waktu dari dbtime helper
) {
	if req == nil || m == nil {
		return
	}

	if req.SchoolMaterialClassSubjectID != nil {
		m.SchoolMaterialClassSubjectID = req.SchoolMaterialClassSubjectID
	}
	if req.SchoolMaterialTitle != nil {
		m.SchoolMaterialTitle = *req.SchoolMaterialTitle
	}
	if req.SchoolMaterialDescription != nil {
		m.SchoolMaterialDescription = req.SchoolMaterialDescription
	}
	if req.SchoolMaterialType != nil && *req.SchoolMaterialType != "" {
		m.SchoolMaterialType = parseMaterialType(*req.SchoolMaterialType)
	}
	if req.SchoolMaterialImportance != nil && *req.SchoolMaterialImportance != "" {
		m.SchoolMaterialImportance = parseMaterialImportance(*req.SchoolMaterialImportance)
	}
	if req.SchoolMaterialIsRequiredForPass != nil {
		m.SchoolMaterialIsRequiredForPass = *req.SchoolMaterialIsRequiredForPass
	}
	if req.SchoolMaterialAffectsScoring != nil {
		m.SchoolMaterialAffectsScoring = *req.SchoolMaterialAffectsScoring
	}
	if req.SchoolMaterialContentHTML != nil {
		m.SchoolMaterialContentHTML = req.SchoolMaterialContentHTML
	}
	if req.SchoolMaterialFileURL != nil {
		m.SchoolMaterialFileURL = req.SchoolMaterialFileURL
	}
	if req.SchoolMaterialFileName != nil {
		m.SchoolMaterialFileName = req.SchoolMaterialFileName
	}
	if req.SchoolMaterialFileMimeType != nil {
		m.SchoolMaterialFileMimeType = req.SchoolMaterialFileMimeType
	}
	if req.SchoolMaterialFileSizeBytes != nil {
		m.SchoolMaterialFileSizeBytes = req.SchoolMaterialFileSizeBytes
	}
	if req.SchoolMaterialExternalURL != nil {
		m.SchoolMaterialExternalURL = req.SchoolMaterialExternalURL
	}
	if req.SchoolMaterialYouTubeID != nil {
		m.SchoolMaterialYouTubeID = req.SchoolMaterialYouTubeID
	}
	if req.SchoolMaterialDurationSec != nil {
		m.SchoolMaterialDurationSec = req.SchoolMaterialDurationSec
	}
	if req.SchoolMaterialMeetingNumber != nil {
		m.SchoolMaterialMeetingNumber = req.SchoolMaterialMeetingNumber
	}
	if req.SchoolMaterialDefaultOrder != nil {
		m.SchoolMaterialDefaultOrder = req.SchoolMaterialDefaultOrder
	}
	if req.SchoolMaterialScopeTag != nil {
		m.SchoolMaterialScopeTag = req.SchoolMaterialScopeTag
	}
	if req.SchoolMaterialIsActive != nil {
		m.SchoolMaterialIsActive = *req.SchoolMaterialIsActive
	}
	if req.SchoolMaterialIsPublished != nil {
		m.SchoolMaterialIsPublished = *req.SchoolMaterialIsPublished
		if *req.SchoolMaterialIsPublished && m.SchoolMaterialPublishedAt == nil {
			m.SchoolMaterialPublishedAt = &now
		}
		if !*req.SchoolMaterialIsPublished {
			m.SchoolMaterialPublishedAt = nil
		}
	}

	m.SchoolMaterialUpdatedAt = now
}

/* =======================================================
   Soft delete helper (biar controller tinggal manggil)
======================================================= */

func BuildSoftDeleteFieldsSchoolMaterial(now time.Time) map[string]any {
	return map[string]any{
		"school_material_deleted":    true,
		"school_material_deleted_at": now,
		"school_material_updated_at": now,
	}
}
