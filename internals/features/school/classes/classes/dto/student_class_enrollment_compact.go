// file: internals/features/school/classes/class_enrollments/dto/enrollment_compact.go
package dto

import (
	"encoding/json"
	"time"

	m "schoolku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Response ringkas: fokus untuk UI list
type StudentClassEnrollmentCompactResponse struct {
	StudentClassEnrollmentID          uuid.UUID               `json:"student_class_enrollments_id"`
	StudentClassEnrollmentStatus      m.ClassEnrollmentStatus `json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR int64                   `json:"student_class_enrollments_total_due_idr"`

	// convenience (mirror snapshot)
	StudentClassEnrollmentSchoolStudentID uuid.UUID `json:"student_class_enrollments_school_student_id"`
	StudentClassEnrollmentStudentName     string    `json:"student_class_enrollments_student_name"`

	StudentClassEnrollmentClassID   uuid.UUID `json:"student_class_enrollments_class_id"`
	StudentClassEnrollmentClassName string    `json:"student_class_enrollments_class_name"`

	// ===== Term (denormalized, optional) =====
	StudentClassEnrollmentTermID                   *uuid.UUID `json:"student_class_enrollments_term_id,omitempty"`
	StudentClassEnrollmentTermNameSnapshot         *string    `json:"student_class_enrollments_term_name_snapshot,omitempty"`
	StudentClassEnrollmentTermAcademicYearSnapshot *string    `json:"student_class_enrollments_term_academic_year_snapshot,omitempty"`
	StudentClassEnrollmentTermAngkatanSnapshot     *int       `json:"student_class_enrollments_term_angkatan_snapshot,omitempty"`

	// opsional tapi sering dipakai di UI pembayaran (diambil dari payment_snapshot)
	PaymentStatus      *string `json:"payment_status,omitempty"`
	PaymentCheckoutURL *string `json:"payment_checkout_url,omitempty"`

	// jejak penting
	AppliedAt time.Time `json:"student_class_enrollments_applied_at"`
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

			// IDs & snapshots yang tersedia di model
			StudentClassEnrollmentSchoolStudentID: r.StudentClassEnrollmentsSchoolStudentID,
			StudentClassEnrollmentStudentName:     r.StudentClassEnrollmentsStudentNameSnapshot,

			StudentClassEnrollmentClassID:   r.StudentClassEnrollmentsClassID,
			StudentClassEnrollmentClassName: r.StudentClassEnrollmentsClassNameSnapshot,

			// term (pointer fields dari model)
			StudentClassEnrollmentTermID:                   r.StudentClassEnrollmentsTermID,
			StudentClassEnrollmentTermNameSnapshot:         r.StudentClassEnrollmentsTermNameSnapshot,
			StudentClassEnrollmentTermAcademicYearSnapshot: r.StudentClassEnrollmentsTermAcademicYearSnapshot,
			StudentClassEnrollmentTermAngkatanSnapshot:     r.StudentClassEnrollmentsTermAngkatanSnapshot,

			AppliedAt: r.StudentClassEnrollmentsAppliedAt,
		}

		// derive fields dari payment snapshot
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
// Catatan: ini tidak mengisi kolom lain yang tidak ada di compact DTO.
func (r StudentClassEnrollmentCompactResponse) ToModelCompact() m.StudentClassEnrollmentModel {
	return m.StudentClassEnrollmentModel{
		StudentClassEnrollmentsID:          r.StudentClassEnrollmentID,
		StudentClassEnrollmentsStatus:      r.StudentClassEnrollmentStatus,
		StudentClassEnrollmentsTotalDueIDR: r.StudentClassEnrollmentTotalDueIDR,

		// IDs & snapshots
		StudentClassEnrollmentsSchoolStudentID:          r.StudentClassEnrollmentSchoolStudentID,
		StudentClassEnrollmentsStudentNameSnapshot:      r.StudentClassEnrollmentStudentName,
		StudentClassEnrollmentsClassID:                  r.StudentClassEnrollmentClassID,
		StudentClassEnrollmentsClassNameSnapshot:        r.StudentClassEnrollmentClassName,
		StudentClassEnrollmentsTermID:                   r.StudentClassEnrollmentTermID,
		StudentClassEnrollmentsTermNameSnapshot:         r.StudentClassEnrollmentTermNameSnapshot,
		StudentClassEnrollmentsTermAcademicYearSnapshot: r.StudentClassEnrollmentTermAcademicYearSnapshot,
		StudentClassEnrollmentsTermAngkatanSnapshot:     r.StudentClassEnrollmentTermAngkatanSnapshot,

		// payment snapshot (optional)
		StudentClassEnrollmentsPaymentSnapshot: datatypes.JSON(makePaymentSnapshot(r.PaymentStatus, r.PaymentCheckoutURL)),

		// jejak waktu
		StudentClassEnrollmentsAppliedAt: r.AppliedAt,
	}
}

// ToModelsCompact untuk batch convert slice DTO → slice Model.
func ToModelsCompact(in []StudentClassEnrollmentCompactResponse) []m.StudentClassEnrollmentModel {
	out := make([]m.StudentClassEnrollmentModel, 0, len(in))
	for _, r := range in {
		out = append(out, r.ToModelCompact())
	}
	return out
}
