package dto

import (
	"time"
)

// ========================
// 📦 DTO Full Mirror (Entity Snapshot)
// ========================
type MasjidTeacher struct {
	MasjidTeachersID        string     `json:"masjid_teachers_id"`
	MasjidTeachersMasjidID  string     `json:"masjid_teachers_masjid_id"`
	MasjidTeachersUserID    string     `json:"masjid_teachers_user_id"`
	MasjidTeachersCreatedAt time.Time  `json:"masjid_teachers_created_at"`
	MasjidTeachersUpdatedAt time.Time  `json:"masjid_teachers_updated_at"`
	MasjidTeachersDeletedAt *time.Time `json:"masjid_teachers_deleted_at,omitempty"`
}

// ========================
// 📥 Create Request DTO
// ========================
type CreateMasjidTeacherRequest struct {
	MasjidTeachersMasjidID string `json:"masjid_teachers_masjid_id" validate:"required,uuid"`
	MasjidTeachersUserID   string `json:"masjid_teachers_user_id" validate:"required,uuid"`
}

// ========================
// ✏️ Update Request DTO (opsional)
// ========================
type UpdateMasjidTeacherRequest struct {
	MasjidTeachersMasjidID *string `json:"masjid_teachers_masjid_id,omitempty" validate:"omitempty,uuid"`
	MasjidTeachersUserID   *string `json:"masjid_teachers_user_id,omitempty" validate:"omitempty,uuid"`
}

// ========================
// 📤 Response DTO (alias supaya hilang S1016)
// ========================
type MasjidTeacherResponse = MasjidTeacher

// ========================
// 🔁 Converters
// ========================
func ToMasjidTeacherResponse(m MasjidTeacher) MasjidTeacherResponse {
	return m
}

func ToMasjidTeacherResponses(items []MasjidTeacher) []MasjidTeacherResponse {
	out := make([]MasjidTeacherResponse, len(items))
	copy(out, items) // ✅ idiomatic, hilang warning S1001
	return out
}
