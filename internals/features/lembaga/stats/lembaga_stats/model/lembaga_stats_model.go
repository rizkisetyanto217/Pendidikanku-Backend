package model

import (
	"time"

	"github.com/google/uuid"
)

type LembagaStats struct {
	LembagaStatsSchoolID       uuid.UUID  `gorm:"column:lembaga_stats_school_id;type:uuid;primaryKey" json:"lembaga_stats_school_id"`
	LembagaStatsActiveClasses  int        `gorm:"column:lembaga_stats_active_classes;not null;default:0" json:"lembaga_stats_active_classes"`
	LembagaStatsActiveSections int        `gorm:"column:lembaga_stats_active_sections;not null;default:0" json:"lembaga_stats_active_sections"`
	LembagaStatsActiveStudents int        `gorm:"column:lembaga_stats_active_students;not null;default:0" json:"lembaga_stats_active_students"`
	LembagaStatsActiveTeachers int        `gorm:"column:lembaga_stats_active_teachers;not null;default:0" json:"lembaga_stats_active_teachers"`
	LembagaStatsCreatedAt      time.Time  `gorm:"column:lembaga_stats_created_at;autoCreateTime" json:"lembaga_stats_created_at"`
	LembagaStatsUpdatedAt      *time.Time `gorm:"column:lembaga_stats_updated_at;autoUpdateTime" json:"lembaga_stats_updated_at,omitempty"`
}

// TableName untuk override nama tabel di GORM
func (LembagaStats) TableName() string {
	return "lembaga_stats"
}
