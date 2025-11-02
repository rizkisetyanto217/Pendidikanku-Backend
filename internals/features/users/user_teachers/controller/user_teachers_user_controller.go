package controller

import (
	"schoolku_backend/internals/features/users/user_teachers/model"
	"strings"

	userdto "schoolku_backend/internals/features/users/user_teachers/dto"

	helper "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
)

// GET /api/user-teachers
// Query params:
//
//	q         : string (opsional) -> cari di user_teacher_name
//	page      : int (default 1)
//	per_page  : int (default 25, max 200)  -- sesuai DefaultOpts
//	sort_by   : "created_at" (default) | "name"
//	order     : "asc" | "desc" (default "desc")
//	(alias lama: limit/offset/sort juga masih didukung via ParseFiber)
func (uc *UserTeacherController) List(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))

	// Pakai preset default; kalau admin/ekspor tinggal ganti ke helper.AdminOpts / helper.ExportOpts
	params := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Whitelist kolom yang boleh disort
	allowed := map[string]string{
		"created_at": "user_teacher_created_at",
		"name":       "user_teacher_name",
	}

	orderClause, err := params.SafeOrderClause(allowed, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Sort tidak valid")
	}
	// SafeOrderClause menghasilkan "ORDER BY <col> <DIR>", sedangkan GORM.Order butuh tanpa prefix "ORDER BY "
	orderBy := strings.TrimPrefix(orderClause, "ORDER BY ")

	db := uc.DB.Model(&model.UserTeacherModel{})

	// Filter (opsional): hanya aktif
	// db = db.Where("user_teacher_is_active = ?", true)

	// Search by name (ILIKE %q%)
	if q != "" {
		if len([]rune(q)) < 2 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Panjang kata kunci minimal 2 karakter")
		}
		db = db.Where("user_teacher_name ILIKE ?", "%"+q+"%")
	}

	// Hitung total sebelum limit/offset
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// Pagination + Sorting
	if !params.All { // kalau per_page=all, biarkan limit dari preset (AllHardCap) via params.Limit()
		db = db.Limit(params.Limit()).Offset(params.Offset())
	} else {
		db = db.Limit(params.Limit()).Offset(0)
	}
	db = db.Order(orderBy)

	// Fetch
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
