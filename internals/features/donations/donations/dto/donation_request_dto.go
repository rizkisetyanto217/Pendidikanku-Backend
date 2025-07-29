package dto

type CreateDonationRequest struct {
	DonationMasjidID    string `json:"donation_masjid_id" validate:"required,uuid"`     // ID masjid penerima
	DonationAmount      int    `json:"donation_amount" validate:"required,gt=0"`         // Nominal donasi (min 1)
	DonationMessage     string `json:"donation_message"`                                 // Pesan/ucapan dari pendonor
	DonationName        string `json:"donation_name" validate:"required"`                // Nama pendonor
	DonationEmail       string `json:"donation_email" validate:"required,email"`         // Email pendonor

	DonationTargetType  *int   `json:"donation_target_type" validate:"omitempty,oneof=1 2 3 4"` // 1=post, 2=lecture, dst
	DonationTargetID    string `json:"donation_target_id" validate:"omitempty,uuid"`     // UUID dari entitas target (opsional)
}
