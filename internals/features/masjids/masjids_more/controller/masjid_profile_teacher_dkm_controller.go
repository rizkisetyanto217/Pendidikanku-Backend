package controller

import (
	"strconv"
	"strings"

	masjidmodel "masjidku_backend/internals/features/masjids/masjids/model"
	"masjidku_backend/internals/features/masjids/masjids_more/dto"
	"masjidku_backend/internals/features/masjids/masjids_more/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidProfileTeacherDkmController struct {
	DB *gorm.DB
}

func NewMasjidProfileTeacherDkmController(db *gorm.DB) *MasjidProfileTeacherDkmController {
	return &MasjidProfileTeacherDkmController{DB: db}
}

var validate = validator.New()

// ===============================
// CREATE
// ===============================
// POST /api/a/masjid-profile-teacher-dkm
func (ctrl *MasjidProfileTeacherDkmController) CreateProfile(c *fiber.Ctx) error {
	var body dto.MasjidProfileTeacherDkmRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}
	if err := validate.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// Opsi: enforce masjid_id dari token (lebih aman)
	// Gunakan ini jika endpoint khusus admin/owner
	if ids, ok := c.Locals("masjid_admin_ids").([]string); ok && len(ids) > 0 && ids[0] != "" {
		if uid, err := uuid.Parse(ids[0]); err == nil {
			body.MasjidProfileTeacherDkmMasjidID = uid
		}
	}

	profile := body.ToModel()

	if err := ctrl.DB.WithContext(c.Context()).
		Create(profile).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan profil")
	}

	return helper.JsonCreated(c, "Profil berhasil ditambahkan", dto.ToResponse(profile))
}

// ===============================
// READ BY ID
// ===============================
// GET /public/masjid-profile-teacher-dkm/:id
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

// ===============================
// LIST BY MASJID SLUG (PUBLIC) + pagination/search/sort
// ===============================
// GET /public/masjid-profile-teacher-dkm/by-slug/:slug?page=&limit=&q=&sort=
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

	// Query param
	page, limit := getPagination(c, 1, 10) // default page=1, limit=10
	q := strings.TrimSpace(c.Query("q"))
	sort := strings.TrimSpace(c.Query("sort"))
	if sort == "" {
		sort = "masjid_profile_teacher_dkm_created_at DESC"
	}

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.MasjidProfileTeacherDkmModel{}).
		Where("masjid_profile_teacher_dkm_masjid_id = ?", masjid.MasjidID)

	// Search by name/role (optional)
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where(
			ctrl.DB.Where("masjid_profile_teacher_dkm_name ILIKE ?", like).
				Or("masjid_profile_teacher_dkm_role ILIKE ?", like),
		)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var profiles []model.MasjidProfileTeacherDkmModel
	if err := tx.
		Order(sort).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&profiles).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data profil")
	}

	responses := make([]dto.MasjidProfileTeacherDkmResponse, 0, len(profiles))
	for i := range profiles {
		responses = append(responses, dto.ToResponse(&profiles[i]))
	}

	return helper.JsonOK(c, "Berhasil mengambil profil", fiber.Map{
		"page":    page,
		"limit":   limit,
		"total":   total,
		"results": responses,
	})
}

// ===============================
// LIST BY MASJID (FROM TOKEN) + pagination/search/sort
// ===============================
// GET /api/a/masjid-profile-teacher-dkm?page=&limit=&q=&sort=
func (ctrl *MasjidProfileTeacherDkmController) GetProfilesByMasjid(c *fiber.Ctx) error {
	masjidIDs, ok := c.Locals("masjid_admin_ids").([]string)
	if !ok || len(masjidIDs) == 0 || masjidIDs[0] == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}
	masjidID := masjidIDs[0]

	page, limit := getPagination(c, 1, 10)
	q := strings.TrimSpace(c.Query("q"))
	sort := strings.TrimSpace(c.Query("sort"))
	if sort == "" {
		sort = "masjid_profile_teacher_dkm_created_at DESC"
	}

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.MasjidProfileTeacherDkmModel{}).
		Where("masjid_profile_teacher_dkm_masjid_id = ?", masjidID)

	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where(
			ctrl.DB.Where("masjid_profile_teacher_dkm_name ILIKE ?", like).
				Or("masjid_profile_teacher_dkm_role ILIKE ?", like),
		)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var profiles []model.MasjidProfileTeacherDkmModel
	if err := tx.
		Order(sort).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&profiles).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data profil")
	}

	responses := make([]dto.MasjidProfileTeacherDkmResponse, 0, len(profiles))
	for i := range profiles {
		responses = append(responses, dto.ToResponse(&profiles[i]))
	}

	return helper.JsonOK(c, "Berhasil mengambil profil", fiber.Map{
		"page":    page,
		"limit":   limit,
		"total":   total,
		"results": responses,
	})
}

// ===============================
// UPDATE (partial update aman)
// ===============================
// PUT /api/a/masjid-profile-teacher-dkm/:id
// PATCH juga bisa diarahkan ke handler ini
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

	// Konstruksi map untuk partial update (hanya field yang dikirim & valid)
	updates := map[string]interface{}{}

	// Boleh update masjid_id jika mau (opsional) — umumnya dikunci
	if body.MasjidProfileTeacherDkmMasjidID != uuid.Nil {
		updates["masjid_profile_teacher_dkm_masjid_id"] = body.MasjidProfileTeacherDkmMasjidID
	}
	if body.MasjidProfileTeacherDkmUserID != nil {
		updates["masjid_profile_teacher_dkm_user_id"] = body.MasjidProfileTeacherDkmUserID
	}
	if strings.TrimSpace(body.MasjidProfileTeacherDkmName) != "" {
		updates["masjid_profile_teacher_dkm_name"] = strings.TrimSpace(body.MasjidProfileTeacherDkmName)
	}
	if strings.TrimSpace(body.MasjidProfileTeacherDkmRole) != "" {
		updates["masjid_profile_teacher_dkm_role"] = strings.TrimSpace(body.MasjidProfileTeacherDkmRole)
	}
	if body.MasjidProfileTeacherDkmDescription != nil {
		updates["masjid_profile_teacher_dkm_description"] = body.MasjidProfileTeacherDkmDescription
	}
	if body.MasjidProfileTeacherDkmMessage != nil {
		updates["masjid_profile_teacher_dkm_message"] = body.MasjidProfileTeacherDkmMessage
	}
	if body.MasjidProfileTeacherDkmImageURL != nil {
		updates["masjid_profile_teacher_dkm_image_url"] = body.MasjidProfileTeacherDkmImageURL
	}

	if len(updates) == 0 {
		// tidak ada perubahan; kembalikan data existing
		return helper.JsonOK(c, "Tidak ada perubahan", dto.ToResponse(&existing))
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Model(&existing).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate profil")
	}

	// reload (opsional) agar UpdatedAt terisi terbaru
	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_profile_teacher_dkm_id = ?", id).
		First(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat ulang profil")
	}

	return helper.JsonUpdated(c, "Profil berhasil diupdate", dto.ToResponse(&existing))
}

// ===============================
// DELETE (soft delete by gorm.DeletedAt)
// ===============================
// DELETE /api/a/masjid-profile-teacher-dkm/:id
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


// getPagination membaca query ?page=&limit= dengan default & batas aman
func getPagination(c *fiber.Ctx, defaultPage, defaultLimit int) (int, int) {
	page := defaultPage
	limit := defaultLimit

	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
		}
	}

	// hard cap biar gak jebol (silakan sesuaikan)
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
