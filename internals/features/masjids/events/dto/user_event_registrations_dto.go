package dto

import (
	"masjidku_backend/internals/features/masjids/events/model"

	"github.com/google/uuid"
)

// Request untuk membuat pendaftaran event
type UserEventRegistrationRequest struct {
	EventID uuid.UUID `json:"user_event_registration_event_id"`
	UserID  uuid.UUID `json:"user_event_registration_user_id"`
	Status  string    `json:"user_event_registration_status"` // Optional, default "registered"
}

// Response untuk menampilkan data pendaftaran event
type UserEventRegistrationResponse struct {
	ID         uuid.UUID `json:"user_event_registration_id"`
	EventID    uuid.UUID `json:"user_event_registration_event_id"`
	UserID     uuid.UUID `json:"user_event_registration_user_id"`
	Status     string    `json:"user_event_registration_status"`
	Registered string    `json:"user_event_registration_registered_at"`
}

// Konversi request ke model
func (r *UserEventRegistrationRequest) ToModel() *model.UserEventRegistrationModel {
	status := r.Status
	if status == "" {
		status = "registered"
	}
	return &model.UserEventRegistrationModel{
		UserEventRegistrationEventID: r.EventID,
		UserEventRegistrationUserID:  r.UserID,
		UserEventRegistrationStatus:  status,
	}
}

// Konversi model ke response
func ToUserEventRegistrationResponse(m *model.UserEventRegistrationModel) *UserEventRegistrationResponse {
	return &UserEventRegistrationResponse{
		ID:         m.UserEventRegistrationID,
		EventID:    m.UserEventRegistrationEventID,
		UserID:     m.UserEventRegistrationUserID,
		Status:     m.UserEventRegistrationStatus,
		Registered: m.UserEventRegistrationCreatedAt.Format("2006-01-02 15:04:05"),
	}
}
