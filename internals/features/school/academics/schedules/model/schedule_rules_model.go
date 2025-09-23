// internals/features/school/sessions/schedules/model/class_schedule_rule_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type WeekParityEnum string

const (
	WeekParityAll  WeekParityEnum = "all"
	WeekParityOdd  WeekParityEnum = "odd"
	WeekParityEven WeekParityEnum = "even"
)

type ClassScheduleRuleModel struct {
	ClassScheduleRulesID uuid.UUID `gorm:"column:class_schedule_rules_id;type:uuid;default:gen_random_uuid();primaryKey"`

	ClassScheduleRuleMasjidID   uuid.UUID `gorm:"column:class_schedule_rule_masjid_id;type:uuid;not null"`
	ClassScheduleRuleScheduleID uuid.UUID `gorm:"column:class_schedule_rule_schedule_id;type:uuid;not null"`

	ClassScheduleRuleDayOfWeek int       `gorm:"column:class_schedule_rule_day_of_week;not null"`
	ClassScheduleRuleStartTime time.Time `gorm:"column:class_schedule_rule_start_time;type:time;not null"`
	ClassScheduleRuleEndTime   time.Time `gorm:"column:class_schedule_rule_end_time;type:time;not null"`

	ClassScheduleRuleIntervalWeeks    int            `gorm:"column:class_schedule_rule_interval_weeks;not null;default:1"`
	ClassScheduleRuleStartOffsetWeeks int            `gorm:"column:class_schedule_rule_start_offset_weeks;not null;default:0"`
	ClassScheduleRuleWeekParity       WeekParityEnum `gorm:"column:class_schedule_rule_week_parity;type:week_parity_enum;not null;default:'all'"`
	ClassScheduleRuleWeeksOfMonth     pq.Int64Array  `gorm:"column:class_schedule_rule_weeks_of_month;type:int[]"`
	ClassScheduleRuleLastWeekOfMonth  bool           `gorm:"column:class_schedule_rule_last_week_of_month;not null;default:false"`

	ClassScheduleRuleCreatedAt time.Time      `gorm:"column:class_schedule_rule_created_at;type:timestamptz;not null;autoCreateTime"`
	ClassScheduleRuleUpdatedAt time.Time      `gorm:"column:class_schedule_rule_updated_at;type:timestamptz;not null;autoUpdateTime"`
	ClassScheduleRuleDeletedAt gorm.DeletedAt `gorm:"column:class_schedule_rule_deleted_at;index"`
}

func (ClassScheduleRuleModel) TableName() string { return "class_schedule_rules" }
