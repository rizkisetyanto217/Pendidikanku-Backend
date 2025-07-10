package model

import (
	"time"
)

type RankRequirement struct {
	RankReqID       uint      `gorm:"column:rank_req_id;primaryKey" json:"rank_req_id"`                        // ID unik
	RankReqRank     int       `gorm:"column:rank_req_rank;unique;not null" json:"rank_req_rank"`              // Nomor urut pangkat
	RankReqName     string    `gorm:"column:rank_req_name;type:varchar(100)" json:"rank_req_name"`            // Nama pangkat
	RankReqMinLevel int       `gorm:"column:rank_req_min_level;not null" json:"rank_req_min_level"`           // Level minimum
	RankReqMaxLevel *int      `gorm:"column:rank_req_max_level" json:"rank_req_max_level,omitempty"`          // Level maksimum (nullable)
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                     // Timestamp dibuat
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                     // Timestamp update
}

func (RankRequirement) TableName() string {
	return "rank_requirements"
}
