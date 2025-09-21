// internals/features/school/sessions/schedules/model/class_schedule_rule_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Go-side enum buat week_parity_enum
type WeekParityEnum string

const (
	WeekParityAll  WeekParityEnum = "all"
	WeekParityOdd  WeekParityEnum = "odd"
	WeekParityEven WeekParityEnum = "even"
)

type ClassScheduleRuleModel struct {
	ClassScheduleRulesID uuid.UUID `json:"class_schedule_rules_id" gorm:"column:class_schedule_rules_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// FK komposit → class_schedules (tenant-safe)
	ClassScheduleRuleMasjidID   uuid.UUID `json:"class_schedule_rule_masjid_id"   gorm:"column:class_schedule_rule_masjid_id;type:uuid;not null"`
	ClassScheduleRuleScheduleID uuid.UUID `json:"class_schedule_rule_schedule_id" gorm:"column:class_schedule_rule_schedule_id;type:uuid;not null"`

	// Pola mingguan
	ClassScheduleRuleDayOfWeek int       `json:"class_schedule_rule_day_of_week" gorm:"column:class_schedule_rule_day_of_week;not null"` // 1..7 (ISO)
	ClassScheduleRuleStartTime time.Time `json:"class_schedule_rule_start_time"  gorm:"column:class_schedule_rule_start_time;type:time;not null"`
	ClassScheduleRuleEndTime   time.Time `json:"class_schedule_rule_end_time"    gorm:"column:class_schedule_rule_end_time;type:time;not null"`

	// Opsi pola
	ClassScheduleRuleIntervalWeeks    int            `json:"class_schedule_rule_interval_weeks"     gorm:"column:class_schedule_rule_interval_weeks;not null;default:1"`
	ClassScheduleRuleStartOffsetWeeks int            `json:"class_schedule_rule_start_offset_weeks" gorm:"column:class_schedule_rule_start_offset_weeks;not null;default:0"`
	ClassScheduleRuleWeekParity       WeekParityEnum `json:"class_schedule_rule_week_parity"        gorm:"column:class_schedule_rule_week_parity;type:week_parity_enum;not null;default:'all'"`
	ClassScheduleRuleWeeksOfMonth     pq.Int64Array  `json:"class_schedule_rule_weeks_of_month"     gorm:"column:class_schedule_rule_weeks_of_month;type:int[]"`
	ClassScheduleRuleLastWeekOfMonth  bool           `json:"class_schedule_rule_last_week_of_month" gorm:"column:class_schedule_rule_last_week_of_month;not null;default:false"`

	// Audit
	ClassScheduleRuleCreatedAt time.Time      `json:"class_schedule_rule_created_at" gorm:"column:class_schedule_rule_created_at;type:timestamptz;not null;autoCreateTime"`
	ClassScheduleRuleUpdatedAt time.Time      `json:"class_schedule_rule_updated_at" gorm:"column:class_schedule_rule_updated_at;type:timestamptz;not null;autoUpdateTime"`
	ClassScheduleRuleDeletedAt gorm.DeletedAt `json:"class_schedule_rule_deleted_at" gorm:"column:class_schedule_rule_deleted_at;index"`

	// Optional relation ke header (pakai FK komposit)
	Schedule ClassScheduleModel `json:"-" gorm:"foreignKey:ClassScheduleRuleScheduleID,ClassScheduleRuleMasjidID;references:ClassScheduleID,ClassSchedulesMasjidID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (ClassScheduleRuleModel) TableName() string { return "class_schedule_rules" }
