// internals/features/lembaga/classes/user_classes/main/controller/user_my_class_controller.go
package controller

import (
	"masjidku_backend/internals/features/school/classes/classes/dto"
	"masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GET /admin/classes/slug/:slug
func (ctrl *ClassController) GetClassBySlug(c *fiber.Ctx) error {
	// Ambil masjid dari token (ganti ke GetUserIDFromToken jika itu yang tersedia di project)
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	slug := helper.GenerateSlug(c.Params("slug"))

	var m model.ClassModel
	if err := ctrl.DB.
		Where(`
			class_masjid_id = ?
			AND lower(class_slug) = lower(?)
			AND class_deleted_at IS NULL
			AND class_delete_pending_until IS NULL
		`, masjidID, slug).
		First(&m).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Tidak perlu cek masjid lagi karena sudah difilter di WHERE
	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}
