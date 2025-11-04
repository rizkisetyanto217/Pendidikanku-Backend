// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_controller_list.go
package controller

import (
	dto "schoolku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	model "schoolku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /quizzes
// GET /quizzes
func (ctrl *QuizController) List(c *fiber.Ctx) error {
	// Inject DB buat helper slug→id
	c.Locals("DB", ctrl.DB)

	// 1) Resolve school context
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}

	// 2) slug → id bila perlu
	var schoolID uuid.UUID
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
		return helperAuth.ErrSchoolContextMissing
	}

	// 3) Authorize: minimal member school
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// 4) Query params
	var q dto.ListQuizzesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	// Force scope tenant
	q.SchoolID = &schoolID

	// 5) Pagination (jsonresponse style; default sort created_at desc dikelola di applySort)
	p := helper.ResolvePaging(c, 20, 200)

	// 6) Base query + filters
	dbq := ctrl.DB.WithContext(c.Context()).Model(&model.QuizModel{})
	dbq = applyFiltersQuizzes(dbq, &q)

	// 7) Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 8) Sort + Page window
	dbq = applySort(dbq, q.Sort)
	if p.Limit > 0 {
		dbq = dbq.Offset(p.Offset).Limit(p.Limit)
	}

	// 9) Optional: preload questions
	if q.WithQuestions {
		dbq = preloadQuestions(dbq, &q)
	}

	// 10) Fetch
	var rows []model.QuizModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 11) Optional: count questions (tanpa preload data)
	var countMap map[uuid.UUID]int
	if q.WithQuestionsCount && len(rows) > 0 {
		countMap = make(map[uuid.UUID]int, len(rows))
		ids := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			ids = append(ids, rows[i].QuizID)
		}
		type pair struct {
			QuizID uuid.UUID `gorm:"column:quiz_id"`
			N      int       `gorm:"column:n"`
		}
		var tmp []pair
		if err := ctrl.DB.WithContext(c.Context()).
			Table("quiz_questions").
			Select("quiz_questions_quiz_id AS quiz_id, COUNT(*) AS n").
			Where("quiz_questions_deleted_at IS NULL").
			Where("quiz_questions_quiz_id IN ?", ids).
			Group("quiz_questions_quiz_id").
			Scan(&tmp).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		for _, t := range tmp {
			countMap[t.QuizID] = t.N
		}
	}

	// 12) DTO mapping
	out := make([]dto.QuizResponse, 0, len(rows))
	for i := range rows {
		var resp dto.QuizResponse
		if q.WithQuestions {
			resp = dto.FromModelWithQuestions(&rows[i])
		} else {
			resp = dto.FromModel(&rows[i])
		}
		if q.WithQuestionsCount {
			if n, ok := countMap[rows[i].QuizID]; ok {
				resp.QuestionsCount = &n
			} else {
				z := 0
				resp.QuestionsCount = &z
			}
		}
		out = append(out, resp)
	}

	// 13) Response (pagination lengkap)
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)
	return helper.JsonList(c, "ok", out, pg)
}
