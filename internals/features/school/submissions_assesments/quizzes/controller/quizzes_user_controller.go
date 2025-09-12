package controller

import (
	dto "masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET / (READ — semua role anggota masjid)
func (ctrl *QuizController) List(c *fiber.Ctx) error {
	// Tenant & auth
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil {
		return err
	}

	// Query params
	var q dto.ListQuizzesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force scope ke tenant dari token (hindari spoof)
	q.MasjidID = &mid

	// Pagination (default created_at desc)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Base + filters
	dbq := ctrl.DB.WithContext(c.Context()).Model(&model.QuizModel{})
	dbq = applyFiltersQuizzes(dbq, &q)

	// Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Sort + Page
	dbq = applySort(dbq, q.Sort)
	if !p.All {
		dbq = dbq.Offset(p.Offset()).Limit(p.Limit())
	}

	// Optional: preload questions
	if q.WithQuestions {
		dbq = preloadQuestions(dbq, &q)
	}

	// Fetch
	var rows []model.QuizModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Optional: count total soal per quiz (tanpa preload data)
	var countMap map[uuid.UUID]int
	if q.WithQuestionsCount {
		countMap = make(map[uuid.UUID]int, len(rows))
		if len(rows) > 0 {
			ids := make([]uuid.UUID, 0, len(rows))
			for i := range rows {
				ids = append(ids, rows[i].QuizzesID)
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
	}

	// Map DTO
	out := make([]dto.QuizResponse, 0, len(rows))
	for i := range rows {
		var resp dto.QuizResponse
		if q.WithQuestions {
			resp = dto.FromModelWithQuestions(&rows[i])
		} else {
			resp = dto.FromModel(&rows[i])
		}
		if q.WithQuestionsCount {
			if n, ok := countMap[rows[i].QuizzesID]; ok {
				resp.QuestionsCount = &n
			} else {
				zero := 0
				resp.QuestionsCount = &zero
			}
		}
		out = append(out, resp)
	}

	return helper.JsonList(c, out, helper.BuildMeta(total, p))
}
