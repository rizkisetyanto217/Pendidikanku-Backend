// file: internals/features/school/classes/class_enrollments/model/student_class_enrollment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ======================================================
   ENUM: class_enrollment_status
====================================================== */

type ClassEnrollmentStatus string

const (
	ClassEnrollmentInitiated       ClassEnrollmentStatus = "initiated"
	ClassEnrollmentPendingReview   ClassEnrollmentStatus = "pending_review"
	ClassEnrollmentAwaitingPayment ClassEnrollmentStatus = "awaiting_payment"
	ClassEnrollmentAccepted        ClassEnrollmentStatus = "accepted"
	ClassEnrollmentWaitlisted      ClassEnrollmentStatus = "waitlisted"
	ClassEnrollmentRejected        ClassEnrollmentStatus = "rejected"
	ClassEnrollmentCanceled        ClassEnrollmentStatus = "canceled"
)

/* ======================================================
   Model: student_class_enrollments
====================================================== */

type StudentClassEnrollmentModel struct {
	// PK & Tenant
	StudentClassEnrollmentsID       uuid.UUID `gorm:"column:student_class_enrollments_id;type:uuid;default:gen_random_uuid();primaryKey" json:"student_class_enrollments_id"`
	StudentClassEnrollmentsSchoolID uuid.UUID `gorm:"column:student_class_enrollments_school_id;type:uuid;not null;index" json:"student_class_enrollments_school_id"`

	// Relasi tenant-safe
	StudentClassEnrollmentsSchoolStudentID uuid.UUID `gorm:"column:student_class_enrollments_school_student_id;type:uuid;not null;index" json:"student_class_enrollments_school_student_id"`
	StudentClassEnrollmentsClassID         uuid.UUID `gorm:"column:student_class_enrollments_class_id;type:uuid;not null;index" json:"student_class_enrollments_class_id"`

	// Status & biaya
	StudentClassEnrollmentsStatus      ClassEnrollmentStatus `gorm:"column:student_class_enrollments_status;type:class_enrollment_status;not null;default:'initiated'" json:"student_class_enrollments_status"`
	StudentClassEnrollmentsTotalDueIDR int64                 `gorm:"column:student_class_enrollments_total_due_idr;type:numeric(12,0);not null;default:0" json:"student_class_enrollments_total_due_idr"`

	// Pembayaran (opsional)
	StudentClassEnrollmentsPaymentID       *uuid.UUID     `gorm:"column:student_class_enrollments_payment_id;type:uuid" json:"student_class_enrollments_payment_id,omitempty"`
	StudentClassEnrollmentsPaymentSnapshot datatypes.JSON `gorm:"column:student_class_enrollments_payment_snapshot;type:jsonb" json:"student_class_enrollments_payment_snapshot,omitempty"`

	// Preferensi (JSONB object)
	StudentClassEnrollmentsPreferences datatypes.JSON `gorm:"column:student_class_enrollments_preferences;type:jsonb;not null;default:'{}'" json:"student_class_enrollments_preferences"`

	// ===== Snapshots dari classes =====
	StudentClassEnrollmentsClassNameSnapshot string  `gorm:"column:student_class_enrollments_class_name_snapshot;type:varchar(160)" json:"student_class_enrollments_class_name_snapshot"`
	StudentClassEnrollmentsClassSlugSnapshot *string `gorm:"column:student_class_enrollments_class_slug_snapshot;type:varchar(160)" json:"student_class_enrollments_class_slug_snapshot,omitempty"`

	// ===== TERM (denormalized dari classes → academic_terms) =====
	StudentClassEnrollmentsTermID                   *uuid.UUID `gorm:"column:student_class_enrollments_term_id;type:uuid" json:"student_class_enrollments_term_id,omitempty"`
	StudentClassEnrollmentsTermAcademicYearSnapshot *string    `gorm:"column:student_class_enrollments_term_academic_year_snapshot;type:text" json:"student_class_enrollments_term_academic_year_snapshot,omitempty"`
	StudentClassEnrollmentsTermNameSnapshot         *string    `gorm:"column:student_class_enrollments_term_name_snapshot;type:text" json:"student_class_enrollments_term_name_snapshot,omitempty"`
	StudentClassEnrollmentsTermSlugSnapshot         *string    `gorm:"column:student_class_enrollments_term_slug_snapshot;type:text" json:"student_class_enrollments_term_slug_snapshot,omitempty"`
	StudentClassEnrollmentsTermAngkatanSnapshot     *int       `gorm:"column:student_class_enrollments_term_angkatan_snapshot;type:int" json:"student_class_enrollments_term_angkatan_snapshot,omitempty"`

	// ===== SNAPSHOT dari school_students / user_profile =====
	StudentClassEnrollmentsUserProfileNameSnapshot              string  `gorm:"column:student_class_enrollments_user_profile_name_snapshot;type:varchar(80)" json:"student_class_enrollments_user_profile_name_snapshot"`
	StudentClassEnrollmentsUserProfileAvatarURLSnapshot         *string `gorm:"column:student_class_enrollments_user_profile_avatar_url_snapshot;type:varchar(255)" json:"student_class_enrollments_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassEnrollmentsUserProfileWhatsappURLSnapshot       *string `gorm:"column:student_class_enrollments_user_profile_whatsapp_url_snapshot;type:varchar(50)" json:"student_class_enrollments_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassEnrollmentsUserProfileParentNameSnapshot        *string `gorm:"column:student_class_enrollments_user_profile_parent_name_snapshot;type:varchar(80)" json:"student_class_enrollments_user_profile_parent_name_snapshot,omitempty"`
	StudentClassEnrollmentsUserProfileParentWhatsappURLSnapshot *string `gorm:"column:student_class_enrollments_user_profile_parent_whatsapp_url_snapshot;type:varchar(50)" json:"student_class_enrollments_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	StudentClassEnrollmentsUserProfileGenderSnapshot            *string `gorm:"column:student_class_enrollments_user_profile_gender_snapshot;type:varchar(20)" json:"student_class_enrollments_user_profile_gender_snapshot,omitempty"`

	StudentClassEnrollmentsStudentCodeSnapshot *string `gorm:"column:student_class_enrollments_student_code_snapshot;type:varchar(50)" json:"student_class_enrollments_student_code_snapshot,omitempty"`
	StudentClassEnrollmentsStudentSlugSnapshot *string `gorm:"column:student_class_enrollments_student_slug_snapshot;type:varchar(50)" json:"student_class_enrollments_student_slug_snapshot,omitempty"`

	// ===== CLASS SECTION (opsional) =====
	StudentClassEnrollmentsClassSectionID           *uuid.UUID `gorm:"column:student_class_enrollments_class_section_id;type:uuid" json:"student_class_enrollments_class_section_id,omitempty"`
	StudentClassEnrollmentsClassSectionNameSnapshot *string    `gorm:"column:student_class_enrollments_class_section_name_snapshot;type:varchar(50)" json:"student_class_enrollments_class_section_name_snapshot,omitempty"`
	StudentClassEnrollmentsClassSectionSlugSnapshot *string    `gorm:"column:student_class_enrollments_class_section_slug_snapshot;type:varchar(50)" json:"student_class_enrollments_class_section_slug_snapshot,omitempty"`

	// Jejak waktu proses
	StudentClassEnrollmentsAppliedAt    time.Time  `gorm:"column:student_class_enrollments_applied_at;type:timestamptz;not null;default:now()" json:"student_class_enrollments_applied_at"`
	StudentClassEnrollmentsReviewedAt   *time.Time `gorm:"column:student_class_enrollments_reviewed_at;type:timestamptz" json:"student_class_enrollments_reviewed_at,omitempty"`
	StudentClassEnrollmentsAcceptedAt   *time.Time `gorm:"column:student_class_enrollments_accepted_at;type:timestamptz" json:"student_class_enrollments_accepted_at,omitempty"`
	StudentClassEnrollmentsWaitlistedAt *time.Time `gorm:"column:student_class_enrollments_waitlisted_at;type:timestamptz" json:"student_class_enrollments_waitlisted_at,omitempty"`
	StudentClassEnrollmentsRejectedAt   *time.Time `gorm:"column:student_class_enrollments_rejected_at;type:timestamptz" json:"student_class_enrollments_rejected_at,omitempty"`
	StudentClassEnrollmentsCanceledAt   *time.Time `gorm:"column:student_class_enrollments_canceled_at;type:timestamptz" json:"student_class_enrollments_canceled_at,omitempty"`

	// Audit & soft delete
	StudentClassEnrollmentsCreatedAt time.Time      `gorm:"column:student_class_enrollments_created_at;type:timestamptz;not null;default:now();autoCreateTime" json:"student_class_enrollments_created_at"`
	StudentClassEnrollmentsUpdatedAt time.Time      `gorm:"column:student_class_enrollments_updated_at;type:timestamptz;not null;default:now();autoUpdateTime" json:"student_class_enrollments_updated_at"`
	StudentClassEnrollmentsDeletedAt gorm.DeletedAt `gorm:"column:student_class_enrollments_deleted_at;index" json:"student_class_enrollments_deleted_at,omitempty"`
}

func (StudentClassEnrollmentModel) TableName() string {
	return "student_class_enrollments"
}
