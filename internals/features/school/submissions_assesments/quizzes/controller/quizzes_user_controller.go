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
	// Inject DB agar helper slug→id bisa jalan
	c.Locals("DB", ctrl.DB)

	// 1) Resolve masjid context (path/header/cookie/query/host/token)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err // sudah *fiber.Error* dari helper
	}

	// 2) slug → id bila perlu
	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if mc.Slug != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
		if er != nil || id == uuid.Nil {
			return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return helperAuth.ErrMasjidContextMissing
	}

	// 3) Authorize: minimal member masjid (any role)
	if !helperAuth.UserHasMasjid(c, mid) {
		return fiber.NewError(fiber.StatusForbidden, "Anda tidak terdaftar pada masjid ini (membership).")
	}

	// 4) Query params
	var q dto.ListQuizzesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force scope ke tenant dari context (hindari spoof)
	q.MasjidID = &mid

	// 5) Pagination (default created_at desc)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// 6) Base + filters
	dbq := ctrl.DB.WithContext(c.Context()).Model(&model.QuizModel{})
	dbq = applyFiltersQuizzes(dbq, &q)

	// 7) Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 8) Sort + Page
	dbq = applySort(dbq, q.Sort)
	if !p.All {
		dbq = dbq.Offset(p.Offset()).Limit(p.Limit())
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

	// 11) Optional: count total soal per quiz (tanpa preload data)
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

	// 12) Map DTO
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
