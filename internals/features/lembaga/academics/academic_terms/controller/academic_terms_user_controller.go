package controller

import (
	"errors"
	"masjidku_backend/internals/features/lembaga/academics/academic_terms/dto"
	"masjidku_backend/internals/features/lembaga/academics/academic_terms/model"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
)

// -----------------------------
// GetByID (scoped ke masjid di token)
// -----------------------------
// -----------------------------
// GetByID (scoped ke masjid di token)
// -----------------------------
func (ctl *AcademicTermController) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
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

	return helper.JsonOK(c, "Academic term fetched successfully", dto.FromModel(ent))
}


// -----------------------------
// List (multi-tenant via token) + Filter + Pagination + Sorting
// -----------------------------
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	var q dto.AcademicTermFilterDTO
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid query: "+err.Error())
	}
	if err := ctl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	// Dapatkan SEMUA masjid yang boleh diakses dari token
	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
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
		return helper.JsonError(c, fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	var list []model.AcademicTermModel
	if err := dbq.
		Order(sortBy + " " + sortDir).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// Gunakan helper.JsonList untuk response list + pagination
	pagination := fiber.Map{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	return helper.JsonList(c, dto.FromModels(list), pagination)
}



// GET /academics/terms/search?year=2026&pagesize=20&page=1
func (ctl *AcademicTermController) SearchByYear(c *fiber.Ctx) error {
	yearQ := strings.TrimSpace(c.Query("year"))
	if yearQ == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Query param 'year' wajib diisi")
	}

	// multi-tenant via token
	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// paginasi sederhana (fallback default)
	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 { page = n }
	}
	if v := c.Query("page_size"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 200 { pageSize = n }
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
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
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
