package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserEventRegistrationModel struct {
	UserEventRegistrationID       uuid.UUID `gorm:"column:user_event_registration_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"user_event_registration_id"`
	UserEventRegistrationEventID  uuid.UUID `gorm:"column:user_event_registration_event_session_id;type:uuid;not null" json:"user_event_registration_event_session_id"`
	UserEventRegistrationUserID   uuid.UUID `gorm:"column:user_event_registration_user_id;type:uuid;not null" json:"user_event_registration_user_id"`
	UserEventRegistrationSchoolID uuid.UUID `gorm:"column:user_event_registration_school_id;type:uuid;not null" json:"user_event_registration_school_id"`
	UserEventRegistrationStatus   string    `gorm:"column:user_event_registration_status;type:varchar(50);default:'registered'" json:"user_event_registration_status"`

	UserEventRegistrationCreatedAt time.Time      `gorm:"column:user_event_registration_registered_at;autoCreateTime" json:"user_event_registration_registered_at"`
	UserEventRegistrationUpdatedAt time.Time      `gorm:"column:user_event_registration_updated_at;autoUpdateTime" json:"user_event_registration_updated_at"`
	UserEventRegistrationDeletedAt gorm.DeletedAt `gorm:"column:user_event_registration_deleted_at;index" json:"user_event_registration_deleted_at,omitempty"`
}

func (UserEventRegistrationModel) TableName() string {
	return "user_event_registrations"
}
