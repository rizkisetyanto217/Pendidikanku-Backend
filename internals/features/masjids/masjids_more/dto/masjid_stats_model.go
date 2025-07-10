package dto

import (
	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/google/uuid"
)

type MasjidStatsRequest struct {
	MasjidStatsTotalLectures     int       `json:"masjid_stats_total_lectures"`
	MasjidStatsTotalSessions     int       `json:"masjid_stats_total_sessions"`
	MasjidStatsTotalParticipants int       `json:"masjid_stats_total_participants"`
	MasjidStatsTotalDonations    int64     `json:"masjid_stats_total_donations"`
	MasjidStatsMasjidID          uuid.UUID `json:"masjid_stats_masjid_id"`
}

type MasjidStatsResponse struct {
	MasjidStatsID                uuid.UUID `json:"masjid_stats_id"`
	MasjidStatsTotalLectures     int       `json:"masjid_stats_total_lectures"`
	MasjidStatsTotalSessions     int       `json:"masjid_stats_total_sessions"`
	MasjidStatsTotalParticipants int       `json:"masjid_stats_total_participants"`
	MasjidStatsTotalDonations    int64     `json:"masjid_stats_total_donations"`
	MasjidStatsMasjidID          uuid.UUID `json:"masjid_stats_masjid_id"`
	MasjidStatsUpdatedAt         string    `json:"masjid_stats_updated_at"`
}

// Convert request → model
func (r *MasjidStatsRequest) ToModel() *model.MasjidStatsModel {
	return &model.MasjidStatsModel{
		MasjidStatsTotalLectures:     r.MasjidStatsTotalLectures,
		MasjidStatsTotalSessions:     r.MasjidStatsTotalSessions,
		MasjidStatsTotalParticipants: r.MasjidStatsTotalParticipants,
		MasjidStatsTotalDonations:    r.MasjidStatsTotalDonations,
		MasjidStatsMasjidID:          r.MasjidStatsMasjidID,
	}
}

// Convert model → response
func ToMasjidStatsResponse(m *model.MasjidStatsModel) *MasjidStatsResponse {
	return &MasjidStatsResponse{
		MasjidStatsID:                m.MasjidStatsID,
		MasjidStatsTotalLectures:     m.MasjidStatsTotalLectures,
		MasjidStatsTotalSessions:     m.MasjidStatsTotalSessions,
		MasjidStatsTotalParticipants: m.MasjidStatsTotalParticipants,
		MasjidStatsTotalDonations:    m.MasjidStatsTotalDonations,
		MasjidStatsMasjidID:          m.MasjidStatsMasjidID,
		MasjidStatsUpdatedAt:         m.MasjidStatsUpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// Convert slice → response list
func ToMasjidStatsResponseList(models []model.MasjidStatsModel) []MasjidStatsResponse {
	var result []MasjidStatsResponse
	for _, m := range models {
		result = append(result, *ToMasjidStatsResponse(&m))
	}
	return result
}
