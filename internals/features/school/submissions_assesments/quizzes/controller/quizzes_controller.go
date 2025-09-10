// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_controller.go
package controller

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type QuizController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewQuizController(db *gorm.DB) *QuizController {
	return &QuizController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* =======================
   Helpers Tenant / Scope
======================= */

// WRITE endpoints → wajib admin/dkm/teacher
func (ctrl *QuizController) tenantMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Hanya admin/dkm/guru yang diizinkan")
	}
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	return mid, nil
}

// READ endpoints → boleh admin/dkm/teacher/student
// Mengembalikan satu masjid_id “aktif/terpilih” yang dimiliki user.
func (ctrl *QuizController) tenantMasjidIDAny(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) coba prefer teacher/admin chain
	if mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && mid != uuid.Nil {
		return mid, nil
	}
	// 2) coba active_masjid_id
	if mid, err := helperAuth.GetActiveMasjidID(c); err == nil && mid != uuid.Nil {
		return mid, nil
	}
	// 3) fallback: pakai list masjid dari token (termasuk student_records)
	if ids, err := helperAuth.GetMasjidIDsFromToken(c); err == nil && len(ids) > 0 {
		return ids[0], nil
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Butuh autentikasi")
}

// Pastikan user punya akses ke masjid_id tertentu (dipakai saat query mengirim masjid_id eksplisit)
func ensureMasjidScope(c *fiber.Ctx, target uuid.UUID) error {
	ids, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	for _, id := range ids {
		if id == target {
			return nil
		}
	}
	return fiber.NewError(fiber.StatusForbidden, "Masjid ID tidak sesuai scope pengguna")
}

/* =======================
   Filter & Sort
======================= */

func applyFilters(db *gorm.DB, q *dto.ListQuizzesQuery) *gorm.DB {
	if q == nil {
		return db
	}
	if q.MasjidID != nil {
		db = db.Where("quizzes_masjid_id = ?", *q.MasjidID)
	}
	if q.AssessmentID != nil {
		db = db.Where("quizzes_assessment_id = ?", *q.AssessmentID)
	}
	if q.IsPublished != nil {
		db = db.Where("quizzes_is_published = ?", *q.IsPublished)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		db = db.Where(
			"(LOWER(quizzes_title) LIKE ? OR LOWER(COALESCE(quizzes_description,'')) LIKE ?)",
			like, like,
		)
	}
	return db
}

func applySort(db *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(strings.ToLower(sort)) {
	case "created_at":
		return db.Order("quizzes_created_at ASC")
	case "desc_created_at", "":
		return db.Order("quizzes_created_at DESC")
	case "title":
		return db.Order("quizzes_title ASC NULLS LAST")
	case "desc_title":
		return db.Order("quizzes_title DESC NULLS LAST")
	case "published":
		return db.Order("quizzes_is_published ASC")
	case "desc_published":
		return db.Order("quizzes_is_published DESC")
	default:
		return db.Order("quizzes_created_at DESC")
	}
}


func applyFiltersQuizzes(db *gorm.DB, q *dto.ListQuizzesQuery) *gorm.DB {
	if q == nil {
		return db
	}

	// Pakai partial-index friendly predicate
	db = db.Where("quizzes_deleted_at IS NULL")

	// Filter by ID (PK)
	if q.ID != nil && *q.ID != uuid.Nil {
		db = db.Where("quizzes_id = ?", *q.ID)
	}

	// Scope tenant
	if q.MasjidID != nil && *q.MasjidID != uuid.Nil {
		db = db.Where("quizzes_masjid_id = ?", *q.MasjidID)
	}

	// Relasi assessment (opsional)
	if q.AssessmentID != nil && *q.AssessmentID != uuid.Nil {
		db = db.Where("quizzes_assessment_id = ?", *q.AssessmentID)
	}

	// Published flag (opsional)
	if q.IsPublished != nil {
		db = db.Where("quizzes_is_published = ?", *q.IsPublished)
	}

	// Pencarian teks (title/description) — manfaatkan gin_trgm_ops
	if s := strings.TrimSpace(q.Q); s != "" {
		like := "%" + s + "%"
		db = db.Where(
			"(quizzes_title ILIKE ? OR COALESCE(quizzes_description,'') ILIKE ?)",
			like, like,
		)
	}

	return db
}

func preloadQuestions(tx *gorm.DB, q *dto.ListQuizzesQuery) *gorm.DB {
	return tx.Preload("Questions", func(db *gorm.DB) *gorm.DB {
		db = db.Where("quiz_questions_deleted_at IS NULL")
		if q.MasjidID != nil && *q.MasjidID != uuid.Nil {
			db = db.Where("quiz_questions_masjid_id = ?", *q.MasjidID)
		}
		switch strings.ToLower(strings.TrimSpace(q.QuestionsOrder)) {
		case "created_at":
			db = db.Order("quiz_questions_created_at ASC")
		default: // "desc_created_at" or empty
			db = db.Order("quiz_questions_created_at DESC")
		}
		if q.QuestionsLimit > 0 {
			db = db.Limit(q.QuestionsLimit)
		}
		return db
	})
}



/* =======================
   Handlers
======================= */

// GET / (READ — admin/teacher/student)
// GET / (READ — admin/teacher/student)
func (ctrl *QuizController) List(c *fiber.Ctx) error {
	var q dto.ListQuizzesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// tentukan masjid_id untuk query
	if q.MasjidID != nil && *q.MasjidID != uuid.Nil {
		if err := ensureMasjidScope(c, *q.MasjidID); err != nil {
			return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
		}
	} else {
		mid, err := ctrl.tenantMasjidIDAny(c)
		if err != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
		}
		q.MasjidID = &mid
	}

	// Pagination (default created_at desc)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Base + filters
	dbq := ctrl.DB.Model(&model.QuizModel{})
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
			if err := ctrl.DB.
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

// POST / (WRITE — admin/teacher)
func (ctrl *QuizController) Create(c *fiber.Ctx) error {
	var body dto.CreateQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	mid, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Force masjid_id ke tenant untuk menghindari spoofing
	body.QuizzesMasjidID = mid

	m := body.ToModel()
	if err := ctrl.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "Quiz berhasil dibuat", dto.FromModel(m))
}

// PATCH /:id (WRITE — admin/teacher)
func (ctrl *QuizController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var m model.QuizModel
	if err := ctrl.DB.
		First(&m, "quizzes_id = ? AND quizzes_masjid_id = ?", id, mid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var body dto.PatchQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		// jika mau detail validasi, bisa helper.ValidationError
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	updates := body.ToUpdates()
	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.FromModel(&m))
	}

	if err := ctrl.DB.Model(&m).Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// reload
	if err := ctrl.DB.First(&m, "quizzes_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Quiz diperbarui", dto.FromModel(&m))
}

// DELETE /:id (WRITE — admin/teacher)
func (ctrl *QuizController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var m model.QuizModel
	if err := ctrl.DB.Select("quizzes_id").
		First(&m, "quizzes_id = ? AND quizzes_masjid_id = ?", id, mid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctrl.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Quiz dihapus", nil)
}
