// internals/features/masjid/service_plans/controller/masjid_service_plan_controller.go
package controller

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	yDTO "masjidku_backend/internals/features/lembaga/masjids/dto"
	yModel "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers" // ← helper lama (JSON utils + OSS utils)
	helperOSS "masjidku_backend/internals/helpers/oss"
)

// Controller cukup DB saja (tanpa BlobService)
type MasjidServicePlanController struct {
	DB *gorm.DB
}

func NewMasjidServicePlanController(db *gorm.DB) *MasjidServicePlanController {
	return &MasjidServicePlanController{DB: db}
}

/* ===== util lokal upload pakai helper lama ===== */

func uploadSPImage(c *fiber.Ctx, fh *multipart.FileHeader) (publicURL, objectKey string, err error) {
	svc, err := helperOSS.NewOSSServiceFromEnv("uploads") // kosongkan "" kalau tak pakai prefix
	if err != nil {
		return "", "", fiber.NewError(fiber.StatusBadGateway, "OSS tidak siap")
	}
	// simpan mentah (tanpa recompress) di folder global
	key, _, uerr := svc.UploadFromFormFileToDir(c.Context(), "service-plans/images", fh)
	if uerr != nil {
		return "", "", fiber.NewError(fiber.StatusBadGateway, "Gagal upload ke OSS")
	}
	return svc.PublicURL(key), key, nil
}

/* ===================== HANDLERS ===================== */

// POST /api/o/masjid-service-plans
// - form-data (multipart): code, name, (description optional), image (File, optional)
// - JSON: payload lama (image_url & image_object_key harus berpasangan)
func (h *MasjidServicePlanController) Create(c *fiber.Ctx) error {
	// MODE 1: multipart
	if helperOSS.IsMultipart(c) {
		code := strings.TrimSpace(c.FormValue("masjid_service_plan_code"))
		name := strings.TrimSpace(c.FormValue("masjid_service_plan_name"))
		desc := strings.TrimSpace(c.FormValue("masjid_service_plan_description"))
		if code == "" { return helper.JsonError(c, fiber.StatusBadRequest, "Kode plan wajib diisi") }
		if name == "" { return helper.JsonError(c, fiber.StatusBadRequest, "Nama plan wajib diisi") }

		var imgURLPtr, imgKeyPtr *string
		if fh, err := helperOSS.GetImageFile(c); err != nil {
			return helper.JsonError(c, fiber.StatusUnsupportedMediaType, err.Error())
		} else if fh != nil {
			publicURL, objectKey, uerr := uploadSPImage(c, fh)
			if uerr != nil { return helper.JsonError(c, fiber.StatusBadGateway, uerr.Error()) }
			imgURLPtr, imgKeyPtr = &publicURL, &objectKey
		}

		var descPtr *string
		if desc != "" { descPtr = &desc }

		m := &yModel.MasjidServicePlan{
			MasjidServicePlanCode:        code,
			MasjidServicePlanName:        name,
			MasjidServicePlanDescription: descPtr,
			MasjidServicePlanImageURL:       imgURLPtr,
			MasjidServicePlanImageObjectKey: imgKeyPtr,
		}
		if err := h.DB.Create(m).Error; err != nil {
			if yDTO.IsUniqueViolation(err) {
				return helper.JsonError(c, fiber.StatusConflict, "Kode plan sudah digunakan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat service plan")
		}
		return helper.JsonCreated(c, "Service plan berhasil dibuat", yDTO.NewMasjidServicePlanResponse(m))
	}

	// MODE 2: JSON (lama)
	var req yDTO.CreateMasjidServicePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if strings.TrimSpace(req.MasjidServicePlanCode) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Kode plan wajib diisi")
	}
	if strings.TrimSpace(req.MasjidServicePlanName) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama plan wajib diisi")
	}
	if (req.MasjidServicePlanImageURL == nil) != (req.MasjidServicePlanImageObjectKey == nil) {
		return helper.JsonError(c, fiber.StatusBadRequest, "image_url & image_object_key harus berpasangan")
	}

	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		if yDTO.IsUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode plan sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat service plan")
	}
	return helper.JsonCreated(c, "Service plan berhasil dibuat", yDTO.NewMasjidServicePlanResponse(m))
}

// PATCH /api/o/masjid-service-plans/:id
// - form-data (multipart) hanya untuk ganti gambar (field: image)
// - JSON patch: seluruh field + pasangan image_url/object_key
func (h *MasjidServicePlanController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid") }

	m, err := h.findByID(id, false)
	if err != nil { return err }

	// MODE 1: multipart → ganti gambar
	if helperOSS.IsMultipart(c) {
		fh, ferr := helperOSS.GetImageFile(c)
		if ferr != nil { return helper.JsonError(c, fiber.StatusUnsupportedMediaType, ferr.Error()) }
		if fh == nil { return helper.JsonError(c, fiber.StatusBadRequest, "File tidak ditemukan pada field image/file/photo/picture") }

		publicURL, objectKey, uerr := uploadSPImage(c, fh)
		if uerr != nil { return helper.JsonError(c, fiber.StatusBadGateway, uerr.Error()) }

		// rotasi current → old
		if m.MasjidServicePlanImageURL != nil || m.MasjidServicePlanImageObjectKey != nil {
			m.MasjidServicePlanImageURLOld       = m.MasjidServicePlanImageURL
			m.MasjidServicePlanImageObjectKeyOld = m.MasjidServicePlanImageObjectKey
			t := time.Now().Add(helperOSS.TrashRetention())
			m.MasjidServicePlanImageDeletePendingUntil = &t
		} else {
			m.MasjidServicePlanImageURLOld = nil
			m.MasjidServicePlanImageObjectKeyOld = nil
			m.MasjidServicePlanImageDeletePendingUntil = nil
		}

		m.MasjidServicePlanImageURL       = &publicURL
		m.MasjidServicePlanImageObjectKey = &objectKey
		m.MasjidServicePlanUpdatedAt      = time.Now()

		if err := h.DB.Save(m).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui service plan")
		}
		return helper.JsonUpdated(c, "Service plan diperbarui (image)", yDTO.NewMasjidServicePlanResponse(m))
	}

	// MODE 2: JSON patch
	var req yDTO.UpdateMasjidServicePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := req.ApplyToModelWithImageSwap(m, helperOSS.TrashRetention()); err != nil {
		if errors.Is(err, yDTO.ErrImagePairMismatch) {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
		return helper.JsonError(c, fiber.StatusBadRequest, "Patch tidak valid")
	}

	m.MasjidServicePlanUpdatedAt = time.Now()
	if err := h.DB.Save(m).Error; err != nil {
		if yDTO.IsUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode plan sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui service plan")
	}
	return helper.JsonUpdated(c, "Service plan diperbarui", yDTO.NewMasjidServicePlanResponse(m))
}

// DELETE /api/o/masjid-service-plans/:id (?hard=true)
func (h *MasjidServicePlanController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid") }

	hard := strings.EqualFold(c.Query("hard"), "true")
	m, err := h.findByID(id, hard)
	if err != nil { return err }

	if hard {
		// simpan URL buat cleanup
		cur := m.MasjidServicePlanImageURL
		old := m.MasjidServicePlanImageURLOld

		if err := h.DB.Unscoped().Delete(&yModel.MasjidServicePlan{}, "masjid_service_plan_id = ?", id).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus permanen")
		}
		// best-effort hapus file
		if cur != nil { _ = helperOSS.DeleteByPublicURLENV(*cur, 0) }
		if old != nil { _ = helperOSS.DeleteByPublicURLENV(*old, 0) }

		return helper.JsonDeleted(c, "Service plan dihapus permanen", fiber.Map{"id": id})
	}

	if err := h.DB.Delete(&yModel.MasjidServicePlan{}, "masjid_service_plan_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus service plan")
	}
	return helper.JsonDeleted(c, "Service plan dihapus", fiber.Map{"id": id})
}

// POST /api/o/masjid-service-plans/:id/restore
func (h *MasjidServicePlanController) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid") }

	m, err := h.findByID(id, true)
	if err != nil { return err }
	if !m.MasjidServicePlanDeletedAt.Valid {
		return helper.JsonError(c, fiber.StatusBadRequest, "Service plan tidak dalam status terhapus")
	}

	if err := h.DB.Unscoped().
		Model(&yModel.MasjidServicePlan{}).
		Where("masjid_service_plan_id = ?", id).
		Update("masjid_service_plan_deleted_at", nil).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulihkan service plan")
	}
	m, _ = h.findByID(id, false)
	return helper.JsonOK(c, "Service plan dipulihkan", yDTO.NewMasjidServicePlanResponse(m))
}

// GET /api/o/masjid-service-plans/:id
func (h *MasjidServicePlanController) Detail(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid") }
	m, err := h.findByID(id, false)
	if err != nil { return err }
	return helper.JsonOK(c, "Detail service plan", yDTO.NewMasjidServicePlanResponse(m))
}

// GET /api/o/masjid-service-plans
func (h *MasjidServicePlanController) List(c *fiber.Ctx) error {
	req, _ := http.NewRequest("GET", "http://local"+c.OriginalURL(), nil)
	p := helper.ParseWith(req, "created_at", "desc", helper.AdminOpts)

	orderClause, err := p.SafeOrderClause(map[string]string{
		"created_at":    "masjid_service_plan_created_at",
		"updated_at":    "masjid_service_plan_updated_at",
		"name":          "lower(masjid_service_plan_name)",
		"code":          "lower(masjid_service_plan_code)",
		"price_monthly": "masjid_service_plan_price_monthly",
	}, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak dikenal")
	}
	orderExpr := strings.TrimPrefix(strings.TrimSpace(orderClause), "ORDER BY ")

	dbq := h.DB.Model(&yModel.MasjidServicePlan{})

	if v := strings.TrimSpace(c.Query("code")); v != "" {
		dbq = dbq.Where("LOWER(masjid_service_plan_code) = LOWER(?)", v)
	}
	if v := strings.TrimSpace(c.Query("name")); v != "" {
		dbq = dbq.Where("masjid_service_plan_name ILIKE ?", "%"+v+"%")
	}
	if v := c.Query("active"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			dbq = dbq.Where("masjid_service_plan_is_active = ?", b)
		}
	}
	if v := c.Query("allow_custom_theme"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			dbq = dbq.Where("masjid_service_plan_allow_custom_theme = ?", b)
		}
	}
	if v := c.Query("price_monthly_min"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			dbq = dbq.Where("masjid_service_plan_price_monthly >= ?", f)
		}
	}
	if v := c.Query("price_monthly_max"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			dbq = dbq.Where("masjid_service_plan_price_monthly <= ?", f)
		}
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var rows []yModel.MasjidServicePlan
	if err := dbq.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]*yDTO.MasjidServicePlanResponse, 0, len(rows))
	for i := range rows {
		items = append(items, yDTO.NewMasjidServicePlanResponse(&rows[i]))
	}
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}

/* ===================== HELPERS ===================== */

func (h *MasjidServicePlanController) findByID(id uuid.UUID, includeDeleted bool) (*yModel.MasjidServicePlan, error) {
	var m yModel.MasjidServicePlan
	q := h.DB.Model(&yModel.MasjidServicePlan{})
	if includeDeleted { q = q.Unscoped() }
	if err := q.Where("masjid_service_plan_id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Service plan tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return &m, nil
}
