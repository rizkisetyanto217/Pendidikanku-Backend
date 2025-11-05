// file: internals/features/finance/spp/model/student_bill.go
package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ==============================
   ENUM — status student bill
============================== */

type StudentBillStatus string

const (
	StudentBillStatusUnpaid   StudentBillStatus = "unpaid"
	StudentBillStatusPaid     StudentBillStatus = "paid"
	StudentBillStatusCanceled StudentBillStatus = "canceled"
)

/* ==============================================
   MODEL — selaras dengan SQL DDL terbaru
============================================== */

type StudentBill struct {
	// PK
	StudentBillID uuid.UUID `gorm:"column:student_bill_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"student_bill_id"`

	// FK → bill_batches(bill_batch_id)
	StudentBillBatchID uuid.UUID `gorm:"column:student_bill_batch_id;type:uuid;not null;index;uniqueIndex:uniq_batch_student,priority:1" json:"student_bill_batch_id"`

	// Tenant & subject (composite FK ke school_students)
	StudentBillSchoolID        uuid.UUID  `gorm:"column:student_bill_school_id;type:uuid;not null;index" json:"student_bill_school_id"`
	StudentBillSchoolStudentID *uuid.UUID `gorm:"column:student_bill_school_student_id;type:uuid;index;uniqueIndex:uniq_batch_student,priority:2" json:"student_bill_school_student_id"`

	// Payer (opsional)
	StudentBillPayerUserID *uuid.UUID `gorm:"column:student_bill_payer_user_id;type:uuid;index" json:"student_bill_payer_user_id,omitempty"`

	// Jenis + periode
	StudentBillGeneralBillingKindID *uuid.UUID `gorm:"column:student_bill_general_billing_kind_id;type:uuid;index" json:"student_bill_general_billing_kind_id,omitempty"`
	StudentBillBillCode             string     `gorm:"column:student_bill_bill_code;type:varchar(60);not null;default:SPP;index" json:"student_bill_bill_code"`
	StudentBillYear                 *int16     `gorm:"column:student_bill_year;type:smallint" json:"student_bill_year,omitempty"`
	StudentBillMonth                *int16     `gorm:"column:student_bill_month;type:smallint" json:"student_bill_month,omitempty"`
	StudentBillTermID               *uuid.UUID `gorm:"column:student_bill_term_id;type:uuid;index" json:"student_bill_term_id,omitempty"`

	// Option (one-off)
	StudentBillOptionCode  *string `gorm:"column:student_bill_option_code;type:varchar(60);index" json:"student_bill_option_code,omitempty"`
	StudentBillOptionLabel *string `gorm:"column:student_bill_option_label;type:varchar(60)" json:"student_bill_option_label,omitempty"`

	// Amount
	StudentBillAmountIDR int `gorm:"column:student_bill_amount_idr;type:int;not null;check:student_bill_amount_idr>=0;index" json:"student_bill_amount_idr"`

	// Status
	StudentBillStatus StudentBillStatus `gorm:"column:student_bill_status;type:varchar(20);not null;default:'unpaid';index" json:"student_bill_status"`
	StudentBillPaidAt *time.Time        `gorm:"column:student_bill_paid_at" json:"student_bill_paid_at,omitempty"`
	StudentBillNote   *string           `gorm:"column:student_bill_note;type:text" json:"student_bill_note,omitempty"`

	// =========================
	// Relasi kelas & section + snapshot label/slug
	// =========================
	StudentBillClassID   *uuid.UUID `gorm:"column:student_bill_class_id;type:uuid;index" json:"student_bill_class_id,omitempty"`
	StudentBillSectionID *uuid.UUID `gorm:"column:student_bill_section_id;type:uuid;index" json:"student_bill_section_id,omitempty"`

	StudentBillClassNameSnapshot   *string `gorm:"column:student_bill_class_name_snapshot;type:text" json:"student_bill_class_name_snapshot,omitempty"`
	StudentBillClassSlugSnapshot   *string `gorm:"column:student_bill_class_slug_snapshot;type:varchar(80)" json:"student_bill_class_slug_snapshot,omitempty"`
	StudentBillSectionNameSnapshot *string `gorm:"column:student_bill_section_name_snapshot;type:text" json:"student_bill_section_name_snapshot,omitempty"`
	StudentBillSectionSlugSnapshot *string `gorm:"column:student_bill_section_slug_snapshot;type:varchar(80)" json:"student_bill_section_slug_snapshot,omitempty"`

	// Audit
	StudentBillCreatedAt time.Time      `gorm:"column:student_bill_created_at;type:timestamptz;not null;default:now();index" json:"student_bill_created_at"`
	StudentBillUpdatedAt time.Time      `gorm:"column:student_bill_updated_at;type:timestamptz;not null;default:now()" json:"student_bill_updated_at"`
	StudentBillDeletedAt gorm.DeletedAt `gorm:"column:student_bill_deleted_at;type:timestamptz;index" json:"-"`

	// Hint: unique (batch_id, school_student_id) sudah di-tag via uniqueIndex di atas
}

func (StudentBill) TableName() string { return "student_bills" }

/* ======================================
   HOOKS — default bill_code & timestamps
====================================== */

func (m *StudentBill) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now()

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
	if strings.TrimSpace(m.StudentBillBillCode) == "" {
		m.StudentBillBillCode = "SPP"
	}
	m.StudentBillUpdatedAt = time.Now()
	return nil
}
