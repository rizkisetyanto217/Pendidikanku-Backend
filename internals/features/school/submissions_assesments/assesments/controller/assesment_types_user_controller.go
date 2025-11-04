// file: internals/features/school/submissions_assesments/assesments/controller/assessment_type_controller.go
package controller

import (
	dto "schoolku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "schoolku_backend/internals/features/school/submissions_assesments/assesments/model"
	helper "schoolku_backend/internals/helpers"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	helperAuth "schoolku_backend/internals/helpers/auth"
)

// GET /assessment-types?active=&q=&limit=&offset=&sort_by=&sort_dir=
func (ctl *AssessmentTypeController) List(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	// 1) Resolve school context
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// slug → id (jika perlu)
	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return helper.JsonError(c, helperAuth.ErrSchoolContextMissing.Code, helperAuth.ErrSchoolContextMissing.Message)
	}

	// 2) Authorize: minimal member school (any role)
	if !helperAuth.UserHasSchool(c, mid) {
		return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak terdaftar pada school ini (membership).")
	}

	// 3) Build filter & validate
	var filt dto.ListAssessmentTypeFilter
	filt.AssessmentTypeSchoolID = mid

	// Filters opsional
	if v := strings.TrimSpace(c.Query("active")); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		filt.Active = &b
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		filt.Q = &q
	}

	// Paging & sorting
	filt.Limit = atoiOr(20, c.Query("limit"))
	filt.Offset = atoiOr(0, c.Query("offset"))
	if sb := strings.TrimSpace(c.Query("sort_by")); sb != "" { // name|created_at
		filt.SortBy = &sb
	}
	if sd := strings.TrimSpace(c.Query("sort_dir")); sd != "" { // asc|desc
		filt.SortDir = &sd
	}

	if err := ctl.Validator.Struct(&filt); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 4) Query tenant-scoped
	qry := ctl.DB.Model(&model.AssessmentTypeModel{}).
		Where("assessment_type_school_id = ?", filt.AssessmentTypeSchoolID)

	if filt.Active != nil {
		qry = qry.Where("assessment_type_is_active = ?", *filt.Active)
	}
	if filt.Q != nil {
		like := "%" + strings.ToLower(strings.TrimSpace(*filt.Q)) + "%"
		qry = qry.Where(
			"(LOWER(assessment_type_name) LIKE ? OR LOWER(assessment_type_key) LIKE ?)",
			like, like,
		)
	}

	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.AssessmentTypeModel
	if err := qry.
		Order(func() string {
			if filt.SortBy == nil {
				return "assessment_type_created_at DESC"
			}
			sb := strings.ToLower(strings.TrimSpace(*filt.SortBy))
			dir := "DESC"
			if filt.SortDir != nil && strings.EqualFold(strings.TrimSpace(*filt.SortDir), "asc") {
				dir = "ASC"
			}
			switch sb {
			case "name":
				return "assessment_type_name " + dir
			case "created_at":
				return "assessment_type_created_at " + dir
			default:
				return "assessment_type_created_at DESC"
			}
		}()).
		Limit(filt.Limit).
		Offset(filt.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.AssessmentTypeResponse, 0, len(rows))
	for i := range rows {
		out = append(out, mapToResponse(&rows[i]))
	}

	// ⬇️ gunakan meta konsisten (page, per_page, total, total_pages, has_next, has_prev, count, per_page_options)
	meta := helper.BuildPaginationFromOffset(total, filt.Offset, filt.Limit)
	return helper.JsonList(c, "ok", out, meta)
}
