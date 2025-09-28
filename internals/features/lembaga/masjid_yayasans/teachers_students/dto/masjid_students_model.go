package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"
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
   REQUEST: CREATE
========================================================= */

type MasjidStudentCreateReq struct {
	MasjidStudentMasjidID uuid.UUID `json:"masjid_student_masjid_id"`
	MasjidStudentUserID   uuid.UUID `json:"masjid_student_user_id"`

	MasjidStudentSlug string `json:"masjid_student_slug"` // required

	MasjidStudentCode   *string `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `json:"masjid_student_status"` // optional; defaults to "active"
	MasjidStudentNote   *string `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `json:"masjid_student_left_at,omitempty"`

	// Snapshots opsional di saat create
	MasjidStudentNameUserSnapshot              *string `json:"masjid_student_name_user_snapshot,omitempty"`
	MasjidStudentAvatarURLUserSnapshot         *string `json:"masjid_student_avatar_url_user_snapshot,omitempty"`
	MasjidStudentWhatsappURLUserSnapshot       *string `json:"masjid_student_whatsapp_url_user_snapshot,omitempty"`
	MasjidStudentParentNameUserSnapshot        *string `json:"masjid_student_parent_name_user_snapshot,omitempty"`
	MasjidStudentParentWhatsappURLUserSnapshot *string `json:"masjid_student_parent_whatsapp_url_user_snapshot,omitempty"`
}

func (r *MasjidStudentCreateReq) Normalize() {
	r.MasjidStudentSlug = strings.ToLower(strings.TrimSpace(r.MasjidStudentSlug))

	r.MasjidStudentStatus = strings.ToLower(strings.TrimSpace(r.MasjidStudentStatus))
	if r.MasjidStudentStatus == "" {
		r.MasjidStudentStatus = string(model.MasjidStudentActive)
	}

	if r.MasjidStudentCode != nil {
		c := strings.TrimSpace(*r.MasjidStudentCode)
		if c == "" {
			r.MasjidStudentCode = nil
		} else {
			r.MasjidStudentCode = &c
		}
	}

	if r.MasjidStudentNote != nil {
		n := strings.TrimSpace(*r.MasjidStudentNote)
		if n == "" {
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
	switch r.MasjidStudentStatus {
	case string(model.MasjidStudentActive), string(model.MasjidStudentInactive), string(model.MasjidStudentAlumni):
	default:
		return errors.New("invalid masjid_student_status")
	}
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentLeftAt != nil &&
		r.MasjidStudentLeftAt.Before(*r.MasjidStudentJoinedAt) {
		return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
	}
	return nil
}

func (r *MasjidStudentCreateReq) ToModel() *model.MasjidStudent {
	return &model.MasjidStudent{
		MasjidStudentMasjidID: r.MasjidStudentMasjidID,
		MasjidStudentUserID:   r.MasjidStudentUserID,
		MasjidStudentSlug:     r.MasjidStudentSlug,
		MasjidStudentCode:     r.MasjidStudentCode,
		MasjidStudentStatus:   model.MasjidStudentStatus(r.MasjidStudentStatus),
		MasjidStudentNote:     r.MasjidStudentNote,
		MasjidStudentJoinedAt: r.MasjidStudentJoinedAt,
		MasjidStudentLeftAt:   r.MasjidStudentLeftAt,

		// snapshot langsung copy
		MasjidStudentNameUserSnapshot:              r.MasjidStudentNameUserSnapshot,
		MasjidStudentAvatarURLUserSnapshot:         r.MasjidStudentAvatarURLUserSnapshot,
		MasjidStudentWhatsappURLUserSnapshot:       r.MasjidStudentWhatsappURLUserSnapshot,
		MasjidStudentParentNameUserSnapshot:        r.MasjidStudentParentNameUserSnapshot,
		MasjidStudentParentWhatsappURLUserSnapshot: r.MasjidStudentParentWhatsappURLUserSnapshot,
	}
}

/* =========================================================
   REQUEST: UPDATE (PUT, full)
========================================================= */

type MasjidStudentUpdateReq struct {
	MasjidStudentSlug   string  `json:"masjid_student_slug"`
	MasjidStudentCode   *string `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `json:"masjid_student_status"`
	MasjidStudentNote   *string `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `json:"masjid_student_left_at,omitempty"`

	// snapshots
	MasjidStudentNameUserSnapshot              *string `json:"masjid_student_name_user_snapshot,omitempty"`
	MasjidStudentAvatarURLUserSnapshot         *string `json:"masjid_student_avatar_url_user_snapshot,omitempty"`
	MasjidStudentWhatsappURLUserSnapshot       *string `json:"masjid_student_whatsapp_url_user_snapshot,omitempty"`
	MasjidStudentParentNameUserSnapshot        *string `json:"masjid_student_parent_name_user_snapshot,omitempty"`
	MasjidStudentParentWhatsappURLUserSnapshot *string `json:"masjid_student_parent_whatsapp_url_user_snapshot,omitempty"`
}

func (r *MasjidStudentUpdateReq) Normalize() {
	r.MasjidStudentSlug = strings.ToLower(strings.TrimSpace(r.MasjidStudentSlug))
	r.MasjidStudentStatus = strings.ToLower(strings.TrimSpace(r.MasjidStudentStatus))
	if r.MasjidStudentCode != nil {
		c := strings.TrimSpace(*r.MasjidStudentCode)
		if c == "" {
			r.MasjidStudentCode = nil
		} else {
			r.MasjidStudentCode = &c
		}
	}
	if r.MasjidStudentNote != nil {
		n := strings.TrimSpace(*r.MasjidStudentNote)
		if n == "" {
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
	switch r.MasjidStudentStatus {
	case string(model.MasjidStudentActive), string(model.MasjidStudentInactive), string(model.MasjidStudentAlumni):
	default:
		return errors.New("invalid masjid_student_status")
	}
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentLeftAt != nil &&
		r.MasjidStudentLeftAt.Before(*r.MasjidStudentJoinedAt) {
		return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
	}
	return nil
}

func (r *MasjidStudentUpdateReq) Apply(m *model.MasjidStudent) {
	m.MasjidStudentSlug = r.MasjidStudentSlug
	m.MasjidStudentCode = r.MasjidStudentCode
	m.MasjidStudentStatus = model.MasjidStudentStatus(r.MasjidStudentStatus)
	m.MasjidStudentNote = r.MasjidStudentNote
	m.MasjidStudentJoinedAt = r.MasjidStudentJoinedAt
	m.MasjidStudentLeftAt = r.MasjidStudentLeftAt

	m.MasjidStudentNameUserSnapshot = r.MasjidStudentNameUserSnapshot
	m.MasjidStudentAvatarURLUserSnapshot = r.MasjidStudentAvatarURLUserSnapshot
	m.MasjidStudentWhatsappURLUserSnapshot = r.MasjidStudentWhatsappURLUserSnapshot
	m.MasjidStudentParentNameUserSnapshot = r.MasjidStudentParentNameUserSnapshot
	m.MasjidStudentParentWhatsappURLUserSnapshot = r.MasjidStudentParentWhatsappURLUserSnapshot
}

/* =========================================================
   REQUEST: PATCH (partial)
========================================================= */

type MasjidStudentPatchReq struct {
	MasjidStudentSlug   *PatchField[string]  `json:"masjid_student_slug,omitempty"`
	MasjidStudentCode   *PatchField[*string] `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus *PatchField[string]  `json:"masjid_student_status,omitempty"`
	MasjidStudentNote   *PatchField[*string] `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *PatchField[*time.Time] `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *PatchField[*time.Time] `json:"masjid_student_left_at,omitempty"`

	// snapshot
	MasjidStudentNameUserSnapshot              *PatchField[*string] `json:"masjid_student_name_user_snapshot,omitempty"`
	MasjidStudentAvatarURLUserSnapshot         *PatchField[*string] `json:"masjid_student_avatar_url_user_snapshot,omitempty"`
	MasjidStudentWhatsappURLUserSnapshot       *PatchField[*string] `json:"masjid_student_whatsapp_url_user_snapshot,omitempty"`
	MasjidStudentParentNameUserSnapshot        *PatchField[*string] `json:"masjid_student_parent_name_user_snapshot,omitempty"`
	MasjidStudentParentWhatsappURLUserSnapshot *PatchField[*string] `json:"masjid_student_parent_whatsapp_url_user_snapshot,omitempty"`
}

func (r *MasjidStudentPatchReq) Normalize() {
	if r.MasjidStudentSlug != nil && r.MasjidStudentSlug.Set {
		r.MasjidStudentSlug.Value = strings.ToLower(strings.TrimSpace(r.MasjidStudentSlug.Value))
	}
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		r.MasjidStudentStatus.Value = strings.ToLower(strings.TrimSpace(r.MasjidStudentStatus.Value))
	}
	if r.MasjidStudentCode != nil && r.MasjidStudentCode.Set && r.MasjidStudentCode.Value != nil {
		c := strings.TrimSpace(*r.MasjidStudentCode.Value)
		if c == "" {
			r.MasjidStudentCode.Value = nil
		} else {
			r.MasjidStudentCode.Value = &c
		}
	}
	if r.MasjidStudentNote != nil && r.MasjidStudentNote.Set && r.MasjidStudentNote.Value != nil {
		n := strings.TrimSpace(*r.MasjidStudentNote.Value)
		if n == "" {
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
		switch r.MasjidStudentStatus.Value {
		case string(model.MasjidStudentActive), string(model.MasjidStudentInactive), string(model.MasjidStudentAlumni):
		default:
			return errors.New("invalid masjid_student_status")
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

func (r *MasjidStudentPatchReq) Apply(m *model.MasjidStudent) {
	if r.MasjidStudentSlug != nil && r.MasjidStudentSlug.Set {
		m.MasjidStudentSlug = r.MasjidStudentSlug.Value
	}
	if r.MasjidStudentCode != nil && r.MasjidStudentCode.Set {
		m.MasjidStudentCode = r.MasjidStudentCode.Value
	}
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		m.MasjidStudentStatus = model.MasjidStudentStatus(r.MasjidStudentStatus.Value)
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

	if r.MasjidStudentNameUserSnapshot != nil && r.MasjidStudentNameUserSnapshot.Set {
		m.MasjidStudentNameUserSnapshot = r.MasjidStudentNameUserSnapshot.Value
	}
	if r.MasjidStudentAvatarURLUserSnapshot != nil && r.MasjidStudentAvatarURLUserSnapshot.Set {
		m.MasjidStudentAvatarURLUserSnapshot = r.MasjidStudentAvatarURLUserSnapshot.Value
	}
	if r.MasjidStudentWhatsappURLUserSnapshot != nil && r.MasjidStudentWhatsappURLUserSnapshot.Set {
		m.MasjidStudentWhatsappURLUserSnapshot = r.MasjidStudentWhatsappURLUserSnapshot.Value
	}
	if r.MasjidStudentParentNameUserSnapshot != nil && r.MasjidStudentParentNameUserSnapshot.Set {
		m.MasjidStudentParentNameUserSnapshot = r.MasjidStudentParentNameUserSnapshot.Value
	}
	if r.MasjidStudentParentWhatsappURLUserSnapshot != nil && r.MasjidStudentParentWhatsappURLUserSnapshot.Set {
		m.MasjidStudentParentWhatsappURLUserSnapshot = r.MasjidStudentParentWhatsappURLUserSnapshot.Value
	}
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type MasjidStudentResp struct {
	MasjidStudentID       uuid.UUID `json:"masjid_student_id"`
	MasjidStudentMasjidID uuid.UUID `json:"masjid_student_masjid_id"`
	MasjidStudentUserID   uuid.UUID `json:"masjid_student_user_id"`

	MasjidStudentSlug   string  `json:"masjid_student_slug"`
	MasjidStudentCode   *string `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `json:"masjid_student_status"`
	MasjidStudentNote   *string `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `json:"masjid_student_left_at,omitempty"`

	// snapshots
	MasjidStudentNameUserSnapshot              *string `json:"masjid_student_name_user_snapshot,omitempty"`
	MasjidStudentAvatarURLUserSnapshot         *string `json:"masjid_student_avatar_url_user_snapshot,omitempty"`
	MasjidStudentWhatsappURLUserSnapshot       *string `json:"masjid_student_whatsapp_url_user_snapshot,omitempty"`
	MasjidStudentParentNameUserSnapshot        *string `json:"masjid_student_parent_name_user_snapshot,omitempty"`
	MasjidStudentParentWhatsappURLUserSnapshot *string `json:"masjid_student_parent_whatsapp_url_user_snapshot,omitempty"`

	MasjidStudentCreatedAt time.Time  `json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time  `json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt *time.Time `json:"masjid_student_deleted_at,omitempty"`
}

func FromModel(m *model.MasjidStudent) MasjidStudentResp {
	var delAt *time.Time
	if m.MasjidStudentDeletedAt.Valid {
		t := m.MasjidStudentDeletedAt.Time
		delAt = &t
	}
	return MasjidStudentResp{
		MasjidStudentID:       m.MasjidStudentID,
		MasjidStudentMasjidID: m.MasjidStudentMasjidID,
		MasjidStudentUserID:   m.MasjidStudentUserID,

		MasjidStudentSlug:   m.MasjidStudentSlug,
		MasjidStudentCode:   m.MasjidStudentCode,
		MasjidStudentStatus: string(m.MasjidStudentStatus),
		MasjidStudentNote:   m.MasjidStudentNote,

		MasjidStudentJoinedAt: m.MasjidStudentJoinedAt,
		MasjidStudentLeftAt:   m.MasjidStudentLeftAt,

		MasjidStudentNameUserSnapshot:              m.MasjidStudentNameUserSnapshot,
		MasjidStudentAvatarURLUserSnapshot:         m.MasjidStudentAvatarURLUserSnapshot,
		MasjidStudentWhatsappURLUserSnapshot:       m.MasjidStudentWhatsappURLUserSnapshot,
		MasjidStudentParentNameUserSnapshot:        m.MasjidStudentParentNameUserSnapshot,
		MasjidStudentParentWhatsappURLUserSnapshot: m.MasjidStudentParentWhatsappURLUserSnapshot,

		MasjidStudentCreatedAt: m.MasjidStudentCreatedAt,
		MasjidStudentUpdatedAt: m.MasjidStudentUpdatedAt,
		MasjidStudentDeletedAt: delAt,
	}
}
