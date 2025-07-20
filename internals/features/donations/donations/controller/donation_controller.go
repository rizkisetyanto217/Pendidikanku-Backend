// 📁 controller/donation_controller.go
package controller

import (
	"fmt"
	"log"
	"masjidku_backend/internals/features/donations/donations/dto"
	"masjidku_backend/internals/features/donations/donations/model"
	donationService "masjidku_backend/internals/features/donations/donations/service"
	modelMasjid "masjidku_backend/internals/features/masjids/masjids/model"
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
	// Mem-parsing body request
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] Gagal mem-parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	// Validasi amount yang lebih besar dari 0
	if body.DonationAmount <= 0 {
		log.Println("[ERROR] Amount must be greater than 0:", body.DonationAmount)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Amount must be greater than 0",
		})
	}

	var userUUID *uuid.UUID
	// Jika pengguna login, ambil user ID dari JWT token
	userIDRaw := c.Locals("user_id")
	if userIDRaw != nil {
		userIDStr, ok := userIDRaw.(string)
		if !ok || userIDStr == "" {
			log.Println("[ERROR] User ID tidak valid:", userIDRaw)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User ID tidak valid",
			})
		}
		// Parse userID
		parsedUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Println("[ERROR] User ID parsing failed:", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "User ID dalam token tidak valid",
			})
		}
		userUUID = &parsedUUID
	}

	// 🧾 Generate order ID unik
	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())
	log.Println("[INFO] Generated order ID:", orderID)

	// 🧹 Bangun entitas donasi
	donation := model.Donation{
		DonationUserID:         userUUID,          // Jika tidak login, userUUID = nil
		DonationName:           body.DonationName,
		DonationAmount:         body.DonationAmount,
		DonationMessage:        body.DonationMessage,
		DonationStatus:         "pending",          // Status masih pending karena belum ada pembayaran
		// DonationMasjidID:       body.DonationMasjidID,
		DonationOrderID:        orderID,
		DonationPaymentGateway: "midtrans",     
	}

	// Cek apakah DonationMasjidID ada dan valid, jika tidak, set sebagai NULL
	if body.DonationMasjidID != "" {
		// Convert DonationMasjidID ke UUID jika ada
		masjidUUID, err := uuid.Parse(body.DonationMasjidID)
		if err != nil {
			log.Println("[ERROR] Invalid Masjid ID format:", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid Masjid ID format",
			})
		}
		donation.DonationMasjidID = &masjidUUID
	} else {
		// Jika tidak ada masjid yang dipilih, set donation_masjid_id ke NULL
		donation.DonationMasjidID = nil
	}

	// 📂 Simpan donasi ke database
	if err := ctrl.DB.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Gagal menyimpan donasi:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan donasi",
		})
	}
	log.Println("[INFO] Donasi berhasil disimpan dengan order ID:", donation.DonationOrderID)

	// 🔐 Buat snap token Midtrans untuk pembayaran
	token, err := donationService.GenerateSnapToken(donation, body.DonationName, body.DonationEmail)
	if err != nil {
		log.Println("[ERROR] Gagal membuat token pembayaran:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat token pembayaran",
		})
	}
	log.Println("[INFO] Snap token berhasil dibuat:", token)

	// 📂 Update token pembayaran ke database
	donation.DonationPaymentToken = token
	if err := ctrl.DB.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Gagal memperbarui token pembayaran:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui token pembayaran",
		})
	}
	log.Println("[INFO] Token pembayaran diperbarui di database untuk order ID:", donation.DonationOrderID)

	// ✅ Kirim response sukses dengan snap token
	return c.JSON(fiber.Map{
		"message":    "Donasi berhasil dibuat. Silakan lanjutkan pembayaran.",
		"order_id":   donation.DonationOrderID,
		"snap_token": token, // Snap token untuk pembayaran langsung
	})
}


// 🟢 GET DONATIONS BY MASJID ID: Ambil semua donasi yang ditujukan ke masjid tertentu
func (ctrl *DonationController) GetDonationsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Slug masjid tidak boleh kosong",
		})
	}

	// 🔍 Cari masjid berdasarkan slug
	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		log.Println("[ERROR] Masjid dengan slug tidak ditemukan:", slug)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Masjid tidak ditemukan",
		})
	}

	// 🧾 Cari donasi berdasarkan masjid_id
	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ?", masjid.MasjidID).
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
