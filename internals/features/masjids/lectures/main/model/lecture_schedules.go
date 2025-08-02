package model

import (
	"time"

	"github.com/google/uuid"
)

type LectureSchedulesModel struct {
	LectureSchedulesID                       uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"lecture_schedules_id"`
	LectureSchedulesLectureID                uuid.UUID  `gorm:"type:uuid;not null" json:"lecture_schedules_lecture_id"`
	Lecture                                  LectureModel `gorm:"foreignKey:LectureSchedulesLectureID" json:"lecture"` // relasi optional
	LectureSchedulesTitle                    string        `gorm:"type:varchar(255);not null" json:"lecture_schedules_title"` // 🔥 BARU
	LectureSchedulesDayOfWeek                int        `gorm:"not null" json:"lecture_schedules_day_of_week"`        // 0 = Minggu, ..., 6 = Sabtu
	LectureSchedulesStartTime                string     `gorm:"type:time;not null" json:"lecture_schedules_start_time"`
	LectureSchedulesEndTime                  *string    `gorm:"type:time" json:"lecture_schedules_end_time"`

	LectureSchedulesPlace                    string     `gorm:"type:text" json:"lecture_schedules_place"`
	LectureSchedulesNotes                    string     `gorm:"type:text" json:"lecture_schedules_notes"`

	LectureSchedulesIsActive                 bool       `gorm:"default:true" json:"lecture_schedules_is_active"`
	LectureSchedulesIsPaid                   bool       `gorm:"default:false" json:"lecture_schedules_is_paid"`
	LectureSchedulesPrice                    *int       `json:"lecture_schedules_price"`
	LectureSchedulesCapacity                 *int       `json:"lecture_schedules_capacity"`
	LectureSchedulesIsRegistrationRequired   bool       `gorm:"default:false" json:"lecture_schedules_is_registration_required"`

	LectureSchedulesCreatedAt                time.Time  `gorm:"autoCreateTime" json:"lecture_schedules_created_at"`
	LectureSchedulesUpdatedAt                *time.Time `gorm:"autoUpdateTime" json:"lecture_schedules_updated_at"`
	LectureSchedulesDeletedAt                *time.Time `gorm:"index" json:"lecture_schedules_deleted_at"`
}

func (LectureSchedulesModel) TableName() string {
	return "lecture_schedules"
}