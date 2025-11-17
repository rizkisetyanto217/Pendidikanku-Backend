// file: internals/features/school/classes/class_sections/controller/class_section_list.go
package controller

import (
	"encoding/json"
	"strings"

	csstModel "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/model"
	secDTO "schoolku_backend/internals/features/school/classes/class_sections/dto"
	secModel "schoolku_backend/internals/features/school/classes/class_sections/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =================== parseUUIDList tetap sama =================== */

func parseUUIDList(s string) ([]uuid.UUID, error) {
	parts := strings.Split(s, ",")
	seen := make(map[uuid.UUID]struct{}, len(parts))
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "daftar id kosong")
	}
	return out, nil
}

// GET /api/{a|u}/:school_id/class-sections/list
func (ctrl *ClassSectionController) List(c *fiber.Ctx) error {
	// ---------- School context: token > slug/id ----------
	var schoolID uuid.UUID

	// 1) Coba dari token dulu (student/teacher/dkm/admin/bendahara)
	if sid, errTok := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); errTok == nil && sid != uuid.Nil {
		schoolID = sid
	} else {
		// 2) Fallback ke ResolveSchoolContext (path param / header / dll)
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
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil {
				if er == gorm.ErrRecordNotFound {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id

		default:
			return helper.JsonError(c, fiber.StatusBadRequest, helperAuth.ErrSchoolContextMissing.Error())
		}
	}

	// ---------- Search term ----------
	rawQ := strings.TrimSpace(c.Query("q"))
	rawSearch := strings.TrimSpace(c.Query("search"))
	searchTerm := rawSearch
	if rawQ != "" {
		searchTerm = rawQ
		if len([]rune(searchTerm)) < 2 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter q minimal 2 karakter")
		}
	}

	// ---------- Paging & sorting ----------
	defaultSortBy := "created_at"
	defaultOrder := "desc"
	if searchTerm != "" {
		defaultSortBy = "name"
		defaultOrder = "asc"
	}
	pg := helper.ResolvePaging(c, 20, 200)

	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", defaultSortBy)))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", defaultOrder)))
	if order != "asc" && order != "desc" {
		order = defaultOrder
	}
	col := "class_section_created_at"
	switch sortBy {
	case "name":
		col = "class_section_name"
	case "created_at":
		col = "class_section_created_at"
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ---------- Filters ----------
	var (
		sectionIDs []uuid.UUID
		classIDs   []uuid.UUID
		teacherIDs []uuid.UUID // NEW: filter by school_teacher
		activeOnly *bool
	)

	// filter by section id (existing)
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		ids, e := parseUUIDList(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid: "+e.Error())
		}
		sectionIDs = ids
	}

	// filter by class_id (mendukung multi)
	if s := strings.TrimSpace(c.Query("class_id")); s != "" {
		ids, e := parseUUIDList(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_id tidak valid: "+e.Error())
		}
		classIDs = ids
	}

	// NEW: filter by school_teacher_id (alias: teacher_id), mendukung multi
	rawTeacher := strings.TrimSpace(c.Query("school_teacher_id"))
	if rawTeacher == "" {
		rawTeacher = strings.TrimSpace(c.Query("teacher_id"))
	}
	if rawTeacher != "" {
		ids, e := parseUUIDList(rawTeacher)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_teacher_id/teacher_id tidak valid: "+e.Error())
		}
		teacherIDs = ids
	}

	if s := strings.TrimSpace(c.Query("is_active")); s != "" {
		v := c.QueryBool("is_active")
		activeOnly = &v
	}
	withCSST := c.QueryBool("with_csst")

	// ---------- Query base ----------
	tx := ctrl.DB.
		Model(&secModel.ClassSectionModel{}).
		Where("class_section_deleted_at IS NULL").
		Where("class_section_school_id = ?", schoolID)

	if len(sectionIDs) > 0 {
		tx = tx.Where("class_section_id IN ?", sectionIDs)
	}

	if len(classIDs) > 0 {
		tx = tx.Where("class_section_class_id IN ?", classIDs)
	}

	// 🔥 apply filter wali/teacher section ke kolom FK
	if len(teacherIDs) > 0 {
		tx = tx.Where("class_section_school_teacher_id IN ?", teacherIDs)
	}

	if activeOnly != nil {
		tx = tx.Where("class_section_is_active = ?", *activeOnly)
	}
	if searchTerm != "" {
		s := "%" + strings.ToLower(searchTerm) + "%"
		tx = tx.Where(`
			LOWER(class_section_name) LIKE ?
			OR LOWER(class_section_code) LIKE ?
			OR LOWER(class_section_slug) LIKE ?`,
			s, s, s)
	}

	// ---------- Total ----------
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	// ---------- Optional: all=1 ----------
	if c.QueryBool("all") {
		pg.Offset = 0
		pg.Limit = int(total)
	}

	// ---------- Data ----------
	tx = tx.Order(orderExpr).Limit(pg.Limit).Offset(pg.Offset)

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---------- Build items ----------
	items := make([]secDTO.ClassSectionResponse, 0, len(rows))
	idsInPage := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		items = append(items, secDTO.FromModelClassSection(&rows[i]))
		idsInPage = append(idsInPage, rows[i].ClassSectionID)
	}

	pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)

	// ---------- Inject CSST ----------
	if withCSST {
		targetIDs := sectionIDs
		if len(targetIDs) == 0 {
			targetIDs = idsInPage
		}

		if len(targetIDs) > 0 {
			var csstRows []csstModel.ClassSectionSubjectTeacherModel
			csstQ := ctrl.DB.
				Model(&csstModel.ClassSectionSubjectTeacherModel{}).
				Where("class_section_subject_teacher_deleted_at IS NULL").
				Where("class_section_subject_teacher_school_id = ?", schoolID).
				Where("class_section_subject_teacher_class_section_id IN ?", targetIDs)
			if activeOnly != nil {
				csstQ = csstQ.Where("class_section_subject_teacher_is_active = ?", *activeOnly)
			}
			if err := csstQ.Find(&csstRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil CSST")
			}

			// group by class_section_id
			bySection := make(map[uuid.UUID][]csstModel.ClassSectionSubjectTeacherModel, len(targetIDs))
			for i := range csstRows {
				r := csstRows[i]
				bySection[r.ClassSectionSubjectTeacherClassSectionID] =
					append(bySection[r.ClassSectionSubjectTeacherClassSectionID], r)
			}

			out := make([]fiber.Map, 0, len(items))
			for i := range items {
				b, _ := json.Marshal(items[i])
				var m fiber.Map
				_ = json.Unmarshal(b, &m)

				m["class_sections_csst"] = []secDTO.CSSTItemLite{}
				m["class_sections_csst_count"] = 0
				m["class_sections_csst_active_count"] = 0

				secID := items[i].ClassSectionID
				if list, ok := bySection[secID]; ok && len(list) > 0 {
					lite := secDTO.CSSTLiteSliceFromModels(list)
					active := 0
					for _, it := range list {
						if it.ClassSectionSubjectTeacherIsActive {
							active++
						}
					}
					m["class_sections_csst"] = lite
					m["class_sections_csst_count"] = len(lite)
					m["class_sections_csst_active_count"] = active
				}
				out = append(out, m)
			}
			return helper.JsonList(c, "ok", out, pagination)
		}

		return helper.JsonList(c, "ok", items, pagination)
	}

	return helper.JsonList(c, "ok", items, pagination)
}
