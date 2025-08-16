package dto

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

/* ===================== Constants (samakan dengan model) ===================== */

const (
	DonationTargetPost     = 1
	DonationTargetLecture  = 2
	DonationTargetCampaign = 3
	DonationTargetSPP      = 4
	DonationTargetOther    = 5
)

/* ===================== DTO ===================== */

type CreateDonationRequest struct {
	// Identitas pendonor (email opsional disimpan terpisah ke tabel contact/log, jika DB utamanya belum ada kolomnya)
	DonationName    string  `json:"donation_name" validate:"required"`        // Nama pendonor
	DonationEmail   *string `json:"donation_email" validate:"omitempty,email"`// Email pendonor (opsional)
	DonationMessage *string `json:"donation_message"`                         // Pesan/ucapan (opsional)

	// Nominal total
	DonationAmount int `json:"donation_amount" validate:"required,gt=0"`      // Total seluruh donasi

	// Breakdown (opsional)
	DonationAmountMasjid           *int `json:"donation_amount_masjid" validate:"omitempty,gte=0"`
	DonationAmountMasjidku         *int `json:"donation_amount_masjidku" validate:"omitempty,gte=0"`
	DonationAmountMasjidkuToMasjid *int `json:"donation_amount_masjidku_to_masjid" validate:"omitempty,gte=0"`
	DonationAmountMasjidkuToApp    *int `json:"donation_amount_masjidku_to_app" validate:"omitempty,gte=0"`

	// Target donasi (XOR): SPP ATAU target umum
	DonationTargetType        *int    `json:"donation_target_type" validate:"omitempty,oneof=1 2 3 4 5"`
	DonationTargetID          *string `json:"donation_target_id" validate:"omitempty,uuid"` // untuk 1/2/3/5
	DonationUserSPPBillingID  *string `json:"donation_user_spp_billing_id" validate:"omitempty,uuid"` // untuk 4 (SPP)

	// Opsional: grup order multi-item (kalau dipakai)
	DonationParentOrderID *string `json:"donation_parent_order_id" validate:"omitempty,max=120"`
}

/* ===================== Helper: safe getters ===================== */

func strPtrToUUID(p *string) (*uuid.UUID, error) {
	if p == nil || *p == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*p)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

/* ===================== Validation Logic ===================== */

func (r *CreateDonationRequest) Validate(v *validator.Validate) error {
	if v == nil {
		v = validator.New()
	}
	// 1) Validate basic tags
	if err := v.Struct(r); err != nil {
		return err
	}

	// 2) XOR rule: SPP vs target umum
	if r.DonationTargetType != nil {
		switch *r.DonationTargetType {
		case DonationTargetSPP:
			// Harus ada SPP billing id, dan TIDAK boleh ada target id
			if r.DonationUserSPPBillingID == nil || *r.DonationUserSPPBillingID == "" {
				return errors.New("donation_user_spp_billing_id wajib diisi untuk target_type = SPP (4)")
			}
			if r.DonationTargetID != nil && *r.DonationTargetID != "" {
				return errors.New("donation_target_id harus kosong ketika target_type = SPP (4)")
			}
		case DonationTargetPost, DonationTargetLecture, DonationTargetCampaign, DonationTargetOther:
			// Harus ada target id, dan TIDAK boleh ada SPP billing id
			if r.DonationTargetID == nil || *r.DonationTargetID == "" {
				return errors.New("donation_target_id wajib diisi untuk target_type non-SPP (1,2,3,5)")
			}
			if r.DonationUserSPPBillingID != nil && *r.DonationUserSPPBillingID != "" {
				return errors.New("donation_user_spp_billing_id harus kosong ketika target_type non-SPP (1,2,3,5)")
			}
		default:
			return errors.New("donation_target_type tidak valid")
		}
	}

	// 3) Sum breakdown ≤ total
	sum := 0
	if r.DonationAmountMasjid != nil {
		sum += *r.DonationAmountMasjid
	}
	if r.DonationAmountMasjidku != nil {
		sum += *r.DonationAmountMasjidku
	}
	if r.DonationAmountMasjidkuToMasjid != nil {
		sum += *r.DonationAmountMasjidkuToMasjid
	}
	if r.DonationAmountMasjidkuToApp != nil {
		sum += *r.DonationAmountMasjidkuToApp
	}
	if sum > r.DonationAmount {
		return errors.New("jumlah breakdown melebihi donation_amount")
	}

	// 4) Validasi format UUID untuk field string UUID (defensif)
	if _, err := strPtrToUUID(r.DonationTargetID); err != nil {
		return errors.New("donation_target_id bukan UUID yang valid")
	}
	if _, err := strPtrToUUID(r.DonationUserSPPBillingID); err != nil {
		return errors.New("donation_user_spp_billing_id bukan UUID yang valid")
	}

	return nil
}
