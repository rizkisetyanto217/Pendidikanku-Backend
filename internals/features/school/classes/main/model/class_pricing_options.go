// internals/features/lembaga/classes/pricing_options/model/class_pricing_option_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// Nilai yang valid untuk class_pricing_options_price_type sesuai ENUM di DB.
// (Opsional dipakai di layer lain agar konsisten.)
const (
	PriceTypeOneTime  = "ONE_TIME"
	PriceTypeRecurring = "RECURRING"
)

type ClassPricingOption struct {
	// Kolom & tipe diselaraskan dengan SQL
	ClassPricingOptionsID               uuid.UUID  `json:"class_pricing_options_id"                gorm:"column:class_pricing_options_id;type:uuid;default:gen_random_uuid();primaryKey"`
	ClassPricingOptionsClassID          uuid.UUID  `json:"class_pricing_options_class_id"          gorm:"column:class_pricing_options_class_id;type:uuid;not null;index:idx_class_pricing_options_class_id"`
	ClassPricingOptionsLabel            string     `json:"class_pricing_options_label"             gorm:"column:class_pricing_options_label;type:varchar(80);not null"`
	ClassPricingOptionsPriceType        string     `json:"class_pricing_options_price_type"        gorm:"column:class_pricing_options_price_type;type:class_price_type;not null"` // 'ONE_TIME' | 'RECURRING'
	ClassPricingOptionsAmountIDR        int        `json:"class_pricing_options_amount_idr"        gorm:"column:class_pricing_options_amount_idr;not null"`
	ClassPricingOptionsRecurrenceMonths *int       `json:"class_pricing_options_recurrence_months,omitempty" gorm:"column:class_pricing_options_recurrence_months"`

	ClassPricingOptionsCreatedAt        time.Time  `json:"class_pricing_options_created_at"        gorm:"column:class_pricing_options_created_at;autoCreateTime"`
	ClassPricingOptionsUpdatedAt        *time.Time `json:"class_pricing_options_updated_at,omitempty" gorm:"column:class_pricing_options_updated_at;autoUpdateTime"`
	ClassPricingOptionsDeletedAt        *time.Time `json:"class_pricing_options_deleted_at,omitempty" gorm:"column:class_pricing_options_deleted_at"`
}

func (ClassPricingOption) TableName() string {
	return "class_pricing_options"
}
