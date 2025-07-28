package controller

import (
	"masjidku_backend/internals/features/donations/donations/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 🟢 GET DONATION BY ID: Ambil 1 data donasi berdasarkan ID
func (ctrl *DonationController) GetDonationByID(c *fiber.Ctx) error {
	// 🔸 Ambil ID dari parameter URL
	donationIDParam := c.Params("id")

	// 🔍 Validasi UUID
	donationID, err := uuid.Parse(donationIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID donasi tidak valid",
		})
	}

	// 🔍 Cari data donasi di database
	var donation model.Donation
	if err := ctrl.DB.
		Where("donation_id = ?", donationID).
		First(&donation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Donasi tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data donasi",
		})
	}

	// ✅ Kirim data donasi
	return c.JSON(donation)
}
