package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/lembaga/teachers_students/model"
)

/* =========================================================
   GENERIC: PatchField[T]
========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"set"`
	Value T    `json:"value,omitempty"`
}

func (p *PatchField[T]) IsZero() bool {
	return p == nil || !p.Set
}

/* =========================================================
   REQUEST: CREATE
   - slug wajib (NOT NULL UNIQUE)
   - status default "active"
========================================================= */

type MasjidStudentCreateReq struct {
	MasjidStudentMasjidID uuid.UUID `json:"masjid_student_masjid_id"`
	MasjidStudentUserID   uuid.UUID `json:"masjid_student_user_id"`

	MasjidStudentSlug string `json:"masjid_student_slug"` // required

	MasjidStudentCode *string `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string `json:"masjid_student_status"` // optional; defaults to active
	MasjidStudentNote *string `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `json:"masjid_student_left_at,omitempty"`
}

func (r *MasjidStudentCreateReq) Normalize() {
	// slug: trim & lower (opsional: sesuai kebijakan)
	r.MasjidStudentSlug = strings.ToLower(strings.TrimSpace(r.MasjidStudentSlug))

	// status
	r.MasjidStudentStatus = strings.ToLower(strings.TrimSpace(r.MasjidStudentStatus))
	if r.MasjidStudentStatus == "" {
		r.MasjidStudentStatus = model.MasjidStudentStatusActive
	}

	// code
	if r.MasjidStudentCode != nil {
		c := strings.TrimSpace(*r.MasjidStudentCode)
		if c == "" {
			r.MasjidStudentCode = nil
		} else {
			r.MasjidStudentCode = &c
		}
	}

	// note
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
	case model.MasjidStudentStatusActive, model.MasjidStudentStatusInactive, model.MasjidStudentStatusAlumni:
	default:
		return errors.New("invalid masjid_student_status")
	}
	// joined <= left (jika keduanya ada)
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentLeftAt != nil {
		if r.MasjidStudentLeftAt.Before(*r.MasjidStudentJoinedAt) {
			return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
		}
	}
	return nil
}

func (r *MasjidStudentCreateReq) ToModel() *model.MasjidStudentModel {
	return &model.MasjidStudentModel{
		MasjidStudentMasjidID: r.MasjidStudentMasjidID,
		MasjidStudentUserID:   r.MasjidStudentUserID,

		MasjidStudentSlug: r.MasjidStudentSlug,

		MasjidStudentCode:   r.MasjidStudentCode,
		MasjidStudentStatus: r.MasjidStudentStatus,
		MasjidStudentNote:   r.MasjidStudentNote,

		MasjidStudentJoinedAt: r.MasjidStudentJoinedAt,
		MasjidStudentLeftAt:   r.MasjidStudentLeftAt,
	}
}

/* =========================================================
   REQUEST: UPDATE (PUT, full)
   - Anggap slug ikut dikirim (kalau slug immutable di bisnis,
     boleh di-skip dari PUT atau diabaikan saat Apply)
========================================================= */

type MasjidStudentUpdateReq struct {
	MasjidStudentSlug  string  `json:"masjid_student_slug"` // required (kalau immutable: abaikan di Apply)
	MasjidStudentCode  *string `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `json:"masjid_student_status"`
	MasjidStudentNote  *string `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `json:"masjid_student_left_at,omitempty"`
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
	case model.MasjidStudentStatusActive, model.MasjidStudentStatusInactive, model.MasjidStudentStatusAlumni:
	default:
		return errors.New("invalid masjid_student_status")
	}
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentLeftAt != nil {
		if r.MasjidStudentLeftAt.Before(*r.MasjidStudentJoinedAt) {
			return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
		}
	}
	return nil
}

func (r *MasjidStudentUpdateReq) Apply(m *model.MasjidStudentModel) {
	// jika slug immutable, comment baris berikut
	m.MasjidStudentSlug = r.MasjidStudentSlug

	m.MasjidStudentCode = r.MasjidStudentCode
	m.MasjidStudentStatus = r.MasjidStudentStatus
	m.MasjidStudentNote = r.MasjidStudentNote

	m.MasjidStudentJoinedAt = r.MasjidStudentJoinedAt
	m.MasjidStudentLeftAt = r.MasjidStudentLeftAt
}

/* =========================================================
   REQUEST: PATCH (partial)
   - Code & Note pakai pointer (bisa set null)
   - Slug (string) tidak boleh kosong saat set
   - Tanggal bisa set null (hapus), atau set nilai baru
========================================================= */

type MasjidStudentPatchReq struct {
	MasjidStudentSlug  *PatchField[string]   `json:"masjid_student_slug,omitempty"`
	MasjidStudentCode  *PatchField[*string]  `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus *PatchField[string]   `json:"masjid_student_status,omitempty"`
	MasjidStudentNote  *PatchField[*string]  `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt *PatchField[*time.Time] `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *PatchField[*time.Time] `json:"masjid_student_left_at,omitempty"`
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
	// tanggal tidak perlu normalisasi khusus
}

func (r *MasjidStudentPatchReq) Validate() error {
	if r.MasjidStudentSlug != nil && r.MasjidStudentSlug.Set {
		if r.MasjidStudentSlug.Value == "" {
			return errors.New("masjid_student_slug tidak boleh kosong saat di-set")
		}
	}
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		switch r.MasjidStudentStatus.Value {
		case model.MasjidStudentStatusActive, model.MasjidStudentStatusInactive, model.MasjidStudentStatusAlumni:
		default:
			return errors.New("invalid masjid_student_status")
		}
	}
	// Kalau joined & left sama-sama di-set dan non-nil → validasi ordering
	if r.MasjidStudentJoinedAt != nil && r.MasjidStudentJoinedAt.Set &&
		r.MasjidStudentLeftAt != nil && r.MasjidStudentLeftAt.Set &&
		r.MasjidStudentJoinedAt.Value != nil && r.MasjidStudentLeftAt.Value != nil {

		if r.MasjidStudentLeftAt.Value.Before(*r.MasjidStudentJoinedAt.Value) {
			return errors.New("masjid_student_left_at tidak boleh lebih awal dari masjid_student_joined_at")
		}
	}
	return nil
}

func (r *MasjidStudentPatchReq) Apply(m *model.MasjidStudentModel) {
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
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type MasjidStudentResp struct {
	MasjidStudentID        uuid.UUID  `json:"masjid_student_id"`
	MasjidStudentMasjidID  uuid.UUID  `json:"masjid_student_masjid_id"`
	MasjidStudentUserID    uuid.UUID  `json:"masjid_student_user_id"`

	MasjidStudentSlug      string     `json:"masjid_student_slug"`
	MasjidStudentCode      *string    `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus    string     `json:"masjid_student_status"`
	MasjidStudentNote      *string    `json:"masjid_student_note,omitempty"`

	MasjidStudentJoinedAt  *time.Time `json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt    *time.Time `json:"masjid_student_left_at,omitempty"`

	MasjidStudentCreatedAt time.Time  `json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time  `json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt *time.Time `json:"masjid_student_deleted_at,omitempty"`
}

func FromModel(m *model.MasjidStudentModel) MasjidStudentResp {
	var delAt *time.Time
	if m.MasjidStudentDeletedAt.Valid {
		t := m.MasjidStudentDeletedAt.Time
		delAt = &t
	}
	return MasjidStudentResp{
		MasjidStudentID:        m.MasjidStudentID,
		MasjidStudentMasjidID:  m.MasjidStudentMasjidID,
		MasjidStudentUserID:    m.MasjidStudentUserID,

		MasjidStudentSlug:      m.MasjidStudentSlug,
		MasjidStudentCode:      m.MasjidStudentCode,
		MasjidStudentStatus:    m.MasjidStudentStatus,
		MasjidStudentNote:      m.MasjidStudentNote,

		MasjidStudentJoinedAt:  m.MasjidStudentJoinedAt,
		MasjidStudentLeftAt:    m.MasjidStudentLeftAt,

		MasjidStudentCreatedAt: m.MasjidStudentCreatedAt,
		MasjidStudentUpdatedAt: m.MasjidStudentUpdatedAt,
		MasjidStudentDeletedAt: delAt,
	}
}
