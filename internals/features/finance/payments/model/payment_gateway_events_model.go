// file: internals/features/finance/payment_gateway_events/model/payment_gateway_event.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   ENUM mirrors (Go-side)
   ========================= */

const (
	// gateway_event_status (tipe ENUM di Postgres tetap)
	GatewayEventStatusReceived   = "received"
	GatewayEventStatusProcessed  = "processed"
	GatewayEventStatusIgnored    = "ignored"
	GatewayEventStatusDuplicated = "duplicated"
	GatewayEventStatusFailed     = "failed"
)

const (
	// payment_gateway_provider
	PaymentProviderMidtrans = "midtrans"
	PaymentProviderXendit   = "xendit"
	PaymentProviderTripay   = "tripay"
	PaymentProviderDuitku   = "duitku"
	PaymentProviderNicepay  = "nicepay"
	PaymentProviderStripe   = "stripe"
	PaymentProviderPaypal   = "paypal"
	PaymentProviderOther    = "other"
)

/* =========================
   Model: payment_gateway_events
   ========================= */

type PaymentGatewayEvent struct {
	PaymentGatewayEventID uuid.UUID `json:"payment_gateway_event_id" gorm:"column:payment_gateway_event_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// Nullable FKs (ON DELETE SET NULL)
	PaymentGatewayEventSchoolID  *uuid.UUID `json:"payment_gateway_event_school_id,omitempty"  gorm:"column:payment_gateway_event_school_id;type:uuid"`
	PaymentGatewayEventPaymentID *uuid.UUID `json:"payment_gateway_event_payment_id,omitempty" gorm:"column:payment_gateway_event_payment_id;type:uuid"`

	// Enums
	PaymentGatewayEventProvider string `json:"payment_gateway_event_provider" gorm:"column:payment_gateway_event_provider;type:payment_gateway_provider;not null"`

	// Identitas/metadata event
	PaymentGatewayEventType        *string `json:"payment_gateway_event_type,omitempty"         gorm:"column:payment_gateway_event_type;type:text"`
	PaymentGatewayEventExternalID  *string `json:"payment_gateway_event_external_id,omitempty"  gorm:"column:payment_gateway_event_external_id;type:text"`
	PaymentGatewayEventExternalRef *string `json:"payment_gateway_event_external_ref,omitempty" gorm:"column:payment_gateway_event_external_ref;type:text"`

	// Payloads
	PaymentGatewayEventHeaders   datatypes.JSON `json:"payment_gateway_event_headers,omitempty"  gorm:"column:payment_gateway_event_headers;type:jsonb"`
	PaymentGatewayEventPayload   datatypes.JSON `json:"payment_gateway_event_payload,omitempty"  gorm:"column:payment_gateway_event_payload;type:jsonb"`
	PaymentGatewayEventSignature *string        `json:"payment_gateway_event_signature,omitempty" gorm:"column:payment_gateway_event_signature;type:text"`
	PaymentGatewayEventRawQuery  *string        `json:"payment_gateway_event_raw_query,omitempty" gorm:"column:payment_gateway_event_raw_query;type:text"`

	// Status & retries
	PaymentGatewayEventStatus   string  `json:"payment_gateway_event_status"   gorm:"column:payment_gateway_event_status;type:gateway_event_status;not null;default:'received'"`
	PaymentGatewayEventError    *string `json:"payment_gateway_event_error,omitempty" gorm:"column:payment_gateway_event_error;type:text"`
	PaymentGatewayEventTryCount int     `json:"payment_gateway_event_try_count" gorm:"column:payment_gateway_event_try_count;type:int;not null;default:0"`

	// Timestamps (soft delete manual)
	PaymentGatewayEventReceivedAt  time.Time  `json:"payment_gateway_event_received_at"  gorm:"column:payment_gateway_event_received_at;type:timestamptz;not null;default:now()"`
	PaymentGatewayEventProcessedAt *time.Time `json:"payment_gateway_event_processed_at,omitempty" gorm:"column:payment_gateway_event_processed_at;type:timestamptz"`

	PaymentGatewayEventCreatedAt time.Time  `json:"payment_gateway_event_created_at"           gorm:"column:payment_gateway_event_created_at;type:timestamptz;not null;default:now()"`
	PaymentGatewayEventUpdatedAt time.Time  `json:"payment_gateway_event_updated_at"           gorm:"column:payment_gateway_event_updated_at;type:timestamptz;not null;default:now()"`
	PaymentGatewayEventDeletedAt *time.Time `json:"payment_gateway_event_deleted_at,omitempty" gorm:"column:payment_gateway_event_deleted_at;type:timestamptz"`
}

func (PaymentGatewayEvent) TableName() string { return "payment_gateway_events" }

/* =========================
   Hooks: keep updated_at fresh
   ========================= */

func (e *PaymentGatewayEvent) BeforeCreate(tx *gorm.DB) error {
	e.PaymentGatewayEventUpdatedAt = time.Now().UTC()
	return nil
}
func (e *PaymentGatewayEvent) BeforeUpdate(tx *gorm.DB) error {
	e.PaymentGatewayEventUpdatedAt = time.Now().UTC()
	return nil
}

/* =========================
   Scopes
   ========================= */

func ScopePGWAlive(db *gorm.DB) *gorm.DB {
	return db.Where("payment_gateway_event_deleted_at IS NULL")
}
func ScopePGWByProvider(provider string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("payment_gateway_event_provider = ?", provider)
	}
}
func ScopePGWByPayment(paymentID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("payment_gateway_event_payment_id = ?", paymentID)
	}
}
func ScopePGWBySchool(schoolID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("payment_gateway_event_school_id = ?", schoolID)
	}
}
func ScopePGWByStatus(status string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("payment_gateway_event_status = ?", status)
	}
}

/* =========================
   Convenience helpers (opsional)
   ========================= */

func (e *PaymentGatewayEvent) MarkProcessed() {
	now := time.Now().UTC()
	e.PaymentGatewayEventProcessedAt = &now
	e.PaymentGatewayEventStatus = GatewayEventStatusProcessed
}

func (e *PaymentGatewayEvent) IncrementTryCount() {
	e.PaymentGatewayEventTryCount++
}
