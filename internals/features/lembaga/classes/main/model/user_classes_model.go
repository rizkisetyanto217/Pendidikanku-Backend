// internals/features/lembaga/classes/user_classes/main/model/user_class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	UserClassStatusActive   = "active"
	UserClassStatusInactive = "inactive"
	UserClassStatusEnded    = "ended"
)

type UserClassesModel struct {
	UserClassesID                  uuid.UUID  `json:"user_classes_id"                                gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_classes_id"`
	UserClassesUserID              uuid.UUID  `json:"user_classes_user_id"                           gorm:"type:uuid;not null;column:user_classes_user_id"`
	UserClassesClassID             uuid.UUID  `json:"user_classes_class_id"                          gorm:"type:uuid;not null;column:user_classes_class_id"`
	UserClassesMasjidID            *uuid.UUID `json:"user_classes_masjid_id,omitempty"               gorm:"type:uuid;column:user_classes_masjid_id"`
	UserClassesStatus              string     `json:"user_classes_status"                            gorm:"type:text;not null;default:'active';column:user_classes_status"`
	UserClassesStartedAt           *time.Time `json:"user_classes_started_at,omitempty"              gorm:"type:date;column:user_classes_started_at"`
	UserClassesEndedAt             *time.Time `json:"user_classes_ended_at,omitempty"                gorm:"type:date;column:user_classes_ended_at"`
	UserClassesFeeOverrideMonthlyIDR *int     `json:"user_classes_fee_override_monthly_idr,omitempty" gorm:"column:user_classes_fee_override_monthly_idr"`
	UserClassesNotes               *string    `json:"user_classes_notes,omitempty"                   gorm:"column:user_classes_notes"`
	UserClassesCreatedAt           time.Time  `json:"user_classes_created_at"                        gorm:"column:user_classes_created_at;autoCreateTime"`
	UserClassesUpdatedAt           *time.Time `json:"user_classes_updated_at,omitempty"              gorm:"column:user_classes_updated_at;autoUpdateTime"`
}

func (UserClassesModel) TableName() string {
	return "user_classes"
}
