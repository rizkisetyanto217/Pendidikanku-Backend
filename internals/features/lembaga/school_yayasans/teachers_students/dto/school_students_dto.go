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
   (Opsional) Type item untuk render class_sections (JSONB)
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

	SchoolStudentCode     *string                           `json:"school_student_code,omitempty"`
	SchoolStudentStatus   *studentmodel.SchoolStudentStatus `json:"school_student_status,omitempty"` // default: active
	SchoolStudentNote     *string                           `json:"school_student_note,omitempty"`
	SchoolStudentJoinedAt *time.Time                        `json:"school_student_joined_at,omitempty"`
	SchoolStudentLeftAt   *time.Time                        `json:"school_student_left_at,omitempty"`

	// Snapshots users_profile (opsional di saat create)
	SchoolStudentUserProfileNameSnapshot              *string `json:"school_student_user_profile_name_snapshot,omitempty"`
	SchoolStudentUserProfileAvatarURLSnapshot         *string `json:"school_student_user_profile_avatar_url_snapshot,omitempty"`
	SchoolStudentUserProfileWhatsappURLSnapshot       *string `json:"school_student_user_profile_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileParentNameSnapshot        *string `json:"school_student_user_profile_parent_name_snapshot,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLSnapshot *string `json:"school_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileGenderSnapshot            *string `json:"school_student_user_profile_gender_snapshot,omitempty"` // NEW

	// MASJID SNAPSHOT (sinkron model; opsional)
	SchoolStudentSchoolNameSnapshot          *string `json:"school_student_school_name_snapshot,omitempty"`
	SchoolStudentSchoolSlugSnapshot          *string `json:"school_student_school_slug_snapshot,omitempty"`
	SchoolStudentSchoolLogoURLSnapshot       *string `json:"school_student_school_logo_url_snapshot,omitempty"`
	SchoolStudentSchoolIconURLSnapshot       *string `json:"school_student_school_icon_url_snapshot,omitempty"`
	SchoolStudentSchoolBackgroundURLSnapshot *string `json:"school_student_school_background_url_snapshot,omitempty"`
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
	return &studentmodel.SchoolStudentModel{
		SchoolStudentSchoolID:      r.SchoolStudentSchoolID,
		SchoolStudentUserProfileID: r.SchoolStudentUserProfileID,

		SchoolStudentSlug:   r.SchoolStudentSlug,
		SchoolStudentCode:   r.SchoolStudentCode,
		SchoolStudentStatus: status,
		SchoolStudentNote:   r.SchoolStudentNote,

		SchoolStudentJoinedAt: r.SchoolStudentJoinedAt,
		SchoolStudentLeftAt:   r.SchoolStudentLeftAt,

		// snapshots users_profile
		SchoolStudentUserProfileNameSnapshot:              r.SchoolStudentUserProfileNameSnapshot,
		SchoolStudentUserProfileAvatarURLSnapshot:         r.SchoolStudentUserProfileAvatarURLSnapshot,
		SchoolStudentUserProfileWhatsappURLSnapshot:       r.SchoolStudentUserProfileWhatsappURLSnapshot,
		SchoolStudentUserProfileParentNameSnapshot:        r.SchoolStudentUserProfileParentNameSnapshot,
		SchoolStudentUserProfileParentWhatsappURLSnapshot: r.SchoolStudentUserProfileParentWhatsappURLSnapshot,
		SchoolStudentUserProfileGenderSnapshot:            r.SchoolStudentUserProfileGenderSnapshot,

		// MASJID SNAPSHOT (baru)
		SchoolStudentSchoolNameSnapshot:          r.SchoolStudentSchoolNameSnapshot,
		SchoolStudentSchoolSlugSnapshot:          r.SchoolStudentSchoolSlugSnapshot,
		SchoolStudentSchoolLogoURLSnapshot:       r.SchoolStudentSchoolLogoURLSnapshot,
		SchoolStudentSchoolIconURLSnapshot:       r.SchoolStudentSchoolIconURLSnapshot,
		SchoolStudentSchoolBackgroundURLSnapshot: r.SchoolStudentSchoolBackgroundURLSnapshot,
	}
}

/* =========================================================
   REQUEST: UPDATE (PUT, full)
========================================================= */

type SchoolStudentUpdateReq struct {
	SchoolStudentSlug   string                           `json:"school_student_slug"`
	SchoolStudentCode   *string                          `json:"school_student_code,omitempty"`
	SchoolStudentStatus studentmodel.SchoolStudentStatus `json:"school_student_status"`
	SchoolStudentNote   *string                          `json:"school_student_note,omitempty"`

	SchoolStudentJoinedAt *time.Time `json:"school_student_joined_at,omitempty"`
	SchoolStudentLeftAt   *time.Time `json:"school_student_left_at,omitempty"`

	// snapshots users_profile
	SchoolStudentUserProfileNameSnapshot              *string `json:"school_student_user_profile_name_snapshot,omitempty"`
	SchoolStudentUserProfileAvatarURLSnapshot         *string `json:"school_student_user_profile_avatar_url_snapshot,omitempty"`
	SchoolStudentUserProfileWhatsappURLSnapshot       *string `json:"school_student_user_profile_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileParentNameSnapshot        *string `json:"school_student_user_profile_parent_name_snapshot,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLSnapshot *string `json:"school_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileGenderSnapshot            *string `json:"school_student_user_profile_gender_snapshot,omitempty"` // NEW

	// MASJID SNAPSHOT (baru)
	SchoolStudentSchoolNameSnapshot          *string `json:"school_student_school_name_snapshot,omitempty"`
	SchoolStudentSchoolSlugSnapshot          *string `json:"school_student_school_slug_snapshot,omitempty"`
	SchoolStudentSchoolLogoURLSnapshot       *string `json:"school_student_school_logo_url_snapshot,omitempty"`
	SchoolStudentSchoolIconURLSnapshot       *string `json:"school_student_school_icon_url_snapshot,omitempty"`
	SchoolStudentSchoolBackgroundURLSnapshot *string `json:"school_student_school_background_url_snapshot,omitempty"`
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

	// snapshots users_profile
	m.SchoolStudentUserProfileNameSnapshot = r.SchoolStudentUserProfileNameSnapshot
	m.SchoolStudentUserProfileAvatarURLSnapshot = r.SchoolStudentUserProfileAvatarURLSnapshot
	m.SchoolStudentUserProfileWhatsappURLSnapshot = r.SchoolStudentUserProfileWhatsappURLSnapshot
	m.SchoolStudentUserProfileParentNameSnapshot = r.SchoolStudentUserProfileParentNameSnapshot
	m.SchoolStudentUserProfileParentWhatsappURLSnapshot = r.SchoolStudentUserProfileParentWhatsappURLSnapshot
	m.SchoolStudentUserProfileGenderSnapshot = r.SchoolStudentUserProfileGenderSnapshot

	// MASJID SNAPSHOT
	m.SchoolStudentSchoolNameSnapshot = r.SchoolStudentSchoolNameSnapshot
	m.SchoolStudentSchoolSlugSnapshot = r.SchoolStudentSchoolSlugSnapshot
	m.SchoolStudentSchoolLogoURLSnapshot = r.SchoolStudentSchoolLogoURLSnapshot
	m.SchoolStudentSchoolIconURLSnapshot = r.SchoolStudentSchoolIconURLSnapshot
	m.SchoolStudentSchoolBackgroundURLSnapshot = r.SchoolStudentSchoolBackgroundURLSnapshot
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

	// snapshots users_profile
	SchoolStudentUserProfileNameSnapshot              *PatchField[*string] `json:"school_student_user_profile_name_snapshot,omitempty"`
	SchoolStudentUserProfileAvatarURLSnapshot         *PatchField[*string] `json:"school_student_user_profile_avatar_url_snapshot,omitempty"`
	SchoolStudentUserProfileWhatsappURLSnapshot       *PatchField[*string] `json:"school_student_user_profile_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileParentNameSnapshot        *PatchField[*string] `json:"school_student_user_profile_parent_name_snapshot,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLSnapshot *PatchField[*string] `json:"school_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileGenderSnapshot            *PatchField[*string] `json:"school_student_user_profile_gender_snapshot,omitempty"` // NEW

	// MASJID SNAPSHOT (baru)
	SchoolStudentSchoolNameSnapshot          *PatchField[*string] `json:"school_student_school_name_snapshot,omitempty"`
	SchoolStudentSchoolSlugSnapshot          *PatchField[*string] `json:"school_student_school_slug_snapshot,omitempty"`
	SchoolStudentSchoolLogoURLSnapshot       *PatchField[*string] `json:"school_student_school_logo_url_snapshot,omitempty"`
	SchoolStudentSchoolIconURLSnapshot       *PatchField[*string] `json:"school_student_school_icon_url_snapshot,omitempty"`
	SchoolStudentSchoolBackgroundURLSnapshot *PatchField[*string] `json:"school_student_school_background_url_snapshot,omitempty"`
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
	if r.SchoolStudentUserProfileGenderSnapshot != nil && r.SchoolStudentUserProfileGenderSnapshot.Set && r.SchoolStudentUserProfileGenderSnapshot.Value != nil {
		if g := strings.TrimSpace(*r.SchoolStudentUserProfileGenderSnapshot.Value); g == "" {
			r.SchoolStudentUserProfileGenderSnapshot.Value = nil
		} else {
			r.SchoolStudentUserProfileGenderSnapshot.Value = &g
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

	// snapshots users_profile
	if r.SchoolStudentUserProfileNameSnapshot != nil && r.SchoolStudentUserProfileNameSnapshot.Set {
		m.SchoolStudentUserProfileNameSnapshot = r.SchoolStudentUserProfileNameSnapshot.Value
	}
	if r.SchoolStudentUserProfileAvatarURLSnapshot != nil && r.SchoolStudentUserProfileAvatarURLSnapshot.Set {
		m.SchoolStudentUserProfileAvatarURLSnapshot = r.SchoolStudentUserProfileAvatarURLSnapshot.Value
	}
	if r.SchoolStudentUserProfileWhatsappURLSnapshot != nil && r.SchoolStudentUserProfileWhatsappURLSnapshot.Set {
		m.SchoolStudentUserProfileWhatsappURLSnapshot = r.SchoolStudentUserProfileWhatsappURLSnapshot.Value
	}
	if r.SchoolStudentUserProfileParentNameSnapshot != nil && r.SchoolStudentUserProfileParentNameSnapshot.Set {
		m.SchoolStudentUserProfileParentNameSnapshot = r.SchoolStudentUserProfileParentNameSnapshot.Value
	}
	if r.SchoolStudentUserProfileParentWhatsappURLSnapshot != nil && r.SchoolStudentUserProfileParentWhatsappURLSnapshot.Set {
		m.SchoolStudentUserProfileParentWhatsappURLSnapshot = r.SchoolStudentUserProfileParentWhatsappURLSnapshot.Value
	}
	if r.SchoolStudentUserProfileGenderSnapshot != nil && r.SchoolStudentUserProfileGenderSnapshot.Set {
		m.SchoolStudentUserProfileGenderSnapshot = r.SchoolStudentUserProfileGenderSnapshot.Value
	}

	// MASJID SNAPSHOT
	if r.SchoolStudentSchoolNameSnapshot != nil && r.SchoolStudentSchoolNameSnapshot.Set {
		m.SchoolStudentSchoolNameSnapshot = r.SchoolStudentSchoolNameSnapshot.Value
	}
	if r.SchoolStudentSchoolSlugSnapshot != nil && r.SchoolStudentSchoolSlugSnapshot.Set {
		m.SchoolStudentSchoolSlugSnapshot = r.SchoolStudentSchoolSlugSnapshot.Value
	}
	if r.SchoolStudentSchoolLogoURLSnapshot != nil && r.SchoolStudentSchoolLogoURLSnapshot.Set {
		m.SchoolStudentSchoolLogoURLSnapshot = r.SchoolStudentSchoolLogoURLSnapshot.Value
	}
	if r.SchoolStudentSchoolIconURLSnapshot != nil && r.SchoolStudentSchoolIconURLSnapshot.Set {
		m.SchoolStudentSchoolIconURLSnapshot = r.SchoolStudentSchoolIconURLSnapshot.Value
	}
	if r.SchoolStudentSchoolBackgroundURLSnapshot != nil && r.SchoolStudentSchoolBackgroundURLSnapshot.Set {
		m.SchoolStudentSchoolBackgroundURLSnapshot = r.SchoolStudentSchoolBackgroundURLSnapshot.Value
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

	// snapshots users_profile
	SchoolStudentUserProfileNameSnapshot              *string `json:"school_student_user_profile_name_snapshot,omitempty"`
	SchoolStudentUserProfileAvatarURLSnapshot         *string `json:"school_student_user_profile_avatar_url_snapshot,omitempty"`
	SchoolStudentUserProfileWhatsappURLSnapshot       *string `json:"school_student_user_profile_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileParentNameSnapshot        *string `json:"school_student_user_profile_parent_name_snapshot,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLSnapshot *string `json:"school_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileGenderSnapshot            *string `json:"school_student_user_profile_gender_snapshot,omitempty"` // NEW

	// MASJID SNAPSHOT
	SchoolStudentSchoolNameSnapshot          *string `json:"school_student_school_name_snapshot,omitempty"`
	SchoolStudentSchoolSlugSnapshot          *string `json:"school_student_school_slug_snapshot,omitempty"`
	SchoolStudentSchoolLogoURLSnapshot       *string `json:"school_student_school_logo_url_snapshot,omitempty"`
	SchoolStudentSchoolIconURLSnapshot       *string `json:"school_student_school_icon_url_snapshot,omitempty"`
	SchoolStudentSchoolBackgroundURLSnapshot *string `json:"school_student_school_background_url_snapshot,omitempty"`

	// Class sections (read-only dari backend)
	SchoolStudentSections []SchoolStudentSectionItem `json:"school_student_class_sections"`

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

		// snapshots users_profile
		SchoolStudentUserProfileNameSnapshot:              m.SchoolStudentUserProfileNameSnapshot,
		SchoolStudentUserProfileAvatarURLSnapshot:         m.SchoolStudentUserProfileAvatarURLSnapshot,
		SchoolStudentUserProfileWhatsappURLSnapshot:       m.SchoolStudentUserProfileWhatsappURLSnapshot,
		SchoolStudentUserProfileParentNameSnapshot:        m.SchoolStudentUserProfileParentNameSnapshot,
		SchoolStudentUserProfileParentWhatsappURLSnapshot: m.SchoolStudentUserProfileParentWhatsappURLSnapshot,
		SchoolStudentUserProfileGenderSnapshot:            m.SchoolStudentUserProfileGenderSnapshot,

		// MASJID SNAPSHOT
		SchoolStudentSchoolNameSnapshot:          m.SchoolStudentSchoolNameSnapshot,
		SchoolStudentSchoolSlugSnapshot:          m.SchoolStudentSchoolSlugSnapshot,
		SchoolStudentSchoolLogoURLSnapshot:       m.SchoolStudentSchoolLogoURLSnapshot,
		SchoolStudentSchoolIconURLSnapshot:       m.SchoolStudentSchoolIconURLSnapshot,
		SchoolStudentSchoolBackgroundURLSnapshot: m.SchoolStudentSchoolBackgroundURLSnapshot,

		SchoolStudentSections: sections,

		SchoolStudentCreatedAt: m.SchoolStudentCreatedAt,
		SchoolStudentUpdatedAt: m.SchoolStudentUpdatedAt,
		SchoolStudentDeletedAt: delAt,
	}
}
