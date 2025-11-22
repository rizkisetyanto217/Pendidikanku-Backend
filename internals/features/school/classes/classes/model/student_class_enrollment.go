// file: internals/features/school/classes/class_enrollments/model/student_class_enrollments.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ==============================================
   ENUM mapping (Postgres: class_enrollment_status)
================================================= */

type ClassEnrollmentStatus string

const (
	EnrollmentInitiated     ClassEnrollmentStatus = "initiated"
	EnrollmentPendingReview ClassEnrollmentStatus = "pending_review"
	EnrollmentAwaitingPay   ClassEnrollmentStatus = "awaiting_payment"
	EnrollmentAccepted      ClassEnrollmentStatus = "accepted"
	EnrollmentWaitlisted    ClassEnrollmentStatus = "waitlisted"
	EnrollmentRejected      ClassEnrollmentStatus = "rejected"
	EnrollmentCanceled      ClassEnrollmentStatus = "canceled"
)

/* ==============================================
   Model: student_class_enrollments
================================================= */

type StudentClassEnrollmentModel struct {
	// PK
	StudentClassEnrollmentID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_class_enrollments_id" json:"student_class_enrollments_id"`

	// Tenant & relations
	StudentClassEnrollmentSchoolID        uuid.UUID `gorm:"not null;column:student_class_enrollments_school_id" json:"student_class_enrollments_school_id"`
	StudentClassEnrollmentSchoolStudentID uuid.UUID `gorm:"not null;column:student_class_enrollments_school_student_id" json:"student_class_enrollments_school_student_id"`
	StudentClassEnrollmentClassID         uuid.UUID `gorm:"not null;column:student_class_enrollments_class_id" json:"student_class_enrollments_class_id"`

	// Status & biaya
	StudentClassEnrollmentStatus      ClassEnrollmentStatus `gorm:"type:class_enrollment_status;not null;default:'initiated';column:student_class_enrollments_status" json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR int64                 `gorm:"type:numeric(12,0);not null;default:0;column:student_class_enrollments_total_due_idr" json:"student_class_enrollments_total_due_idr"`

	// Pembayaran (opsional)
	StudentClassEnrollmentPaymentID       *uuid.UUID     `gorm:"column:student_class_enrollments_payment_id" json:"student_class_enrollments_payment_id"`
	StudentClassEnrollmentPaymentSnapshot datatypes.JSON `gorm:"type:jsonb;column:student_class_enrollments_payment_snapshot" json:"student_class_enrollments_payment_snapshot"`

	// Preferensi (opsional) → JSON object
	StudentClassEnrollmentPreferences datatypes.JSON `gorm:"type:jsonb;not null;default:'{}';column:student_class_enrollments_preferences" json:"student_class_enrollments_preferences"`

	// ===== Snapshots dari classes =====
	StudentClassEnrollmentClassNameSnapshot string `gorm:"type:varchar(160);column:student_class_enrollments_class_name_snapshot" json:"student_class_enrollments_class_name_snapshot"`
	StudentClassEnrollmentClassSlugSnapshot string `gorm:"type:varchar(160);column:student_class_enrollments_class_slug_snapshot" json:"student_class_enrollments_class_slug_snapshot"`

	// ===== Denormalized TERM (sinkron via trigger dari classes) =====
	StudentClassEnrollmentTermID                   *uuid.UUID `gorm:"column:student_class_enrollments_term_id" json:"student_class_enrollments_term_id"`
	StudentClassEnrollmentTermAcademicYearSnapshot *string    `gorm:"column:student_class_enrollments_term_academic_year_snapshot" json:"student_class_enrollments_term_academic_year_snapshot"`
	StudentClassEnrollmentTermNameSnapshot         *string    `gorm:"column:student_class_enrollments_term_name_snapshot" json:"student_class_enrollments_term_name_snapshot"`
	StudentClassEnrollmentTermSlugSnapshot         *string    `gorm:"column:student_class_enrollments_term_slug_snapshot" json:"student_class_enrollments_term_slug_snapshot"`
	StudentClassEnrollmentTermAngkatanSnapshot     *int       `gorm:"column:student_class_enrollments_term_angkatan_snapshot" json:"student_class_enrollments_term_angkatan_snapshot"`

	// ===== Snapshots identitas siswa =====
	StudentClassEnrollmentStudentNameSnapshot string `gorm:"type:varchar(80);column:student_class_enrollments_student_name_snapshot" json:"student_class_enrollments_student_name_snapshot"`
	StudentClassEnrollmentStudentCodeSnapshot string `gorm:"type:varchar(50);column:student_class_enrollments_student_code_snapshot" json:"student_class_enrollments_student_code_snapshot"`
	StudentClassEnrollmentStudentSlugSnapshot string `gorm:"type:varchar(50);column:student_class_enrollments_student_slug_snapshot" json:"student_class_enrollments_student_slug_snapshot"`

	// ===== Convenience fields (TIDAK disimpan di DB) =====
	StudentClassEnrollmentStudentName string `gorm:"-" json:"student_class_enrollments_student_name"`
	StudentClassEnrollmentClassName   string `gorm:"-" json:"student_class_enrollments_class_name"`

	// Jejak waktu (audit)
	StudentClassEnrollmentAppliedAt    time.Time  `gorm:"not null;default:now();column:student_class_enrollments_applied_at" json:"student_class_enrollments_applied_at"`
	StudentClassEnrollmentReviewedAt   *time.Time `gorm:"column:student_class_enrollments_reviewed_at" json:"student_class_enrollments_reviewed_at"`
	StudentClassEnrollmentAcceptedAt   *time.Time `gorm:"column:student_class_enrollments_accepted_at" json:"student_class_enrollments_accepted_at"`
	StudentClassEnrollmentWaitlistedAt *time.Time `gorm:"column:student_class_enrollments_waitlisted_at" json:"student_class_enrollments_waitlisted_at"`
	StudentClassEnrollmentRejectedAt   *time.Time `gorm:"column:student_class_enrollments_rejected_at" json:"student_class_enrollments_rejected_at"`
	StudentClassEnrollmentCanceledAt   *time.Time `gorm:"column:student_class_enrollments_canceled_at" json:"student_class_enrollments_canceled_at"`

	StudentClassEnrollmentCreatedAt time.Time      `gorm:"not null;default:now();column:student_class_enrollments_created_at" json:"student_class_enrollments_created_at"`
	StudentClassEnrollmentUpdatedAt time.Time      `gorm:"not null;default:now();column:student_class_enrollments_updated_at" json:"student_class_enrollments_updated_at"`
	StudentClassEnrollmentDeletedAt gorm.DeletedAt `gorm:"column:student_class_enrollments_deleted_at;index" json:"student_class_enrollments_deleted_at"`
}

func (StudentClassEnrollmentModel) TableName() string { return "student_class_enrollments" }
