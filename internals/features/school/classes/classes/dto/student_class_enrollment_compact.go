// file: internals/features/school/classes/class_enrollments/dto/enrollment_compact.go
package dto

import (
	"encoding/json"
	"time"

	m "schoolku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

// Response ringkas: fokus untuk UI list
type StudentClassEnrollmentCompactResponse struct {
	StudentClassEnrollmentID          uuid.UUID               `json:"student_class_enrollments_id"`
	StudentClassEnrollmentStatus      m.ClassEnrollmentStatus `json:"student_class_enrollments_status"`
	StudentClassEnrollmentTotalDueIDR int64                   `json:"student_class_enrollments_total_due_idr"`

	// convenience
	StudentClassEnrollmentStudentName string `json:"student_class_enrollments_student_name"`
	StudentClassEnrollmentClassName   string `json:"student_class_enrollments_class_name"`

	// opsional tapi sering dipakai di UI pembayaran
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
			StudentClassEnrollmentID:          r.StudentClassEnrollmentID,
			StudentClassEnrollmentStatus:      r.StudentClassEnrollmentStatus,
			StudentClassEnrollmentTotalDueIDR: r.StudentClassEnrollmentTotalDueIDR,
			StudentClassEnrollmentStudentName: r.StudentClassEnrollmentStudentNameSnapshot,
			StudentClassEnrollmentClassName:   r.StudentClassEnrollmentClassNameSnapshot,
			AppliedAt:                         r.StudentClassEnrollmentAppliedAt,
		}
		item.PaymentStatus = strFromJSON(r.StudentClassEnrollmentPaymentSnapshot, "payment_status")
		item.PaymentCheckoutURL = strFromJSON(r.StudentClassEnrollmentPaymentSnapshot, "payment_checkout_url")

		out = append(out, item)
	}
	return out
}
