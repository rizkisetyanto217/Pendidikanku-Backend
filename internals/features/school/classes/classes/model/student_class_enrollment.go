// file: internals/features/school/classes/class_enrollments/model/student_class_enrollments.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ======================================================
   ENUM mapping (Postgres: class_enrollment_status)
====================================================== */

type ClassEnrollmentStatus string

const (
	EnrollmentInitiated      ClassEnrollmentStatus = "initiated"
	EnrollmentPendingReview  ClassEnrollmentStatus = "pending_review"
	EnrollmentAwaitingPay    ClassEnrollmentStatus = "awaiting_payment"
	EnrollmentAccepted       ClassEnrollmentStatus = "accepted"
	EnrollmentWaitlisted     ClassEnrollmentStatus = "waitlisted"
	EnrollmentRejected       ClassEnrollmentStatus = "rejected"
	EnrollmentCanceled       ClassEnrollmentStatus = "canceled"
)

/* ======================================================
   Model: student_class_enrollments
====================================================== */

type StudentClassEnrollmentModel struct {
	StudentClassEnrollmentID              uuid.UUID          `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_class_enrollments_id" json:"student_class_enrollments_id"`

	// tenant & relations
	StudentClassEnrollmentSchoolID        uuid.UUID          `gorm:"not null;column:student_class_enrollments_school_id" json:"student_class_enrollments_school_id"`
	StudentClassEnrollmentSchoolStudentID uuid.UUID          `gorm:"not null;column:student_class_enrollments_school_student_id" json:"student_class_enrollments_school_student_id"`
	StudentClassEnrollmentClassID         uuid.UUID          `gorm:"not null;column:student_class_enrollments_class_id" json:"student_class_enrollments_class_id"`

	// status & biaya
	StudentClassEnrollmentStatus          ClassEnrollmentStatus `gorm:"type:class_enrollment_status;not null;default:'initiated';column:student_class_enrollments_status" json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR     int64               `gorm:"type:numeric(12,0);not null;default:0;check:student_class_enrollments_total_due_idr >= 0;column:student_class_enrollments_total_due_idr" json:"student_class_enrollments_total_due_idr"`

	// pembayaran (opsional)
	StudentClassEnrollmentPaymentID       *uuid.UUID         `gorm:"column:student_class_enrollments_payment_id" json:"student_class_enrollments_payment_id"`
	StudentClassEnrollmentPaymentSnapshot datatypes.JSON     `gorm:"column:student_class_enrollments_payment_snapshot" json:"student_class_enrollments_payment_snapshot"`

	// preferensi (opsional)
	StudentClassEnrollmentPreferences     datatypes.JSON     `gorm:"not null;default:'{}';column:student_class_enrollments_preferences" json:"student_class_enrollments_preferences"`

	// jejak waktu (audit)
	StudentClassEnrollmentAppliedAt       time.Time          `gorm:"not null;default:now();column:student_class_enrollments_applied_at" json:"student_class_enrollments_applied_at"`
	StudentClassEnrollmentReviewedAt      *time.Time         `gorm:"column:student_class_enrollments_reviewed_at" json:"student_class_enrollments_reviewed_at"`
	StudentClassEnrollmentAcceptedAt      *time.Time         `gorm:"column:student_class_enrollments_accepted_at" json:"student_class_enrollments_accepted_at"`
	StudentClassEnrollmentWaitlistedAt    *time.Time         `gorm:"column:student_class_enrollments_waitlisted_at" json:"student_class_enrollments_waitlisted_at"`
	StudentClassEnrollmentRejectedAt      *time.Time         `gorm:"column:student_class_enrollments_rejected_at" json:"student_class_enrollments_rejected_at"`
	StudentClassEnrollmentCanceledAt      *time.Time         `gorm:"column:student_class_enrollments_canceled_at" json:"student_class_enrollments_canceled_at"`

	StudentClassEnrollmentCreatedAt       time.Time          `gorm:"not null;default:now();column:student_class_enrollments_created_at" json:"student_class_enrollments_created_at"`
	StudentClassEnrollmentUpdatedAt       time.Time          `gorm:"not null;default:now();column:student_class_enrollments_updated_at" json:"student_class_enrollments_updated_at"`
	StudentClassEnrollmentDeletedAt       gorm.DeletedAt     `gorm:"column:student_class_enrollments_deleted_at;index" json:"student_class_enrollments_deleted_at"`
}

func (StudentClassEnrollmentModel) TableName() string { return "student_class_enrollments" }
