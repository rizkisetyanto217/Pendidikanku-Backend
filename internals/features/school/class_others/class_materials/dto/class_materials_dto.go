// file: internals/features/school/classes/class_materials/dto/class_materials_dto.go
package dto

import (
	"time"

	"madinahsalam_backend/internals/features/school/class_others/class_materials/model"

	"github.com/google/uuid"
)

/* =========================================================
   Request DTOs
========================================================= */

// Dipakai untuk create materi baru di 1 CSST
// - class_material_school_id ambil dari token/context
// - class_material_csst_id biasanya dari path param
type ClassMaterialCreateRequestDTO struct {
	// konten utama
	ClassMaterialTitle       string  `json:"class_material_title" validate:"required"`
	ClassMaterialDescription *string `json:"class_material_description,omitempty"`

	// article | doc | ppt | pdf | image | youtube | video_file | link | embed
	ClassMaterialType string `json:"class_material_type" validate:"required,oneof=article doc ppt pdf image youtube video_file link embed"`

	// kalau type = article
	ClassMaterialContentHTML *string `json:"class_material_content_html,omitempty"`

	// kalau type = doc/ppt/pdf/image/video_file
	ClassMaterialFileURL       *string `json:"class_material_file_url,omitempty"`
	ClassMaterialFileName      *string `json:"class_material_file_name,omitempty"`
	ClassMaterialFileMIMEType  *string `json:"class_material_file_mime_type,omitempty"`
	ClassMaterialFileSizeBytes *int64  `json:"class_material_file_size_bytes,omitempty"`

	// kalau type = link/embed/youtube
	ClassMaterialExternalURL *string `json:"class_material_external_url,omitempty"`
	ClassMaterialYouTubeID   *string `json:"class_material_youtube_id,omitempty"`

	// durasi detik (video)
	ClassMaterialDurationSec *int `json:"class_material_duration_sec,omitempty"`

	// important | additional | optional (default: important kalau kosong)
	ClassMaterialImportance *string `json:"class_material_importance,omitempty"`

	// rules (boleh diabaikan di V1, backend bisa override default)
	ClassMaterialIsRequiredForPass *bool `json:"class_material_is_required_for_pass,omitempty"`
	ClassMaterialAffectsScoring    *bool `json:"class_material_affects_scoring,omitempty"`

	// pertemuan ke berapa & urutan materi dalam CSST
	ClassMaterialMeetingNumber *int `json:"class_material_meeting_number,omitempty"`
	ClassMaterialOrder         *int `json:"class_material_order,omitempty"`

	// info sumber
	// "school" kalau dari template, "teacher" kalau manual (boleh backend yang isi)
	ClassMaterialSourceKind             *string    `json:"class_material_source_kind,omitempty"`
	ClassMaterialSourceSchoolMaterialID *uuid.UUID `json:"class_material_source_school_material_id,omitempty"`

	// langsung publish atau tidak
	ClassMaterialIsPublished *bool `json:"class_material_is_published,omitempty"`
}

// Dipakai untuk update materi (partial update melalui PATCH/PUT)
// Semua field opsional, controller tinggal cek nil vs non-nil
type ClassMaterialUpdateRequestDTO struct {
	ClassMaterialTitle       *string `json:"class_material_title,omitempty"`
	ClassMaterialDescription *string `json:"class_material_description,omitempty"`

	// kalau diizinkan ganti type
	ClassMaterialType        *string `json:"class_material_type,omitempty"`
	ClassMaterialContentHTML *string `json:"class_material_content_html,omitempty"`

	ClassMaterialFileURL       *string `json:"class_material_file_url,omitempty"`
	ClassMaterialFileName      *string `json:"class_material_file_name,omitempty"`
	ClassMaterialFileMIMEType  *string `json:"class_material_file_mime_type,omitempty"`
	ClassMaterialFileSizeBytes *int64  `json:"class_material_file_size_bytes,omitempty"`

	ClassMaterialExternalURL *string `json:"class_material_external_url,omitempty"`
	ClassMaterialYouTubeID   *string `json:"class_material_youtube_id,omitempty"`
	ClassMaterialDurationSec *int    `json:"class_material_duration_sec,omitempty"`

	ClassMaterialImportance        *string `json:"class_material_importance,omitempty"`
	ClassMaterialIsRequiredForPass *bool   `json:"class_material_is_required_for_pass,omitempty"`
	ClassMaterialAffectsScoring    *bool   `json:"class_material_affects_scoring,omitempty"`

	ClassMaterialMeetingNumber *int `json:"class_material_meeting_number,omitempty"`
	ClassMaterialOrder         *int `json:"class_material_order,omitempty"`

	ClassMaterialIsActive    *bool `json:"class_material_is_active,omitempty"`
	ClassMaterialIsPublished *bool `json:"class_material_is_published,omitempty"`
}

/* =========================================================
   Response DTO
   (buat list/detail materi di guru/admin)
   JSON 1:1 dengan model/kolom DB
========================================================= */

type ClassMaterialResponseDTO struct {
	ClassMaterialID       uuid.UUID `json:"class_material_id"`
	ClassMaterialSchoolID uuid.UUID `json:"class_material_school_id"`
	ClassMaterialCSSTID   uuid.UUID `json:"class_material_csst_id"`

	ClassMaterialCreatedByUserID *uuid.UUID `json:"class_material_created_by_user_id"`

	ClassMaterialTitle       string  `json:"class_material_title"`
	ClassMaterialDescription *string `json:"class_material_description,omitempty"`
	ClassMaterialType        string  `json:"class_material_type"`
	ClassMaterialContentHTML *string `json:"class_material_content_html,omitempty"`
	ClassMaterialImportance  string  `json:"class_material_importance"`

	ClassMaterialFileURL       *string `json:"class_material_file_url,omitempty"`
	ClassMaterialFileName      *string `json:"class_material_file_name,omitempty"`
	ClassMaterialFileMIMEType  *string `json:"class_material_file_mime_type,omitempty"`
	ClassMaterialFileSizeBytes *int64  `json:"class_material_file_size_bytes,omitempty"`

	ClassMaterialExternalURL *string `json:"class_material_external_url,omitempty"`
	ClassMaterialYouTubeID   *string `json:"class_material_youtube_id,omitempty"`
	ClassMaterialDurationSec *int    `json:"class_material_duration_sec,omitempty"`

	ClassMaterialIsRequiredForPass bool `json:"class_material_is_required_for_pass"`
	ClassMaterialAffectsScoring    bool `json:"class_material_affects_scoring"`

	ClassMaterialMeetingNumber *int       `json:"class_material_meeting_number,omitempty"`
	ClassMaterialSessionID     *uuid.UUID `json:"class_material_session_id,omitempty"`

	ClassMaterialSourceKind             *string    `json:"class_material_source_kind,omitempty"`
	ClassMaterialSourceSchoolMaterialID *uuid.UUID `json:"class_material_source_school_material_id,omitempty"`

	ClassMaterialOrder       *int       `json:"class_material_order,omitempty"`
	ClassMaterialIsActive    bool       `json:"class_material_is_active"`
	ClassMaterialIsPublished bool       `json:"class_material_is_published"`
	ClassMaterialPublishedAt *time.Time `json:"class_material_published_at,omitempty"`

	ClassMaterialDeleted   bool       `json:"class_material_deleted"`
	ClassMaterialDeletedAt *time.Time `json:"class_material_deleted_at,omitempty"`
	ClassMaterialCreatedAt time.Time  `json:"class_material_created_at"`
	ClassMaterialUpdatedAt time.Time  `json:"class_material_updated_at"`
}

/* =========================================================
   Mapping helpers (dipanggil dari controller)
========================================================= */

func FromModel(m *model.ClassMaterialsModel) *ClassMaterialResponseDTO {
	if m == nil {
		return nil
	}

	return &ClassMaterialResponseDTO{
		ClassMaterialID:       m.ClassMaterialID,
		ClassMaterialSchoolID: m.ClassMaterialSchoolID,
		ClassMaterialCSSTID:   m.ClassMaterialCSSTID,

		ClassMaterialCreatedByUserID: m.ClassMaterialCreatedByUserID,

		ClassMaterialTitle:       m.ClassMaterialTitle,
		ClassMaterialDescription: m.ClassMaterialDescription,
		ClassMaterialType:        string(m.ClassMaterialType),
		ClassMaterialContentHTML: m.ClassMaterialContentHTML,
		ClassMaterialImportance:  string(m.ClassMaterialImportance),

		ClassMaterialFileURL:       m.ClassMaterialFileURL,
		ClassMaterialFileName:      m.ClassMaterialFileName,
		ClassMaterialFileMIMEType:  m.ClassMaterialFileMIMEType,
		ClassMaterialFileSizeBytes: m.ClassMaterialFileSizeBytes,

		ClassMaterialExternalURL: m.ClassMaterialExternalURL,
		ClassMaterialYouTubeID:   m.ClassMaterialYouTubeID,
		ClassMaterialDurationSec: m.ClassMaterialDurationSec,

		ClassMaterialIsRequiredForPass: m.ClassMaterialIsRequiredForPass,
		ClassMaterialAffectsScoring:    m.ClassMaterialAffectsScoring,

		ClassMaterialMeetingNumber: m.ClassMaterialMeetingNumber,
		ClassMaterialSessionID:     m.ClassMaterialSessionID,

		ClassMaterialSourceKind:             m.ClassMaterialSourceKind,
		ClassMaterialSourceSchoolMaterialID: m.ClassMaterialSourceSchoolMaterialID,

		ClassMaterialOrder:       m.ClassMaterialOrder,
		ClassMaterialIsActive:    m.ClassMaterialIsActive,
		ClassMaterialIsPublished: m.ClassMaterialIsPublished,
		ClassMaterialPublishedAt: m.ClassMaterialPublishedAt,

		ClassMaterialDeleted:   m.ClassMaterialDeleted,
		ClassMaterialDeletedAt: m.ClassMaterialDeletedAt,
		ClassMaterialCreatedAt: m.ClassMaterialCreatedAt,
		ClassMaterialUpdatedAt: m.ClassMaterialUpdatedAt,
	}
}

// dipakai di POST
func ApplyCreateDTOToModel(
	req *ClassMaterialCreateRequestDTO,
	m *model.ClassMaterialsModel,
	schoolID, csstID uuid.UUID,
	createdBy *uuid.UUID,
) {
	now := time.Now()

	m.ClassMaterialSchoolID = schoolID
	m.ClassMaterialCSSTID = csstID
	m.ClassMaterialCreatedByUserID = createdBy

	// main content
	m.ClassMaterialTitle = req.ClassMaterialTitle
	m.ClassMaterialDescription = req.ClassMaterialDescription
	m.ClassMaterialType = model.MaterialType(req.ClassMaterialType)
	m.ClassMaterialContentHTML = req.ClassMaterialContentHTML

	// file
	m.ClassMaterialFileURL = req.ClassMaterialFileURL
	m.ClassMaterialFileName = req.ClassMaterialFileName
	m.ClassMaterialFileMIMEType = req.ClassMaterialFileMIMEType
	m.ClassMaterialFileSizeBytes = req.ClassMaterialFileSizeBytes

	// link/youtube
	m.ClassMaterialExternalURL = req.ClassMaterialExternalURL
	m.ClassMaterialYouTubeID = req.ClassMaterialYouTubeID
	m.ClassMaterialDurationSec = req.ClassMaterialDurationSec

	// importance
	imp := model.MaterialImportanceImportant
	if req.ClassMaterialImportance != nil && *req.ClassMaterialImportance != "" {
		imp = model.MaterialImportance(*req.ClassMaterialImportance)
	}
	m.ClassMaterialImportance = imp

	// flags
	if req.ClassMaterialIsRequiredForPass != nil {
		m.ClassMaterialIsRequiredForPass = *req.ClassMaterialIsRequiredForPass
	}
	if req.ClassMaterialAffectsScoring != nil {
		m.ClassMaterialAffectsScoring = *req.ClassMaterialAffectsScoring
	}

	// meeting & order
	m.ClassMaterialMeetingNumber = req.ClassMaterialMeetingNumber
	m.ClassMaterialOrder = req.ClassMaterialOrder

	// source
	m.ClassMaterialSourceKind = req.ClassMaterialSourceKind
	m.ClassMaterialSourceSchoolMaterialID = req.ClassMaterialSourceSchoolMaterialID

	// publish
	if req.ClassMaterialIsPublished != nil {
		m.ClassMaterialIsPublished = *req.ClassMaterialIsPublished
		if *req.ClassMaterialIsPublished && m.ClassMaterialPublishedAt == nil {
			m.ClassMaterialPublishedAt = &now
		}
	}

	// defaults
	m.ClassMaterialIsActive = true
	m.ClassMaterialDeleted = false
	m.ClassMaterialCreatedAt = now
	m.ClassMaterialUpdatedAt = now
}

// dipakai di PATCH
func ApplyUpdateDTOToModel(req *ClassMaterialUpdateRequestDTO, m *model.ClassMaterialsModel) {
	now := time.Now()

	if req.ClassMaterialTitle != nil {
		m.ClassMaterialTitle = *req.ClassMaterialTitle
	}
	if req.ClassMaterialDescription != nil {
		m.ClassMaterialDescription = req.ClassMaterialDescription
	}
	if req.ClassMaterialType != nil && *req.ClassMaterialType != "" {
		m.ClassMaterialType = model.MaterialType(*req.ClassMaterialType)
	}
	if req.ClassMaterialContentHTML != nil {
		m.ClassMaterialContentHTML = req.ClassMaterialContentHTML
	}

	if req.ClassMaterialFileURL != nil {
		m.ClassMaterialFileURL = req.ClassMaterialFileURL
	}
	if req.ClassMaterialFileName != nil {
		m.ClassMaterialFileName = req.ClassMaterialFileName
	}
	if req.ClassMaterialFileMIMEType != nil {
		m.ClassMaterialFileMIMEType = req.ClassMaterialFileMIMEType
	}
	if req.ClassMaterialFileSizeBytes != nil {
		m.ClassMaterialFileSizeBytes = req.ClassMaterialFileSizeBytes
	}

	if req.ClassMaterialExternalURL != nil {
		m.ClassMaterialExternalURL = req.ClassMaterialExternalURL
	}
	if req.ClassMaterialYouTubeID != nil {
		m.ClassMaterialYouTubeID = req.ClassMaterialYouTubeID
	}
	if req.ClassMaterialDurationSec != nil {
		m.ClassMaterialDurationSec = req.ClassMaterialDurationSec
	}

	if req.ClassMaterialImportance != nil && *req.ClassMaterialImportance != "" {
		m.ClassMaterialImportance = model.MaterialImportance(*req.ClassMaterialImportance)
	}
	if req.ClassMaterialIsRequiredForPass != nil {
		m.ClassMaterialIsRequiredForPass = *req.ClassMaterialIsRequiredForPass
	}
	if req.ClassMaterialAffectsScoring != nil {
		m.ClassMaterialAffectsScoring = *req.ClassMaterialAffectsScoring
	}

	if req.ClassMaterialMeetingNumber != nil {
		m.ClassMaterialMeetingNumber = req.ClassMaterialMeetingNumber
	}
	if req.ClassMaterialOrder != nil {
		m.ClassMaterialOrder = req.ClassMaterialOrder
	}

	if req.ClassMaterialIsActive != nil {
		m.ClassMaterialIsActive = *req.ClassMaterialIsActive
	}
	if req.ClassMaterialIsPublished != nil {
		m.ClassMaterialIsPublished = *req.ClassMaterialIsPublished
		if *req.ClassMaterialIsPublished && m.ClassMaterialPublishedAt == nil {
			m.ClassMaterialPublishedAt = &now
		}
		if !*req.ClassMaterialIsPublished {
			m.ClassMaterialPublishedAt = nil
		}
	}

	m.ClassMaterialUpdatedAt = now
}
