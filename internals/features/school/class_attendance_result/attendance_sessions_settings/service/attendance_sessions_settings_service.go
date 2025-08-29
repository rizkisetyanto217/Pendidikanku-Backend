package service

import (
	"errors"
	"fmt"
	"time"

	"masjidku_backend/internals/features/lembaga/stats/semester_stats/service"
	asdto "masjidku_backend/internals/features/school/class_attendance_result/attendance_sessions/dto"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ================== SETTINGS ==================

type AttendanceSetting struct {
	EnableScore, RequireScore                         bool
	EnableGradePassed, RequireGradePassed             bool
	EnableMaterialPersonal, RequireMaterialPersonal   bool
	EnablePersonalNote, RequirePersonalNote           bool
	EnableMemorization, RequireMemorization           bool
	EnableHomework, RequireHomework                   bool
}

type Service struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Service { return &Service{DB: db} }

// Get settings by masjid; tx boleh nil → pakai s.DB
func (s *Service) GetSettings(masjidID uuid.UUID, tx *gorm.DB) (AttendanceSetting, error) {
	db := s.DB
	if tx != nil {
		db = tx
	}
	type row struct {
		EnableScore             bool `gorm:"column:class_attendance_setting_enable_score"`
		RequireScore            bool `gorm:"column:class_attendance_setting_require_score"`
		EnableGradePassed       bool `gorm:"column:class_attendance_setting_enable_grade_passed"`
		RequireGradePassed      bool `gorm:"column:class_attendance_setting_require_grade_passed"`
		EnableMaterialPersonal  bool `gorm:"column:class_attendance_setting_enable_material_personal"`
		RequireMaterialPersonal bool `gorm:"column:class_attendance_setting_require_material_personal"`
		EnablePersonalNote      bool `gorm:"column:class_attendance_setting_enable_personal_note"`
		RequirePersonalNote     bool `gorm:"column:class_attendance_setting_require_personal_note"`
		EnableMemorization      bool `gorm:"column:class_attendance_setting_enable_memorization"`
		RequireMemorization     bool `gorm:"column:class_attendance_setting_require_memorization"`
		EnableHomework          bool `gorm:"column:class_attendance_setting_enable_homework"`
		RequireHomework         bool `gorm:"column:class_attendance_setting_require_homework"`
	}
	var r row
	err := db.Table("class_attendance_settings").
		Select(`class_attendance_setting_enable_score,
		        class_attendance_setting_require_score,
		        class_attendance_setting_enable_grade_passed,
		        class_attendance_setting_require_grade_passed,
		        class_attendance_setting_enable_material_personal,
		        class_attendance_setting_require_material_personal,
		        class_attendance_setting_enable_personal_note,
		        class_attendance_setting_require_personal_note,
		        class_attendance_setting_enable_memorization,
		        class_attendance_setting_require_memorization,
		        class_attendance_setting_enable_homework,
		        class_attendance_setting_require_homework`).
		Where("class_attendance_setting_masjid_id = ?", masjidID).
		Limit(1).Take(&r).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// default: semua false (no error)
		return AttendanceSetting{}, nil
	}
	if err != nil {
		return AttendanceSetting{}, err
	}

	return AttendanceSetting{
		EnableScore:             r.EnableScore,
		RequireScore:            r.RequireScore && r.EnableScore,
		EnableGradePassed:       r.EnableGradePassed,
		RequireGradePassed:      r.RequireGradePassed && r.EnableGradePassed,
		EnableMaterialPersonal:  r.EnableMaterialPersonal,
		RequireMaterialPersonal: r.RequireMaterialPersonal && r.EnableMaterialPersonal,
		EnablePersonalNote:      r.EnablePersonalNote,
		RequirePersonalNote:     r.RequirePersonalNote && r.EnablePersonalNote,
		EnableMemorization:      r.EnableMemorization,
		RequireMemorization:     r.RequireMemorization && r.EnableMemorization,
		EnableHomework:          r.EnableHomework,
		RequireHomework:         r.RequireHomework && r.EnableHomework,
	}, nil
}

// ================== NORMALIZE ==================

func (s *Service) NormalizeCreate(req *asdto.CreateUserClassAttendanceSessionRequest, set AttendanceSetting) error {
	// SCORE
	if !set.EnableScore {
		req.UserClassAttendanceSessionsScore = nil
	} else if set.RequireScore && req.UserClassAttendanceSessionsScore == nil {
		return fmt.Errorf("score wajib diisi")
	}
	// GRADE PASSED
	if !set.EnableGradePassed {
		req.UserClassAttendanceSessionsGradePassed = nil
	} else if set.RequireGradePassed && req.UserClassAttendanceSessionsGradePassed == nil {
		return fmt.Errorf("grade_passed wajib diisi (true/false)")
	}
	// MATERIAL PERSONAL
	if !set.EnableMaterialPersonal {
		req.UserClassAttendanceSessionsMaterialPersonal = nil
	} else if set.RequireMaterialPersonal && (req.UserClassAttendanceSessionsMaterialPersonal == nil || *req.UserClassAttendanceSessionsMaterialPersonal == "") {
		return fmt.Errorf("material_personal wajib diisi")
	}
	// PERSONAL NOTE
	if !set.EnablePersonalNote {
		req.UserClassAttendanceSessionsPersonalNote = nil
	} else if set.RequirePersonalNote && (req.UserClassAttendanceSessionsPersonalNote == nil || *req.UserClassAttendanceSessionsPersonalNote == "") {
		return fmt.Errorf("personal_note wajib diisi")
	}
	// MEMORIZATION
	if !set.EnableMemorization {
		req.UserClassAttendanceSessionsMemorization = nil
	} else if set.RequireMemorization && (req.UserClassAttendanceSessionsMemorization == nil || *req.UserClassAttendanceSessionsMemorization == "") {
		return fmt.Errorf("memorization wajib diisi")
	}
	// HOMEWORK
	if !set.EnableHomework {
		req.UserClassAttendanceSessionsHomework = nil
	} else if set.RequireHomework && (req.UserClassAttendanceSessionsHomework == nil || *req.UserClassAttendanceSessionsHomework == "") {
		return fmt.Errorf("homework wajib diisi")
	}
	return nil
}

// Kembalikan map updates yang sudah dibersihkan sesuai setting.
func (s *Service) NormalizeUpdate(req *asdto.UpdateUserClassAttendanceSessionRequest, set AttendanceSetting) (map[string]any, error) {
	updates := map[string]any{}

	if req.UserClassAttendanceSessionsAttendanceStatus != nil {
		updates["user_class_attendance_sessions_attendance_status"] = *req.UserClassAttendanceSessionsAttendanceStatus
	}

	// SCORE
	if !set.EnableScore {
		// abaikan kalau fitur off
	} else if req.UserClassAttendanceSessionsScore != nil {
		updates["user_class_attendance_sessions_score"] = *req.UserClassAttendanceSessionsScore
		if set.RequireScore && *req.UserClassAttendanceSessionsScore == 0 && req.UserClassAttendanceSessionsScore == nil {
			return nil, fmt.Errorf("score wajib diisi")
		}
	}

	// GRADE PASSED
	if set.EnableGradePassed && req.UserClassAttendanceSessionsGradePassed != nil {
		updates["user_class_attendance_sessions_grade_passed"] = *req.UserClassAttendanceSessionsGradePassed
	}

	// MATERIAL PERSONAL
	if set.EnableMaterialPersonal && req.UserClassAttendanceSessionsMaterialPersonal != nil {
		val := *req.UserClassAttendanceSessionsMaterialPersonal
		if set.RequireMaterialPersonal && val == "" {
			return nil, fmt.Errorf("material_personal wajib diisi")
		}
		updates["user_class_attendance_sessions_material_personal"] = val
	}

	// PERSONAL NOTE
	if set.EnablePersonalNote && req.UserClassAttendanceSessionsPersonalNote != nil {
		val := *req.UserClassAttendanceSessionsPersonalNote
		if set.RequirePersonalNote && val == "" {
			return nil, fmt.Errorf("personal_note wajib diisi")
		}
		updates["user_class_attendance_sessions_personal_note"] = val
	}

	// MEMORIZATION
	if set.EnableMemorization && req.UserClassAttendanceSessionsMemorization != nil {
		val := *req.UserClassAttendanceSessionsMemorization
		if set.RequireMemorization && val == "" {
			return nil, fmt.Errorf("memorization wajib diisi")
		}
		updates["user_class_attendance_sessions_memorization"] = val
	}

	// HOMEWORK
	if set.EnableHomework && req.UserClassAttendanceSessionsHomework != nil {
		val := *req.UserClassAttendanceSessionsHomework
		if set.RequireHomework && val == "" {
			return nil, fmt.Errorf("homework wajib diisi")
		}
		updates["user_class_attendance_sessions_homework"] = val
	}

	return updates, nil
}

// ================== COUNTERS ==================

type Counters struct {
	DPresent, DSick, DLeave, DAbsent int
	DSum                              *int
	DPassed, DFailed                  *int
}

// Hitung delta counter sesuai setting (hormati fitur off)
func (s *Service) ComputeCountersOnCreate(req *asdto.CreateUserClassAttendanceSessionRequest, set AttendanceSetting) Counters {
	var c Counters
	switch req.UserClassAttendanceSessionsAttendanceStatus {
	case "present":
		c.DPresent = 1
	case "sick":
		c.DSick = 1
	case "leave":
		c.DLeave = 1
	case "absent":
		c.DAbsent = 1
	}
	if set.EnableScore && req.UserClassAttendanceSessionsScore != nil {
		c.DSum = req.UserClassAttendanceSessionsScore
	}
	if set.EnableGradePassed && req.UserClassAttendanceSessionsGradePassed != nil {
		one := 1
		if *req.UserClassAttendanceSessionsGradePassed {
			c.DPassed = &one
		} else {
			c.DFailed = &one
		}
	}
	return c
}

// Opsional: wrapper langsung update semester stats
func (s *Service) BumpSemesterStats(
	db *gorm.DB,
	masjidID, userClassID, sectionID uuid.UUID,
	anchorAt any,
	c Counters,
) error {
	at, ok := anchorAt.(time.Time)
	if !ok || at.IsZero() {
		at = time.Now()
	}
	semSvc := service.NewSemesterStatsService()
	return semSvc.BumpCounters(
		db, masjidID, userClassID, sectionID, at,
		c.DPresent, c.DSick, c.DLeave, c.DAbsent,
		c.DSum, c.DPassed, c.DFailed,
	)
}
