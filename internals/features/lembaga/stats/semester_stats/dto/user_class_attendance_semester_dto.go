// internals/features/lembaga/class_sections/attendance_semester_stats/dto/user_class_attendance_semester_stats_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	m "madinahsalam_backend/internals/features/lembaga/stats/semester_stats/model"
)

/* ===================== REQUESTS ===================== */

// Upsert manual (mis. hasil ETL atau admin input)
// Catatan: kolom generated (total_sessions, avg_score) tidak perlu diisi.
type UpsertSemesterStatsRequest struct {
	SchoolID    uuid.UUID `json:"school_id" validate:"required"`
	UserClassID uuid.UUID `json:"user_class_id" validate:"required"`
	SectionID   uuid.UUID `json:"section_id" validate:"required"`

	PeriodStart time.Time `json:"period_start" validate:"required"`
	PeriodEnd   time.Time `json:"period_end" validate:"required,gtefield=PeriodStart"`

	PresentCount int `json:"present_count" validate:"gte=0"`
	SickCount    int `json:"sick_count" validate:"gte=0"`
	LeaveCount   int `json:"leave_count" validate:"gte=0"`
	AbsentCount  int `json:"absent_count" validate:"gte=0"`

	SumScore         *int       `json:"sum_score" validate:"omitempty,gte=0"`
	GradePassedCount *int       `json:"grade_passed_count" validate:"omitempty,gte=0"`
	GradeFailedCount *int       `json:"grade_failed_count" validate:"omitempty,gte=0"`
	LastAggregatedAt *time.Time `json:"last_aggregated_at" validate:"omitempty"`
}

// Query list/pencarian
type ListSemesterStatsQuery struct {
	SchoolID    *uuid.UUID `query:"school_id" validate:"omitempty"`
	UserClassID *uuid.UUID `query:"user_class_id" validate:"omitempty"`
	SectionID   *uuid.UUID `query:"section_id" validate:"omitempty"`
	Start       *time.Time `query:"start" validate:"omitempty"`
	End         *time.Time `query:"end" validate:"omitempty"`

	Limit  int `query:"limit" validate:"omitempty,gte=1,lte=100"`
	Offset int `query:"offset" validate:"omitempty,gte=0"`
}

/* ===================== RESPONSES ===================== */

type SemesterStatsResponse struct {
	ID          uuid.UUID `json:"id"`
	SchoolID    uuid.UUID `json:"school_id"`
	UserClassID uuid.UUID `json:"user_class_id"`
	SectionID   uuid.UUID `json:"section_id"`

	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`

	PresentCount int `json:"present_count"`
	SickCount    int `json:"sick_count"`
	LeaveCount   int `json:"leave_count"`
	AbsentCount  int `json:"absent_count"`

	TotalSessions int      `json:"total_sessions"` // generated
	SumScore      *int     `json:"sum_score"`
	AvgScore      *float64 `json:"avg_score"` // generated

	GradePassedCount *int       `json:"grade_passed_count"`
	GradeFailedCount *int       `json:"grade_failed_count"`
	LastAggregatedAt *time.Time `json:"last_aggregated_at"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// Untuk response berhalaman (opsional)
type SemesterStatsListResponse struct {
	Items []SemesterStatsResponse `json:"items"`
	Total int64                   `json:"total"`
}

/* ===================== MAPPERS ===================== */

func FromModel(x m.UserClassAttendanceSemesterStatsModel) SemesterStatsResponse {
	return SemesterStatsResponse{
		ID:               x.ID,
		SchoolID:         x.SchoolID,
		UserClassID:      x.UserClassID,
		SectionID:        x.SectionID,
		PeriodStart:      x.PeriodStart,
		PeriodEnd:        x.PeriodEnd,
		PresentCount:     x.PresentCount,
		SickCount:        x.SickCount,
		LeaveCount:       x.LeaveCount,
		AbsentCount:      x.AbsentCount,
		TotalSessions:    x.TotalSessions, // generated
		SumScore:         x.SumScore,
		AvgScore:         x.AvgScore, // generated
		GradePassedCount: x.GradePassedCount,
		GradeFailedCount: x.GradeFailedCount,
		LastAggregatedAt: x.LastAggregatedAt,
		CreatedAt:        x.CreatedAt,
		UpdatedAt:        x.UpdatedAt,
	}
}

func FromModels(list []m.UserClassAttendanceSemesterStatsModel, total int64) SemesterStatsListResponse {
	out := make([]SemesterStatsResponse, 0, len(list))
	for _, it := range list {
		out = append(out, FromModel(it))
	}
	return SemesterStatsListResponse{Items: out, Total: total}
}

func (r UpsertSemesterStatsRequest) ToModel(existingID *uuid.UUID) m.UserClassAttendanceSemesterStatsModel {
	mx := m.UserClassAttendanceSemesterStatsModel{
		SchoolID:         r.SchoolID,
		UserClassID:      r.UserClassID,
		SectionID:        r.SectionID,
		PeriodStart:      r.PeriodStart,
		PeriodEnd:        r.PeriodEnd,
		PresentCount:     r.PresentCount,
		SickCount:        r.SickCount,
		LeaveCount:       r.LeaveCount,
		AbsentCount:      r.AbsentCount,
		SumScore:         r.SumScore,
		GradePassedCount: r.GradePassedCount,
		GradeFailedCount: r.GradeFailedCount,
		LastAggregatedAt: r.LastAggregatedAt,
	}
	if existingID != nil {
		mx.ID = *existingID
	}
	return mx
}
