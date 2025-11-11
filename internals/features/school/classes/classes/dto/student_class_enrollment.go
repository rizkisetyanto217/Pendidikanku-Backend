// file: internals/features/school/classes/class_enrollments/dto/student_class_enrollments_dto.go
package dto

import (
	"encoding/json"
	"time"

	m "schoolku_backend/internals/features/school/classes/classes/model"
	h "schoolku_backend/internals/helpers"

	"github.com/google/uuid"
)

/* ======================================================
   Requests
====================================================== */

// Create request body (POST /class-enrollments)
type CreateStudentClassEnrollmentRequest struct {
	// required
	SchoolStudentID uuid.UUID `json:"student_class_enrollments_school_student_id"`
	ClassID         uuid.UUID `json:"student_class_enrollments_class_id"`

	// optional
	TotalDueIDR int64                  `json:"student_class_enrollments_total_due_idr"`
	Preferences map[string]interface{} `json:"student_class_enrollments_preferences"`
}

// Update mutable fields except status (PATCH /:id)
type UpdateStudentClassEnrollmentRequest struct {
	TotalDueIDR *int64                 `json:"student_class_enrollments_total_due_idr"`
	Preferences map[string]interface{} `json:"student_class_enrollments_preferences"`
}

// Update status with optional timestamp stamps (PATCH /:id/status)
type UpdateStudentClassEnrollmentStatusRequest struct {
	Status m.ClassEnrollmentStatus `json:"student_class_enrollments_status"`

	// Optional explicit stamps when moving status (backend may also auto-fill)
	ReviewedAt   *time.Time `json:"student_class_enrollments_reviewed_at"`
	AcceptedAt   *time.Time `json:"student_class_enrollments_accepted_at"`
	WaitlistedAt *time.Time `json:"student_class_enrollments_waitlisted_at"`
	RejectedAt   *time.Time `json:"student_class_enrollments_rejected_at"`
	CanceledAt   *time.Time `json:"student_class_enrollments_canceled_at"`
}

/* ======================================================
   Query params (List)
====================================================== */

type ListStudentClassEnrollmentQuery struct {
	// filters (existing)
	StudentID *uuid.UUID                `query:"student_id"`
	ClassID   *uuid.UUID                `query:"class_id"`
	StatusIn  []m.ClassEnrollmentStatus `query:"status_in"` // comma-separated → parser di controller

	AppliedFrom *time.Time `query:"applied_from"`
	AppliedTo   *time.Time `query:"applied_to"`

	OnlyAlive *bool `query:"only_alive"`

	// NEW: term filters (denormalized kolom)
	TermID       *uuid.UUID `query:"term_id"`
	AcademicYear string     `query:"academic_year"` // ex: "2026/2027"
	Angkatan     *int       `query:"angkatan"`      // ex: 2024

	// NEW: simple search (on snapshots: student/class/term)
	Q string `query:"q"`

	// paging & sort
	Limit   int    `query:"limit"`
	Offset  int    `query:"offset"`
	OrderBy string `query:"order_by"` // one of: created_at, applied_at, updated_at
	Sort    string `query:"sort"`     // asc|desc
}

// Recommended defaults (controller may enforce)
const (
	OrderByCreated = "created_at"
	OrderByApplied = "applied_at"
	OrderByUpdated = "updated_at"
)

/* ======================================================
   Response
====================================================== */

// Wrapper untuk helper.JsonList (optional)
type StudentClassEnrollmentListResponse struct {
	Message    string                           `json:"message"`
	Data       []StudentClassEnrollmentResponse `json:"data"`
	Pagination *h.Pagination                    `json:"pagination,omitempty"`
}

type StudentClassEnrollmentResponse struct {
	StudentClassEnrollmentID              uuid.UUID `json:"student_class_enrollments_id"`
	StudentClassEnrollmentSchoolID        uuid.UUID `json:"student_class_enrollments_school_id"`
	StudentClassEnrollmentSchoolStudentID uuid.UUID `json:"student_class_enrollments_school_student_id"`
	StudentClassEnrollmentClassID         uuid.UUID `json:"student_class_enrollments_class_id"`

	StudentClassEnrollmentStatus      m.ClassEnrollmentStatus `json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR int64                   `json:"student_class_enrollments_total_due_idr"`

	StudentClassEnrollmentPaymentID       *uuid.UUID             `json:"student_class_enrollments_payment_id,omitempty"`
	StudentClassEnrollmentPaymentSnapshot map[string]interface{} `json:"student_class_enrollments_payment_snapshot,omitempty"`

	StudentClassEnrollmentPreferences map[string]interface{} `json:"student_class_enrollments_preferences,omitempty"`

	// ===== Snapshots dari classes (sesuai DDL) =====
	StudentClassEnrollmentClassNameSnapshot   string `json:"student_class_enrollments_class_name_snapshot"`
	StudentClassEnrollmentClassSlugSnapshot   string `json:"student_class_enrollments_class_slug_snapshot"`
	StudentClassEnrollmentStudentNameSnapshot string `json:"student_class_enrollments_student_name_snapshot"`
	StudentClassEnrollmentStudentCodeSnapshot string `json:"student_class_enrollments_student_code_snapshot"`
	StudentClassEnrollmentStudentSlugSnapshot string `json:"student_class_enrollments_student_slug_snapshot"`

	// ===== Denormalized TERM (baru) =====
	StudentClassEnrollmentTermID                   *uuid.UUID `json:"student_class_enrollments_term_id,omitempty"`
	StudentClassEnrollmentTermAcademicYearSnapshot *string    `json:"student_class_enrollments_term_academic_year_snapshot,omitempty"`
	StudentClassEnrollmentTermNameSnapshot         *string    `json:"student_class_enrollments_term_name_snapshot,omitempty"`
	StudentClassEnrollmentTermSlugSnapshot         *string    `json:"student_class_enrollments_term_slug_snapshot,omitempty"`
	StudentClassEnrollmentTermAngkatanSnapshot     *int       `json:"student_class_enrollments_term_angkatan_snapshot,omitempty"`

	// Jejak waktu (audit)
	StudentClassEnrollmentAppliedAt    time.Time  `json:"student_class_enrollments_applied_at"`
	StudentClassEnrollmentReviewedAt   *time.Time `json:"student_class_enrollments_reviewed_at"`
	StudentClassEnrollmentAcceptedAt   *time.Time `json:"student_class_enrollments_accepted_at"`
	StudentClassEnrollmentWaitlistedAt *time.Time `json:"student_class_enrollments_waitlisted_at"`
	StudentClassEnrollmentRejectedAt   *time.Time `json:"student_class_enrollments_rejected_at"`
	StudentClassEnrollmentCanceledAt   *time.Time `json:"student_class_enrollments_canceled_at"`

	StudentClassEnrollmentCreatedAt time.Time `json:"student_class_enrollments_created_at"`
	StudentClassEnrollmentUpdatedAt time.Time `json:"student_class_enrollments_updated_at"`

	// ===== Convenience (mirror snapshot) =====
	StudentClassEnrollmentStudentName string  `json:"student_class_enrollments_student_name,omitempty"` // mirror dari snapshot
	StudentClassEnrollmentUsername    *string `json:"student_class_enrollments_username,omitempty"`     // join user (jika ada)
	StudentClassEnrollmentClassName   string  `json:"student_class_enrollments_class_name,omitempty"`   // mirror dari snapshot
}

/* ======================================================
   Mappers
====================================================== */

func FromModelStudentClassEnrollment(mo *m.StudentClassEnrollmentModel) StudentClassEnrollmentResponse {
	resp := StudentClassEnrollmentResponse{
		StudentClassEnrollmentID:              mo.StudentClassEnrollmentID,
		StudentClassEnrollmentSchoolID:        mo.StudentClassEnrollmentSchoolID,
		StudentClassEnrollmentSchoolStudentID: mo.StudentClassEnrollmentSchoolStudentID,
		StudentClassEnrollmentClassID:         mo.StudentClassEnrollmentClassID,

		StudentClassEnrollmentStatus:      mo.StudentClassEnrollmentStatus,
		StudentClassEnrollmentTotalDueIDR: mo.StudentClassEnrollmentTotalDueIDR,

		// snapshots (class & student)
		StudentClassEnrollmentClassNameSnapshot:   mo.StudentClassEnrollmentClassNameSnapshot,
		StudentClassEnrollmentClassSlugSnapshot:   mo.StudentClassEnrollmentClassSlugSnapshot,
		StudentClassEnrollmentStudentNameSnapshot: mo.StudentClassEnrollmentStudentNameSnapshot,
		StudentClassEnrollmentStudentCodeSnapshot: mo.StudentClassEnrollmentStudentCodeSnapshot,
		StudentClassEnrollmentStudentSlugSnapshot: mo.StudentClassEnrollmentStudentSlugSnapshot,

		// audit
		StudentClassEnrollmentAppliedAt:    mo.StudentClassEnrollmentAppliedAt,
		StudentClassEnrollmentReviewedAt:   mo.StudentClassEnrollmentReviewedAt,
		StudentClassEnrollmentAcceptedAt:   mo.StudentClassEnrollmentAcceptedAt,
		StudentClassEnrollmentWaitlistedAt: mo.StudentClassEnrollmentWaitlistedAt,
		StudentClassEnrollmentRejectedAt:   mo.StudentClassEnrollmentRejectedAt,
		StudentClassEnrollmentCanceledAt:   mo.StudentClassEnrollmentCanceledAt,

		StudentClassEnrollmentCreatedAt: mo.StudentClassEnrollmentCreatedAt,
		StudentClassEnrollmentUpdatedAt: mo.StudentClassEnrollmentUpdatedAt,

		// mirrors
		StudentClassEnrollmentStudentName: mo.StudentClassEnrollmentStudentNameSnapshot,
		StudentClassEnrollmentClassName:   mo.StudentClassEnrollmentClassNameSnapshot,
	}

	// ===== Term denormalized (pointer fields)
	resp.StudentClassEnrollmentTermID = mo.StudentClassEnrollmentTermID
	resp.StudentClassEnrollmentTermAcademicYearSnapshot = mo.StudentClassEnrollmentTermAcademicYearSnapshot
	resp.StudentClassEnrollmentTermNameSnapshot = mo.StudentClassEnrollmentTermNameSnapshot
	resp.StudentClassEnrollmentTermSlugSnapshot = mo.StudentClassEnrollmentTermSlugSnapshot
	resp.StudentClassEnrollmentTermAngkatanSnapshot = mo.StudentClassEnrollmentTermAngkatanSnapshot

	// ===== Payment (optional)
	resp.StudentClassEnrollmentPaymentID = mo.StudentClassEnrollmentPaymentID

	// JSON → map[string]interface{}
	if b := mo.StudentClassEnrollmentPaymentSnapshot; len(b) > 0 && string(b) != "null" {
		_ = json.Unmarshal(b, &resp.StudentClassEnrollmentPaymentSnapshot)
	}
	if b := mo.StudentClassEnrollmentPreferences; len(b) > 0 && string(b) != "null" {
		_ = json.Unmarshal(b, &resp.StudentClassEnrollmentPreferences)
	}

	return resp
}

func FromModels(list []m.StudentClassEnrollmentModel) []StudentClassEnrollmentResponse {
	out := make([]StudentClassEnrollmentResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelStudentClassEnrollment(&list[i]))
	}
	return out
}

/* ======================================================
   Optional DTOs (Payment assign & Bulk status)
====================================================== */

type AssignEnrollmentPaymentRequest struct {
	StudentClassEnrollmentPaymentID       uuid.UUID              `json:"student_class_enrollments_payment_id" validate:"required"`
	StudentClassEnrollmentPaymentSnapshot map[string]interface{} `json:"student_class_enrollments_payment_snapshot"`
}

type BulkUpdateEnrollmentStatusRequest struct {
	EnrollmentIDs []uuid.UUID             `json:"enrollment_ids" validate:"required,min=1,dive,required"`
	Status        m.ClassEnrollmentStatus `json:"student_class_enrollments_status" validate:"required"`
	ReviewedAt    *time.Time              `json:"student_class_enrollments_reviewed_at"`
	AcceptedAt    *time.Time              `json:"student_class_enrollments_accepted_at"`
	WaitlistedAt  *time.Time              `json:"student_class_enrollments_waitlisted_at"`
	RejectedAt    *time.Time              `json:"student_class_enrollments_rejected_at"`
	CanceledAt    *time.Time              `json:"student_class_enrollments_canceled_at"`
}
