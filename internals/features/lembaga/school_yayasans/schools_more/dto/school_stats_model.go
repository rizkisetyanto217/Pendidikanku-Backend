package dto

import (
	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools_more/model"

	"github.com/google/uuid"
)

type SchoolStatsRequest struct {
	SchoolStatsTotalLectures     int       `json:"school_stats_total_lectures"`
	SchoolStatsTotalSessions     int       `json:"school_stats_total_sessions"`
	SchoolStatsTotalParticipants int       `json:"school_stats_total_participants"`
	SchoolStatsTotalDonations    int64     `json:"school_stats_total_donations"`
	SchoolStatsSchoolID          uuid.UUID `json:"school_stats_school_id"`
}

type SchoolStatsResponse struct {
	SchoolStatsID                uuid.UUID `json:"school_stats_id"`
	SchoolStatsTotalLectures     int       `json:"school_stats_total_lectures"`
	SchoolStatsTotalSessions     int       `json:"school_stats_total_sessions"`
	SchoolStatsTotalParticipants int       `json:"school_stats_total_participants"`
	SchoolStatsTotalDonations    int64     `json:"school_stats_total_donations"`
	SchoolStatsSchoolID          uuid.UUID `json:"school_stats_school_id"`
	SchoolStatsUpdatedAt         string    `json:"school_stats_updated_at"`
}

// Convert request → model
func (r *SchoolStatsRequest) ToModel() *model.SchoolStatsModel {
	return &model.SchoolStatsModel{
		SchoolStatsTotalLectures:     r.SchoolStatsTotalLectures,
		SchoolStatsTotalSessions:     r.SchoolStatsTotalSessions,
		SchoolStatsTotalParticipants: r.SchoolStatsTotalParticipants,
		SchoolStatsTotalDonations:    r.SchoolStatsTotalDonations,
		SchoolStatsSchoolID:          r.SchoolStatsSchoolID,
	}
}

// Convert model → response
func ToSchoolStatsResponse(m *model.SchoolStatsModel) *SchoolStatsResponse {
	return &SchoolStatsResponse{
		SchoolStatsID:                m.SchoolStatsID,
		SchoolStatsTotalLectures:     m.SchoolStatsTotalLectures,
		SchoolStatsTotalSessions:     m.SchoolStatsTotalSessions,
		SchoolStatsTotalParticipants: m.SchoolStatsTotalParticipants,
		SchoolStatsTotalDonations:    m.SchoolStatsTotalDonations,
		SchoolStatsSchoolID:          m.SchoolStatsSchoolID,
		SchoolStatsUpdatedAt:         m.SchoolStatsUpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// Convert slice → response list
func ToSchoolStatsResponseList(models []model.SchoolStatsModel) []SchoolStatsResponse {
	var result []SchoolStatsResponse
	for _, m := range models {
		result = append(result, *ToSchoolStatsResponse(&m))
	}
	return result
}
