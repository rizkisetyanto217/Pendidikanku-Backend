// file: internals/features/finance/general_billings/dto/user_general_billing_dto.go
package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "schoolku_backend/internals/features/finance/general_billings/model"
)

/* =========================================================
   PatchField tri-state (Unset / Null / Set(value))
   - Unset: field tidak dikirim -> tidak diubah
   - Null : field dikirim null  -> set ke NULL (kalau kolom nullable)
   - Set  : field ada nilainya  -> set ke nilai baru
   Catatan: untuk kolom model yg pointer (*T), gunakan PatchField[T] (bukan *T)
   agar Value bertipe *T (bukan **T).
   ========================================================= */

func ptr[T any](v T) *T { return &v }

/* =========================================================
   REQUEST: Create
   ========================================================= */

type CreateUserGeneralBillingRequest struct {
	UserGeneralBillingSchoolID        uuid.UUID  `json:"user_general_billing_school_id" validate:"required"`
	UserGeneralBillingSchoolStudentID *uuid.UUID `json:"user_general_billing_school_student_id"` // optional (minimal salah satu ini atau payer harus diisi)
	UserGeneralBillingPayerUserID     *uuid.UUID `json:"user_general_billing_payer_user_id"`     // optional (lihat rule di atas)

	UserGeneralBillingBillingID uuid.UUID `json:"user_general_billing_billing_id" validate:"required"`

	UserGeneralBillingAmountIDR int     `json:"user_general_billing_amount_idr" validate:"required,min=0"`
	UserGeneralBillingStatus    *string `json:"user_general_billing_status" validate:"omitempty,oneof=unpaid paid canceled"`

	UserGeneralBillingPaidAt           *time.Time     `json:"user_general_billing_paid_at"`
	UserGeneralBillingNote             *string        `json:"user_general_billing_note"`
	UserGeneralBillingMeta             map[string]any `json:"user_general_billing_meta"`
	UserGeneralBillingTitleSnapshot    *string        `json:"user_general_billing_title_snapshot"`
	UserGeneralBillingKindCodeSnapshot *string        `json:"user_general_billing_kind_code_snapshot"`
	UserGeneralBillingKindNameSnapshot *string        `json:"user_general_billing_kind_name_snapshot"`
}

func (r *CreateUserGeneralBillingRequest) Validate() error {
	// Minimal salah satu target harus ada: student atau payer
	if r.UserGeneralBillingSchoolStudentID == nil && r.UserGeneralBillingPayerUserID == nil {
		return errors.New("either user_general_billing_school_student_id or user_general_billing_payer_user_id must be provided")
	}
	return nil
}

func (r CreateUserGeneralBillingRequest) ToModel() model.UserGeneralBilling {
	status := model.UserGeneralBillingStatusUnpaid
	if r.UserGeneralBillingStatus != nil && *r.UserGeneralBillingStatus != "" {
		status = *r.UserGeneralBillingStatus
	}

	var meta datatypes.JSONMap
	if r.UserGeneralBillingMeta != nil {
		meta = datatypes.JSONMap(r.UserGeneralBillingMeta)
	}

	return model.UserGeneralBilling{
		UserGeneralBillingSchoolID:        r.UserGeneralBillingSchoolID,
		UserGeneralBillingSchoolStudentID: r.UserGeneralBillingSchoolStudentID,
		UserGeneralBillingPayerUserID:     r.UserGeneralBillingPayerUserID,
		UserGeneralBillingBillingID:       r.UserGeneralBillingBillingID,

		UserGeneralBillingAmountIDR: r.UserGeneralBillingAmountIDR,
		UserGeneralBillingStatus:    status,

		UserGeneralBillingPaidAt: r.UserGeneralBillingPaidAt,
		UserGeneralBillingNote:   r.UserGeneralBillingNote,

		UserGeneralBillingTitleSnapshot:    r.UserGeneralBillingTitleSnapshot,
		UserGeneralBillingKindCodeSnapshot: r.UserGeneralBillingKindCodeSnapshot,
		UserGeneralBillingKindNameSnapshot: r.UserGeneralBillingKindNameSnapshot,

		UserGeneralBillingMeta: meta,
	}
}

/* =========================================================
   REQUEST: Patch / Update (Partial)
   ========================================================= */

type PatchUserGeneralBillingRequest struct {
	// Tidak mengizinkan update SchoolID atau BillingID via patch (biasanya immutable)
	UserGeneralBillingSchoolStudentID PatchField[uuid.UUID] `json:"user_general_billing_school_student_id"` // boleh null-kan (cabut relasi)
	UserGeneralBillingPayerUserID     PatchField[uuid.UUID] `json:"user_general_billing_payer_user_id"`     // boleh null-kan

	UserGeneralBillingAmountIDR PatchField[int]       `json:"user_general_billing_amount_idr"`
	UserGeneralBillingStatus    PatchField[string]    `json:"user_general_billing_status"` // unpaid|paid|canceled
	UserGeneralBillingPaidAt    PatchField[time.Time] `json:"user_general_billing_paid_at"`
	UserGeneralBillingNote      PatchField[string]    `json:"user_general_billing_note"`

	UserGeneralBillingTitleSnapshot    PatchField[string] `json:"user_general_billing_title_snapshot"`
	UserGeneralBillingKindCodeSnapshot PatchField[string] `json:"user_general_billing_kind_code_snapshot"`
	UserGeneralBillingKindNameSnapshot PatchField[string] `json:"user_general_billing_kind_name_snapshot"`

	// Meta: bisa null (hapus), set object baru, atau tidak diubah
	UserGeneralBillingMeta PatchField[map[string]any] `json:"user_general_billing_meta"`
}

func (p PatchUserGeneralBillingRequest) ValidateAfterApply(m model.UserGeneralBilling) error {
	// Pastikan minimal salah satu tetap ada setelah patch (student/payer)
	if m.UserGeneralBillingSchoolStudentID == nil && m.UserGeneralBillingPayerUserID == nil {
		return errors.New("after patch, at least one of school_student_id or payer_user_id must be non-null")
	}
	// Validasi status kalau di-set
	if p.UserGeneralBillingStatus.Set && !p.UserGeneralBillingStatus.Null && p.UserGeneralBillingStatus.Value != nil {
		s := *p.UserGeneralBillingStatus.Value
		if s != model.UserGeneralBillingStatusUnpaid && s != model.UserGeneralBillingStatusPaid && s != model.UserGeneralBillingStatusCanceled {
			return errors.New("user_general_billing_status must be one of: unpaid, paid, canceled")
		}
	}
	// Validasi amount >= 0
	if p.UserGeneralBillingAmountIDR.Set && !p.UserGeneralBillingAmountIDR.Null && p.UserGeneralBillingAmountIDR.Value != nil {
		if *p.UserGeneralBillingAmountIDR.Value < 0 {
			return errors.New("user_general_billing_amount_idr must be >= 0")
		}
	}
	return nil
}

func (p PatchUserGeneralBillingRequest) Apply(m *model.UserGeneralBilling) (changed bool) {
	// SchoolStudentID (*uuid.UUID)
	if p.UserGeneralBillingSchoolStudentID.Set {
		if p.UserGeneralBillingSchoolStudentID.Null {
			m.UserGeneralBillingSchoolStudentID = nil
		} else if p.UserGeneralBillingSchoolStudentID.Value != nil {
			m.UserGeneralBillingSchoolStudentID = ptr(*p.UserGeneralBillingSchoolStudentID.Value)
		}
		changed = true
	}

	// PayerUserID (*uuid.UUID)
	if p.UserGeneralBillingPayerUserID.Set {
		if p.UserGeneralBillingPayerUserID.Null {
			m.UserGeneralBillingPayerUserID = nil
		} else if p.UserGeneralBillingPayerUserID.Value != nil {
			m.UserGeneralBillingPayerUserID = ptr(*p.UserGeneralBillingPayerUserID.Value)
		}
		changed = true
	}

	// Amount (int)
	if p.UserGeneralBillingAmountIDR.Set && p.UserGeneralBillingAmountIDR.Value != nil {
		m.UserGeneralBillingAmountIDR = *p.UserGeneralBillingAmountIDR.Value
		changed = true
	}

	// Status (NOT NULL string)
	if p.UserGeneralBillingStatus.Set {
		if p.UserGeneralBillingStatus.Null {
			m.UserGeneralBillingStatus = model.UserGeneralBillingStatusUnpaid
		} else if p.UserGeneralBillingStatus.Value != nil {
			m.UserGeneralBillingStatus = *p.UserGeneralBillingStatus.Value
		}
		changed = true
	}

	// PaidAt (*time.Time)
	if p.UserGeneralBillingPaidAt.Set {
		if p.UserGeneralBillingPaidAt.Null {
			m.UserGeneralBillingPaidAt = nil
		} else if p.UserGeneralBillingPaidAt.Value != nil {
			m.UserGeneralBillingPaidAt = ptr(*p.UserGeneralBillingPaidAt.Value)
		}
		changed = true
	}

	// Note (*string)
	if p.UserGeneralBillingNote.Set {
		if p.UserGeneralBillingNote.Null {
			m.UserGeneralBillingNote = nil
		} else if p.UserGeneralBillingNote.Value != nil {
			m.UserGeneralBillingNote = ptr(*p.UserGeneralBillingNote.Value)
		}
		changed = true
	}

	// Snapshots (*string)
	if p.UserGeneralBillingTitleSnapshot.Set {
		if p.UserGeneralBillingTitleSnapshot.Null {
			m.UserGeneralBillingTitleSnapshot = nil
		} else if p.UserGeneralBillingTitleSnapshot.Value != nil {
			m.UserGeneralBillingTitleSnapshot = ptr(*p.UserGeneralBillingTitleSnapshot.Value)
		}
		changed = true
	}
	if p.UserGeneralBillingKindCodeSnapshot.Set {
		if p.UserGeneralBillingKindCodeSnapshot.Null {
			m.UserGeneralBillingKindCodeSnapshot = nil
		} else if p.UserGeneralBillingKindCodeSnapshot.Value != nil {
			m.UserGeneralBillingKindCodeSnapshot = ptr(*p.UserGeneralBillingKindCodeSnapshot.Value)
		}
		changed = true
	}
	if p.UserGeneralBillingKindNameSnapshot.Set {
		if p.UserGeneralBillingKindNameSnapshot.Null {
			m.UserGeneralBillingKindNameSnapshot = nil
		} else if p.UserGeneralBillingKindNameSnapshot.Value != nil {
			m.UserGeneralBillingKindNameSnapshot = ptr(*p.UserGeneralBillingKindNameSnapshot.Value)
		}
		changed = true
	}

	// Meta (jsonb)
	if p.UserGeneralBillingMeta.Set {
		if p.UserGeneralBillingMeta.Null {
			m.UserGeneralBillingMeta = datatypes.JSONMap(nil)
		} else if p.UserGeneralBillingMeta.Value != nil {
			m.UserGeneralBillingMeta = datatypes.JSONMap(*p.UserGeneralBillingMeta.Value)
		}
		changed = true
	}

	return
}

/* =========================================================
   RESPONSE
   ========================================================= */

type UserGeneralBillingResponse struct {
	UserGeneralBillingID uuid.UUID `json:"user_general_billing_id"`

	UserGeneralBillingSchoolID        uuid.UUID  `json:"user_general_billing_school_id"`
	UserGeneralBillingSchoolStudentID *uuid.UUID `json:"user_general_billing_school_student_id"`
	UserGeneralBillingPayerUserID     *uuid.UUID `json:"user_general_billing_payer_user_id"`

	UserGeneralBillingBillingID uuid.UUID `json:"user_general_billing_billing_id"`

	UserGeneralBillingAmountIDR int        `json:"user_general_billing_amount_idr"`
	UserGeneralBillingStatus    string     `json:"user_general_billing_status"`
	UserGeneralBillingPaidAt    *time.Time `json:"user_general_billing_paid_at"`
	UserGeneralBillingNote      *string    `json:"user_general_billing_note"`

	UserGeneralBillingTitleSnapshot    *string `json:"user_general_billing_title_snapshot"`
	UserGeneralBillingKindCodeSnapshot *string `json:"user_general_billing_kind_code_snapshot"`
	UserGeneralBillingKindNameSnapshot *string `json:"user_general_billing_kind_name_snapshot"`

	UserGeneralBillingMeta map[string]any `json:"user_general_billing_meta"`

	UserGeneralBillingCreatedAt time.Time `json:"user_general_billing_created_at"`
	UserGeneralBillingUpdatedAt time.Time `json:"user_general_billing_updated_at"`
}

func FromModelUserGeneralBilling(m model.UserGeneralBilling) UserGeneralBillingResponse {
	var meta map[string]any
	if m.UserGeneralBillingMeta != nil {
		meta = map[string]any(m.UserGeneralBillingMeta)
	}
	return UserGeneralBillingResponse{
		UserGeneralBillingID:               m.UserGeneralBillingID,
		UserGeneralBillingSchoolID:         m.UserGeneralBillingSchoolID,
		UserGeneralBillingSchoolStudentID:  m.UserGeneralBillingSchoolStudentID,
		UserGeneralBillingPayerUserID:      m.UserGeneralBillingPayerUserID,
		UserGeneralBillingBillingID:        m.UserGeneralBillingBillingID,
		UserGeneralBillingAmountIDR:        m.UserGeneralBillingAmountIDR,
		UserGeneralBillingStatus:           m.UserGeneralBillingStatus,
		UserGeneralBillingPaidAt:           m.UserGeneralBillingPaidAt,
		UserGeneralBillingNote:             m.UserGeneralBillingNote,
		UserGeneralBillingTitleSnapshot:    m.UserGeneralBillingTitleSnapshot,
		UserGeneralBillingKindCodeSnapshot: m.UserGeneralBillingKindCodeSnapshot,
		UserGeneralBillingKindNameSnapshot: m.UserGeneralBillingKindNameSnapshot,
		UserGeneralBillingMeta:             meta,
		UserGeneralBillingCreatedAt:        m.UserGeneralBillingCreatedAt,
		UserGeneralBillingUpdatedAt:        m.UserGeneralBillingUpdatedAt,
	}
}

/* =========================================================
   (Opsional) QUERY untuk list & paging sederhana
   ========================================================= */

type ListUserGeneralBillingQuery struct {
	// Filter
	SchoolID        *uuid.UUID `query:"school_id"`
	BillingID       *uuid.UUID `query:"billing_id"`
	SchoolStudentID *uuid.UUID `query:"school_student_id"`
	PayerUserID     *uuid.UUID `query:"payer_user_id"`
	Status          *string    `query:"status"` // unpaid|paid|canceled

	// Pagination
	Page     int `query:"page" validate:"omitempty,min=1"`              // default 1
	PageSize int `query:"page_size" validate:"omitempty,min=1,max=200"` // default 20/50
}
