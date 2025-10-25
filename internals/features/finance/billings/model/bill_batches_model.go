// file: internals/features/finance/spp/model/bill_batch.go
package model

import (
	"fmt"
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

	// Periode
	BillBatchMonth  int16      `gorm:"type:smallint;not null;column:bill_batch_month" json:"bill_batch_month"`
	BillBatchYear   int16      `gorm:"type:smallint;not null;column:bill_batch_year" json:"bill_batch_year"`
	BillBatchTermID *uuid.UUID `gorm:"type:uuid;column:bill_batch_term_id" json:"bill_batch_term_id,omitempty"`

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

// BeforeCreate: set ID & guard minimal
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
	// created_at akan diisi DB (default now()), tapi aman jika perlu set manual
	if b.BillBatchCreatedAt.IsZero() {
		b.BillBatchCreatedAt = time.Now()
	}
	// sync updated_at
	b.BillBatchUpdatedAt = b.BillBatchCreatedAt
	return nil
}

// BeforeUpdate: enforce XOR & update updated_at
func (b *BillBatch) BeforeUpdate(tx *gorm.DB) error {
	if (b.BillBatchClassID == nil && b.BillBatchSectionID == nil) ||
		(b.BillBatchClassID != nil && b.BillBatchSectionID != nil) {
		return fmt.Errorf("exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}
	b.BillBatchUpdatedAt = time.Now()
	return nil
}


