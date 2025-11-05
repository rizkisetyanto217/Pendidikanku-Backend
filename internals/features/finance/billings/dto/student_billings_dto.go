// file: internals/features/finance/spp/dto/student_bill_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	// ganti path sesuai modelmu
	billing "schoolku_backend/internals/features/finance/billings/model"
)

////////////////////////////////////////////////////////////////////////////////
// STUDENT BILLS — DTO
////////////////////////////////////////////////////////////////////////////////

// Create (manual create 1 row student_bills)
type StudentBillCreateDTO struct {
	StudentBillBatchID         uuid.UUID  `json:"student_bill_batch_id" validate:"required"`
	StudentBillSchoolID        uuid.UUID  `json:"student_bill_school_id" validate:"required"`
	StudentBillSchoolStudentID *uuid.UUID `json:"student_bill_school_student_id,omitempty"`
	StudentBillPayerUserID     *uuid.UUID `json:"student_bill_payer_user_id,omitempty"`

	// Denorm jenis + periode (opsional saat create manual)
	StudentBillGeneralBillingKindID *uuid.UUID `json:"student_bill_general_billing_kind_id,omitempty"`
	StudentBillBillCode             string     `json:"student_bill_bill_code,omitempty"` // default "SPP" by model hook
	StudentBillYear                 *int16     `json:"student_bill_year,omitempty"`
	StudentBillMonth                *int16     `json:"student_bill_month,omitempty"`
	StudentBillTermID               *uuid.UUID `json:"student_bill_term_id,omitempty"`

	// Option one-off (boleh NULL untuk periodic/SPP)
	StudentBillOptionCode  *string `json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string `json:"student_bill_option_label,omitempty"`

	// Relasi kelas/section (opsional)
	StudentBillClassID   *uuid.UUID `json:"student_bill_class_id,omitempty"`
	StudentBillSectionID *uuid.UUID `json:"student_bill_section_id,omitempty"`

	// Snapshot label/slug (opsional; bisa diisi service saat generate)
	StudentBillClassNameSnapshot   *string `json:"student_bill_class_name_snapshot,omitempty"`
	StudentBillClassSlugSnapshot   *string `json:"student_bill_class_slug_snapshot,omitempty"`
	StudentBillSectionNameSnapshot *string `json:"student_bill_section_name_snapshot,omitempty"`
	StudentBillSectionSlugSnapshot *string `json:"student_bill_section_slug_snapshot,omitempty"`

	StudentBillAmountIDR int     `json:"student_bill_amount_idr" validate:"required,min=0"`
	StudentBillNote      *string `json:"student_bill_note,omitempty"`
}

// Update (partial) — tidak untuk ubah status; status pakai DTO khusus di bawah
type StudentBillUpdateDTO struct {
	StudentBillPayerUserID *uuid.UUID `json:"student_bill_payer_user_id,omitempty"`

	// Denorm jenis + periode
	StudentBillGeneralBillingKindID *uuid.UUID `json:"student_bill_general_billing_kind_id,omitempty"`
	StudentBillBillCode             *string    `json:"student_bill_bill_code,omitempty"`
	StudentBillYear                 *int16     `json:"student_bill_year,omitempty"`
	StudentBillMonth                *int16     `json:"student_bill_month,omitempty"`
	StudentBillTermID               *uuid.UUID `json:"student_bill_term_id,omitempty"`

	// Option one-off
	StudentBillOptionCode  *string `json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string `json:"student_bill_option_label,omitempty"`

	// Relasi kelas/section
	StudentBillClassID   *uuid.UUID `json:"student_bill_class_id,omitempty"`
	StudentBillSectionID *uuid.UUID `json:"student_bill_section_id,omitempty"`

	// Snapshot label/slug
	StudentBillClassNameSnapshot   *string `json:"student_bill_class_name_snapshot,omitempty"`
	StudentBillClassSlugSnapshot   *string `json:"student_bill_class_slug_snapshot,omitempty"`
	StudentBillSectionNameSnapshot *string `json:"student_bill_section_name_snapshot,omitempty"`
	StudentBillSectionSlugSnapshot *string `json:"student_bill_section_slug_snapshot,omitempty"`

	// Amount & note
	StudentBillAmountIDR *int    `json:"student_bill_amount_idr,omitempty"`
	StudentBillNote      *string `json:"student_bill_note,omitempty"`
}

// Response
type StudentBillResponse struct {
	StudentBillID              uuid.UUID  `json:"student_bill_id"`
	StudentBillBatchID         uuid.UUID  `json:"student_bill_batch_id"`
	StudentBillSchoolID        uuid.UUID  `json:"student_bill_school_id"`
	StudentBillSchoolStudentID *uuid.UUID `json:"student_bill_school_student_id,omitempty"`
	StudentBillPayerUserID     *uuid.UUID `json:"student_bill_payer_user_id,omitempty"`

	StudentBillGeneralBillingKindID *uuid.UUID `json:"student_bill_general_billing_kind_id,omitempty"`
	StudentBillBillCode             string     `json:"student_bill_bill_code"`
	StudentBillYear                 *int16     `json:"student_bill_year,omitempty"`
	StudentBillMonth                *int16     `json:"student_bill_month,omitempty"`
	StudentBillTermID               *uuid.UUID `json:"student_bill_term_id,omitempty"`

	StudentBillOptionCode  *string `json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string `json:"student_bill_option_label,omitempty"`

	// Relasi kelas/section + snapshot
	StudentBillClassID             *uuid.UUID `json:"student_bill_class_id,omitempty"`
	StudentBillSectionID           *uuid.UUID `json:"student_bill_section_id,omitempty"`
	StudentBillClassNameSnapshot   *string    `json:"student_bill_class_name_snapshot,omitempty"`
	StudentBillClassSlugSnapshot   *string    `json:"student_bill_class_slug_snapshot,omitempty"`
	StudentBillSectionNameSnapshot *string    `json:"student_bill_section_name_snapshot,omitempty"`
	StudentBillSectionSlugSnapshot *string    `json:"student_bill_section_slug_snapshot,omitempty"`

	StudentBillAmountIDR int    `json:"student_bill_amount_idr"`
	StudentBillStatus    string `json:"student_bill_status"` // unpaid|paid|canceled

	StudentBillPaidAt *time.Time `json:"student_bill_paid_at,omitempty"`
	StudentBillNote   *string    `json:"student_bill_note,omitempty"`

	StudentBillCreatedAt time.Time  `json:"student_bill_created_at"`
	StudentBillUpdatedAt time.Time  `json:"student_bill_updated_at"`
	StudentBillDeletedAt *time.Time `json:"student_bill_deleted_at,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////
// STUDENT BILL STATUS — DTO
////////////////////////////////////////////////////////////////////////////////

type StudentBillMarkPaidDTO struct {
	PaidAt *time.Time `json:"paid_at,omitempty"` // jika nil, backend isi now()
	Note   *string    `json:"note,omitempty"`
}

type StudentBillMarkUnpaidDTO struct {
	Note *string `json:"note,omitempty"`
}

type StudentBillCancelDTO struct {
	Note *string `json:"note,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////
// MAPPERS — Model <-> DTO
////////////////////////////////////////////////////////////////////////////////

func ToStudentBillResponse(m billing.StudentBill) StudentBillResponse {
	return StudentBillResponse{
		StudentBillID:                   m.StudentBillID,
		StudentBillBatchID:              m.StudentBillBatchID,
		StudentBillSchoolID:             m.StudentBillSchoolID,
		StudentBillSchoolStudentID:      m.StudentBillSchoolStudentID,
		StudentBillPayerUserID:          m.StudentBillPayerUserID,
		StudentBillGeneralBillingKindID: m.StudentBillGeneralBillingKindID,
		StudentBillBillCode:             m.StudentBillBillCode,
		StudentBillYear:                 m.StudentBillYear,
		StudentBillMonth:                m.StudentBillMonth,
		StudentBillTermID:               m.StudentBillTermID,
		StudentBillOptionCode:           m.StudentBillOptionCode,
		StudentBillOptionLabel:          m.StudentBillOptionLabel,

		// relasi kelas/section + snapshot
		StudentBillClassID:             m.StudentBillClassID,
		StudentBillSectionID:           m.StudentBillSectionID,
		StudentBillClassNameSnapshot:   m.StudentBillClassNameSnapshot,
		StudentBillClassSlugSnapshot:   m.StudentBillClassSlugSnapshot,
		StudentBillSectionNameSnapshot: m.StudentBillSectionNameSnapshot,
		StudentBillSectionSlugSnapshot: m.StudentBillSectionSlugSnapshot,

		StudentBillAmountIDR: m.StudentBillAmountIDR,
		StudentBillStatus:    string(m.StudentBillStatus),
		StudentBillPaidAt:    m.StudentBillPaidAt,
		StudentBillNote:      m.StudentBillNote,

		StudentBillCreatedAt: m.StudentBillCreatedAt,
		StudentBillUpdatedAt: m.StudentBillUpdatedAt,
		StudentBillDeletedAt: toPtrTimeFromDeletedAt(m.StudentBillDeletedAt),
	}
}

func StudentBillCreateDTOToModel(d StudentBillCreateDTO) billing.StudentBill {
	return billing.StudentBill{
		StudentBillBatchID:              d.StudentBillBatchID,
		StudentBillSchoolID:             d.StudentBillSchoolID,
		StudentBillSchoolStudentID:      d.StudentBillSchoolStudentID,
		StudentBillPayerUserID:          d.StudentBillPayerUserID,
		StudentBillGeneralBillingKindID: d.StudentBillGeneralBillingKindID,
		StudentBillBillCode:             d.StudentBillBillCode, // default "SPP" by model hook jika kosong
		StudentBillYear:                 d.StudentBillYear,
		StudentBillMonth:                d.StudentBillMonth,
		StudentBillTermID:               d.StudentBillTermID,
		StudentBillOptionCode:           d.StudentBillOptionCode,
		StudentBillOptionLabel:          d.StudentBillOptionLabel,

		// relasi kelas/section
		StudentBillClassID:   d.StudentBillClassID,
		StudentBillSectionID: d.StudentBillSectionID,

		// snapshot label/slug
		StudentBillClassNameSnapshot:   d.StudentBillClassNameSnapshot,
		StudentBillClassSlugSnapshot:   d.StudentBillClassSlugSnapshot,
		StudentBillSectionNameSnapshot: d.StudentBillSectionNameSnapshot,
		StudentBillSectionSlugSnapshot: d.StudentBillSectionSlugSnapshot,

		StudentBillAmountIDR: d.StudentBillAmountIDR,
		StudentBillStatus:    billing.StudentBillStatusUnpaid,
		StudentBillNote:      d.StudentBillNote,
	}
}

// UpdateDTO -> Model (apply partial, tidak menyentuh status/paid_at)
func ApplyStudentBillUpdate(m *billing.StudentBill, d StudentBillUpdateDTO) {
	if d.StudentBillPayerUserID != nil {
		m.StudentBillPayerUserID = d.StudentBillPayerUserID
	}
	if d.StudentBillGeneralBillingKindID != nil {
		m.StudentBillGeneralBillingKindID = d.StudentBillGeneralBillingKindID
	}
	if d.StudentBillBillCode != nil {
		m.StudentBillBillCode = *d.StudentBillBillCode // model hook jaga default bila kosong
	}
	if d.StudentBillYear != nil {
		m.StudentBillYear = d.StudentBillYear
	}
	if d.StudentBillMonth != nil {
		m.StudentBillMonth = d.StudentBillMonth
	}
	if d.StudentBillTermID != nil {
		m.StudentBillTermID = d.StudentBillTermID
	}
	if d.StudentBillOptionCode != nil {
		m.StudentBillOptionCode = d.StudentBillOptionCode
	}
	if d.StudentBillOptionLabel != nil {
		m.StudentBillOptionLabel = d.StudentBillOptionLabel
	}

	// relasi kelas/section
	if d.StudentBillClassID != nil {
		m.StudentBillClassID = d.StudentBillClassID
	}
	if d.StudentBillSectionID != nil {
		m.StudentBillSectionID = d.StudentBillSectionID
	}

	// snapshot label/slug
	if d.StudentBillClassNameSnapshot != nil {
		m.StudentBillClassNameSnapshot = d.StudentBillClassNameSnapshot
	}
	if d.StudentBillClassSlugSnapshot != nil {
		m.StudentBillClassSlugSnapshot = d.StudentBillClassSlugSnapshot
	}
	if d.StudentBillSectionNameSnapshot != nil {
		m.StudentBillSectionNameSnapshot = d.StudentBillSectionNameSnapshot
	}
	if d.StudentBillSectionSlugSnapshot != nil {
		m.StudentBillSectionSlugSnapshot = d.StudentBillSectionSlugSnapshot
	}

	if d.StudentBillAmountIDR != nil {
		m.StudentBillAmountIDR = *d.StudentBillAmountIDR
	}
	if d.StudentBillNote != nil {
		m.StudentBillNote = d.StudentBillNote
	}
}

////////////////////////////////////////////////////////////////////////////////
// SMALL UTILS
////////////////////////////////////////////////////////////////////////////////

// Helpers list mapping
func ToStudentBillResponses(list []billing.StudentBill) []StudentBillResponse {
	out := make([]StudentBillResponse, 0, len(list))
	for _, v := range list {
		out = append(out, ToStudentBillResponse(v))
	}
	return out
}
