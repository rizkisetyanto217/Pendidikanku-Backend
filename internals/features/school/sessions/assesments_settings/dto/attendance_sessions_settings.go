package dto

import (
	"masjidku_backend/internals/features/school/sessions/assesments_settings/model"
	"time"

	"github.com/google/uuid"
)

// Dipakai untuk request & response
type ClassAttendanceSettingDTO struct {
    ClassAttendanceSettingID       uuid.UUID `json:"class_attendance_setting_id,omitempty"`
    ClassAttendanceSettingMasjidID uuid.UUID `json:"class_attendance_setting_masjid_id,omitempty"`

    ClassAttendanceSettingEnableScore             bool `json:"class_attendance_setting_enable_score"`
    ClassAttendanceSettingRequireScore            bool `json:"class_attendance_setting_require_score"`

    ClassAttendanceSettingEnableGradePassed       bool `json:"class_attendance_setting_enable_grade_passed"`
    ClassAttendanceSettingRequireGradePassed      bool `json:"class_attendance_setting_require_grade_passed"`

    ClassAttendanceSettingEnableMaterialPersonal  bool `json:"class_attendance_setting_enable_material_personal"`
    ClassAttendanceSettingRequireMaterialPersonal bool `json:"class_attendance_setting_require_material_personal"`

    ClassAttendanceSettingEnablePersonalNote      bool `json:"class_attendance_setting_enable_personal_note"`
    ClassAttendanceSettingRequirePersonalNote     bool `json:"class_attendance_setting_require_personal_note"`

    ClassAttendanceSettingEnableMemorization      bool `json:"class_attendance_setting_enable_memorization"`
    ClassAttendanceSettingRequireMemorization     bool `json:"class_attendance_setting_require_memorization"`

    ClassAttendanceSettingEnableHomework          bool `json:"class_attendance_setting_enable_homework"`
    ClassAttendanceSettingRequireHomework         bool `json:"class_attendance_setting_require_homework"`

    ClassAttendanceSettingCreatedAt time.Time `json:"class_attendance_setting_created_at,omitempty"`
}

/* ====================
   Converter Functions
   ==================== */

// FromModel -> DTO (untuk response GET/POST/PUT)
func FromModel(m *model.ClassAttendanceSetting) *ClassAttendanceSettingDTO {
	if m == nil {
		return nil
	}
	return &ClassAttendanceSettingDTO{
		ClassAttendanceSettingID:       m.ClassAttendanceSettingID,
		ClassAttendanceSettingMasjidID: m.ClassAttendanceSettingMasjidID,

		ClassAttendanceSettingEnableScore:             m.ClassAttendanceSettingEnableScore,
		ClassAttendanceSettingRequireScore:            m.ClassAttendanceSettingRequireScore,
		ClassAttendanceSettingEnableGradePassed:       m.ClassAttendanceSettingEnableGradePassed,
		ClassAttendanceSettingRequireGradePassed:      m.ClassAttendanceSettingRequireGradePassed,
		ClassAttendanceSettingEnableMaterialPersonal:  m.ClassAttendanceSettingEnableMaterialPersonal,
		ClassAttendanceSettingRequireMaterialPersonal: m.ClassAttendanceSettingRequireMaterialPersonal,
		ClassAttendanceSettingEnablePersonalNote:      m.ClassAttendanceSettingEnablePersonalNote,
		ClassAttendanceSettingRequirePersonalNote:     m.ClassAttendanceSettingRequirePersonalNote,
		ClassAttendanceSettingEnableMemorization:      m.ClassAttendanceSettingEnableMemorization,
		ClassAttendanceSettingRequireMemorization:     m.ClassAttendanceSettingRequireMemorization,
		ClassAttendanceSettingEnableHomework:          m.ClassAttendanceSettingEnableHomework,
		ClassAttendanceSettingRequireHomework:         m.ClassAttendanceSettingRequireHomework,

		ClassAttendanceSettingCreatedAt: m.ClassAttendanceSettingCreatedAt,
	}
}

// ToModel -> Model (dipakai controller untuk Create/Update)
// Catatan: controller override MasjidID dari path; untuk POST set ID = uuid.Nil.
func (d *ClassAttendanceSettingDTO) ToModel() *model.ClassAttendanceSetting {
	if d == nil {
		return nil
	}
	return &model.ClassAttendanceSetting{
		ClassAttendanceSettingID:       d.ClassAttendanceSettingID,
		ClassAttendanceSettingMasjidID: d.ClassAttendanceSettingMasjidID,

		ClassAttendanceSettingEnableScore:             d.ClassAttendanceSettingEnableScore,
		ClassAttendanceSettingRequireScore:            d.ClassAttendanceSettingRequireScore,
		ClassAttendanceSettingEnableGradePassed:       d.ClassAttendanceSettingEnableGradePassed,
		ClassAttendanceSettingRequireGradePassed:      d.ClassAttendanceSettingRequireGradePassed,
		ClassAttendanceSettingEnableMaterialPersonal:  d.ClassAttendanceSettingEnableMaterialPersonal,
		ClassAttendanceSettingRequireMaterialPersonal: d.ClassAttendanceSettingRequireMaterialPersonal,
		ClassAttendanceSettingEnablePersonalNote:      d.ClassAttendanceSettingEnablePersonalNote,
		ClassAttendanceSettingRequirePersonalNote:     d.ClassAttendanceSettingRequirePersonalNote,
		ClassAttendanceSettingEnableMemorization:      d.ClassAttendanceSettingEnableMemorization,
		ClassAttendanceSettingRequireMemorization:     d.ClassAttendanceSettingRequireMemorization,
		ClassAttendanceSettingEnableHomework:          d.ClassAttendanceSettingEnableHomework,
		ClassAttendanceSettingRequireHomework:         d.ClassAttendanceSettingRequireHomework,

		// created_at dikelola DB (autoCreateTime); boleh diabaikan saat write
		ClassAttendanceSettingCreatedAt: d.ClassAttendanceSettingCreatedAt,
	}
}
