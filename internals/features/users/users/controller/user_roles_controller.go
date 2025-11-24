// file: internals/features/authz/controller/user_role_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	pq "github.com/lib/pq"
	"gorm.io/gorm"

	"madinahsalam_backend/internals/features/users/users/dto"
	"madinahsalam_backend/internals/features/users/users/model"
	helper "madinahsalam_backend/internals/helpers"
)

type UserRoleController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewUserRoleController(db *gorm.DB, validate *validator.Validate) *UserRoleController {
	if validate == nil {
		validate = validator.New()
	}
	return &UserRoleController{DB: db, Validate: validate}
}

// =====================================================
// Helpers
// =====================================================

func isUnique(err error) bool {
	var e *pq.Error
	if errors.As(err, &e) {
		return e.Code == "23505"
	}
	// fallback string check
	lo := strings.ToLower(err.Error())
	return strings.Contains(lo, "duplicate") || strings.Contains(lo, "unique")
}

func clamp(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// =====================================================
// CREATE: POST /authz/user-roles
// =====================================================

func (ctl *UserRoleController) Create(c *fiber.Ctx) error {
	var req dto.CreateUserRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()
	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		if isUnique(err) {
			return helper.JsonError(c, fiber.StatusConflict, "User role sudah ada (alive)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat user role")
	}

	return helper.JsonCreated(c, "User role dibuat", dto.FromModelUserRole(m))
}

// =====================================================
// LIST: GET /authz/user-roles?user_id=&role_id=&school_id=&only_alive=&limit=&offset=&order_by=&sort=
// Note: school_id= null  (string literal "null") untuk filter global
// =====================================================
func (ctl *UserRoleController) List(c *fiber.Ctx) error {
	var q dto.ListUserRoleQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ===== defaults
	if q.OnlyAlive == nil {
		t := true
		q.OnlyAlive = &t
	}

	// ===== order by + sort
	orderBy := "assigned_at"
	switch strings.ToLower(strings.TrimSpace(q.OrderBy)) {
	case "user_id":
		orderBy = "user_id"
	case "role_id":
		orderBy = "role_id"
	case "assigned_at", "":
		orderBy = "assigned_at"
	}
	sortDir := "DESC"
	if strings.EqualFold(strings.TrimSpace(q.Sort), "asc") {
		sortDir = "ASC"
	}

	// ===== base query
	tx := ctl.DB.WithContext(c.Context()).Model(&model.UserRole{})
	if q.OnlyAlive != nil && *q.OnlyAlive {
		tx = tx.Where("deleted_at IS NULL")
	}
	if q.UserID != nil {
		tx = tx.Where("user_id = ?", *q.UserID)
	}
	if q.RoleID != nil {
		tx = tx.Where("role_id = ?", *q.RoleID)
	}

	// ===== school_id filter: "null" → IS NULL, UUID valid → = ?, kosong → abaikan
	rawMid := strings.TrimSpace(c.Query("school_id"))
	if strings.EqualFold(rawMid, "null") {
		tx = tx.Where("school_id IS NULL")
	} else if rawMid != "" {
		if mid, err := uuid.Parse(rawMid); err == nil {
			tx = tx.Where("school_id = ?", mid)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak valid")
		}
	}

	// ===== total (sebelum limit/offset)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// ===== paging helper (+ dukung ?all=1)
	all := parseBool(c.Query("all"))
	pg := helper.ResolvePaging(c, 20, 200) // default per_page=20, max=200

	// ===== query dengan sort + paging
	qx := tx.Order(fmt.Sprintf("%s %s", orderBy, sortDir))
	if !all {
		qx = qx.Offset(pg.Offset).Limit(pg.Limit)
	}

	var rows []model.UserRole
	if err := qx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// ===== map ke DTO
	resp := make([]dto.UserRoleResponse, 0, len(rows))
	for _, m := range rows {
		resp = append(resp, dto.FromModelUserRole(m))
	}

	// ===== build pagination object untuk JsonList
	var pagination helper.Pagination
	if all {
		per := int(total)
		if per == 0 {
			per = 1
		}
		pagination = helper.BuildPaginationFromPage(total, 1, per)
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	return helper.JsonList(c, "OK", resp, pagination)
}

// util kecil (boleh dipindah ke util umummu)
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// =====================================================
// UPDATE (PATCH): PATCH /authz/user-roles/:id
// =====================================================

func (ctl *UserRoleController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateUserRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// validasi ringan (opsional)
	if req.SchoolID != nil && *req.SchoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak boleh UUID nil (pakai clear_school_id)")
	}

	var m model.UserRole
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_role_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// apply partial
	req.Apply(&m)

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		if isUnique(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kombinasi user/role/school sudah ada (alive)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "User role diperbarui", dto.FromModelUserRole(m))
}

// =====================================================
// DELETE (soft/hard):
//   DELETE /authz/user-roles/:id
//   ?hard=true untuk hard delete
// =====================================================

func (ctl *UserRoleController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	hard := strings.EqualFold(strings.TrimSpace(c.Query("hard")), "true")

	var m model.UserRole
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_role_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	if hard {
		if err := ctl.DB.WithContext(c.Context()).Unscoped().Delete(&m).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus (hard)")
		}
		return helper.JsonDeleted(c, "User role dihapus permanen", fiber.Map{"user_role_id": id})
	}

	now := time.Now()
	m.DeletedAt = &now
	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus (soft)")
	}
	return helper.JsonDeleted(c, "User role dihapus", fiber.Map{"user_role_id": id})
}

// =====================================================
// RESTORE (optional): POST /authz/user-roles/:id/restore
// =====================================================

func (ctl *UserRoleController) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UserRole
	if err := ctl.DB.WithContext(c.Context()).
		Unscoped().
		Where("user_role_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	m.DeletedAt = nil
	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		if isUnique(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kombinasi user/role/school sudah aktif")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal me-restore data")
	}

	return helper.JsonUpdated(c, "User role direstore", dto.FromModelUserRole(m))
}
