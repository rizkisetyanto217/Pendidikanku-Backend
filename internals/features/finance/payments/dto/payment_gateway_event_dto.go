package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "masjidku_backend/internals/features/finance/payments/model"
)

/* =========================================================
   CREATE
========================================================= */

type CreatePaymentGatewayEventRequest struct {
	PaymentGatewayEventMasjidID  *uuid.UUID `json:"payment_gateway_event_masjid_id"`
	PaymentGatewayEventPaymentID *uuid.UUID `json:"payment_gateway_event_payment_id"`

	PaymentGatewayEventProvider string  `json:"payment_gateway_event_provider" validate:"required"` // enum: midtrans|xendit|...
	PaymentGatewayEventType     *string `json:"payment_gateway_event_type"`

	PaymentGatewayEventExternalID  *string `json:"payment_gateway_event_external_id"`
	PaymentGatewayEventExternalRef *string `json:"payment_gateway_event_external_ref"`

	PaymentGatewayEventHeaders   datatypes.JSON `json:"payment_gateway_event_headers"`
	PaymentGatewayEventPayload   datatypes.JSON `json:"payment_gateway_event_payload"`
	PaymentGatewayEventSignature *string        `json:"payment_gateway_event_signature"`
	PaymentGatewayEventRawQuery  *string        `json:"payment_gateway_event_raw_query"`

	PaymentGatewayEventStatus   *string `json:"payment_gateway_event_status"` // default: received
	PaymentGatewayEventError    *string `json:"payment_gateway_event_error"`
	PaymentGatewayEventTryCount *int    `json:"payment_gateway_event_try_count"`

	PaymentGatewayEventReceivedAt  *time.Time `json:"payment_gateway_event_received_at"`
	PaymentGatewayEventProcessedAt *time.Time `json:"payment_gateway_event_processed_at"`
}

func (r *CreatePaymentGatewayEventRequest) Validate() error {
	// provider whitelist
	if !inStr(r.PaymentGatewayEventProvider, model.PaymentProviderMidtrans, model.PaymentProviderXendit, model.PaymentProviderTripay,
		model.PaymentProviderDuitku, model.PaymentProviderNicepay, model.PaymentProviderStripe, model.PaymentProviderPaypal, model.PaymentProviderOther) {
		return errors.New("invalid payment_gateway_event_provider")
	}
	// status (jika diisi)
	if r.PaymentGatewayEventStatus != nil && !inStr(strings.ToLower(*r.PaymentGatewayEventStatus),
		model.GatewayEventStatusReceived, model.GatewayEventStatusProcessed, model.GatewayEventStatusIgnored,
		model.GatewayEventStatusDuplicated, model.GatewayEventStatusFailed) {
		return errors.New("invalid payment_gateway_event_status")
	}
	// try_count non negatif
	if r.PaymentGatewayEventTryCount != nil && *r.PaymentGatewayEventTryCount < 0 {
		return errors.New("payment_gateway_event_try_count must be >= 0")
	}
	return nil
}

func (r *CreatePaymentGatewayEventRequest) ToModel() *model.PaymentGatewayEvent {
	now := time.Now().UTC()
	out := &model.PaymentGatewayEvent{
		PaymentGatewayEventMasjidID:  r.PaymentGatewayEventMasjidID,
		PaymentGatewayEventPaymentID: r.PaymentGatewayEventPaymentID,

		PaymentGatewayEventProvider: r.PaymentGatewayEventProvider,
		PaymentGatewayEventType:     r.PaymentGatewayEventType,

		PaymentGatewayEventExternalID:  r.PaymentGatewayEventExternalID,
		PaymentGatewayEventExternalRef: r.PaymentGatewayEventExternalRef,

		PaymentGatewayEventHeaders:   r.PaymentGatewayEventHeaders,
		PaymentGatewayEventPayload:   r.PaymentGatewayEventPayload,
		PaymentGatewayEventSignature: r.PaymentGatewayEventSignature,
		PaymentGatewayEventRawQuery:  r.PaymentGatewayEventRawQuery,

		PaymentGatewayEventStatus:     model.GatewayEventStatusReceived,
		PaymentGatewayEventError:      r.PaymentGatewayEventError,
		PaymentGatewayEventTryCount:   0,
		PaymentGatewayEventReceivedAt: now,

		PaymentGatewayEventCreatedAt: now,
		PaymentGatewayEventUpdatedAt: now,
	}
	if r.PaymentGatewayEventStatus != nil {
		out.PaymentGatewayEventStatus = strings.ToLower(*r.PaymentGatewayEventStatus)
	}
	if r.PaymentGatewayEventTryCount != nil {
		out.PaymentGatewayEventTryCount = *r.PaymentGatewayEventTryCount
	}
	if r.PaymentGatewayEventReceivedAt != nil {
		out.PaymentGatewayEventReceivedAt = *r.PaymentGatewayEventReceivedAt
	}
	if r.PaymentGatewayEventProcessedAt != nil {
		out.PaymentGatewayEventProcessedAt = r.PaymentGatewayEventProcessedAt
	}
	return out
}

/* =========================================================
   UPDATE (PATCH)
========================================================= */

type UpdatePaymentGatewayEventRequest struct {
	PaymentGatewayEventMasjidID  PatchField[uuid.UUID] `json:"payment_gateway_event_masjid_id"`
	PaymentGatewayEventPaymentID PatchField[uuid.UUID] `json:"payment_gateway_event_payment_id"`

	PaymentGatewayEventProvider PatchField[string] `json:"payment_gateway_event_provider"`
	PaymentGatewayEventType     PatchField[string] `json:"payment_gateway_event_type"`

	PaymentGatewayEventExternalID  PatchField[string] `json:"payment_gateway_event_external_id"`
	PaymentGatewayEventExternalRef PatchField[string] `json:"payment_gateway_event_external_ref"`

	PaymentGatewayEventHeaders   PatchField[datatypes.JSON] `json:"payment_gateway_event_headers"`
	PaymentGatewayEventPayload   PatchField[datatypes.JSON] `json:"payment_gateway_event_payload"`
	PaymentGatewayEventSignature PatchField[string]         `json:"payment_gateway_event_signature"`
	PaymentGatewayEventRawQuery  PatchField[string]         `json:"payment_gateway_event_raw_query"`

	PaymentGatewayEventStatus   PatchField[string] `json:"payment_gateway_event_status"`
	PaymentGatewayEventError    PatchField[string] `json:"payment_gateway_event_error"`
	PaymentGatewayEventTryCount PatchField[int]    `json:"payment_gateway_event_try_count"`

	PaymentGatewayEventReceivedAt  PatchField[time.Time] `json:"payment_gateway_event_received_at"`
	PaymentGatewayEventProcessedAt PatchField[time.Time] `json:"payment_gateway_event_processed_at"`
}

func (p *UpdatePaymentGatewayEventRequest) Apply(m *model.PaymentGatewayEvent) error {
	applyPtr(&m.PaymentGatewayEventMasjidID, p.PaymentGatewayEventMasjidID)
	applyPtr(&m.PaymentGatewayEventPaymentID, p.PaymentGatewayEventPaymentID)

	// provider (enum)
	if p.PaymentGatewayEventProvider.Set {
		if p.PaymentGatewayEventProvider.Null || p.PaymentGatewayEventProvider.Value == nil {
			return errors.New("payment_gateway_event_provider cannot be null")
		}
		prov := strings.ToLower(*p.PaymentGatewayEventProvider.Value)
		if !inStr(prov, model.PaymentProviderMidtrans, model.PaymentProviderXendit, model.PaymentProviderTripay,
			model.PaymentProviderDuitku, model.PaymentProviderNicepay, model.PaymentProviderStripe, model.PaymentProviderPaypal, model.PaymentProviderOther) {
			return errors.New("invalid payment_gateway_event_provider")
		}
		m.PaymentGatewayEventProvider = prov
	}

	applyPtr(&m.PaymentGatewayEventType, p.PaymentGatewayEventType)
	applyPtr(&m.PaymentGatewayEventExternalID, p.PaymentGatewayEventExternalID)
	applyPtr(&m.PaymentGatewayEventExternalRef, p.PaymentGatewayEventExternalRef)

	applyVal(&m.PaymentGatewayEventHeaders, p.PaymentGatewayEventHeaders)
	applyVal(&m.PaymentGatewayEventPayload, p.PaymentGatewayEventPayload)
	applyPtr(&m.PaymentGatewayEventSignature, p.PaymentGatewayEventSignature)
	applyPtr(&m.PaymentGatewayEventRawQuery, p.PaymentGatewayEventRawQuery)

	// status (enum)
	if p.PaymentGatewayEventStatus.Set {
		if p.PaymentGatewayEventStatus.Null || p.PaymentGatewayEventStatus.Value == nil {
			return errors.New("payment_gateway_event_status cannot be null")
		}
		st := strings.ToLower(*p.PaymentGatewayEventStatus.Value)
		if !inStr(st, model.GatewayEventStatusReceived, model.GatewayEventStatusProcessed, model.GatewayEventStatusIgnored,
			model.GatewayEventStatusDuplicated, model.GatewayEventStatusFailed) {
			return errors.New("invalid payment_gateway_event_status")
		}
		m.PaymentGatewayEventStatus = st
	}

	applyPtr(&m.PaymentGatewayEventError, p.PaymentGatewayEventError)

	// try count (>=0)
	if p.PaymentGatewayEventTryCount.Set {
		if p.PaymentGatewayEventTryCount.Null || p.PaymentGatewayEventTryCount.Value == nil {
			return errors.New("payment_gateway_event_try_count cannot be null")
		}
		if *p.PaymentGatewayEventTryCount.Value < 0 {
			return errors.New("payment_gateway_event_try_count must be >= 0")
		}
		m.PaymentGatewayEventTryCount = *p.PaymentGatewayEventTryCount.Value
	}

	applyPtr(&m.PaymentGatewayEventProcessedAt, p.PaymentGatewayEventProcessedAt)
	if p.PaymentGatewayEventReceivedAt.Set {
		// received_at tidak boleh null & umumnya immutable, tapi kalau mau diizinkan set:
		if p.PaymentGatewayEventReceivedAt.Null || p.PaymentGatewayEventReceivedAt.Value == nil {
			return errors.New("payment_gateway_event_received_at cannot be null")
		}
		m.PaymentGatewayEventReceivedAt = *p.PaymentGatewayEventReceivedAt.Value
	}

	// updated_at dikelola oleh hook gorm BeforeUpdate
	return nil
}

/* =========================================================
   RESPONSE
========================================================= */

type PaymentGatewayEventResponse struct {
	PaymentGatewayEventID uuid.UUID `json:"payment_gateway_event_id"`

	PaymentGatewayEventMasjidID  *uuid.UUID `json:"payment_gateway_event_masjid_id,omitempty"`
	PaymentGatewayEventPaymentID *uuid.UUID `json:"payment_gateway_event_payment_id,omitempty"`

	PaymentGatewayEventProvider string  `json:"payment_gateway_event_provider"`
	PaymentGatewayEventType     *string `json:"payment_gateway_event_type,omitempty"`

	PaymentGatewayEventExternalID  *string `json:"payment_gateway_event_external_id,omitempty"`
	PaymentGatewayEventExternalRef *string `json:"payment_gateway_event_external_ref,omitempty"`

	PaymentGatewayEventHeaders   datatypes.JSON `json:"payment_gateway_event_headers,omitempty"`
	PaymentGatewayEventPayload   datatypes.JSON `json:"payment_gateway_event_payload,omitempty"`
	PaymentGatewayEventSignature *string        `json:"payment_gateway_event_signature,omitempty"`
	PaymentGatewayEventRawQuery  *string        `json:"payment_gateway_event_raw_query,omitempty"`

	PaymentGatewayEventStatus   string  `json:"payment_gateway_event_status"`
	PaymentGatewayEventError    *string `json:"payment_gateway_event_error,omitempty"`
	PaymentGatewayEventTryCount int     `json:"payment_gateway_event_try_count"`

	PaymentGatewayEventReceivedAt  time.Time  `json:"payment_gateway_event_received_at"`
	PaymentGatewayEventProcessedAt *time.Time `json:"payment_gateway_event_processed_at,omitempty"`

	PaymentGatewayEventCreatedAt time.Time  `json:"payment_gateway_event_created_at"`
	PaymentGatewayEventUpdatedAt time.Time  `json:"payment_gateway_event_updated_at"`
	PaymentGatewayEventDeletedAt *time.Time `json:"payment_gateway_event_deleted_at,omitempty"`
}

func FromModelPGW(m *model.PaymentGatewayEvent) *PaymentGatewayEventResponse {
	if m == nil {
		return nil
	}
	return &PaymentGatewayEventResponse{
		PaymentGatewayEventID: m.PaymentGatewayEventID,

		PaymentGatewayEventMasjidID:  m.PaymentGatewayEventMasjidID,
		PaymentGatewayEventPaymentID: m.PaymentGatewayEventPaymentID,

		PaymentGatewayEventProvider: m.PaymentGatewayEventProvider,
		PaymentGatewayEventType:     m.PaymentGatewayEventType,

		PaymentGatewayEventExternalID:  m.PaymentGatewayEventExternalID,
		PaymentGatewayEventExternalRef: m.PaymentGatewayEventExternalRef,

		PaymentGatewayEventHeaders:   m.PaymentGatewayEventHeaders,
		PaymentGatewayEventPayload:   m.PaymentGatewayEventPayload,
		PaymentGatewayEventSignature: m.PaymentGatewayEventSignature,
		PaymentGatewayEventRawQuery:  m.PaymentGatewayEventRawQuery,

		PaymentGatewayEventStatus:   m.PaymentGatewayEventStatus,
		PaymentGatewayEventError:    m.PaymentGatewayEventError,
		PaymentGatewayEventTryCount: m.PaymentGatewayEventTryCount,

		PaymentGatewayEventReceivedAt:  m.PaymentGatewayEventReceivedAt,
		PaymentGatewayEventProcessedAt: m.PaymentGatewayEventProcessedAt,

		PaymentGatewayEventCreatedAt: m.PaymentGatewayEventCreatedAt,
		PaymentGatewayEventUpdatedAt: m.PaymentGatewayEventUpdatedAt,
		PaymentGatewayEventDeletedAt: m.PaymentGatewayEventDeletedAt,
	}
}

/* =========================================================
   Utils
========================================================= */

func inStr(x string, set ...string) bool {
	x = strings.ToLower(strings.TrimSpace(x))
	for _, s := range set {
		if x == strings.ToLower(s) {
			return true
		}
	}
	return false
}