package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type PaymentItemModel struct {
	PaymentItemID uuid.UUID `gorm:"column:payment_item_id;type:uuid;default:gen_random_uuid();primaryKey" json:"payment_item_id"`

	// Tenant & relasi ke header
	PaymentItemSchoolID  uuid.UUID `gorm:"column:payment_item_school_id;type:uuid;not null" json:"payment_item_school_id"`
	PaymentItemPaymentID uuid.UUID `gorm:"column:payment_item_payment_id;type:uuid;not null" json:"payment_item_payment_id"`
	PaymentItemIndex     int16     `gorm:"column:payment_item_index;not null" json:"payment_item_index"`

	// === Target per item (SESUI DDL BARU) ===
	PaymentItemUserGeneralBillingID *uuid.UUID `gorm:"column:payment_item_user_general_billing_id;type:uuid" json:"payment_item_user_general_billing_id"`
	PaymentItemGeneralBillingID     *uuid.UUID `gorm:"column:payment_item_general_billing_id;type:uuid" json:"payment_item_general_billing_id"`
	PaymentItemBillBatchID          *uuid.UUID `gorm:"column:payment_item_bill_batch_id;type:uuid" json:"payment_item_bill_batch_id"`

	// ❌ Kolom lama yang tidak ada di DB → hapus / jangan dipetakan:
	// PaymentItemStudentBillID        *uuid.UUID `gorm:"-" json:"payment_item_student_bill_id"`
	// PaymentItemGeneralBillingKindID *uuid.UUID `gorm:"-" json:"payment_item_general_billing_kind_id"`

	// Subjek murid per item
	PaymentItemSchoolStudentID *uuid.UUID `gorm:"column:payment_item_school_student_id;type:uuid" json:"payment_item_school_student_id"`

	// Context kelas/enrollment (opsional)
	PaymentItemClassID      *uuid.UUID `gorm:"column:payment_item_class_id;type:uuid" json:"payment_item_class_id"`
	PaymentItemEnrollmentID *uuid.UUID `gorm:"column:payment_item_enrollment_id;type:uuid" json:"payment_item_enrollment_id"`

	// Nominal per item
	PaymentItemAmountIDR int `gorm:"column:payment_item_amount_idr;not null" json:"payment_item_amount_idr"`

	// === Fee rule snapshots per item ===
	PaymentItemFeeRuleID                  *uuid.UUID `gorm:"column:payment_item_fee_rule_id;type:uuid" json:"payment_item_fee_rule_id"`
	PaymentItemFeeRuleOptionCodeSnapshot  *string    `gorm:"column:payment_item_fee_rule_option_code_snapshot;type:varchar(20)" json:"payment_item_fee_rule_option_code_snapshot"`
	PaymentItemFeeRuleOptionIndexSnapshot *int16     `gorm:"column:payment_item_fee_rule_option_index_snapshot" json:"payment_item_fee_rule_option_index_snapshot"`
	PaymentItemFeeRuleAmountSnapshot      *int       `gorm:"column:payment_item_fee_rule_amount_snapshot" json:"payment_item_fee_rule_amount_snapshot"`
	PaymentItemFeeRuleScopeSnapshot       *FeeScope  `gorm:"column:payment_item_fee_rule_scope_snapshot;type:fee_scope" json:"payment_item_fee_rule_scope_snapshot"`
	PaymentItemFeeRuleNoteSnapshot        *string    `gorm:"column:payment_item_fee_rule_note_snapshot" json:"payment_item_fee_rule_note_snapshot"`

	// === Academic term snapshots per item ===
	PaymentItemAcademicTermID           *uuid.UUID `gorm:"column:payment_item_academic_term_id;type:uuid" json:"payment_item_academic_term_id"`
	PaymentItemAcademicTermAcademicYear *string    `gorm:"column:payment_item_academic_term_academic_year_cache;type:varchar(40)" json:"payment_item_academic_term_academic_year_cache"`
	PaymentItemAcademicTermName         *string    `gorm:"column:payment_item_academic_term_name_cache;type:varchar(100)" json:"payment_item_academic_term_name_cache"`
	PaymentItemAcademicTermSlug         *string    `gorm:"column:payment_item_academic_term_slug_cache;type:varchar(160)" json:"payment_item_academic_term_slug_cache"`
	PaymentItemAcademicTermAngkatan     *string    `gorm:"column:payment_item_academic_term_angkatan_cache;type:varchar(40)" json:"payment_item_academic_term_angkatan_cache"`

	// 🧾 Invoice per item
	PaymentItemInvoiceNumber *string    `gorm:"column:payment_item_invoice_number" json:"payment_item_invoice_number"`
	PaymentItemInvoiceTitle  *string    `gorm:"column:payment_item_invoice_title" json:"payment_item_invoice_title"`
	PaymentItemInvoiceDue    *time.Time `gorm:"column:payment_item_invoice_due_date" json:"payment_item_invoice_due_date"`

	// Line title / deskripsi buat tampilan
	PaymentItemTitle       *string        `gorm:"column:payment_item_title" json:"payment_item_title"`
	PaymentItemDescription *string        `gorm:"column:payment_item_description" json:"payment_item_description"`
	PaymentItemMeta        datatypes.JSON `gorm:"column:payment_item_meta;type:jsonb" json:"payment_item_meta"`

	PaymentItemCreatedAt time.Time  `gorm:"column:payment_item_created_at;not null;default:now()" json:"payment_item_created_at"`
	PaymentItemUpdatedAt time.Time  `gorm:"column:payment_item_updated_at;not null;default:now()" json:"payment_item_updated_at"`
	PaymentItemDeletedAt *time.Time `gorm:"column:payment_item_deleted_at" json:"payment_item_deleted_at"`
}

func (PaymentItemModel) TableName() string {
	return "payment_items"
}
