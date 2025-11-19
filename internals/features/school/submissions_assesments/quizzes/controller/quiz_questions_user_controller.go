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

/* =========================================================
   READ / LIST
========================================================= */

// GET /quiz-questions
// Contoh:
//
//	/quiz-questions?quiz_id=...&with_quiz=true
//	/quiz-questions?id=...&with_quiz=true
func (ctl *QuizQuestionsController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// =====================================================
	// 1) Tentukan schoolID:
	//    - Prioritas: dari token (GetSchoolIDFromTokenPreferTeacher)
	//    - Fallback: dari ResolveSchoolContext (id / slug)
	// =====================================================

	var schoolID uuid.UUID

	// 1. Coba dari token dulu
	if tokenSchoolID, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && tokenSchoolID != uuid.Nil {
		schoolID = tokenSchoolID
	} else {
		// 2. Fallback: pakai resolver lama (id / slug)
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
			if er != nil || id == uuid.Nil {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "School context hilang")
		}
	}

	// Minimal member school
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// 2) Parse query
	var q qdto.ListQuizQuestionsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	// Force tenant
	q.SchoolID = &schoolID

	// 3) Paging
	p := helper.ResolvePaging(c, 20, 200)

	// 4) Base query
	dbq := ctl.DB.WithContext(c.Context()).Model(&qmodel.QuizQuestionModel{})

	// Filter by specific question ID (opsional)
	if q.ID != nil && *q.ID != uuid.Nil {
		dbq = dbq.Where("quiz_question_id = ?", *q.ID)
	}

	// Filter lain: quiz_id, type, q
	dbq = ctl.applyFilters(dbq, schoolID, q.QuizID, q.Type, q.Q)

	// 5) Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 6) Sort + pagination
	dbq = ctl.applySort(dbq, q.Sort)
	if p.Limit > 0 {
		dbq = dbq.Offset(p.Offset).Limit(p.Limit)
	}

	// 7) Optional preload parent quiz
	if q.WithQuiz {
		dbq = dbq.Preload("Quiz")
	}

	// 8) Fetch
	var rows []qmodel.QuizQuestionModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 9) DTO
	out := qdto.FromModelsQuizQuestions(rows)

	// 10) Pagination response
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)
	return helper.JsonList(c, "ok", out, pg)
}
