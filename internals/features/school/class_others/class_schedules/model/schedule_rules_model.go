// file: internals/features/school/sessions/schedules/model/class_schedule_rule_model.go
package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

/* =========================================================
   TimeOnly type
   - Simpan hanya jam-menit-detik ("HH:MM:SS")
========================================================= */

type TimeOnly struct {
	time.Time
}

func (t *TimeOnly) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case []byte:
		return t.parse(string(v))
	case string:
		return t.parse(v)
	default:
		return fmt.Errorf("cannot scan type %T into TimeOnly", value)
	}
}

func (t *TimeOnly) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		t.Time = time.Time{}
		return nil
	}

	// normalize "HH:MM" jadi "HH:MM:SS"
	if len(s) == 5 {
		s += ":00"
	}

	parsed, err := time.Parse("15:04:05", s)
	if err != nil {
		return err
	}

	// tanggal dummy (nggak dipakai secara bisnis)
	t.Time = time.Date(2000, 1, 1, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.Local)
	return nil
}

func (t TimeOnly) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	// Simpan ke DB sebagai "HH:MM:SS"
	return t.Format("15:04:05"), nil
}

func (t TimeOnly) HHMM() string {
	return t.Format("15:04")
}

/* =========================================================
   Enum
========================================================= */

type WeekParityEnum string

const (
	WeekParityAll  WeekParityEnum = "all"
	WeekParityOdd  WeekParityEnum = "odd"
	WeekParityEven WeekParityEnum = "even"
)

/* =========================================================
   Main Model (SLIM, tanpa CSST cache)
========================================================= */

type ClassScheduleRuleModel struct {
	/* -----------------------------
	   PK & Tenant
	----------------------------- */
	ClassScheduleRuleID         uuid.UUID `gorm:"column:class_schedule_rule_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_schedule_rule_id"`
	ClassScheduleRuleSchoolID   uuid.UUID `gorm:"column:class_schedule_rule_school_id;type:uuid;not null" json:"class_schedule_rule_school_id"`
	ClassScheduleRuleScheduleID uuid.UUID `gorm:"column:class_schedule_rule_schedule_id;type:uuid;not null" json:"class_schedule_rule_schedule_id"`

	/* -----------------------------
	   Base Weekly Pattern
	----------------------------- */
	ClassScheduleRuleDayOfWeek int      `gorm:"column:class_schedule_rule_day_of_week;not null" json:"class_schedule_rule_day_of_week"`
	ClassScheduleRuleStartTime TimeOnly `gorm:"column:class_schedule_rule_start_time;type:time;not null" json:"class_schedule_rule_start_time"`
	ClassScheduleRuleEndTime   TimeOnly `gorm:"column:class_schedule_rule_end_time;type:time;not null" json:"class_schedule_rule_end_time"`

	/* -----------------------------
	   Advanced Weekly Options
	----------------------------- */
	ClassScheduleRuleIntervalWeeks    int            `gorm:"column:class_schedule_rule_interval_weeks;not null;default:1" json:"class_schedule_rule_interval_weeks"`
	ClassScheduleRuleStartOffsetWeeks int            `gorm:"column:class_schedule_rule_start_offset_weeks;not null;default:0" json:"class_schedule_rule_start_offset_weeks"`
	ClassScheduleRuleWeekParity       WeekParityEnum `gorm:"column:class_schedule_rule_week_parity;type:week_parity_enum;not null;default:'all'" json:"class_schedule_rule_week_parity"`
	ClassScheduleRuleWeeksOfMonth     pq.Int64Array  `gorm:"column:class_schedule_rule_weeks_of_month;type:int[]" json:"class_schedule_rule_weeks_of_month,omitempty"`
	ClassScheduleRuleLastWeekOfMonth  bool           `gorm:"column:class_schedule_rule_last_week_of_month;not null;default:false" json:"class_schedule_rule_last_week_of_month"`

	/* =========================================================
	   CSST Reference (FK langsung; join ke CSST saat query)
	========================================================= */
	ClassScheduleRuleCSSTID uuid.UUID `gorm:"column:class_schedule_rule_csst_id;type:uuid;not null" json:"class_schedule_rule_csst_id"`

	/* -----------------------------
	   Generated Time Range (menit)
	----------------------------- */
	ClassScheduleRuleStartMin int16 `gorm:"column:class_schedule_rule_start_min;type:smallint;->" json:"class_schedule_rule_start_min"`
	ClassScheduleRuleEndMin   int16 `gorm:"column:class_schedule_rule_end_min;type:smallint;->" json:"class_schedule_rule_end_min"`

	/* -----------------------------
	   Timestamps
	----------------------------- */
	ClassScheduleRuleCreatedAt time.Time      `gorm:"column:class_schedule_rule_created_at;type:timestamptz;not null;autoCreateTime" json:"class_schedule_rule_created_at"`
	ClassScheduleRuleUpdatedAt time.Time      `gorm:"column:class_schedule_rule_updated_at;type:timestamptz;not null;autoUpdateTime" json:"class_schedule_rule_updated_at"`
	ClassScheduleRuleDeletedAt gorm.DeletedAt `gorm:"column:class_schedule_rule_deleted_at;index" json:"class_schedule_rule_deleted_at,omitempty"`
}

func (ClassScheduleRuleModel) TableName() string { return "class_schedule_rules" }
