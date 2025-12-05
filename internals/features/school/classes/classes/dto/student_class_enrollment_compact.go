// file: internals/features/school/classes/class_enrollments/dto/enrollment_compact.go
package dto

import (
	"encoding/json"
	"time"

	m "madinahsalam_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

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

// ==================== NEW: ToModel ====================

// makePaymentSnapshot membentuk JSON ([]byte) hanya dengan key yang tersedia.
func makePaymentSnapshot(status, checkoutURL *string) []byte {
	if status == nil && checkoutURL == nil {
		return nil
	}
	payload := make(map[string]*string, 2)
	if status != nil {
		payload["payment_status"] = status
	}
	if checkoutURL != nil {
		payload["payment_checkout_url"] = checkoutURL
	}
	b, _ := json.Marshal(payload)
	return b
}

// ToModelCompact mengisi field yang tersedia di compact DTO ke dalam model.
// Catatan: ini tidak mengisi kolom lain yang
