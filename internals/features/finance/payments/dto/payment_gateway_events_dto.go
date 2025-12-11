package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "madinahsalam_backend/internals/features/finance/payments/model"
	"madinahsalam_backend/internals/helpers/dbtime"
)

/* =========================================================
   CREATE
========================================================= */

type CreatePaymentGatewayEventRequest struct {
	GatewayEventSchoolID  *uuid.UUID `json:"gateway_event_school_id"`
	GatewayEventPaymentID *uuid.UUID `json:"gateway_event_payment_id"`

	// enum: midtrans|xendit|tripay|...
	GatewayEventProvider string  `json:"gateway_event_provider" validate:"required"`
	GatewayEventType     *string `json:"gateway_event_type"`

	GatewayEventExternalID  *string `json:"gateway_event_external_id"`
	GatewayEventExternalRef *string `json:"gateway_event_external_ref"`

	GatewayEventHeaders   datatypes.JSON `json:"gateway_event_headers"`
	GatewayEventPayload   datatypes.JSON `json:"gateway_event_payload"`
	GatewayEventSignature *string        `json:"gateway_event_signature"`
	GatewayEventRawQuery  *string        `json:"gateway_event_raw_query"`

	// default: received
	GatewayEventStatus   *string `json:"gateway_event_status"`
	GatewayEventError    *string `json:"gateway_event_error"`
	GatewayEventTryCount *int    `json:"gateway_event_try_count"`

	GatewayEventReceivedAt  *time.Time `json:"gateway_event_received_at"`
	GatewayEventProcessedAt *time.Time `json:"gateway_event_processed_at"`
}

func (r *CreatePaymentGatewayEventRequest) Validate() error {
	// provider whitelist (pakai enum di model)
	if !inStr(
		r.GatewayEventProvider,
		string(model.GatewayProviderMidtrans),
		string(model.GatewayProviderXendit),
		string(model.GatewayProviderTripay),
		string(model.GatewayProviderDuitku),
		string(model.GatewayProviderNicepay),
		string(model.GatewayProviderStripe),
		string(model.GatewayProviderPaypal),
		string(model.GatewayProviderOther),
	) {
		return errors.New("invalid gateway_event_provider")
	}

	// status (jika diisi) → harus salah satu dari enum di model
	if r.GatewayEventStatus != nil && !inStr(
		strings.ToLower(*r.GatewayEventStatus),
		string(model.GatewayEventStatusReceived),
		string(model.GatewayEventStatusProcessing),
		string(model.GatewayEventStatusSuccess),
		string(model.GatewayEventStatusFailed),
	) {
		return errors.New("invalid gateway_event_status")
	}

	// try_count non negatif
	if r.GatewayEventTryCount != nil && *r.GatewayEventTryCount < 0 {
		return errors.New("gateway_event_try_count must be >= 0")
	}
	return nil
}

func (r *CreatePaymentGatewayEventRequest) ToModel() *model.PaymentGatewayEventModel {
	now := time.Now().UTC()

	// provider & status dinormalisasi → lowercase lalu cast ke enum
	prov := model.PaymentGatewayProvider(strings.ToLower(r.GatewayEventProvider))

	status := model.GatewayEventStatusReceived
	if r.GatewayEventStatus != nil {
		status = model.GatewayEventStatus(strings.ToLower(*r.GatewayEventStatus))
	}

	tryCount := 0
	if r.GatewayEventTryCount != nil {
		tryCount = *r.GatewayEventTryCount
	}

	receivedAt := now
	if r.GatewayEventReceivedAt != nil {
		receivedAt = *r.GatewayEventReceivedAt
	}

	return &model.PaymentGatewayEventModel{
		GatewayEventSchoolID:  r.GatewayEventSchoolID,
		GatewayEventPaymentID: r.GatewayEventPaymentID,

		GatewayEventProvider:    prov,
		GatewayEventType:        r.GatewayEventType,
		GatewayEventExternalID:  r.GatewayEventExternalID,
		GatewayEventExternalRef: r.GatewayEventExternalRef,

		GatewayEventHeaders:   r.GatewayEventHeaders,
		GatewayEventPayload:   r.GatewayEventPayload,
		GatewayEventSignature: r.GatewayEventSignature,
		GatewayEventRawQuery:  r.GatewayEventRawQuery,

		GatewayEventStatus:      status,
		GatewayEventError:       r.GatewayEventError,
		GatewayEventTryCount:    tryCount,
		GatewayEventReceivedAt:  receivedAt,
		GatewayEventProcessedAt: r.GatewayEventProcessedAt,

		GatewayEventCreatedAt: now,
		GatewayEventUpdatedAt: now,
	}
}

/* =========================================================
   UPDATE (PATCH)
========================================================= */

type UpdatePaymentGatewayEventRequest struct {
	GatewayEventSchoolID  PatchField[uuid.UUID] `json:"gateway_event_school_id"`
	GatewayEventPaymentID PatchField[uuid.UUID] `json:"gateway_event_payment_id"`

	GatewayEventProvider PatchField[string] `json:"gateway_event_provider"`
	GatewayEventType     PatchField[string] `json:"gateway_event_type"`

	GatewayEventExternalID  PatchField[string] `json:"gateway_event_external_id"`
	GatewayEventExternalRef PatchField[string] `json:"gateway_event_external_ref"`

	GatewayEventHeaders   PatchField[datatypes.JSON] `json:"gateway_event_headers"`
	GatewayEventPayload   PatchField[datatypes.JSON] `json:"gateway_event_payload"`
	GatewayEventSignature PatchField[string]         `json:"gateway_event_signature"`
	GatewayEventRawQuery  PatchField[string]         `json:"gateway_event_raw_query"`

	GatewayEventStatus   PatchField[string] `json:"gateway_event_status"`
	GatewayEventError    PatchField[string] `json:"gateway_event_error"`
	GatewayEventTryCount PatchField[int]    `json:"gateway_event_try_count"`

	GatewayEventReceivedAt  PatchField[time.Time] `json:"gateway_event_received_at"`
	GatewayEventProcessedAt PatchField[time.Time] `json:"gateway_event_processed_at"`
}

func (p *UpdatePaymentGatewayEventRequest) Apply(m *model.PaymentGatewayEventModel) error {
	applyPtr(&m.GatewayEventSchoolID, p.GatewayEventSchoolID)
	applyPtr(&m.GatewayEventPaymentID, p.GatewayEventPaymentID)

	// provider (enum)
	if p.GatewayEventProvider.Set {
		if p.GatewayEventProvider.Null || p.GatewayEventProvider.Value == nil {
			return errors.New("gateway_event_provider cannot be null")
		}
		provStr := strings.ToLower(*p.GatewayEventProvider.Value)
		if !inStr(
			provStr,
			string(model.GatewayProviderMidtrans),
			string(model.GatewayProviderXendit),
			string(model.GatewayProviderTripay),
			string(model.GatewayProviderDuitku),
			string(model.GatewayProviderNicepay),
			string(model.GatewayProviderStripe),
			string(model.GatewayProviderPaypal),
			string(model.GatewayProviderOther),
		) {
			return errors.New("invalid gateway_event_provider")
		}
		m.GatewayEventProvider = model.PaymentGatewayProvider(provStr)
	}

	applyPtr(&m.GatewayEventType, p.GatewayEventType)
	applyPtr(&m.GatewayEventExternalID, p.GatewayEventExternalID)
	applyPtr(&m.GatewayEventExternalRef, p.GatewayEventExternalRef)

	applyVal(&m.GatewayEventHeaders, p.GatewayEventHeaders)
	applyVal(&m.GatewayEventPayload, p.GatewayEventPayload)
	applyPtr(&m.GatewayEventSignature, p.GatewayEventSignature)
	applyPtr(&m.GatewayEventRawQuery, p.GatewayEventRawQuery)

	// status (enum)
	if p.GatewayEventStatus.Set {
		if p.GatewayEventStatus.Null || p.GatewayEventStatus.Value == nil {
			return errors.New("gateway_event_status cannot be null")
		}
		st := strings.ToLower(*p.GatewayEventStatus.Value)
		if !inStr(
			st,
			string(model.GatewayEventStatusReceived),
			string(model.GatewayEventStatusProcessing),
			string(model.GatewayEventStatusSuccess),
			string(model.GatewayEventStatusFailed),
		) {
			return errors.New("invalid gateway_event_status")
		}
		m.GatewayEventStatus = model.GatewayEventStatus(st)
	}

	applyPtr(&m.GatewayEventError, p.GatewayEventError)

	// try count (>=0)
	if p.GatewayEventTryCount.Set {
		if p.GatewayEventTryCount.Null || p.GatewayEventTryCount.Value == nil {
			return errors.New("gateway_event_try_count cannot be null")
		}
		if *p.GatewayEventTryCount.Value < 0 {
			return errors.New("gateway_event_try_count must be >= 0")
		}
		m.GatewayEventTryCount = *p.GatewayEventTryCount.Value
	}

	applyPtr(&m.GatewayEventProcessedAt, p.GatewayEventProcessedAt)

	if p.GatewayEventReceivedAt.Set {
		if p.GatewayEventReceivedAt.Null || p.GatewayEventReceivedAt.Value == nil {
			return errors.New("gateway_event_received_at cannot be null")
		}
		m.GatewayEventReceivedAt = *p.GatewayEventReceivedAt.Value
	}

	return nil
}

/*
	=========================================================
	  RESPONSE

=========================================================
*/
type PaymentGatewayEventResponse struct {
	GatewayEventID uuid.UUID `json:"gateway_event_id"`

	GatewayEventSchoolID  *uuid.UUID `json:"gateway_event_school_id,omitempty"`
	GatewayEventPaymentID *uuid.UUID `json:"gateway_event_payment_id,omitempty"`

	GatewayEventProvider string  `json:"gateway_event_provider"`
	GatewayEventType     *string `json:"gateway_event_type,omitempty"`

	GatewayEventExternalID  *string `json:"gateway_event_external_id,omitempty"`
	GatewayEventExternalRef *string `json:"gateway_event_external_ref,omitempty"`

	GatewayEventHeaders   datatypes.JSON `json:"gateway_event_headers,omitempty"`
	GatewayEventPayload   datatypes.JSON `json:"gateway_event_payload,omitempty"`
	GatewayEventSignature *string        `json:"gateway_event_signature,omitempty"`
	GatewayEventRawQuery  *string        `json:"gateway_event_raw_query,omitempty"`

	GatewayEventStatus   string  `json:"gateway_event_status"`
	GatewayEventError    *string `json:"gateway_event_error,omitempty"`
	GatewayEventTryCount int     `json:"gateway_event_try_count"`

	GatewayEventReceivedAt  time.Time  `json:"gateway_event_received_at"`
	GatewayEventProcessedAt *time.Time `json:"gateway_event_processed_at,omitempty"`

	GatewayEventCreatedAt time.Time  `json:"gateway_event_created_at"`
	GatewayEventUpdatedAt time.Time  `json:"gateway_event_updated_at"`
	GatewayEventDeletedAt *time.Time `json:"gateway_event_deleted_at,omitempty"`
}

func FromModelPGW(c *fiber.Ctx, m *model.PaymentGatewayEventModel) *PaymentGatewayEventResponse {
	if m == nil {
		return nil
	}

	// 🔹 Konversi semua timestamptz ke timezone sekolah
	receivedAt := dbtime.ToSchoolTime(c, m.GatewayEventReceivedAt)
	processedAt := dbtime.ToSchoolTimePtr(c, m.GatewayEventProcessedAt)
	createdAt := dbtime.ToSchoolTime(c, m.GatewayEventCreatedAt)
	updatedAt := dbtime.ToSchoolTime(c, m.GatewayEventUpdatedAt)
	deletedAt := dbtime.ToSchoolTimePtr(c, m.GatewayEventDeletedAt)

	return &PaymentGatewayEventResponse{
		GatewayEventID: m.GatewayEventID,

		GatewayEventSchoolID:  m.GatewayEventSchoolID,
		GatewayEventPaymentID: m.GatewayEventPaymentID,

		GatewayEventProvider: string(m.GatewayEventProvider),
		GatewayEventType:     m.GatewayEventType,

		GatewayEventExternalID:  m.GatewayEventExternalID,
		GatewayEventExternalRef: m.GatewayEventExternalRef,

		GatewayEventHeaders:   m.GatewayEventHeaders,
		GatewayEventPayload:   m.GatewayEventPayload,
		GatewayEventSignature: m.GatewayEventSignature,
		GatewayEventRawQuery:  m.GatewayEventRawQuery,

		GatewayEventStatus:   string(m.GatewayEventStatus),
		GatewayEventError:    m.GatewayEventError,
		GatewayEventTryCount: m.GatewayEventTryCount,

		GatewayEventReceivedAt:  receivedAt,
		GatewayEventProcessedAt: processedAt,

		GatewayEventCreatedAt: createdAt,
		GatewayEventUpdatedAt: updatedAt,
		GatewayEventDeletedAt: deletedAt,
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
