// file: internals/features/classes/openings/model/class_term_opening_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// ClassTermOpeningModel merepresentasikan tabel `class_term_openings`
type ClassTermOpeningModel struct {
	ClassTermOpeningsID        uuid.UUID  `json:"class_term_openings_id"        gorm:"column:class_term_openings_id;type:uuid;default:gen_random_uuid();primaryKey"`
	ClassTermOpeningsMasjidID  uuid.UUID  `json:"class_term_openings_masjid_id" gorm:"column:class_term_openings_masjid_id;type:uuid;not null"`
	ClassTermOpeningsClassID   uuid.UUID  `json:"class_term_openings_class_id"  gorm:"column:class_term_openings_class_id;type:uuid;not null"`
	ClassTermOpeningsTermID    uuid.UUID  `json:"class_term_openings_term_id"   gorm:"column:class_term_openings_term_id;type:uuid;not null"`

	ClassTermOpeningsIsOpen    bool       `json:"class_term_openings_is_open"   gorm:"column:class_term_openings_is_open;not null;default:true"`

	ClassTermOpeningsRegistrationOpensAt  *time.Time `json:"class_term_openings_registration_opens_at,omitempty"  gorm:"column:class_term_openings_registration_opens_at;type:timestamp"`
	ClassTermOpeningsRegistrationClosesAt *time.Time `json:"class_term_openings_registration_closes_at,omitempty" gorm:"column:class_term_openings_registration_closes_at;type:timestamp"`

	ClassTermOpeningsQuotaTotal           *int       `json:"class_term_openings_quota_total,omitempty"              gorm:"column:class_term_openings_quota_total"`
	ClassTermOpeningsQuotaTaken           int        `json:"class_term_openings_quota_taken"                        gorm:"column:class_term_openings_quota_taken;not null;default:0"`
	ClassTermOpeningsFeeOverrideMonthlyIDR *int      `json:"class_term_openings_fee_override_monthly_idr,omitempty" gorm:"column:class_term_openings_fee_override_monthly_idr"`

	ClassTermOpeningsNotes                *string    `json:"class_term_openings_notes,omitempty" gorm:"column:class_term_openings_notes;type:text"`

	ClassTermOpeningsCreatedAt            time.Time  `json:"class_term_openings_created_at"        gorm:"column:class_term_openings_created_at;type:timestamp;autoCreateTime"`
	ClassTermOpeningsUpdatedAt            *time.Time `json:"class_term_openings_updated_at,omitempty" gorm:"column:class_term_openings_updated_at;type:timestamp"`
	ClassTermOpeningsDeletedAt            *time.Time `json:"class_term_openings_deleted_at,omitempty" gorm:"column:class_term_openings_deleted_at;type:timestamp;index"`
}

func (ClassTermOpeningModel) TableName() string { return "class_term_openings" }
