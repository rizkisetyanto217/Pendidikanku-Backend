package dto

import "github.com/google/uuid"

type CreateDonationRequest struct {
	DonationMasjidID   uuid.UUID `json:"donation_masjid_id"`   // ID masjid yang menerima donasi
	DonationAmount             int       `json:"donation_amount"`               // Jumlah donasi
	DonationMessage            string    `json:"donation_message"`              // Pesan donasi
	DonationName              string    `json:"donation_name"`                 // Nama pengdonasi
	DonationEmail              string    `json:"donation_email"`                // Email pengdonasi
}
