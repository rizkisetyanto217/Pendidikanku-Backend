package controller

import (
	"fmt"
	"log"
	"strings"
	"time"

	modelMasjid "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"
	"masjidku_backend/internals/features/payment/donations/dto"
	"masjidku_backend/internals/features/payment/donations/model"
	donationService "masjidku_backend/internals/features/payment/donations/service"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
	========================================================
	  Controller

========================================================
*/
type DonationController struct {
	DB *gorm.DB
}

func NewDonationController(db *gorm.DB) *DonationController {
	return &DonationController{DB: db}
}

/*
	========================================================
	  Simple Create (no slug) — for testing Snap

========================================================
*/
type SimpleDonationRequest struct {
	DonationName   string  `json:"donation_name" validate:"required"`
	DonationEmail  *string `json:"donation_email" validate:"omitempty,email"`
	DonationAmount int     `json:"donation_amount" validate:"required,gt=0"`
}

// POST /public/donations/simple
func (ctrl *DonationController) CreateDonationSimple(c *fiber.Ctx) error {
	var body SimpleDonationRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if strings.TrimSpace(body.DonationName) == "" || body.DonationAmount <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "donation_name & donation_amount wajib")
	}

	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())

	// HOTFIX: penuhi CHECK XOR & TYPE (pakai Non-SPP = 3 + target_id random)
	tType := 3
	tID := uuid.New()

	// 1) Insert minimal row
	donation := model.Donation{
		DonationName:           body.DonationName,
		DonationAmount:         body.DonationAmount,
		DonationStatus:         model.DonationStatusPending,
		DonationOrderID:        orderID,
		DonationPaymentGateway: "midtrans",
		DonationTargetType:     &tType,
		DonationTargetID:       &tID,
	}
	if err := ctrl.DB.Create(&donation).Error; err != nil {
		log.Println("[ERROR] Failed to create donation:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan donasi")
	}

	// 2) Snap token
	email := ""
	if body.DonationEmail != nil {
		email = *body.DonationEmail
	}
	token, redirectURL, err := donationService.GenerateSnapToken(donation, body.DonationName, email)
	if err != nil {
		log.Println("[ERROR] GenerateSnapToken failed:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat token pembayaran")
	}

	// 3) Update token
	donation.DonationPaymentToken = &token
	if err := ctrl.DB.Model(&donation).Update("donation_payment_token", &token).Error; err != nil {
		log.Println("[ERROR] Failed to update token:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui token pembayaran")
	}

	return helper.JsonCreated(c, "OK", struct {
		OrderID     string `json:"order_id"`
		SnapToken   string `json:"snap_token"`
		RedirectURL string `json:"redirect_url"`
	}{
		OrderID:     orderID,
		SnapToken:   token,
		RedirectURL: redirectURL,
	})
}

/*
	========================================================
	  Create (by masjid slug) — full

========================================================
*/
func (ctrl *DonationController) CreateDonation(c *fiber.Ctx) error {
	var body dto.CreateDonationRequest
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] BodyParser failed:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := body.Validate(nil); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid tidak boleh kosong")
	}

	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		log.Println("[ERROR] Masjid not found by slug:", slug, "err:", err)
		return fiber.NewError(fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// user (opsional)
	userUUID := helper.GetUserUUID(c)
	var userIDPtr *uuid.UUID
	if userUUID != uuid.Nil {
		userIDPtr = &userUUID
	}

	total := body.DonationAmount

	// auto-split masjidku 50:50 jika subnya kosong
	var amountToMasjid, amountToApp *int
	if body.DonationAmountMasjidku != nil &&
		body.DonationAmountMasjidkuToMasjid == nil &&
		body.DonationAmountMasjidkuToApp == nil {
		half := *body.DonationAmountMasjidku / 2
		rest := *body.DonationAmountMasjidku - half
		amountToMasjid = &half
		amountToApp = &rest
	}

	var targetTypePtr *int = body.DonationTargetType
	var targetIDPtr *uuid.UUID
	var userSPPBillingIDPtr *uuid.UUID

	if body.DonationTargetType != nil {
		switch *body.DonationTargetType {
		case dto.DonationTargetSPP: // 4
			if body.DonationUserSPPBillingID != nil && *body.DonationUserSPPBillingID != "" {
				id, err := uuid.Parse(*body.DonationUserSPPBillingID)
				if err != nil {
					return fiber.NewError(fiber.StatusBadRequest, "donation_user_spp_billing_id bukan UUID valid")
				}
				userSPPBillingIDPtr = &id
			}
		default: // 1/2/3/5
			if body.DonationTargetID != nil && *body.DonationTargetID != "" {
				id, err := uuid.Parse(*body.DonationTargetID)
				if err != nil {
					return fiber.NewError(fiber.StatusBadRequest, "donation_target_id bukan UUID valid")
				}
				targetIDPtr = &id
			}
		}
	}

	orderID := fmt.Sprintf("DONATION-%d", time.Now().UnixNano())

	donation := model.Donation{
		DonationUserID:   userIDPtr,
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

	if err := ctrl.DB.Create(&donation).Error; err != nil {
		log.Println("[ERROR] Failed to create donation:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan donasi")
	}

	// Snap token
	email := ""
	if body.DonationEmail != nil {
		email = *body.DonationEmail
	}
	token, redirectURL, err := donationService.GenerateSnapToken(donation, body.DonationName, email)
	if err != nil {
		log.Println("[ERROR] GenerateSnapToken failed:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat token pembayaran")
	}

	// simpan token
	donation.DonationPaymentToken = &token
	if err := ctrl.DB.Model(&donation).
		Update("donation_payment_token", &token).Error; err != nil {
		log.Println("[ERROR] Failed to update token:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui token pembayaran")
	}

	return helper.JsonCreated(c, "Donasi berhasil dibuat. Silakan lanjutkan pembayaran.",
		struct {
			OrderID     string `json:"order_id"`
			SnapToken   string `json:"snap_token"`
			RedirectURL string `json:"redirect_url"`
		}{
			OrderID:     donation.DonationOrderID,
			SnapToken:   token,
			RedirectURL: redirectURL,
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

/* ===================== Query (ringkas) ===================== */

func (ctrl *DonationController) GetDonationsByMasjidID(c *fiber.Ctx) error {
	masjidIDParam := c.Params("masjid_id")
	masjidID, err := uuid.Parse(masjidIDParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak valid")
	}

	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ? AND donation_status = ? AND deleted_at IS NULL", masjidID, model.DonationStatusCompleted).
		Order("created_at DESC").
		Find(&donations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi berdasarkan masjid")
	}

	return helper.JsonOK(c, "Data donasi berhasil diambil.", donations)
}

func (ctrl *DonationController) GetDonationsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid tidak boleh kosong")
	}

	var masjid modelMasjid.MasjidModel
	if err := ctrl.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Masjid dengan slug tersebut tidak ditemukan")
	}

	var donations []model.Donation
	if err := ctrl.DB.
		Where("donation_masjid_id = ? AND donation_status = ? AND deleted_at IS NULL", masjid.MasjidID, model.DonationStatusCompleted).
		Order("created_at DESC").
		Find(&donations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi")
	}

	return helper.JsonOK(c, "Data donasi berhasil diambil.", donations)
}

func (ctrl *DonationController) GetAllDonations(c *fiber.Ctx) error {
	var donations []model.Donation
	if err := ctrl.DB.Where("deleted_at IS NULL").Order("created_at DESC").Find(&donations).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi")
	}
	return helper.JsonOK(c, "Data donasi berhasil diambil.", donations)
}

/* ===================== Webhook ===================== */

// Map status Midtrans → status internal app.
func mapMidtransStatus(txStatus, fraudStatus string) string {
	switch txStatus {
	case "capture", "settlement", "success":
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

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

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

func (ctrl *DonationController) HandleDonationStatusWebhook(db *gorm.DB, body map[string]interface{}) error {
	orderID := getString(body, "order_id")
	transactionStatus := strings.ToLower(getString(body, "transaction_status"))
	if orderID == "" || transactionStatus == "" {
		return fmt.Errorf("payload tidak valid: order_id atau transaction_status tidak ditemukan")
	}
	log.Printf("🔔 Webhook diterima: order_id=%s, status=%s\n", orderID, transactionStatus)

	newStatus := mapMidtransStatus(transactionStatus, strings.ToLower(getString(body, "fraud_status")))
	if newStatus == "" {
		log.Printf("[WARN] Status Midtrans tidak dikenali: %s (diabaikan)\n", transactionStatus)
		return nil
	}

	paymentType := strings.TrimSpace(getString(body, "payment_type"))

	var paidAtPtr *time.Time
	if newStatus == model.DonationStatusPaid {
		t := parseMidtransTime(body)
		paidAtPtr = &t
	}

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
		if donation.DonationStatus != newStatus {
			updates["donation_status"] = newStatus
		}
		if paymentType != "" && (donation.DonationPaymentMethod == nil || *donation.DonationPaymentMethod != paymentType) {
			updates["donation_payment_method"] = paymentType
		}
		if paidAtPtr != nil && (donation.DonationPaidAt == nil || !donation.DonationPaidAt.Equal(*paidAtPtr)) {
			updates["donation_paid_at"] = *paidAtPtr
		}
		if len(updates) == 0 {
			log.Printf("ℹ️ Donasi %s tidak berubah (status=%s).\n", orderID, donation.DonationStatus)
			return nil
		}

		if err := tx.Model(&donation).Updates(updates).Error; err != nil {
			log.Printf("[ERROR] Gagal update donasi %s: %v\n", orderID, err)
			return fmt.Errorf("gagal memperbarui donasi %s: %w", orderID, err)
		}
		log.Printf("✅ Donasi %s diperbarui: %+v\n", orderID, updates)
		return nil
	})
}

func (ctrl *DonationController) HandleMidtransNotification(c *fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		log.Println("[ERROR] Gagal memparsing body webhook:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid webhook body")
	}

	log.Println("📥 Received Midtrans webhook payload:", body)

	orderID := getString(body, "order_id")
	txStatus := strings.ToLower(getString(body, "transaction_status"))
	appStatus := mapMidtransStatus(txStatus, strings.ToLower(getString(body, "fraud_status")))

	// Pakai ctrl.DB langsung
	if err := ctrl.HandleDonationStatusWebhook(ctrl.DB, body); err != nil {
		log.Println("[ERROR] Webhook processing failed:", err)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Gagal memproses webhook: %v", err))
	}

	return helper.JsonOK(c, "Midtrans webhook processed successfully",
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
