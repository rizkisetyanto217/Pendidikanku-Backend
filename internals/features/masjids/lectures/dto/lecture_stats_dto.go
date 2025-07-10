package dto

import (
	"masjidku_backend/internals/features/masjids/lectures/model"

	"github.com/google/uuid"
)

// Request: biasanya hanya perlu LectureID untuk inisialisasi
type LectureStatsRequest struct {
	LectureStatsLectureID         uuid.UUID `json:"lecture_stats_lecture_id"`
	LectureStatsTotalParticipants int       `json:"lecture_stats_total_participants"`
	LectureStatsAverageGrade      float64   `json:"lecture_stats_average_grade"`
}

// Response: menyesuaikan semua field dengan model
type LectureStatsResponse struct {
	LectureStatsID                uuid.UUID `json:"lecture_stats_id"`
	LectureStatsLectureID         uuid.UUID `json:"lecture_stats_lecture_id"`
	LectureStatsTotalParticipants int       `json:"lecture_stats_total_participants"`
	LectureStatsAverageGrade      float64   `json:"lecture_stats_average_grade"`
	LectureStatsUpdatedAt         string    `json:"lecture_stats_updated_at"`
}

// Convert request → model
func (r *LectureStatsRequest) ToModel() *model.LectureStatsModel {
	return &model.LectureStatsModel{
		LectureStatsLectureID:         r.LectureStatsLectureID,
		LectureStatsTotalParticipants: r.LectureStatsTotalParticipants,
		LectureStatsAverageGrade:      r.LectureStatsAverageGrade,
	}
}

// Convert model → response
func ToLectureStatsResponse(m *model.LectureStatsModel) *LectureStatsResponse {
	return &LectureStatsResponse{
		LectureStatsID:                m.LectureStatsID,
		LectureStatsLectureID:         m.LectureStatsLectureID,
		LectureStatsTotalParticipants: m.LectureStatsTotalParticipants,
		LectureStatsAverageGrade:      m.LectureStatsAverageGrade,
		LectureStatsUpdatedAt:         m.LectureStatsUpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// Optional: Convert list model → list response
func ToLectureStatsResponseList(models []model.LectureStatsModel) []LectureStatsResponse {
	var result []LectureStatsResponse
	for _, m := range models {
		result = append(result, *ToLectureStatsResponse(&m))
	}
	return result
}
