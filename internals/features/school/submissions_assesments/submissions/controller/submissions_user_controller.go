package controller

import (
	dto "masjidku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/submissions/model"
	helper "masjidku_backend/internals/helpers"
	"math"

	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET / (READ — all members; student hanya lihat miliknya)
func (ctrl *SubmissionController) List(c *fiber.Ctx) error {
	// Tenant & auth (semua anggota boleh lihat)
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil {
		return err
	}

	var q dto.ListSubmissionsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force scope ke tenant dari token
	q.MasjidID = &mid

	// Jika caller adalah student (bukan DKM/Teacher), PAKSA hanya melihat miliknya
	isStudentOnly := helperAuth.IsStudent(c) && !helperAuth.IsDKM(c) && !helperAuth.IsTeacher(c)
	if isStudentOnly {
		if sid, e := helperAuth.GetMasjidStudentIDForMasjid(c, mid); e == nil && sid != uuid.Nil {
			q.StudentID = &sid
		} else {
			// Tidak punya student record di masjid → tidak ada data
			empty := []dto.SubmissionResponse{}
			return helper.JsonList(c, empty, fiber.Map{
				"page":        1,
				"per_page":    20,
				"total":       0,
				"total_pages": 0,
			})
		}
	}

	page := clampPage(q.Page)
	perPage := clampPerPage(q.PerPage)

	var total int64
	dbq := ctrl.DB.WithContext(c.Context()).Model(&model.Submission{})
	dbq = applyFilters(dbq, &q)

	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

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