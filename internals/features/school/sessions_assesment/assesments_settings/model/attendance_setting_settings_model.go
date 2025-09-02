package model

import (
	"time"

	"github.com/google/uuid"
)

type ClassAttendanceSetting struct {
	ClassAttendanceSettingID       uuid.UUID `gorm:"column:class_attendance_setting_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"class_attendance_setting_id"`
	ClassAttendanceSettingMasjidID uuid.UUID `gorm:"column:class_attendance_setting_masjid_id;not null" json:"class_attendance_setting_masjid_id"`

	// switches
	ClassAttendanceSettingEnableScore             bool `gorm:"column:class_attendance_setting_enable_score;not null;default:false" json:"class_attendance_setting_enable_score"`
	ClassAttendanceSettingRequireScore            bool `gorm:"column:class_attendance_setting_require_score;not null;default:false" json:"class_attendance_setting_require_score"`
	ClassAttendanceSettingEnableGradePassed       bool `gorm:"column:class_attendance_setting_enable_grade_passed;not null;default:false" json:"class_attendance_setting_enable_grade_passed"`
	ClassAttendanceSettingRequireGradePassed      bool `gorm:"column:class_attendance_setting_require_grade_passed;not null;default:false" json:"class_attendance_setting_require_grade_passed"`
	ClassAttendanceSettingEnableMaterialPersonal  bool `gorm:"column:class_attendance_setting_enable_material_personal;not null;default:false" json:"class_attendance_setting_enable_material_personal"`
	ClassAttendanceSettingRequireMaterialPersonal bool `gorm:"column:class_attendance_setting_require_material_personal;not null;default:false" json:"class_attendance_setting_require_material_personal"`
	ClassAttendanceSettingEnablePersonalNote      bool `gorm:"column:class_attendance_setting_enable_personal_note;not null;default:false" json:"class_attendance_setting_enable_personal_note"`
	ClassAttendanceSettingRequirePersonalNote     bool `gorm:"column:class_attendance_setting_require_personal_note;not null;default:false" json:"class_attendance_setting_require_personal_note"`
	ClassAttendanceSettingEnableMemorization      bool `gorm:"column:class_attendance_setting_enable_memorization;not null;default:false" json:"class_attendance_setting_enable_memorization"`
	ClassAttendanceSettingRequireMemorization     bool `gorm:"column:class_attendance_setting_require_memorization;not null;default:false" json:"class_attendance_setting_require_memorization"`
	ClassAttendanceSettingEnableHomework          bool `gorm:"column:class_attendance_setting_enable_homework;not null;default:false" json:"class_attendance_setting_enable_homework"`
	ClassAttendanceSettingRequireHomework         bool `gorm:"column:class_attendance_setting_require_homework;not null;default:false" json:"class_attendance_setting_require_homework"`

	// audit minimum
	ClassAttendanceSettingCreatedAt time.Time `gorm:"column:class_attendance_setting_created_at;autoCreateTime" json:"class_attendance_setting_created_at"`
}

// TableName override
func (ClassAttendanceSetting) TableName() string {
	return "class_attendance_settings"
}
