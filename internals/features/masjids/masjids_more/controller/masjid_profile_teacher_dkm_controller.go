package controller

import (
	"masjidku_backend/internals/features/masjids/masjids_more/dto"
	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MasjidProfileTeacherDkmController struct {
	DB *gorm.DB
}

func NewMasjidProfileTeacherDkmController(db *gorm.DB) *MasjidProfileTeacherDkmController {
	return &MasjidProfileTeacherDkmController{DB: db}
}

// ✅ Tambah profil pengajar/DKM masjid
func (ctrl *MasjidProfileTeacherDkmController) CreateProfile(c *fiber.Ctx) error {
	var body dto.MasjidProfileTeacherDkmRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Input tidak valid",
			"error":   err.Error(),
		})
	}

	profile := body.ToModel()

	if err := ctrl.DB.Create(profile).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan profil",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Profil berhasil ditambahkan",
		"data":    dto.ToResponse(profile),
	})
}

// ✅ Ambil profil pengajar/DKM berdasarkan ID
func (ctrl *MasjidProfileTeacherDkmController) GetProfileByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Parameter ID wajib dikirim",
		})
	}

	var profile model.MasjidProfileTeacherDkmModel
	if err := ctrl.DB.
		Where("masjid_profile_teacher_dkm_id = ?", id).
		First(&profile).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Profil tidak ditemukan",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil profil",
		"data":    dto.ToResponse(&profile),
	})
}


// ✅ Ambil semua profil pengajar/DKM berdasarkan masjid (dari token)
func (ctrl *MasjidProfileTeacherDkmController) GetProfilesByMasjid(c *fiber.Ctx) error {
	// Ambil masjid_id dari token
	masjidIDs, ok := c.Locals("masjid_admin_ids").([]string)
	if !ok || len(masjidIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}
	masjidID := masjidIDs[0]

	var profiles []model.MasjidProfileTeacherDkmModel
	if err := ctrl.DB.
		Where("masjid_profile_teacher_dkm_masjid_id = ?", masjidID).
		Find(&profiles).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data profil",
			"error":   err.Error(),
		})
	}

	// Transform ke response
	var responses []dto.MasjidProfileTeacherDkmResponse
	for _, p := range profiles {
		responses = append(responses, dto.ToResponse(&p))
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil profil",
		"data":    responses,
	})
}


// ✅ Update profil pengajar/DKM
func (ctrl *MasjidProfileTeacherDkmController) UpdateProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Parameter ID wajib dikirim",
		})
	}

	var body dto.MasjidProfileTeacherDkmRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Input tidak valid",
			"error":   err.Error(),
		})
	}

	var existing model.MasjidProfileTeacherDkmModel
	if err := ctrl.DB.Where("masjid_profile_teacher_dkm_id = ?", id).First(&existing).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Profil tidak ditemukan",
			"error":   err.Error(),
		})
	}

	// Update dari DTO
	updated := body.ToModel()
	updated.MasjidProfileTeacherDkmID = existing.MasjidProfileTeacherDkmID

	if err := ctrl.DB.Model(&existing).Updates(updated).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengupdate profil",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Profil berhasil diupdate",
		"data":    updated,
	})
}

// ✅ Hapus profil pengajar/DKM
func (ctrl *MasjidProfileTeacherDkmController) DeleteProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Parameter ID wajib dikirim",
		})
	}

	if err := ctrl.DB.
		Where("masjid_profile_teacher_dkm_id = ?", id).
		Delete(&model.MasjidProfileTeacherDkmModel{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menghapus profil",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Profil berhasil dihapus",
	})
}
