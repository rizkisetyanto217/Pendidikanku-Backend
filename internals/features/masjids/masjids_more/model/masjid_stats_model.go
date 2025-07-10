package model

import (
	Masjid "masjidku_backend/internals/features/masjids/masjids/model"
	"time"

	"github.com/google/uuid"
)

type MasjidStatsModel struct {
	MasjidStatsID                uuid.UUID `gorm:"column:masjid_stats_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"masjid_stats_id"`
	MasjidStatsTotalLectures     int       `gorm:"column:masjid_stats_total_lectures;default:0" json:"masjid_stats_total_lectures"`
	MasjidStatsTotalSessions     int       `gorm:"column:masjid_stats_total_sessions;default:0" json:"masjid_stats_total_sessions"`
	MasjidStatsTotalParticipants int       `gorm:"column:masjid_stats_total_participants;default:0" json:"masjid_stats_total_participants"`
	MasjidStatsTotalDonations    int64     `gorm:"column:masjid_stats_total_donations;default:0" json:"masjid_stats_total_donations"`
	MasjidStatsMasjidID          uuid.UUID `gorm:"column:masjid_stats_masjid_id;type:uuid;not null" json:"masjid_stats_masjid_id"`
	MasjidStatsUpdatedAt         time.Time `gorm:"column:masjid_stats_updated_at;autoUpdateTime" json:"masjid_stats_updated_at"`

	// Optional relation ke tabel masjids
	Masjid Masjid.MasjidModel `gorm:"foreignKey:MasjidStatsMasjidID;references:MasjidID" json:"masjid,omitempty"`
}

// TableName override
func (MasjidStatsModel) TableName() string {
	return "masjid_stats"
}
