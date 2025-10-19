package dto

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"masjidku_backend/internals/features/finance/payments/model"
)

/* =========================================================
   REQUEST DTOs  (nama field = sama persis dengan model.Payment)
   JSON tags = nama kolom DB (snake_case)
========================================================= */

// CreatePaymentRequest: bikin record payment baru
type CreatePaymentRequest struct {
	PaymentMasjidID *uuid.UUID `json:"payment_masjid_id" validate:"required"`
	PaymentUserID   *uuid.UUID `json:"payment_user_id" validate:"required"`

	PaymentUserSPPBillingID *uuid.UUID `json:"payment_user_spp_billing_id,omitempty"`
	PaymentSPPBillingID     *uuid.UUID `json:"payment_spp_billing_id,omitempty"`

	PaymentAmountIDR int    `json:"payment_amount_idr" validate:"required,gt=0"`
	PaymentCurrency  string `json:"payment_currency" validate:"omitempty,oneof=IDR" default:"IDR"`

	PaymentStatus string `json:"payment_status" validate:"omitempty,oneof=initiated pending awaiting_callback paid partially_refunded refunded failed canceled expired"`
	PaymentMethod string `json:"payment_method" validate:"required,oneof=gateway bank_transfer cash qris other"`

	PaymentGatewayProvider  *string `json:"payment_gateway_provider,omitempty" validate:"omitempty,oneof=midtrans xendit tripay duitku nicepay stripe paypal other"`
	PaymentExternalID       *string `json:"payment_external_id,omitempty"`
	PaymentGatewayReference *string `json:"payment_gateway_reference,omitempty"`
	PaymentCheckoutURL      *string `json:"payment_checkout_url,omitempty"`
	PaymentQRString         *string `json:"payment_qr_string,omitempty"`
	PaymentSignature        *string `json:"payment_signature,omitempty"`
	PaymentIdempotencyKey   *string `json:"payment_idempotency_key,omitempty"`

	PaymentRequestedAt *time.Time `json:"payment_requested_at,omitempty"`
	PaymentExpiresAt   *time.Time `json:"payment_expires_at,omitempty"`
	PaymentPaidAt      *time.Time `json:"payment_paid_at,omitempty"`
	PaymentCanceledAt  *time.Time `json:"payment_canceled_at,omitempty"`
	PaymentFailedAt    *time.Time `json:"payment_failed_at,omitempty"`
	PaymentRefundedAt  *time.Time `json:"payment_refunded_at,omitempty"`

	PaymentDescription *string        `json:"payment_description,omitempty"`
	PaymentNote        *string        `json:"payment_note,omitempty"`
	PaymentMeta        map[string]any `json:"payment_meta,omitempty"`
}

// CreateSnapRequest: generate Snap token untuk Payment tertentu
type CreateSnapRequest struct {
	PaymentID uuid.UUID      `json:"payment_id" validate:"required"`
	Customer  *CustomerInput `json:"customer,omitempty"`
}

type CustomerInput struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email" validate:"required,email"`
	Phone     string `json:"phone,omitempty"`
	Address   string `json:"address,omitempty"`
	City      string `json:"city,omitempty"`
	Postcode  string `json:"postcode,omitempty"`
	Country   string `json:"country,omitempty" default:"IDN"`
}

// UpdatePaymentStatusRequest: update status (webhook/admin)
type UpdatePaymentStatusRequest struct {
	PaymentID uuid.UUID `json:"payment_id" validate:"required"`

	PaymentStatus     string     `json:"payment_status" validate:"required,oneof=initiated pending awaiting_callback paid partially_refunded refunded failed canceled expired"`
	PaymentPaidAt     *time.Time `json:"payment_paid_at,omitempty"`
	PaymentFailedAt   *time.Time `json:"payment_failed_at,omitempty"`
	PaymentCanceledAt *time.Time `json:"payment_canceled_at,omitempty"`
	PaymentRefundedAt *time.Time `json:"payment_refunded_at,omitempty"`

	PaymentNote *string `json:"payment_note,omitempty"`
	// Patch metadata (akan di-merge ke payment_meta)
	PaymentMetaPatch map[string]any `json:"payment_meta_patch,omitempty"`
}

// ListPaymentsQuery: filter/pagination GET /payments
type ListPaymentsQuery struct {
	Page                    int        `json:"page" query:"page" validate:"omitempty,min=1"`
	Size                    int        `json:"size" query:"size" validate:"omitempty,min=1,max=200"`
	PaymentMasjidID         *uuid.UUID `json:"payment_masjid_id,omitempty" query:"payment_masjid_id"`
	PaymentUserID           *uuid.UUID `json:"payment_user_id,omitempty" query:"payment_user_id"`
	PaymentStatus           *string    `json:"payment_status,omitempty" query:"payment_status"`
	PaymentGatewayProvider  *string    `json:"payment_gateway_provider,omitempty" query:"payment_gateway_provider"`
	PaymentMethod           *string    `json:"payment_method,omitempty" query:"payment_method"`
	PaymentSPPBillingID     *uuid.UUID `json:"payment_spp_billing_id,omitempty" query:"payment_spp_billing_id"`
	PaymentUserSPPBillingID *uuid.UUID `json:"payment_user_spp_billing_id,omitempty" query:"payment_user_spp_billing_id"`
	From                    *time.Time `json:"from,omitempty" query:"from"`     // filter created_at >=
	To                      *time.Time `json:"to,omitempty" query:"to"`         // filter created_at <=
	Search                  *string    `json:"search,omitempty" query:"search"` // ext_id/ref/desc
}

/* =========================================================
   RESPONSE DTOs  (nama field = sama dengan model)
========================================================= */

type PaymentResponse struct {
	PaymentID uuid.UUID `json:"payment_id"`

	PaymentMasjidID *uuid.UUID `json:"payment_masjid_id,omitempty"`
	PaymentUserID   *uuid.UUID `json:"payment_user_id,omitempty"`

	PaymentUserSPPBillingID *uuid.UUID `json:"payment_user_spp_billing_id,omitempty"`
	PaymentSPPBillingID     *uuid.UUID `json:"payment_spp_billing_id,omitempty"`

	PaymentAmountIDR int    `json:"payment_amount_idr"`
	PaymentCurrency  string `json:"payment_currency"`

	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`

	PaymentGatewayProvider  *string `json:"payment_gateway_provider,omitempty"`
	PaymentExternalID       *string `json:"payment_external_id,omitempty"`
	PaymentGatewayReference *string `json:"payment_gateway_reference,omitempty"`
	PaymentCheckoutURL      *string `json:"payment_checkout_url,omitempty"`
	PaymentQRString         *string `json:"payment_qr_string,omitempty"`

	PaymentRequestedAt *time.Time `json:"payment_requested_at,omitempty"`
	PaymentExpiresAt   *time.Time `json:"payment_expires_at,omitempty"`
	PaymentPaidAt      *time.Time `json:"payment_paid_at,omitempty"`
	PaymentCanceledAt  *time.Time `json:"payment_canceled_at,omitempty"`
	PaymentFailedAt    *time.Time `json:"payment_failed_at,omitempty"`
	PaymentRefundedAt  *time.Time `json:"payment_refunded_at,omitempty"`

	PaymentDescription *string           `json:"payment_description,omitempty"`
	PaymentNote        *string           `json:"payment_note,omitempty"`
	PaymentMeta        datatypes.JSONMap `json:"payment_meta,omitempty"`

	CreatedAt time.Time `json:"payment_created_at"`
	UpdatedAt time.Time `json:"payment_updated_at"`
}

type ListPaymentsResponse struct {
	Data       []PaymentResponse `json:"data"`
	Page       int               `json:"page"`
	Size       int               `json:"size"`
	TotalItems int64             `json:"total_items"`
	TotalPages int               `json:"total_pages"`
}

/* =========================================================
   MAPPERS (1:1 nama field)
========================================================= */

func ToPaymentModel(req CreatePaymentRequest) model.Payment {
	var meta datatypes.JSONMap
	if req.PaymentMeta != nil {
		meta = datatypes.JSONMap(req.PaymentMeta)
	}
	// default currency
	if req.PaymentCurrency == "" {
		req.PaymentCurrency = "IDR"
	}
	// default status bila kosong
	if req.PaymentStatus == "" {
		req.PaymentStatus = model.PaymentStatusInitiated
	}

	return model.Payment{
		PaymentMasjidID:         req.PaymentMasjidID,
		PaymentUserID:           req.PaymentUserID,
		PaymentUserSPPBillingID: req.PaymentUserSPPBillingID,
		PaymentSPPBillingID:     req.PaymentSPPBillingID,

		PaymentAmountIDR: req.PaymentAmountIDR,
		PaymentCurrency:  req.PaymentCurrency,

		PaymentStatus: req.PaymentStatus,
		PaymentMethod: req.PaymentMethod,

		PaymentGatewayProvider:  req.PaymentGatewayProvider,
		PaymentExternalID:       req.PaymentExternalID,
		PaymentGatewayReference: req.PaymentGatewayReference,
		PaymentCheckoutURL:      req.PaymentCheckoutURL,
		PaymentQRString:         req.PaymentQRString,
		PaymentSignature:        req.PaymentSignature,
		PaymentIdempotencyKey:   req.PaymentIdempotencyKey,

		PaymentRequestedAt: req.PaymentRequestedAt,
		PaymentExpiresAt:   req.PaymentExpiresAt,
		PaymentPaidAt:      req.PaymentPaidAt,
		PaymentCanceledAt:  req.PaymentCanceledAt,
		PaymentFailedAt:    req.PaymentFailedAt,
		PaymentRefundedAt:  req.PaymentRefundedAt,

		PaymentDescription: req.PaymentDescription,
		PaymentNote:        req.PaymentNote,
		PaymentMeta:        meta,
	}
}

func ToPaymentResponse(m model.Payment) PaymentResponse {
	return PaymentResponse{
		PaymentID: m.PaymentID,

		PaymentMasjidID: m.PaymentMasjidID,
		PaymentUserID:   m.PaymentUserID,

		PaymentUserSPPBillingID: m.PaymentUserSPPBillingID,
		PaymentSPPBillingID:     m.PaymentSPPBillingID,

		PaymentAmountIDR: m.PaymentAmountIDR,
		PaymentCurrency:  m.PaymentCurrency,

		PaymentStatus: m.PaymentStatus,
		PaymentMethod: m.PaymentMethod,

		PaymentGatewayProvider:  m.PaymentGatewayProvider,
		PaymentExternalID:       m.PaymentExternalID,
		PaymentGatewayReference: m.PaymentGatewayReference,
		PaymentCheckoutURL:      m.PaymentCheckoutURL,
		PaymentQRString:         m.PaymentQRString,

		PaymentRequestedAt: m.PaymentRequestedAt,
		PaymentExpiresAt:   m.PaymentExpiresAt,
		PaymentPaidAt:      m.PaymentPaidAt,
		PaymentCanceledAt:  m.PaymentCanceledAt,
		PaymentFailedAt:    m.PaymentFailedAt,
		PaymentRefundedAt:  m.PaymentRefundedAt,

		PaymentDescription: m.PaymentDescription,
		PaymentNote:        m.PaymentNote,
		PaymentMeta:        m.PaymentMeta,

		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
