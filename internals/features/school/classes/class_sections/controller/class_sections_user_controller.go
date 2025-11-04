// file: internals/features/school/classes/class_sections/controller/class_section_list.go
package controller

import (
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
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	/* ---------- School context ---------- */
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	var schoolID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		schoolID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		schoolID = id
	default:
		return helper.JsonError(c, fiber.StatusBadRequest, helperAuth.ErrSchoolContextMissing.Error())
	}

	/* ---------- Search term ---------- */
	rawQ := strings.TrimSpace(c.Query("q"))
	rawSearch := strings.TrimSpace(c.Query("search"))
	searchTerm := rawSearch
	if rawQ != "" {
		searchTerm = rawQ
		if len([]rune(searchTerm)) < 2 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter q minimal 2 karakter")
		}
	}

	/* ---------- Paging & sorting (jsonresponse style) ---------- */
	// Default: created_at desc, tapi kalau ada search → name asc
	defaultSortBy := "created_at"
	defaultOrder := "desc"
	if searchTerm != "" {
		defaultSortBy = "name"
		defaultOrder = "asc"
	}

	pg := helper.ResolvePaging(c, 20, 200) // per_page default=20, max=200

	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", defaultSortBy)))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", defaultOrder)))
	if order != "asc" && order != "desc" {
		order = defaultOrder
	}

	// whitelist kolom → nama kolom DB
	col := "class_section_created_at"
	switch sortBy {
	case "name":
		col = "class_section_name"
	case "created_at":
		col = "class_section_created_at"
	}
	orderExpr := col + " " + strings.ToUpper(order)

	/* ---------- Filters ---------- */
	var (
		sectionIDs []uuid.UUID
		activeOnly *bool
	)
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		ids, e := parseUUIDList(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid: "+e.Error())
		}
		sectionIDs = ids
	}
	if s := strings.TrimSpace(c.Query("is_active")); s != "" {
		v := c.QueryBool("is_active")
		activeOnly = &v
	}
	withCSST := c.QueryBool("with_csst")

	/* ---------- Query base (tenant-safe) ---------- */
	tx := ctrl.DB.
		Model(&secModel.ClassSectionModel{}).
		Where("class_section_deleted_at IS NULL").
		Where("class_section_school_id = ?", schoolID)

	if len(sectionIDs) > 0 {
		tx = tx.Where("class_section_id IN ?", sectionIDs)
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

	/* ---------- Total ---------- */
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	/* ---------- Optional: all=1 untuk ambil semua ---------- */
	if c.QueryBool("all") {
		pg.Offset = 0
		pg.Limit = int(total)
	}

	/* ---------- Data ---------- */
	tx = tx.Order(orderExpr).Limit(pg.Limit).Offset(pg.Offset)

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	/* ---------- Build items ---------- */
	items := make([]secDTO.ClassSectionResponse, 0, len(rows))
	idsInPage := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		items = append(items, secDTO.FromModelClassSection(&rows[i]))
		idsInPage = append(idsInPage, rows[i].ClassSectionID)
	}

	// pagination object jsonresponse
	pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)

	/* ---------- (Opsional) CSST includes ---------- */
	if withCSST {
		targetIDs := sectionIDs
		if len(targetIDs) == 0 {
			targetIDs = idsInPage
		}

		csstBySection := make(map[uuid.UUID][]csstModel.ClassSectionSubjectTeacherModel, len(targetIDs))

		if len(targetIDs) > 0 {
			var csstRows []csstModel.ClassSectionSubjectTeacherModel
			csstQ := ctrl.DB.
				Model(&csstModel.ClassSectionSubjectTeacherModel{}).
				Where("class_section_subject_teacher_deleted_at IS NULL").
				Where("class_section_subject_teacher_school_id = ?", schoolID).
				Where("class_section_subject_teacher_section_id IN ?", targetIDs)

			if activeOnly != nil {
				csstQ = csstQ.Where("class_section_subject_teacher_is_active = ?", *activeOnly)
			}

			if err := csstQ.Find(&csstRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil CSST")
			}

			for i := range csstRows {
				r := csstRows[i]
				csstBySection[r.ClassSectionSubjectTeacherSectionID] =
					append(csstBySection[r.ClassSectionSubjectTeacherSectionID], r)
			}
		}

		includes := fiber.Map{
			"csst_by_section": csstBySection, // map[UUID][]ClassSectionSubjectTeacherModel
		}
		return helper.JsonListEx(c, "ok", items, pagination, includes)
	}

	// ✅ default tanpa includes
	return helper.JsonList(c, "ok", items, pagination)
}
