package controller

import (
	"errors"
	"strings"
	"time"

	cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
	cpmodel "masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassParentController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassParentController(db *gorm.DB, v *validator.Validate) *ClassParentController {
	if v == nil {
		v = validator.New()
	}
	return &ClassParentController{DB: db, Validate: v}
}

func (ctl *ClassParentController) v() *validator.Validate {
	if ctl.Validate == nil {
		ctl.Validate = validator.New()
	}
	return ctl.Validate
}

// ---------- helpers ----------
func clampLimit(limit, def, max int) int {
	if limit <= 0 { return def }
	if limit > max { return max }
	return limit
}

// unik code per masjid (alive only), excludeID opsional
func (ctl *ClassParentController) codeExists(masjidID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	tx := ctl.DB.
		Model(&cpmodel.ClassParentModel{}).
		Where(`
			class_parent_masjid_id = ?
			AND class_parent_deleted_at IS NULL
			AND class_parent_delete_pending_until IS NULL
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

// ---------- CREATE ----------
func (ctl *ClassParentController) Create(c *fiber.Ctx) error {
	var req cpdto.CreateClassParentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.v().Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Masjid dari token (bukan dari body)
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	// Kalau client tetap kirim masjid_id dan beda → tolak
	if req.ClassParentMasjidID != uuid.Nil && req.ClassParentMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "class_parent_masjid_id pada body tidak boleh berbeda dengan token")
	}

	// Unik code per masjid
	if code := strings.TrimSpace(req.ClassParentCode); code != "" {
		exists, err := ctl.codeExists(masjidID, code, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		if exists {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
	}

	m := req.ToModel()
	m.ClassParentMasjidID = masjidID

	// Multipart image? → upload ke OSS (scope masjid)
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		publicURL, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		m.ClassParentImageURL = publicURL
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "uq_class_parent") && strings.Contains(low, "code") {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat class parent")
	}

	return helper.JsonCreated(c, "Class parent berhasil dibuat", cpdto.ToClassParentResponse(m))
}

// ---------- GET BY ID (tenant-safe) ----------
func (ctl *ClassParentController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	return helper.JsonOK(c, "OK", cpdto.ToClassParentResponse(m))
}

// ---------- LIST (tenant-safe) ----------
func (ctl *ClassParentController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var q cpdto.ListClassParentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	q.Limit = clampLimit(q.Limit, 20, 200)
	if q.Offset < 0 { q.Offset = 0 }

	tx := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where("class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", masjidID)

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
	if err := tx.
		Order("class_parent_created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := cpdto.ToClassParentResponses(rows)
	meta := cpdto.NewPaginationMeta(total, q.Limit, q.Offset, len(resps))
	return helper.JsonList(c, resps, meta)
}

// ---------- UPDATE (PATCH, tenant-safe) ----------
func (ctl *ClassParentController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req cpdto.UpdateClassParentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.v().Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// unik code bila diubah
	if req.ClassParentCode != nil && strings.TrimSpace(*req.ClassParentCode) != "" {
		exists, err := ctl.codeExists(masjidID, *req.ClassParentCode, &m.ClassParentID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		if exists {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
	}

	// apply patch
	req.ApplyPatch(&m)

	// clear image via empty string → pindah ke spam
	if req.ClassParentImageURL != nil &&
		strings.TrimSpace(*req.ClassParentImageURL) == "" &&
		strings.TrimSpace(m.ClassParentImageURL) != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second)
		m.ClassParentImageURL = ""
	}

	// file baru? upload & replace + spam-kan lama
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		publicURL, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		if strings.TrimSpace(m.ClassParentImageURL) != "" {
			_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second)
		}
		m.ClassParentImageURL = publicURL
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}
	return helper.JsonUpdated(c, "Class parent berhasil diperbarui", cpdto.ToClassParentResponse(m))
}

// ---------- DELETE (soft, tenant-safe) ----------
func (ctl *ClassParentController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	if strings.TrimSpace(m.ClassParentImageURL) != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second)
	}

	// soft delete (pastikan model pakai gorm.DeletedAt di kolom class_parent_deleted_at)
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ?", id, masjidID).
		Delete(&cpmodel.ClassParentModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	return helper.JsonDeleted(c, "Class parent berhasil dihapus", fiber.Map{"class_parent_id": id})
}
