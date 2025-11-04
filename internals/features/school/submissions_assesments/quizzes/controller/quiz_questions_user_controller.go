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
	var schoolID uuid.UUID
	if mc.ID != uuid.Nil {
		schoolID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		schoolID = id
	} else {
		return helper.JsonError(c, helperAuth.ErrSchoolContextMissing.Code, helperAuth.ErrSchoolContextMissing.Message)
	}

	// 2) Authorize: minimal member school (semua role)
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
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
	qType := strings.TrimSpace(c.Query("type")) // "single"|"essay"|empty
	q := strings.TrimSpace(c.Query("q"))
	sort := strings.TrimSpace(c.Query("sort"))

	// 4) Paging (jsonresponse style)
	p := helper.ResolvePaging(c, 20, 200) // default 20, max 200

	// 5) Query data (tenant-scoped)
	dbq := ctl.DB.WithContext(c.Context()).
		Model(&qmodel.QuizQuestionModel{})
	dbq = ctl.applyFilters(dbq, schoolID, quizID, qType, q)

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	dbq = ctl.applySort(dbq, sort)
	if p.Limit > 0 {
		dbq = dbq.Offset(p.Offset).Limit(p.Limit)
	}

	var rows []qmodel.QuizQuestionModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := qdto.FromModelsQuizQuestions(rows)

	// 6) Response (pagination lengkap)
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)
	return helper.JsonList(c, "ok", out, pg)
}
