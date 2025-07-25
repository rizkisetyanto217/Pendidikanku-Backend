// 📁 controller/donation_controller.go
package controller

import (
	"fmt"
	"log"
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/donations/donations/dto"
	"masjidku_backend/internals/features/donations/donations/model"
	donationService "masjidku_backend/internals/features/donations/donations/service"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DonationController struct {
	DB *gorm.DB
}

func NewDonationController(db *gorm.DB) *DonationController {
	return &DonationController{DB: db}
}


// 🟢 CREATE DONATION: Buat donasi baru & simpan snap token Midtrans, bisa tanpa login (guest) maupun dengan login (user)

func (ctrl *DonationController) CreateDonation(c *fiber.Ctx) error {
	var body dto.CreateDonationRequest
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] Gagal mem-parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	if body.DonationAmount <= 0 {
		log.Println("[ERROR] Amount must be greater than 0:", body.DonationAmount)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Amount must be greater than 0",
		})
	}

	// ⬇️ Default: Dummy User (guest)
	userUUID := constants.DummyUserID

	// Jika pengguna login, override dengan user_id dari token
	if userIDRaw := c.Locals("user_id"); userIDRaw != nil {
		if userIDStr, ok := userIDRaw.(string); ok && userIDStr != "" {
			if parsedUUID, err := uuid.Parse(userIDStr); err == nil {
				userUUID = parsedUUID
			} else {
				log.Println("[ERROR] User ID parsing failed:", err)
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "User ID dalam token tidak valid",
				})
			}
		}
	}

	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())
	log.Println("[INFO] Generated order ID:", orderID)

	donation := model.Donation{
		DonationUserID:         &userUUID,
		DonationName:           body.DonationName,
		DonationAmount:         body.DonationAmount,
		DonationMessage:        body.DonationMessage,
		DonationStatus:         "pending",
		DonationOrderID:        orderID,
		DonationPaymentGateway: "midtrans",
	}

	if body.DonationMasjidID != "" {
		if masjidUUID, err := uuid.Parse(body.DonationMasjidID); err == nil {
			donation.DonationMasjidID = &masjidUUID
		} else {
			log.Println("[ERROR] Invalid Masjid ID format:", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid Masjid ID format",
			})
		}
	}

	if err := ctrl.DB.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Gagal menyimpan donasi:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan donasi",
		})
	}
	log.Println("[INFO] Donasi berhasil disimpan dengan order ID:", donation.DonationOrderID)

	token, err := donationService.GenerateSnapToken(donation, body.DonationName, body.DonationEmail)
	if err != nil {
		log.Println("[ERROR] Gagal membuat token pembayaran:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat token pembayaran",
		})
	}
	log.Println("[INFO] Snap token berhasil dibuat:", token)

	donation.DonationPaymentToken = token
	if err := ctrl.DB.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Gagal memperbarui token pembayaran:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui token pembayaran",
		})
	}

	return c.JSON(fiber.Map{
		"message":    "Donasi berhasil dibuat. Silakan lanjutkan pembayaran.",
		"order_id":   donation.DonationOrderID,
		"snap_token": token,
	})
}



// 🟢 GET DONATIONS BY MASJID ID: Ambil semua donasi yang telah *completed* untuk masjid tertentu
func (ctrl *DonationController) GetDonationsByMasjidID(c *fiber.Ctx) error {
	// Ambil masjid_id dari parameter URL
	masjidIDParam := c.Params("masjid_id")

	// Validasi UUID masjid_id
	masjidID, err := uuid.Parse(masjidIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Masjid ID tidak valid",
		})
	}

	// Ambil donasi yang statusnya 'completed' dan ditujukan ke masjid ini
	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ? AND donation_status = ?", masjidID, "completed").
		Order("created_at desc").
		Find(&donations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data donasi berdasarkan masjid",
		})
	}

	return c.JSON(donations)
}



// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleDonationStatusWebhook(db *gorm.DB, body map[string]interface{}) error {
	// Pastikan `order_id` dan `transaction_status` ada dalam payload
	orderID, orderExists := body["order_id"].(string)
	transactionStatus, statusExists := body["transaction_status"].(string)

	if !orderExists || !statusExists {
		return fmt.Errorf("payload tidak valid: order_id atau transaction_status tidak ditemukan")
	}

	log.Printf("Processing webhook for order_id: %s with transaction_status: %s\n", orderID, transactionStatus)

	// Cari donasi berdasarkan order ID
	var donation model.Donation
	if err := db.Where("donation_order_id = ?", orderID).First(&donation).Error; err != nil {
		log.Printf("[ERROR] Donasi tidak ditemukan untuk order_id: %s\n", orderID)
		return fmt.Errorf("donasi tidak ditemukan untuk order_id %s: %v", orderID, err)
	}

	// Update status donasi berdasarkan status transaksi
	switch transactionStatus {
	case "settlement", "success":
		donation.DonationStatus = "completed"
	case "failed", "cancelled":
		donation.DonationStatus = "failed"
	default:
		donation.DonationStatus = "pending"
	}

	// Simpan perubahan status donasi ke database
	if err := db.Save(&donation).Error; err != nil {
		log.Printf("[ERROR] Gagal memperbarui status donasi untuk order_id: %s\n", orderID)
		return fmt.Errorf("gagal memperbarui status donasi untuk order_id %s: %v", orderID, err)
	}

	log.Printf("Status donasi untuk order_id: %s berhasil diperbarui menjadi: %s\n", orderID, donation.DonationStatus)

	return nil
}

// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleMidtransNotification(c *fiber.Ctx) error {
	// 🔄 Ambil payload dari webhook
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] Gagal memparsing body webhook:", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid webhook",
		})
	}

	// Log untuk memastikan payload yang diterima
	log.Println("Received webhook:", body)

	// Ambil koneksi DB dari context (pastikan koneksi sudah diatur di middleware)
	db := c.Locals("db").(*gorm.DB)
	if db == nil {
		log.Println("[ERROR] Koneksi database tidak tersedia")
		return c.Status(500).JSON(fiber.Map{
			"error": "Koneksi database tidak tersedia",
		})
	}

	// 🔁 Proses webhook untuk memperbarui status donasi
	if err := ctrl.HandleDonationStatusWebhook(db, body); err != nil {
		log.Println("[ERROR] Webhook gagal:", err)
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Webhook gagal: %v", err),
		})
	}

	// ✅ Kirim status berhasil ke Midtrans
	log.Println("Webhook processed successfully")
	return c.SendStatus(200)
}


// 🟢 GET ALL DONATIONS: Ambil seluruh data donasi (admin)
func (ctrl *DonationController) GetAllDonations(c *fiber.Ctx) error {
	// 🔍 Query semua data donasi, urutkan dari yang terbaru
	var donations []model.Donation
	if err := ctrl.DB.Order("created_at desc").Find(&donations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data donasi",
		})
	}

	// ✅ Kirim data donasi
	return c.JSON(donations)
}

// 🟢 GET DONATIONS BY USER ID: Ambil donasi milik user tertentu
func (ctrl *DonationController) GetDonationsByUserID(c *fiber.Ctx) error {
	// 🔹 Ambil user_id dari parameter URL
	userIDParam := c.Params("user_id")

	// 🔁 Validasi dan konversi user_id ke UUID
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id tidak valid",
		})
	}

	// 🔍 Ambil semua donasi milik user ini
	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_user_id = ?", userID).
		Order("created_at desc").
		Find(&donations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data donasi user",
		})
	}

	// ✅ Kirim data donasi user
	return c.JSON(donations)
}
