package controller

import (
	cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
	cpmodel "masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ---------- LIST (tenant-safe) ----------
func (ctl *ClassParentController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var q cpdto.ListClassParentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	q.Limit = clampLimit(q.Limit, 20, 200)
	if q.Offset < 0 { q.Offset = 0 }

	tx := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where("class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", masjidID)

	// ===== NEW: filter by class_parent_id (single) =====
	if s := strings.TrimSpace(c.Query("class_parent_id")); s != "" {
		id, perr := uuid.Parse(s)
		if perr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_parent_id tidak valid (uuid)")
		}
		tx = tx.Where("class_parent_id = ?", id)
	}

	// ===== NEW: filter by class_parent_ids (comma-separated) =====
	if s := strings.TrimSpace(c.Query("class_parent_ids")); s != "" {
		parts := strings.Split(s, ",")
		ids := make([]uuid.UUID, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, perr := uuid.Parse(p)
			if perr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "class_parent_ids mengandung UUID tidak valid")
			}
			ids = append(ids, id)
		}
		if len(ids) > 0 {
			tx = tx.Where("class_parent_id IN ?", ids)
		}
	}

	if q.Active != nil {
		tx = tx.Where("class_parent_is_active = ?", *q.Active)
	}
	if q.LevelMin != nil {
		tx = tx.Where("(class_parent_level IS NOT NULL AND class_parent_level >= ?)", *q.LevelMin)
	}
	if q.LevelMax != nil {
		tx = tx.Where("(class_parent_level IS NOT NULL AND class_parent_level <= ?)", *q.LevelMax)
	}
	if q.CreatedGt != nil {
		tx = tx.Where("class_parent_created_at > ?", *q.CreatedGt)
	}
	if q.CreatedLt != nil {
		tx = tx.Where("class_parent_created_at < ?", *q.CreatedLt)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		pat := "%" + s + "%"
		tx = tx.Where(`
			class_parent_name ILIKE ? OR
			class_parent_code ILIKE ? OR
			class_parent_description ILIKE ?
		`, pat, pat, pat)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	var rows []cpmodel.ClassParentModel
	if err := tx.
		Order("class_parent_created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := cpdto.ToClassParentResponses(rows)
	meta := cpdto.NewPaginationMeta(total, q.Limit, q.Offset, len(resps))
	return helper.JsonList(c, resps, meta)
}