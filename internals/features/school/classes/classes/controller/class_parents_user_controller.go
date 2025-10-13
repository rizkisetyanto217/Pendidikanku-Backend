package controller

import (
	cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
	cpmodel "masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================

	LIST (tenant-safe; READ: DKM=semua, member=only_my)
	=========================================================
*/
func (ctl *ClassParentController) List(c *fiber.Ctx) error {
	// -------- Resolve masjid context --------
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else {
		if strings.TrimSpace(mc.Slug) == "" {
			return helperAuth.ErrMasjidContextMissing
		}
		id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
		if er != nil {
			if er == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve masjid")
		}
		masjidID = id
	}

	// -------- query params & paging --------
	var q cpdto.ListClassParentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	q.Limit = clampLimit(q.Limit, 20, 200)
	if q.Offset < 0 {
		q.Offset = 0
	}

	// only_my flag
	onlyMy := false
	if v := strings.TrimSpace(c.Query("only_my")); v != "" {
		onlyMy = strings.EqualFold(v, "1") || strings.EqualFold(v, "true")
	}

	tx := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where("class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", masjidID)

	// ----- filter by id(s) -----
	if s := strings.TrimSpace(c.Query("class_parent_id")); s != "" {
		id, perr := uuid.Parse(s)
		if perr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_parent_id tidak valid (uuid)")
		}
		tx = tx.Where("class_parent_id = ?", id)
	}
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

	// ----- filter kolom-kolom -----
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

	// ----- full-text sederhana: q ke name/code/description -----
	if s := strings.TrimSpace(q.Q); s != "" {
		pat := "%" + s + "%"
		tx = tx.Where(`
			class_parent_name ILIKE ? OR
			class_parent_code ILIKE ? OR
			class_parent_description ILIKE ?
		`, pat, pat, pat)
	}

	// ----- NEW: filter khusus berdasarkan name -----
	if s := strings.TrimSpace(q.Name); s != "" {
		tx = tx.Where("class_parent_name ILIKE ?", "%"+s+"%")
	}

	// ----- ONLY_MY (opsional) -----
	if onlyMy {
		if userID, _ := helperAuth.GetUserIDFromToken(c); userID != uuid.Nil {
			tx = tx.Where(`
				EXISTS (
					SELECT 1
					FROM classes c
					JOIN class_sections s
					  ON s.class_sections_class_id = c.class_id
					 AND s.class_sections_deleted_at IS NULL
					LEFT JOIN class_section_students css
					  ON css.class_section_students_section_id = s.class_sections_id
					 AND css.class_section_students_deleted_at IS NULL
					LEFT JOIN masjid_students ms
					  ON ms.masjid_student_id = css.class_section_students_masjid_student_id
					 AND ms.masjid_student_deleted_at IS NULL
					LEFT JOIN masjid_teachers mt
					  ON mt.masjid_teacher_id = s.class_sections_teacher_id
					 AND mt.masjid_teacher_deleted_at IS NULL
					WHERE c.class_parent_id = class_parent_id
					  AND c.class_masjid_id = ?
					  AND (
							(ms.masjid_student_user_id = ?)
						 OR (mt.masjid_teacher_user_id = ?)
						 OR (s.class_sections_teacher_user_id = ?)
					  )
				)
			`, masjidID, userID, userID, userID)
		}
	}

	// ----- eksekusi -----
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
