package dto

import (
	"time"
)

// ========================
// 📦 DTO Full Mirror Model
// ========================
type MasjidTeacher struct {
	MasjidTeachersID        string    `gorm:"column:masjid_teachers_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"masjid_teachers_id"`
	MasjidTeachersMasjidID  string    `gorm:"column:masjid_teachers_masjid_id;type:uuid;not null" json:"masjid_teachers_masjid_id"`
	MasjidTeachersUserID    string    `gorm:"column:masjid_teachers_user_id;type:uuid;not null" json:"masjid_teachers_user_id"`
	MasjidTeachersCreatedAt time.Time `gorm:"column:masjid_teachers_created_at;autoCreateTime" json:"masjid_teachers_created_at"`
	MasjidTeachersUpdatedAt time.Time `gorm:"column:masjid_teachers_updated_at;autoUpdateTime" json:"masjid_teachers_updated_at"`
}

func (MasjidTeacher) TableName() string {
	return "masjid_teachers"
}

// ========================
// 📥 Create Request DTO
// ========================
type CreateMasjidTeacherRequest struct {
	MasjidTeachersMasjidID string `json:"masjid_teachers_masjid_id" validate:"required,uuid"`
	MasjidTeachersUserID   string `json:"masjid_teachers_user_id" validate:"required,uuid"`
}

// ========================
// 📤 Response DTO
// ========================
type MasjidTeacherResponse struct {
	MasjidTeachersID        string    `json:"masjid_teachers_id"`
	MasjidTeachersMasjidID  string    `json:"masjid_teachers_masjid_id"`
	MasjidTeachersUserID    string    `json:"masjid_teachers_user_id"`
	MasjidTeachersCreatedAt time.Time `json:"masjid_teachers_created_at"`
	MasjidTeachersUpdatedAt time.Time `json:"masjid_teachers_updated_at"`
}

// ========================
// 🔁 Converter
// ========================
func ToMasjidTeacherResponse(m MasjidTeacher) MasjidTeacherResponse {
	return MasjidTeacherResponse{
		MasjidTeachersID:        m.MasjidTeachersID,
		MasjidTeachersMasjidID:  m.MasjidTeachersMasjidID,
		MasjidTeachersUserID:    m.MasjidTeachersUserID,
		MasjidTeachersCreatedAt: m.MasjidTeachersCreatedAt,
		MasjidTeachersUpdatedAt: m.MasjidTeachersUpdatedAt,
	}
}
