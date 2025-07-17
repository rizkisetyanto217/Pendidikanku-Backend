package model

import (
	"time"
)

type MasjidTeacher struct {
	MasjidTeachersID        string    `gorm:"column:masjid_teachers_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"masjid_teachers_id"`
	MasjidTeachersMasjidID  string    `gorm:"column:masjid_teachers_masjid_id;type:uuid;not null" json:"masjid_teachers_masjid_id"`
	MasjidTeachersUserID    string    `gorm:"column:masjid_teachers_user_id;type:uuid;not null" json:"masjid_teachers_user_id"`
	MasjidTeachersCreatedAt time.Time `gorm:"column:masjid_teachers_created_at;autoCreateTime" json:"masjid_teachers_created_at"`
	MasjidTeachersUpdatedAt time.Time `gorm:"column:masjid_teachers_updated_at;autoUpdateTime" json:"masdjid_teachers_updated_at"`
}

// TableName override
func (MasjidTeacher) TableName() string {
	return "masjid_teachers"
}
