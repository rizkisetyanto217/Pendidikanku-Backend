// internals/features/lembaga/stats/semester_stats/service/semester_stats_service.go
package service

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	model "madinahsalam_backend/internals/features/lembaga/stats/semester_stats/model"
)

type SemesterStatsService struct{}

func NewSemesterStatsService() *SemesterStatsService { return &SemesterStatsService{} }

// Tentukan rentang semester kalender dari tanggal anchor
func semesterRangeFor(anchor time.Time) (time.Time, time.Time) {
	// pakai UTC supaya konsisten dengan kolom DATE
	anchor = anchor.UTC()
	y := anchor.Year()
	if anchor.Month() <= 6 {
		return time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC),
			time.Date(y, time.June, 30, 23, 59, 59, 0, time.UTC)
	}
	return time.Date(y, time.July, 1, 0, 0, 0, 0, time.UTC),
		time.Date(y, time.December, 31, 23, 59, 59, 0, time.UTC)
}

// Upsert satu row stats (unik via index komposit)
func upsertEmptySemesterStats(tx *gorm.DB, schoolID, userClassID, sectionID uuid.UUID, start, end time.Time) error {
	rec := model.UserClassAttendanceSemesterStatsModel{
		SchoolID:    schoolID,
		UserClassID: userClassID,
		SectionID:   sectionID,
		PeriodStart: start,
		PeriodEnd:   end,

		PresentCount: 0,
		SickCount:    0,
		LeaveCount:   0,
		AbsentCount:  0,

		SumScore:         nil,
		GradePassedCount: nil,
		GradeFailedCount: nil,
		LastAggregatedAt: nil,
	}

	return tx.Clauses(clause.OnConflict{
		// atau: Constraint: "uq_ucass_tenant_userclass_section_period"
		Columns: []clause.Column{
			{Name: "user_class_attendance_semester_stats_school_id"},
			{Name: "user_class_attendance_semester_stats_user_class_id"},
			{Name: "user_class_attendance_semester_stats_section_id"},
			{Name: "user_class_attendance_semester_stats_period_start"},
			{Name: "user_class_attendance_semester_stats_period_end"},
		},
		DoNothing: true,
	}).Create(&rec).Error
}

// Publik: 1 user_class & section, semester dihitung dari anchor
func (s *SemesterStatsService) EnsureSemesterStatsForUserClassWithAnchor(
	tx *gorm.DB, schoolID, userClassID, sectionID uuid.UUID, anchor time.Time,
) error {
	if anchor.IsZero() {
		anchor = time.Now()
	}
	start, end := semesterRangeFor(anchor)
	return upsertEmptySemesterStats(tx, schoolID, userClassID, sectionID, start, end)
}

// Publik: 1 user_class & section, anchor = waktu saat ini (fallback lama)
func (s *SemesterStatsService) EnsureSemesterStatsForUserClass(
	tx *gorm.DB, schoolID, userClassID, sectionID uuid.UUID,
) error {
	return s.EnsureSemesterStatsForUserClassWithAnchor(tx, schoolID, userClassID, sectionID, time.Now())
}

// Opsional: semua user di section → anchor pakai hari ini
func (s *SemesterStatsService) EnsureSemesterStatsForSection(
	tx *gorm.DB, schoolID, sectionID uuid.UUID,
) error {
	start, end := semesterRangeFor(time.Now())

	type ucRow struct {
		ID uuid.UUID `gorm:"column:user_classes_id"`
	}
	var ucs []ucRow
	if err := tx.Table("user_classes").
		Where("user_classes_school_id = ? AND user_classes_section_id = ?", schoolID, sectionID).
		Select("user_classes_id").
		Find(&ucs).Error; err != nil {
		return err
	}

	for _, r := range ucs {
		if err := upsertEmptySemesterStats(tx, schoolID, r.ID, sectionID, start, end); err != nil {
			return err
		}
	}
	return nil
}

// ... (imports & type SemesterStatsService, semesterRangeFor, upsertEmptySemesterStats, EnsureSemesterStatsForUserClassWithAnchor sudah ada)

// Ensure semua murid di section punya baris stats utk semester tanggal anchor
func (s *SemesterStatsService) EnsureSemesterStatsForSectionAtAnchor(
	tx *gorm.DB, schoolID, sectionID uuid.UUID, anchor time.Time,
) error {
	if anchor.IsZero() {
		anchor = time.Now()
	}
	start, end := semesterRangeFor(anchor)

	// Ambil semua user_class yang sedang ter-assign ke section pada tanggal anchor
	type row struct {
		UserClassID uuid.UUID `gorm:"column:user_class_sections_user_class_id"`
	}
	var rows []row
	if err := tx.Table("user_class_sections").
		Select("user_class_sections_user_class_id").
		Where("user_class_sections_school_id = ?", schoolID).
		Where("user_class_sections_section_id = ?", sectionID).
		Where("user_class_sections_assigned_at <= ?", anchor).
		Where("(user_class_sections_unassigned_at IS NULL OR user_class_sections_unassigned_at > ?)", anchor).
		Find(&rows).Error; err != nil {
		return err
	}

	for _, r := range rows {
		if err := upsertEmptySemesterStats(tx, schoolID, r.UserClassID, sectionID, start, end); err != nil {
			return err
		}
	}
	return nil
}

// (Opsional, dipakai saat input status per murid)
// Bump counter aman (anti minus) untuk 1 user_class + section di semester yg memuat anchor.
// BumpCounters: pastikan baris semester ada, lalu increment counter aman (anti-minus)
func (s *SemesterStatsService) BumpCounters(
	tx *gorm.DB,
	schoolID, userClassID, sectionID uuid.UUID,
	anchor time.Time,
	dPresent, dSick, dLeave, dAbsent int,
	dSumScore *int, dPassed *int, dFailed *int,
) error {
	if anchor.IsZero() {
		anchor = time.Now()
	}
	// pastikan row ada (idempotent)
	if err := s.EnsureSemesterStatsForUserClassWithAnchor(tx, schoolID, userClassID, sectionID, anchor); err != nil {
		return err
	}

	set := map[string]any{
		"user_class_attendance_semester_stats_present_count": gorm.Expr(
			"CASE WHEN user_class_attendance_semester_stats_present_count + ? < 0 THEN 0 ELSE user_class_attendance_semester_stats_present_count + ? END",
			dPresent, dPresent,
		),
		"user_class_attendance_semester_stats_sick_count": gorm.Expr(
			"CASE WHEN user_class_attendance_semester_stats_sick_count + ? < 0 THEN 0 ELSE user_class_attendance_semester_stats_sick_count + ? END",
			dSick, dSick,
		),
		"user_class_attendance_semester_stats_leave_count": gorm.Expr(
			"CASE WHEN user_class_attendance_semester_stats_leave_count + ? < 0 THEN 0 ELSE user_class_attendance_semester_stats_leave_count + ? END",
			dLeave, dLeave,
		),
		"user_class_attendance_semester_stats_absent_count": gorm.Expr(
			"CASE WHEN user_class_attendance_semester_stats_absent_count + ? < 0 THEN 0 ELSE user_class_attendance_semester_stats_absent_count + ? END",
			dAbsent, dAbsent,
		),
		"user_class_attendance_semester_stats_last_aggregated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		"user_class_attendance_semester_stats_updated_at":         gorm.Expr("CURRENT_TIMESTAMP"),
	}
	if dSumScore != nil {
		set["user_class_attendance_semester_stats_sum_score"] = gorm.Expr(
			"COALESCE(user_class_attendance_semester_stats_sum_score,0) + ?",
			*dSumScore,
		)
	}
	if dPassed != nil {
		set["user_class_attendance_semester_stats_grade_passed_count"] = gorm.Expr(
			"COALESCE(user_class_attendance_semester_stats_grade_passed_count,0) + ?",
			*dPassed,
		)
	}
	if dFailed != nil {
		set["user_class_attendance_semester_stats_grade_failed_count"] = gorm.Expr(
			"COALESCE(user_class_attendance_semester_stats_grade_failed_count,0) + ?",
			*dFailed,
		)
	}

	return tx.Table("user_class_attendance_semester_stats").
		Where("user_class_attendance_semester_stats_school_id = ?", schoolID).
		Where("user_class_attendance_semester_stats_user_class_id = ?", userClassID).
		Where("user_class_attendance_semester_stats_section_id = ?", sectionID).
		Where("?::date BETWEEN user_class_attendance_semester_stats_period_start AND user_class_attendance_semester_stats_period_end", anchor).
		Updates(set).Error
}
