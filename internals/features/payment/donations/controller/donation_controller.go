// 📁 controller/donation_controller.go
package controller

import (
	"fmt"
	"log"
	modelMasjid "masjidku_backend/internals/features/masjids/masjids/model"
	"masjidku_backend/internals/features/payment/donations/dto"
	"masjidku_backend/internals/features/payment/donations/model"
	donationService "masjidku_backend/internals/features/payment/donations/service"
	helper "masjidku_backend/internals/helpers"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DonationController struct {
	DB *gorm.DB
}

func NewDonationController(db *gorm.DB) *DonationController {
	return &DonationController{DB: db}
}


// 🟢 CREATE DONATION (by slug): Buat donasi baru berdasarkan slug masjid
// 🟢 CREATE DONATION (by slug): Buat donasi baru berdasarkan slug masjid
func (ctrl *DonationController) CreateDonation(c *fiber.Ctx) error {
	var body dto.CreateDonationRequest
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] BodyParser failed:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// ✅ Validasi DTO (termasuk XOR & sum breakdown)
	if err := body.Validate(nil); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil slug dari URL
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid tidak boleh kosong")
	}

	// 🎯 Cari masjid berdasarkan slug
	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		log.Println("[ERROR] Masjid not found by slug:", slug, "err:", err)
		return fiber.NewError(fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// 🔐 Ambil user ID (opsional/anon jika kosong)
	userUUID := helper.GetUserUUID(c) // pastikan return uuid.UUID (zero jika tidak login)
	var userIDPtr *uuid.UUID
	if userUUID != uuid.Nil {
		userIDPtr = &userUUID
	}

	// 🔢 Total dari body (sudah >0 oleh validator)
	total := body.DonationAmount

	// 🔀 If user hanya isi donation_amount_masjidku (global) tanpa sub-breakdown,
	//     split 50:50 → to_masjid & to_app (kecuali user sudah isi subnya)
	var amountToMasjid, amountToApp *int
	if body.DonationAmountMasjidku != nil {
		if body.DonationAmountMasjidkuToMasjid == nil && body.DonationAmountMasjidkuToApp == nil {
			half := *body.DonationAmountMasjidku / 2
			rest := *body.DonationAmountMasjidku - half
			amountToMasjid = &half
			amountToApp = &rest
		}
	}

	// 🔗 SPP vs Target umum (sudah divalidasi XOR di DTO)
	var targetTypePtr *int = body.DonationTargetType
	var targetIDPtr *uuid.UUID
	var userSPPBillingIDPtr *uuid.UUID

	// Parsing ID berbasis tipe target
	if body.DonationTargetType != nil {
		switch *body.DonationTargetType {
		case dto.DonationTargetSPP: // 4
			// Wajib ada donation_user_spp_billing_id
			if body.DonationUserSPPBillingID != nil && *body.DonationUserSPPBillingID != "" {
				id, err := uuid.Parse(*body.DonationUserSPPBillingID)
				if err != nil {
					return fiber.NewError(fiber.StatusBadRequest, "donation_user_spp_billing_id bukan UUID valid")
				}
				userSPPBillingIDPtr = &id
			}
		default:
			// 1/2/3/5 → wajib ada donation_target_id
			if body.DonationTargetID != nil && *body.DonationTargetID != "" {
				id, err := uuid.Parse(*body.DonationTargetID)
				if err != nil {
					return fiber.NewError(fiber.StatusBadRequest, "donation_target_id bukan UUID valid")
				}
				targetIDPtr = &id
			}
		}
	}

	// 🧾 Buat OrderID (maks 100 char di DB)
	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())

	// 💾 Persiapkan objek donasi (model sudah pake pointer utk nullable)
	donation := model.Donation{
		DonationUserID:  userIDPtr,
		DonationMasjidID: &masjid.MasjidID,

		DonationName:   body.DonationName,
		DonationAmount: total,

		DonationAmountMasjid:           body.DonationAmountMasjid,
		DonationAmountMasjidku:         body.DonationAmountMasjidku,
		DonationAmountMasjidkuToMasjid: coalesceIntPtr(body.DonationAmountMasjidkuToMasjid, amountToMasjid),
		DonationAmountMasjidkuToApp:    coalesceIntPtr(body.DonationAmountMasjidkuToApp, amountToApp),

		DonationMessage: body.DonationMessage,

		DonationStatus:         model.DonationStatusPending,
		DonationOrderID:        orderID,
		DonationPaymentGateway: "midtrans",

		DonationTargetType:       targetTypePtr,
		DonationTargetID:         targetIDPtr,
		DonationUserSPPBillingID: userSPPBillingIDPtr,
	}

	// Simpan ke DB
	if err := ctrl.DB.Create(&donation).Error; err != nil {
		log.Println("[ERROR] Failed to create donation:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan donasi")
	}

	// 🔑 Snap token
	var email string
	if body.DonationEmail != nil {
		email = *body.DonationEmail
	}
	token, err := donationService.GenerateSnapToken(donation, body.DonationName, email)
	if err != nil {
		log.Println("[ERROR] GenerateSnapToken failed:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat token pembayaran")
	}

	// Simpan token (model pakai *string)
	donation.DonationPaymentToken = &token
	if err := ctrl.DB.Model(&donation).
		Updates(map[string]interface{}{"donation_payment_token": &token}).Error; err != nil {
		log.Println("[ERROR] Failed to update token:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui token pembayaran")
	}

	return helper.JsonCreated(
		c,
		"Donasi berhasil dibuat. Silakan lanjutkan pembayaran.",
		struct {
			OrderID   string `json:"order_id"`
			SnapToken string `json:"snap_token"`
		}{
			OrderID:   donation.DonationOrderID,
			SnapToken: token,
		},
	)
}

/* ===================== Utils ===================== */

func coalesceIntPtr(a, b *int) *int {
	if a != nil {
		return a
	}
	return b
}


// 🟢 GET DONATIONS BY MASJID ID: Ambil semua donasi yang telah *completed* untuk masjid tertentu
func (ctrl *DonationController) GetDonationsByMasjidID(c *fiber.Ctx) error {
	masjidIDParam := c.Params("masjid_id")
	masjidID, err := uuid.Parse(masjidIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Masjid ID tidak valid"})
	}

	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ? AND donation_status = ? AND deleted_at IS NULL", masjidID, model.DonationStatusCompleted).
		Order("created_at DESC").
		Find(&donations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data donasi berdasarkan masjid"})
	}

	// ⬇️ gunakan helper
	return helper.JsonOK(
		c,
		"Data donasi berhasil diambil.",
		donations,
	)
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

	// 🧑 Ambil user_id (opsional)
	var userID string
	if uid, ok := c.Locals("user_id").(string); ok {
		userID = uid
	}

	// 📥 Ambil donasi 'completed' untuk masjid ini (exclude soft-deleted)
	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ? AND donation_status = ? AND deleted_at IS NULL", masjid.MasjidID, model.DonationStatusCompleted).
		Order("created_at DESC").
		Find(&donations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi")
	}

	// 🔁 Format respons dengan like count dan liked status
	type DonationWithLike struct {
		model.Donation
		LikeCount     int  `json:"like_count"`
		IsLikedByUser bool `json:"is_liked_by_user"`
	}

	response := make([]DonationWithLike, 0, len(donations))
	for _, d := range donations {
		// Count likes
		var count int64
		if err := ctrl.DB.
			Model(&model.DonationLikeModel{}).
			Where("donation_like_donation_id = ? AND donation_like_is_liked = TRUE", d.DonationID).
			Count(&count).Error; err != nil {
			count = 0
		}

		liked := false
		if userID != "" {
			var like model.DonationLikeModel
			if err := ctrl.DB.
				Where("donation_like_donation_id = ? AND donation_like_user_id = ? AND donation_like_is_liked = TRUE",
					d.DonationID, userID).
				First(&like).Error; err == nil {
				liked = true
			}
		}

		response = append(response, DonationWithLike{
			Donation:      d,
			LikeCount:     int(count),
			IsLikedByUser: liked,
		})
	}

	// ✅ pakai helper JSON konsisten
	return helper.JsonOK(
		c,
		"Data donasi berhasil diambil.",
		response,
	)
}



// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
// imports yang dibutuhkan:
// import (
//   "fmt"
//   "log"
//   "strings"
//   "time"
//
//   "gorm.io/gorm"
//   "gorm.io/gorm/clause"
//
//   "masjidku_backend/internals/features/donations/model" // sesuaikan path
// )

// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleDonationStatusWebhook(db *gorm.DB, body map[string]interface{}) error {
	// ✅ Ambil field utama
	orderID := getString(body, "order_id")
	transactionStatus := strings.ToLower(getString(body, "transaction_status"))
	if orderID == "" || transactionStatus == "" {
		return fmt.Errorf("payload tidak valid: order_id atau transaction_status tidak ditemukan")
	}
	log.Printf("🔔 Webhook diterima: order_id=%s, status=%s\n", orderID, transactionStatus)

	// 🔁 Mapping Midtrans → status internal
	newStatus := mapMidtransStatus(transactionStatus, strings.ToLower(getString(body, "fraud_status")))
	if newStatus == "" {
		log.Printf("[WARN] Status Midtrans tidak dikenali: %s (diabaikan)\n", transactionStatus)
		return nil
	}

	// 💳 payment_type (bila ada)
	paymentType := strings.TrimSpace(getString(body, "payment_type"))

	// 🕒 paid_at jika status paid
	var paidAtPtr *time.Time
	if newStatus == model.DonationStatusPaid {
		t := parseMidtransTime(body) // settlement_time / transaction_time / now
		paidAtPtr = &t
	}

	// 🔒 Transaksi + row lock agar idempotent terhadap update paralel
	return db.Transaction(func(tx *gorm.DB) error {
		var donation model.Donation
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("donation_order_id = ?", orderID).
			First(&donation).Error; err != nil {
			log.Printf("[ERROR] Donasi tidak ditemukan untuk order_id: %s\n", orderID)
			return fmt.Errorf("donasi tidak ditemukan untuk order_id %s: %w", orderID, err)
		}

		updates := map[string]interface{}{}

		// Status berubah?
		if donation.DonationStatus != newStatus {
			updates["donation_status"] = newStatus
		}

		// payment_method berubah?
		if paymentType != "" && (donation.DonationPaymentMethod == nil || *donation.DonationPaymentMethod != paymentType) {
			updates["donation_payment_method"] = paymentType
		}

		// paid_at berubah?
		if paidAtPtr != nil && (donation.DonationPaidAt == nil || !donation.DonationPaidAt.Equal(*paidAtPtr)) {
			updates["donation_paid_at"] = *paidAtPtr
		}

		// Tidak ada yang berubah → idempotent
		if len(updates) == 0 {
			log.Printf("ℹ️ Donasi %s tidak berubah (status=%s).\n", orderID, donation.DonationStatus)
			return nil
		}

		// Simpan perubahan
		if err := tx.Model(&donation).Updates(updates).Error; err != nil {
			log.Printf("[ERROR] Gagal update donasi %s: %v\n", orderID, err)
			return fmt.Errorf("gagal memperbarui donasi %s: %w", orderID, err)
		}

		log.Printf("✅ Donasi %s diperbarui: %+v\n", orderID, updates)
		return nil
	})
}

/* ===================== Helpers ===================== */

// Map status Midtrans → status internal app.
// - capture/settlement/success → paid (kecuali credit_card challenge → pending)
// - pending → pending
// - expire/expired → expired
// - cancel/canceled/deny/failure/failed/refund/partial_refund → canceled
func mapMidtransStatus(txStatus, fraudStatus string) string {
	switch txStatus {
	case "capture", "settlement", "success":
		// credit_card + challenge → pending (menunggu manual review)
		if txStatus == "capture" && fraudStatus == "challenge" {
			return model.DonationStatusPending
		}
		return model.DonationStatusPaid
	case "pending":
		return model.DonationStatusPending
	case "expire", "expired":
		return model.DonationStatusExpired
	case "cancel", "canceled", "deny", "failure", "failed", "refund", "partial_refund":
		return model.DonationStatusCanceled
	default:
		return ""
	}
}

// Ambil string dari payload JSON
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Midtrans time parser: "2006-01-02 15:04:05" (settlement_time → transaction_time → now)
func parseMidtransTime(body map[string]interface{}) time.Time {
	const layout = "2006-01-02 15:04:05"
	if s := getString(body, "settlement_time"); s != "" {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t
		}
	}
	if s := getString(body, "transaction_time"); s != "" {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t
		}
	}
	return time.Now()
}



// 🟢 HANDLE MIDTRANS WEBHOOK: Update status donasi berdasarkan notifikasi Midtrans
func (ctrl *DonationController) HandleMidtransNotification(c *fiber.Ctx) error {
	// 🔄 Ambil payload dari webhook
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] Gagal memparsing body webhook:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid webhook body")
	}

	log.Println("📥 Received Midtrans webhook payload:", body)

	// 🔌 Ambil koneksi DB dari context
	dbRaw := c.Locals("db")
	db, ok := dbRaw.(*gorm.DB)
	if !ok || db == nil {
		log.Println("[ERROR] Koneksi database tidak tersedia di context")
		return fiber.NewError(fiber.StatusInternalServerError, "Koneksi database tidak tersedia")
	}

	// (Opsional) ekstrak info untuk dikembalikan di respons
	orderID := getString(body, "order_id")
	txStatus := strings.ToLower(getString(body, "transaction_status"))
	appStatus := mapMidtransStatus(txStatus, strings.ToLower(getString(body, "fraud_status")))

	// ⚙️ Proses status donasi berdasarkan notifikasi Midtrans
	if err := ctrl.HandleDonationStatusWebhook(db, body); err != nil {
		log.Println("[ERROR] Webhook processing failed:", err)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Gagal memproses webhook: %v", err))
	}

	// ✅ Respons sukses (pakai helper JSON konsisten)
	return helper.JsonOK(
		c,
		"Midtrans webhook processed successfully",
		struct {
			OrderID        string `json:"order_id"`
			MidtransStatus string `json:"midtrans_status"`
			AppStatus      string `json:"app_status"`
		}{
			OrderID:        orderID,
			MidtransStatus: txStatus,
			AppStatus:      appStatus,
		},
	)
}


// 🟢 GET ALL DONATIONS: Ambil seluruh data donasi (admin)
func (ctrl *DonationController) GetAllDonations(c *fiber.Ctx) error {
	var donations []model.Donation
	if err := ctrl.DB.
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Find(&donations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi")
	}

	return helper.JsonOK(
		c,
		"Data donasi berhasil diambil.",
		donations,
	)
}



// 🟢 GET DONATIONS BY USER & MASJID SLUG: Ambil donasi milik user berdasarkan masjid
func (ctrl *DonationController) GetDonationsByUserIDWithSlug(c *fiber.Ctx) error {
	// 🔐 Ambil user_id dari session
	userIDValue := c.Locals("user_id")
	if userIDValue == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}
	userIDStr, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userIDStr) == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID tidak valid")
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID bukan UUID yang valid")
	}

	// 📥 Slug
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid tidak boleh kosong")
	}

	// 🔍 Masjid by slug
	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Masjid dengan slug tersebut tidak ditemukan")
	}

	// 🔍 Donasi milik user untuk masjid ini (exclude soft-deleted)
	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_user_id = ? AND donation_masjid_id = ? AND deleted_at IS NULL", userUUID, masjid.MasjidID).
		Order("created_at DESC").
		Find(&donations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi user")
	}

	// 🔁 Tambahkan informasi like
	type DonationWithLike struct {
		model.Donation
		LikeCount     int  `json:"like_count"`
		IsLikedByUser bool `json:"is_liked_by_user"`
	}

	response := make([]DonationWithLike, 0, len(donations))
	for _, d := range donations {
		var count int64
		if err := ctrl.DB.
			Model(&model.DonationLikeModel{}).
			Where("donation_like_donation_id = ? AND donation_like_is_liked = TRUE", d.DonationID).
			Count(&count).Error; err != nil {
			count = 0
		}

		liked := false
		var like model.DonationLikeModel
		if err := ctrl.DB.
			Where("donation_like_donation_id = ? AND donation_like_user_id = ? AND donation_like_is_liked = TRUE",
				d.DonationID, userUUID).
			First(&like).Error; err == nil {
			liked = true
		}

		response = append(response, DonationWithLike{
			Donation:      d,
			LikeCount:     int(count),
			IsLikedByUser: liked,
		})
	}

	return helper.JsonOK(
		c,
		"Data donasi user berhasil diambil.",
		response,
	)
}
