// file: internals/features/finance/spp/model/student_bill.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =========================================================
// ENUM (opsional) — status student bill
// =========================================================

type StudentBillStatus string

const (
	StudentBillStatusUnpaid   StudentBillStatus = "unpaid"
	StudentBillStatusPaid     StudentBillStatus = "paid"
	StudentBillStatusCanceled StudentBillStatus = "canceled"
)

// =========================================================
// MODEL
// =========================================================

type StudentBill struct {
	// PK
	StudentBillID uuid.UUID `gorm:"column:student_bill_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"student_bill_id"`

	// FK → bill_batches(bill_batch_id)
	StudentBillBatchID uuid.UUID `gorm:"column:student_bill_batch_id;type:uuid;not null;index" json:"student_bill_batch_id"`

	// FK (composite) → masjid_students (masjid_student_id, masjid_student_masjid_id)
	StudentBillMasjidID        uuid.UUID  `gorm:"column:student_bill_masjid_id;type:uuid;not null;index:ix_student_bill_masjid" json:"student_bill_masjid_id"`
	StudentBillMasjidStudentID *uuid.UUID `gorm:"column:student_bill_masjid_student_id;type:uuid;index;index:uniq_batch_student,unique,priority:2" json:"student_bill_masjid_student_id"`

	// FK → users(id) (optional)
	StudentBillPayerUserID *uuid.UUID `gorm:"column:student_bill_payer_user_id;type:uuid;index" json:"student_bill_payer_user_id"`

	// Label/option
	StudentBillOptionCode  *string `gorm:"column:student_bill_option_code;type:varchar(20)" json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string `gorm:"column:student_bill_option_label;type:varchar(60)" json:"student_bill_option_label,omitempty"`

	// Amount
	StudentBillAmountIDR int `gorm:"column:student_bill_amount_idr;not null;check:student_bill_amount_idr>=0;index:ix_student_bill_amount" json:"student_bill_amount_idr"`

	// Status & payment
	StudentBillStatus StudentBillStatus `gorm:"column:student_bill_status;type:varchar(20);not null;default:'unpaid';index:ix_student_bill_status" json:"student_bill_status"`
	StudentBillPaidAt *time.Time        `gorm:"column:student_bill_paid_at" json:"student_bill_paid_at,omitempty"`
	StudentBillNote   *string           `gorm:"column:student_bill_note" json:"student_bill_note,omitempty"`

	// Timestamps (eksplisit)
	StudentBillCreatedAt time.Time      `gorm:"column:student_bill_created_at;not null;default:now();index:ix_student_bill_created_at" json:"student_bill_created_at"`
	StudentBillUpdatedAt time.Time      `gorm:"column:student_bill_updated_at;not null;default:now()" json:"student_bill_updated_at"`
	StudentBillDeletedAt gorm.DeletedAt `gorm:"column:student_bill_deleted_at;index" json:"-"`

	// Unique constraint (batch_id + masjid_student_id)
	// NOTE: GORM composite unique via tag di field kedua:
	_ struct{} `gorm:"uniqueIndex:uniq_batch_student,priority:1"` // priority:1 akan menempel pada kolom StudentBillBatchID
}

// TableName overrides the table name used by StudentBill to `student_bills`
func (StudentBill) TableName() string {
	return "student_bills"
}

// =========================================================
// HOOKS — set timestamps eksplisit
// =========================================================

func (m *StudentBill) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now()
	if m.StudentBillCreatedAt.IsZero() {
		m.StudentBillCreatedAt = now
	}
	m.StudentBillUpdatedAt = now
	return nil
}

func (m *StudentBill) BeforeUpdate(tx *gorm.DB) (err error) {
	m.StudentBillUpdatedAt = time.Now()
	return nil
}
