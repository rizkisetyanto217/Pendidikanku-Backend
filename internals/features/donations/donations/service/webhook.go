package service

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	donationQuestionModel "masjidku_backend/internals/features/donations/donation_questions/model"
	donationModel "masjidku_backend/internals/features/donations/donations/model"
)

// HandleDonationStatusWebhook dipanggil saat menerima notifikasi dari Midtrans
func HandleDonationStatusWebhook(db *gorm.DB, body map[string]interface{}) error {
	orderID, ok1 := body["order_id"].(string)
	status, ok2 := body["transaction_status"].(string)

	if !ok1 || !ok2 {
		log.Println("[ERROR] Payload webhook tidak lengkap:", body)
		return fmt.Errorf("invalid payload")
	}

	log.Println("📄 Order ID:", orderID)
	log.Println("📌 Transaction Status:", status)

	// Ambil donasi berdasarkan order_id
	var donation donationModel.Donation
	if err := db.Where("donation_order_id = ?", orderID).First(&donation).Error; err != nil {
		log.Println("[ERROR] Donasi tidak ditemukan:", err)
		return fmt.Errorf("donation with order_id %s not found", orderID)
	}

	// Proses perubahan status donasi
	switch status {
	case "capture", "settlement":
		now := time.Now()
		donation.DonationStatus = "paid"
		donation.DonationPaidAt = &now

		// Hitung jumlah soal berdasarkan jumlah donasi
		totalSoal := donation.DonationAmount / 5000
		for i := 0; i < totalSoal; i++ {
			soal := donationQuestionModel.DonationQuestionModel{
				DonationQuestionDonationID:  donation.DonationID,
				DonationQuestionQuestionID:  0, // default, bisa diisi nanti
				DonationQuestionUserMessage: donation.DonationMessage,
			}
			if err := db.Create(&soal).Error; err != nil {
				log.Printf("[ERROR] Gagal membuat donation_question (%d/%d): %v", i+1, totalSoal, err)
			}
		}

	case "expire":
		donation.DonationStatus = "expired"
	case "cancel":
		donation.DonationStatus = "canceled"
	default:
		log.Println("[INFO] Status tidak diproses:", status)
	}

	// Simpan update status donasi ke database
	if err := db.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Gagal menyimpan status donasi:", err)
		return err
	}

	return nil
}
