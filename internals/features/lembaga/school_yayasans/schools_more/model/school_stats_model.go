package model

import (
	School "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"
	"time"

	"github.com/google/uuid"
)

type SchoolStatsModel struct {
	SchoolStatsID                uuid.UUID `gorm:"column:school_stats_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"school_stats_id"`
	SchoolStatsTotalLectures     int       `gorm:"column:school_stats_total_lectures;default:0" json:"school_stats_total_lectures"`
	SchoolStatsTotalSessions     int       `gorm:"column:school_stats_total_sessions;default:0" json:"school_stats_total_sessions"`
	SchoolStatsTotalParticipants int       `gorm:"column:school_stats_total_participants;default:0" json:"school_stats_total_participants"`
	SchoolStatsTotalDonations    int64     `gorm:"column:school_stats_total_donations;default:0" json:"school_stats_total_donations"`
	SchoolStatsSchoolID          uuid.UUID `gorm:"column:school_stats_school_id;type:uuid;not null" json:"school_stats_school_id"`
	SchoolStatsUpdatedAt         time.Time `gorm:"column:school_stats_updated_at;autoUpdateTime" json:"school_stats_updated_at"`

	// Optional relation ke tabel schools
	School School.SchoolModel `gorm:"foreignKey:SchoolStatsSchoolID;references:SchoolID" json:"school,omitempty"`
}

// TableName override
func (SchoolStatsModel) TableName() string {
	return "school_stats"
}
