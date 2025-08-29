// file: internals/features/academics/terms/controller/academic_term_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/academics/academic_terms/dto"
	model "masjidku_backend/internals/features/school/academics/academic_terms/model"
	helper "masjidku_backend/internals/helpers"
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

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
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

	// Create (CREATE tidak mengabaikan zero value, jadi aman untuk boolean false)
	if err := ctl.DB.Create(&ent).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Create failed: "+err.Error())
	}

	return helper.JsonCreated(c, "Academic term created successfully", dto.FromModel(ent))
}

/* -----------------------------
 * SEARCH (by year saja)
 * GET /academics/terms/search?year=2026&pagesize=20&page=1
 * ----------------------------- */

func (ctl *AcademicTermController) SearchOnlyByYear(c *fiber.Ctx) error {
	yearQ := strings.TrimSpace(c.Query("year"))
	if yearQ == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Query param 'year' wajib diisi")
	}

	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// paginasi sederhana
	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 200 {
			pageSize = n
		}
	}

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_masjid_id IN (?) AND academic_terms_deleted_at IS NULL", masjidIDs).
		Where("academic_terms_academic_year ILIKE ?", "%"+yearQ+"%")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	var list []model.AcademicTermModel
	if err := dbq.
		Order("academic_terms_start_date DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	resp := struct {
		Data       []dto.AcademicTermResponseDTO `json:"data"`
		Pagination struct {
			Total    int64  `json:"total"`
			Page     int    `json:"page"`
			PageSize int    `json:"page_size"`
			Query    string `json:"query"`
		} `json:"pagination"`
	}{
		Data: dto.FromModels(list),
	}
	resp.Pagination.Total = total
	resp.Pagination.Page = page
	resp.Pagination.PageSize = pageSize
	resp.Pagination.Query = yearQ

	return c.JSON(resp)
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

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
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

	// Set updated_at (model kita pakai non-pointer time.Time)
	ent.AcademicTermsUpdatedAt = time.Now()

	// Update — pakai Select agar kolom boolean false tidak diabaikan
	if err := ctl.DB.
		Model(&ent).
		Select(
			"AcademicTermsAcademicYear",
			"AcademicTermsName",
			"AcademicTermsStartDate",
			"AcademicTermsEndDate",
			"AcademicTermsIsActive",
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

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
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
	// Gunakan Updates(map) supaya kita bisa set kolom secara eksplisit (termasuk boolean false)
	if err := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_id = ?", ent.AcademicTermsID).
		Updates(map[string]any{
			"academic_terms_is_active":  false,
			"academic_terms_deleted_at": now,
			"academic_terms_updated_at": now,
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Delete failed: "+err.Error())
	}

	// Refetch untuk response konsisten (mapper akan konversi gorm.DeletedAt → *time.Time)
	if err := ctl.DB.First(&ent, "academic_terms_id = ?", ent.AcademicTermsID).Error; err != nil {
		// kalau gagal refetch, tetap kirim minimal payload
		ent.AcademicTermsIsActive = false
		ent.AcademicTermsUpdatedAt = now
	}

	return helper.JsonDeleted(c, "Academic term deleted (soft) successfully", dto.FromModel(ent))
}
