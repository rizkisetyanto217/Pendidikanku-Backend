package dto

import (
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/payment/spp/model"
)

/* ================== REQUESTS ================== */

// Create
type CreateUserSppBillingRequest struct {
	UserSppBillingBillingID  uuid.UUID                 `json:"user_spp_billing_billing_id"  validate:"required"`
	UserSppBillingUserID     *uuid.UUID                `json:"user_spp_billing_user_id"     validate:"omitempty"`
	UserSppBillingAmountIDR  int                       `json:"user_spp_billing_amount_idr"  validate:"required,gte=0"`
	UserSppBillingStatus     *m.UserSppBillingStatus   `json:"user_spp_billing_status"      validate:"omitempty,oneof=unpaid paid canceled"`
	UserSppBillingPaidAt     *time.Time                `json:"user_spp_billing_paid_at"     validate:"omitempty"`
	UserSppBillingNote       *string                   `json:"user_spp_billing_note"        validate:"omitempty"`
}

func (r CreateUserSppBillingRequest) ToModel() *m.UserSppBillingModel {
	status := m.SppUnpaid
	if r.UserSppBillingStatus != nil {
		status = *r.UserSppBillingStatus
	}
	return &m.UserSppBillingModel{
		UserSppBillingBillingID:  r.UserSppBillingBillingID,
		UserSppBillingUserID:     r.UserSppBillingUserID,
		UserSppBillingAmountIDR:  r.UserSppBillingAmountIDR,
		UserSppBillingStatus:     status,
		UserSppBillingPaidAt:     r.UserSppBillingPaidAt,
		UserSppBillingNote:       r.UserSppBillingNote,
	}
}

// Update (partial)
type UpdateUserSppBillingRequest struct {
	UserSppBillingUserID    *uuid.UUID               `json:"user_spp_billing_user_id"     validate:"omitempty"`
	UserSppBillingAmountIDR *int                     `json:"user_spp_billing_amount_idr"  validate:"omitempty,gte=0"`
	UserSppBillingStatus    *m.UserSppBillingStatus  `json:"user_spp_billing_status"      validate:"omitempty,oneof=unpaid paid canceled"`
	UserSppBillingPaidAt    *time.Time               `json:"user_spp_billing_paid_at"     validate:"omitempty"`
	UserSppBillingNote      *string                  `json:"user_spp_billing_note"        validate:"omitempty"`
}

// Terapkan perubahan ke model existing
func (r UpdateUserSppBillingRequest) ApplyTo(mo *m.UserSppBillingModel) {
	if r.UserSppBillingUserID != nil {
		mo.UserSppBillingUserID = r.UserSppBillingUserID
	}
	if r.UserSppBillingAmountIDR != nil {
		mo.UserSppBillingAmountIDR = *r.UserSppBillingAmountIDR
	}
	if r.UserSppBillingStatus != nil {
		mo.UserSppBillingStatus = *r.UserSppBillingStatus
	}
	if r.UserSppBillingPaidAt != nil {
		mo.UserSppBillingPaidAt = r.UserSppBillingPaidAt
	}
	if r.UserSppBillingNote != nil {
		mo.UserSppBillingNote = r.UserSppBillingNote
	}
}

// List / Query params
type ListUserSppBillingQuery struct {
	BillingID uuid.UUID                `query:"billing_id" validate:"required"`
	UserID    *uuid.UUID               `query:"user_id"    validate:"omitempty"`
	Status    *m.UserSppBillingStatus  `query:"status"     validate:"omitempty,oneof=unpaid paid canceled"`

	MinAmount *int `query:"min_amount" validate:"omitempty,gte=0"`
	MaxAmount *int `query:"max_amount" validate:"omitempty,gte=0"`

	PaidFrom *time.Time `query:"paid_from" validate:"omitempty"`
	PaidTo   *time.Time `query:"paid_to"   validate:"omitempty,gtefield=PaidFrom"`

	Q      *string `query:"q"      validate:"omitempty"`              // cari di note (opsional, jika dipakai)
	Limit  int     `query:"limit"  validate:"omitempty,gte=1,lte=100"`
	Offset int     `query:"offset" validate:"omitempty,gte=0"`
}

/* ================== RESPONSES ================== */

type UserSppBillingItemResponse struct {
	UserSppBillingID         uuid.UUID               `json:"user_spp_billing_id"`
	UserSppBillingBillingID  uuid.UUID               `json:"user_spp_billing_billing_id"`
	UserSppBillingUserID     *uuid.UUID              `json:"user_spp_billing_user_id,omitempty"`
	UserSppBillingAmountIDR  int                     `json:"user_spp_billing_amount_idr"`
	UserSppBillingStatus     m.UserSppBillingStatus  `json:"user_spp_billing_status"`
	UserSppBillingPaidAt     *time.Time              `json:"user_spp_billing_paid_at,omitempty"`
	UserSppBillingNote       *string                 `json:"user_spp_billing_note,omitempty"`
	UserSppBillingCreatedAt  time.Time               `json:"user_spp_billing_created_at"`
	UserSppBillingUpdatedAt  *time.Time              `json:"user_spp_billing_updated_at,omitempty"`
}

type UserSppBillingListResponse struct {
	Items []UserSppBillingItemResponse `json:"items"`
	Total int64                        `json:"total"`
}

/* ================== MAPPERS ================== */

func FromUserSppBillingModel(x m.UserSppBillingModel) UserSppBillingItemResponse {
	return UserSppBillingItemResponse{
		UserSppBillingID:        x.UserSppBillingID,
		UserSppBillingBillingID: x.UserSppBillingBillingID,
		UserSppBillingUserID:    x.UserSppBillingUserID,
		UserSppBillingAmountIDR: x.UserSppBillingAmountIDR,
		UserSppBillingStatus:    x.UserSppBillingStatus,
		UserSppBillingPaidAt:    x.UserSppBillingPaidAt,
		UserSppBillingNote:      x.UserSppBillingNote,
		UserSppBillingCreatedAt: x.UserSppBillingCreatedAt,
		UserSppBillingUpdatedAt: x.UserSppBillingUpdatedAt,
	}
}

func FromUserSppBillingModels(list []m.UserSppBillingModel, total int64) UserSppBillingListResponse {
	out := make([]UserSppBillingItemResponse, 0, len(list))
	for _, it := range list {
		out = append(out, FromUserSppBillingModel(it))
	}
	return UserSppBillingListResponse{Items: out, Total: total}
}



type ListMySppBillingQuery struct {
	Status *string   `query:"status"  validate:"omitempty,oneof=unpaid paid canceled"`
	Month  *int      `query:"month"   validate:"omitempty,min=1,max=12"`
	Year   *int      `query:"year"    validate:"omitempty,gte=2000,lte=2100"`
	Q      *string   `query:"q"       validate:"omitempty"` // cari di title
	Limit  int       `query:"limit"   validate:"omitempty,gte=1,lte=100"`
	Offset int       `query:"offset"  validate:"omitempty,gte=0"`
}

type MySppBillingItem struct {
	// from user_spp_billings
	UserSppBillingID        uuid.UUID  `json:"user_spp_billing_id"`
	UserSppBillingBillingID uuid.UUID  `json:"spp_billing_id"`
	UserSppBillingAmountIDR int        `json:"amount_idr"`
	UserSppBillingStatus    string     `json:"status"`
	UserSppBillingPaidAt    *time.Time `json:"paid_at,omitempty"`

	// from spp_billings (header)
	BillingTitle   string     `json:"title"`
	BillingMonth   int16      `json:"month"`
	BillingYear    int16      `json:"year"`
	BillingDueDate *time.Time `json:"due_date,omitempty"`
	BillingClassID *uuid.UUID `json:"class_id,omitempty"`
}

type MySppBillingListResponse struct {
	Items []MySppBillingItem `json:"items"`
	Total int64              `json:"total"`
}