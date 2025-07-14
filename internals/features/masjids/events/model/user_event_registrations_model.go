package model

import (
	"time"

	"github.com/google/uuid"
)

type UserEventRegistrationModel struct {
	UserEventRegistrationID        uuid.UUID `gorm:"column:user_event_registration_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"user_event_registration_id"`
	UserEventRegistrationEventID   uuid.UUID `gorm:"column:user_event_registration_event_id;type:uuid;not null" json:"user_event_registration_event_id"`
	UserEventRegistrationUserID    uuid.UUID `gorm:"column:user_event_registration_user_id;type:uuid;not null" json:"user_event_registration_user_id"`
	UserEventRegistrationMasjidID  uuid.UUID `gorm:"column:user_event_registration_masjid_id;type:uuid;not null" json:"user_event_registration_masjid_id"` // ✅ Ditambahkan
	UserEventRegistrationStatus    string    `gorm:"column:user_event_registration_status;type:varchar(50);default:'registered'" json:"user_event_registration_status"`
	UserEventRegistrationCreatedAt time.Time `gorm:"column:user_event_registration_registered_at;autoCreateTime" json:"user_event_registration_registered_at"`
	UserEventRegistrationUpdatedAt time.Time `gorm:"column:user_event_registration_updated_at;autoUpdateTime" json:"user_event_registration_updated_at"` // ✅ Disarankan untuk ditambahkan agar sinkron dengan DB
}

func (UserEventRegistrationModel) TableName() string {
	return "user_event_registrations"
}
