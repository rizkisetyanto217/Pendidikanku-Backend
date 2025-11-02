package dto

import (
	"schoolku_backend/internals/features/schools/events/model"

	"github.com/google/uuid"
)

// ===== Request =====

type UserEventRegistrationRequest struct {
	EventSessionID uuid.UUID `json:"user_event_registration_event_session_id"` // match DB
	UserID         uuid.UUID `json:"user_event_registration_user_id"`
	SchoolID       uuid.UUID `json:"user_event_registration_school_id"`
	Status         string    `json:"user_event_registration_status"` // optional, default "registered"
}

// ===== Response =====

type UserEventRegistrationResponse struct {
	ID         uuid.UUID `json:"user_event_registration_id"`
	EventID    uuid.UUID `json:"user_event_registration_event_session_id"`
	UserID     uuid.UUID `json:"user_event_registration_user_id"`
	SchoolID   uuid.UUID `json:"user_event_registration_school_id"`
	Status     string    `json:"user_event_registration_status"`
	Registered string    `json:"user_event_registration_registered_at"`
	UpdatedAt  string    `json:"user_event_registration_updated_at"`
}

// ===== Converters =====

func (r *UserEventRegistrationRequest) ToModel() *model.UserEventRegistrationModel {
	status := r.Status
	if status == "" {
		status = "registered"
	}
	return &model.UserEventRegistrationModel{
		UserEventRegistrationEventID:  r.EventSessionID, // column: user_event_registration_event_session_id
		UserEventRegistrationUserID:   r.UserID,
		UserEventRegistrationSchoolID: r.SchoolID,
		UserEventRegistrationStatus:   status,
	}
}

func ToUserEventRegistrationResponse(m *model.UserEventRegistrationModel) *UserEventRegistrationResponse {
	const fmt = "2006-01-02 15:04:05"
	return &UserEventRegistrationResponse{
		ID:         m.UserEventRegistrationID,
		EventID:    m.UserEventRegistrationEventID, // event_session_id
		UserID:     m.UserEventRegistrationUserID,
		SchoolID:   m.UserEventRegistrationSchoolID,
		Status:     m.UserEventRegistrationStatus,
		Registered: toTimeString(m.UserEventRegistrationCreatedAt, fmt),
		UpdatedAt:  toTimeString(m.UserEventRegistrationUpdatedAt, fmt),
	}
}

func ToUserEventRegistrationResponseList(models []model.UserEventRegistrationModel) []UserEventRegistrationResponse {
	out := make([]UserEventRegistrationResponse, 0, len(models))
	for i := range models {
		out = append(out, *ToUserEventRegistrationResponse(&models[i]))
	}
	return out
}
