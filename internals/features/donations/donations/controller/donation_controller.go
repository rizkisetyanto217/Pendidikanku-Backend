// 📁 controller/donation_controller.go
package controller

import (
	"fmt"
	"log"
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

// 🟢 CREATE DONATION: Buat donasi baru & generate snap token Midtrans
func (ctrl *DonationController) CreateDonation(c *fiber.Ctx) error {
	var body dto.CreateDonationRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	// 🔐 Ambil user ID dari JWT token
	userIDRaw := c.Locals("user_id")
	userIDStr, ok := userIDRaw.(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID tidak valid",
		})
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID dalam token tidak valid",
		})
	}

	// 🧾 Generate order ID unik
	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())

	// 🧹 Bangun entitas donasi
	donation := model.Donation{
		DonationUserID:         &userUUID,
		DonationAmount:         body.Amount,
		DonationMessage:        body.Message,
		DonationStatus:         "pending",
		DonationOrderID:        orderID,
		DonationPaymentGateway: "midtrans",
	}

	// 📂 Simpan donasi ke database
	if err := ctrl.DB.Create(&donation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan donasi",
		})
	}

	// 🔐 Buat snap token Midtrans untuk pembayaran
	token, err := donationService.GenerateSnapToken(donation, body.Name, body.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat token pembayaran",
		})
	}

	// 📂 Update payment token ke database
	donation.DonationPaymentToken = token
	ctrl.DB.Save(&donation)

	// ✅ Kirim response sukses
	return c.JSON(fiber.Map{
		"message":    "Donasi berhasil dibuat",
		"order_id":   donation.DonationOrderID,
		"snap_token": token,
	})
}

// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleMidtransNotification(c *fiber.Ctx) error {
	// 🔄 Ambil payload dari webhook
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid webhook",
		})
	}

	// 🧹 Ambil koneksi DB dari context
	db := c.Locals("db").(*gorm.DB)

	// 🔁 Proses webhook menggunakan service
	if err := donationService.HandleDonationStatusWebhook(db, body); err != nil {
		log.Println("[ERROR] Webhook gagal:", err)
		return c.SendStatus(500)
	}

	// ✅ Kirim status berhasil ke Midtrans
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
