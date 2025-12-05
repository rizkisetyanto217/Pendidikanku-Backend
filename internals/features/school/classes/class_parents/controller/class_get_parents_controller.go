// file: internals/features/school/classes/classes/controller/class_parent_list_controller.go
package controller

import (
	cpdto "madinahsalam_backend/internals/features/school/classes/class_parents/dto"
	cpmodel "madinahsalam_backend/internals/features/school/classes/class_parents/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

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
	// Biar helper lain bisa ambil DB dari Locals kalau perlu
	c.Locals("DB", ctl.DB)

	// =====================================================
	// 0) Parse include (opsional)
	//    contoh: ?include=meta atau ?include=stats
	// =====================================================
	rawInclude := strings.TrimSpace(c.Query("include", ""))
	includeMeta := false
	if rawInclude != "" {
		for _, part := range strings.Split(rawInclude, ",") {
			p := strings.ToLower(strings.TrimSpace(part))
			switch p {
			case "meta", "stats":
				includeMeta = true
			}
		}
	}

	// =====================================================
	// 1) Tentukan schoolID:
	//    - Prioritas: dari token (GetSchoolIDFromTokenPreferTeacher)
	//    - Fallback: dari ResolveSchoolContext (id / slug)
	// =====================================================

	var schoolID uuid.UUID

	// 1. Coba dulu dari token (teacher/student/dll)
	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		// 2. Kalau gagal / tidak ada token → pakai context multi-sumber
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				if er == gorm.ErrRecordNotFound {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school")
			}
			schoolID = id
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, helperAuth.ErrSchoolContextMissing.Error())
		}
	}

	// -------- query params → DTO --------
	var q cpdto.ListClassParentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ✅ Paging (standar jsonresponse + override dari DTO.limit/offset)
	p := helper.ResolvePaging(c, 20, 200) // default 20, max 200
	if q.Limit > 0 && q.Limit <= 200 {
		p.Limit = q.Limit
	}
	if q.Offset >= 0 {
		p.Offset = q.Offset
	}

	// only_my flag
	onlyMy := false
	if v := strings.TrimSpace(c.Query("only_my")); v != "" {
		onlyMy = strings.EqualFold(v, "1") ||
			strings.EqualFold(v, "true") ||
			strings.EqualFold(v, "yes")
	}

	// -------- base query --------
	tx := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where("class_parent_school_id = ? AND class_parent_deleted_at IS NULL", schoolID)

	// ----- filter by id (single) -----
	//   prefer: ?id=UUID
	//   legacy: ?class_parent_id=UUID
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		id, perr := uuid.Parse(s)
		if perr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid (uuid)")
		}
		tx = tx.Where("class_parent_id = ?", id)
	} else if s := strings.TrimSpace(c.Query("class_parent_id")); s != "" {
		id, perr := uuid.Parse(s)
		if perr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_parent_id tidak valid (uuid)")
		}
		tx = tx.Where("class_parent_id = ?", id)
	}

	// ----- filter by ids (multiple, comma-separated) -----
	//   prefer: ?ids=UUID1,UUID2
	//   legacy: ?class_parent_ids=UUID1,UUID2
	if raw := strings.TrimSpace(c.Query("ids")); raw != "" || strings.TrimSpace(c.Query("class_parent_ids")) != "" {
		s := raw
		if s == "" {
			s = strings.TrimSpace(c.Query("class_parent_ids"))
		}

		parts := strings.Split(s, ",")
		ids := make([]uuid.UUID, 0, len(parts))
		for _, pstr := range parts {
			pstr = strings.TrimSpace(pstr)
			if pstr == "" {
				continue
			}
			id, perr := uuid.Parse(pstr)
			if perr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "ids/class_parent_ids mengandung UUID tidak valid")
			}
			ids = append(ids, id)
		}
		if len(ids) > 0 {
			tx = tx.Where("class_parent_id IN ?", ids)
		}
	}

	// ----- filter kolom-kolom dari DTO -----
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

	// ----- filter khusus berdasarkan name -----
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
					  ON s.class_section_class_id = c.class_id
					 AND s.class_section_deleted_at IS NULL
					LEFT JOIN student_class_sections scs
					  ON scs.student_class_section_section_id = s.class_section_id
					 AND scs.student_class_section_deleted_at IS NULL
					 AND scs.student_class_section_status = 'active'
					LEFT JOIN school_students ms
					  ON ms.school_student_id = scs.student_class_section_school_student_id
					 AND ms.school_student_deleted_at IS NULL
					LEFT JOIN school_teachers mt
					  ON mt.school_teacher_id = s.class_section_teacher_id
					 AND mt.school_teacher_deleted_at IS NULL
					WHERE c.class_parent_id = class_parent_id
					  AND c.class_school_id = ?
					  AND (
							(ms.school_student_user_id = ?)
						 OR (mt.school_teacher_user_id = ?)
						 OR (s.class_section_teacher_user_id = ?)
					  )
				)
			`, schoolID, userID, userID, userID)
		}
	}

	// -------- eksekusi --------
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	var rows []cpmodel.ClassParentModel
	if err := tx.
		Order("class_parent_created_at DESC").
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := cpdto.ToClassParentResponses(rows)

	// ✅ pagination jsonresponse (pakai helper standar)
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	// ====== Tanpa include → tetap JsonList (behavior lama) ======
	if !includeMeta {
		return helper.JsonList(c, "ok", resps, pg)
	}

	// ====== Dengan include (meta/stats) → JsonListWithInclude ======

	// Contoh meta sederhana: total & total_active di seluruh sekolah
	var totalActive int64
	if err := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where("class_parent_school_id = ? AND class_parent_deleted_at IS NULL AND class_parent_is_active = TRUE", schoolID).
		Count(&totalActive).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	include := fiber.Map{
		"meta": fiber.Map{
			"total":        total,
			"total_active": totalActive,
		},
	}

	return helper.JsonListWithInclude(c, "ok", resps, include, pg)
}
