package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"masjidku_backend/internals/features/masjids/events/model"
)

//
// ========= Request DTO =========
//

// ✅ Request untuk create event session
// Catatan: event_session_created_by diisi di controller dari token.
type EventSessionRequest struct {
	EventSessionEventID               uuid.UUID  `json:"event_session_event_id"`
	EventSessionSlug                  string     `json:"event_session_slug"` // boleh kosong → auto dari title
	EventSessionTitle                 string     `json:"event_session_title"`
	EventSessionDescription           string     `json:"event_session_description"`
	EventSessionStartTime             time.Time  `json:"event_session_start_time"`
	EventSessionEndTime               time.Time  `json:"event_session_end_time"`
	EventSessionLocation              string     `json:"event_session_location"`
	EventSessionImageURL              string     `json:"event_session_image_url"`
	EventSessionCapacity              *int       `json:"event_session_capacity"` // nullable
	EventSessionMasjidID              uuid.UUID  `json:"event_session_masjid_id"`
	EventSessionIsPublic              bool       `json:"event_session_is_public"`
	EventSessionIsRegistrationRequired bool      `json:"event_session_is_registration_required"`
	EventSessionCreatedBy             *uuid.UUID `json:"event_session_created_by"` // akan dioverride di controller
}

// Request untuk partial update (PATCH) — gunakan pointer
type EventSessionUpdateRequest struct {
	EventSessionSlug                   *string    `json:"event_session_slug"`
	EventSessionTitle                  *string    `json:"event_session_title"`
	EventSessionDescription            *string    `json:"event_session_description"`
	EventSessionStartTime              *time.Time `json:"event_session_start_time"`
	EventSessionEndTime                *time.Time `json:"event_session_end_time"`
	EventSessionLocation               *string    `json:"event_session_location"`
	EventSessionImageURL               *string    `json:"event_session_image_url"`
	EventSessionCapacity               *int       `json:"event_session_capacity"`
	EventSessionMasjidID               *uuid.UUID `json:"event_session_masjid_id"`
	EventSessionIsPublic               *bool      `json:"event_session_is_public"`
	EventSessionIsRegistrationRequired *bool      `json:"event_session_is_registration_required"`
	EventSessionCreatedBy              *uuid.UUID `json:"event_session_created_by"` // opsional
}

//
// ========= Response DTO =========
//

type EventSessionResponse struct {
	EventSessionID                    uuid.UUID  `json:"event_session_id"`
	EventSessionEventID               uuid.UUID  `json:"event_session_event_id"`
	EventSessionSlug                  string     `json:"event_session_slug"`
	EventSessionTitle                 string     `json:"event_session_title"`
	EventSessionDescription           string     `json:"event_session_description"`
	EventSessionStartTime             string     `json:"event_session_start_time"`
	EventSessionEndTime               string     `json:"event_session_end_time"`
	EventSessionLocation              string     `json:"event_session_location"`
	EventSessionImageURL              string     `json:"event_session_image_url"`
	EventSessionCapacity              *int       `json:"event_session_capacity,omitempty"`
	EventSessionIsPublic              bool       `json:"event_session_is_public"`
	EventSessionIsRegistrationRequired bool      `json:"event_session_is_registration_required"`
	EventSessionMasjidID              uuid.UUID  `json:"event_session_masjid_id"`
	EventSessionCreatedBy             *uuid.UUID `json:"event_session_created_by"`
	EventSessionCreatedAt             string     `json:"event_session_created_at"`
	EventSessionUpdatedAt             string     `json:"event_session_updated_at"`
}

//
// ========= Helpers & Converters =========
//

// 🔄 Request → Model (Create)
func (r *EventSessionRequest) ToModel() *model.EventSessionModel {
	slug := r.EventSessionSlug
	if strings.TrimSpace(slug) == "" {
		slug = GenerateSlug(r.EventSessionTitle)
	}
	return &model.EventSessionModel{
		EventSessionEventID:               r.EventSessionEventID,
		EventSessionSlug:                  slug,
		EventSessionTitle:                 r.EventSessionTitle,
		EventSessionDescription:           r.EventSessionDescription,
		EventSessionStartTime:             r.EventSessionStartTime,
		EventSessionEndTime:               r.EventSessionEndTime,
		EventSessionLocation:              r.EventSessionLocation,
		EventSessionImageURL:              r.EventSessionImageURL,
		EventSessionCapacity:              r.EventSessionCapacity,
		EventSessionIsPublic:              r.EventSessionIsPublic,
		EventSessionIsRegistrationRequired: r.EventSessionIsRegistrationRequired,
		EventSessionMasjidID:              r.EventSessionMasjidID,
		EventSessionCreatedBy:             r.EventSessionCreatedBy, // controller boleh override dari token
	}
}

// 🔧 Terapkan PATCH ke Model (Partial Update)
func (r *EventSessionUpdateRequest) ApplyToModel(m *model.EventSessionModel) {
	if r.EventSessionSlug != nil {
		m.EventSessionSlug = strings.TrimSpace(*r.EventSessionSlug)
	}
	if r.EventSessionTitle != nil {
		m.EventSessionTitle = *r.EventSessionTitle
	}
	if r.EventSessionDescription != nil {
		m.EventSessionDescription = *r.EventSessionDescription
	}
	if r.EventSessionStartTime != nil {
		m.EventSessionStartTime = *r.EventSessionStartTime
	}
	if r.EventSessionEndTime != nil {
		m.EventSessionEndTime = *r.EventSessionEndTime
	}
	if r.EventSessionLocation != nil {
		m.EventSessionLocation = *r.EventSessionLocation
	}
	if r.EventSessionImageURL != nil {
		m.EventSessionImageURL = *r.EventSessionImageURL
	}
	if r.EventSessionCapacity != nil {
		m.EventSessionCapacity = r.EventSessionCapacity
	}
	if r.EventSessionIsPublic != nil {
		m.EventSessionIsPublic = *r.EventSessionIsPublic
	}
	if r.EventSessionIsRegistrationRequired != nil {
		m.EventSessionIsRegistrationRequired = *r.EventSessionIsRegistrationRequired
	}
	if r.EventSessionMasjidID != nil {
		m.EventSessionMasjidID = *r.EventSessionMasjidID
	}
	if r.EventSessionCreatedBy != nil {
		m.EventSessionCreatedBy = r.EventSessionCreatedBy
	}
}

// 🔄 Model → Response
func ToEventSessionResponse(m *model.EventSessionModel) *EventSessionResponse {
	const fmt = "2006-01-02 15:04:05"
	return &EventSessionResponse{
		EventSessionID:                     m.EventSessionID,
		EventSessionEventID:                m.EventSessionEventID,
		EventSessionSlug:                   m.EventSessionSlug,
		EventSessionTitle:                  m.EventSessionTitle,
		EventSessionDescription:            m.EventSessionDescription,
		EventSessionStartTime:              m.EventSessionStartTime.Format(fmt),
		EventSessionEndTime:                m.EventSessionEndTime.Format(fmt),
		EventSessionLocation:               m.EventSessionLocation,
		EventSessionImageURL:               m.EventSessionImageURL,
		EventSessionCapacity:               m.EventSessionCapacity,
		EventSessionIsPublic:               m.EventSessionIsPublic,
		EventSessionIsRegistrationRequired: m.EventSessionIsRegistrationRequired,
		EventSessionMasjidID:               m.EventSessionMasjidID,
		EventSessionCreatedBy:              m.EventSessionCreatedBy,
		EventSessionCreatedAt:              m.EventSessionCreatedAt.Format(fmt),
		EventSessionUpdatedAt:              m.EventSessionUpdatedAt.Format(fmt),
	}
}

// 🔄 List Model → List Response
func ToEventSessionResponseList(models []model.EventSessionModel) []EventSessionResponse {
	out := make([]EventSessionResponse, 0, len(models))
	for i := range models {
		out = append(out, *ToEventSessionResponse(&models[i]))
	}
	return out
}
