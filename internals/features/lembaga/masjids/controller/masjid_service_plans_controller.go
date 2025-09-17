// internals/features/masjid/service_plans/controller/masjid_service_plan_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	spDTO "masjidku_backend/internals/features/lembaga/masjids/dto"
	spModel "masjidku_backend/internals/features/lembaga/masjids/model"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/* =========================================================
   Controller
========================================================= */

type MasjidServicePlanController struct {
	DB *gorm.DB
}

func NewMasjidServicePlanController(db *gorm.DB) *MasjidServicePlanController {
	return &MasjidServicePlanController{DB: db}
}

/* =========================================================
   Utils
========================================================= */

func httpErr(c *fiber.Ctx, code int, msg string) error {
	return c.Status(code).JSON(fiber.Map{"error": msg})
}

func parseRetentionDays() time.Duration {
	d := 30 // default 30 hari
	if v := strings.TrimSpace(os.Getenv("IMAGE_RETENTION_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			d = n
		}
	}
	return time.Hour * 24 * time.Duration(d)
}

func getUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Params(name))
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "invalid "+name)
	}
	return id, nil
}

/* =========================================================
   Routes wiring (opsional)
========================================================= */

func (ctl *MasjidServicePlanController) Mount(r fiber.Router) {
	g := r.Group("/masjid-service-plans")
	g.Post("/", ctl.Create)
	g.Get("/", ctl.List)
	g.Get("/:id", ctl.GetByID)
	g.Patch("/:id", ctl.Patch)
	g.Delete("/:id", ctl.SoftDelete)
	g.Post("/:id/restore", ctl.Restore)

	// Upload image (multipart/form-data, field name: file)
	g.Post("/:id/image", ctl.UploadImageAndSwap)

	// Cleanup gambar lama yang sudah lewat retensi
	g.Post("/_cleanup-expired-images", ctl.CleanupExpiredOldImages)
}

/* =========================================================
   CREATE
========================================================= */

func (ctl *MasjidServicePlanController) Create(c *fiber.Ctx) error {
	var req spDTO.CreateMasjidServicePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return httpErr(c, fiber.StatusBadRequest, "invalid payload")
	}

	// Validasi sederhana (kamu bisa sambungkan ke validator.v10 bila perlu)
	if strings.TrimSpace(req.MasjidServicePlanCode) == "" ||
		strings.TrimSpace(req.MasjidServicePlanName) == "" {
		return httpErr(c, fiber.StatusBadRequest, "code & name are required")
	}

	m := req.ToModel()
	now := time.Now()
	m.MasjidServicePlanCreatedAt = now
	m.MasjidServicePlanUpdatedAt = now

	if err := ctl.DB.Create(m).Error; err != nil {
		if spDTO.IsUniqueViolation(err) {
			return httpErr(c, fiber.StatusConflict, "duplicate code")
		}
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(spDTO.NewMasjidServicePlanResponse(m))
}

/* =========================================================
   LIST (filter + sort + pagination)
========================================================= */

func (ctl *MasjidServicePlanController) List(c *fiber.Ctx) error {
	var q spDTO.ListMasjidServicePlanQuery
	if err := c.QueryParser(&q); err != nil {
		return httpErr(c, fiber.StatusBadRequest, "invalid query")
	}

	db := ctl.DB.Model(&spModel.MasjidServicePlan{})

	if q.Code != nil && strings.TrimSpace(*q.Code) != "" {
		db = db.Where("masjid_service_plan_code ILIKE ?", "%"+strings.TrimSpace(*q.Code)+"%")
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		db = db.Where("masjid_service_plan_name ILIKE ?", "%"+strings.TrimSpace(*q.Name)+"%")
	}
	if q.Active != nil {
		db = db.Where("masjid_service_plan_is_active = ?", *q.Active)
	}
	if q.AllowCustomTheme != nil {
		db = db.Where("masjid_service_plan_allow_custom_theme = ?", *q.AllowCustomTheme)
	}
	if q.PriceMonthlyMin != nil {
		db = db.Where("(masjid_service_plan_price_monthly IS NOT NULL AND masjid_service_plan_price_monthly >= ?)", *q.PriceMonthlyMin)
	}
	if q.PriceMonthlyMax != nil {
		db = db.Where("(masjid_service_plan_price_monthly IS NOT NULL AND masjid_service_plan_price_monthly <= ?)", *q.PriceMonthlyMax)
	}

	// Sorting
	sort := "masjid_service_plan_created_at DESC"
	if q.Sort != nil {
		switch *q.Sort {
		case "name_asc":
			sort = "masjid_service_plan_name ASC"
		case "name_desc":
			sort = "masjid_service_plan_name DESC"
		case "price_monthly_asc":
			sort = "masjid_service_plan_price_monthly ASC NULLS LAST"
		case "price_monthly_desc":
			sort = "masjid_service_plan_price_monthly DESC NULLS LAST"
		case "created_at_asc":
			sort = "masjid_service_plan_created_at ASC"
		case "created_at_desc":
			sort = "masjid_service_plan_created_at DESC"
		case "updated_at_asc":
			sort = "masjid_service_plan_updated_at ASC"
		case "updated_at_desc":
			sort = "masjid_service_plan_updated_at DESC"
		}
	}
	db = db.Order(sort)

	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	var rows []spModel.MasjidServicePlan
	if err := db.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}

	resp := make([]*spDTO.MasjidServicePlanResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, spDTO.NewMasjidServicePlanResponse(&rows[i]))
	}
	return c.JSON(fiber.Map{
		"data":   resp,
		"limit":  limit,
		"offset": offset,
	})
}

/* =========================================================
   GET BY ID
========================================================= */

func (ctl *MasjidServicePlanController) GetByID(c *fiber.Ctx) error {
	id, err := getUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var m spModel.MasjidServicePlan
	if err := ctl.DB.First(&m, "masjid_service_plan_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "not found")
		}
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}
	return c.JSON(spDTO.NewMasjidServicePlanResponse(&m))
}

/* =========================================================
   PATCH (JSON) — termasuk mekanisme image 2-slot via DTO.ApplyToModelWithImageSwap
========================================================= */

func (ctl *MasjidServicePlanController) Patch(c *fiber.Ctx) error {
	id, err := getUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req spDTO.UpdateMasjidServicePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return httpErr(c, fiber.StatusBadRequest, "invalid payload")
	}

	var m spModel.MasjidServicePlan
	if err := ctl.DB.First(&m, "masjid_service_plan_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "not found")
		}
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}

	if err := req.ApplyToModelWithImageSwap(&m, parseRetentionDays()); err != nil {
		if errors.Is(err, spDTO.ErrImagePairMismatch) {
			return httpErr(c, fiber.StatusBadRequest, err.Error())
		}
		return httpErr(c, fiber.StatusBadRequest, "apply patch: "+err.Error())
	}

	// Simpan
	if err := ctl.DB.Save(&m).Error; err != nil {
		if spDTO.IsUniqueViolation(err) {
			return httpErr(c, fiber.StatusConflict, "duplicate code")
		}
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}

	return c.JSON(spDTO.NewMasjidServicePlanResponse(&m))
}

/* =========================================================
   SOFT-DELETE & RESTORE
========================================================= */

func (ctl *MasjidServicePlanController) SoftDelete(c *fiber.Ctx) error {
	id, err := getUUIDParam(c, "id")
	if err != nil {
		return err
	}
	if err := ctl.DB.Delete(&spModel.MasjidServicePlan{}, "masjid_service_plan_id = ?", id).Error; err != nil {
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}
	return c.SendStatus(http.StatusNoContent)
}

func (ctl *MasjidServicePlanController) Restore(c *fiber.Ctx) error {
	id, err := getUUIDParam(c, "id")
	if err != nil {
		return err
	}
	// Unscoped + Update deleted_at = NULL
	if err := ctl.DB.Unscoped().
		Model(&spModel.MasjidServicePlan{}).
		Where("masjid_service_plan_id = ?", id).
		Update("masjid_service_plan_deleted_at", gorm.DeletedAt{}).Error; err != nil {
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}
	return c.SendStatus(http.StatusNoContent)
}

/* =========================================================
   UPLOAD IMAGE via multipart (field: "file")
   - Re-encode ke WebP via helper.OSSService.UploadAsWebP
   - Swap ke model (current→old) + set delete_pending_until
========================================================= */

func (ctl *MasjidServicePlanController) UploadImageAndSwap(c *fiber.Ctx) error {
	id, err := getUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// Ambil record
	var m spModel.MasjidServicePlan
	if err := ctl.DB.First(&m, "masjid_service_plan_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "not found")
		}
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}

	// File wajib
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return httpErr(c, fiber.StatusBadRequest, "missing file")
	}

	// OSS service
	oss, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		return httpErr(c, fiber.StatusBadGateway, "oss init: "+err.Error())
	}

	// Upload → dapat public URL (webp)
	ctx := c.Context()
	publicURL, upErr := oss.UploadAsWebP(ctx, fh, fmt.Sprintf("masjid-service-plans/%s", id.String()))
	if upErr != nil {
		// helper sudah memetakan error media unsupported jadi fiber error
		return httpErr(c, fiber.StatusBadGateway, upErr.Error())
	}

	// Swap ke 2-slot (current→old + retention)
	ret := parseRetentionDays()
	now := time.Now()

	// Simpan old dulu bila ada current
	if m.MasjidServicePlanImageURL != nil && m.MasjidServicePlanImageObjectKey != nil {
		// Extract key dari current
		curKey, _ := helperOSS.ExtractKeyFromPublicURL(*m.MasjidServicePlanImageURL)
		m.MasjidServicePlanImageURLOld = m.MasjidServicePlanImageURL
		m.MasjidServicePlanImageObjectKeyOld = m.MasjidServicePlanImageObjectKey
		if ret > 0 {
			t := now.Add(ret)
			m.MasjidServicePlanImageDeletePendingUntil = &t
		} else {
			m.MasjidServicePlanImageDeletePendingUntil = nil
		}
		// NOTE: kita tidak menghapus object current sekarang; tetap tunggu cleanup retensi
		_ = curKey
	} else {
		m.MasjidServicePlanImageURLOld = nil
		m.MasjidServicePlanImageObjectKeyOld = nil
		m.MasjidServicePlanImageDeletePendingUntil = nil
	}

	// Set current dari upload baru
	// object_key untuk current
	newKey, err := helperOSS.ExtractKeyFromPublicURL(publicURL)
	if err != nil {
		return httpErr(c, fiber.StatusBadGateway, "extract key: "+err.Error())
	}
	m.MasjidServicePlanImageURL = &publicURL
	m.MasjidServicePlanImageObjectKey = &newKey
	m.MasjidServicePlanUpdatedAt = now

	// Save
	if err := ctl.DB.Save(&m).Error; err != nil {
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}
	return c.Status(fiber.StatusOK).JSON(spDTO.NewMasjidServicePlanResponse(&m))
}

/* =========================================================
   CLEANUP gambar lama yang sudah lewat retensi
   - Hapus object OSS untuk *_old jika now > delete_pending_until
   - Null-kan kolom *_old + *_delete_pending_until
========================================================= */

func (ctl *MasjidServicePlanController) CleanupExpiredOldImages(c *fiber.Ctx) error {
	now := time.Now()

	// Ambil kandidat (pakai lock for update untuk menghindari race di multi worker)
	var plans []spModel.MasjidServicePlan
	if err := ctl.DB.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("masjid_service_plan_image_url_old IS NOT NULL").
		Where("masjid_service_plan_image_object_key_old IS NOT NULL").
		Where("masjid_service_plan_image_delete_pending_until IS NOT NULL").
		Where("masjid_service_plan_image_delete_pending_until <= ?", now).
		Find(&plans).Error; err != nil {
		return httpErr(c, fiber.StatusBadGateway, "db error: "+err.Error())
	}

	if len(plans) == 0 {
		return c.JSON(fiber.Map{"deleted": 0})
	}

	oss, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		return httpErr(c, fiber.StatusBadGateway, "oss init: "+err.Error())
	}

	deleted := 0
	for i := range plans {
		m := &plans[i]
		if m.MasjidServicePlanImageURLOld == nil {
			continue
		}
		oldURL := *m.MasjidServicePlanImageURLOld
		if strings.TrimSpace(oldURL) == "" {
			continue
		}
		// hapus di OSS
		if err := oss.DeleteByPublicURL(c.Context(), oldURL); err != nil {
			// jika 404 / not found, kita tetap lanjut null-kan kolom; selain itu, log aja
			if !isOSSNotFound(err) {
				// log saja; jangan gagal total
				// c.App().Logger().Warnf("cleanup: delete %s: %v", oldURL, err)
			}
		}
		// null-kan kolom old
		m.MasjidServicePlanImageURLOld = nil
		m.MasjidServicePlanImageObjectKeyOld = nil
		m.MasjidServicePlanImageDeletePendingUntil = nil

		if err := ctl.DB.Model(m).Updates(map[string]any{
			"masjid_service_plan_image_url_old":              gorm.Expr("NULL"),
			"masjid_service_plan_image_object_key_old":       gorm.Expr("NULL"),
			"masjid_service_plan_image_delete_pending_until": gorm.Expr("NULL"),
			"masjid_service_plan_updated_at":                 time.Now(),
		}).Error; err != nil {
			// log lalu lanjut
			continue
		}
		deleted++
	}

	return c.JSON(fiber.Map{"deleted": deleted})
}

func isOSSNotFound(err error) bool {
	// mirror helper.isNotFound tapi tanpa ekspor; fallback string check
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "not found")
}
