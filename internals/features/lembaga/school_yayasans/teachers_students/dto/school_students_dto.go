// file: internals/features/school/students/dto/school_student_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	studentmodel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
)

/* =========================================================
   GENERIC: PatchField[T]
========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"set"`
	Value T    `json:"value,omitempty"`
}

func (p *PatchField[T]) IsZero() bool { return p == nil || !p.Set }

/* =========================================================
   Helpers: status enum
========================================================= */

func normalizeStatus(s studentmodel.SchoolStudentStatus) (studentmodel.SchoolStudentStatus, error) {
	v := studentmodel.SchoolStudentStatus(strings.ToLower(strings.TrimSpace(string(s))))
	switch v {
	case studentmodel.SchoolStudentActive,
		studentmodel.SchoolStudentInactive,
		studentmodel.SchoolStudentAlumni:
		return v, nil
	default:
		return "", errors.New("invalid school_student_status (boleh: active, inactive, alumni)")
	}
}

func normalizeStatusPtr(s *studentmodel.SchoolStudentStatus) (*studentmodel.SchoolStudentStatus, error) {
	if s == nil {
		return nil, nil
	}
	v, err := normalizeStatus(*s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

/* =========================================================
   Type item untuk render class_sections (JSONB)
   — backend yang memelihara; hanya tampil di response
========================================================= */

type SchoolStudentSectionItem struct {
	ClassSectionID             uuid.UUID `json:"class_section_id"`
	IsActive                   bool      `json:"is_active"`
	From                       *string   `json:"from,omitempty"` // YYYY-MM-DD
	To                         *string   `json:"to,omitempty"`   // YYYY-MM-DD | null
	ClassSectionName           *string   `json:"class_section_name,omitempty"`
	ClassSectionSlug           *string   `json:"class_section_slug,omitempty"`
	ClassSectionImageURL       *string   `json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey *string   `json:"class_section_image_object_key,omitempty"`
}

/* =========================================================
   REQUEST: CREATE
========================================================= */

type SchoolStudentCreateReq struct {
	SchoolStudentSchoolID      uuid.UUID `json:"school_student_school_id"`
	SchoolStudentUserProfileID uuid.UUID `json:"school_student_user_profile_id"`

	SchoolStudentSlug string `json:"school_student_slug"` // required

	SchoolStudentCode               *string                           `json:"school_student_code,omitempty"`
	SchoolStudentStatus             *studentmodel.SchoolStudentStatus `json:"school_student_status,omitempty"` // default: active
	SchoolStudentNote               *string                           `json:"school_student_note,omitempty"`
	SchoolStudentJoinedAt           *time.Time                        `json:"school_student_joined_at,omitempty"`
	SchoolStudentLeftAt             *time.Time                        `json:"school_student_left_at,omitempty"`
	SchoolStudentNeedsClassSections *bool                             `json:"school_student_needs_class_sections,omitempty"`

	// Caches users_profile (opsional di saat create)
	SchoolStudentUserProfileNameCache              *string `json:"school_student_user_profile_name_cache,omitempty"`
	SchoolStudentUserProfileAvatarURLCache         *string `json:"school_student_user_profile_avatar_url_cache,omitempty"`
	SchoolStudentUserProfileWhatsappURLCache       *string `json:"school_student_user_profile_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileParentNameCache        *string `json:"school_student_user_profile_parent_name_cache,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLCache *string `json:"school_student_user_profile_parent_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileGenderCache            *string `json:"school_student_user_profile_gender_cache,omitempty"`
}

func (r *SchoolStudentCreateReq) Normalize() {
	r.SchoolStudentSlug = strings.ToLower(strings.TrimSpace(r.SchoolStudentSlug))

	// default status: active
	if r.SchoolStudentStatus == nil {
		def := studentmodel.SchoolStudentActive
		r.SchoolStudentStatus = &def
	} else if norm, err := normalizeStatusPtr(r.SchoolStudentStatus); err == nil {
		r.SchoolStudentStatus = norm
	}

	if r.SchoolStudentCode != nil {
		if c := strings.TrimSpace(*r.SchoolStudentCode); c == "" {
			r.SchoolStudentCode = nil
		} else {
			r.SchoolStudentCode = &c
		}
	}
	if r.SchoolStudentNote != nil {
		if n := strings.TrimSpace(*r.SchoolStudentNote); n == "" {
			r.SchoolStudentNote = nil
		} else {
			r.SchoolStudentNote = &n
		}
	}
}

func (r *SchoolStudentCreateReq) Validate() error {
	if r.SchoolStudentSlug == "" {
		return errors.New("school_student_slug wajib diisi")
	}
	// validate status (set ke default sebelumnya jika nil)
	if r.SchoolStudentStatus != nil {
		if _, err := normalizeStatus(*r.SchoolStudentStatus); err != nil {
			return err
		}
	}
	if r.SchoolStudentJoinedAt != nil && r.SchoolStudentLeftAt != nil &&
		r.SchoolStudentLeftAt.Before(*r.SchoolStudentJoinedAt) {
		return errors.New("school_student_left_at tidak boleh lebih awal dari school_student_joined_at")
	}
	return nil
}

func (r *SchoolStudentCreateReq) ToModel() *studentmodel.SchoolStudentModel {
	status := studentmodel.SchoolStudentActive
	if r.SchoolStudentStatus != nil {
		status = *r.SchoolStudentStatus
	}

	needsClassSections := false
	if r.SchoolStudentNeedsClassSections != nil {
		needsClassSections = *r.SchoolStudentNeedsClassSections
	}

	return &studentmodel.SchoolStudentModel{
		SchoolStudentSchoolID:      r.SchoolStudentSchoolID,
		SchoolStudentUserProfileID: r.SchoolStudentUserProfileID,

		SchoolStudentSlug:   r.SchoolStudentSlug,
		SchoolStudentCode:   r.SchoolStudentCode,
		SchoolStudentStatus: status,
		SchoolStudentNote:   r.SchoolStudentNote,

		SchoolStudentJoinedAt: r.SchoolStudentJoinedAt,
		SchoolStudentLeftAt:   r.SchoolStudentLeftAt,

		SchoolStudentNeedsClassSections: needsClassSections,

		// caches users_profile
		SchoolStudentUserProfileNameCache:              r.SchoolStudentUserProfileNameCache,
		SchoolStudentUserProfileAvatarURLCache:         r.SchoolStudentUserProfileAvatarURLCache,
		SchoolStudentUserProfileWhatsappURLCache:       r.SchoolStudentUserProfileWhatsappURLCache,
		SchoolStudentUserProfileParentNameCache:        r.SchoolStudentUserProfileParentNameCache,
		SchoolStudentUserProfileParentWhatsappURLCache: r.SchoolStudentUserProfileParentWhatsappURLCache,
		SchoolStudentUserProfileGenderCache:            r.SchoolStudentUserProfileGenderCache,
	}
}

/* =========================================================
   REQUEST: UPDATE (PUT, full)
========================================================= */

type SchoolStudentUpdateReq struct {
	SchoolStudentSlug string `json:"school_student_slug"`

	SchoolStudentCode   *string                          `json:"school_student_code,omitempty"`
	SchoolStudentStatus studentmodel.SchoolStudentStatus `json:"school_student_status"`
	SchoolStudentNote   *string                          `json:"school_student_note,omitempty"`

	SchoolStudentJoinedAt *time.Time `json:"school_student_joined_at,omitempty"`
	SchoolStudentLeftAt   *time.Time `json:"school_student_left_at,omitempty"`

	SchoolStudentNeedsClassSections *bool `json:"school_student_needs_class_sections,omitempty"`

	// caches users_profile
	SchoolStudentUserProfileNameCache              *string `json:"school_student_user_profile_name_cache,omitempty"`
	SchoolStudentUserProfileAvatarURLCache         *string `json:"school_student_user_profile_avatar_url_cache,omitempty"`
	SchoolStudentUserProfileWhatsappURLCache       *string `json:"school_student_user_profile_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileParentNameCache        *string `json:"school_student_user_profile_parent_name_cache,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLCache *string `json:"school_student_user_profile_parent_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileGenderCache            *string `json:"school_student_user_profile_gender_cache,omitempty"`
}

func (r *SchoolStudentUpdateReq) Normalize() {
	r.SchoolStudentSlug = strings.ToLower(strings.TrimSpace(r.SchoolStudentSlug))
	if v, err := normalizeStatus(r.SchoolStudentStatus); err == nil {
		r.SchoolStudentStatus = v
	}
	if r.SchoolStudentCode != nil {
		if c := strings.TrimSpace(*r.SchoolStudentCode); c == "" {
			r.SchoolStudentCode = nil
		} else {
			r.SchoolStudentCode = &c
		}
	}
	if r.SchoolStudentNote != nil {
		if n := strings.TrimSpace(*r.SchoolStudentNote); n == "" {
			r.SchoolStudentNote = nil
		} else {
			r.SchoolStudentNote = &n
		}
	}
}

func (r *SchoolStudentUpdateReq) Validate() error {
	if r.SchoolStudentSlug == "" {
		return errors.New("school_student_slug wajib diisi")
	}
	if _, err := normalizeStatus(r.SchoolStudentStatus); err != nil {
		return err
	}
	if r.SchoolStudentJoinedAt != nil && r.SchoolStudentLeftAt != nil &&
		r.SchoolStudentLeftAt.Before(*r.SchoolStudentJoinedAt) {
		return errors.New("school_student_left_at tidak boleh lebih awal dari school_student_joined_at")
	}
	return nil
}

func (r *SchoolStudentUpdateReq) Apply(m *studentmodel.SchoolStudentModel) {
	m.SchoolStudentSlug = r.SchoolStudentSlug
	m.SchoolStudentCode = r.SchoolStudentCode
	m.SchoolStudentStatus = r.SchoolStudentStatus
	m.SchoolStudentNote = r.SchoolStudentNote
	m.SchoolStudentJoinedAt = r.SchoolStudentJoinedAt
	m.SchoolStudentLeftAt = r.SchoolStudentLeftAt

	if r.SchoolStudentNeedsClassSections != nil {
		m.SchoolStudentNeedsClassSections = *r.SchoolStudentNeedsClassSections
	}

	// caches users_profile
	m.SchoolStudentUserProfileNameCache = r.SchoolStudentUserProfileNameCache
	m.SchoolStudentUserProfileAvatarURLCache = r.SchoolStudentUserProfileAvatarURLCache
	m.SchoolStudentUserProfileWhatsappURLCache = r.SchoolStudentUserProfileWhatsappURLCache
	m.SchoolStudentUserProfileParentNameCache = r.SchoolStudentUserProfileParentNameCache
	m.SchoolStudentUserProfileParentWhatsappURLCache = r.SchoolStudentUserProfileParentWhatsappURLCache
	m.SchoolStudentUserProfileGenderCache = r.SchoolStudentUserProfileGenderCache
}

/* =========================================================
   REQUEST: PATCH (partial)
========================================================= */

type SchoolStudentPatchReq struct {
	SchoolStudentSlug   *PatchField[string]                           `json:"school_student_slug,omitempty"`
	SchoolStudentCode   *PatchField[*string]                          `json:"school_student_code,omitempty"`
	SchoolStudentStatus *PatchField[studentmodel.SchoolStudentStatus] `json:"school_student_status,omitempty"`
	SchoolStudentNote   *PatchField[*string]                          `json:"school_student_note,omitempty"`

	SchoolStudentJoinedAt *PatchField[*time.Time] `json:"school_student_joined_at,omitempty"`
	SchoolStudentLeftAt   *PatchField[*time.Time] `json:"school_student_left_at,omitempty"`

	SchoolStudentNeedsClassSections *PatchField[bool] `json:"school_student_needs_class_sections,omitempty"`

	// caches users_profile
	SchoolStudentUserProfileNameCache              *PatchField[*string] `json:"school_student_user_profile_name_cache,omitempty"`
	SchoolStudentUserProfileAvatarURLCache         *PatchField[*string] `json:"school_student_user_profile_avatar_url_cache,omitempty"`
	SchoolStudentUserProfileWhatsappURLCache       *PatchField[*string] `json:"school_student_user_profile_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileParentNameCache        *PatchField[*string] `json:"school_student_user_profile_parent_name_cache,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLCache *PatchField[*string] `json:"school_student_user_profile_parent_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileGenderCache            *PatchField[*string] `json:"school_student_user_profile_gender_cache,omitempty"`
}

func (r *SchoolStudentPatchReq) Normalize() {
	if r.SchoolStudentSlug != nil && r.SchoolStudentSlug.Set {
		r.SchoolStudentSlug.Value = strings.ToLower(strings.TrimSpace(r.SchoolStudentSlug.Value))
	}
	if r.SchoolStudentStatus != nil && r.SchoolStudentStatus.Set {
		if v, err := normalizeStatus(r.SchoolStudentStatus.Value); err == nil {
			r.SchoolStudentStatus.Value = v
		}
	}
	if r.SchoolStudentCode != nil && r.SchoolStudentCode.Set && r.SchoolStudentCode.Value != nil {
		if c := strings.TrimSpace(*r.SchoolStudentCode.Value); c == "" {
			r.SchoolStudentCode.Value = nil
		} else {
			r.SchoolStudentCode.Value = &c
		}
	}
	if r.SchoolStudentNote != nil && r.SchoolStudentNote.Set && r.SchoolStudentNote.Value != nil {
		if n := strings.TrimSpace(*r.SchoolStudentNote.Value); n == "" {
			r.SchoolStudentNote.Value = nil
		} else {
			r.SchoolStudentNote.Value = &n
		}
	}
	if r.SchoolStudentUserProfileGenderCache != nil && r.SchoolStudentUserProfileGenderCache.Set && r.SchoolStudentUserProfileGenderCache.Value != nil {
		if g := strings.TrimSpace(*r.SchoolStudentUserProfileGenderCache.Value); g == "" {
			r.SchoolStudentUserProfileGenderCache.Value = nil
		} else {
			r.SchoolStudentUserProfileGenderCache.Value = &g
		}
	}
}

func (r *SchoolStudentPatchReq) Validate() error {
	if r.SchoolStudentSlug != nil && r.SchoolStudentSlug.Set {
		if r.SchoolStudentSlug.Value == "" {
			return errors.New("school_student_slug tidak boleh kosong saat di-set")
		}
	}
	if r.SchoolStudentStatus != nil && r.SchoolStudentStatus.Set {
		if _, err := normalizeStatus(r.SchoolStudentStatus.Value); err != nil {
			return err
		}
	}
	if r.SchoolStudentJoinedAt != nil && r.SchoolStudentJoinedAt.Set &&
		r.SchoolStudentLeftAt != nil && r.SchoolStudentLeftAt.Set &&
		r.SchoolStudentJoinedAt.Value != nil && r.SchoolStudentLeftAt.Value != nil &&
		r.SchoolStudentLeftAt.Value.Before(*r.SchoolStudentJoinedAt.Value) {
		return errors.New("school_student_left_at tidak boleh lebih awal dari school_student_joined_at")
	}
	return nil
}

func (r *SchoolStudentPatchReq) Apply(m *studentmodel.SchoolStudentModel) {
	if r.SchoolStudentSlug != nil && r.SchoolStudentSlug.Set {
		m.SchoolStudentSlug = r.SchoolStudentSlug.Value
	}
	if r.SchoolStudentCode != nil && r.SchoolStudentCode.Set {
		m.SchoolStudentCode = r.SchoolStudentCode.Value
	}
	if r.SchoolStudentStatus != nil && r.SchoolStudentStatus.Set {
		m.SchoolStudentStatus = r.SchoolStudentStatus.Value
	}
	if r.SchoolStudentNote != nil && r.SchoolStudentNote.Set {
		m.SchoolStudentNote = r.SchoolStudentNote.Value
	}
	if r.SchoolStudentJoinedAt != nil && r.SchoolStudentJoinedAt.Set {
		m.SchoolStudentJoinedAt = r.SchoolStudentJoinedAt.Value
	}
	if r.SchoolStudentLeftAt != nil && r.SchoolStudentLeftAt.Set {
		m.SchoolStudentLeftAt = r.SchoolStudentLeftAt.Value
	}
	if r.SchoolStudentNeedsClassSections != nil && r.SchoolStudentNeedsClassSections.Set {
		m.SchoolStudentNeedsClassSections = r.SchoolStudentNeedsClassSections.Value
	}

	// caches users_profile
	if r.SchoolStudentUserProfileNameCache != nil && r.SchoolStudentUserProfileNameCache.Set {
		m.SchoolStudentUserProfileNameCache = r.SchoolStudentUserProfileNameCache.Value
	}
	if r.SchoolStudentUserProfileAvatarURLCache != nil && r.SchoolStudentUserProfileAvatarURLCache.Set {
		m.SchoolStudentUserProfileAvatarURLCache = r.SchoolStudentUserProfileAvatarURLCache.Value
	}
	if r.SchoolStudentUserProfileWhatsappURLCache != nil && r.SchoolStudentUserProfileWhatsappURLCache.Set {
		m.SchoolStudentUserProfileWhatsappURLCache = r.SchoolStudentUserProfileWhatsappURLCache.Value
	}
	if r.SchoolStudentUserProfileParentNameCache != nil && r.SchoolStudentUserProfileParentNameCache.Set {
		m.SchoolStudentUserProfileParentNameCache = r.SchoolStudentUserProfileParentNameCache.Value
	}
	if r.SchoolStudentUserProfileParentWhatsappURLCache != nil && r.SchoolStudentUserProfileParentWhatsappURLCache.Set {
		m.SchoolStudentUserProfileParentWhatsappURLCache = r.SchoolStudentUserProfileParentWhatsappURLCache.Value
	}
	if r.SchoolStudentUserProfileGenderCache != nil && r.SchoolStudentUserProfileGenderCache.Set {
		m.SchoolStudentUserProfileGenderCache = r.SchoolStudentUserProfileGenderCache.Value
	}
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type SchoolStudentResp struct {
	SchoolStudentID            uuid.UUID `json:"school_student_id"`
	SchoolStudentSchoolID      uuid.UUID `json:"school_student_school_id"`
	SchoolStudentUserProfileID uuid.UUID `json:"school_student_user_profile_id"`

	SchoolStudentSlug   string                           `json:"school_student_slug"`
	SchoolStudentCode   *string                          `json:"school_student_code,omitempty"`
	SchoolStudentStatus studentmodel.SchoolStudentStatus `json:"school_student_status"`
	SchoolStudentNote   *string                          `json:"school_student_note,omitempty"`

	SchoolStudentJoinedAt *time.Time `json:"school_student_joined_at,omitempty"`
	SchoolStudentLeftAt   *time.Time `json:"school_student_left_at,omitempty"`

	// flag: butuh penempatan ke class_sections?
	SchoolStudentNeedsClassSections bool `json:"school_student_needs_class_sections"`

	// caches users_profile
	SchoolStudentUserProfileNameCache              *string `json:"school_student_user_profile_name_cache,omitempty"`
	SchoolStudentUserProfileAvatarURLCache         *string `json:"school_student_user_profile_avatar_url_cache,omitempty"`
	SchoolStudentUserProfileWhatsappURLCache       *string `json:"school_student_user_profile_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileParentNameCache        *string `json:"school_student_user_profile_parent_name_cache,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLCache *string `json:"school_student_user_profile_parent_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileGenderCache            *string `json:"school_student_user_profile_gender_cache,omitempty"`

	// Class sections (read-only dari backend)
	SchoolStudentSections []SchoolStudentSectionItem `json:"school_student_class_sections"`

	// CSST: kita expose sebagai JSON mentah (array)
	SchoolStudentClassSectionSubjectTeachers json.RawMessage `json:"school_student_class_section_subject_teachers"`

	// Stats (ALL)
	SchoolStudentTotalClassSections               int `json:"school_student_total_class_sections"`
	SchoolStudentTotalClassSectionSubjectTeachers int `json:"school_student_total_class_section_subject_teachers"`

	// Stats (ACTIVE)
	SchoolStudentTotalClassSectionsActive               int `json:"school_student_total_class_sections_active"`
	SchoolStudentTotalClassSectionSubjectTeachersActive int `json:"school_student_total_class_section_subject_teachers_active"`

	SchoolStudentCreatedAt time.Time  `json:"school_student_created_at"`
	SchoolStudentUpdatedAt time.Time  `json:"school_student_updated_at"`
	SchoolStudentDeletedAt *time.Time `json:"school_student_deleted_at,omitempty"`
}

func FromModel(m *studentmodel.SchoolStudentModel) SchoolStudentResp {
	var delAt *time.Time
	if m.SchoolStudentDeletedAt.Valid {
		t := m.SchoolStudentDeletedAt.Time
		delAt = &t
	}

	// decode JSONB class_sections -> []SchoolStudentSectionItem
	sections := make([]SchoolStudentSectionItem, 0)
	if len(m.SchoolStudentClassSections) > 0 {
		_ = json.Unmarshal(m.SchoolStudentClassSections, &sections) // fallback: [] kalau error
	}

	// CSST: biarin raw JSON (array) supaya fleksibel
	csstRaw := json.RawMessage([]byte("[]"))
	if len(m.SchoolStudentClassSectionSubjectTeachers) > 0 {
		csstRaw = json.RawMessage(m.SchoolStudentClassSectionSubjectTeachers)
	}

	return SchoolStudentResp{
		SchoolStudentID:            m.SchoolStudentID,
		SchoolStudentSchoolID:      m.SchoolStudentSchoolID,
		SchoolStudentUserProfileID: m.SchoolStudentUserProfileID,

		SchoolStudentSlug:   m.SchoolStudentSlug,
		SchoolStudentCode:   m.SchoolStudentCode,
		SchoolStudentStatus: m.SchoolStudentStatus,
		SchoolStudentNote:   m.SchoolStudentNote,

		SchoolStudentJoinedAt: m.SchoolStudentJoinedAt,
		SchoolStudentLeftAt:   m.SchoolStudentLeftAt,

		SchoolStudentNeedsClassSections: m.SchoolStudentNeedsClassSections,

		// caches users_profile
		SchoolStudentUserProfileNameCache:              m.SchoolStudentUserProfileNameCache,
		SchoolStudentUserProfileAvatarURLCache:         m.SchoolStudentUserProfileAvatarURLCache,
		SchoolStudentUserProfileWhatsappURLCache:       m.SchoolStudentUserProfileWhatsappURLCache,
		SchoolStudentUserProfileParentNameCache:        m.SchoolStudentUserProfileParentNameCache,
		SchoolStudentUserProfileParentWhatsappURLCache: m.SchoolStudentUserProfileParentWhatsappURLCache,
		SchoolStudentUserProfileGenderCache:            m.SchoolStudentUserProfileGenderCache,

		SchoolStudentSections:                               sections,
		SchoolStudentClassSectionSubjectTeachers:            csstRaw,
		SchoolStudentTotalClassSections:                     m.SchoolStudentTotalClassSections,
		SchoolStudentTotalClassSectionSubjectTeachers:       m.SchoolStudentTotalClassSectionSubjectTeachers,
		SchoolStudentTotalClassSectionsActive:               m.SchoolStudentTotalClassSectionsActive,
		SchoolStudentTotalClassSectionSubjectTeachersActive: m.SchoolStudentTotalClassSectionSubjectTeachersActive,

		SchoolStudentCreatedAt: m.SchoolStudentCreatedAt,
		SchoolStudentUpdatedAt: m.SchoolStudentUpdatedAt,
		SchoolStudentDeletedAt: delAt,
	}
}
