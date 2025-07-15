package dto

import (
	"encoding/json"
	"masjidku_backend/internals/features/masjids/lectures/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Struct Teacher untuk frontend & penyimpanan JSON
type Teacher struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ========================
// REQUEST
// ========================
type LectureRequest struct {
	LectureTitle                 string     `json:"lecture_title"`
	LectureDescription           string     `json:"lecture_description"`
	LectureMasjidID              uuid.UUID  `json:"lecture_masjid_id"`
	TotalLectureSessions         *int       `json:"total_lecture_sessions"`
	LectureIsRecurring           bool       `json:"lecture_is_recurring"`
	LectureRecurrenceInterval    *int       `json:"lecture_recurrence_interval"`
	LectureImageURL              *string    `json:"lecture_image_url"`
	LectureTeachers              []Teacher  `json:"lecture_teachers"`
	LectureIsRegistrationRequired bool      `json:"lecture_is_registration_required"`
	LectureIsPaid                bool       `json:"lecture_is_paid"`
	LecturePrice                 *int       `json:"lecture_price"`
	LecturePaymentDeadline       *time.Time `json:"lecture_payment_deadline"`
	LectureCapacity              *int       `json:"lecture_capacity"`
	LectureIsPublic              bool       `json:"lecture_is_public"`
	LectureIsActive              bool       `json:"lecture_is_active"`

}

// ========================
// RESPONSE
// ========================
type LectureResponse struct {
	LectureID                   uuid.UUID  `json:"lecture_id"`
	LectureTitle                string     `json:"lecture_title"`
	LectureDescription          string     `json:"lecture_description"`
	LectureMasjidID             uuid.UUID  `json:"lecture_masjid_id"`
	TotalLectureSessions        *int       `json:"total_lecture_sessions"`
	LectureIsRecurring          bool       `json:"lecture_is_recurring"`
	LectureRecurrenceInterval   *int       `json:"lecture_recurrence_interval"`
	LectureImageURL             *string    `json:"lecture_image_url"`
	LectureTeachers             []Teacher  `json:"lecture_teachers"`
	LectureIsRegistrationRequired bool     `json:"lecture_is_registration_required"`
	LectureIsPaid               bool       `json:"lecture_is_paid"`
	LecturePrice                *int       `json:"lecture_price"`
	LecturePaymentDeadline      *time.Time `json:"lecture_payment_deadline"`
	LectureCapacity             *int       `json:"lecture_capacity"`
	LectureIsPublic             bool       `json:"lecture_is_public"`
	LectureIsActive             bool       `json:"lecture_is_active"`
	LectureIsCertificateGenerated bool `json:"lecture_is_certificate_generated"`
	LectureCreatedAt            string     `json:"lecture_created_at"`
	LectureUpdatedAt            *string    `json:"lecture_updated_at,omitempty"`
	LectureDeletedAt            *string    `json:"lecture_deleted_at,omitempty"`
}

// ========================
// CONVERTER
// ========================
func (r *LectureRequest) ToModel() *model.LectureModel {
	teacherJSON, _ := json.Marshal(r.LectureTeachers)

	return &model.LectureModel{
		LectureTitle:                 r.LectureTitle,
		LectureDescription:           r.LectureDescription,
		LectureMasjidID:              r.LectureMasjidID,
		TotalLectureSessions:         r.TotalLectureSessions,
		LectureIsRecurring:           r.LectureIsRecurring,
		LectureRecurrenceInterval:    r.LectureRecurrenceInterval,
		LectureImageURL:              r.LectureImageURL,
		LectureTeachers:              datatypes.JSON(teacherJSON),
		LectureIsRegistrationRequired: r.LectureIsRegistrationRequired,
		LectureIsPaid:                r.LectureIsPaid,
		LecturePrice:                 r.LecturePrice,
		LecturePaymentDeadline:       r.LecturePaymentDeadline,
		LectureCapacity:              r.LectureCapacity,
		LectureIsPublic:              r.LectureIsPublic,
		LectureIsActive:              r.LectureIsActive,
	}
}

func ToLectureResponse(m *model.LectureModel) *LectureResponse {
	var teachers []Teacher
	if m.LectureTeachers != nil {
		_ = json.Unmarshal(m.LectureTeachers, &teachers)
	}

	var updatedAtStr *string
	if m.LectureUpdatedAt != nil {
		str := m.LectureUpdatedAt.Format("2006-01-02 15:04:05")
		updatedAtStr = &str
	}

	var deletedAtStr *string
	if m.DeletedAt.Valid {
		str := m.DeletedAt.Time.Format("2006-01-02 15:04:05")
		deletedAtStr = &str
	}

	return &LectureResponse{
		LectureID:                   m.LectureID,
		LectureTitle:                m.LectureTitle,
		LectureDescription:          m.LectureDescription,
		LectureMasjidID:             m.LectureMasjidID,
		TotalLectureSessions:        m.TotalLectureSessions,
		LectureIsRecurring:          m.LectureIsRecurring,
		LectureRecurrenceInterval:   m.LectureRecurrenceInterval,
		LectureImageURL:             m.LectureImageURL,
		LectureTeachers:             teachers,
		LectureIsRegistrationRequired: m.LectureIsRegistrationRequired,
		LectureIsPaid:               m.LectureIsPaid,
		LecturePrice:                m.LecturePrice,
		LecturePaymentDeadline:      m.LecturePaymentDeadline,
		LectureCapacity:             m.LectureCapacity,
		LectureIsPublic:             m.LectureIsPublic,
		LectureIsActive:             m.LectureIsActive,
		LectureIsCertificateGenerated: m.LectureIsCerticateGenerated,
		LectureCreatedAt:            m.LectureCreatedAt.Format("2006-01-02 15:04:05"),
		LectureUpdatedAt:            updatedAtStr,
		LectureDeletedAt:            deletedAtStr,
	}
}

func ToLectureResponseList(lectures []model.LectureModel) []*LectureResponse {
	var responses []*LectureResponse
	for _, lecture := range lectures {
		responses = append(responses, ToLectureResponse(&lecture))
	}
	return responses
}
