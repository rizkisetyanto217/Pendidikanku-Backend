// internals/features/lembaga/class_sections/attendance_semester_stats/model/user_class_attendance_semester_stats_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type UserClassAttendanceSemesterStatsModel struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_attendance_semester_stats_id"`

	SchoolID    uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_semester_stats_school_id"`
	UserClassID uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_semester_stats_user_class_id"`
	SectionID   uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_semester_stats_section_id"`

	PeriodStart time.Time `gorm:"type:date;not null;column:user_class_attendance_semester_stats_period_start"`
	PeriodEnd   time.Time `gorm:"type:date;not null;column:user_class_attendance_semester_stats_period_end"`

	PresentCount int `gorm:"not null;default:0;column:user_class_attendance_semester_stats_present_count"`
	SickCount    int `gorm:"not null;default:0;column:user_class_attendance_semester_stats_sick_count"`
	LeaveCount   int `gorm:"not null;default:0;column:user_class_attendance_semester_stats_leave_count"`
	AbsentCount  int `gorm:"not null;default:0;column:user_class_attendance_semester_stats_absent_count"`

	// generated columns (read-only)
	TotalSessions int      `gorm:"->;column:user_class_attendance_semester_stats_total_sessions"`
	AvgScore      *float64 `gorm:"->;column:user_class_attendance_semester_stats_avg_score"`

	// agregat score & kelulusan
	SumScore         *int `gorm:"column:user_class_attendance_semester_stats_sum_score"`
	GradePassedCount *int `gorm:"column:user_class_attendance_semester_stats_grade_passed_count"`
	GradeFailedCount *int `gorm:"column:user_class_attendance_semester_stats_grade_failed_count"`

	// ETL marker
	LastAggregatedAt *time.Time `gorm:"column:user_class_attendance_semester_stats_last_aggregated_at"`

	CreatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;column:user_class_attendance_semester_stats_created_at"`
	UpdatedAt *time.Time `gorm:"column:user_class_attendance_semester_stats_updated_at"`
}

func (UserClassAttendanceSemesterStatsModel) TableName() string {
	return "user_class_attendance_semester_stats"
}
