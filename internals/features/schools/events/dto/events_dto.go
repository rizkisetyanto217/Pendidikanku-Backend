package dto

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"schoolku_backend/internals/features/schools/events/model"
)

/* =========================================================
   Request / Response
========================================================= */

// 🔹 Request untuk membuat event
type EventRequest struct {
	EventTitle       string    `json:"event_title"`                 // required
	EventDescription string    `json:"event_description,omitempty"` // optional
	EventLocation    string    `json:"event_location,omitempty"`    // optional
	EventSchoolID    uuid.UUID `json:"event_school_id"`             // required
}

// 🔹 Response untuk menampilkan event (ikutkan updated_at biar konsisten dengan DB)
type EventResponse struct {
	EventID          uuid.UUID `json:"event_id"`
	EventTitle       string    `json:"event_title"`
	EventSlug        string    `json:"event_slug"`
	EventDescription string    `json:"event_description,omitempty"`
	EventLocation    string    `json:"event_location,omitempty"`
	EventSchoolID    uuid.UUID `json:"event_school_id"`
	EventCreatedAt   string    `json:"event_created_at"` // "YYYY-MM-DD HH:mm:ss"
	EventUpdatedAt   string    `json:"event_updated_at"` // "YYYY-MM-DD HH:mm:ss"
	// NOTE: deleted_at sengaja tidak diekspos di response publik
}

// 🔹 Request partial update (pointer untuk bedakan "tidak dikirim" vs "string kosong")
type EventUpdateRequest struct {
	EventTitle       *string    `json:"event_title"`       // jika diisi → slug ikut diperbarui
	EventDescription *string    `json:"event_description"` // boleh string kosong
	EventLocation    *string    `json:"event_location"`    // boleh string kosong
	EventSchoolID    *uuid.UUID `json:"event_school_id"`   // hati-hati: mengubah scope slug uniqueness
}

/* =========================================================
   Helper & Converter
========================================================= */

// Slugify yang lebih rapi:
// - trim
// - lower
// - ganti spasi & underscore jadi '-'
// - hapus selain [a-z0-9-]
// - collapse multiple '-' → satu '-'
// - trim '-' di awal/akhir
var (
	reNotAllowed   = regexp.MustCompile(`[^a-z0-9\-]+`)
	reSpacesUnders = regexp.MustCompile(`[ _]+`)
	reMultiDash    = regexp.MustCompile(`\-{2,}`)
)

func GenerateSlug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = reSpacesUnders.ReplaceAllString(s, "-")
	s = reNotAllowed.ReplaceAllString(s, "-")
	s = reMultiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// Konversi dari request → model (untuk Create)
func (r *EventRequest) ToModel() *model.EventModel {
	return &model.EventModel{
		EventTitle:       r.EventTitle,
		EventSlug:        GenerateSlug(r.EventTitle),
		EventDescription: r.EventDescription,
		EventLocation:    r.EventLocation,
		EventSchoolID:    r.EventSchoolID,
		// created_at, updated_at diisi otomatis oleh DB / GORM
	}
}

// Terapkan partial update ke model (untuk Update)
func (r *EventUpdateRequest) ApplyToModel(m *model.EventModel) {
	if r.EventTitle != nil {
		m.EventTitle = *r.EventTitle
		// ikut perbarui slug (controller tetap perlu handle konflik unik di DB)
		m.EventSlug = GenerateSlug(m.EventTitle)
	}
	if r.EventDescription != nil {
		m.EventDescription = *r.EventDescription
	}
	if r.EventLocation != nil {
		m.EventLocation = *r.EventLocation
	}
	if r.EventSchoolID != nil {
		m.EventSchoolID = *r.EventSchoolID
	}
	// EventUpdatedAt diisi oleh DB trigger / GORM autoUpdateTime
}

// Konversi dari model → response
func ToEventResponse(m *model.EventModel) *EventResponse {
	const fmt = "2006-01-02 15:04:05"
	return &EventResponse{
		EventID:          m.EventID,
		EventTitle:       m.EventTitle,
		EventSlug:        m.EventSlug,
		EventDescription: m.EventDescription,
		EventLocation:    m.EventLocation,
		EventSchoolID:    m.EventSchoolID,
		EventCreatedAt:   toTimeString(m.EventCreatedAt, fmt),
		EventUpdatedAt:   toTimeString(m.EventUpdatedAt, fmt),
	}
}

// Konversi list model → list response
func ToEventResponseList(models []model.EventModel) []EventResponse {
	result := make([]EventResponse, 0, len(models))
	for i := range models {
		result = append(result, *ToEventResponse(&models[i]))
	}
	return result
}

/* =========================================================
   Util
========================================================= */

func toTimeString(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(layout)
}
