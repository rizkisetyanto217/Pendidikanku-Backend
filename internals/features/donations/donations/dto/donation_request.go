package dto

type CreateDonationRequest struct {
	DonationMasjidID   string `json:"donation_masjid_id"`   // ID masjid yang menerima donasi
	DonationAmount             int       `json:"donation_amount"`               // Jumlah donasi
	DonationMessage            string    `json:"donation_message"`              // Pesan donasi
	DonationName              string    `json:"donation_name"`                 // Nama pengdonasi
	DonationEmail              string    `json:"donation_email"`                // Email pengdonasi
}
