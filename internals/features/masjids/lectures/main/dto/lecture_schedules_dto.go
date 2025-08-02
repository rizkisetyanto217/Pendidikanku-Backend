package dto

import "github.com/google/uuid"

type CreateLectureScheduleRequest struct {
	LectureSchedulesLectureID              uuid.UUID `json:"lecture_schedules_lecture_id" validate:"required"`
	LectureSchedulesTitle                  string    `json:"lecture_schedules_title" validate:"required"`
	LectureSchedulesDayOfWeek              int       `json:"lecture_schedules_day_of_week" validate:"required,min=0,max=6"`
	LectureSchedulesStartTime              string    `json:"lecture_schedules_start_time" validate:"required"` // format: HH:mm
	LectureSchedulesEndTime                *string   `json:"lecture_schedules_end_time"`

	LectureSchedulesPlace                  string    `json:"lecture_schedules_place" validate:"required"`
	LectureSchedulesNotes                  string    `json:"lecture_schedules_notes"`

	LectureSchedulesIsActive               *bool     `json:"lecture_schedules_is_active"`
	LectureSchedulesIsPaid                 *bool     `json:"lecture_schedules_is_paid"`
	LectureSchedulesPrice                  *int      `json:"lecture_schedules_price"`
	LectureSchedulesCapacity               *int      `json:"lecture_schedules_capacity"`
	LectureSchedulesIsRegistrationRequired *bool     `json:"lecture_schedules_is_registration_required"`
}

type UpdateLectureScheduleRequest struct {
	LectureSchedulesTitle                  *string   `json:"lecture_schedules_title"`
	LectureSchedulesDayOfWeek              *int      `json:"lecture_schedules_day_of_week" validate:"omitempty,min=0,max=6"`
	LectureSchedulesStartTime              *string   `json:"lecture_schedules_start_time"`
	LectureSchedulesEndTime                *string   `json:"lecture_schedules_end_time"`

	LectureSchedulesPlace                  *string   `json:"lecture_schedules_place"`
	LectureSchedulesNotes                  *string   `json:"lecture_schedules_notes"`

	LectureSchedulesIsActive               *bool     `json:"lecture_schedules_is_active"`
	LectureSchedulesIsPaid                 *bool     `json:"lecture_schedules_is_paid"`
	LectureSchedulesPrice                  *int      `json:"lecture_schedules_price"`
	LectureSchedulesCapacity               *int      `json:"lecture_schedules_capacity"`
	LectureSchedulesIsRegistrationRequired *bool     `json:"lecture_schedules_is_registration_required"`
}

type LectureScheduleResponse struct {
	LectureSchedulesID                     uuid.UUID `json:"lecture_schedules_id"`
	LectureSchedulesLectureID              uuid.UUID `json:"lecture_schedules_lecture_id"`
	LectureSchedulesTitle                  string    `json:"lecture_schedules_title"`
	LectureSchedulesDayOfWeek              int       `json:"lecture_schedules_day_of_week"`
	LectureSchedulesStartTime              string    `json:"lecture_schedules_start_time"`
	LectureSchedulesEndTime                *string   `json:"lecture_schedules_end_time"`
	LectureSchedulesPlace                  string    `json:"lecture_schedules_place"`
	LectureSchedulesNotes                  string    `json:"lecture_schedules_notes"`
	LectureSchedulesIsActive               bool      `json:"lecture_schedules_is_active"`
	LectureSchedulesIsPaid                 bool      `json:"lecture_schedules_is_paid"`
	LectureSchedulesPrice                  *int      `json:"lecture_schedules_price"`
	LectureSchedulesCapacity               *int      `json:"lecture_schedules_capacity"`
	LectureSchedulesIsRegistrationRequired bool      `json:"lecture_schedules_is_registration_required"`
}
