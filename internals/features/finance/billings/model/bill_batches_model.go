// file: internals/features/finance/spp/model/bill_batch.go
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BillBatch merepresentasikan tabel bill_batches
type BillBatch struct {
	// PK
	BillBatchID uuid.UUID `gorm:"type:uuid;primaryKey;column:bill_batch_id" json:"bill_batch_id"`

	// Tenant
	BillBatchMasjidID uuid.UUID `gorm:"type:uuid;not null;column:bill_batch_masjid_id" json:"bill_batch_masjid_id"`

	// Scope (XOR: tepat salah satu yang terisi)
	BillBatchClassID   *uuid.UUID `gorm:"type:uuid;column:bill_batch_class_id" json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `gorm:"type:uuid;column:bill_batch_section_id" json:"bill_batch_section_id,omitempty"`

	// Periode (nullable untuk one-off)
	BillBatchMonth  *int16     `gorm:"type:smallint;column:bill_batch_month" json:"bill_batch_month,omitempty"`
	BillBatchYear   *int16     `gorm:"type:smallint;column:bill_batch_year" json:"bill_batch_year,omitempty"`
	BillBatchTermID *uuid.UUID `gorm:"type:uuid;column:bill_batch_term_id" json:"bill_batch_term_id,omitempty"`

	// Katalog jenis + kode
	BillBatchGeneralBillingKindID *uuid.UUID `gorm:"type:uuid;column:bill_batch_general_billing_kind_id" json:"bill_batch_general_billing_kind_id,omitempty"`
	BillBatchBillCode             string     `gorm:"type:varchar(60);not null;default:SPP;column:bill_batch_bill_code" json:"bill_batch_bill_code"`
	BillBatchOptionCode           *string    `gorm:"type:varchar(60);column:bill_batch_option_code" json:"bill_batch_option_code,omitempty"` // wajib untuk one-off

	// Info tagihan
	BillBatchTitle   string     `gorm:"type:text;not null;column:bill_batch_title" json:"bill_batch_title"`
	BillBatchDueDate *time.Time `gorm:"type:date;column:bill_batch_due_date" json:"bill_batch_due_date,omitempty"`
	BillBatchNote    *string    `gorm:"type:text;column:bill_batch_note" json:"bill_batch_note,omitempty"`

	// Denormalized totals (diupdate dari backend)
	BillBatchTotalAmountIDR    int `gorm:"type:int;not null;default:0;column:bill_batch_total_amount_idr" json:"bill_batch_total_amount_idr"`
	BillBatchTotalPaidIDR      int `gorm:"type:int;not null;default:0;column:bill_batch_total_paid_idr" json:"bill_batch_total_paid_idr"`
	BillBatchTotalStudents     int `gorm:"type:int;not null;default:0;column:bill_batch_total_students" json:"bill_batch_total_students"`
	BillBatchTotalStudentsPaid int `gorm:"type:int;not null;default:0;column:bill_batch_total_students_paid" json:"bill_batch_total_students_paid"`

	// Timestamps (eksplisit)
	BillBatchCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:bill_batch_created_at" json:"bill_batch_created_at"`
	BillBatchUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:bill_batch_updated_at" json:"bill_batch_updated_at"`
	BillBatchDeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index;column:bill_batch_deleted_at" json:"-"` // soft delete
}

func (BillBatch) TableName() string { return "bill_batches" }

// helper
func (b *BillBatch) isOneOff() bool {
	return b.BillBatchOptionCode != nil && strings.TrimSpace(*b.BillBatchOptionCode) != ""
}

// BeforeCreate: set ID, enforce XOR, periodic vs one-off, dan default code
func (b *BillBatch) BeforeCreate(tx *gorm.DB) error {
	if b.BillBatchID == uuid.Nil {
		b.BillBatchID = uuid.New()
	}
	if b.BillBatchMasjidID == uuid.Nil {
		return fmt.Errorf("bill_batch_masjid_id is required")
	}
	// XOR guard class/section
	if (b.BillBatchClassID == nil && b.BillBatchSectionID == nil) ||
		(b.BillBatchClassID != nil && b.BillBatchSectionID != nil) {
		return fmt.Errorf("exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}
	// default bill code
	if strings.TrimSpace(b.BillBatchBillCode) == "" {
		b.BillBatchBillCode = "SPP"
	}

	// Periodic vs One-off
	if b.isOneOff() {
		// one-off: YM opsional (DDL tidak mewajibkan)
		// tidak ada validasi tambahan di sini, biarkan service layer yang atur nilai YM jika memang ingin diisi
	} else {
		// periodic: YM wajib
		if b.BillBatchMonth == nil || b.BillBatchYear == nil {
			return fmt.Errorf("periodic batch requires bill_batch_month and bill_batch_year")
		}
		if *b.BillBatchMonth < 1 || *b.BillBatchMonth > 12 {
			return fmt.Errorf("bill_batch_month must be between 1 and 12")
		}
		if *b.BillBatchYear < 2000 || *b.BillBatchYear > 2100 {
			return fmt.Errorf("bill_batch_year must be between 2000 and 2100")
		}
	}

	// timestamps
	now := time.Now()
	if b.BillBatchCreatedAt.IsZero() {
		b.BillBatchCreatedAt = now
	}
	b.BillBatchUpdatedAt = b.BillBatchCreatedAt
	return nil
}

// BeforeUpdate: enforce XOR, set updated_at, dan validasi ringan YM
func (b *BillBatch) BeforeUpdate(tx *gorm.DB) error {
	if (b.BillBatchClassID == nil && b.BillBatchSectionID == nil) ||
		(b.BillBatchClassID != nil && b.BillBatchSectionID != nil) {
		return fmt.Errorf("exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}

	// default bill code jika kosong
	if strings.TrimSpace(b.BillBatchBillCode) == "" {
		b.BillBatchBillCode = "SPP"
	}

	// Validasi ringan YM saat ada isian
	if b.BillBatchMonth != nil {
		if *b.BillBatchMonth < 1 || *b.BillBatchMonth > 12 {
			return fmt.Errorf("bill_batch_month must be between 1 and 12")
		}
	}
	if b.BillBatchYear != nil {
		if *b.BillBatchYear < 2000 || *b.BillBatchYear > 2100 {
			return fmt.Errorf("bill_batch_year must be between 2000 and 2100")
		}
	}

	b.BillBatchUpdatedAt = time.Now()
	return nil
}
