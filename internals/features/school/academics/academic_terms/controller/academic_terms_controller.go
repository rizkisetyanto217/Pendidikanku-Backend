// file: internals/features/academics/terms/controller/academic_term_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/academics/academic_terms/dto"
	model "masjidku_backend/internals/features/school/academics/academic_terms/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type AcademicTermController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAcademicTermController(db *gorm.DB) *AcademicTermController {
	return &AcademicTermController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* -----------------------------
 * CREATE
 * ----------------------------- */

func (ctl *AcademicTermController) Create(c *fiber.Ctx) error {
	// Deteksi apakah field is_active dikirim (supaya false tidak "hilang")
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &raw); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Body must be valid JSON: "+err.Error())
	}
	var (
		isActiveProvided bool
		isActiveValue    bool
	)
	if v, ok := raw["academic_terms_is_active"]; ok {
		isActiveProvided = true
		if string(v) != "null" && len(v) > 0 {
			if err := json.Unmarshal(v, &isActiveValue); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "academic_terms_is_active must be boolean")
			}
		}
	}

	// Parse & validate DTO
	var body dto.AcademicTermCreateDTO
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid body: "+err.Error())
	}
	body.Normalize()
	if err := ctl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Validasi tanggal minimal
	if !body.AcademicTermsEndDate.After(body.AcademicTermsStartDate) {
		return helper.JsonError(c, fiber.StatusBadRequest, "End date must be after start date")
	}

	// Bentuk entity & hormati nilai explicit is_active bila dikirim
	ent := body.ToModel(masjidID)
	if isActiveProvided {
		ent.AcademicTermsIsActive = isActiveValue
	}

	// Create
	if err := ctl.DB.Create(&ent).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Create failed: "+err.Error())
	}

	return helper.JsonCreated(c, "Academic term created successfully", dto.FromModel(ent))
}

/* -----------------------------
 * SEARCH (by year saja + optional angkatan)
 * GET /academics/terms/search?year=2026&angkatan=10&page=1&page_size=20
 * ----------------------------- */

// GET /academics/terms/search?year=2026&per_page=20&page=1&sort_by=start_date&sort=desc
func (ctl *AcademicTermController) SearchOnlyByYear(c *fiber.Ctx) error {
    yearQ := strings.TrimSpace(c.Query("year"))
    if yearQ == "" {
        return fiber.NewError(fiber.StatusBadRequest, "Query param 'year' wajib diisi")
    }

    masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
    if err != nil {
        return err
    }

    // ==== Pagination (helper) ====
    rawQ := string(c.Request().URI().QueryString())
    httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
    p := helper.ParseWith(httpReq, "start_date", "desc", helper.DefaultOpts)

    // Kolom sort yang diizinkan
    allowedSort := map[string]string{
        "start_date": "academic_terms_start_date",
        "end_date":   "academic_terms_end_date",
        "created_at": "academic_terms_created_at",
        "updated_at": "academic_terms_updated_at",
        "name":       "academic_terms_name",
        "year":       "academic_terms_academic_year",
        "angkatan":   "academic_terms_angkatan",
    }
    orderClause, err := p.SafeOrderClause(allowedSort, "start_date")
    if err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
    }
    orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

    // ==== Query ====
    dbq := ctl.DB.Model(&model.AcademicTermModel{}).
        Where("academic_terms_masjid_id IN (?) AND academic_terms_deleted_at IS NULL", masjidIDs).
        Where("academic_terms_academic_year ILIKE ?", "%"+yearQ+"%")

    var total int64
    if err := dbq.Count(&total).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Count failed: "+err.Error())
    }

    var list []model.AcademicTermModel
    if err := dbq.
        Order(orderExpr).
        Limit(p.Limit()).
        Offset(p.Offset()).
        Find(&list).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
    }

    meta := helper.BuildMeta(total, p)
    return helper.JsonList(c, dto.FromModels(list), meta)
}




/* -----------------------------
 * UPDATE (partial)
 * ----------------------------- */

func (ctl *AcademicTermController) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var body dto.AcademicTermUpdateDTO
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid body: "+err.Error())
	}
	if err := ctl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_id = ? AND academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", id, masjidID).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Record not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// Apply perubahan parsial
	body.ApplyUpdates(&ent)

	// Validasi tanggal minimal
	if !ent.AcademicTermsEndDate.After(ent.AcademicTermsStartDate) {
		return helper.JsonError(c, fiber.StatusBadRequest, "End date must be after start date")
	}

	// Set updated_at
	ent.AcademicTermsUpdatedAt = time.Now()

	// Update — pakai Select agar kolom boolean & integer 0 tidak diabaikan
	if err := ctl.DB.
		Model(&ent).
		Select(
			"AcademicTermsAcademicYear",
			"AcademicTermsName",
			"AcademicTermsStartDate",
			"AcademicTermsEndDate",
			"AcademicTermsIsActive",
			"AcademicTermsAngkatan",   // <-- kolom baru
			"AcademicTermsUpdatedAt",
		).
		Updates(&ent).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Update failed: "+err.Error())
	}

	return helper.JsonUpdated(c, "Academic term updated successfully", dto.FromModel(ent))
}

/* -----------------------------
 * SOFT DELETE (set inactive + deleted_at)
 * ----------------------------- */

func (ctl *AcademicTermController) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Pastikan record milik tenant & belum terhapus
	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_id = ? AND academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", id, masjidID).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Record not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	now := time.Now()
	// Gunakan Updates(map) supaya eksplisit
	if err := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_id = ?", ent.AcademicTermsID).
		Updates(map[string]any{
			"academic_terms_is_active":  false,
			"academic_terms_deleted_at": now,
			"academic_terms_updated_at": now,
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Delete failed: "+err.Error())
	}

	// Refetch untuk response konsisten
	if err := ctl.DB.First(&ent, "academic_terms_id = ?", ent.AcademicTermsID).Error; err != nil {
		ent.AcademicTermsIsActive = false
		ent.AcademicTermsUpdatedAt = now
	}

	return helper.JsonDeleted(c, "Academic term deleted (soft) successfully", dto.FromModel(ent))
}
