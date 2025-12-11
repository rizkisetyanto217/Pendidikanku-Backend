// file: internals/features/school/classes/class_enrollments/dto/student_class_enrollments_dto.go
package dto

import (
	"encoding/json"
	"time"

	csDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	m "madinahsalam_backend/internals/features/school/classes/classes/model"
	h "madinahsalam_backend/internals/helpers"
	dbtime "madinahsalam_backend/internals/helpers/dbtime"

	"github.com/gofiber/fiber/v2"
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

	// simple search (snapshots/cache: student / class / term)
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
	ClassSections []csDTO.ClassSectionCompactResponse `json:"class_sections,omitempty"`

	StudentClassEnrollmentStatus      m.ClassEnrollmentStatus `json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR int64                   `json:"student_class_enrollments_total_due_idr"`

	StudentClassEnrollmentPaymentID       *uuid.UUID             `json:"student_class_enrollments_payment_id,omitempty"`
	StudentClassEnrollmentPaymentSnapshot map[string]interface{} `json:"student_class_enrollments_payment_snapshot,omitempty"`

	StudentClassEnrollmentPreferences map[string]interface{} `json:"student_class_enrollments_preferences,omitempty"`

	// ===== Cache dari classes (sesuai model) =====
	StudentClassEnrollmentClassNameCache string  `json:"student_class_enrollments_class_name_cache"`
	StudentClassEnrollmentClassSlugCache *string `json:"student_class_enrollments_class_slug_cache,omitempty"`

	// ===== CACHE dari school_students / user_profile =====
	StudentClassEnrollmentUserProfileNameCache              string  `json:"student_class_enrollments_user_profile_name_cache"`
	StudentClassEnrollmentUserProfileAvatarURLCache         *string `json:"student_class_enrollments_user_profile_avatar_url_cache,omitempty"`
	StudentClassEnrollmentUserProfileWhatsappURLCache       *string `json:"student_class_enrollments_user_profile_whatsapp_url_cache,omitempty"`
	StudentClassEnrollmentUserProfileParentNameCache        *string `json:"student_class_enrollments_user_profile_parent_name_cache,omitempty"`
	StudentClassEnrollmentUserProfileParentWhatsappURLCache *string `json:"student_class_enrollments_user_profile_parent_whatsapp_url_cache,omitempty"`
	StudentClassEnrollmentUserProfileGenderCache            *string `json:"student_class_enrollments_user_profile_gender_cache,omitempty"`

	StudentClassEnrollmentStudentCodeCache *string `json:"student_class_enrollments_student_code_cache,omitempty"`
	StudentClassEnrollmentStudentSlugCache *string `json:"student_class_enrollments_student_slug_cache,omitempty"`

	// ===== Denormalized TERM (cache) =====
	StudentClassEnrollmentTermID                *uuid.UUID `json:"student_class_enrollments_term_id,omitempty"`
	StudentClassEnrollmentTermAcademicYearCache *string    `json:"student_class_enrollments_term_academic_year_cache,omitempty"`
	StudentClassEnrollmentTermNameCache         *string    `json:"student_class_enrollments_term_name_cache,omitempty"`
	StudentClassEnrollmentTermSlugCache         *string    `json:"student_class_enrollments_term_slug_cache,omitempty"`
	StudentClassEnrollmentTermAngkatanCache     *int       `json:"student_class_enrollments_term_angkatan_cache,omitempty"`

	// ===== CLASS SECTION (baru, opsional) =====
	StudentClassEnrollmentClassSectionID        *uuid.UUID `json:"student_class_enrollments_class_section_id"`
	StudentClassEnrollmentClassSectionNameCache *string    `json:"student_class_enrollments_class_section_name_cache"`
	StudentClassEnrollmentClassSectionSlugCache *string    `json:"student_class_enrollments_class_section_slug_cache"`

	// 🆕 Convenience field internal untuk section yang sedang diikuti siswa
	// dipakai di enrichEnrollmentClassSections (scope & is_student)
	ClassSectionID *uuid.UUID `json:"-"`

	// Jejak waktu (audit)
	StudentClassEnrollmentAppliedAt    time.Time  `json:"student_class_enrollments_applied_at"`
	StudentClassEnrollmentReviewedAt   *time.Time `json:"student_class_enrollments_reviewed_at"`
	StudentClassEnrollmentAcceptedAt   *time.Time `json:"student_class_enrollments_accepted_at"`
	StudentClassEnrollmentWaitlistedAt *time.Time `json:"student_class_enrollments_waitlisted_at"`
	StudentClassEnrollmentRejectedAt   *time.Time `json:"student_class_enrollments_rejected_at"`
	StudentClassEnrollmentCanceledAt   *time.Time `json:"student_class_enrollments_canceled_at"`

	StudentClassEnrollmentCreatedAt time.Time `json:"student_class_enrollments_created_at"`
	StudentClassEnrollmentUpdatedAt time.Time `json:"student_class_enrollments_updated_at"`

	// ===== Convenience (mirror cache, bukan kolom DB) =====
	// pakai nama lebih pendek buat konsumsi frontend
	StudentClassEnrollmentStudentName string  `json:"student_class_enrollments_student_name,omitempty"`
	StudentClassEnrollmentUsername    *string `json:"student_class_enrollments_username,omitempty"`
	StudentClassEnrollmentClassName   string  `json:"student_class_enrollments_class_name,omitempty"`
}

/*
	======================================================
	  Mappers
======================================================
*/

// Lama: tanpa konversi timezone (biarin tetap ada kalau masih kepakai)
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

		// class cache
		StudentClassEnrollmentClassNameCache: mo.StudentClassEnrollmentsClassNameCache,
		StudentClassEnrollmentClassSlugCache: mo.StudentClassEnrollmentsClassSlugCache,

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
		StudentClassEnrollmentStudentName: mo.StudentClassEnrollmentsUserProfileNameCache,
		StudentClassEnrollmentClassName:   mo.StudentClassEnrollmentsClassNameCache,
	}

	// CACHE user_profile / student
	resp.StudentClassEnrollmentUserProfileNameCache = mo.StudentClassEnrollmentsUserProfileNameCache
	resp.StudentClassEnrollmentUserProfileAvatarURLCache = mo.StudentClassEnrollmentsUserProfileAvatarURLCache
	resp.StudentClassEnrollmentUserProfileWhatsappURLCache = mo.StudentClassEnrollmentsUserProfileWhatsappURLCache
	resp.StudentClassEnrollmentUserProfileParentNameCache = mo.StudentClassEnrollmentsUserProfileParentNameCache
	resp.StudentClassEnrollmentUserProfileParentWhatsappURLCache = mo.StudentClassEnrollmentsUserProfileParentWhatsappURLCache
	resp.StudentClassEnrollmentUserProfileGenderCache = mo.StudentClassEnrollmentsUserProfileGenderCache
	resp.StudentClassEnrollmentStudentCodeCache = mo.StudentClassEnrollmentsStudentCodeCache
	resp.StudentClassEnrollmentStudentSlugCache = mo.StudentClassEnrollmentsStudentSlugCache

	// Term cache
	resp.StudentClassEnrollmentTermID = mo.StudentClassEnrollmentsTermID
	resp.StudentClassEnrollmentTermAcademicYearCache = mo.StudentClassEnrollmentsTermAcademicYearCache
	resp.StudentClassEnrollmentTermNameCache = mo.StudentClassEnrollmentsTermNameCache
	resp.StudentClassEnrollmentTermSlugCache = mo.StudentClassEnrollmentsTermSlugCache
	resp.StudentClassEnrollmentTermAngkatanCache = mo.StudentClassEnrollmentsTermAngkatanCache

	// Class section cache
	resp.StudentClassEnrollmentClassSectionID = mo.StudentClassEnrollmentsClassSectionID
	resp.StudentClassEnrollmentClassSectionNameCache = mo.StudentClassEnrollmentsClassSectionNameCache
	resp.StudentClassEnrollmentClassSectionSlugCache = mo.StudentClassEnrollmentsClassSectionSlugCache

	// 🆕 convenience: mirror ke ClassSectionID (internal)
	resp.ClassSectionID = mo.StudentClassEnrollmentsClassSectionID

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

// Baru: mapping + konversi timezone ke timezone sekolah
func FromModelStudentClassEnrollmentWithContext(c *fiber.Ctx, mo *m.StudentClassEnrollmentModel) StudentClassEnrollmentResponse {
	if mo == nil {
		return StudentClassEnrollmentResponse{}
	}

	resp := FromModelStudentClassEnrollment(mo)

	// convert semua time.Time / *time.Time
	resp.StudentClassEnrollmentAppliedAt = dbtime.ToSchoolTime(c, mo.StudentClassEnrollmentsAppliedAt)
	resp.StudentClassEnrollmentCreatedAt = dbtime.ToSchoolTime(c, mo.StudentClassEnrollmentsCreatedAt)
	resp.StudentClassEnrollmentUpdatedAt = dbtime.ToSchoolTime(c, mo.StudentClassEnrollmentsUpdatedAt)

	resp.StudentClassEnrollmentReviewedAt = dbtime.ToSchoolTimePtr(c, mo.StudentClassEnrollmentsReviewedAt)
	resp.StudentClassEnrollmentAcceptedAt = dbtime.ToSchoolTimePtr(c, mo.StudentClassEnrollmentsAcceptedAt)
	resp.StudentClassEnrollmentWaitlistedAt = dbtime.ToSchoolTimePtr(c, mo.StudentClassEnrollmentsWaitlistedAt)
	resp.StudentClassEnrollmentRejectedAt = dbtime.ToSchoolTimePtr(c, mo.StudentClassEnrollmentsRejectedAt)
	resp.StudentClassEnrollmentCanceledAt = dbtime.ToSchoolTimePtr(c, mo.StudentClassEnrollmentsCanceledAt)

	return resp
}

func FromModels(list []m.StudentClassEnrollmentModel) []StudentClassEnrollmentResponse {
	out := make([]StudentClassEnrollmentResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelStudentClassEnrollment(&list[i]))
	}
	return out
}

func FromModelsWithContext(c *fiber.Ctx, list []m.StudentClassEnrollmentModel) []StudentClassEnrollmentResponse {
	out := make([]StudentClassEnrollmentResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelStudentClassEnrollmentWithContext(c, &list[i]))
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

// Response ringkas: fokus untuk UI list
type StudentClassEnrollmentCompactResponse struct {
	StudentClassEnrollmentID          uuid.UUID               `json:"student_class_enrollments_id"`
	StudentClassEnrollmentStatus      m.ClassEnrollmentStatus `json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR int64                   `json:"student_class_enrollments_total_due_idr"`

	// convenience (mirror cache)
	StudentClassEnrollmentSchoolStudentID uuid.UUID `json:"student_class_enrollments_school_student_id"`

	// ====== MURID (cache dari user_profile / school_students) ======
	StudentClassEnrollmentStudentName        string  `json:"student_class_enrollments_student_name"`
	StudentClassEnrollmentStudentAvatarURL   *string `json:"student_class_enrollments_student_avatar_url,omitempty"`
	StudentClassEnrollmentStudentWhatsappURL *string `json:"student_class_enrollments_student_whatsapp_url,omitempty"`
	StudentClassEnrollmentParentName         *string `json:"student_class_enrollments_parent_name,omitempty"`
	StudentClassEnrollmentParentWhatsappURL  *string `json:"student_class_enrollments_parent_whatsapp_url,omitempty"`

	// 👇 TANPA omitempty supaya selalu muncul di JSON meski nil
	StudentClassEnrollmentStudentGender *string `json:"student_class_enrollments_student_gender"`
	StudentClassEnrollmentStudentCode   *string `json:"student_class_enrollments_student_code"`

	StudentClassEnrollmentStudentSlug *string `json:"student_class_enrollments_student_slug,omitempty"`

	// ====== CLASS (cache) ======
	StudentClassEnrollmentClassID   uuid.UUID `json:"student_class_enrollments_class_id"`
	StudentClassEnrollmentClassName string    `json:"student_class_enrollments_class_name"`

	// ====== CLASS SECTION (opsional, cache) ======
	StudentClassEnrollmentClassSectionID           *uuid.UUID `json:"student_class_enrollments_class_section_id,omitempty"`
	StudentClassEnrollmentClassSectionNameSnapshot *string    `json:"student_class_enrollments_class_section_name_snapshot,omitempty"`
	StudentClassEnrollmentClassSectionSlugSnapshot *string    `json:"student_class_enrollments_class_section_slug_snapshot,omitempty"`

	// ===== Term (denormalized, optional; cache) =====
	StudentClassEnrollmentTermID                   *uuid.UUID `json:"student_class_enrollments_term_id,omitempty"`
	StudentClassEnrollmentTermNameSnapshot         *string    `json:"student_class_enrollments_term_name_snapshot,omitempty"`
	StudentClassEnrollmentTermAcademicYearSnapshot *string    `json:"student_class_enrollments_term_academic_year_snapshot,omitempty"`
	StudentClassEnrollmentTermAngkatanSnapshot     *int       `json:"student_class_enrollments_term_angkatan_snapshot,omitempty"`

	// opsional tapi sering dipakai di UI pembayaran (diambil dari payment_snapshot)
	PaymentStatus      *string `json:"payment_status,omitempty"`
	PaymentCheckoutURL *string `json:"payment_checkout_url,omitempty"`

	// jejak penting
	AppliedAt time.Time `json:"student_class_enrollments_applied_at"`

	// 🆕 Convenience internal (tidak di-JSON-kan)
	// dipakai kalau suatu saat kita mau include/nested class_sections di compact view
	ClassID        uuid.UUID  `json:"-"`
	ClassSectionID *uuid.UUID `json:"-"`
	// 🆕 flag apakah siswa yang sedang dilihat terdaftar di section ini
	IsStudent bool `json:"is_student,omitempty"`
}

func strFromJSON(b []byte, key string) *string {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	var mm map[string]interface{}
	if err := json.Unmarshal(b, &mm); err != nil {
		return nil
	}
	if v, ok := mm[key]; ok && v != nil {
		if s, ok := v.(string); ok && s != "" {
			return &s
		}
	}
	return nil
}

// Lama: tanpa timezone convert
func FromModelsCompact(src []m.StudentClassEnrollmentModel) []StudentClassEnrollmentCompactResponse {
	out := make([]StudentClassEnrollmentCompactResponse, 0, len(src))
	for _, r := range src {
		item := StudentClassEnrollmentCompactResponse{
			StudentClassEnrollmentID:          r.StudentClassEnrollmentsID,
			StudentClassEnrollmentStatus:      r.StudentClassEnrollmentsStatus,
			StudentClassEnrollmentTotalDueIDR: r.StudentClassEnrollmentsTotalDueIDR,

			// IDs
			StudentClassEnrollmentSchoolStudentID: r.StudentClassEnrollmentsSchoolStudentID,

			// ===== murid (cache) =====
			StudentClassEnrollmentStudentName:        r.StudentClassEnrollmentsUserProfileNameCache,
			StudentClassEnrollmentStudentAvatarURL:   r.StudentClassEnrollmentsUserProfileAvatarURLCache,
			StudentClassEnrollmentStudentWhatsappURL: r.StudentClassEnrollmentsUserProfileWhatsappURLCache,
			StudentClassEnrollmentParentName:         r.StudentClassEnrollmentsUserProfileParentNameCache,
			StudentClassEnrollmentParentWhatsappURL:  r.StudentClassEnrollmentsUserProfileParentWhatsappURLCache,
			StudentClassEnrollmentStudentGender:      r.StudentClassEnrollmentsUserProfileGenderCache,
			StudentClassEnrollmentStudentCode:        r.StudentClassEnrollmentsStudentCodeCache,
			StudentClassEnrollmentStudentSlug:        r.StudentClassEnrollmentsStudentSlugCache,

			// class (cache)
			StudentClassEnrollmentClassID:   r.StudentClassEnrollmentsClassID,
			StudentClassEnrollmentClassName: r.StudentClassEnrollmentsClassNameCache,

			// class section (opsional, cache)
			StudentClassEnrollmentClassSectionID:           r.StudentClassEnrollmentsClassSectionID,
			StudentClassEnrollmentClassSectionNameSnapshot: r.StudentClassEnrollmentsClassSectionNameCache,
			StudentClassEnrollmentClassSectionSlugSnapshot: r.StudentClassEnrollmentsClassSectionSlugCache,

			// term (cache)
			StudentClassEnrollmentTermID:                   r.StudentClassEnrollmentsTermID,
			StudentClassEnrollmentTermNameSnapshot:         r.StudentClassEnrollmentsTermNameCache,
			StudentClassEnrollmentTermAcademicYearSnapshot: r.StudentClassEnrollmentsTermAcademicYearCache,
			StudentClassEnrollmentTermAngkatanSnapshot:     r.StudentClassEnrollmentsTermAngkatanCache,

			AppliedAt: r.StudentClassEnrollmentsAppliedAt,

			// 🆕 internal convenience
			ClassID:        r.StudentClassEnrollmentsClassID,
			ClassSectionID: r.StudentClassEnrollmentsClassSectionID,
		}

		// derive fields dari payment snapshot (jsonb)
		item.PaymentStatus = strFromJSON([]byte(r.StudentClassEnrollmentsPaymentSnapshot), "payment_status")
		item.PaymentCheckoutURL = strFromJSON([]byte(r.StudentClassEnrollmentsPaymentSnapshot), "payment_checkout_url")

		out = append(out, item)
	}
	return out
}

// Baru: compact + konversi AppliedAt ke timezone sekolah
func FromModelsCompactWithContext(c *fiber.Ctx, src []m.StudentClassEnrollmentModel) []StudentClassEnrollmentCompactResponse {
	out := FromModelsCompact(src)
	for i := range out {
		out[i].AppliedAt = dbtime.ToSchoolTime(c, out[i].AppliedAt)
	}
	return out
}
