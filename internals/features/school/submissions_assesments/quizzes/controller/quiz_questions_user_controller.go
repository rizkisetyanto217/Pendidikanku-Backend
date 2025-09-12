package controller

import (
	qdto "masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   READ (User/Admin/Teacher)
========================================================= */

// GET /quiz-questions
// Query: quiz_id, type, q, page, per_page, sort
func (ctl *QuizQuestionsController) List(c *fiber.Ctx) error {
	// ambil masjid_id prefer teacher
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	// authorize: anggota masjid (semua role)
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil { return err }

	var (
		quizID *uuid.UUID
	)
	if s := strings.TrimSpace(c.Query("quiz_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			quizID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
	}
	qType := c.Query("type") // "single"|"essay"|empty
	q := c.Query("q")
	sort := c.Query("sort")

	// pagination (simple)
	limit := atoiOr(20, c.Query("per_page"), c.Query("limit"))
	offset := pageOffset(atoiOr(0, c.Query("page")), limit)

	dbq := ctl.DB.Model(&qmodel.QuizQuestionModel{})
	dbq = ctl.applyFilters(dbq, mid, quizID, qType, q)

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	dbq = ctl.applySort(dbq, sort)
	if limit > 0 {
		dbq = dbq.Offset(offset).Limit(limit)
	}

	var rows []qmodel.QuizQuestionModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	out := qdto.FromModelsQuizQuestions(rows)

	meta := fiber.Map{
		"total":    total,
		"page":     atoiOr(0, c.Query("page")),
		"per_page": limit,
	}
	return helper.JsonList(c, out, meta)
}
