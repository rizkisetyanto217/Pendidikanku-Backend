package controller

import (
	"strconv"
	"strings"

	userdto "madinahsalam_backend/internals/features/users/user_teachers/dto"
	"madinahsalam_backend/internals/features/users/user_teachers/model"
	helper "madinahsalam_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
)

// GET /api/user-teachers
// Query params:
//
//	q            : string (opsional) -> cari di user_teacher_name
//	page         : int (default 1)
//	per_page     : int (default 25, max 200)  -- sesuai DefaultOpts
//	sort_by      : "created_at" (default) | "name" | "completed" | "verified"
//	order        : "asc" | "desc" (default "desc")
//	is_active    : "true"/"false" (opsional, default: semua)
//	is_verified  : "true"/"false" (opsional)
//	is_completed : "true"/"false" (opsional)
//
//	(alias lama: limit/offset/sort juga masih didukung via ParseFiber)
func (uc *UserTeacherController) List(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))

	// Pakai preset default; kalau admin/ekspor tinggal ganti ke helper.AdminOpts / helper.ExportOpts
	params := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Whitelist kolom yang boleh disort
	allowed := map[string]string{
		"created_at": "user_teacher_created_at",
		"name":       "user_teacher_name",
		"completed":  "user_teacher_is_completed",
		"verified":   "user_teacher_is_verified",
	}

	orderClause, err := params.SafeOrderClause(allowed, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Sort tidak valid")
	}
	// SafeOrderClause menghasilkan "ORDER BY <col> <DIR>", sedangkan GORM.Order butuh tanpa prefix "ORDER BY "
	orderBy := strings.TrimPrefix(orderClause, "ORDER BY ")

	db := uc.DB.Model(&model.UserTeacherModel{})

	// ================== FILTER OPSIONAL: STATUS ==================

	// is_active
	if raw := strings.TrimSpace(c.Query("is_active")); raw != "" {
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter is_active harus boolean (true/false)")
		}
		db = db.Where("user_teacher_is_active = ?", val)
	}

	// is_verified
	if raw := strings.TrimSpace(c.Query("is_verified")); raw != "" {
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter is_verified harus boolean (true/false)")
		}
		db = db.Where("user_teacher_is_verified = ?", val)
	}

	// is_completed
	if raw := strings.TrimSpace(c.Query("is_completed")); raw != "" {
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter is_completed harus boolean (true/false)")
		}
		db = db.Where("user_teacher_is_completed = ?", val)
	}

	// ================== SEARCH ==================
	// Search by name (ILIKE %q%)
	if q != "" {
		if len([]rune(q)) < 2 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Panjang kata kunci minimal 2 karakter")
		}
		db = db.Where("user_teacher_name ILIKE ?", "%"+q+"%")
	}

	// ================== COUNT ==================
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ================== PAGINATION + SORTING ==================
	if !params.All { // kalau per_page=all, biarkan limit dari preset (AllHardCap) via params.Limit()
		db = db.Limit(params.Limit()).Offset(params.Offset())
	} else {
		db = db.Limit(params.Limit()).Offset(0)
	}
	db = db.Order(orderBy)

	// ================== FETCH ==================
	var rows []model.UserTeacherModel
	if err := db.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Map response items
	items := make([]userdto.UserTeacherResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, userdto.ToUserTeacherResponse(r))
	}

	// Meta
	meta := helper.BuildMeta(total, params)

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"items": items,
		"meta":  meta,
		"q":     q,
	})
}
