package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JSONBTeacher struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Encode ke database (JSON string)
func (j JSONBTeacher) Value() (driver.Value, error) {
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil // ⬅️ ubah hasil ke string, bukan []byte
}

// Decode dari database (JSON string)
func (j *JSONBTeacher) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONBTeacher value: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

type LectureSessionModel struct {
	LectureSessionID                     uuid.UUID    `gorm:"column:lecture_session_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_session_id"`
	LectureSessionTitle                  string       `gorm:"column:lecture_session_title;type:varchar(255);not null" json:"lecture_session_title"`
	LectureSessionDescription            string       `gorm:"column:lecture_session_description;type:text" json:"lecture_session_description"`
	LectureSessionTeacher                JSONBTeacher `gorm:"column:lecture_session_teacher;type:jsonb;not null" json:"lecture_session_teacher"`
	LectureSessionStartTime              time.Time    `gorm:"column:lecture_session_start_time;not null" json:"lecture_session_start_time"`
	LectureSessionEndTime                time.Time    `gorm:"column:lecture_session_end_time;not null" json:"lecture_session_end_time"`
	LectureSessionPlace                  *string      `gorm:"column:lecture_session_place;type:text" json:"lecture_session_place"`
	LectureSessionImageURL               *string      `gorm:"column:lecture_session_image_url;type:text" json:"lecture_session_image_url"`
	LectureSessionLectureID              *uuid.UUID   `gorm:"column:lecture_session_lecture_id;type:uuid" json:"lecture_session_lecture_id"`
	LectureSessionCapacity               *int         `gorm:"column:lecture_session_capacity" json:"lecture_session_capacity"`
	LectureSessionIsPublic               bool         `gorm:"column:lecture_session_is_public;default:true" json:"lecture_session_is_public"`
	LectureSessionIsRegistrationRequired bool         `gorm:"column:lecture_session_is_registration_required;default:false" json:"lecture_session_is_registration_required"`
	LectureSessionIsPaid                 bool         `gorm:"column:lecture_session_is_paid;default:false" json:"lecture_session_is_paid"`
	LectureSessionPrice                  *int         `gorm:"column:lecture_session_price" json:"lecture_session_price"`
	LectureSessionPaymentDeadline        *time.Time   `gorm:"column:lecture_session_payment_deadline" json:"lecture_session_payment_deadline"`
	LectureSessionCreatedAt              time.Time    `gorm:"column:lecture_session_created_at;autoCreateTime" json:"lecture_session_created_at"`
}

// TableName overrides the default table name
func (LectureSessionModel) TableName() string {
	return "lecture_sessions"
}
