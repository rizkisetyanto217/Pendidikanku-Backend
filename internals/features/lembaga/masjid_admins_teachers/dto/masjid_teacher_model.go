package dto

import (
	"time"
)

// ========================
// 📦 DTO Full Mirror (Entity Snapshot)
// ========================
type MasjidTeacher struct {
	MasjidTeacherID        string     `json:"masjid_teacher_id"`
	MasjidTeacherMasjidID  string     `json:"masjid_teacher_masjid_id"`
	MasjidTeacherUserID    string     `json:"masjid_teacher_user_id"`
	MasjidTeacherCreatedAt time.Time  `json:"masjid_teacher_created_at"`
	MasjidTeacherUpdatedAt time.Time  `json:"masjid_teacher_updated_at"`
	MasjidTeacherDeletedAt *time.Time `json:"masjid_teacher_deleted_at,omitempty"`
}

// ========================
// 📥 Create Request DTO
// ========================
type CreateMasjidTeacherRequest struct {
	MasjidTeacherMasjidID string `json:"masjid_teacher_masjid_id" validate:"required,uuid"`
	MasjidTeacherUserID   string `json:"masjid_teacher_user_id" validate:"required,uuid"`
}

// ========================
// ✏️ Update Request DTO (opsional)
// ========================
type UpdateMasjidTeacherRequest struct {
	MasjidTeacherMasjidID *string `json:"masjid_teacher_masjid_id,omitempty" validate:"omitempty,uuid"`
	MasjidTeacherUserID   *string `json:"masjid_teacher_user_id,omitempty" validate:"omitempty,uuid"`
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
	copy(out, items)
	return out
}
