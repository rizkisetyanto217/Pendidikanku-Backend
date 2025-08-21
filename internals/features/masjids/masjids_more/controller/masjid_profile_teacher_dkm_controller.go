package controller

import (
	masjidmodel "masjidku_backend/internals/features/masjids/masjids/model"
	"masjidku_backend/internals/features/masjids/masjids_more/dto"
	"masjidku_backend/internals/features/masjids/masjids_more/model"
	helper "masjidku_backend/internals/helpers"

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
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}

	profile := body.ToModel()
	if err := ctrl.DB.WithContext(c.Context()).Create(profile).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan profil")
	}

	return helper.JsonCreated(c, "Profil berhasil ditambahkan", dto.ToResponse(profile))
}

// ✅ Ambil profil pengajar/DKM berdasarkan ID
func (ctrl *MasjidProfileTeacherDkmController) GetProfileByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	var profile model.MasjidProfileTeacherDkmModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_profile_teacher_dkm_id = ?", id).
		First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil profil")
	}

	return helper.JsonOK(c, "Berhasil mengambil profil", dto.ToResponse(&profile))
}

// ✅ Ambil semua profil pengajar/DKM berdasarkan masjid slug (PUBLIC)
func (ctrl *MasjidProfileTeacherDkmController) GetProfilesByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter slug wajib dikirim")
	}

	// 1) Cari masjid_id dari slug
	var masjid masjidmodel.MasjidModel
	if err := ctrl.DB.WithContext(c.Context()).
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		First(&masjid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid")
	}

	// 2) Ambil profil berdasarkan masjid_id
	var profiles []model.MasjidProfileTeacherDkmModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_profile_teacher_dkm_masjid_id = ?", masjid.MasjidID).
		Order("masjid_profile_teacher_dkm_created_at DESC").
		Find(&profiles).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data profil")
	}

	// 3) Transform ke response DTO
	responses := make([]dto.MasjidProfileTeacherDkmResponse, 0, len(profiles))
	for i := range profiles {
		responses = append(responses, dto.ToResponse(&profiles[i]))
	}

	return helper.JsonOK(c, "Berhasil mengambil profil", responses)
}

// ✅ Ambil semua profil pengajar/DKM berdasarkan masjid (dari token/admin)
func (ctrl *MasjidProfileTeacherDkmController) GetProfilesByMasjid(c *fiber.Ctx) error {
	masjidIDs, ok := c.Locals("masjid_admin_ids").([]string)
	if !ok || len(masjidIDs) == 0 || masjidIDs[0] == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}
	masjidID := masjidIDs[0]

	var profiles []model.MasjidProfileTeacherDkmModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_profile_teacher_dkm_masjid_id = ?", masjidID).
		Order("masjid_profile_teacher_dkm_created_at DESC").
		Find(&profiles).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data profil")
	}

	responses := make([]dto.MasjidProfileTeacherDkmResponse, 0, len(profiles))
	for i := range profiles {
		responses = append(responses, dto.ToResponse(&profiles[i]))
	}

	return helper.JsonOK(c, "Berhasil mengambil profil", responses)
}

// ✅ Update profil pengajar/DKM
func (ctrl *MasjidProfileTeacherDkmController) UpdateProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	var body dto.MasjidProfileTeacherDkmRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}

	var existing model.MasjidProfileTeacherDkmModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_profile_teacher_dkm_id = ?", id).
		First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil profil")
	}

	// body.ToModel() diasumsikan return *model.MasjidProfileTeacherDkmModel
	updated := body.ToModel()
	updated.MasjidProfileTeacherDkmID = existing.MasjidProfileTeacherDkmID

	if err := ctrl.DB.WithContext(c.Context()).
		Model(&existing).
		Updates(updated).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate profil")
	}

	// ⛳ Jangan pakai &updated (itu jadi **pointer)
	return helper.JsonUpdated(c, "Profil berhasil diupdate", dto.ToResponse(updated))
}

// ✅ Hapus profil pengajar/DKM
func (ctrl *MasjidProfileTeacherDkmController) DeleteProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_profile_teacher_dkm_id = ?", id).
		Delete(&model.MasjidProfileTeacherDkmModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus profil")
	}

	return helper.JsonDeleted(c, "Profil berhasil dihapus", fiber.Map{
		"masjid_profile_teacher_dkm_id": id,
	})
}
