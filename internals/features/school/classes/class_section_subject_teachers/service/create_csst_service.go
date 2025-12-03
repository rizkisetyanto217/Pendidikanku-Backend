// file: internals/features/lembaga/class_section_subject_teachers/service/csst_service.go
package service

import (
	"context"
	"strings"

	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"
	dto "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ptrAttendance(m csstModel.AttendanceEntryMode) *csstModel.AttendanceEntryMode { return &m }

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// small helper: normalisasi attendance mode → selalu salah satu dari 3 nilai valid
func normalizeAttendanceModeStr(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(csstModel.AttendanceEntryTeacherOnly):
		return string(csstModel.AttendanceEntryTeacherOnly)
	case string(csstModel.AttendanceEntryStudentOnly):
		return string(csstModel.AttendanceEntryStudentOnly)
	case string(csstModel.AttendanceEntryBoth):
		return string(csstModel.AttendanceEntryBoth)
	default:
		// fallback global kalau entah kenapa kosong / invalid
		return string(csstModel.AttendanceEntryBoth)
	}
}

// hitung effective attendance mode untuk CSST:
// 1) kalau payload cache diisi → pakai itu
// 2) else pakai default dari school
// 3) kalau school kosong/invalid → fallback "both"
func effectiveAttendanceMode(
	school *schoolModel.SchoolModel,
	payload *dto.CreateClassSectionSubjectTeacherRequest,
) csstModel.AttendanceEntryMode {
	// 1) dari payload (CSST override)
	if payload != nil &&
		payload.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache != nil &&
		*payload.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache != "" {

		raw := string(*payload.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache)
		norm := normalizeAttendanceModeStr(raw)
		return csstModel.AttendanceEntryMode(norm)
	}

	// 2) dari default school
	if school != nil {
		raw := string(school.SchoolDefaultAttendanceEntryMode) // enum di model sekolah
		if strings.TrimSpace(raw) != "" {
			norm := normalizeAttendanceModeStr(raw)
			return csstModel.AttendanceEntryMode(norm)
		}
	}

	// 3) fallback hard
	return csstModel.AttendanceEntryBoth
}

func (s *Service) CreateCSST(
	ctx context.Context,
	db *gorm.DB,
	schoolID uuid.UUID,
	payload *dto.CreateClassSectionSubjectTeacherRequest,
) (*csstModel.ClassSectionSubjectTeacherModel, error) {

	if db == nil {
		db = s.DB
	}

	var school schoolModel.SchoolModel
	if err := db.WithContext(ctx).
		Where("school_id = ? AND school_deleted_at IS NULL", schoolID).
		First(&school).Error; err != nil {
		return nil, err
	}

	csstModelValue := payload.ToModel()
	csstModelValue.ClassSectionSubjectTeacherSchoolID = schoolID

	// 🔹 effective attendance entry mode (cache)
	eff := effectiveAttendanceMode(&school, payload)
	csstModelValue.ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache = ptrAttendance(eff)

	if err := db.WithContext(ctx).Create(&csstModelValue).Error; err != nil {
		return nil, err
	}

	return &csstModelValue, nil
}
