package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	studentmodel "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"
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

func normalizeStatus(s studentmodel.MasjidStudentStatus) (studentmodel.MasjidStudentStatus, error) {
	v := studentmodel.MasjidStudentStatus(strings.ToLower(strings.TrimSpace(string(s))))
	switch v {
	case studentmodel.MasjidStudentActive,
		studentmodel.MasjidStudentInactive,
		studentmodel.MasjidStudentAlumni:
		return v, nil
	default:
		return "", errors.New("invalid masjid_student_status (boleh: active, inactive, alumni)")
	}
}

func normalizeStatusPtr(s *studentmodel.MasjidStudentStatus) (*studentmodel.MasjidStudentStatus, error) {
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
   (Opsional) Type item untuk render sections (JSONB)
   — backend yang memelihara; hanya tampil di response
========================================================= */

type MasjidStudentSectionItem struct {
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

type MasjidStudentCreateReq struct {
	MasjidStudentMasjidID      uuid.UUID `json:"masjid_student_masjid_id"`
	MasjidStudentUserProfileID uuid.UUID `json:"masjid_student_user_profile_id"`

	MasjidStudentSlug string `json:"masjid_student_slug"` // required

	MasjidStudentCode     *string                           `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus   *studentmodel.MasjidStudentStatus `json:"masjid_student_status,omitempty"` // default: active
	MasjidStudentNote     *string                           `json:"masjid_student_note,omitempty"`
	MasjidStudentJoinedAt *time.Time                        `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time                        `json:"masjid_student_left_at,omitempty"`

	// Snapshots users_profile (opsional di saat create)
	MasjidStudentUserProfileNameSnapshot              *string `json:"masjid_student_user_profile_name_snapshot,omitempty"`
	MasjidStudentUserProfileAvatarURLSnapshot         *string `json:"masjid_student_user_profile_avatar_url_snapshot,omitempty"`
	MasjidStudentUserProfileWhatsappURLSnapshot       *string `json:"masjid_student_user_profile_whatsapp_url_snapshot,omitempty"`
	MasjidStudentUserProfileParentNameSnapshot        *string `json:"masjid_student_user_profile_parent_name_snapshot,omitempty"`
	MasjidStudentUserProfileParentWhatsappURLSnapshot *string `json:"masjid_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// MASJID SNAPSHOT (sinkron model; opsional)
	MasjidStudentMasjidNameSnapshot          *string `json:"masjid_student_masjid_name_snapshot,omitempty"`
	MasjidStudentMasjidSlugSnapshot          *string `json:"masjid_student_masjid_slug_snapshot,omitempty"`
	MasjidStudentMasjidLogoURLSnapshot       *string `json:"masjid_student_masjid_logo_url_snapshot,omitempty"`
	MasjidStudentMasjidIconURLSnapshot       *string `json:"masjid_student_masjid_icon_url_snapshot,omitempty"`
	MasjidStudentMasjidBackgroundURLSnapshot *string `json:"masjid_student_masjid_background_url_snapshot,omitempty"`
}

func (r *MasjidStudentCreateReq) Normalize() {
	r.MasjidStudentSlug = strings.ToLower(strings.TrimSpace(r.MasjidStudentSlug))

	// default status: active
	if r.MasjidStudentStatus == nil {
		def := studentmodel.MasjidStudentActive
		r.MasjidStudentStatus = &def
	} else if norm, err := normalizeStatusPtr(r.MasjidStudentStatus); err == nil {
		r.MasjidStudentStatus = norm
	}

	if r.MasjidStudentCode != nil {
		if c := strings.TrimSpace(*r.MasjidStudentCode); c == "" {
			r.MasjidStudentCode = nil
		} else {
			r.MasjidStudentCode = &c
		}
	}
	if r.MasjidStudentNote != nil {
		if n := strings.TrimSpace(*r.MasjidStudentNote); n == "" {
			r.MasjidStudentNote = nil
		} else {
			r.MasjidStudentNote = &n
		}
	}
}

func (r *MasjidStudentCreateReq) Validate() error {
	if r.MasjidStudentSlug == "" {
		return errors.New("masjid_student_slug wajib diisi")
	}
	// validate status (set ke default sebelumnya jika nil)
	if r.MasjidStudentStatus != nil {
		if _, err := normalizeStatus(*r.MasjidStudentStatus); err != nil {
			return err
		}
	}
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentLeftAt != nil &&
		r.MasjidStudentLeftAt.Before(*r.MasjidStudentJoinedAt) {
		return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
	}
	return nil
}

func (r *MasjidStudentCreateReq) ToModel() *studentmodel.MasjidStudentModel {
	status := studentmodel.MasjidStudentActive
	if r.MasjidStudentStatus != nil {
		status = *r.MasjidStudentStatus
	}
	return &studentmodel.MasjidStudentModel{
		MasjidStudentMasjidID:      r.MasjidStudentMasjidID,
		MasjidStudentUserProfileID: r.MasjidStudentUserProfileID,

		MasjidStudentSlug:   r.MasjidStudentSlug,
		MasjidStudentCode:   r.MasjidStudentCode,
		MasjidStudentStatus: status,
		MasjidStudentNote:   r.MasjidStudentNote,

		MasjidStudentJoinedAt: r.MasjidStudentJoinedAt,
		MasjidStudentLeftAt:   r.MasjidStudentLeftAt,

		// snapshots users_profile
		MasjidStudentUserProfileNameSnapshot:              r.MasjidStudentUserProfileNameSnapshot,
		MasjidStudentUserProfileAvatarURLSnapshot:         r.MasjidStudentUserProfileAvatarURLSnapshot,
		MasjidStudentUserProfileWhatsappURLSnapshot:       r.MasjidStudentUserProfileWhatsappURLSnapshot,
		MasjidStudentUserProfileParentNameSnapshot:        r.MasjidStudentUserProfileParentNameSnapshot,
		MasjidStudentUserProfileParentWhatsappURLSnapshot: r.MasjidStudentUserProfileParentWhatsappURLSnapshot,

		// MASJID SNAPSHOT (baru)
		MasjidStudentMasjidNameSnapshot:          r.MasjidStudentMasjidNameSnapshot,
		MasjidStudentMasjidSlugSnapshot:          r.MasjidStudentMasjidSlugSnapshot,
		MasjidStudentMasjidLogoURLSnapshot:       r.MasjidStudentMasjidLogoURLSnapshot,
		MasjidStudentMasjidIconURLSnapshot:       r.MasjidStudentMasjidIconURLSnapshot,
		MasjidStudentMasjidBackgroundURLSnapshot: r.MasjidStudentMasjidBackgroundURLSnapshot,
	}
}

/* =========================================================
   REQUEST: UPDATE (PUT, full)
========================================================= */

type MasjidStudentUpdateReq struct {
	MasjidStudentSlug   string                           `json:"masjid_student_slug"`
	MasjidStudentCode   *string                          `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus studentmodel.MasjidStudentStatus `json:"masjid_student_status"`
	MasjidStudentNote   *string                          `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `json:"masjid_student_left_at,omitempty"`

	// snapshots users_profile
	MasjidStudentUserProfileNameSnapshot              *string `json:"masjid_student_user_profile_name_snapshot,omitempty"`
	MasjidStudentUserProfileAvatarURLSnapshot         *string `json:"masjid_student_user_profile_avatar_url_snapshot,omitempty"`
	MasjidStudentUserProfileWhatsappURLSnapshot       *string `json:"masjid_student_user_profile_whatsapp_url_snapshot,omitempty"`
	MasjidStudentUserProfileParentNameSnapshot        *string `json:"masjid_student_user_profile_parent_name_snapshot,omitempty"`
	MasjidStudentUserProfileParentWhatsappURLSnapshot *string `json:"masjid_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// MASJID SNAPSHOT (baru)
	MasjidStudentMasjidNameSnapshot          *string `json:"masjid_student_masjid_name_snapshot,omitempty"`
	MasjidStudentMasjidSlugSnapshot          *string `json:"masjid_student_masjid_slug_snapshot,omitempty"`
	MasjidStudentMasjidLogoURLSnapshot       *string `json:"masjid_student_masjid_logo_url_snapshot,omitempty"`
	MasjidStudentMasjidIconURLSnapshot       *string `json:"masjid_student_masjid_icon_url_snapshot,omitempty"`
	MasjidStudentMasjidBackgroundURLSnapshot *string `json:"masjid_student_masjid_background_url_snapshot,omitempty"`
}

func (r *MasjidStudentUpdateReq) Normalize() {
	r.MasjidStudentSlug = strings.ToLower(strings.TrimSpace(r.MasjidStudentSlug))
	if v, err := normalizeStatus(r.MasjidStudentStatus); err == nil {
		r.MasjidStudentStatus = v
	}
	if r.MasjidStudentCode != nil {
		if c := strings.TrimSpace(*r.MasjidStudentCode); c == "" {
			r.MasjidStudentCode = nil
		} else {
			r.MasjidStudentCode = &c
		}
	}
	if r.MasjidStudentNote != nil {
		if n := strings.TrimSpace(*r.MasjidStudentNote); n == "" {
			r.MasjidStudentNote = nil
		} else {
			r.MasjidStudentNote = &n
		}
	}
}

func (r *MasjidStudentUpdateReq) Validate() error {
	if r.MasjidStudentSlug == "" {
		return errors.New("masjid_student_slug wajib diisi")
	}
	if _, err := normalizeStatus(r.MasjidStudentStatus); err != nil {
		return err
	}
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentLeftAt != nil &&
		r.MasjidStudentLeftAt.Before(*r.MasjidStudentJoinedAt) {
		return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
	}
	return nil
}

func (r *MasjidStudentUpdateReq) Apply(m *studentmodel.MasjidStudentModel) {
	m.MasjidStudentSlug = r.MasjidStudentSlug
	m.MasjidStudentCode = r.MasjidStudentCode
	m.MasjidStudentStatus = r.MasjidStudentStatus
	m.MasjidStudentNote = r.MasjidStudentNote
	m.MasjidStudentJoinedAt = r.MasjidStudentJoinedAt
	m.MasjidStudentLeftAt = r.MasjidStudentLeftAt

	// snapshots users_profile
	m.MasjidStudentUserProfileNameSnapshot = r.MasjidStudentUserProfileNameSnapshot
	m.MasjidStudentUserProfileAvatarURLSnapshot = r.MasjidStudentUserProfileAvatarURLSnapshot
	m.MasjidStudentUserProfileWhatsappURLSnapshot = r.MasjidStudentUserProfileWhatsappURLSnapshot
	m.MasjidStudentUserProfileParentNameSnapshot = r.MasjidStudentUserProfileParentNameSnapshot
	m.MasjidStudentUserProfileParentWhatsappURLSnapshot = r.MasjidStudentUserProfileParentWhatsappURLSnapshot

	// MASJID SNAPSHOT
	m.MasjidStudentMasjidNameSnapshot = r.MasjidStudentMasjidNameSnapshot
	m.MasjidStudentMasjidSlugSnapshot = r.MasjidStudentMasjidSlugSnapshot
	m.MasjidStudentMasjidLogoURLSnapshot = r.MasjidStudentMasjidLogoURLSnapshot
	m.MasjidStudentMasjidIconURLSnapshot = r.MasjidStudentMasjidIconURLSnapshot
	m.MasjidStudentMasjidBackgroundURLSnapshot = r.MasjidStudentMasjidBackgroundURLSnapshot
}

/* =========================================================
   REQUEST: PATCH (partial)
========================================================= */

type MasjidStudentPatchReq struct {
	MasjidStudentSlug   *PatchField[string]                           `json:"masjid_student_slug,omitempty"`
	MasjidStudentCode   *PatchField[*string]                          `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus *PatchField[studentmodel.MasjidStudentStatus] `json:"masjid_student_status,omitempty"`
	MasjidStudentNote   *PatchField[*string]                          `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *PatchField[*time.Time] `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *PatchField[*time.Time] `json:"masjid_student_left_at,omitempty"`

	// snapshots users_profile
	MasjidStudentUserProfileNameSnapshot              *PatchField[*string] `json:"masjid_student_user_profile_name_snapshot,omitempty"`
	MasjidStudentUserProfileAvatarURLSnapshot         *PatchField[*string] `json:"masjid_student_user_profile_avatar_url_snapshot,omitempty"`
	MasjidStudentUserProfileWhatsappURLSnapshot       *PatchField[*string] `json:"masjid_student_user_profile_whatsapp_url_snapshot,omitempty"`
	MasjidStudentUserProfileParentNameSnapshot        *PatchField[*string] `json:"masjid_student_user_profile_parent_name_snapshot,omitempty"`
	MasjidStudentUserProfileParentWhatsappURLSnapshot *PatchField[*string] `json:"masjid_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// MASJID SNAPSHOT (baru)
	MasjidStudentMasjidNameSnapshot          *PatchField[*string] `json:"masjid_student_masjid_name_snapshot,omitempty"`
	MasjidStudentMasjidSlugSnapshot          *PatchField[*string] `json:"masjid_student_masjid_slug_snapshot,omitempty"`
	MasjidStudentMasjidLogoURLSnapshot       *PatchField[*string] `json:"masjid_student_masjid_logo_url_snapshot,omitempty"`
	MasjidStudentMasjidIconURLSnapshot       *PatchField[*string] `json:"masjid_student_masjid_icon_url_snapshot,omitempty"`
	MasjidStudentMasjidBackgroundURLSnapshot *PatchField[*string] `json:"masjid_student_masjid_background_url_snapshot,omitempty"`
}

func (r *MasjidStudentPatchReq) Normalize() {
	if r.MasjidStudentSlug != nil && r.MasjidStudentSlug.Set {
		r.MasjidStudentSlug.Value = strings.ToLower(strings.TrimSpace(r.MasjidStudentSlug.Value))
	}
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		if v, err := normalizeStatus(r.MasjidStudentStatus.Value); err == nil {
			r.MasjidStudentStatus.Value = v
		}
	}
	if r.MasjidStudentCode != nil && r.MasjidStudentCode.Set && r.MasjidStudentCode.Value != nil {
		if c := strings.TrimSpace(*r.MasjidStudentCode.Value); c == "" {
			r.MasjidStudentCode.Value = nil
		} else {
			r.MasjidStudentCode.Value = &c
		}
	}
	if r.MasjidStudentNote != nil && r.MasjidStudentNote.Set && r.MasjidStudentNote.Value != nil {
		if n := strings.TrimSpace(*r.MasjidStudentNote.Value); n == "" {
			r.MasjidStudentNote.Value = nil
		} else {
			r.MasjidStudentNote.Value = &n
		}
	}
}

func (r *MasjidStudentPatchReq) Validate() error {
	if r.MasjidStudentSlug != nil && r.MasjidStudentSlug.Set {
		if r.MasjidStudentSlug.Value == "" {
			return errors.New("masjid_student_slug tidak boleh kosong saat di-set")
		}
	}
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		if _, err := normalizeStatus(r.MasjidStudentStatus.Value); err != nil {
			return err
		}
	}
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentJoinedAt.Set &&
		r.MasjidStudentLeftAt != nil && r.MasjidStudentLeftAt.Set &&
		r.MasjidStudentJoinedAt.Value != nil && r.MasjidStudentLeftAt.Value != nil &&
		r.MasjidStudentLeftAt.Value.Before(*r.MasjidStudentJoinedAt.Value) {
		return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
	}
	return nil
}

func (r *MasjidStudentPatchReq) Apply(m *studentmodel.MasjidStudentModel) {
	if r.MasjidStudentSlug != nil && r.MasjidStudentSlug.Set {
		m.MasjidStudentSlug = r.MasjidStudentSlug.Value
	}
	if r.MasjidStudentCode != nil && r.MasjidStudentCode.Set {
		m.MasjidStudentCode = r.MasjidStudentCode.Value
	}
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		m.MasjidStudentStatus = r.MasjidStudentStatus.Value
	}
	if r.MasjidStudentNote != nil && r.MasjidStudentNote.Set {
		m.MasjidStudentNote = r.MasjidStudentNote.Value
	}
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentJoinedAt.Set {
		m.MasjidStudentJoinedAt = r.MasjidStudentJoinedAt.Value
	}
	if r.MasjidStudentLeftAt != nil && r.MasjidStudentLeftAt.Set {
		m.MasjidStudentLeftAt = r.MasjidStudentLeftAt.Value
	}

	// snapshots users_profile
	if r.MasjidStudentUserProfileNameSnapshot != nil && r.MasjidStudentUserProfileNameSnapshot.Set {
		m.MasjidStudentUserProfileNameSnapshot = r.MasjidStudentUserProfileNameSnapshot.Value
	}
	if r.MasjidStudentUserProfileAvatarURLSnapshot != nil && r.MasjidStudentUserProfileAvatarURLSnapshot.Set {
		m.MasjidStudentUserProfileAvatarURLSnapshot = r.MasjidStudentUserProfileAvatarURLSnapshot.Value
	}
	if r.MasjidStudentUserProfileWhatsappURLSnapshot != nil && r.MasjidStudentUserProfileWhatsappURLSnapshot.Set {
		m.MasjidStudentUserProfileWhatsappURLSnapshot = r.MasjidStudentUserProfileWhatsappURLSnapshot.Value
	}
	if r.MasjidStudentUserProfileParentNameSnapshot != nil && r.MasjidStudentUserProfileParentNameSnapshot.Set {
		m.MasjidStudentUserProfileParentNameSnapshot = r.MasjidStudentUserProfileParentNameSnapshot.Value
	}
	if r.MasjidStudentUserProfileParentWhatsappURLSnapshot != nil && r.MasjidStudentUserProfileParentWhatsappURLSnapshot.Set {
		m.MasjidStudentUserProfileParentWhatsappURLSnapshot = r.MasjidStudentUserProfileParentWhatsappURLSnapshot.Value
	}

	// MASJID SNAPSHOT
	if r.MasjidStudentMasjidNameSnapshot != nil && r.MasjidStudentMasjidNameSnapshot.Set {
		m.MasjidStudentMasjidNameSnapshot = r.MasjidStudentMasjidNameSnapshot.Value
	}
	if r.MasjidStudentMasjidSlugSnapshot != nil && r.MasjidStudentMasjidSlugSnapshot.Set {
		m.MasjidStudentMasjidSlugSnapshot = r.MasjidStudentMasjidSlugSnapshot.Value
	}
	if r.MasjidStudentMasjidLogoURLSnapshot != nil && r.MasjidStudentMasjidLogoURLSnapshot.Set {
		m.MasjidStudentMasjidLogoURLSnapshot = r.MasjidStudentMasjidLogoURLSnapshot.Value
	}
	if r.MasjidStudentMasjidIconURLSnapshot != nil && r.MasjidStudentMasjidIconURLSnapshot.Set {
		m.MasjidStudentMasjidIconURLSnapshot = r.MasjidStudentMasjidIconURLSnapshot.Value
	}
	if r.MasjidStudentMasjidBackgroundURLSnapshot != nil && r.MasjidStudentMasjidBackgroundURLSnapshot.Set {
		m.MasjidStudentMasjidBackgroundURLSnapshot = r.MasjidStudentMasjidBackgroundURLSnapshot.Value
	}
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type MasjidStudentResp struct {
	MasjidStudentID            uuid.UUID `json:"masjid_student_id"`
	MasjidStudentMasjidID      uuid.UUID `json:"masjid_student_masjid_id"`
	MasjidStudentUserProfileID uuid.UUID `json:"masjid_student_user_profile_id"`

	MasjidStudentSlug   string                           `json:"masjid_student_slug"`
	MasjidStudentCode   *string                          `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus studentmodel.MasjidStudentStatus `json:"masjid_student_status"`
	MasjidStudentNote   *string                          `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `json:"masjid_student_left_at,omitempty"`

	// snapshots users_profile
	MasjidStudentUserProfileNameSnapshot              *string `json:"masjid_student_user_profile_name_snapshot,omitempty"`
	MasjidStudentUserProfileAvatarURLSnapshot         *string `json:"masjid_student_user_profile_avatar_url_snapshot,omitempty"`
	MasjidStudentUserProfileWhatsappURLSnapshot       *string `json:"masjid_student_user_profile_whatsapp_url_snapshot,omitempty"`
	MasjidStudentUserProfileParentNameSnapshot        *string `json:"masjid_student_user_profile_parent_name_snapshot,omitempty"`
	MasjidStudentUserProfileParentWhatsappURLSnapshot *string `json:"masjid_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// MASJID SNAPSHOT
	MasjidStudentMasjidNameSnapshot          *string `json:"masjid_student_masjid_name_snapshot,omitempty"`
	MasjidStudentMasjidSlugSnapshot          *string `json:"masjid_student_masjid_slug_snapshot,omitempty"`
	MasjidStudentMasjidLogoURLSnapshot       *string `json:"masjid_student_masjid_logo_url_snapshot,omitempty"`
	MasjidStudentMasjidIconURLSnapshot       *string `json:"masjid_student_masjid_icon_url_snapshot,omitempty"`
	MasjidStudentMasjidBackgroundURLSnapshot *string `json:"masjid_student_masjid_background_url_snapshot,omitempty"`

	// Sections (read-only dari backend)
	MasjidStudentSections []MasjidStudentSectionItem `json:"masjid_student_sections"`

	MasjidStudentCreatedAt time.Time  `json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time  `json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt *time.Time `json:"masjid_student_deleted_at,omitempty"`
}

func FromModel(m *studentmodel.MasjidStudentModel) MasjidStudentResp {
	var delAt *time.Time
	if m.MasjidStudentDeletedAt.Valid {
		t := m.MasjidStudentDeletedAt.Time
		delAt = &t
	}

	// decode JSONB sections -> []MasjidStudentSectionItem
	sections := make([]MasjidStudentSectionItem, 0)
	if len(m.MasjidStudentSections) > 0 {
		_ = json.Unmarshal(m.MasjidStudentSections, &sections) // aman: fallback ke [] saat error
	}

	return MasjidStudentResp{
		MasjidStudentID:            m.MasjidStudentID,
		MasjidStudentMasjidID:      m.MasjidStudentMasjidID,
		MasjidStudentUserProfileID: m.MasjidStudentUserProfileID,

		MasjidStudentSlug:   m.MasjidStudentSlug,
		MasjidStudentCode:   m.MasjidStudentCode,
		MasjidStudentStatus: m.MasjidStudentStatus,
		MasjidStudentNote:   m.MasjidStudentNote,

		MasjidStudentJoinedAt: m.MasjidStudentJoinedAt,
		MasjidStudentLeftAt:   m.MasjidStudentLeftAt,

		// snapshots users_profile
		MasjidStudentUserProfileNameSnapshot:              m.MasjidStudentUserProfileNameSnapshot,
		MasjidStudentUserProfileAvatarURLSnapshot:         m.MasjidStudentUserProfileAvatarURLSnapshot,
		MasjidStudentUserProfileWhatsappURLSnapshot:       m.MasjidStudentUserProfileWhatsappURLSnapshot,
		MasjidStudentUserProfileParentNameSnapshot:        m.MasjidStudentUserProfileParentNameSnapshot,
		MasjidStudentUserProfileParentWhatsappURLSnapshot: m.MasjidStudentUserProfileParentWhatsappURLSnapshot,

		// MASJID SNAPSHOT
		MasjidStudentMasjidNameSnapshot:          m.MasjidStudentMasjidNameSnapshot,
		MasjidStudentMasjidSlugSnapshot:          m.MasjidStudentMasjidSlugSnapshot,
		MasjidStudentMasjidLogoURLSnapshot:       m.MasjidStudentMasjidLogoURLSnapshot,
		MasjidStudentMasjidIconURLSnapshot:       m.MasjidStudentMasjidIconURLSnapshot,
		MasjidStudentMasjidBackgroundURLSnapshot: m.MasjidStudentMasjidBackgroundURLSnapshot,

		MasjidStudentSections: sections,

		MasjidStudentCreatedAt: m.MasjidStudentCreatedAt,
		MasjidStudentUpdatedAt: m.MasjidStudentUpdatedAt,
		MasjidStudentDeletedAt: delAt,
	}
}
