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
	BillBatchID        uuid.UUID  `gorm:"type:uuid;primaryKey;column:bill_batch_id"`
	BillBatchMasjidID  uuid.UUID  `gorm:"type:uuid;not null;column:bill_batch_masjid_id"` // meski di DB boleh NULL, di domain kita anggap wajib
	BillBatchClassID   *uuid.UUID `gorm:"type:uuid;column:bill_batch_class_id"`
	BillBatchSectionID *uuid.UUID `gorm:"type:uuid;column:bill_batch_section_id"`
	BillBatchMonth     int16      `gorm:"type:smallint;not null;column:bill_batch_month"`
	BillBatchYear      int16      `gorm:"type:smallint;not null;column:bill_batch_year"`
	BillBatchTermID    *uuid.UUID `gorm:"type:uuid;column:bill_batch_term_id"`
	BillBatchTitle     string     `gorm:"type:text;not null;column:bill_batch_title"`
	BillBatchDueDate   *time.Time `gorm:"type:date;column:bill_batch_due_date"`
	BillBatchNote      *string    `gorm:"type:text;column:bill_batch_note"`

	// kolom waktu eksplisit (bukan gorm.Model)
	BillBatchCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:bill_batch_created_at"`
	BillBatchUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:bill_batch_updated_at"`
	BillBatchDeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index;column:bill_batch_deleted_at"`
}

// TableName override
func (BillBatch) TableName() string {
	return "bill_batches"
}

// BeforeCreate: set ID jika kosong
func (b *BillBatch) BeforeCreate(tx *gorm.DB) error {
	if b.BillBatchID == uuid.Nil {
		b.BillBatchID = uuid.New()
	}
	// guard minimal: masjid_id wajib di sisi aplikasi
	if b.BillBatchMasjidID == uuid.Nil {
		return fmt.Errorf("bill_batch_masjid_id is required")
	}
	// XOR guard class/section (mirror constraint yang direkomendasikan)
	if (b.BillBatchClassID == nil && b.BillBatchSectionID == nil) ||
		(b.BillBatchClassID != nil && b.BillBatchSectionID != nil) {
		return fmt.Errorf("exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}
	return nil
}

// BeforeSave: jaga konsistensi waktu & XOR saat update
func (b *BillBatch) BeforeSave(tx *gorm.DB) error {
	// XOR guard tetap berlaku saat update
	if (b.BillBatchClassID == nil && b.BillBatchSectionID == nil) ||
		(b.BillBatchClassID != nil && b.BillBatchSectionID != nil) {
		return fmt.Errorf("exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}
	// perbarui updated_at kalau belum di-set oleh DB
	if b.BillBatchUpdatedAt.IsZero() {
		b.BillBatchUpdatedAt = time.Now()
	}
	return nil
}
