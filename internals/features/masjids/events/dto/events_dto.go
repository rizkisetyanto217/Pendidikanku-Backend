package dto

import (
	"masjidku_backend/internals/features/masjids/events/model"
	"strings"

	"github.com/google/uuid"
)

// 🔹 Request untuk membuat event
type EventRequest struct {
	EventTitle       string    `json:"event_title"`
	EventDescription string    `json:"event_description"`
	EventLocation    string    `json:"event_location"`
	EventMasjidID    uuid.UUID `json:"event_masjid_id"`
}

// 🔹 Response untuk menampilkan event
type EventResponse struct {
	EventID          uuid.UUID `json:"event_id"`
	EventTitle       string    `json:"event_title"`
	EventSlug        string    `json:"event_slug"`
	EventDescription string    `json:"event_description"`
	EventLocation    string    `json:"event_location"`
	EventMasjidID    uuid.UUID `json:"event_masjid_id"`
	EventCreatedAt   string    `json:"event_created_at"`
}


// 🔹 Request partial update pakai pointer agar bisa bedakan "tidak dikirim" vs "string kosong"
type EventUpdateRequest struct {
	EventTitle       *string    `json:"event_title"`       // jika diisi → slug ikut diperbarui
	EventDescription *string    `json:"event_description"`
	EventLocation    *string    `json:"event_location"`
	EventMasjidID    *uuid.UUID `json:"event_masjid_id"`
}


// 🔄 Exported biar bisa dipakai di controller
func GenerateSlug(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}

// 🔄 Konversi dari request → model
func (r *EventRequest) ToModel() *model.EventModel {
	return &model.EventModel{
		EventTitle:       r.EventTitle,
		EventSlug:        GenerateSlug(r.EventTitle),
		EventDescription: r.EventDescription,
		EventLocation:    r.EventLocation,
		EventMasjidID:    r.EventMasjidID,
	}
}

// 🔄 Konversi dari model → response
func ToEventResponse(m *model.EventModel) *EventResponse {
	return &EventResponse{
		EventID:          m.EventID,
		EventTitle:       m.EventTitle,
		EventSlug:        m.EventSlug,
		EventDescription: m.EventDescription,
		EventLocation:    m.EventLocation,
		EventMasjidID:    m.EventMasjidID,
		EventCreatedAt:   m.EventCreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// 🔄 Konversi list model → list response
func ToEventResponseList(models []model.EventModel) []EventResponse {
	var result []EventResponse
	for _, m := range models {
		result = append(result, *ToEventResponse(&m))
	}
	return result
}
