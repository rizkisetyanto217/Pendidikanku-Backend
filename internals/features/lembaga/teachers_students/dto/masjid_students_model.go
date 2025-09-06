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
   ========================================================= */

type MasjidStudentCreateReq struct {
	MasjidStudentMasjidID uuid.UUID `json:"masjid_student_masjid_id"`
	MasjidStudentUserID   uuid.UUID `json:"masjid_student_user_id"`

	MasjidStudentCode   *string `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `json:"masjid_student_status"`
	MasjidStudentNote   *string `json:"masjid_student_note,omitempty"`
}

func (r *MasjidStudentCreateReq) Normalize() {
	r.MasjidStudentStatus = strings.ToLower(strings.TrimSpace(r.MasjidStudentStatus))
	if r.MasjidStudentStatus == "" {
		r.MasjidStudentStatus = model.MasjidStudentStatusActive
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
	switch r.MasjidStudentStatus {
	case model.MasjidStudentStatusActive, model.MasjidStudentStatusInactive, model.MasjidStudentStatusAlumni:
		return nil
	}
	return errors.New("invalid masjid_student_status")
}

func (r *MasjidStudentCreateReq) ToModel() *model.MasjidStudentModel {
	return &model.MasjidStudentModel{
		MasjidStudentMasjidID: r.MasjidStudentMasjidID,
		MasjidStudentUserID:   r.MasjidStudentUserID,
		MasjidStudentCode:     r.MasjidStudentCode,
		MasjidStudentStatus:   r.MasjidStudentStatus,
		MasjidStudentNote:     r.MasjidStudentNote,
	}
}

/* =========================================================
   REQUEST: UPDATE (PUT, full)
   ========================================================= */

type MasjidStudentUpdateReq struct {
	MasjidStudentCode   *string `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus string  `json:"masjid_student_status"`
	MasjidStudentNote   *string `json:"masjid_student_note,omitempty"`
}

func (r *MasjidStudentUpdateReq) Normalize() {
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
	switch r.MasjidStudentStatus {
	case model.MasjidStudentStatusActive, model.MasjidStudentStatusInactive, model.MasjidStudentStatusAlumni:
		return nil
	}
	return errors.New("invalid masjid_student_status")
}

func (r *MasjidStudentUpdateReq) Apply(m *model.MasjidStudentModel) {
	m.MasjidStudentCode = r.MasjidStudentCode
	m.MasjidStudentStatus = r.MasjidStudentStatus
	m.MasjidStudentNote = r.MasjidStudentNote
}

/* =========================================================
   REQUEST: PATCH (partial)
   ========================================================= */

type MasjidStudentPatchReq struct {
	MasjidStudentCode   *PatchField[*string] `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus *PatchField[string]  `json:"masjid_student_status,omitempty"`
	MasjidStudentNote   *PatchField[*string] `json:"masjid_student_note,omitempty"`
}

func (r *MasjidStudentPatchReq) Normalize() {
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
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		switch r.MasjidStudentStatus.Value {
		case model.MasjidStudentStatusActive, model.MasjidStudentStatusInactive, model.MasjidStudentStatusAlumni:
			return nil
		default:
			return errors.New("invalid masjid_student_status")
		}
	}
	return nil
}

func (r *MasjidStudentPatchReq) Apply(m *model.MasjidStudentModel) {
	if r.MasjidStudentCode != nil && r.MasjidStudentCode.Set {
		m.MasjidStudentCode = r.MasjidStudentCode.Value
	}
	if r.MasjidStudentStatus != nil && r.MasjidStudentStatus.Set {
		m.MasjidStudentStatus = r.MasjidStudentStatus.Value
	}
	if r.MasjidStudentNote != nil && r.MasjidStudentNote.Set {
		m.MasjidStudentNote = r.MasjidStudentNote.Value
	}
}

/* =========================================================
   RESPONSE DTO
   ========================================================= */

type MasjidStudentResp struct {
	MasjidStudentID        uuid.UUID  `json:"masjid_student_id"`
	MasjidStudentMasjidID  uuid.UUID  `json:"masjid_student_masjid_id"`
	MasjidStudentUserID    uuid.UUID  `json:"masjid_student_user_id"`
	MasjidStudentCode      *string    `json:"masjid_student_code,omitempty"`
	MasjidStudentStatus    string     `json:"masjid_student_status"`
	MasjidStudentNote      *string    `json:"masjid_student_note,omitempty"`
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
		MasjidStudentCode:      m.MasjidStudentCode,
		MasjidStudentStatus:    m.MasjidStudentStatus,
		MasjidStudentNote:      m.MasjidStudentNote,
		MasjidStudentCreatedAt: m.MasjidStudentCreatedAt,
		MasjidStudentUpdatedAt: m.MasjidStudentUpdatedAt,
		MasjidStudentDeletedAt: delAt,
	}
}
