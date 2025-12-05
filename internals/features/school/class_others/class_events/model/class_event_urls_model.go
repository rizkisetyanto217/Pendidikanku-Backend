// file: internals/features/school/others/events/model/class_event_urls_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClassEventURLModel merepresentasikan lampiran/URL fleksibel untuk class_events.
type ClassEventURLModel struct {
	// PK & tenant
	ClassEventURLID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_event_url_id"        json:"class_event_url_id"`
	ClassEventURLSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_event_url_school_id"                             json:"class_event_url_school_id"`

	// Relasi ke event
	ClassEventURLEventID uuid.UUID `gorm:"type:uuid;not null;column:class_event_url_event_id"                               json:"class_event_url_event_id"`

	// Klasifikasi & label
	ClassEventURLKind  string  `gorm:"type:varchar(32);not null;column:class_event_url_kind"                                json:"class_event_url_kind"`
	ClassEventURLLabel *string `gorm:"type:varchar(160);column:class_event_url_label"                                       json:"class_event_url_label,omitempty"`

	// Storage (slot aktif & kandidat delete)
	ClassEventURLURL                *string    `gorm:"type:text;column:class_event_url_url"                                       json:"class_event_url_url,omitempty"`
	ClassEventURLObjectKey          *string    `gorm:"type:text;column:class_event_url_object_key"                                json:"class_event_url_object_key,omitempty"`
	ClassEventURLURLOld             *string    `gorm:"type:text;column:class_event_url_url_old"                                   json:"class_event_url_url_old,omitempty"`
	ClassEventURLObjectKeyOld       *string    `gorm:"type:text;column:class_event_url_object_key_old"                            json:"class_event_url_object_key_old,omitempty"`
	ClassEventURLDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:class_event_url_delete_pending_until"        json:"class_event_url_delete_pending_until,omitempty"`

	// Flag
	ClassEventURLIsPrimary bool `gorm:"not null;default:false;column:class_event_url_is_primary"                            json:"class_event_url_is_primary"`

	// Audit
	ClassEventURLCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:class_event_url_created_at" json:"class_event_url_created_at"`
	ClassEventURLUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:class_event_url_updated_at" json:"class_event_url_updated_at"`
	ClassEventURLDeletedAt gorm.DeletedAt `gorm:"column:class_event_url_deleted_at"                                                           json:"class_event_url_deleted_at,omitempty"`
}

func (ClassEventURLModel) TableName() string { return "class_event_urls" }
