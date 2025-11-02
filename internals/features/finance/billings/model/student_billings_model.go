// file: internals/features/finance/spp/model/student_bill.go
package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =========================================================
// ENUM — status student bill
// =========================================================

type StudentBillStatus string

const (
	StudentBillStatusUnpaid   StudentBillStatus = "unpaid"
	StudentBillStatusPaid     StudentBillStatus = "paid"
	StudentBillStatusCanceled StudentBillStatus = "canceled"
)

// =========================================================
// MODEL — selaras dengan SQL DDL terbaru
// =========================================================

type StudentBill struct {
	// PK
	StudentBillID uuid.UUID `gorm:"column:student_bill_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"student_bill_id"`

	// FK → bill_batches(bill_batch_id)
	StudentBillBatchID uuid.UUID `gorm:"column:student_bill_batch_id;type:uuid;not null;index" json:"student_bill_batch_id"`

	// FK (composite) → school_students (school_student_id, school_student_school_id)
	StudentBillSchoolID        uuid.UUID  `gorm:"column:student_bill_school_id;type:uuid;not null;index:ix_student_bill_school" json:"student_bill_school_id"`
	StudentBillSchoolStudentID *uuid.UUID `gorm:"column:student_bill_school_student_id;type:uuid;index;index:uniq_batch_student,unique,priority:2" json:"student_bill_school_student_id"`

	// FK → users(id) (optional)
	StudentBillPayerUserID *uuid.UUID `gorm:"column:student_bill_payer_user_id;type:uuid;index" json:"student_bill_payer_user_id"`

	// ========== Denorm jenis + periode (ikut batch) ==========
	StudentBillGeneralBillingKindID *uuid.UUID `gorm:"column:student_bill_general_billing_kind_id;type:uuid;index" json:"student_bill_general_billing_kind_id,omitempty"`

	// Not null + default 'SPP'
	StudentBillBillCode string `gorm:"column:student_bill_bill_code;type:varchar(60);not null;default:SPP;index" json:"student_bill_bill_code"`

	// YM nullable untuk one-off
	StudentBillYear   *int16     `gorm:"column:student_bill_year;type:smallint" json:"student_bill_year,omitempty"`
	StudentBillMonth  *int16     `gorm:"column:student_bill_month;type:smallint" json:"student_bill_month,omitempty"`
	StudentBillTermID *uuid.UUID `gorm:"column:student_bill_term_id;type:uuid;index" json:"student_bill_term_id,omitempty"`

	// Option untuk one-off (boleh NULL untuk SPP/periodic)
	StudentBillOptionCode  *string `gorm:"column:student_bill_option_code;type:varchar(60);index" json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string `gorm:"column:student_bill_option_label;type:varchar(60)" json:"student_bill_option_label,omitempty"`

	// Amount
	StudentBillAmountIDR int `gorm:"column:student_bill_amount_idr;type:int;not null;check:student_bill_amount_idr>=0;index:ix_student_bill_amount" json:"student_bill_amount_idr"`

	// Status & payment
	StudentBillStatus StudentBillStatus `gorm:"column:student_bill_status;type:varchar(20);not null;default:'unpaid';index:ix_student_bill_status" json:"student_bill_status"`
	StudentBillPaidAt *time.Time        `gorm:"column:student_bill_paid_at" json:"student_bill_paid_at,omitempty"`
	StudentBillNote   *string           `gorm:"column:student_bill_note;type:text" json:"student_bill_note,omitempty"`

	// Timestamps (eksplisit)
	StudentBillCreatedAt time.Time      `gorm:"column:student_bill_created_at;type:timestamptz;not null;default:now();index:ix_student_bill_created_at" json:"student_bill_created_at"`
	StudentBillUpdatedAt time.Time      `gorm:"column:student_bill_updated_at;type:timestamptz;not null;default:now()" json:"student_bill_updated_at"`
	StudentBillDeletedAt gorm.DeletedAt `gorm:"column:student_bill_deleted_at;type:timestamptz;index" json:"-"`

	// Unique constraint (batch_id + school_student_id)
	// NOTE: GORM composite unique via tag di field kedua:
	_ struct{} `gorm:"uniqueIndex:uniq_batch_student,priority:1"` // menempel pada kolom StudentBillBatchID
}

func (StudentBill) TableName() string { return "student_bills" }

// =========================================================
// HOOKS — set timestamps & default bill_code
// =========================================================

func (m *StudentBill) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now()

	// default bill code (selaras DDL DEFAULT 'SPP')
	if strings.TrimSpace(m.StudentBillBillCode) == "" {
		m.StudentBillBillCode = "SPP"
	}

	if m.StudentBillCreatedAt.IsZero() {
		m.StudentBillCreatedAt = now
	}
	m.StudentBillUpdatedAt = now
	return nil
}

func (m *StudentBill) BeforeUpdate(tx *gorm.DB) (err error) {
	// keep default if someone tries to blank it out
	if strings.TrimSpace(m.StudentBillBillCode) == "" {
		m.StudentBillBillCode = "SPP"
	}
	m.StudentBillUpdatedAt = time.Now()
	return nil
}
