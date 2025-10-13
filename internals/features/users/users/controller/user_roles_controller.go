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

	"masjidku_backend/internals/features/users/users/dto"
	"masjidku_backend/internals/features/users/users/model"
	helper "masjidku_backend/internals/helpers"
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
// LIST: GET /authz/user-roles?user_id=&role_id=&masjid_id=&only_alive=&limit=&offset=&order_by=&sort=
// Note: masjid_id= null  (string literal "null") untuk filter global
// =====================================================

func (ctl *UserRoleController) List(c *fiber.Ctx) error {
	var q dto.ListUserRoleQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// Defaults
	if q.OnlyAlive == nil {
		t := true
		q.OnlyAlive = &t
	}
	q.Limit = clamp(q.Limit, 20, 200)
	if q.Offset < 0 {
		q.Offset = 0
	}
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

	// MasjidID filter:
	// - jika query masjid_id literal "null" → IS NULL
	// - jika UUID valid → = ?
	// - jika kosong → abaikan (semua)
	rawMid := strings.TrimSpace(c.Query("masjid_id"))
	if strings.EqualFold(rawMid, "null") {
		tx = tx.Where("masjid_id IS NULL")
	} else if rawMid != "" {
		if mid, err := uuid.Parse(rawMid); err == nil {
			tx = tx.Where("masjid_id = ?", mid)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id tidak valid")
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	var rows []model.UserRole
	if err := tx.
		Order(fmt.Sprintf("%s %s", orderBy, sortDir)).
		Limit(q.Limit).Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resp := make([]dto.UserRoleResponse, 0, len(rows))
	for _, m := range rows {
		resp = append(resp, dto.FromModelUserRole(m))
	}

	meta := dto.Pagination{
		Total:      int(total),
		Limit:      q.Limit,
		Offset:     q.Offset,
		Returned:   len(resp),
		NextOffset: q.Offset + q.Limit,
		PrevOffset: func() int { if q.Offset-q.Limit < 0 { return 0 }; return q.Offset - q.Limit }(),
	}
	return helper.JsonList(c, resp, meta)
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
	if req.MasjidID != nil && *req.MasjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id tidak boleh UUID nil (pakai clear_masjid_id)")
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
			return helper.JsonError(c, fiber.StatusConflict, "Kombinasi user/role/masjid sudah ada (alive)")
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
			return helper.JsonError(c, fiber.StatusConflict, "Kombinasi user/role/masjid sudah aktif")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal me-restore data")
	}

	return helper.JsonUpdated(c, "User role direstore", dto.FromModelUserRole(m))
}