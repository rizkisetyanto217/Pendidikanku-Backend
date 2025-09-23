// file: internals/features/lembaga/classes/dto/class_room_virtual_link_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	clsModel "masjidku_backend/internals/features/school/academics/rooms/model"
)

/* =========================================================
   REQUEST DTO — CREATE (writable fields only)
   Catatan:
   - Field wajib: label, join_url, masjid_id, room_id, platform
   - HostURL/MeetingID/Passcode/Notes opsional: "" => NULL (di-trim)
========================================================= */

type ClassRoomVirtualLinkCreateRequest struct {
	// Scope
	ClassRoomVirtualLinkMasjidID uuid.UUID `json:"class_room_virtual_link_masjid_id"`
	ClassRoomVirtualLinkRoomID   uuid.UUID `json:"class_room_virtual_link_room_id"`

	// Identitas link
	ClassRoomVirtualLinkLabel     string `json:"class_room_virtual_link_label"`
	ClassRoomVirtualLinkJoinURL   string `json:"class_room_virtual_link_join_url"`
	ClassRoomVirtualLinkHostURL   string `json:"class_room_virtual_link_host_url"`
	ClassRoomVirtualLinkMeetingID string `json:"class_room_virtual_link_meeting_id"`
	ClassRoomVirtualLinkPasscode  string `json:"class_room_virtual_link_passcode"`
	ClassRoomVirtualLinkNotes     string `json:"class_room_virtual_link_notes"`

	// Platform (wajib; ex: "zoom" | "google_meet" | "microsoft_teams" | "other")
	ClassRoomVirtualLinkPlatform string `json:"class_room_virtual_link_platform"`

	// Status
	ClassRoomVirtualLinkIsActive bool `json:"class_room_virtual_link_is_active"`
}

/* =========================================================
   RESPONSE DTO — lengkap untuk client
========================================================= */

type ClassRoomVirtualLinkResponse struct {
	ClassRoomVirtualLinkID uuid.UUID `json:"class_room_virtual_link_id"`

	// Scope
	ClassRoomVirtualLinkMasjidID uuid.UUID `json:"class_room_virtual_link_masjid_id"`
	ClassRoomVirtualLinkRoomID   uuid.UUID `json:"class_room_virtual_link_room_id"`

	// Identitas link
	ClassRoomVirtualLinkLabel     string `json:"class_room_virtual_link_label"`
	ClassRoomVirtualLinkJoinURL   string `json:"class_room_virtual_link_join_url"`
	ClassRoomVirtualLinkHostURL   string `json:"class_room_virtual_link_host_url"`
	ClassRoomVirtualLinkMeetingID string `json:"class_room_virtual_link_meeting_id"`
	ClassRoomVirtualLinkPasscode  string `json:"class_room_virtual_link_passcode"`
	ClassRoomVirtualLinkNotes     string `json:"class_room_virtual_link_notes"`

	// Platform
	ClassRoomVirtualLinkPlatform string `json:"class_room_virtual_link_platform"`

	// Status
	ClassRoomVirtualLinkIsActive bool `json:"class_room_virtual_link_is_active"`

	// Audit
	ClassRoomVirtualLinkCreatedAt time.Time `json:"class_room_virtual_link_created_at"`
	ClassRoomVirtualLinkUpdatedAt time.Time `json:"class_room_virtual_link_updated_at"`
}

/* =========================================================
   PARTIAL UPDATE DTO — pointer semua writable fields
   Catatan:
   - nil → tidak diubah
   - Clear[] → set kolom opsional menjadi NULL eksplisit
   - Platform TIDAK nullable → tidak ada di Clear
========================================================= */

type ClassRoomVirtualLinkUpdateRequest struct {
	// Scope (boleh diubah jika perlu pindah masjid/room)
	ClassRoomVirtualLinkMasjidID *uuid.UUID `json:"class_room_virtual_link_masjid_id"`
	ClassRoomVirtualLinkRoomID   *uuid.UUID `json:"class_room_virtual_link_room_id"`

	// Identitas link
	ClassRoomVirtualLinkLabel     *string `json:"class_room_virtual_link_label"`
	ClassRoomVirtualLinkJoinURL   *string `json:"class_room_virtual_link_join_url"`
	ClassRoomVirtualLinkHostURL   *string `json:"class_room_virtual_link_host_url"`
	ClassRoomVirtualLinkMeetingID *string `json:"class_room_virtual_link_meeting_id"`
	ClassRoomVirtualLinkPasscode  *string `json:"class_room_virtual_link_passcode"`
	ClassRoomVirtualLinkNotes     *string `json:"class_room_virtual_link_notes"`

	// Platform (non-nullable)
	ClassRoomVirtualLinkPlatform *string `json:"class_room_virtual_link_platform"`

	// Status
	ClassRoomVirtualLinkIsActive *bool `json:"class_room_virtual_link_is_active"`

	// Clear → set NULL eksplisit utk kolom opsional
	// allowed: host_url meeting_id passcode notes
	Clear []string `json:"__clear,omitempty"`
}

/* =========================================================
   KONVERSI MODEL <-> DTO
========================================================= */

func FromModelClassRoomVirtualLink(m *clsModel.ClassRoomVirtualLinkModel) ClassRoomVirtualLinkResponse {
	return ClassRoomVirtualLinkResponse{
		ClassRoomVirtualLinkID:        m.ClassRoomVirtualLinkID,
		ClassRoomVirtualLinkMasjidID:  m.ClassRoomVirtualLinkMasjidID,
		ClassRoomVirtualLinkRoomID:    m.ClassRoomVirtualLinkRoomID,
		ClassRoomVirtualLinkLabel:     m.ClassRoomVirtualLinkLabel,
		ClassRoomVirtualLinkJoinURL:   m.ClassRoomVirtualLinkJoinURL,
		ClassRoomVirtualLinkHostURL:   valOrEmpty(m.ClassRoomVirtualLinkHostURL),
		ClassRoomVirtualLinkMeetingID: valOrEmpty(m.ClassRoomVirtualLinkMeetingID),
		ClassRoomVirtualLinkPasscode:  valOrEmpty(m.ClassRoomVirtualLinkPasscode),
		ClassRoomVirtualLinkNotes:     valOrEmpty(m.ClassRoomVirtualLinkNotes),
		ClassRoomVirtualLinkPlatform:  string(m.ClassRoomVirtualLinkPlatform), // enum -> string
		ClassRoomVirtualLinkIsActive:  m.ClassRoomVirtualLinkIsActive,
		ClassRoomVirtualLinkCreatedAt: m.ClassRoomVirtualLinkCreatedAt,
		ClassRoomVirtualLinkUpdatedAt: m.ClassRoomVirtualLinkUpdatedAt,
	}
}

// ToModel: buat instance model dari request (untuk INSERT)
// id biasa dibiarkan default di DB (gen_random_uuid), tapi disediakan param kalau mau set manual.
func ToModelClassRoomVirtualLink(in *ClassRoomVirtualLinkCreateRequest, id *uuid.UUID) *clsModel.ClassRoomVirtualLinkModel {
	out := &clsModel.ClassRoomVirtualLinkModel{
		ClassRoomVirtualLinkMasjidID:  in.ClassRoomVirtualLinkMasjidID,
		ClassRoomVirtualLinkRoomID:    in.ClassRoomVirtualLinkRoomID,
		ClassRoomVirtualLinkLabel:     strings.TrimSpace(in.ClassRoomVirtualLinkLabel),
		ClassRoomVirtualLinkJoinURL:   strings.TrimSpace(in.ClassRoomVirtualLinkJoinURL),
		ClassRoomVirtualLinkHostURL:   normalizeOptionalStringToPtr(in.ClassRoomVirtualLinkHostURL),
		ClassRoomVirtualLinkMeetingID: normalizeOptionalStringToPtr(in.ClassRoomVirtualLinkMeetingID),
		ClassRoomVirtualLinkPasscode:  normalizeOptionalStringToPtr(in.ClassRoomVirtualLinkPasscode),
		ClassRoomVirtualLinkNotes:     normalizeOptionalStringToPtr(in.ClassRoomVirtualLinkNotes),

		// platform: DTO string -> model enum (string underlying)
		ClassRoomVirtualLinkPlatform: clsModel.VirtualPlatform(strings.TrimSpace(in.ClassRoomVirtualLinkPlatform)),

		ClassRoomVirtualLinkIsActive: in.ClassRoomVirtualLinkIsActive,
	}
	if id != nil && *id != uuid.Nil {
		out.ClassRoomVirtualLinkID = *id
	}
	return out
}

// ApplyUpdate: patch model dari UpdateRequest (gunakan sebelum save)
func ApplyUpdateClassRoomVirtualLink(m *clsModel.ClassRoomVirtualLinkModel, u *ClassRoomVirtualLinkUpdateRequest) {
	if u == nil {
		return
	}

	// Scope
	if u.ClassRoomVirtualLinkMasjidID != nil {
		m.ClassRoomVirtualLinkMasjidID = *u.ClassRoomVirtualLinkMasjidID
	}
	if u.ClassRoomVirtualLinkRoomID != nil {
		m.ClassRoomVirtualLinkRoomID = *u.ClassRoomVirtualLinkRoomID
	}

	// Identitas
	if u.ClassRoomVirtualLinkLabel != nil {
		m.ClassRoomVirtualLinkLabel = strings.TrimSpace(*u.ClassRoomVirtualLinkLabel)
	}
	if u.ClassRoomVirtualLinkJoinURL != nil {
		m.ClassRoomVirtualLinkJoinURL = strings.TrimSpace(*u.ClassRoomVirtualLinkJoinURL)
	}
	if u.ClassRoomVirtualLinkHostURL != nil {
		m.ClassRoomVirtualLinkHostURL = normalizeOptionalStringToPtr(strings.TrimSpace(*u.ClassRoomVirtualLinkHostURL))
	}
	if u.ClassRoomVirtualLinkMeetingID != nil {
		m.ClassRoomVirtualLinkMeetingID = normalizeOptionalStringToPtr(strings.TrimSpace(*u.ClassRoomVirtualLinkMeetingID))
	}
	if u.ClassRoomVirtualLinkPasscode != nil {
		m.ClassRoomVirtualLinkPasscode = normalizeOptionalStringToPtr(strings.TrimSpace(*u.ClassRoomVirtualLinkPasscode))
	}
	if u.ClassRoomVirtualLinkNotes != nil {
		m.ClassRoomVirtualLinkNotes = normalizeOptionalStringToPtr(strings.TrimSpace(*u.ClassRoomVirtualLinkNotes))
	}

	// Platform (non-nullable)
	if u.ClassRoomVirtualLinkPlatform != nil {
		m.ClassRoomVirtualLinkPlatform = clsModel.VirtualPlatform(strings.TrimSpace(*u.ClassRoomVirtualLinkPlatform))
	}

	// Status
	if u.ClassRoomVirtualLinkIsActive != nil {
		m.ClassRoomVirtualLinkIsActive = *u.ClassRoomVirtualLinkIsActive
	}

	// Clear → NULL eksplisit
	for _, col := range u.Clear {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "host_url", "class_room_virtual_link_host_url":
			m.ClassRoomVirtualLinkHostURL = nil
		case "meeting_id", "class_room_virtual_link_meeting_id":
			m.ClassRoomVirtualLinkMeetingID = nil
		case "passcode", "class_room_virtual_link_passcode":
			m.ClassRoomVirtualLinkPasscode = nil
		case "notes", "class_room_virtual_link_notes":
			m.ClassRoomVirtualLinkNotes = nil
		}
	}
}

/* =========================================================
   HELPERS
========================================================= */

// "" atau whitespace → nil, selain itu trim
func normalizeOptionalStringToPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	return &trim
}

// util respon: kembalikan "" jika nil
func valOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
