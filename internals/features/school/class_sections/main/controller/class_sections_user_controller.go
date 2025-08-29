package controller

import (
	secDTO "masjidku_backend/internals/features/school/class_sections/main/dto"
	secModel "masjidku_backend/internals/features/school/class_sections/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GET /admin/class-sections/slug/:slug
func (ctrl *ClassSectionController) GetClassSectionBySlug(c *fiber.Ctx) error {
	masjidID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err
	}
	slug := helper.GenerateSlug(c.Params("slug"))

	var m secModel.ClassSectionModel
	if err := ctrl.DB.First(&m, "class_sections_slug = ? AND class_sections_deleted_at IS NULL", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses section milik masjid lain")
	}
	return helper.JsonOK(c, "OK", secDTO.NewClassSectionResponse(&m))
}