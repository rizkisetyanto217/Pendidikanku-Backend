// file: internals/features/school/events/themes/controller/class_event_theme_controller.go
package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	dto "masjidku_backend/internals/features/school/others/events/dto"
	model "masjidku_backend/internals/features/school/others/events/model"
)

/* =========================
   Controller
   ========================= */

type ClassEventThemeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewClassEventThemeController(db *gorm.DB) *ClassEventThemeController {
	return &ClassEventThemeController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* =========================
   Small helpers
   ========================= */

func (ctl *ClassEventThemeController) getID(c *fiber.Ctx) (uuid.UUID, error) {
	param := strings.TrimSpace(c.Params("id"))
	if param == "" {
		return uuid.Nil, errors.New("missing id")
	}
	id, err := uuid.Parse(param)
	if err != nil {
		return uuid.Nil, errors.New("invalid id")
	}
	return id, nil
}

func queryIntDefault(c *fiber.Ctx, key string, def int) int {
	v := strings.TrimSpace(c.Query(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// isDuplicateKey: cek pelanggaran unique Postgres (SQLSTATE 23505).
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}

// resolveMasjidAndEnsureDKM: gunakan helper resolver & guard baru, balas JSON konsisten
func (ctl *ClassEventThemeController) resolveMasjidAndEnsureDKM(c *fiber.Ctx) (uuid.UUID, error) {
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			_ = helper.JsonError(c, fe.Code, fe.Message)
			return uuid.Nil, err
		}
		_ = helper.JsonError(c, http.StatusBadRequest, err.Error())
		return uuid.Nil, err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			_ = helper.JsonError(c, fe.Code, fe.Message)
			return uuid.Nil, err
		}
		_ = helper.JsonError(c, http.StatusForbidden, err.Error())
		return uuid.Nil, err
	}
	return masjidID, nil
}

/*
=========================================================

	LIST
	GET /api/a/:masjid_id/events/themes
	Query: q, is_active, limit, offset, order_by
	=========================================================
*/
func (ctl *ClassEventThemeController) List(c *fiber.Ctx) error {
	masjidID, err := ctl.resolveMasjidAndEnsureDKM(c)
	if err != nil {
		return nil // response sudah dikirim di helper
	}

	// Parse query ke DTO
	var q dto.ListClassEventThemeQuery
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		q.SearchName = &[]string{v}[0]
	}
	if v := strings.ToLower(strings.TrimSpace(c.Query("is_active"))); v == "true" || v == "false" {
		b := v == "true"
		q.IsActive = &b
	}
	q.OrderBy = c.Query("order_by")
	q.Limit = queryIntDefault(c, "limit", 20)
	q.Offset = queryIntDefault(c, "offset", 0)
	q.Normalize()

	tx := ctl.DB.
		Model(&model.ClassEventTheme{}).
		Where("class_event_themes_masjid_id = ? AND class_event_themes_deleted_at IS NULL", masjidID)

	if q.IsActive != nil {
		tx = tx.Where("class_event_themes_is_active = ?", *q.IsActive)
	}
	if q.SearchName != nil {
		tx = tx.Where("class_event_themes_name ILIKE ?", "%"+*q.SearchName+"%")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	var rows []model.ClassEventTheme
	if err := tx.
		Order(q.OrderExpr()).
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonList(c, dto.FromModels(rows), fiber.Map{
		"limit":  q.Limit,
		"offset": q.Offset,
		"total":  total,
	})
}

/*
=========================================================

	CREATE
	POST /api/a/:masjid_id/events/themes
	Body: JSON CreateClassEventThemeRequest
	=========================================================
*/
func (ctl *ClassEventThemeController) Create(c *fiber.Ctx) error {
	masjidID, err := ctl.resolveMasjidAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	var req dto.CreateClassEventThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	req.Normalize()
	if err := req.Validate(ctl.Validator); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	m := req.ToModel(masjidID)

	if err := ctl.DB.Create(m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, http.StatusConflict, "theme code already exists for this masjid")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Created", dto.FromModel(m))
}

/*
=========================================================

	PATCH
	PATCH /api/a/:masjid_id/events/themes/:id
	Body: JSON PatchClassEventThemeRequest (tri-state)
	=========================================================
*/
func (ctl *ClassEventThemeController) Patch(c *fiber.Ctx) error {
	masjidID, err := ctl.resolveMasjidAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	id, err := ctl.getID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var req dto.PatchClassEventThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	req.Normalize()
	if err := req.ValidatePartial(ctl.Validator); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var m model.ClassEventTheme
	if err := ctl.DB.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_event_themes_id = ? AND class_event_themes_masjid_id = ? AND class_event_themes_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "resource not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	req.ApplyPatch(&m)

	if err := ctl.DB.Save(&m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, http.StatusConflict, "theme code already exists for this masjid")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Updated", dto.FromModel(&m))
}

/*
=========================================================

	DELETE (soft)
	DELETE /api/a/:masjid_id/events/themes/:id
	=========================================================
*/
func (ctl *ClassEventThemeController) Delete(c *fiber.Ctx) error {
	masjidID, err := ctl.resolveMasjidAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	id, err := ctl.getID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var m model.ClassEventTheme
	if err := ctl.DB.
		Where("class_event_themes_id = ? AND class_event_themes_masjid_id = ? AND class_event_themes_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "resource not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Deleted", fiber.Map{
		"class_event_themes_id": id,
	})
}

/*
=========================================================

	OPTIONAL: Upsert (by masjid_id, code)
	POST /api/a/:masjid_id/events/themes:upsert
	Body: CreateClassEventThemeRequest
	=========================================================
*/
func (ctl *ClassEventThemeController) Upsert(c *fiber.Ctx) error {
	masjidID, err := ctl.resolveMasjidAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	var req dto.CreateClassEventThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	req.Normalize()
	if err := req.Validate(ctl.Validator); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	now := time.Now()

	m := req.ToModel(masjidID)
	m.ClassEventThemesUpdatedAt = now

	if err := ctl.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "class_event_themes_masjid_id"},
			{Name: "class_event_themes_code"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"class_event_themes_name":         m.ClassEventThemesName,
			"class_event_themes_color":        m.ClassEventThemesColor,
			"class_event_themes_custom_color": m.ClassEventThemesCustomColor,
			"class_event_themes_is_active":    m.ClassEventThemesIsActive,
			"class_event_themes_updated_at":   now,
		}),
	}).Create(m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	var out model.ClassEventTheme
	if err := ctl.DB.
		Where("class_event_themes_masjid_id = ? AND class_event_themes_code = ? AND class_event_themes_deleted_at IS NULL",
			masjidID, m.ClassEventThemesCode).
		First(&out).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "Upserted", dto.FromModel(&out))
}
