// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_controller.go
package controller

import (
	"encoding/json"
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
   Filter & Sort
======================= */

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
	db = db.Where("quizzes_deleted_at IS NULL")

	if q.ID != nil && *q.ID != uuid.Nil {
		db = db.Where("quizzes_id = ?", *q.ID)
	}
	if q.MasjidID != nil && *q.MasjidID != uuid.Nil {
		db = db.Where("quizzes_masjid_id = ?", *q.MasjidID)
	}
	if q.AssessmentID != nil && *q.AssessmentID != uuid.Nil {
		db = db.Where("quizzes_assessment_id = ?", *q.AssessmentID)
	}
	if q.IsPublished != nil {
		db = db.Where("quizzes_is_published = ?", *q.IsPublished)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		like := "%" + s + "%"
		db = db.Where("(quizzes_title ILIKE ? OR COALESCE(quizzes_description,'') ILIKE ?)", like, like)
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
		default:
			db = db.Order("quiz_questions_created_at DESC")
		}
		if q.QuestionsLimit > 0 {
			db = db.Limit(q.QuestionsLimit)
		}
		return db
	})
}

/* =======================
   Auth helper (DKM/Admin ATAU Teacher)
======================= */

func resolveMasjidForDKMOrTeacher(c *fiber.Ctx, db *gorm.DB) (uuid.UUID, error) {
	// injek DB agar GetMasjidIDBySlug bisa jalan
	c.Locals("DB", db)

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return uuid.Nil, helperAuth.ErrMasjidContextMissing
	}

	// 1) DKM/Admin?
	if err := helperAuth.EnsureDKMMasjid(c, mid); err == nil {
		return mid, nil
	}
	// 2) Teacher pada masjid ini?
	if helperAuth.IsTeacher(c) {
		if tMid, _ := helperAuth.GetTeacherMasjidIDFromToken(c); tMid != uuid.Nil && tMid == mid {
			return mid, nil
		}
	}
	// 3) gagal
	return uuid.Nil, helperAuth.ErrMasjidContextForbidden
}

/* =======================
   Handlers
======================= */

// POST / (WRITE — DKM/Teacher/Admin)
func (ctrl *QuizController) Create(c *fiber.Ctx) error {
	var body dto.CreateQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔐 Force masjid scope (DKM/Teacher)
	mid, err := resolveMasjidForDKMOrTeacher(c, ctrl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	body.QuizzesMasjidID = mid

	// Build model dari DTO
	m := body.ToModel()

	// 🏷️ Generate slug (selalu kita isi)
	// - pakai slug dari body kalau ada → normalisasi
	// - else pakai judul
	base := ""
	if body.QuizzesSlug != nil && strings.TrimSpace(*body.QuizzesSlug) != "" {
		base = helper.Slugify(*body.QuizzesSlug, 160)
	} else {
		base = helper.Slugify(body.QuizzesTitle, 160)
	}
	// Pastikan unik per tenant, hanya yang belum soft-deleted
	uniq, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		ctrl.DB,
		"quizzes",
		"quizzes_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("quizzes_masjid_id = ? AND quizzes_deleted_at IS NULL", mid)
		},
		160,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyiapkan slug")
	}
	m.QuizzesSlug = &uniq

	// Simpan
	if err := ctrl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		// (opsional) bisa mapping 23505 → slug duplikat jika race condition
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Quiz berhasil dibuat", dto.FromModel(m))
}

// PATCH /:id (WRITE — DKM/Teacher/Admin)
func (ctrl *QuizController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := resolveMasjidForDKMOrTeacher(c, ctrl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.QuizModel
	if err := ctrl.DB.WithContext(c.Context()).
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
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	updates := body.ToUpdates()

	// --- Optional: handle quizzes_slug dari payload (tidak ada di DTO) ---
	var slugBody struct {
		QuizzesSlug *string `json:"quizzes_slug"`
	}
	_ = json.Unmarshal(c.Body(), &slugBody) // abaikan error; optional saja

	if slugBody.QuizzesSlug != nil {
		raw := strings.TrimSpace(*slugBody.QuizzesSlug)
		if raw == "" {
			// Kosongkan slug → set NULL
			updates["quizzes_slug"] = gorm.Expr("NULL")
		} else {
			base := helper.Slugify(raw, 160)
			uniq, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				ctrl.DB,
				"quizzes",
				"quizzes_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					// scope: per-tenant, alive-only, exclude diri sendiri
					return q.Where("quizzes_masjid_id = ? AND quizzes_deleted_at IS NULL AND quizzes_id <> ?", mid, id)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyiapkan slug")
			}
			updates["quizzes_slug"] = uniq
		}
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.FromModel(&m))
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Model(&m).
		Where("quizzes_id = ? AND quizzes_masjid_id = ?", id, mid).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// reload
	if err := ctrl.DB.WithContext(c.Context()).
		First(&m, "quizzes_id = ? AND quizzes_masjid_id = ?", id, mid).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Quiz diperbarui", dto.FromModel(&m))
}

// DELETE /:id (WRITE — DKM/Teacher/Admin)
func (ctrl *QuizController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := resolveMasjidForDKMOrTeacher(c, ctrl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.QuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Select("quizzes_id").
		First(&m, "quizzes_id = ? AND quizzes_masjid_id = ?", id, mid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctrl.DB.WithContext(c.Context()).Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Quiz dihapus", fiber.Map{
		"quizzes_id": id,
	})
}
