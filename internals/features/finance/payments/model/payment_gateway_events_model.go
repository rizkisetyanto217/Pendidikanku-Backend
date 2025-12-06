// file: internals/features/finance/payments/model/payment_gateway_event_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/*
  payment_gateway_events = LOG WEBHOOK / CALLBACK PAYMENT GATEWAY
  - Bisa banyak row per 1 payment (tiap callback / notif)
  - Nyimpen raw headers, payload, signature, status processing.
*/

type PaymentGatewayEventModel struct {
	GatewayEventID uuid.UUID `gorm:"column:gateway_event_id;type:uuid;default:gen_random_uuid();primaryKey" json:"gateway_event_id"`

	GatewayEventSchoolID  *uuid.UUID `gorm:"column:gateway_event_school_id;type:uuid" json:"gateway_event_school_id"`
	GatewayEventPaymentID *uuid.UUID `gorm:"column:gateway_event_payment_id;type:uuid" json:"gateway_event_payment_id"`

	// Provider & identitas event
	GatewayEventProvider    PaymentGatewayProvider `gorm:"column:gateway_event_provider;type:payment_gateway_provider;not null" json:"gateway_event_provider"`
	GatewayEventType        *string                `gorm:"column:gateway_event_type" json:"gateway_event_type"`
	GatewayEventExternalID  *string                `gorm:"column:gateway_event_external_id" json:"gateway_event_external_id"`
	GatewayEventExternalRef *string                `gorm:"column:gateway_event_external_ref" json:"gateway_event_external_ref"`

	// Raw data (buat debug / replay)
	GatewayEventHeaders   datatypes.JSON `gorm:"column:gateway_event_headers;type:jsonb" json:"gateway_event_headers"`
	GatewayEventPayload   datatypes.JSON `gorm:"column:gateway_event_payload;type:jsonb" json:"gateway_event_payload"`
	GatewayEventSignature *string        `gorm:"column:gateway_event_signature" json:"gateway_event_signature"`
	GatewayEventRawQuery  *string        `gorm:"column:gateway_event_raw_query" json:"gateway_event_raw_query"`

	// Status processing internal
	GatewayEventStatus   GatewayEventStatus `gorm:"column:gateway_event_status;type:gateway_event_status;not null;default:'received'" json:"gateway_event_status"`
	GatewayEventError    *string            `gorm:"column:gateway_event_error" json:"gateway_event_error"`
	GatewayEventTryCount int                `gorm:"column:gateway_event_try_count;not null;default:0" json:"gateway_event_try_count"`

	// Timestamps
	GatewayEventReceivedAt  time.Time  `gorm:"column:gateway_event_received_at;not null;default:now()" json:"gateway_event_received_at"`
	GatewayEventProcessedAt *time.Time `gorm:"column:gateway_event_processed_at" json:"gateway_event_processed_at"`

	GatewayEventCreatedAt time.Time  `gorm:"column:gateway_event_created_at;not null;default:now()" json:"gateway_event_created_at"`
	GatewayEventUpdatedAt time.Time  `gorm:"column:gateway_event_updated_at;not null;default:now()" json:"gateway_event_updated_at"`
	GatewayEventDeletedAt *time.Time `gorm:"column:gateway_event_deleted_at" json:"gateway_event_deleted_at"`
}

func (PaymentGatewayEventModel) TableName() string {
	return "payment_gateway_events"
}
