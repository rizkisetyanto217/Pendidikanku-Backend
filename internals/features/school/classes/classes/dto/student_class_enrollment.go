// file: internals/features/school/classes/class_enrollments/dto/student_class_enrollments_dto.go
package dto

import (
	"encoding/json"
	"time"

	csDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	m "madinahsalam_backend/internals/features/school/classes/classes/model"
	h "madinahsalam_backend/internals/helpers"

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
	StatusIn  []m.ClassEnrollmentStatus `query:"status_in"` // comma-separated → parser di controller

	AppliedFrom *time.Time `query:"applied_from"`
	AppliedTo   *time.Time `query:"applied_to"`

	OnlyAlive *bool `query:"only_alive"`

	// TERM filters (denormalized kolom)
	TermID       *uuid.UUID `query:"term_id"`
	AcademicYear string     `query:"academic_year"` // ex: "2026/2027"
	Angkatan     *int       `query:"angkatan"`

	// optional: kalau mau mapping ke entitas lain
	AcademicTermID *uuid.UUID `query:"academic_term_id"`

	// simple search (snapshots: student / class / term)
	Q string `query:"q"`

	// paging & sort
	Limit   int    `query:"limit"`
	Offset  int    `query:"offset"`
	OrderBy string `query:"order_by"` // created_at | applied_at | updated_at
	Sort    string `query:"sort"`     // asc | desc
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

	// ⬇️ Class ID utama (dipakai untuk JSON response)
	StudentClassEnrollmentClassID uuid.UUID `json:"student_class_enrollments_class_id"`

	// ⬇️ Convenience field internal (dipakai di controller/payment)
	// nggak keluar di JSON
	ClassID uuid.UUID `json:"-"`

	// include=class_sections → akan diisi di enrichEnrollmentClassSections
	ClassSections []csDTO.ClassSectionCompact `json:"class_sections,omitempty"`

	StudentClassEnrollmentStatus      m.ClassEnrollmentStatus `json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR int64                   `json:"student_class_enrollments_total_due_idr"`

	StudentClassEnrollmentPaymentID       *uuid.UUID             `json:"student_class_enrollments_payment_id,omitempty"`
	StudentClassEnrollmentPaymentSnapshot map[string]interface{} `json:"student_class_enrollments_payment_snapshot,omitempty"`

	StudentClassEnrollmentPreferences map[string]interface{} `json:"student_class_enrollments_preferences,omitempty"`

	// ===== Snapshots dari classes (sesuai DDL & model) =====
	StudentClassEnrollmentClassNameSnapshot string  `json:"student_class_enrollments_class_name_snapshot"`
	StudentClassEnrollmentClassSlugSnapshot *string `json:"student_class_enrollments_class_slug_snapshot,omitempty"`

	// ===== SNAPSHOT dari school_students / user_profile =====
	StudentClassEnrollmentUserProfileNameSnapshot              string  `json:"student_class_enrollments_user_profile_name_snapshot"`
	StudentClassEnrollmentUserProfileAvatarURLSnapshot         *string `json:"student_class_enrollments_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassEnrollmentUserProfileWhatsappURLSnapshot       *string `json:"student_class_enrollments_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassEnrollmentUserProfileParentNameSnapshot        *string `json:"student_class_enrollments_user_profile_parent_name_snapshot,omitempty"`
	StudentClassEnrollmentUserProfileParentWhatsappURLSnapshot *string `json:"student_class_enrollments_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	StudentClassEnrollmentUserProfileGenderSnapshot            *string `json:"student_class_enrollments_user_profile_gender_snapshot,omitempty"`

	StudentClassEnrollmentStudentCodeSnapshot *string `json:"student_class_enrollments_student_code_snapshot,omitempty"`
	StudentClassEnrollmentStudentSlugSnapshot *string `json:"student_class_enrollments_student_slug_snapshot,omitempty"`

	// ===== Denormalized TERM =====
	StudentClassEnrollmentTermID                   *uuid.UUID `json:"student_class_enrollments_term_id,omitempty"`
	StudentClassEnrollmentTermAcademicYearSnapshot *string    `json:"student_class_enrollments_term_academic_year_snapshot,omitempty"`
	StudentClassEnrollmentTermNameSnapshot         *string    `json:"student_class_enrollments_term_name_snapshot,omitempty"`
	StudentClassEnrollmentTermSlugSnapshot         *string    `json:"student_class_enrollments_term_slug_snapshot,omitempty"`
	StudentClassEnrollmentTermAngkatanSnapshot     *int       `json:"student_class_enrollments_term_angkatan_snapshot,omitempty"`

	// ===== CLASS SECTION (baru, opsional) =====
	StudentClassEnrollmentClassSectionID           *uuid.UUID `json:"student_class_enrollments_class_section_id"`
	StudentClassEnrollmentClassSectionNameSnapshot *string    `json:"student_class_enrollments_class_section_name_snapshot"`
	StudentClassEnrollmentClassSectionSlugSnapshot *string    `json:"student_class_enrollments_class_section_slug_snapshot"`

	// Jejak waktu (audit)
	StudentClassEnrollmentAppliedAt    time.Time  `json:"student_class_enrollments_applied_at"`
	StudentClassEnrollmentReviewedAt   *time.Time `json:"student_class_enrollments_reviewed_at"`
	StudentClassEnrollmentAcceptedAt   *time.Time `json:"student_class_enrollments_accepted_at"`
	StudentClassEnrollmentWaitlistedAt *time.Time `json:"student_class_enrollments_waitlisted_at"`
	StudentClassEnrollmentRejectedAt   *time.Time `json:"student_class_enrollments_rejected_at"`
	StudentClassEnrollmentCanceledAt   *time.Time `json:"student_class_enrollments_canceled_at"`

	StudentClassEnrollmentCreatedAt time.Time `json:"student_class_enrollments_created_at"`
	StudentClassEnrollmentUpdatedAt time.Time `json:"student_class_enrollments_updated_at"`

	// ===== Convenience (mirror snapshot, bukan kolom DB) =====
	// pakai nama lama biar frontend nggak rusak,
	// tapi isi dari user_profile_name_snapshot
	StudentClassEnrollmentStudentName string  `json:"student_class_enrollments_student_name,omitempty"`
	StudentClassEnrollmentUsername    *string `json:"student_class_enrollments_username,omitempty"`
	StudentClassEnrollmentClassName   string  `json:"student_class_enrollments_class_name,omitempty"`
}

/*
	======================================================
	  Mappers
======================================================
*/

func FromModelStudentClassEnrollment(mo *m.StudentClassEnrollmentModel) StudentClassEnrollmentResponse {
	resp := StudentClassEnrollmentResponse{
		StudentClassEnrollmentID:              mo.StudentClassEnrollmentsID,
		StudentClassEnrollmentSchoolID:        mo.StudentClassEnrollmentsSchoolID,
		StudentClassEnrollmentSchoolStudentID: mo.StudentClassEnrollmentsSchoolStudentID,

		// ⬇️ isi dua-duanya: field JSON + convenience
		StudentClassEnrollmentClassID: mo.StudentClassEnrollmentsClassID,
		ClassID:                       mo.StudentClassEnrollmentsClassID,

		StudentClassEnrollmentStatus:      mo.StudentClassEnrollmentsStatus,
		StudentClassEnrollmentTotalDueIDR: mo.StudentClassEnrollmentsTotalDueIDR,

		// snapshots (class)
		StudentClassEnrollmentClassNameSnapshot: mo.StudentClassEnrollmentsClassNameSnapshot,
		StudentClassEnrollmentClassSlugSnapshot: mo.StudentClassEnrollmentsClassSlugSnapshot,

		// audit
		StudentClassEnrollmentAppliedAt:    mo.StudentClassEnrollmentsAppliedAt,
		StudentClassEnrollmentReviewedAt:   mo.StudentClassEnrollmentsReviewedAt,
		StudentClassEnrollmentAcceptedAt:   mo.StudentClassEnrollmentsAcceptedAt,
		StudentClassEnrollmentWaitlistedAt: mo.StudentClassEnrollmentsWaitlistedAt,
		StudentClassEnrollmentRejectedAt:   mo.StudentClassEnrollmentsRejectedAt,
		StudentClassEnrollmentCanceledAt:   mo.StudentClassEnrollmentsCanceledAt,

		StudentClassEnrollmentCreatedAt: mo.StudentClassEnrollmentsCreatedAt,
		StudentClassEnrollmentUpdatedAt: mo.StudentClassEnrollmentsUpdatedAt,

		// mirrors (convenience)
		StudentClassEnrollmentStudentName: mo.StudentClassEnrollmentsUserProfileNameSnapshot,
		StudentClassEnrollmentClassName:   mo.StudentClassEnrollmentsClassNameSnapshot,
	}

	// SNAPSHOT user_profile / student
	resp.StudentClassEnrollmentUserProfileNameSnapshot = mo.StudentClassEnrollmentsUserProfileNameSnapshot
	resp.StudentClassEnrollmentUserProfileAvatarURLSnapshot = mo.StudentClassEnrollmentsUserProfileAvatarURLSnapshot
	resp.StudentClassEnrollmentUserProfileWhatsappURLSnapshot = mo.StudentClassEnrollmentsUserProfileWhatsappURLSnapshot
	resp.StudentClassEnrollmentUserProfileParentNameSnapshot = mo.StudentClassEnrollmentsUserProfileParentNameSnapshot
	resp.StudentClassEnrollmentUserProfileParentWhatsappURLSnapshot = mo.StudentClassEnrollmentsUserProfileParentWhatsappURLSnapshot
	resp.StudentClassEnrollmentUserProfileGenderSnapshot = mo.StudentClassEnrollmentsUserProfileGenderSnapshot
	resp.StudentClassEnrollmentStudentCodeSnapshot = mo.StudentClassEnrollmentsStudentCodeSnapshot
	resp.StudentClassEnrollmentStudentSlugSnapshot = mo.StudentClassEnrollmentsStudentSlugSnapshot

	// Term
	resp.StudentClassEnrollmentTermID = mo.StudentClassEnrollmentsTermID
	resp.StudentClassEnrollmentTermAcademicYearSnapshot = mo.StudentClassEnrollmentsTermAcademicYearSnapshot
	resp.StudentClassEnrollmentTermNameSnapshot = mo.StudentClassEnrollmentsTermNameSnapshot
	resp.StudentClassEnrollmentTermSlugSnapshot = mo.StudentClassEnrollmentsTermSlugSnapshot
	resp.StudentClassEnrollmentTermAngkatanSnapshot = mo.StudentClassEnrollmentsTermAngkatanSnapshot

	// Class section
	resp.StudentClassEnrollmentClassSectionID = mo.StudentClassEnrollmentsClassSectionID
	resp.StudentClassEnrollmentClassSectionNameSnapshot = mo.StudentClassEnrollmentsClassSectionNameSnapshot
	resp.StudentClassEnrollmentClassSectionSlugSnapshot = mo.StudentClassEnrollmentsClassSectionSlugSnapshot

	// Payment
	resp.StudentClassEnrollmentPaymentID = mo.StudentClassEnrollmentsPaymentID

	if b := mo.StudentClassEnrollmentsPaymentSnapshot; len(b) > 0 && string(b) != "null" {
		_ = json.Unmarshal(b, &resp.StudentClassEnrollmentPaymentSnapshot)
	}
	if b := mo.StudentClassEnrollmentsPreferences; len(b) > 0 && string(b) != "null" {
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

// Optional DTOs (Payment assign & Bulk status)
// ...

type JoinClassSectionRequest struct {
	ClassSectionID uuid.UUID `json:"class_section_id" validate:"required"`
}
