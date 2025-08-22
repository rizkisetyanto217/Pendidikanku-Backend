// file: internals/features/academics/terms/controller/academic_term_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/lembaga/academics/academic_year/dto"
	"masjidku_backend/internals/features/lembaga/academics/academic_year/model"
	helper "masjidku_backend/internals/helpers" // ⬅️ sesuaikan path jika beda
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



// -----------------------------
// Overlap checker
// -----------------------------
func (ctl *AcademicTermController) assertNoOverlap(ent *model.AcademicTermModel) error {
	var cnt int64
	if err := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", ent.AcademicTermsMasjidID).
		Where("academic_terms_academic_year = ?", ent.AcademicTermsAcademicYear).
		Where("academic_terms_id <> ?", ent.AcademicTermsID).
		// Tidak overlap jika end <= other.start ATAU start >= other.end
		// Jadi kita ambil negasinya (NOT (...)) untuk mendapat yang bertabrakan.
		Where("NOT (academic_terms_end_date <= ? OR academic_terms_start_date >= ?)",
			ent.AcademicTermsStartDate, ent.AcademicTermsEndDate).
		Count(&cnt).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Overlap check failed: "+err.Error())
	}
	if cnt > 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Periode bertabrakan dengan term lain pada tahun ajaran yang sama")
	}
	return nil
}

// -----------------------------
// Create
// -----------------------------
// file: internals/features/lembaga/academics/academic_year/controller/academic_year_controller.go
// -----------------------------
// Create (multi-active & overlap allowed)
// -----------------------------
func (ctl *AcademicTermController) Create(c *fiber.Ctx) error {
	// A) Deteksi apakah field is_active dikirim (agar false tidak "hilang")
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &raw); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Body must be valid JSON: "+err.Error())
	}
	var (
		isActiveProvided bool
		isActiveValue    bool
	)
	if v, ok := raw["academic_terms_is_active"]; ok {
		isActiveProvided = true
		if string(v) != "null" && len(v) > 0 {
			if err := json.Unmarshal(v, &isActiveValue); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "academic_terms_is_active must be boolean")
			}
		}
	}

	// B) Parse DTO
	var body dto.AcademicTermCreateDTO
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid body: "+err.Error())
	}
	body.Normalize()
	if err := ctl.Validator.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// C) Validasi tanggal minimal
	if !body.AcademicTermsEndDate.After(body.AcademicTermsStartDate) {
		return fiber.NewError(fiber.StatusBadRequest, "End date must be after start date")
	}

	// D) Bentuk entity & hormati nilai explicit is_active bila dikirim
	ent := body.ToModel(masjidID)
	if isActiveProvided {
		ent.AcademicTermsIsActive = isActiveValue
	}

	// E) Insert — paksa kolom boolean ikut dikirim walau false
	if err := ctl.DB.
		Session(&gorm.Session{}).
		Select(
			"AcademicTermsMasjidID",
			"AcademicTermsAcademicYear",
			"AcademicTermsName",
			"AcademicTermsStartDate",
			"AcademicTermsEndDate",
			"AcademicTermsIsActive",
		).
		Create(&ent).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Create failed: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(dto.FromModel(ent))
}


// -----------------------------
// Overlap checker (range half-open [start, end))
// -----------------------------


// -----------------------------
// Update (partial)
// -----------------------------
// -----------------------------
// Update (partial) — overlap & multi-active allowed
// -----------------------------
func (ctl *AcademicTermController) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// Parse body (pointer fields => bisa bedakan "tidak dikirim" vs set false)
	var body dto.AcademicTermUpdateDTO
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid body: "+err.Error())
	}
	if err := ctl.Validator.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	// Ambil entity
	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_id = ? AND academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", id, masjidID).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Record not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// Terapkan perubahan parsial
	body.ApplyUpdates(&ent)

	// Validasi tanggal minimal (overlap diperbolehkan, jadi cukup relasi start<end)
	if !ent.AcademicTermsEndDate.After(ent.AcademicTermsStartDate) {
		return fiber.NewError(fiber.StatusBadRequest, "End date must be after start date")
	}

	// (Overlap check DIHAPUS / diizinkan)
	// if err := ctl.assertNoOverlap(&ent); err != nil { ... }

	// Set updated_at (meskipun ada trigger, ini buat konsisten di response)
	now := time.Now()
	ent.AcademicTermsUpdatedAt = &now

	// Update — Select kolom agar boolean false tidak diabaikan
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
		return fiber.NewError(fiber.StatusInternalServerError, "Update failed: "+err.Error())
	}

	return c.JSON(dto.FromModel(ent))
}



// -----------------------------
// GetByID (scoped ke masjid di token)
// -----------------------------
func (ctl *AcademicTermController) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_id = ? AND academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", id, masjidID).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Record not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	return c.JSON(dto.FromModel(ent))
}

// -----------------------------
// List (multi-tenant via token) + Filter + Pagination + Sorting
// -----------------------------
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	var q dto.AcademicTermFilterDTO
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid query: "+err.Error())
	}
	if err := ctl.Validator.Struct(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	// Dapatkan SEMUA masjid yang boleh diakses dari token
	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	page := q.Page
	if page == 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize == 0 {
		pageSize = 20
	}

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_masjid_id IN (?) AND academic_terms_deleted_at IS NULL", masjidIDs)

	if q.Year != nil && strings.TrimSpace(*q.Year) != "" {
		dbq = dbq.Where("academic_terms_academic_year = ?", strings.TrimSpace(*q.Year))
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		dbq = dbq.Where("academic_terms_name = ?", strings.TrimSpace(*q.Name))
	}
	if q.Active != nil {
		dbq = dbq.Where("academic_terms_is_active = ?", *q.Active)
	}

	sortBy := "academic_terms_created_at"
	if q.SortBy != nil {
		switch *q.SortBy {
		case "created_at":
			sortBy = "academic_terms_created_at"
		case "updated_at":
			sortBy = "academic_terms_updated_at"
		case "start_date":
			sortBy = "academic_terms_start_date"
		case "end_date":
			sortBy = "academic_terms_end_date"
		case "name":
			sortBy = "academic_terms_name"
		case "year":
			sortBy = "academic_terms_academic_year"
		}
	}
	sortDir := "desc"
	if q.SortDir != nil && (*q.SortDir == "asc" || *q.SortDir == "desc") {
		sortDir = *q.SortDir
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	var list []model.AcademicTermModel
	if err := dbq.
		Order(sortBy + " " + sortDir).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	resp := struct {
		Data       []dto.AcademicTermResponseDTO `json:"data"`
		Pagination struct {
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
		} `json:"pagination"`
	}{
		Data: dto.FromModels(list),
	}
	resp.Pagination.Total = total
	resp.Pagination.Page = page
	resp.Pagination.PageSize = pageSize

	return c.JSON(resp)
}


// -----------------------------
// SoftDelete (set inactive + deleted_at)
// -----------------------------
func (ctl *AcademicTermController) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_id = ? AND academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", id, masjidID).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Record not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	now := time.Now()
	ent.AcademicTermsIsActive = false          // <- pastikan jadi nonaktif
	ent.AcademicTermsDeletedAt = &now          // <- soft delete
	ent.AcademicTermsUpdatedAt = &now          // <- update timestamp

	// Gunakan Select agar boolean false tidak di-skip oleh GORM
	if err := ctl.DB.
		Model(&ent).
		Select("AcademicTermsIsActive", "AcademicTermsDeletedAt", "AcademicTermsUpdatedAt").
		Updates(&ent).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Delete failed: "+err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}
