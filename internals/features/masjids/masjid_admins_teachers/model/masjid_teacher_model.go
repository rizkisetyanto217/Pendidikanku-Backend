package model

import (
	"time"

	"gorm.io/gorm"
)

type MasjidTeacher struct {
	MasjidTeachersID       string         `gorm:"column:masjid_teachers_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"masjid_teachers_id"`
	MasjidTeachersMasjidID string         `gorm:"column:masjid_teachers_masjid_id;type:uuid;not null" json:"masjid_teachers_masjid_id"`
	MasjidTeachersUserID   string         `gorm:"column:masjid_teachers_user_id;type:uuid;not null" json:"masjid_teachers_user_id"`

	// timestamps
	MasjidTeachersCreatedAt time.Time      `gorm:"column:masjid_teachers_created_at;autoCreateTime" json:"masjid_teachers_created_at"`
	MasjidTeachersUpdatedAt time.Time      `gorm:"column:masjid_teachers_updated_at;autoUpdateTime" json:"masjid_teachers_updated_at"`
	MasjidTeachersDeletedAt gorm.DeletedAt `gorm:"column:masjid_teachers_deleted_at;index" json:"masjid_teachers_deleted_at"`
}

// TableName override
func (MasjidTeacher) TableName() string {
	return "masjid_teachers"
}
