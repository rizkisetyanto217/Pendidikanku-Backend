package dto

import (
	"schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	"time"

	"github.com/google/uuid"
)

type UserLectureSessionsAttendanceRequest struct {
	UserLectureSessionsAttendanceLectureSessionID string `json:"user_lecture_sessions_attendance_lecture_session_id"`
	UserLectureSessionsAttendanceStatus           int    `json:"user_lecture_sessions_attendance_status"`
	UserLectureSessionsAttendanceNotes            string `json:"user_lecture_sessions_attendance_notes"`
	UserLectureSessionsAttendancePersonalNotes    string `json:"user_lecture_sessions_attendance_personal_notes"`
}

type UserLectureSessionsAttendanceResponse struct {
	UserLectureSessionsAttendanceID               string    `json:"user_lecture_sessions_attendance_id"`
	UserLectureSessionsAttendanceUserID           string    `json:"user_lecture_sessions_attendance_user_id"`
	UserLectureSessionsAttendanceLectureSessionID string    `json:"user_lecture_sessions_attendance_lecture_session_id"`
	UserLectureSessionsAttendanceLectureID        string    `json:"user_lecture_sessions_attendance_lecture_id"`
	UserLectureSessionsAttendanceStatus           int       `json:"user_lecture_sessions_attendance_status"`
	UserLectureSessionsAttendanceNotes            string    `json:"user_lecture_sessions_attendance_notes"`
	UserLectureSessionsAttendancePersonalNotes    string    `json:"user_lecture_sessions_attendance_personal_notes"`
	UserLectureSessionsAttendanceCreatedAt        time.Time `json:"user_lecture_sessions_attendance_created_at"`
}

// 🔁 Convert from Model to DTO Response
func FromModelUserLectureSessionsAttendance(m *model.UserLectureSessionsAttendanceModel) UserLectureSessionsAttendanceResponse {
	return UserLectureSessionsAttendanceResponse{
		UserLectureSessionsAttendanceID:               m.UserLectureSessionsAttendanceID.String(),
		UserLectureSessionsAttendanceUserID:           m.UserLectureSessionsAttendanceUserID.String(),
		UserLectureSessionsAttendanceLectureSessionID: m.UserLectureSessionsAttendanceLectureSessionID.String(),
		UserLectureSessionsAttendanceLectureID:        m.UserLectureSessionsAttendanceLectureID.String(),
		UserLectureSessionsAttendanceStatus:           m.UserLectureSessionsAttendanceStatus,
		UserLectureSessionsAttendanceNotes:            m.UserLectureSessionsAttendanceNotes,
		UserLectureSessionsAttendancePersonalNotes:    m.UserLectureSessionsAttendancePersonalNotes,
		UserLectureSessionsAttendanceCreatedAt:        m.UserLectureSessionsAttendanceCreatedAt,
	}
}

// 🔁 Convert from DTO Request to Model
func ToModelUserLectureSessionsAttendance(input *UserLectureSessionsAttendanceRequest, userID uuid.UUID) *model.UserLectureSessionsAttendanceModel {
	sessionID, _ := uuid.Parse(input.UserLectureSessionsAttendanceLectureSessionID)

	return &model.UserLectureSessionsAttendanceModel{
		UserLectureSessionsAttendanceUserID:           userID,
		UserLectureSessionsAttendanceLectureSessionID: sessionID,
		UserLectureSessionsAttendanceStatus:           input.UserLectureSessionsAttendanceStatus,
		UserLectureSessionsAttendanceNotes:            input.UserLectureSessionsAttendanceNotes,
		UserLectureSessionsAttendancePersonalNotes:    input.UserLectureSessionsAttendancePersonalNotes,
	}
}
