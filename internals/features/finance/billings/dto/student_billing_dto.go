// file: internals/features/finance/spp/dto/student_bill_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	// ganti path sesuai modelmu
	billing "masjidku_backend/internals/features/finance/billings/model"
)

////////////////////////////////////////////////////////////////////////////////
// STUDENT BILLS — DTO
////////////////////////////////////////////////////////////////////////////////

// Create (manual create 1 row student_bills)
type StudentBillCreateDTO struct {
	StudentBillBatchID         uuid.UUID  `json:"student_bill_batch_id" validate:"required"`
	StudentBillMasjidID        uuid.UUID  `json:"student_bill_masjid_id" validate:"required"`
	StudentBillMasjidStudentID *uuid.UUID `json:"student_bill_masjid_student_id,omitempty"`
	StudentBillPayerUserID     *uuid.UUID `json:"student_bill_payer_user_id,omitempty"`
	StudentBillOptionCode      *string    `json:"student_bill_option_code,omitempty"`  // contoh: "SPP"
	StudentBillOptionLabel     *string    `json:"student_bill_option_label,omitempty"` // contoh: "SPP Oktober 2025"
	StudentBillAmountIDR       int        `json:"student_bill_amount_idr" validate:"required,min=0"`
	StudentBillNote            *string    `json:"student_bill_note,omitempty"`
}

// Update (partial) — tidak untuk ubah status; status pakai DTO khusus di bawah
type StudentBillUpdateDTO struct {
	StudentBillPayerUserID *uuid.UUID `json:"student_bill_payer_user_id,omitempty"`
	StudentBillOptionCode  *string    `json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string    `json:"student_bill_option_label,omitempty"`
	StudentBillAmountIDR   *int       `json:"student_bill_amount_idr,omitempty"`
	StudentBillNote        *string    `json:"student_bill_note,omitempty"`
}

// Response
type StudentBillResponse struct {
	StudentBillID              uuid.UUID  `json:"student_bill_id"`
	StudentBillBatchID         uuid.UUID  `json:"student_bill_batch_id"`
	StudentBillMasjidID        uuid.UUID  `json:"student_bill_masjid_id"`
	StudentBillMasjidStudentID *uuid.UUID `json:"student_bill_masjid_student_id,omitempty"`
	StudentBillPayerUserID     *uuid.UUID `json:"student_bill_payer_user_id,omitempty"`

	StudentBillOptionCode  *string `json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string `json:"student_bill_option_label,omitempty"`

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

// Model -> Response
func ToStudentBillResponse(m billing.StudentBill) StudentBillResponse {
	return StudentBillResponse{
		StudentBillID:              m.StudentBillID,
		StudentBillBatchID:         m.StudentBillBatchID,
		StudentBillMasjidID:        m.StudentBillMasjidID,
		StudentBillMasjidStudentID: m.StudentBillMasjidStudentID,
		StudentBillPayerUserID:     m.StudentBillPayerUserID,
		StudentBillOptionCode:      m.StudentBillOptionCode,
		StudentBillOptionLabel:     m.StudentBillOptionLabel,
		StudentBillAmountIDR:       m.StudentBillAmountIDR,
		StudentBillStatus:          string(m.StudentBillStatus),
		StudentBillPaidAt:          m.StudentBillPaidAt,
		StudentBillNote:            m.StudentBillNote,
		StudentBillCreatedAt:       m.StudentBillCreatedAt,
		StudentBillUpdatedAt:       m.StudentBillUpdatedAt,
		StudentBillDeletedAt:       toPtrTimeFromDeletedAt(m.StudentBillDeletedAt),
	}
}

// CreateDTO -> Model
func StudentBillCreateDTOToModel(d StudentBillCreateDTO) billing.StudentBill {
	return billing.StudentBill{
		StudentBillBatchID:         d.StudentBillBatchID,
		StudentBillMasjidID:        d.StudentBillMasjidID,
		StudentBillMasjidStudentID: d.StudentBillMasjidStudentID,
		StudentBillPayerUserID:     d.StudentBillPayerUserID,
		StudentBillOptionCode:      d.StudentBillOptionCode,
		StudentBillOptionLabel:     d.StudentBillOptionLabel,
		StudentBillAmountIDR:       d.StudentBillAmountIDR,
		StudentBillStatus:          billing.StudentBillStatusUnpaid, // default unpaid
		StudentBillNote:            d.StudentBillNote,
	}
}

// UpdateDTO -> Model (apply partial, tidak menyentuh status/paid_at)
func ApplyStudentBillUpdate(m *billing.StudentBill, d StudentBillUpdateDTO) {
	if d.StudentBillPayerUserID != nil {
		m.StudentBillPayerUserID = d.StudentBillPayerUserID
	}
	if d.StudentBillOptionCode != nil {
		m.StudentBillOptionCode = d.StudentBillOptionCode
	}
	if d.StudentBillOptionLabel != nil {
		m.StudentBillOptionLabel = d.StudentBillOptionLabel
	}
	if d.StudentBillAmountIDR != nil {
		m.StudentBillAmountIDR = *d.StudentBillAmountIDR
	}
	if d.StudentBillNote != nil {
		m.StudentBillNote = d.StudentBillNote
	}
}

// Helpers list mapping
func ToStudentBillResponses(list []billing.StudentBill) []StudentBillResponse {
	out := make([]StudentBillResponse, 0, len(list))
	for _, v := range list {
		out = append(out, ToStudentBillResponse(v))
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////
// SMALL UTILS
////////////////////////////////////////////////////////////////////////////////

func toPtrTimeFromDeletedAt(d gorm.DeletedAt) *time.Time {
	if d.Valid {
		return &d.Time
	}
	return nil
}
