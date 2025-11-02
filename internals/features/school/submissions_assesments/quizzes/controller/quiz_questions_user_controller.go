package controller

import (
	qdto "schoolku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "schoolku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /quiz-questions
// Query: quiz_id, type, q, page, per_page, sort
func (ctl *QuizQuestionsController) List(c *fiber.Ctx) error {
	// biar helper GetSchoolIDBySlug bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	// 1) Resolve school context (path/header/cookie/query/host/token)
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// slug → id jika perlu
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

	// 2) Authorize: minimal member school (semua role)
	if err := helperAuth.EnsureMemberSchool(c, mid); err != nil {
		return err
	}

	// 3) Query params
	var quizID *uuid.UUID
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

	// pagination (0-based page; kompatibel dgn pageOffset helper yg sudah ada)
	limit := atoiOr(20, c.Query("per_page"), c.Query("limit"))
	offset := pageOffset(atoiOr(0, c.Query("page")), limit)

	// 4) Query data (tenant-scoped, kolom singular)
	dbq := ctl.DB.WithContext(c.Context()).Model(&qmodel.QuizQuestionModel{})
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

	// 5) Response
	meta := fiber.Map{
		"total":    total,
		"page":     atoiOr(0, c.Query("page")),
		"per_page": limit,
	}
	return helper.JsonList(c, out, meta)
}
