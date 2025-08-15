package controller

import (
	"masjidku_backend/internals/features/lembaga/classes/main/dto"
	"masjidku_backend/internals/features/lembaga/classes/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GET /admin/classes/slug/:slug
func (ctrl *ClassController) GetClassBySlug(c *fiber.Ctx) error {
	masjidID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err
	}
	slug := helper.GenerateSlug(c.Params("slug"))

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_slug = ? AND class_deleted_at IS NULL", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses kelas di masjid lain")
	}
	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}
