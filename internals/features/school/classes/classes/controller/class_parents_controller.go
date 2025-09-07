package controller

import (
	"errors"
	"strings"
	"time"

	cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
	cpmodel "masjidku_backend/internals/features/school/classes/classes/model"
	helperOSS "masjidku_backend/internals/helpers/oss"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassParentController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassParentController(db *gorm.DB, validate *validator.Validate) *ClassParentController {
	return &ClassParentController{DB: db, Validate: validate}
}

/* =========================
   Helpers (private)
========================= */

func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func (ctl *ClassParentController) codeExists(c *fiber.Ctx, masjidID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	tx := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where(`
			class_parent_masjid_id = ?
			AND class_parent_deleted_at IS NULL
			AND class_parent_code IS NOT NULL
			AND LOWER(class_parent_code) = LOWER(?)
		`, masjidID, code)

	if excludeID != nil {
		tx = tx.Where("class_parent_id <> ?", *excludeID)
	}

	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}


/* =========================
   CREATE
========================= */

func (ctl *ClassParentController) Create(c *fiber.Ctx) error {
	var req cpdto.CreateClassParentRequest

	// Fiber bisa parse JSON & multipart (field text)
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Unique code per masjid (alive only)
	if strings.TrimSpace(req.ClassParentCode) != "" {
		exists, err := ctl.codeExists(c, req.ClassParentMasjidID, req.ClassParentCode, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		if exists {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
	}

	m := req.ToModel()

	// Cek ada file gambar? -> upload ke OSS (scoped)
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		url, upErr := helperOSS.UploadImageToOSSScoped(req.ClassParentMasjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		m.ClassParentImageURL = url
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat class parent")
	}

	resp := cpdto.ToClassParentResponse(m)
	return helper.JsonCreated(c, "Class parent berhasil dibuat", resp)
}

/* =========================
   GET BY ID
========================= */

func (ctl *ClassParentController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	return helper.JsonOK(c, "OK", cpdto.ToClassParentResponse(m))
}

/* =========================
   LIST
========================= */

func (ctl *ClassParentController) List(c *fiber.Ctx) error {
	var q cpdto.ListClassParentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// paging
	q.Limit = clampLimit(q.Limit, 20, 100)
	if q.Offset < 0 {
		q.Offset = 0
	}

	tx := ctl.DB.WithContext(c.Context()).Model(&cpmodel.ClassParentModel{})

	if q.MasjidID != nil {
		tx = tx.Where("class_parent_masjid_id = ?", *q.MasjidID)
	}
	if q.Active != nil {
		tx = tx.Where("class_parent_is_active = ?", *q.Active)
	}
	if q.LevelMin != nil {
		tx = tx.Where("(class_parent_level IS NOT NULL AND class_parent_level >= ?)", *q.LevelMin)
	}
	if q.LevelMax != nil {
		tx = tx.Where("(class_parent_level IS NOT NULL AND class_parent_level <= ?)", *q.LevelMax)
	}
	if q.CreatedGt != nil {
		tx = tx.Where("class_parent_created_at > ?", *q.CreatedGt)
	}
	if q.CreatedLt != nil {
		tx = tx.Where("class_parent_created_at < ?", *q.CreatedLt)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		pat := "%" + s + "%"
		tx = tx.Where(`
			class_parent_name ILIKE ? OR
			class_parent_code ILIKE ? OR
			class_parent_description ILIKE ?
		`, pat, pat, pat)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	var rows []cpmodel.ClassParentModel
	if err := tx.Order("class_parent_created_at DESC").
		Limit(q.Limit).Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := cpdto.ToClassParentResponses(rows)
	meta := cpdto.NewPaginationMeta(total, q.Limit, q.Offset, len(resps))

	return helper.JsonList(c, resps, meta)
}

/* =========================
   UPDATE (PATCH)
========================= */

func (ctl *ClassParentController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req cpdto.UpdateClassParentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// unique code check (if code changed)
	if req.ClassParentCode != nil && strings.TrimSpace(*req.ClassParentCode) != "" {
		exists, err := ctl.codeExists(c, m.ClassParentMasjidID, *req.ClassParentCode, &m.ClassParentID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		if exists {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
	}

	// Apply patch text fields
	req.ApplyPatch(&m)

	// Jika klien ingin clear image_url → pindah ke "spam/" (bukan delete langsung)
	if req.ClassParentImageURL != nil && strings.TrimSpace(*req.ClassParentImageURL) == "" && strings.TrimSpace(m.ClassParentImageURL) != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second) // best-effort
		m.ClassParentImageURL = ""
	}

	// Ada file baru? upload & replace (lama dipindah ke spam/)
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		newURL, upErr := helperOSS.UploadImageToOSSScoped(m.ClassParentMasjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		// pindahkan yang lama ke spam
		if strings.TrimSpace(m.ClassParentImageURL) != "" {
			_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second) // best-effort
		}
		m.ClassParentImageURL = newURL
	}

	// Save
	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Class parent berhasil diperbarui", cpdto.ToClassParentResponse(m))
}

/* =========================
   DELETE (soft)
========================= */

func (ctl *ClassParentController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil record dulu untuk memindahkan gambar ke spam/
	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ?", id).
		First(&m).Error; err == nil {
		if strings.TrimSpace(m.ClassParentImageURL) != "" {
			_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second) // best-effort
		}
	}

	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ?", id).
		Delete(&cpmodel.ClassParentModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Class parent berhasil dihapus", fiber.Map{"class_parent_id": id})
}
