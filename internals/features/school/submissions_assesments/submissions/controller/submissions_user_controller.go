package controller

import (
	dto "masjidku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/submissions/model"
	helper "masjidku_backend/internals/helpers"
	"math"
	"strings"

	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET / (READ — all members; student hanya lihat miliknya)
func (ctrl *SubmissionController) List(c *fiber.Ctx) error {
	// =========================
	// 1) Resolve masjid context
	// =========================
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	// slug → id (jika perlu)
	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if strings.TrimSpace(mc.Slug) != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
		if er != nil || id == uuid.Nil {
			return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return helperAuth.ErrMasjidContextMissing
	}

	// ==========================================
	// 2) Authorize: minimal member masjid (any role)
	// ==========================================
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil {
		return err
	}

	// =========================
	// 3) Parse & validate query
	// =========================
	var q dto.ListSubmissionsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force scope ke tenant dari context
	q.MasjidID = &mid

	// Jika caller student (bukan DKM/Teacher) → hanya lihat miliknya
	isStudentOnly := helperAuth.IsStudent(c) && !helperAuth.IsDKM(c) && !helperAuth.IsTeacher(c)
	if isStudentOnly {
		if sid, e := helperAuth.GetMasjidStudentIDForMasjid(c, mid); e == nil && sid != uuid.Nil {
			q.StudentID = &sid
		} else {
			// Tidak punya student record di masjid → 0 hasil
			return helper.JsonList(c, []dto.SubmissionResponse{}, fiber.Map{
				"page":        1,
				"per_page":    20,
				"total":       0,
				"total_pages": 0,
			})
		}
	}

	page := clampPage(q.Page)
	perPage := clampPerPage(q.PerPage)

	// =========================
	// 4) Query + count
	// =========================
	dbq := ctrl.DB.WithContext(c.Context()).Model(&model.Submission{})
	dbq = applyFilters(dbq, &q)

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// =========================
	// 5) Page data
	// =========================
	var rows []model.Submission
	dbq = applySort(dbq, q.Sort)
	if err := dbq.Offset((page - 1) * perPage).Limit(perPage).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.SubmissionResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModel(&rows[i]))
	}

	pagination := fiber.Map{
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(perPage))),
	}

	return helper.JsonList(c, out, pagination)
}
