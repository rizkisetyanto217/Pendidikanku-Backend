// 📁 controller/donation_controller.go
package controller

import (
	"fmt"
	"log"
	"masjidku_backend/internals/features/donations/donations/dto"
	"masjidku_backend/internals/features/donations/donations/model"
	donationService "masjidku_backend/internals/features/donations/donations/service"
	modelMasjid "masjidku_backend/internals/features/masjids/masjids/model"
	helper "masjidku_backend/internals/helpers"
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
		log.Println("[ERROR] BodyParser failed:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if body.DonationAmount <= 0 {
		log.Println("[ERROR] Invalid amount:", body.DonationAmount)
		return fiber.NewError(fiber.StatusBadRequest, "Amount must be greater than 0")
	}

	// 🔐 Ambil user ID dari token atau fallback ke dummy
	userUUID := helper.GetUserUUID(c)

	// 🔖 Buat order ID unik
	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())
	log.Println("[INFO] Generated order ID:", orderID)

	// 📝 Persiapkan objek Donasi
	donation := model.Donation{
		DonationUserID:         &userUUID,
		DonationName:           body.DonationName,
		DonationAmount:         body.DonationAmount,
		DonationMessage:        body.DonationMessage,
		DonationStatus:         "pending",
		DonationOrderID:        orderID,
		DonationPaymentGateway: "midtrans",
	}

	// ⛪ Masjid ID (opsional)
	if body.DonationMasjidID != "" {
		masjidUUID, err := uuid.Parse(body.DonationMasjidID)
		if err != nil {
			log.Println("[ERROR] Invalid Masjid ID:", err)
			return fiber.NewError(fiber.StatusBadRequest, "Invalid Masjid ID format")
		}
		donation.DonationMasjidID = &masjidUUID
	}

	// 💾 Simpan data donasi awal
	if err := ctrl.DB.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Failed to save donation:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan donasi")
	}

	// 💳 Buat Snap Token Midtrans
	token, err := donationService.GenerateSnapToken(donation, body.DonationName, body.DonationEmail)
	if err != nil {
		log.Println("[ERROR] GenerateSnapToken failed:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat token pembayaran")
	}
	log.Println("[INFO] Snap token created:", token)

	// 🔁 Update dengan Snap Token
	donation.DonationPaymentToken = token
	if err := ctrl.DB.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Failed to update token:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui token pembayaran")
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



// 🟢 GET DONATIONS BY MASJID SLUG: Ambil semua donasi *completed* berdasarkan slug masjid
func (ctrl *DonationController) GetDonationsByMasjidSlug(c *fiber.Ctx) error {
	// Ambil slug dari parameter URL
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Slug masjid tidak boleh kosong",
		})
	}

	// Cari masjid berdasarkan slug
	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Masjid dengan slug tersebut tidak ditemukan",
		})
	}

	// Ambil donasi yang statusnya 'completed' untuk masjid ini
	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ? AND donation_status = ?", masjid.MasjidID, "completed").
		Order("created_at desc").
		Find(&donations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data donasi berdasarkan slug masjid",
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
