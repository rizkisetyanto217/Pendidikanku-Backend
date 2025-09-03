// file: internals/features/lembaga/classes/main/controller/cpo_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/school/classes/classes/dto"
	"masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* ================= Controller & Constructor ================= */

type CPOController struct {
	DB *gorm.DB
}

func NewCPOController(db *gorm.DB) *CPOController {
	return &CPOController{DB: db}
}

/* ================= Helpers (tenant & mapping) ================= */

func (ctl *CPOController) ensureClassBelongsToMasjid(classID, masjidID uuid.UUID) error {
	var cnt int64
	if err := ctl.DB.Table("classes").
		Where("class_id = ? AND class_masjid_id = ? AND class_deleted_at IS NULL", classID, masjidID).
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return fiber.NewError(fiber.StatusForbidden, "Kelas tidak ditemukan di masjid ini")
	}
	return nil
}

func (ctl *CPOController) ensureCPOBelongsToMasjid(cpoID, masjidID uuid.UUID) (*model.ClassPricingOption, error) {
	var m model.ClassPricingOption
	// join ke classes agar dapat tenant
	err := ctl.DB.
		Table("class_pricing_options AS cpo").
		Select("cpo.*").
		Joins("JOIN classes c ON c.class_id = cpo.class_pricing_options_class_id").
		Where("cpo.class_pricing_options_id = ? AND c.class_masjid_id = ? AND c.class_deleted_at IS NULL", cpoID, masjidID).
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func mapCPOList(list []model.ClassPricingOption) []dto.ClassPricingOptionResp {
	out := make([]dto.ClassPricingOptionResp, 0, len(list))
	for _, m := range list {
		out = append(out, dto.FromModel(m))
	}
	return out
}

/* ================= USER ROUTES (read-only) ================= */

// GET /user/classes/:class_id/pricing-options?type=ONE_TIME|RECURRING&limit=&offset=
func (ctl *CPOController) UserListCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("class_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
	}

	// tenant check terhadap kelas
	if err := ctl.ensureClassBelongsToMasjid(classID, masjidID); err != nil {
		return err
	}

	priceType := strings.TrimSpace(c.Query("type"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	q := ctl.DB.Where("class_pricing_options_class_id = ? AND class_pricing_options_deleted_at IS NULL", classID)
	if priceType != "" {
		q = q.Where("class_pricing_options_price_type = ?", strings.ToUpper(priceType))
	}

	var list []model.ClassPricingOption
	if err := q.Order("class_pricing_options_created_at DESC, class_pricing_options_id DESC").
		Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// hitung total untuk pagination
	var total int64
	if err := ctl.DB.Model(&model.ClassPricingOption{}).
		Where("class_pricing_options_class_id = ? AND class_pricing_options_deleted_at IS NULL", classID).
		Scopes(func(db *gorm.DB) *gorm.DB {
			if priceType != "" {
				return db.Where("class_pricing_options_price_type = ?", strings.ToUpper(priceType))
			}
			return db
		}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	return helper.JsonList(c, mapCPOList(list), fiber.Map{"limit": limit, "offset": offset, "total": int(total)})
}

// GET /user/classes/:class_id/pricing-options/latest?type=ONE_TIME|RECURRING
func (ctl *CPOController) UserLatestCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("class_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
	}
	if err := ctl.ensureClassBelongsToMasjid(classID, masjidID); err != nil {
		return err
	}

	priceType := strings.TrimSpace(c.Query("type"))
	if priceType == "" {
		// kembalikan max 2 baris (latest per type)
		var latest []model.ClassPricingOption
		if err := ctl.DB.Raw(`
			SELECT DISTINCT ON (class_pricing_options_price_type) *
			FROM (
				SELECT * FROM class_pricing_options
				WHERE class_pricing_options_class_id = ? AND class_pricing_options_deleted_at IS NULL
				ORDER BY class_pricing_options_price_type, class_pricing_options_created_at DESC, class_pricing_options_id DESC
			) t
			ORDER BY class_pricing_options_price_type
		`, classID).Scan(&latest).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		return helper.JsonOK(c, "OK", mapCPOList(latest))
	}

	var m model.ClassPricingOption
	if err := ctl.DB.
		Where("class_pricing_options_class_id = ? AND class_pricing_options_price_type = ? AND class_pricing_options_deleted_at IS NULL",
			classID, strings.ToUpper(priceType)).
		Order("class_pricing_options_created_at DESC, class_pricing_options_id DESC").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Tidak ada data")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return helper.JsonOK(c, "OK", dto.FromModel(m))
}

// GET /user/pricing-options/:id
func (ctl *CPOController) UserGetCPOByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := ctl.ensureCPOBelongsToMasjid(id, masjidID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassPricingOptionsDeletedAt != nil {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}
	return helper.JsonOK(c, "OK", dto.FromModel(*m))
}

/* ================= ADMIN-DKM ROUTES (CRUD + restore) ================= */

// POST /admin-dkm/classes/:class_id/pricing-options
func (ctl *CPOController) AdminCreateCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("class_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
	}
	if err := ctl.ensureClassBelongsToMasjid(classID, masjidID); err != nil {
		return err
	}

	var req dto.CreateClassPricingOptionReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// ambil class_id dari path (single source of truth)
	req.ClassPricingOptionsClassID = classID

	// validate + cross-field combo
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := req.NormalizeAndValidateCombo(); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()
	if err := ctl.DB.Create(&m).Error; err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "uq_class_pricing_options_label_per_class") || strings.Contains(low, "unique") || strings.Contains(low, "duplicate") {
			return fiber.NewError(fiber.StatusConflict, "Label pricing sudah dipakai di kelas ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}

	return helper.JsonCreated(c, "Berhasil membuat pricing option", dto.FromModel(m))
}

// GET /admin-dkm/classes/:class_id/pricing-options?type=&include_deleted=true|false&limit=&offset=
func (ctl *CPOController) AdminListCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("class_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
	}
	if err := ctl.ensureClassBelongsToMasjid(classID, masjidID); err != nil {
		return err
	}

	priceType := strings.TrimSpace(c.Query("type"))
	includeDeleted := strings.EqualFold(c.Query("include_deleted"), "true")

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	q := ctl.DB.Where("class_pricing_options_class_id = ?", classID)
	if priceType != "" {
		q = q.Where("class_pricing_options_price_type = ?", strings.ToUpper(priceType))
	}
	if !includeDeleted {
		q = q.Where("class_pricing_options_deleted_at IS NULL")
	}

	var list []model.ClassPricingOption
	if err := q.Order("class_pricing_options_created_at DESC, class_pricing_options_id DESC").
		Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// total
	var total int64
	if err := ctl.DB.Model(&model.ClassPricingOption{}).
		Where("class_pricing_options_class_id = ?", classID).
		Scopes(func(db *gorm.DB) *gorm.DB {
			if priceType != "" {
				db = db.Where("class_pricing_options_price_type = ?", strings.ToUpper(priceType))
			}
			if !includeDeleted {
				db = db.Where("class_pricing_options_deleted_at IS NULL")
			}
			return db
		}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	return helper.JsonList(c, mapCPOList(list), fiber.Map{"limit": limit, "offset": offset, "total": int(total)})
}

// GET /admin-dkm/classes/:class_id/pricing-options/latest?type=
func (ctl *CPOController) AdminLatestCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("class_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
	}
	if err := ctl.ensureClassBelongsToMasjid(classID, masjidID); err != nil {
		return err
	}

	priceType := strings.TrimSpace(c.Query("type"))
	if priceType == "" {
		var latest []model.ClassPricingOption
		if err := ctl.DB.Raw(`
			SELECT DISTINCT ON (class_pricing_options_price_type) *
			FROM (
				SELECT * FROM class_pricing_options
				WHERE class_pricing_options_class_id = ? AND class_pricing_options_deleted_at IS NULL
				ORDER BY class_pricing_options_price_type, class_pricing_options_created_at DESC, class_pricing_options_id DESC
			) t
			ORDER BY class_pricing_options_price_type
		`, classID).Scan(&latest).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		return helper.JsonOK(c, "OK", mapCPOList(latest))
	}

	var m model.ClassPricingOption
	if err := ctl.DB.
		Where("class_pricing_options_class_id = ? AND class_pricing_options_price_type = ? AND class_pricing_options_deleted_at IS NULL",
			classID, strings.ToUpper(priceType)).
		Order("class_pricing_options_created_at DESC, class_pricing_options_id DESC").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Tidak ada data")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return helper.JsonOK(c, "OK", dto.FromModel(m))
}

// GET /admin-dkm/pricing-options/:id
func (ctl *CPOController) AdminGetCPOByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := ctl.ensureCPOBelongsToMasjid(id, masjidID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return helper.JsonOK(c, "OK", dto.FromModel(*m))
}

// PUT /admin-dkm/pricing-options/:id  (full replace)
func (ctl *CPOController) AdminReplaceCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Pastikan CPO milik masjid & load data (boleh terhapus/aktif)
	m, err := ctl.ensureCPOBelongsToMasjid(id, masjidID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat data")
	}
	// Tidak boleh update jika sudah soft-deleted
	if m.ClassPricingOptionsDeletedAt != nil {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	var req dto.PutClassPricingOptionReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := req.NormalizeAndValidateCombo(); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Full replace fields (kecuali class_id & timestamps)
	m.ClassPricingOptionsLabel = req.ClassPricingOptionsLabel
	m.ClassPricingOptionsPriceType = strings.ToUpper(req.ClassPricingOptionsPriceType)
	m.ClassPricingOptionsAmountIDR = req.ClassPricingOptionsAmountIDR
	if m.ClassPricingOptionsPriceType == dto.PriceTypeOneTime {
		m.ClassPricingOptionsRecurrenceMonths = nil
	} else {
		m.ClassPricingOptionsRecurrenceMonths = req.ClassPricingOptionsRecurrenceMonths
	}

	if err := ctl.DB.Save(m).Error; err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "uq_class_pricing_options_label_per_class") || strings.Contains(low, "unique") || strings.Contains(low, "duplicate") {
			return fiber.NewError(fiber.StatusConflict, "Label pricing sudah dipakai di kelas ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Berhasil memperbarui pricing option", dto.FromModel(*m))
}

// DELETE /admin-dkm/pricing-options/:id (soft delete)
func (ctl *CPOController) AdminSoftDeleteCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// pastikan kepemilikan (join ke classes)
	if _, err := ctl.ensureCPOBelongsToMasjid(id, masjidID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat data")
	}

	now := time.Now().UTC()
	res := ctl.DB.Model(&model.ClassPricingOption{}).
		Where("class_pricing_options_id = ? AND class_pricing_options_deleted_at IS NULL", id).
		Update("class_pricing_options_deleted_at", now)
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	if res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}
	return helper.JsonDeleted(c, "Pricing option berhasil dihapus", fiber.Map{"class_pricing_options_id": id})
}

// POST /admin-dkm/pricing-options/:id/restore
func (ctl *CPOController) AdminRestoreCPO(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// pastikan kepemilikan (meski deleted, join ke classes via class_id)
	if _, err := ctl.ensureCPOBelongsToMasjid(id, masjidID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan / kelas sudah tidak aktif")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat data")
	}

	res := ctl.DB.Model(&model.ClassPricingOption{}).
		Where("class_pricing_options_id = ? AND class_pricing_options_deleted_at IS NOT NULL", id).
		Update("class_pricing_options_deleted_at", nil)
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal restore data")
	}
	if res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan / sudah aktif")
	}
	return helper.JsonOK(c, "Pricing option berhasil direstore", fiber.Map{"class_pricing_options_id": id})
}
