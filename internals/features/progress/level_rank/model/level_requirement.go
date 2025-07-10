package model

import (
	"time"
)

type LevelRequirement struct {
	LevelReqID        uint      `gorm:"column:level_req_id;primaryKey" json:"level_req_id"`
	LevelReqLevel     int       `gorm:"column:level_req_level;unique;not null" json:"level_req_level"`
	LevelReqName      string    `gorm:"column:level_req_name" json:"level_req_name"`
	LevelReqMinPoints int       `gorm:"column:level_req_min_points;not null" json:"level_req_min_points"`
	LevelReqMaxPoints *int      `gorm:"column:level_req_max_points" json:"level_req_max_points,omitempty"` // nullable
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (LevelRequirement) TableName() string {
	return "level_requirements"
}
