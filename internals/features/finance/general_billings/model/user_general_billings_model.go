// file: internals/features/finance/general_billings/model/user_general_billing.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ===================== Status Constants ===================== */

const (
	UserGeneralBillingStatusUnpaid   = "unpaid"
	UserGeneralBillingStatusPaid     = "paid"
	UserGeneralBillingStatusCanceled = "canceled"
)

/* ===================== Model ===================== */

type UserGeneralBilling struct {
	// PK
	UserGeneralBillingID uuid.UUID `json:"user_general_billing_id" gorm:"type:uuid;primaryKey;column:user_general_billing_id;default:gen_random_uuid()"`

	// Tenant & subject (student)
	UserGeneralBillingSchoolID        uuid.UUID  `json:"user_general_billing_school_id" gorm:"type:uuid;not null;column:user_general_billing_school_id"`
	UserGeneralBillingSchoolStudentID *uuid.UUID `json:"user_general_billing_school_student_id" gorm:"type:uuid;column:user_general_billing_school_student_id"`

	// Payer (users)
	UserGeneralBillingPayerUserID *uuid.UUID `json:"user_general_billing_payer_user_id" gorm:"type:uuid;column:user_general_billing_payer_user_id"`

	// Billing reference
	UserGeneralBillingBillingID uuid.UUID `json:"user_general_billing_billing_id" gorm:"type:uuid;not null;column:user_general_billing_billing_id"`

	// Amount & status
	UserGeneralBillingAmountIDR int    `json:"user_general_billing_amount_idr" gorm:"type:integer;not null;column:user_general_billing_amount_idr;check:user_general_billing_amount_idr>=0"`
	UserGeneralBillingStatus    string `json:"user_general_billing_status" gorm:"type:varchar(20);not null;default:unpaid;column:user_general_billing_status"`

	UserGeneralBillingPaidAt *time.Time `json:"user_general_billing_paid_at" gorm:"type:timestamptz;column:user_general_billing_paid_at"`
	UserGeneralBillingNote   *string    `json:"user_general_billing_note" gorm:"type:text;column:user_general_billing_note"`

	// Snapshots
	UserGeneralBillingTitleSnapshot    *string `json:"user_general_billing_title_snapshot" gorm:"type:text;column:user_general_billing_title_snapshot"`
	UserGeneralBillingKindCodeSnapshot *string `json:"user_general_billing_kind_code_snapshot" gorm:"type:text;column:user_general_billing_kind_code_snapshot"`
	UserGeneralBillingKindNameSnapshot *string `json:"user_general_billing_kind_name_snapshot" gorm:"type:text;column:user_general_billing_kind_name_snapshot"`

	// Metadata
	UserGeneralBillingMeta datatypes.JSONMap `json:"user_general_billing_meta" gorm:"type:jsonb;column:user_general_billing_meta"`

	// Timestamps
	UserGeneralBillingCreatedAt time.Time      `json:"user_general_billing_created_at" gorm:"type:timestamptz;not null;autoCreateTime;column:user_general_billing_created_at"`
	UserGeneralBillingUpdatedAt time.Time      `json:"user_general_billing_updated_at" gorm:"type:timestamptz;not null;autoUpdateTime;column:user_general_billing_updated_at"`
	UserGeneralBillingDeletedAt gorm.DeletedAt `json:"user_general_billing_deleted_at" gorm:"type:timestamptz;index;column:user_general_billing_deleted_at"`

	/* ========== Relations (optional) ========== */
	// - School Student (composite FK)
	SchoolStudent *SchoolStudent `json:"school_student,omitempty" gorm:"foreignKey:UserGeneralBillingSchoolStudentID,UserGeneralBillingSchoolID;references:SchoolStudentID,SchoolStudentSchoolID"`

	// - Payer User
	PayerUser *User `json:"payer_user,omitempty" gorm:"foreignKey:UserGeneralBillingPayerUserID;references:ID"`

	// - General Billing
	GeneralBilling *GeneralBilling `json:"general_billing,omitempty" gorm:"foreignKey:UserGeneralBillingBillingID;references:GeneralBillingID"`
}

/* ===================== Table & Indexes ===================== */

func (UserGeneralBilling) TableName() string { return "user_general_billings" }

// Index definitions using GORM tags on fields above:
//
//   CONSTRAINT uq_ugb_per_student UNIQUE (user_general_billing_billing_id, user_general_billing_school_student_id)
//   CONSTRAINT uq_ugb_per_payer   UNIQUE (user_general_billing_billing_id, user_general_billing_payer_user_id)
//
// Achieved via the following tags (already applied on fields):
//
//   UserGeneralBillingBillingID        gorm:"uniqueIndex:uq_ugb_per_student;uniqueIndex:uq_ugb_per_payer"
//   UserGeneralBillingSchoolStudentID  gorm:"uniqueIndex:uq_ugb_per_student"
//   UserGeneralBillingPayerUserID      gorm:"uniqueIndex:uq_ugb_per_payer"

/* ===================== Lightweight related stubs ===================== */
/* Sesuaikan dengan model asli di repo Anda (path & kolom) */

type User struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;column:id"`
}

type SchoolStudent struct {
	SchoolStudentID       uuid.UUID `gorm:"type:uuid;primaryKey;column:school_student_id"`
	SchoolStudentSchoolID uuid.UUID `gorm:"type:uuid;column:school_student_school_id"`
}

// type GeneralBilling struct {
// 	GeneralBillingID uuid.UUID `gorm:"type:uuid;primaryKey;column:general_billing_id"`
// }
