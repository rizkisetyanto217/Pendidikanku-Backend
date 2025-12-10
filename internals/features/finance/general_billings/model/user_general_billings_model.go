// file: internals/features/finance/general_billings/model/user_general_billing.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ===================== Status Constants ===================== */

const (
	UserGeneralBillingStatusUnpaid   = "unpaid"
	UserGeneralBillingStatusPaid     = "paid"
	UserGeneralBillingStatusCanceled = "canceled"
)

/* ===================== Model ===================== */

type UserGeneralBillingModel struct {
	// PK
	UserGeneralBillingID uuid.UUID `json:"user_general_billing_id" gorm:"column:user_general_billing_id;type:uuid;primaryKey"`

	// Tenant & subject (student)
	UserGeneralBillingSchoolID        uuid.UUID  `json:"user_general_billing_school_id" gorm:"column:user_general_billing_school_id;type:uuid;not null"`
	UserGeneralBillingSchoolStudentID *uuid.UUID `json:"user_general_billing_school_student_id,omitempty" gorm:"column:user_general_billing_school_student_id;type:uuid"`

	// Payer (users)
	UserGeneralBillingPayerUserID *uuid.UUID `json:"user_general_billing_payer_user_id,omitempty" gorm:"column:user_general_billing_payer_user_id;type:uuid"`

	// Billing reference
	UserGeneralBillingBillingID uuid.UUID `json:"user_general_billing_billing_id" gorm:"column:user_general_billing_billing_id;type:uuid;not null"`

	// Amount & status
	// CHECK (>= 0) & DEFAULT 'unpaid' sudah di-handle di SQL
	UserGeneralBillingAmountIDR int    `json:"user_general_billing_amount_idr" gorm:"column:user_general_billing_amount_idr;type:int;not null"`
	UserGeneralBillingStatus    string `json:"user_general_billing_status" gorm:"column:user_general_billing_status;type:varchar(20);not null"`

	UserGeneralBillingPaidAt *time.Time `json:"user_general_billing_paid_at,omitempty" gorm:"column:user_general_billing_paid_at;type:timestamptz"`
	UserGeneralBillingNote   *string    `json:"user_general_billing_note,omitempty" gorm:"column:user_general_billing_note;type:text"`

	// Snapshots (selaras dengan SQL)
	UserGeneralBillingTitleSnapshot    *string                 `json:"user_general_billing_title_snapshot,omitempty" gorm:"column:user_general_billing_title_snapshot;type:text"`
	UserGeneralBillingCategorySnapshot *GeneralBillingCategory `json:"user_general_billing_category_snapshot,omitempty" gorm:"column:user_general_billing_category_snapshot;type:general_billing_category"`
	UserGeneralBillingBillCodeSnapshot *string                 `json:"user_general_billing_bill_code_snapshot,omitempty" gorm:"column:user_general_billing_bill_code_snapshot;type:varchar(60)"`

	// Metadata fleksibel (DEFAULT '{}'::jsonb ada di SQL)
	UserGeneralBillingMeta datatypes.JSONMap `json:"user_general_billing_meta" gorm:"column:user_general_billing_meta;type:jsonb"`

	// Timestamps (soft delete manual) — default now() di SQL
	UserGeneralBillingCreatedAt time.Time  `json:"user_general_billing_created_at" gorm:"column:user_general_billing_created_at;type:timestamptz;not null"`
	UserGeneralBillingUpdatedAt time.Time  `json:"user_general_billing_updated_at" gorm:"column:user_general_billing_updated_at;type:timestamptz;not null"`
	UserGeneralBillingDeletedAt *time.Time `json:"user_general_billing_deleted_at,omitempty" gorm:"column:user_general_billing_deleted_at;type:timestamptz"`

	/* ========== Relations (sesuai composite FK di SQL) ========== */

	// SchoolStudent: composite FK (school_student_id, school_id)
	SchoolStudent *SchoolStudent `json:"school_student,omitempty" gorm:"foreignKey:UserGeneralBillingSchoolStudentID,UserGeneralBillingSchoolID;references:SchoolStudentID,SchoolStudentSchoolID"`

	// Payer User
	PayerUser *User `json:"payer_user,omitempty" gorm:"foreignKey:UserGeneralBillingPayerUserID;references:ID"`

	// General Billing
	GeneralBilling *GeneralBillingModel `json:"general_billing,omitempty" gorm:"foreignKey:UserGeneralBillingBillingID;references:GeneralBillingID"`
}

/* ===================== Table Name ===================== */

func (UserGeneralBillingModel) TableName() string { return "user_general_billings" }

/* ===================== Lightweight related stubs ===================== */
/* Sesuaikan dengan model asli di repo kamu (path & kolom) */

type User struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;column:id"`
}

type SchoolStudent struct {
	SchoolStudentID       uuid.UUID `gorm:"type:uuid;primaryKey;column:school_student_id"`
	SchoolStudentSchoolID uuid.UUID `gorm:"type:uuid;column:school_student_school_id"`
}

// GeneralBillingModel sudah ada di file general_billing.go
// type GeneralBillingModel struct { ... }
