package controller

import (
	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"
	helper "masjidku_backend/internals/helpers"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	helperAuth "masjidku_backend/internals/helpers/auth"
)

// GET /assessment-types?active=&q=&limit=&offset=&sort_by=&sort_dir=
func (ctl *AssessmentTypeController) List(c *fiber.Ctx) error {
	// ambil masjid_id prefer teacher
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	// authorize: anggota masjid (semua role)
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil { return err }

	var filt dto.ListAssessmentTypeFilter
	filt.AssessmentTypesMasjidID = mid

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

	// Query tenant-scoped
	qry := ctl.DB.Model(&model.AssessmentTypeModel{}).
		Where("assessment_types_masjid_id = ?", filt.AssessmentTypesMasjidID)

	if filt.Active != nil {
		qry = qry.Where("assessment_types_is_active = ?", *filt.Active)
	}
	if filt.Q != nil {
		like := "%" + strings.ToLower(strings.TrimSpace(*filt.Q)) + "%"
		qry = qry.Where(
			"(LOWER(assessment_types_name) LIKE ? OR LOWER(assessment_types_key) LIKE ?)",
			like, like,
		)
	}

	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.AssessmentTypeModel
	if err := qry.
		// khusus tabel ini: name/created_at
		Order(func() string {
			if filt.SortBy == nil { return "assessment_types_created_at DESC" }
			sb := strings.ToLower(strings.TrimSpace(*filt.SortBy))
			dir := "DESC"
			if filt.SortDir != nil && strings.EqualFold(strings.TrimSpace(*filt.SortDir), "asc") {
				dir = "ASC"
			}
			switch sb {
			case "name":
				return "assessment_types_name " + dir
			case "created_at":
				return "assessment_types_created_at " + dir
			default:
				return "assessment_types_created_at DESC"
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

	return helper.JsonList(c, out, fiber.Map{
		"total":  total,
		"limit":  filt.Limit,
		"offset": filt.Offset,
	})
}
