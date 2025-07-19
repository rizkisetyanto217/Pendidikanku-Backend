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
// 🟢 CREATE DONATION: Buat donasi baru & simpan snap token Midtrans, bisa tanpa login (guest) maupun dengan login (user)
func (ctrl *DonationController) CreateDonation(c *fiber.Ctx) error {
	var body dto.CreateDonationRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	var userUUID *uuid.UUID
	// 🔐 Jika pengguna login, ambil user ID dari JWT token
	userIDRaw := c.Locals("user_id")
	if userIDRaw != nil {
		userIDStr, ok := userIDRaw.(string)
		if !ok || userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User ID tidak valid",
			})
		}
		// Parse userID
		parsedUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "User ID dalam token tidak valid",
			})
		}
		userUUID = &parsedUUID
	}

	// 🧾 Generate order ID unik
	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())

	// 🧹 Bangun entitas donasi
	donation := model.Donation{
		DonationUserID:         userUUID,      // Jika tidak login, userUUID = nil
		DonationAmount:         body.Amount,
		DonationMessage:        body.Message,
		DonationStatus:         "pending",      // Status masih pending karena belum ada pembayaran
		DonationOrderID:        orderID,
		DonationPaymentGateway: "midtrans",     // Menggunakan Midtrans sebagai gateway
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

	// 📂 Update token pembayaran ke database
	donation.DonationPaymentToken = token
	ctrl.DB.Save(&donation)

	// ✅ Kirim response sukses dengan snap token
	return c.JSON(fiber.Map{
		"message":    "Donasi berhasil dibuat. Silakan lanjutkan pembayaran.",
		"order_id":   donation.DonationOrderID,
		"snap_token": token, // Snap token untuk pembayaran langsung
	})
}


// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleDonationStatusWebhook(db *gorm.DB, body map[string]interface{}) error {
	orderID := body["order_id"].(string)
	transactionStatus := body["transaction_status"].(string)

	// Cari donasi berdasarkan order ID
	var donation model.Donation
	if err := db.Where("donation_order_id = ?", orderID).First(&donation).Error; err != nil {
		return fmt.Errorf("donasi tidak ditemukan: %v", err)
	}

	// Update status donasi berdasarkan status transaksi
	switch transactionStatus {
	case "settlement", "success":
		// Pembayaran berhasil
		donation.DonationStatus = "completed"
	case "failed", "cancelled":
		// Pembayaran gagal
		donation.DonationStatus = "failed"
	default:
		// Status lainnya, misalnya pending
		donation.DonationStatus = "pending"
	}

	// Simpan perubahan status donasi ke database
	if err := db.Save(&donation).Error; err != nil {
		return fmt.Errorf("gagal memperbarui status donasi: %v", err)
	}

	return nil
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

    // Log untuk memastikan payload yang diterima
    log.Println("Received webhook:", body)

    // Ambil koneksi DB dari context (pastikan koneksi sudah diatur di middleware)
    db := c.Locals("db").(*gorm.DB)

    // 🔁 Proses webhook untuk memperbarui status donasi
    if err := ctrl.HandleDonationStatusWebhook(db, body); err != nil {
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
