// file: internals/features/lembaga/masjids/controller/masjid_url_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	d "masjidku_backend/internals/features/lembaga/masjids/dto"
	m "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
)

/* =========================================
   CONTROLLER & CONSTRUCTOR
========================================= */

type MasjidURLController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

// allowed types; kalau kamu punya d.MasjidURLTypeOneOf, boleh ganti pakai itu
const oneOfTypes = "oneof=logo stempel ttd_ketua banner profile_cover gallery qr other bg_behind_main main linktree_bg"

func NewMasjidURLController(db *gorm.DB, v *validator.Validate) *MasjidURLController {
	if v == nil {
		v = validator.New()
	}
	return &MasjidURLController{DB: db, Validate: v}
}

/* =========================================
   HELPERS
========================================= */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	if idStr == "" {
		return uuid.Nil, errors.New(name + " is required")
	}
	return uuid.Parse(idStr)
}

// Hanya dipakai create JSON (bukan public list). Untuk list, masjid_id selalu dari query.
func resolveMasjidID(c *fiber.Ctx, given *uuid.UUID) (uuid.UUID, error) {
	if given != nil {
		return *given, nil
	}
	if v := c.Locals("masjid_id"); v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return uuid.Parse(s)
		}
	}
	return uuid.Nil, errors.New("masjid_id is required")
}


/* =========================================
   HANDLERS
========================================= */

// POST /masjid-urls
// - multipart: upload file ke OSS, generate public URL
// - json: pakai file_url langsung
func (h *MasjidURLController) Create(c *fiber.Ctx) error {
	ctype := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ====== MULTIPART ======
	if strings.HasPrefix(ctype, "multipart/form-data") {
		typeStr := strings.TrimSpace(c.FormValue("type"))
		if err := h.Validate.Var(typeStr, "required,"+d.MasjidURLTypeOneOf); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "type invalid")
		}

		// ambil masjid_id dari form atau dari Locals
		var finalMasjidID uuid.UUID
		if s := strings.TrimSpace(c.FormValue("masjid_id")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id must be uuid")
			}
			finalMasjidID = id
		} else {
			id, err := resolveMasjidID(c, nil)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
			finalMasjidID = id
		}

		// ambil file dari beberapa kemungkinan field
		fh, err := helperOSS.GetImageFile(c)
		if err != nil || fh == nil {
			// fallback ke "file" untuk kompat lama
			if fh2, err2 := c.FormFile("masjid_url_file_url"); err2 == nil && fh2 != nil {
				fh = fh2
			}
		}
		if fh == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "file is required")
		}

		// parse bool opsional
		parseBoolPtr := func(name string) (*bool, error) {
			v := strings.TrimSpace(c.FormValue(name))
			if v == "" {
				return nil, nil
			}
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("%s must be boolean", name)
			}
			return &b, nil
		}
		isPrimaryPtr, err := parseBoolPtr("is_primary")
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
		isActivePtr, err := parseBoolPtr("is_active")
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		// init OSS dari ENV
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "OSS init: "+err.Error())
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// upload & convert ke webp (helper.UploadImageToOSS)
		publicURL, err := helperOSS.UploadImageToOSS(ctx, svc, finalMasjidID, typeStr, fh)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		isPrimary := false
		if isPrimaryPtr != nil {
			isPrimary = *isPrimaryPtr
		}
		isActive := true
		if isActivePtr != nil {
			isActive = *isActivePtr
		}

		row := &m.MasjidURL{
			MasjidURLMasjidID:  finalMasjidID,
			MasjidURLType:      m.MasjidURLType(typeStr),
			MasjidURLFileURL:   publicURL,
			MasjidURLIsPrimary: isPrimary,
			MasjidURLIsActive:  isActive,
		}
		if err := h.DB.Create(row).Error; err != nil {
			return helper.JsonError(c, fiber.StatusConflict, err.Error())
		}
		return helper.JsonCreated(c, "created", d.ToMasjidURLResponse(row))
	}

	// ====== JSON ======
	var req d.CreateMasjidURLRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid payload")
	}
	if err := h.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	finalMasjidID, err := resolveMasjidID(c, req.MasjidID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	row := req.ToModel(finalMasjidID)
	if err := h.DB.Create(row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusConflict, err.Error())
	}
	return helper.JsonCreated(c, "created", d.ToMasjidURLResponse(row))
}

// GET /masjid-urls/:id
func (h *MasjidURLController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	var row m.MasjidURL
	if err := h.DB.First(&row, "masjid_url_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "ok", d.ToMasjidURLResponse(&row))
}

// GET /masjid-urls/list?masjid_id=...&type=...&only_active=true&only_primary=false&page=1&per_page=20
// PUBLIC-FRIENDLY: tidak akses Locals/guard; hanya pakai querystring.
func (h *MasjidURLController) List(c *fiber.Ctx) error {
	var q d.ListMasjidURLQuery

	masjidIDStr := strings.TrimSpace(c.Query("masjid_id"))
	if masjidIDStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id is required")
	}
	mID, err := uuid.Parse(masjidIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id must be uuid")
	}
	q.MasjidID = mID

	if t := strings.TrimSpace(c.Query("type")); t != "" {
		q.Type = &t
	}
	if v := strings.TrimSpace(c.Query("only_active")); v != "" {
		b, _ := strconv.ParseBool(v)
		q.OnlyActive = &b
	}
	if v := strings.TrimSpace(c.Query("only_primary")); v != "" {
		b, _ := strconv.ParseBool(v)
		q.OnlyPrimary = &b
	}
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			q.Page = i
		}
	}
	if v := strings.TrimSpace(c.Query("per_page")); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			q.PerPage = i
		}
	}

	// aman walau ada perubahan dto: fallback default kalau Normalize() tidak ada/berbeda
	if h.Validate != nil {
		if err := h.Validate.Struct(&q); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}
	// fallback normalize
	// after
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PerPage <= 0 {
		q.PerPage = 20
	}
	if q.PerPage < 1 {
		q.PerPage = 1
	}
	if q.PerPage > 100 {
		q.PerPage = 100
	}


	tx := h.DB.Model(&m.MasjidURL{}).Where("masjid_url_masjid_id = ?", q.MasjidID)
	if q.Type != nil && *q.Type != "" {
		tx = tx.Where("masjid_url_type = ?", *q.Type)
	}
	if q.OnlyActive != nil && *q.OnlyActive {
		tx = tx.Where("masjid_url_is_active = TRUE")
	}
	if q.OnlyPrimary != nil && *q.OnlyPrimary {
		tx = tx.Where("masjid_url_is_primary = TRUE")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	offset := (q.Page - 1) * q.PerPage
	var rows []m.MasjidURL
	if err := tx.Order("masjid_url_created_at DESC").
		Limit(q.PerPage).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	pg := d.PageMeta{Page: q.Page, PerPage: q.PerPage, Total: int(total)}
	return helper.JsonList(c, d.ToMasjidURLResponseSlice(rows), pg)
}

// PATCH /masjid-urls/:id
func (h *MasjidURLController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var row m.MasjidURL
	if err := h.DB.First(&row, "masjid_url_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	oldURL := row.MasjidURLFileURL

	ctype := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	if strings.HasPrefix(ctype, "multipart/form-data") {
		// type (opsional)
		if typeStr := strings.TrimSpace(c.FormValue("type")); typeStr != "" {
			if err := h.Validate.Var(typeStr, oneOfTypes); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "type invalid")
			}
			row.MasjidURLType = m.MasjidURLType(typeStr)
		}

		// booleans (opsional)
		parseBoolPtr := func(name string) (*bool, error) {
			v := strings.TrimSpace(c.FormValue(name))
			if v == "" {
				return nil, nil
			}
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("%s must be boolean", name)
			}
			return &b, nil
		}
		if b, err := parseBoolPtr("is_primary"); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		} else if b != nil {
			row.MasjidURLIsPrimary = *b
		}
		if b, err := parseBoolPtr("is_active"); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		} else if b != nil {
			row.MasjidURLIsActive = *b
		}

		// file (opsional)
		fh, _ := c.FormFile("masjid_url_file_url")
		var newURL string
		if fh != nil {
			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, "OSS init: "+err.Error())
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			slot := string(row.MasjidURLType)
			if typeStr := strings.TrimSpace(c.FormValue("type")); typeStr != "" {
				slot = typeStr
			}
			publicURL, upErr := helperOSS.UploadImageToOSS(ctx, svc, row.MasjidURLMasjidID, slot, fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, upErr.Error())
			}
			newURL = publicURL
			row.MasjidURLFileURL = publicURL

			if err := h.DB.Save(&row).Error; err != nil {
				_ = helperOSS.DeleteByPublicURLENV(publicURL, 15*time.Second) // rollback
				return helper.JsonError(c, fiber.StatusConflict, err.Error())
			}

			// move lama ke spam (best-effort)
			if strings.TrimSpace(oldURL) != "" && oldURL != newURL {
				_, _ = helperOSS.MoveToSpamByPublicURLENV(oldURL, 0)
			}
			return helper.JsonUpdated(c, "updated", d.ToMasjidURLResponse(&row))
		}

		// tidak ada file → save fields saja
		if err := h.DB.Save(&row).Error; err != nil {
			return helper.JsonError(c, fiber.StatusConflict, err.Error())
		}
		return helper.JsonUpdated(c, "updated", d.ToMasjidURLResponse(&row))
	}

	// ==== JSON PATCH (tanpa file) ====
	var req d.PatchMasjidURLRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid payload")
	}
	if h.Validate != nil {
		if err := h.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	req.Apply(&row)
	if err := h.DB.Save(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusConflict, err.Error())
	}
	return helper.JsonUpdated(c, "updated", d.ToMasjidURLResponse(&row))
}

// PATCH /masjid-urls/bulk
func (h *MasjidURLController) BulkPatch(c *fiber.Ctx) error {
	var req d.BulkPatchMasjidURLRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid payload")
	}
	if h.Validate != nil {
		if err := h.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for i := range req.Items {
			item := &req.Items[i]

			var row m.MasjidURL
			if err := tx.First(&row, "masjid_url_id = ?", item.ID).Error; err != nil {
				return err
			}
			item.Apply(&row)
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return helper.JsonError(c, fiber.StatusConflict, err.Error())
	}
	return helper.JsonOK(c, "bulk updated", fiber.Map{"updated": len(req.Items)})
}

// DELETE /masjid-urls/:id (soft delete + move to spam)
func (h *MasjidURLController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var row m.MasjidURL
	if err := h.DB.First(&row, "masjid_url_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// move file lama ke spam (best-effort)
	var spamURL string
	var ossWarn string
	if strings.TrimSpace(row.MasjidURLFileURL) != "" {
		if u, err := helperOSS.MoveToSpamByPublicURLENV(row.MasjidURLFileURL, 15*time.Second); err == nil {
			spamURL = u
		} else {
			ossWarn = err.Error()
		}
	}

	// soft delete DB
	if err := h.DB.Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp := fiber.Map{"masjid_url_id": id}
	if spamURL != "" {
		resp["moved_to_spam"] = spamURL
	}
	if ossWarn != "" {
		resp["oss_warning"] = ossWarn
	}
	return helper.JsonDeleted(c, "deleted", resp)
}

// Hard delete (admin-only)
func (h *MasjidURLController) HardDelete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var row m.MasjidURL
	if err := h.DB.Unscoped().First(&row, "masjid_url_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var ossWarn string
	if strings.TrimSpace(row.MasjidURLFileURL) != "" {
		if err := helperOSS.DeleteByPublicURLENV(row.MasjidURLFileURL, 15*time.Second); err != nil {
			ossWarn = err.Error()
		}
	}
	if err := h.DB.Unscoped().Delete(&m.MasjidURL{}, "masjid_url_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp := fiber.Map{"masjid_url_id": id}
	if ossWarn != "" {
		resp["oss_warning"] = ossWarn
	}
	return helper.JsonDeleted(c, "hard deleted", resp)
}
