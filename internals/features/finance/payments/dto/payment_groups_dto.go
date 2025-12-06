// file: internals/features/finance/payments/dto/payment_group_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "madinahsalam_backend/internals/features/finance/payments/model"
)

/* =========================================================
   RESPONSE DTO
========================================================= */

type PaymentGroupResponse struct {
	PaymentGroupID              uuid.UUID  `json:"payment_group_id"`
	PaymentGroupSchoolID        *uuid.UUID `json:"payment_group_school_id,omitempty"`
	PaymentGroupUserID          *uuid.UUID `json:"payment_group_user_id,omitempty"`
	PaymentGroupSchoolStudentID *uuid.UUID `json:"payment_group_school_student_id,omitempty"`
	PaymentGroupNumber          *int64     `json:"payment_group_number,omitempty"`

	PaymentGroupAmountIDR int    `json:"payment_group_amount_idr"`
	PaymentGroupCurrency  string `json:"payment_group_currency"`

	PaymentGroupStatus          model.PaymentStatus           `json:"payment_group_status"`
	PaymentGroupMethod          model.PaymentMethod           `json:"payment_group_method"`
	PaymentGroupGatewayProvider *model.PaymentGatewayProvider `json:"payment_group_gateway_provider,omitempty"`

	PaymentGroupExternalID       *string `json:"payment_group_external_id,omitempty"`
	PaymentGroupGatewayReference *string `json:"payment_group_gateway_reference,omitempty"`
	PaymentGroupCheckoutURL      *string `json:"payment_group_checkout_url,omitempty"`
	PaymentGroupQRString         *string `json:"payment_group_qr_string,omitempty"`
	PaymentGroupSignature        *string `json:"payment_group_signature,omitempty"`
	PaymentGroupIdempotencyKey   *string `json:"payment_group_idempotency_key,omitempty"`

	PaymentGroupRequestedAt time.Time  `json:"payment_group_requested_at"`
	PaymentGroupExpiresAt   *time.Time `json:"payment_group_expires_at,omitempty"`
	PaymentGroupPaidAt      *time.Time `json:"payment_group_paid_at,omitempty"`
	PaymentGroupCanceledAt  *time.Time `json:"payment_group_canceled_at,omitempty"`
	PaymentGroupFailedAt    *time.Time `json:"payment_group_failed_at,omitempty"`
	PaymentGroupRefundedAt  *time.Time `json:"payment_group_refunded_at,omitempty"`

	PaymentGroupDescription *string        `json:"payment_group_description,omitempty"`
	PaymentGroupNote        *string        `json:"payment_group_note,omitempty"`
	PaymentGroupMeta        datatypes.JSON `json:"payment_group_meta,omitempty"`

	PaymentGroupCreatedAt time.Time `json:"payment_group_created_at"`
	PaymentGroupUpdatedAt time.Time `json:"payment_group_updated_at"`
}

/* =========================================================
   MAPPERS
========================================================= */

func FromPaymentGroupModel(m *model.PaymentGroupModel) PaymentGroupResponse {
	if m == nil {
		// caller sebaiknya nggak kirim nil, tapi biar aman kita balik struct kosong
		return PaymentGroupResponse{}
	}

	return PaymentGroupResponse{
		PaymentGroupID:              m.PaymentGroupID,
		PaymentGroupSchoolID:        m.PaymentGroupSchoolID,
		PaymentGroupUserID:          m.PaymentGroupUserID,
		PaymentGroupSchoolStudentID: m.PaymentGroupSchoolStudentID,
		PaymentGroupNumber:          m.PaymentGroupNumber,

		PaymentGroupAmountIDR: m.PaymentGroupAmountIDR,
		PaymentGroupCurrency:  m.PaymentGroupCurrency,

		PaymentGroupStatus:          m.PaymentGroupStatus,
		PaymentGroupMethod:          m.PaymentGroupMethod,
		PaymentGroupGatewayProvider: m.PaymentGroupGatewayProvider,

		PaymentGroupExternalID:       m.PaymentGroupExternalID,
		PaymentGroupGatewayReference: m.PaymentGroupGatewayReference,
		PaymentGroupCheckoutURL:      m.PaymentGroupCheckoutURL,
		PaymentGroupQRString:         m.PaymentGroupQRString,
		PaymentGroupSignature:        m.PaymentGroupSignature,
		PaymentGroupIdempotencyKey:   m.PaymentGroupIdempotencyKey,

		PaymentGroupRequestedAt: m.PaymentGroupRequestedAt,
		PaymentGroupExpiresAt:   m.PaymentGroupExpiresAt,
		PaymentGroupPaidAt:      m.PaymentGroupPaidAt,
		PaymentGroupCanceledAt:  m.PaymentGroupCanceledAt,
		PaymentGroupFailedAt:    m.PaymentGroupFailedAt,
		PaymentGroupRefundedAt:  m.PaymentGroupRefundedAt,

		PaymentGroupDescription: m.PaymentGroupDescription,
		PaymentGroupNote:        m.PaymentGroupNote,
		PaymentGroupMeta:        m.PaymentGroupMeta,

		PaymentGroupCreatedAt: m.PaymentGroupCreatedAt,
		PaymentGroupUpdatedAt: m.PaymentGroupUpdatedAt,
	}
}

func FromPaymentGroupModels(list []model.PaymentGroupModel) []PaymentGroupResponse {
	out := make([]PaymentGroupResponse, 0, len(list))
	for i := range list {
		out = append(out, FromPaymentGroupModel(&list[i]))
	}
	return out
}
