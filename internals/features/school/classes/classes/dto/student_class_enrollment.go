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
	// filters
	StudentID *uuid.UUID                `query:"student_id"`
	ClassID   *uuid.UUID                `query:"class_id"`
	StatusIn  []m.ClassEnrollmentStatus `query:"status_in"` // comma-separated → use custom parser in controller if needed

	AppliedFrom *time.Time `query:"applied_from"`
	AppliedTo   *time.Time `query:"applied_to"`

	OnlyAlive *bool `query:"only_alive"`

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

// Wrapper untuk kompatibel dengan helper.JsonList (optional untuk consumer)
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

	StudentClassEnrollmentPaymentID       *uuid.UUID             `json:"student_class_enrollments_payment_id"`
	StudentClassEnrollmentPaymentSnapshot map[string]interface{} `json:"student_class_enrollments_payment_snapshot"`

	StudentClassEnrollmentPreferences map[string]interface{} `json:"student_class_enrollments_preferences"`

	StudentClassEnrollmentAppliedAt    time.Time  `json:"student_class_enrollments_applied_at"`
	StudentClassEnrollmentReviewedAt   *time.Time `json:"student_class_enrollments_reviewed_at"`
	StudentClassEnrollmentAcceptedAt   *time.Time `json:"student_class_enrollments_accepted_at"`
	StudentClassEnrollmentWaitlistedAt *time.Time `json:"student_class_enrollments_waitlisted_at"`
	StudentClassEnrollmentRejectedAt   *time.Time `json:"student_class_enrollments_rejected_at"`
	StudentClassEnrollmentCanceledAt   *time.Time `json:"student_class_enrollments_canceled_at"`

	StudentClassEnrollmentCreatedAt time.Time `json:"student_class_enrollments_created_at"`
	StudentClassEnrollmentUpdatedAt time.Time `json:"student_class_enrollments_updated_at"`
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
		StudentClassEnrollmentStatus:          mo.StudentClassEnrollmentStatus,
		StudentClassEnrollmentTotalDueIDR:     mo.StudentClassEnrollmentTotalDueIDR,
		StudentClassEnrollmentPaymentID:       mo.StudentClassEnrollmentPaymentID,
		StudentClassEnrollmentAppliedAt:       mo.StudentClassEnrollmentAppliedAt,
		StudentClassEnrollmentReviewedAt:      mo.StudentClassEnrollmentReviewedAt,
		StudentClassEnrollmentAcceptedAt:      mo.StudentClassEnrollmentAcceptedAt,
		StudentClassEnrollmentWaitlistedAt:    mo.StudentClassEnrollmentWaitlistedAt,
		StudentClassEnrollmentRejectedAt:      mo.StudentClassEnrollmentRejectedAt,
		StudentClassEnrollmentCanceledAt:      mo.StudentClassEnrollmentCanceledAt,
		StudentClassEnrollmentCreatedAt:       mo.StudentClassEnrollmentCreatedAt,
		StudentClassEnrollmentUpdatedAt:       mo.StudentClassEnrollmentUpdatedAt,
	}

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
