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

	// Partial-index friendly predicate
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

// POST / (WRITE — DKM/Teacher/Admin)
func (ctrl *QuizController) Create(c *fiber.Ctx) error {
	var body dto.CreateQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
	}

	// Force masjid_id ke tenant untuk menghindari spoofing
	body.QuizzesMasjidID = mid

	m := body.ToModel()
	if err := ctrl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
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

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
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

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
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
