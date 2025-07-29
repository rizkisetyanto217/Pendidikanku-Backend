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


// 🟢 CREATE DONATION (by slug): Buat donasi baru berdasarkan slug masjid
func (ctrl *DonationController) CreateDonation(c *fiber.Ctx) error {
	var body dto.CreateDonationRequest
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] BodyParser failed:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Ambil slug dari URL
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid tidak boleh kosong")
	}

	// 🎯 Cari masjid berdasarkan slug
	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		log.Println("[ERROR] Masjid not found by slug:", slug)
		return fiber.NewError(fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// 🔢 Hitung total donation_amount
	total := 0
	if body.DonationAmountMasjid != nil {
		total += *body.DonationAmountMasjid
	}
	if body.DonationAmountMasjidku != nil {
		total += *body.DonationAmountMasjidku
	}
	if total <= 0 {
		log.Println("[ERROR] Total donation amount invalid:", total)
		return fiber.NewError(fiber.StatusBadRequest, "Total amount must be greater than 0")
	}

	// 🔐 Ambil user ID
	userUUID := helper.GetUserUUID(c)
	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())

	// 🔀 Split masjidku
	var amountToMasjid, amountToApp *int
	if body.DonationAmountMasjidku != nil {
		half := *body.DonationAmountMasjidku / 2
		amountToMasjid = &half
		amountToApp = new(int)
		*amountToApp = *body.DonationAmountMasjidku - half
	}

	// 💾 Persiapkan objek donasi
	donation := model.Donation{
		DonationUserID:                 &userUUID,
		DonationName:                   body.DonationName,
		DonationMessage:                body.DonationMessage,
		DonationStatus:                 "pending",
		DonationOrderID:                orderID,
		DonationPaymentGateway:         "midtrans",
		DonationMasjidID:               &masjid.MasjidID,
		DonationAmount:                 total,
		DonationAmountMasjid:           body.DonationAmountMasjid,
		DonationAmountMasjidku:         body.DonationAmountMasjidku,
		DonationAmountMasjidkuToMasjid: coalesceIntPtr(body.DonationAmountMasjidkuToMasjid, amountToMasjid),
		DonationAmountMasjidkuToApp:    coalesceIntPtr(body.DonationAmountMasjidkuToApp, amountToApp),
	}

	// Optional: Target donasi
	if body.DonationTargetType != nil {
		donation.DonationTargetType = body.DonationTargetType
	}
	if body.DonationTargetID != "" {
		if targetUUID, err := uuid.Parse(body.DonationTargetID); err == nil {
			donation.DonationTargetID = &targetUUID
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid Target ID format")
		}
	}

	// Simpan ke DB
	if err := ctrl.DB.Save(&donation).Error; err != nil {
		log.Println("[ERROR] Failed to save donation:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan donasi")
	}

	// Snap token
	token, err := donationService.GenerateSnapToken(donation, body.DonationName, body.DonationEmail)
	if err != nil {
		log.Println("[ERROR] GenerateSnapToken failed:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat token pembayaran")
	}
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


func coalesceIntPtr(preferred *int, fallback *int) *int {
	if preferred != nil {
		return preferred
	}
	return fallback
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
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid tidak boleh kosong")
	}

	// 🔍 Cari masjid berdasarkan slug
	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Masjid dengan slug tersebut tidak ditemukan")
	}

	// 🧑 Ambil user_id dari session jika ada
	var userID string
	if uid, ok := c.Locals("user_id").(string); ok {
		userID = uid
	}

	// 📥 Ambil donasi 'completed' untuk masjid ini
	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ? AND donation_status = ?", masjid.MasjidID, "completed").
		Order("created_at desc").
		Find(&donations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi")
	}

	// 🔁 Format respons dengan like count dan liked status
	type DonationWithLike struct {
		model.Donation

		LikeCount     int  `json:"like_count"`
		IsLikedByUser bool `json:"is_liked_by_user"`
	}

	var response []DonationWithLike
	for _, d := range donations {
		var count int64
		ctrl.DB.
			Model(&model.DonationLikeModel{}).
			Where("donation_like_donation_id = ? AND donation_like_is_liked = true", d.DonationID).
			Count(&count)

		liked := false
		if userID != "" {
			var like model.DonationLikeModel
			err := ctrl.DB.
				Where("donation_like_donation_id = ? AND donation_like_user_id = ? AND donation_like_is_liked = true",
					d.DonationID, userID).
				First(&like).Error
			if err == nil {
				liked = true
			}
		}

		response = append(response, DonationWithLike{
			Donation:       d,
			LikeCount:      int(count),
			IsLikedByUser:  liked,
		})
	}

	return c.JSON(response)
}


// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleDonationStatusWebhook(db *gorm.DB, body map[string]interface{}) error {
	// ✅ Ambil order_id & transaction_status dari payload
	orderID, orderExists := body["order_id"].(string)
	transactionStatus, statusExists := body["transaction_status"].(string)

	if !orderExists || !statusExists {
		return fmt.Errorf("payload tidak valid: order_id atau transaction_status tidak ditemukan")
	}

	log.Printf("🔔 Webhook diterima: order_id=%s, status=%s\n", orderID, transactionStatus)

	// 🔍 Cari donasi berdasarkan order ID
	var donation model.Donation
	if err := db.Where("donation_order_id = ?", orderID).First(&donation).Error; err != nil {
		log.Printf("[ERROR] Donasi tidak ditemukan untuk order_id: %s\n", orderID)
		return fmt.Errorf("donasi tidak ditemukan untuk order_id %s: %v", orderID, err)
	}

	// 🔁 Update status berdasarkan transaction_status
	switch transactionStatus {
	case "settlement", "capture", "success":
		donation.DonationStatus = "completed"
	case "cancel", "deny", "expire", "failure", "failed":
		donation.DonationStatus = "failed"
	case "pending":
		donation.DonationStatus = "pending"
	default:
		log.Printf("[WARN] Status tidak dikenali: %s (diabaikan)\n", transactionStatus)
		return nil // Status tidak dikenali → tidak update apa pun
	}

	// 💾 Simpan perubahan status donasi
	if err := db.Save(&donation).Error; err != nil {
		log.Printf("[ERROR] Gagal update status donasi: %s\n", err)
		return fmt.Errorf("gagal memperbarui status donasi: %v", err)
	}

	log.Printf("✅ Donasi %s diperbarui ke status: %s\n", orderID, donation.DonationStatus)
	return nil
}

// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleMidtransNotification(c *fiber.Ctx) error {
	// 🔄 Ambil payload dari webhook
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] Gagal memparsing body webhook:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook body",
		})
	}

	// 📦 Log payload untuk debugging
	log.Println("📥 Received Midtrans webhook payload:", body)

	// 🔌 Ambil koneksi DB dari context
	dbRaw := c.Locals("db")
	db, ok := dbRaw.(*gorm.DB)
	if !ok || db == nil {
		log.Println("[ERROR] Koneksi database tidak tersedia di context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Koneksi database tidak tersedia",
		})
	}

	// ⚙️ Proses status donasi berdasarkan notifikasi Midtrans
	if err := ctrl.HandleDonationStatusWebhook(db, body); err != nil {
		log.Println("[ERROR] Webhook processing failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Gagal memproses webhook: %v", err),
		})
	}

	// ✅ Kirim respons sukses ke Midtrans
	log.Println("✅ Midtrans webhook processed successfully")
	return c.SendStatus(fiber.StatusOK)
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


// 🟢 GET DONATIONS BY USER SESSION: Ambil donasi milik user dari session
func (ctrl *DonationController) GetDonationsByUserID(c *fiber.Ctx) error {
	// 🔐 Ambil user_id dari session (Locals)
	userIDValue := c.Locals("user_id")
	if userIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User belum login",
		})
	}

	userID, ok := userIDValue.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID tidak valid",
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

	// 🔁 Format respons seperti GetDonationsByMasjidSlug
	type DonationWithLike struct {
		model.Donation

		LikeCount     int  `json:"like_count"`
		IsLikedByUser bool `json:"is_liked_by_user"`
	}

	var response []DonationWithLike
	for _, d := range donations {
		var count int64
		ctrl.DB.
			Model(&model.DonationLikeModel{}).
			Where("donation_like_donation_id = ? AND donation_like_is_liked = true", d.DonationID).
			Count(&count)

		liked := false
		var like model.DonationLikeModel
		err := ctrl.DB.
			Where("donation_like_donation_id = ? AND donation_like_user_id = ? AND donation_like_is_liked = true",
				d.DonationID, userID).
			First(&like).Error
		if err == nil {
			liked = true
		}

		response = append(response, DonationWithLike{
			Donation:       d,
			LikeCount:      int(count),
			IsLikedByUser:  liked,
		})
	}

	return c.JSON(response)
}

