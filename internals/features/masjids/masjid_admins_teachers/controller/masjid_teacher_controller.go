package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/dto"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MasjidTeacherController struct {
	DB *gorm.DB
}

func NewMasjidTeacherController(db *gorm.DB) *MasjidTeacherController {
	return &MasjidTeacherController{DB: db}
}

func (ctrl *MasjidTeacherController) Create(c *fiber.Ctx) error {
	var body dto.CreateMasjidTeacherRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := validator.New().Struct(body); err != nil {
		return helper.ValidationError(c, err)
	}

	// ✅ Ambil masjid_id dari context, diset oleh middleware IsMasjidAdmin
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan atau tidak valid")
	}

	// ✅ Simpan data berdasarkan masjid yang tervalidasi
	data := model.MasjidTeacher{
		MasjidTeachersMasjidID: masjidID,
		MasjidTeachersUserID:   body.MasjidTeachersUserID,
	}

	if err := ctrl.DB.Create(&data).Error; err != nil {
		log.Println("[ERROR] Failed to insert teacher:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menambahkan pengajar")
	}

	return helper.Success(c, "Pengajar berhasil ditambahkan", dto.ToMasjidTeacherResponse(dto.MasjidTeacher(data)))
}


func (ctrl *MasjidTeacherController) GetByMasjid(c *fiber.Ctx) error {
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}

	var teachers []model.MasjidTeacher
	if err := ctrl.DB.Where("masjid_teachers_masjid_id = ?", masjidID).Find(&teachers).Error; err != nil {
		log.Println("[ERROR] Gagal query teachers:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
	}

	return helper.Success(c, "Daftar pengajar ditemukan", fiber.Map{
		"total":    len(teachers),
		"teachers": teachers,
	})
}


func (ctrl *MasjidTeacherController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan")
	}

	// 🔐 Validasi apakah teacher ini benar-benar milik masjid tersebut
	var teacher model.MasjidTeacher
	if err := ctrl.DB.First(&teacher, "masjid_teachers_id = ? AND masjid_teachers_masjid_id = ?", id, masjidID).Error; err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Pengajar tidak ditemukan atau bukan milik masjid kamu")
	}

	if err := ctrl.DB.Delete(&teacher).Error; err != nil {
		log.Println("[ERROR] Failed to delete masjid teacher:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menghapus pengajar")
	}

	return helper.Success(c, "Pengajar berhasil dihapus", nil)
}
