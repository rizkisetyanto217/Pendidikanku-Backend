package dto

import (
	"encoding/json"
	"masjidku_backend/internals/features/masjids/lectures/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Struct Teacher untuk frontend & penyimpanan JSON
type Teacher struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Request dari frontend → backend
type LectureRequest struct {
	LectureTitle              string    `json:"lecture_title"`
	LectureDescription        string    `json:"lecture_description"`
	LectureMasjidID           uuid.UUID `json:"lecture_masjid_id"`
	TotalLectureSessions      *int      `json:"total_lecture_sessions"`
	LectureIsRecurring        bool      `json:"lecture_is_recurring"`
	LectureRecurrenceInterval *int      `json:"lecture_recurrence_interval"`
	LectureImageURL           *string   `json:"lecture_image_url"`
	LectureTeachers           []Teacher `json:"lecture_teachers"`
}

// Response ke frontend
type LectureResponse struct {
	LectureID                 uuid.UUID `json:"lecture_id"`
	LectureTitle              string    `json:"lecture_title"`
	LectureDescription        string    `json:"lecture_description"`
	LectureMasjidID           uuid.UUID `json:"lecture_masjid_id"`
	TotalLectureSessions      *int      `json:"total_lecture_sessions"`
	LectureIsRecurring        bool      `json:"lecture_is_recurring"`
	LectureRecurrenceInterval *int      `json:"lecture_recurrence_interval"`
	LectureImageURL           *string   `json:"lecture_image_url"`
	LectureTeachers           []Teacher `json:"lecture_teachers"`
	LectureCreatedAt          string    `json:"lecture_created_at"`
}

// Convert request → model
func (r *LectureRequest) ToModel() *model.LectureModel {
	// Encode teachers ke JSON
	teacherJSON, _ := json.Marshal(r.LectureTeachers)

	return &model.LectureModel{
		LectureTitle:              r.LectureTitle,
		LectureDescription:        r.LectureDescription,
		LectureMasjidID:           r.LectureMasjidID,
		TotalLectureSessions:      r.TotalLectureSessions,
		LectureIsRecurring:        r.LectureIsRecurring,
		LectureRecurrenceInterval: r.LectureRecurrenceInterval,
		LectureImageURL:           r.LectureImageURL,
		LectureTeachers:           datatypes.JSON(teacherJSON),
	}
}

// Convert model → response
func ToLectureResponse(m *model.LectureModel) *LectureResponse {
	var teachers []Teacher
	if m.LectureTeachers != nil {
		_ = json.Unmarshal(m.LectureTeachers, &teachers)
	}

	return &LectureResponse{
		LectureID:                 m.LectureID,
		LectureTitle:              m.LectureTitle,
		LectureDescription:        m.LectureDescription,
		LectureMasjidID:           m.LectureMasjidID,
		TotalLectureSessions:      m.TotalLectureSessions,
		LectureIsRecurring:        m.LectureIsRecurring,
		LectureRecurrenceInterval: m.LectureRecurrenceInterval,
		LectureImageURL:           m.LectureImageURL,
		LectureTeachers:           teachers,
		LectureCreatedAt:          m.LectureCreatedAt.Format("2006-01-02 15:04:05"),
	}
}
