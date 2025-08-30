package controller

import (
	secDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GET /admin/class-sections/slug/:slug
// Mengambil data section berdasarkan slug dan memastikan data milik masjid yang valid
func (ctrl *ClassSectionController) GetClassSectionBySlug(c *fiber.Ctx) error {
	// Ambil masjid ID dari token user
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// Ambil slug dari URL params dan normalisasi
	slug := helper.GenerateSlug(c.Params("slug"))

	// Ambil data section berdasarkan slug yang diberikan
	var m secModel.ClassSectionModel
	if err := ctrl.DB.First(&m, "class_sections_slug = ? AND class_sections_deleted_at IS NULL", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Pastikan section milik masjid yang valid (tenant)
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses section milik masjid lain")
	}

	// Kembalikan response yang sudah diformat
	return helper.JsonOK(c, "OK", secDTO.NewClassSectionResponse(&m, ""))
}
