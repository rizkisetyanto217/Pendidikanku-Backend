// file: internals/features/school/sessions/schedules/model/class_schedule_rule_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WeekParityEnum string

const (
	WeekParityAll  WeekParityEnum = "all"
	WeekParityOdd  WeekParityEnum = "odd"
	WeekParityEven WeekParityEnum = "even"
)

type ClassScheduleRuleModel struct {
	// PK
	ClassScheduleRuleID uuid.UUID `gorm:"column:class_schedule_rule_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_schedule_rule_id"`

	// Tenant & header (FK komposit → tenant-safe)
	ClassScheduleRuleMasjidID   uuid.UUID `gorm:"column:class_schedule_rule_masjid_id;type:uuid;not null" json:"class_schedule_rule_masjid_id"`
	ClassScheduleRuleScheduleID uuid.UUID `gorm:"column:class_schedule_rule_schedule_id;type:uuid;not null" json:"class_schedule_rule_schedule_id"`

	// Pola per pekan
	ClassScheduleRuleDayOfWeek int       `gorm:"column:class_schedule_rule_day_of_week;not null" json:"class_schedule_rule_day_of_week"`
	ClassScheduleRuleStartTime time.Time `gorm:"column:class_schedule_rule_start_time;type:time;not null" json:"class_schedule_rule_start_time"`
	ClassScheduleRuleEndTime   time.Time `gorm:"column:class_schedule_rule_end_time;type:time;not null" json:"class_schedule_rule_end_time"`

	// Opsi pola
	ClassScheduleRuleIntervalWeeks    int            `gorm:"column:class_schedule_rule_interval_weeks;not null;default:1" json:"class_schedule_rule_interval_weeks"`
	ClassScheduleRuleStartOffsetWeeks int            `gorm:"column:class_schedule_rule_start_offset_weeks;not null;default:0" json:"class_schedule_rule_start_offset_weeks"`
	ClassScheduleRuleWeekParity       WeekParityEnum `gorm:"column:class_schedule_rule_week_parity;type:week_parity_enum;not null;default:'all'" json:"class_schedule_rule_week_parity"`
	ClassScheduleRuleWeeksOfMonth     pq.Int64Array  `gorm:"column:class_schedule_rule_weeks_of_month;type:int[]" json:"class_schedule_rule_weeks_of_month,omitempty"`
	ClassScheduleRuleLastWeekOfMonth  bool           `gorm:"column:class_schedule_rule_last_week_of_month;not null;default:false" json:"class_schedule_rule_last_week_of_month"`

	// Default penugasan CSST (tenant-safe, wajib)
	ClassScheduleRuleCSSTID       uuid.UUID `gorm:"column:class_schedule_rule_csst_id;type:uuid;not null" json:"class_schedule_rule_csst_id"`
	ClassScheduleRuleCSSTMasjidID uuid.UUID `gorm:"column:class_schedule_rule_csst_masjid_id;type:uuid;not null" json:"class_schedule_rule_csst_masjid_id"`

	// Snapshot CSST (denormalized)
	ClassScheduleRuleCSSTSnapshot datatypes.JSONMap `gorm:"column:class_schedule_rule_csst_snapshot;type:jsonb;not null;default:'{}'" json:"class_schedule_rule_csst_snapshot"`

	// Generated columns (read-only) dari snapshot
	ClassScheduleRuleCSSTTeacherID      *uuid.UUID `gorm:"column:class_schedule_rule_csst_teacher_id;type:uuid;->" json:"class_schedule_rule_csst_teacher_id,omitempty"`
	ClassScheduleRuleCSSTSectionID      *uuid.UUID `gorm:"column:class_schedule_rule_csst_section_id;type:uuid;->" json:"class_schedule_rule_csst_section_id,omitempty"`
	ClassScheduleRuleCSSTClassSubjectID *uuid.UUID `gorm:"column:class_schedule_rule_csst_class_subject_id;type:uuid;->" json:"class_schedule_rule_csst_class_subject_id,omitempty"`
	ClassScheduleRuleCSSTRoomID         *uuid.UUID `gorm:"column:class_schedule_rule_csst_room_id;type:uuid;->" json:"class_schedule_rule_csst_room_id,omitempty"`

	// Audit
	ClassScheduleRuleCreatedAt time.Time      `gorm:"column:class_schedule_rule_created_at;type:timestamptz;not null;autoCreateTime" json:"class_schedule_rule_created_at"`
	ClassScheduleRuleUpdatedAt time.Time      `gorm:"column:class_schedule_rule_updated_at;type:timestamptz;not null;autoUpdateTime" json:"class_schedule_rule_updated_at"`
	ClassScheduleRuleDeletedAt gorm.DeletedAt `gorm:"column:class_schedule_rule_deleted_at;index" json:"class_schedule_rule_deleted_at,omitempty"`

	// Generated untuk anti-overlap (read-only)
	ClassScheduleRuleStartMin int16 `gorm:"column:class_schedule_rule_start_min;type:smallint;->" json:"class_schedule_rule_start_min"`
	ClassScheduleRuleEndMin   int16 `gorm:"column:class_schedule_rule_end_min;type:smallint;->" json:"class_schedule_rule_end_min"`
}

func (ClassScheduleRuleModel) TableName() string { return "class_schedule_rules" }
