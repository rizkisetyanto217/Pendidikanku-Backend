package controller

import (
	"errors"
	"masjidku_backend/internals/features/payment/donations/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)


// 🟢 GET DONATION BY ID: Ambil 1 data donasi berdasarkan ID
func (ctrl *DonationController) GetDonationByID(c *fiber.Ctx) error {
	// 🔸 Ambil & validasi ID dari URL
	donationIDParam := c.Params("id")
	donationID, err := uuid.Parse(donationIDParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID donasi tidak valid")
	}

	// 🔍 Cari data donasi (exclude soft-deleted)
	var donation model.Donation
	if err := ctrl.DB.
		Where("donation_id = ? AND deleted_at IS NULL", donationID).
		First(&donation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Donasi tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data donasi")
	}

	// ✅ Response OK (format konsisten)
	return helper.JsonOK(
		c,
		"Data donasi berhasil diambil.",
		donation,
	)
}
