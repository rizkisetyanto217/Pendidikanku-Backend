// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_controller_list.go
package controller

import (
	"strings"

	dto "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	model "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /quizzes
// Catatan:
//   - Semua config + total soal (quiz_total_questions) sudah ikut ter-serialize
//     lewat dto.FromModel (kalau Questions di-preload, dia ikut dimap).
func (ctrl *QuizController) List(c *fiber.Ctx) error {
	// Inject DB buat helper slug→id
	c.Locals("DB", ctrl.DB)

	// =====================================================
	// 1) Resolve school dari TOKEN dulu
	// =====================================================

	var schoolID uuid.UUID

	if sid, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	} else if sid != uuid.Nil {
		schoolID = sid
	} else {
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
			return helper.JsonError(c, fiber.StatusBadRequest, "School context tidak ditemukan")
		}
	}

	// =====================================================
	// 2) Authorize: minimal member school
	// =====================================================
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// =====================================================
	// 3) Query params + validation
	// =====================================================
	var q dto.ListQuizzesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// Tambahan safety: parse manual assessment_id (kalau dikirim string acak)
	if s := strings.TrimSpace(c.Query("assessment_id")); s != "" {
		aid, err := uuid.Parse(s)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
		}
		q.AssessmentID = &aid
	}

	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force scope tenant
	q.SchoolID = &schoolID

	// =====================================================
	// 4) Pagination
	// =====================================================
	p := helper.ResolvePaging(c, 20, 200)

	// =====================================================
	// 5) Base query + filters
	// =====================================================
	dbq := ctrl.DB.WithContext(c.Context()).Model(&model.QuizModel{})
	dbq = applyFiltersQuizzes(dbq, &q)

	// 6) Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 7) Sort + Page window
	dbq = applySort(dbq, q.Sort)
	if p.Limit > 0 {
		dbq = dbq.Offset(p.Offset).Limit(p.Limit)
	}

	// 8) Optional: preload questions
	if q.WithQuestions {
		dbq = preloadQuestions(dbq, &q)
	}

	// 9) Fetch
	var rows []model.QuizModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 10) DTO mapping
	out := make([]dto.QuizResponse, 0, len(rows))
	for i := range rows {
		// Pakai versi timezone-aware
		resp := dto.FromModelWithCtx(c, &rows[i])

		// WithQuestionsCount → pakai denorm quiz_total_questions
		if q.WithQuestionsCount {
			n := rows[i].QuizTotalQuestions
			resp.QuestionsCount = &n
		}

		out = append(out, resp)
	}

	// 11) Response
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)
	return helper.JsonList(c, "ok", out, pg)
}
