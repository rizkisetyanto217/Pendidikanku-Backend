// file: internals/features/finance/general_billings/model/general_billing.go
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   Snapshot helper structs (MINIMAL)
   ========================= */

type GeneralBillingKindSnapshotPayload struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code,omitempty"`
	Name string    `json:"name,omitempty"`
}

type GeneralBillingClassSnapshotPayload struct {
	ID   uuid.UUID `json:"id"`             // class_id
	Name string    `json:"name,omitempty"` // class_name
	Slug string    `json:"slug,omitempty"` // class_slug
}

type GeneralBillingSectionSnapshotPayload struct {
	ID   uuid.UUID `json:"id"`             // class_section_id
	Name string    `json:"name,omitempty"` // section name
	Code string    `json:"code,omitempty"` // section code (optional)
}

type GeneralBillingTermSnapshotPayload struct {
	ID           uuid.UUID `json:"id"`                      // academic_term_id
	AcademicYear string    `json:"academic_year,omitempty"` // "2025/2026"
	Name         string    `json:"name,omitempty"`          // term name
	Slug         string    `json:"slug,omitempty"`          // term slug
}

/* =========================
   Model: general_billings
   ========================= */

type GeneralBilling struct {
	GeneralBillingID uuid.UUID `json:"general_billing_id" gorm:"column:general_billing_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// tenant scope (ON DELETE CASCADE)
	GeneralBillingMasjidID uuid.UUID `json:"general_billing_masjid_id" gorm:"column:general_billing_masjid_id;type:uuid;not null;constraint:OnDelete:CASCADE"`

	// kind (ON UPDATE CASCADE, ON DELETE RESTRICT)
	GeneralBillingKindID uuid.UUID `json:"general_billing_kind_id" gorm:"column:general_billing_kind_id;type:uuid;not null;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	// basic fields
	GeneralBillingCode  *string `json:"general_billing_code,omitempty"  gorm:"column:general_billing_code;type:varchar(60)"`
	GeneralBillingTitle string  `json:"general_billing_title"           gorm:"column:general_billing_title;type:text;not null"`
	GeneralBillingDesc  *string `json:"general_billing_desc,omitempty"  gorm:"column:general_billing_desc;type:text"`

	// academic scope (nullable, ON DELETE SET NULL)
	GeneralBillingClassID   *uuid.UUID `json:"general_billing_class_id,omitempty"   gorm:"column:general_billing_class_id;type:uuid;constraint:OnDelete:SET NULL"`
	GeneralBillingSectionID *uuid.UUID `json:"general_billing_section_id,omitempty" gorm:"column:general_billing_section_id;type:uuid;constraint:OnDelete:SET NULL"`
	GeneralBillingTermID    *uuid.UUID `json:"general_billing_term_id,omitempty"    gorm:"column:general_billing_term_id;type:uuid;constraint:OnDelete:SET NULL"`

	// schedule/flags
	GeneralBillingDueDate  *time.Time `json:"general_billing_due_date,omitempty" gorm:"column:general_billing_due_date;type:date"`
	GeneralBillingIsActive bool       `json:"general_billing_is_active"          gorm:"column:general_billing_is_active;not null;default:true"`

	// default amount (INT >= 0) — gunakan validator di layer atas untuk cek >= 0
	GeneralBillingDefaultAmountIdr *int `json:"general_billing_default_amount_idr,omitempty" gorm:"column:general_billing_default_amount_idr;type:int"`

	// snapshots (JSONB, nullable)
	GeneralBillingKindSnapshot    datatypes.JSON `json:"general_billing_kind_snapshot,omitempty"    gorm:"column:general_billing_kind_snapshot;type:jsonb"`
	GeneralBillingClassSnapshot   datatypes.JSON `json:"general_billing_class_snapshot,omitempty"   gorm:"column:general_billing_class_snapshot;type:jsonb"`
	GeneralBillingSectionSnapshot datatypes.JSON `json:"general_billing_section_snapshot,omitempty" gorm:"column:general_billing_section_snapshot;type:jsonb"`
	GeneralBillingTermSnapshot    datatypes.JSON `json:"general_billing_term_snapshot,omitempty"    gorm:"column:general_billing_term_snapshot;type:jsonb"`

	// timestamps (soft delete manual, bukan gorm.DeletedAt)
	GeneralBillingCreatedAt time.Time  `json:"general_billing_created_at"           gorm:"column:general_billing_created_at;type:timestamptz;not null;default:now()"`
	GeneralBillingUpdatedAt time.Time  `json:"general_billing_updated_at"           gorm:"column:general_billing_updated_at;type:timestamptz;not null;default:now()"`
	GeneralBillingDeletedAt *time.Time `json:"general_billing_deleted_at,omitempty" gorm:"column:general_billing_deleted_at;type:timestamptz"`
}

func (GeneralBilling) TableName() string { return "general_billings" }

/* =========================
   Hooks: refresh updated_at
   ========================= */

func (g *GeneralBilling) BeforeCreate(tx *gorm.DB) error {
	g.GeneralBillingUpdatedAt = time.Now().UTC()
	return nil
}
func (g *GeneralBilling) BeforeUpdate(tx *gorm.DB) error {
	g.GeneralBillingUpdatedAt = time.Now().UTC()
	return nil
}

/* =========================
   Scopes
   ========================= */

func ScopeAlive(db *gorm.DB) *gorm.DB {
	return db.Where("general_billing_deleted_at IS NULL")
}
func ScopeByTenant(masjidID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("general_billing_masjid_id = ?", masjidID)
	}
}
func ScopeByKind(kindID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("general_billing_kind_id = ?", kindID)
	}
}

/* =========================
   Snapshot setters (JSONB)
   ========================= */

func (g *GeneralBilling) SetGeneralBillingKindSnapshot(v GeneralBillingKindSnapshotPayload) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	g.GeneralBillingKindSnapshot = datatypes.JSON(b)
	return nil
}
func (g *GeneralBilling) SetGeneralBillingClassSnapshot(v *GeneralBillingClassSnapshotPayload) error {
	if v == nil {
		g.GeneralBillingClassSnapshot = nil
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	g.GeneralBillingClassSnapshot = datatypes.JSON(b)
	return nil
}
func (g *GeneralBilling) SetGeneralBillingSectionSnapshot(v *GeneralBillingSectionSnapshotPayload) error {
	if v == nil {
		g.GeneralBillingSectionSnapshot = nil
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	g.GeneralBillingSectionSnapshot = datatypes.JSON(b)
	return nil
}
func (g *GeneralBilling) SetGeneralBillingTermSnapshot(v *GeneralBillingTermSnapshotPayload) error {
	if v == nil {
		g.GeneralBillingTermSnapshot = nil
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	g.GeneralBillingTermSnapshot = datatypes.JSON(b)
	return nil
}
