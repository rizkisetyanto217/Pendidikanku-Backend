package controller

import (
	"errors"
	"strconv"
	"strings"

	userdto "madinahsalam_backend/internals/features/users/user_teachers/dto"
	"madinahsalam_backend/internals/features/users/user_teachers/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func (uc *UserTeacherController) List(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))

	// Pastikan DB ada di Locals (kalau ada helper lain yang butuh)
	if c.Locals("DB") == nil {
		c.Locals("DB", uc.DB)
	}

	// Pakai helper ParseFiber untuk baca sort_by & order
	params := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// mapping nama sort → kolom DB
	allowed := map[string]string{
		"created_at": "user_teacher_created_at",
		"name":       "user_teacher_user_full_name_cache",
		"completed":  "user_teacher_is_completed",
		"verified":   "user_teacher_is_verified",
	}

	orderClause, err := params.SafeOrderClause(allowed, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Sort tidak valid")
	}
	orderBy := strings.TrimPrefix(orderClause, "ORDER BY ")

	db := uc.DB.Model(&model.UserTeacherModel{})

	// 🔒 Hanya data milik user ini (by user_id token)
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	db = db.Where("user_teacher_user_id = ?", userID)

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
	if q != "" {
		if len([]rune(q)) < 2 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Panjang kata kunci minimal 2 karakter")
		}

		// Cari di full_name + username cache
		db = db.Where(
			"(user_teacher_user_full_name_cache ILIKE ? OR user_teacher_user_name_cache ILIKE ?)",
			"%"+q+"%", "%"+q+"%",
		)
	}

	// ================== SINGLE RECORD ==================
	var row model.UserTeacherModel
	if err := db.Order(orderBy).Limit(1).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// ❗ BUKAN 404 lagi
			// Balikkan struktur lengkap dengan nilai default/kosong
			empty := model.UserTeacherModel{
				UserTeacherUserID:     userID,
				UserTeacherIsActive:   true, // samain dengan default di create
				UserTeacherIsVerified: false,
				// kolom lain akan zero value (string "", nil, false, dsb)
			}
			resp := userdto.ToUserTeacherResponse(empty)
			return helper.JsonOK(c, "Profil guru belum dibuat", resp)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := userdto.ToUserTeacherResponse(row)

	// 👇 LANGSUNG kirim objek, TANPA "item", TANPA "items", TANPA "meta"
	return helper.JsonOK(c, "Berhasil", resp)
}
