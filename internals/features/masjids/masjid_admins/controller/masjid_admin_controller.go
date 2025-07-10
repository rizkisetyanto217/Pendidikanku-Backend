package controller

import (
	"errors"
	"masjidku_backend/internals/features/masjids/masjid_admins/dto"
	"masjidku_backend/internals/features/masjids/masjid_admins/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MasjidAdminController struct {
	DB *gorm.DB
}

func NewMasjidAdminController(db *gorm.DB) *MasjidAdminController {
	return &MasjidAdminController{DB: db}
}

// ✅ Tambah user sebagai admin masjid
func (ctrl *MasjidAdminController) AddAdmin(c *fiber.Ctx) error {
	var body dto.MasjidAdminRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid input",
			"error":   err.Error(),
		})
	}

	admin := model.MasjidAdminModel{
		MasjidID: body.MasjidAdminMasjidID,
		UserID:   body.MasjidAdminsUserID,
		IsActive: true,
	}

	// Cek apakah sudah ada admin yang sama
	result := ctrl.DB.
		Where("masjid_admins_masjid_id = ? AND masjid_admins_user_id = ?", body.MasjidAdminMasjidID, body.MasjidAdminsUserID).
		First(&model.MasjidAdminModel{})

	if result.Error == nil {
		// Sudah ada
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "User sudah jadi admin di masjid ini",
		})
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mencari data",
			"error":   result.Error.Error(),
		})
	}

	// Lanjut create admin baru
	if err := ctrl.DB.Create(&admin).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menambahkan admin",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Admin berhasil ditambahkan",
		"data":    dto.ToMasjidAdminResponse(admin),
	})

}

// ✅ Ambil semua admin aktif untuk sebuah masjid
func (ctrl *MasjidAdminController) GetAdminsByMasjid(c *fiber.Ctx) error {
	var body struct {
		MasjidAdminsMasjidID string `json:"masjid_admins_masjid_id"` // ✅ sama persis dengan model dan DTO
	}

	if err := c.BodyParser(&body); err != nil || body.MasjidAdminsMasjidID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Masjid ID wajib dikirim di body",
			"error":   err.Error(),
		})
	}

	var admins []model.MasjidAdminModel

	if err := ctrl.DB.Preload("User").
		Where("masjid_admin_masjid_id = ? AND masjid_admins_is_active = true", body.MasjidAdminsMasjidID).
		Find(&admins).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil daftar admin aktif",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Daftar admin aktif berhasil diambil",
		"data":    admins,
	})
}

// ✅ Nonaktifkan admin
func (ctrl *MasjidAdminController) RevokeAdmin(c *fiber.Ctx) error {
	var body struct {
		MasjidAdminsUserID   string `json:"masjid_admins_user_id"`
		MasjidAdminsMasjidID string `json:"masjid_admins_masjid_id"`
	}

	if err := c.BodyParser(&body); err != nil || body.MasjidAdminsUserID == "" || body.MasjidAdminsMasjidID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "masjid_admins_user_id dan masjid_admins_masjid_id wajib dikirim",
			"error":   err.Error(),
		})
	}

	// Update is_active menjadi false berdasarkan masjid_admins_user_id & masjid_admin_masjid_id
	result := ctrl.DB.Model(&model.MasjidAdminModel{}).
		Where("masjid_admins_user_id = ? AND masjid_admins_masjid_id = ? AND masjid_admins_is_active = true",
			body.MasjidAdminsUserID, body.MasjidAdminsMasjidID).
		Update("masjid_admins_is_active", false)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menonaktifkan admin",
			"error":   result.Error.Error(),
		})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Tidak ditemukan admin aktif untuk user ini di masjid ini",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Admin berhasil dinonaktifkan",
	})
}
