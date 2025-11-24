package controller

import (
	"strconv"
	"strings"

	schoolmodel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"
	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools_more/dto"
	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools_more/model"
	helper "madinahsalam_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolProfileTeacherDkmController struct {
	DB *gorm.DB
}

func NewSchoolProfileTeacherDkmController(db *gorm.DB) *SchoolProfileTeacherDkmController {
	return &SchoolProfileTeacherDkmController{DB: db}
}

var validate = validator.New()

// ===============================
// CREATE
// ===============================
// POST /api/a/school-profile-teacher-dkm
func (ctrl *SchoolProfileTeacherDkmController) CreateProfile(c *fiber.Ctx) error {
	var body dto.SchoolProfileTeacherDkmRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}
	if err := validate.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// Opsi: enforce school_id dari token (lebih aman)
	// Gunakan ini jika endpoint khusus admin/owner
	if ids, ok := c.Locals("school_admin_ids").([]string); ok && len(ids) > 0 && ids[0] != "" {
		if uid, err := uuid.Parse(ids[0]); err == nil {
			body.SchoolProfileTeacherDkmSchoolID = uid
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
// GET /public/school-profile-teacher-dkm/:id
func (ctrl *SchoolProfileTeacherDkmController) GetProfileByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	var profile model.SchoolProfileTeacherDkmModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("school_profile_teacher_dkm_id = ?", id).
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
// GET /public/school-profile-teacher-dkm/by-slug/:slug?page=&limit=&q=&sort=
func (ctrl *SchoolProfileTeacherDkmController) GetProfilesBySchoolSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter slug wajib dikirim")
	}

	// 1) Cari school_id dari slug
	var school schoolmodel.SchoolModel
	if err := ctrl.DB.WithContext(c.Context()).
		Select("school_id").
		Where("school_slug = ?", slug).
		First(&school).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
	}

	// Query param
	page, limit := getPagination(c, 1, 10) // default page=1, limit=10
	q := strings.TrimSpace(c.Query("q"))
	sort := strings.TrimSpace(c.Query("sort"))
	if sort == "" {
		sort = "school_profile_teacher_dkm_created_at DESC"
	}

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.SchoolProfileTeacherDkmModel{}).
		Where("school_profile_teacher_dkm_school_id = ?", school.SchoolID)

	// Search by name/role (optional)
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where(
			ctrl.DB.Where("school_profile_teacher_dkm_name ILIKE ?", like).
				Or("school_profile_teacher_dkm_role ILIKE ?", like),
		)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var profiles []model.SchoolProfileTeacherDkmModel
	if err := tx.
		Order(sort).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&profiles).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data profil")
	}

	responses := make([]dto.SchoolProfileTeacherDkmResponse, 0, len(profiles))
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
// GET /api/a/school-profile-teacher-dkm?page=&limit=&q=&sort=
func (ctrl *SchoolProfileTeacherDkmController) GetProfilesBySchool(c *fiber.Ctx) error {
	schoolIDs, ok := c.Locals("school_admin_ids").([]string)
	if !ok || len(schoolIDs) == 0 || schoolIDs[0] == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "School ID tidak ditemukan di token")
	}
	schoolID := schoolIDs[0]

	page, limit := getPagination(c, 1, 10)
	q := strings.TrimSpace(c.Query("q"))
	sort := strings.TrimSpace(c.Query("sort"))
	if sort == "" {
		sort = "school_profile_teacher_dkm_created_at DESC"
	}

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.SchoolProfileTeacherDkmModel{}).
		Where("school_profile_teacher_dkm_school_id = ?", schoolID)

	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where(
			ctrl.DB.Where("school_profile_teacher_dkm_name ILIKE ?", like).
				Or("school_profile_teacher_dkm_role ILIKE ?", like),
		)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var profiles []model.SchoolProfileTeacherDkmModel
	if err := tx.
		Order(sort).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&profiles).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data profil")
	}

	responses := make([]dto.SchoolProfileTeacherDkmResponse, 0, len(profiles))
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
// PUT /api/a/school-profile-teacher-dkm/:id
// PATCH juga bisa diarahkan ke handler ini
func (ctrl *SchoolProfileTeacherDkmController) UpdateProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	var body dto.SchoolProfileTeacherDkmRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}

	var existing model.SchoolProfileTeacherDkmModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("school_profile_teacher_dkm_id = ?", id).
		First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil profil")
	}

	// Konstruksi map untuk partial update (hanya field yang dikirim & valid)
	updates := map[string]interface{}{}

	// Boleh update school_id jika mau (opsional) — umumnya dikunci
	if body.SchoolProfileTeacherDkmSchoolID != uuid.Nil {
		updates["school_profile_teacher_dkm_school_id"] = body.SchoolProfileTeacherDkmSchoolID
	}
	if body.SchoolProfileTeacherDkmUserID != nil {
		updates["school_profile_teacher_dkm_user_id"] = body.SchoolProfileTeacherDkmUserID
	}
	if strings.TrimSpace(body.SchoolProfileTeacherDkmName) != "" {
		updates["school_profile_teacher_dkm_name"] = strings.TrimSpace(body.SchoolProfileTeacherDkmName)
	}
	if strings.TrimSpace(body.SchoolProfileTeacherDkmRole) != "" {
		updates["school_profile_teacher_dkm_role"] = strings.TrimSpace(body.SchoolProfileTeacherDkmRole)
	}
	if body.SchoolProfileTeacherDkmDescription != nil {
		updates["school_profile_teacher_dkm_description"] = body.SchoolProfileTeacherDkmDescription
	}
	if body.SchoolProfileTeacherDkmMessage != nil {
		updates["school_profile_teacher_dkm_message"] = body.SchoolProfileTeacherDkmMessage
	}
	if body.SchoolProfileTeacherDkmImageURL != nil {
		updates["school_profile_teacher_dkm_image_url"] = body.SchoolProfileTeacherDkmImageURL
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
		Where("school_profile_teacher_dkm_id = ?", id).
		First(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat ulang profil")
	}

	return helper.JsonUpdated(c, "Profil berhasil diupdate", dto.ToResponse(&existing))
}

// ===============================
// DELETE (soft delete by gorm.DeletedAt)
// ===============================
// DELETE /api/a/school-profile-teacher-dkm/:id
func (ctrl *SchoolProfileTeacherDkmController) DeleteProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Where("school_profile_teacher_dkm_id = ?", id).
		Delete(&model.SchoolProfileTeacherDkmModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus profil")
	}

	return helper.JsonDeleted(c, "Profil berhasil dihapus", fiber.Map{
		"school_profile_teacher_dkm_id": id,
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
