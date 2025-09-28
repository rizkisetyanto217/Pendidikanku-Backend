package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "masjidku_backend/internals/features/school/classes/class_sections/model"
)

/* =========================================================
   REQUEST: CREATE
========================================================= */

type UserClassSectionCreateReq struct {
	UserClassSectionMasjidStudentID uuid.UUID `json:"user_class_section_masjid_student_id"`
	UserClassSectionSectionID       uuid.UUID `json:"user_class_section_section_id"`
	UserClassSectionMasjidID        uuid.UUID `json:"user_class_section_masjid_id"`

	UserClassSectionStatus string          `json:"user_class_section_status,omitempty"` // default: active
	UserClassSectionResult *string         `json:"user_class_section_result,omitempty"`
	UserClassSectionFee    *datatypes.JSON `json:"user_class_section_fee_snapshot,omitempty"`

	UserClassSectionAssignedAt   *time.Time `json:"user_class_section_assigned_at,omitempty"`
	UserClassSectionUnassignedAt *time.Time `json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *time.Time `json:"user_class_section_completed_at,omitempty"`
}

func (r *UserClassSectionCreateReq) Normalize() {
	if r.UserClassSectionStatus == "" {
		r.UserClassSectionStatus = string(model.UserClassSectionActive)
	}
	r.UserClassSectionStatus = strings.ToLower(r.UserClassSectionStatus)
	if r.UserClassSectionResult != nil {
		res := strings.ToLower(strings.TrimSpace(*r.UserClassSectionResult))
		if res == "" {
			r.UserClassSectionResult = nil
		} else {
			r.UserClassSectionResult = &res
		}
	}
}

func (r *UserClassSectionCreateReq) Validate() error {
	switch r.UserClassSectionStatus {
	case string(model.UserClassSectionActive),
		string(model.UserClassSectionInactive),
		string(model.UserClassSectionCompleted):
	default:
		return errors.New("invalid user_class_section_status")
	}
	if r.UserClassSectionStatus == string(model.UserClassSectionCompleted) {
		if r.UserClassSectionResult == nil {
			return errors.New("user_class_section_result wajib diisi jika status completed")
		}
		if r.UserClassSectionCompletedAt == nil {
			return errors.New("user_class_section_completed_at wajib diisi jika status completed")
		}
	}
	return nil
}

func (r *UserClassSectionCreateReq) ToModel() *model.UserClassSection {
	m := &model.UserClassSection{
		UserClassSectionMasjidStudentID: r.UserClassSectionMasjidStudentID,
		UserClassSectionSectionID:       r.UserClassSectionSectionID,
		UserClassSectionMasjidID:        r.UserClassSectionMasjidID,
		UserClassSectionStatus:          model.UserClassSectionStatus(r.UserClassSectionStatus),
		UserClassSectionFeeSnapshot:     datatypes.JSON([]byte(`{}`)),
	}
	if r.UserClassSectionResult != nil {
		res := model.UserClassSectionResult(*r.UserClassSectionResult)
		m.UserClassSectionResult = &res
	}
	if r.UserClassSectionFee != nil {
		m.UserClassSectionFeeSnapshot = *r.UserClassSectionFee
	}
	if r.UserClassSectionAssignedAt != nil {
		m.UserClassSectionAssignedAt = *r.UserClassSectionAssignedAt
	}
	if r.UserClassSectionUnassignedAt != nil {
		m.UserClassSectionUnassignedAt = r.UserClassSectionUnassignedAt
	}
	if r.UserClassSectionCompletedAt != nil {
		m.UserClassSectionCompletedAt = r.UserClassSectionCompletedAt
	}
	return m
}

/* =========================================================
   REQUEST: PATCH (partial update)
========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"set"`
	Value T    `json:"value,omitempty"`
}

func (p *PatchField[T]) IsZero() bool { return p == nil || !p.Set }

type UserClassSectionPatchReq struct {
	UserClassSectionStatus *PatchField[string]          `json:"user_class_section_status,omitempty"`
	UserClassSectionResult *PatchField[*string]         `json:"user_class_section_result,omitempty"`
	UserClassSectionFee    *PatchField[*datatypes.JSON] `json:"user_class_section_fee_snapshot,omitempty"`

	UserClassSectionAssignedAt   *PatchField[*time.Time] `json:"user_class_section_assigned_at,omitempty"`
	UserClassSectionUnassignedAt *PatchField[*time.Time] `json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *PatchField[*time.Time] `json:"user_class_section_completed_at,omitempty"`
}

func (r *UserClassSectionPatchReq) Apply(m *model.UserClassSection) {
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set {
		m.UserClassSectionStatus = model.UserClassSectionStatus(r.UserClassSectionStatus.Value)
	}
	if r.UserClassSectionResult != nil && r.UserClassSectionResult.Set {
		if r.UserClassSectionResult.Value != nil {
			res := model.UserClassSectionResult(*r.UserClassSectionResult.Value)
			m.UserClassSectionResult = &res
		} else {
			m.UserClassSectionResult = nil
		}
	}
	if r.UserClassSectionFee != nil && r.UserClassSectionFee.Set {
		if r.UserClassSectionFee.Value != nil {
			m.UserClassSectionFeeSnapshot = *r.UserClassSectionFee.Value
		} else {
			m.UserClassSectionFeeSnapshot = datatypes.JSON([]byte(`{}`))
		}
	}

	if r.UserClassSectionAssignedAt != nil && r.UserClassSectionAssignedAt.Set {
		if r.UserClassSectionAssignedAt.Value != nil {
			m.UserClassSectionAssignedAt = *r.UserClassSectionAssignedAt.Value
		}
	}
	if r.UserClassSectionUnassignedAt != nil && r.UserClassSectionUnassignedAt.Set {
		m.UserClassSectionUnassignedAt = r.UserClassSectionUnassignedAt.Value
	}
	if r.UserClassSectionCompletedAt != nil && r.UserClassSectionCompletedAt.Set {
		m.UserClassSectionCompletedAt = r.UserClassSectionCompletedAt.Value
	}
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type UserClassSectionResp struct {
	UserClassSectionID uuid.UUID `json:"user_class_section_id"`

	UserClassSectionMasjidStudentID uuid.UUID `json:"user_class_section_masjid_student_id"`
	UserClassSectionSectionID       uuid.UUID `json:"user_class_section_section_id"`
	UserClassSectionMasjidID        uuid.UUID `json:"user_class_section_masjid_id"`

	UserClassSectionStatus string          `json:"user_class_section_status"`
	UserClassSectionResult *string         `json:"user_class_section_result,omitempty"`
	UserClassSectionFee    *datatypes.JSON `json:"user_class_section_fee_snapshot,omitempty"`

	UserClassSectionAssignedAt   time.Time  `json:"user_class_section_assigned_at"`
	UserClassSectionUnassignedAt *time.Time `json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *time.Time `json:"user_class_section_completed_at,omitempty"`

	UserClassSectionCreatedAt time.Time  `json:"user_class_section_created_at"`
	UserClassSectionUpdatedAt time.Time  `json:"user_class_section_updated_at"`
	UserClassSectionDeletedAt *time.Time `json:"user_class_section_deleted_at,omitempty"`
}

func FromModel(m *model.UserClassSection) UserClassSectionResp {
	var res *string
	if m.UserClassSectionResult != nil {
		v := string(*m.UserClassSectionResult)
		res = &v
	}
	var delAt *time.Time
	if m.UserClassSectionDeletedAt.Valid {
		t := m.UserClassSectionDeletedAt.Time
		delAt = &t
	}
	return UserClassSectionResp{
		UserClassSectionID: m.UserClassSectionID,

		UserClassSectionMasjidStudentID: m.UserClassSectionMasjidStudentID,
		UserClassSectionSectionID:       m.UserClassSectionSectionID,
		UserClassSectionMasjidID:        m.UserClassSectionMasjidID,

		UserClassSectionStatus: string(m.UserClassSectionStatus),
		UserClassSectionResult: res,
		UserClassSectionFee:    &m.UserClassSectionFeeSnapshot,

		UserClassSectionAssignedAt:   m.UserClassSectionAssignedAt,
		UserClassSectionUnassignedAt: m.UserClassSectionUnassignedAt,
		UserClassSectionCompletedAt:  m.UserClassSectionCompletedAt,

		UserClassSectionCreatedAt: m.UserClassSectionCreatedAt,
		UserClassSectionUpdatedAt: m.UserClassSectionUpdatedAt,
		UserClassSectionDeletedAt: delAt,
	}
}



func (r *UserClassSectionPatchReq) Normalize() {
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set {
		r.UserClassSectionStatus.Value = strings.ToLower(strings.TrimSpace(r.UserClassSectionStatus.Value))
	}
	if r.UserClassSectionResult != nil && r.UserClassSectionResult.Set && r.UserClassSectionResult.Value != nil {
		res := strings.ToLower(strings.TrimSpace(*r.UserClassSectionResult.Value))
		if res == "" {
			r.UserClassSectionResult.Value = nil
		} else {
			r.UserClassSectionResult.Value = &res
		}
	}
}

func (r *UserClassSectionPatchReq) Validate() error {
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set {
		switch r.UserClassSectionStatus.Value {
		case "active", "inactive", "completed":
		default:
			return errors.New("invalid status")
		}
	}

	// kalau completed → wajib ada completed_at dan result
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set &&
		r.UserClassSectionStatus.Value == "completed" {

		if r.UserClassSectionCompletedAt == nil || !r.UserClassSectionCompletedAt.Set || r.UserClassSectionCompletedAt.Value == nil {
			return errors.New("completed_at wajib diisi jika status completed")
		}
		if r.UserClassSectionResult == nil || !r.UserClassSectionResult.Set || r.UserClassSectionResult.Value == nil {
			return errors.New("result wajib diisi jika status completed")
		}
	}

	// validasi tanggal unassigned >= assigned
	if r.UserClassSectionAssignedAt != nil && r.UserClassSectionAssignedAt.Set &&
		r.UserClassSectionUnassignedAt != nil && r.UserClassSectionUnassignedAt.Set &&
		r.UserClassSectionAssignedAt.Value != nil && r.UserClassSectionUnassignedAt.Value != nil {

		if r.UserClassSectionUnassignedAt.Value.Before(*r.UserClassSectionAssignedAt.Value) {
			return errors.New("unassigned_at tidak boleh lebih awal dari assigned_at")
		}
	}

	return nil
}

