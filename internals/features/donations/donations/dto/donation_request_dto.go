package dto

type CreateDonationRequest struct {
	DonationName     string `json:"donation_name" validate:"required"`               // Nama pendonor
	DonationEmail    string `json:"donation_email" validate:"required,email"`        // Email pendonor
	DonationMessage  string `json:"donation_message"`                                // Pesan/ucapan

	DonationAmount int `json:"donation_amount" validate:"required,gt=0"`             // Total seluruh donasi (masjid + masjidku)

	// 🔹 Breakdown donasi (opsional/fleksibel)
	DonationAmountMasjid             *int `json:"donation_amount_masjid" validate:"omitempty,gte=0"`
	DonationAmountMasjidku           *int `json:"donation_amount_masjidku" validate:"omitempty,gte=0"`
	DonationAmountMasjidkuToMasjid   *int `json:"donation_amount_masjidku_to_masjid" validate:"omitempty,gte=0"`
	DonationAmountMasjidkuToApp      *int `json:"donation_amount_masjidku_to_app" validate:"omitempty,gte=0"`

	// 🔗 Target donasi spesifik (optional)
	DonationTargetType *int   `json:"donation_target_type" validate:"omitempty,oneof=1 2 3 4"`
	DonationTargetID   string `json:"donation_target_id" validate:"omitempty,uuid"`
}
