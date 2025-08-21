package controller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/dto"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	helper "masjidku_backend/internals/helpers"
)

type MasjidAdminController struct {
	DB *gorm.DB
}

func NewMasjidAdminController(db *gorm.DB) *MasjidAdminController {
	return &MasjidAdminController{DB: db}
}

/*
 * POST /api/a/masjid-admins
 * Body: { "masjid_admins_masjid_id": "...", "masjid_admins_user_id": "...", "masjid_admins_is_active": true? }
 */
func (ctrl *MasjidAdminController) AddAdmin(c *fiber.Ctx) error {
	var body dto.MasjidAdminRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input")
	}

	// Build model dari DTO (default is_active=true bila tidak dikirim)
	admin := body.ToModelCreate()

	// Cek existing (baris hidup saja, GORM otomatis exclude soft-delete krn pakai gorm.DeletedAt)
	var existing model.MasjidAdminModel
	if err := ctrl.DB.
		Where("masjid_admins_masjid_id = ? AND masjid_admins_user_id = ?",
			admin.MasjidAdminsMasjidID, admin.MasjidAdminsUserID).
		First(&existing).Error; err == nil {
		// Sudah ada
		return helper.JsonError(c, fiber.StatusConflict, "User sudah jadi admin di masjid ini")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa admin existing")
	}

	// Create
	if err := ctrl.DB.Create(admin).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "User sudah jadi admin di masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan admin")
	}

	return helper.JsonCreated(c, "Admin berhasil ditambahkan", dto.ToMasjidAdminResponse(admin))
}

/*
 * GET /api/a/masjid-admins/by-masjid
 * Body: { "masjid_admins_masjid_id": "..." }
 * (Kalau mau, bisa diubah ke path/query param)
 */
func (ctrl *MasjidAdminController) GetAdminsByMasjid(c *fiber.Ctx) error {
	var body struct {
		MasjidAdminsMasjidID string `json:"masjid_admins_masjid_id"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.MasjidAdminsMasjidID) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_admins_masjid_id wajib dikirim")
	}

	var admins []model.MasjidAdminModel
	if err := ctrl.DB.
		Preload("User").
		Where("masjid_admins_masjid_id = ? AND masjid_admins_is_active = TRUE",
			body.MasjidAdminsMasjidID).
		Find(&admins).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar admin aktif")
	}

	// Map ke response DTO
	out := make([]dto.MasjidAdminResponse, 0, len(admins))
	for i := range admins {
		out = append(out, dto.ToMasjidAdminResponse(&admins[i]))
	}

	return helper.JsonOK(c, "Daftar admin aktif berhasil diambil", out)
}

/*
 * PATCH /api/a/masjid-admins/revoke
 * Body: { "masjid_admins_user_id": "...", "masjid_admins_masjid_id": "..." }
 */
func (ctrl *MasjidAdminController) RevokeAdmin(c *fiber.Ctx) error {
	var body struct {
		MasjidAdminsUserID   string `json:"masjid_admins_user_id"`
		MasjidAdminsMasjidID string `json:"masjid_admins_masjid_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input")
	}
	if strings.TrimSpace(body.MasjidAdminsUserID) == "" || strings.TrimSpace(body.MasjidAdminsMasjidID) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_admins_user_id dan masjid_admins_masjid_id wajib dikirim")
	}

	// Nonaktifkan (baris hidup saja sudah difilter default oleh GORM soft delete)
	res := ctrl.DB.Model(&model.MasjidAdminModel{}).
		Where("masjid_admins_user_id = ? AND masjid_admins_masjid_id = ? AND masjid_admins_is_active = TRUE",
			body.MasjidAdminsUserID, body.MasjidAdminsMasjidID).
		Update("masjid_admins_is_active", false)

	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menonaktifkan admin")
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tidak ditemukan admin aktif untuk user ini di masjid ini")
	}

	return helper.JsonUpdated(c, "Admin berhasil dinonaktifkan", nil)
}
