// file: internals/features/school/posts/themes/controller/post_theme_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/school/others/post/dto"
	pmodel "schoolku_backend/internals/features/school/others/post/model" // samakan dengan import di DTO
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =========================================================
   Controller & Ctor
========================================================= */

type PostThemeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewPostThemeController(db *gorm.DB, v *validator.Validate) *PostThemeController {
	if v == nil {
		v = validator.New()
	}
	return &PostThemeController{DB: db, Validator: v}
}

/* =========================================================
   PG error mapper sederhana
========================================================= */

type pgSQLErr interface {
	SQLState() string
	Error() string
}

func writeDBErr(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	var pgErr pgSQLErr
	if errors.As(err, &pgErr) {
		switch pgErr.SQLState() {
		case "23505":
			return helper.JsonError(c, http.StatusConflict, "Duplikat data (unique violation).")
		case "23503":
			return helper.JsonError(c, http.StatusBadRequest, "Referensi tidak valid (foreign key).")
		}
	}
	// fallback
	return helper.JsonError(c, http.StatusInternalServerError, err.Error())
}

/* =========================================================
   CREATE (DKM/Admin atau Owner)
   POST /post-themes
========================================================= */

func (ctl *PostThemeController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve + authorize
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	mid, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req dto.CreatePostThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	// Force tenant dari context
	req.PostThemeSchoolID = mid

	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	m, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		return writeDBErr(c, err)
	}

	return helper.JsonCreated(c, "Post theme created", dto.FromModel(m))
}

/* =========================================================
   PATCH (DKM/Admin atau Owner)
   PATCH /post-themes/:id
========================================================= */

func (ctl *PostThemeController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	mid, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	var m pmodel.PostThemeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("post_theme_id = ? AND post_theme_school_id = ? AND post_theme_deleted_at IS NULL", id, mid).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Post theme tidak ditemukan")
		}
		return writeDBErr(c, err)
	}

	var req dto.PatchPostThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}

	if err := req.ApplyToModel(&m); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).
		Model(&pmodel.PostThemeModel{}).
		Where("post_theme_id = ?", m.PostThemeID).
		Select("*").
		Updates(&m).Error; err != nil {
		return writeDBErr(c, err)
	}

	// reload
	if err := ctl.DB.WithContext(c.Context()).
		First(&m, "post_theme_id = ?", m.PostThemeID).Error; err != nil {
		return writeDBErr(c, err)
	}

	return helper.JsonUpdated(c, "Post theme updated", dto.FromModel(&m))
}

/* =========================================================
   DELETE (soft) (DKM/Admin atau Owner)
   DELETE /post-themes/:id
========================================================= */

func (ctl *PostThemeController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	mid, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	// pastikan milik tenant & belum terhapus
	var existing pmodel.PostThemeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("post_theme_id = ? AND post_theme_school_id = ? AND post_theme_deleted_at IS NULL", id, mid).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Post theme tidak ditemukan")
		}
		return writeDBErr(c, err)
	}

	// soft delete via GORM
	if err := ctl.DB.WithContext(c.Context()).Delete(&existing).Error; err != nil {
		return writeDBErr(c, err)
	}

	return helper.JsonDeleted(c, "Post theme deleted", fiber.Map{
		"post_theme_id": existing.PostThemeID,
	})
}
