// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_controller.go
package controller

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	model "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
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
		return db.Order("quiz_created_at ASC")
	case "desc_created_at", "":
		return db.Order("quiz_created_at DESC")
	case "title":
		return db.Order("quiz_title ASC NULLS LAST")
	case "desc_title":
		return db.Order("quiz_title DESC NULLS LAST")
	case "published":
		return db.Order("quiz_is_published ASC")
	case "desc_published":
		return db.Order("quiz_is_published DESC")
	default:
		return db.Order("quiz_created_at DESC")
	}
}

func applyFiltersQuizzes(db *gorm.DB, q *dto.ListQuizzesQuery) *gorm.DB {
	if q == nil {
		return db
	}
	db = db.Where("quiz_deleted_at IS NULL")

	if q.ID != nil && *q.ID != uuid.Nil {
		db = db.Where("quiz_id = ?", *q.ID)
	}
	if q.SchoolID != nil && *q.SchoolID != uuid.Nil {
		db = db.Where("quiz_school_id = ?", *q.SchoolID)
	}
	if q.AssessmentID != nil && *q.AssessmentID != uuid.Nil {
		db = db.Where("quiz_assessment_id = ?", *q.AssessmentID)
	}
	if q.Slug != nil && strings.TrimSpace(*q.Slug) != "" {
		db = db.Where("LOWER(quiz_slug) = LOWER(?)", strings.TrimSpace(*q.Slug))
	}
	if q.IsPublished != nil {
		db = db.Where("quiz_is_published = ?", *q.IsPublished)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		like := "%" + s + "%"
		db = db.Where("(quiz_title ILIKE ? OR COALESCE(quiz_description,'') ILIKE ?)", like, like)
	}
	return db
}

func preloadQuestions(tx *gorm.DB, q *dto.ListQuizzesQuery) *gorm.DB {
	return tx.Preload("Questions", func(db *gorm.DB) *gorm.DB {
		// HAPUS filter deleted_at
		if q.SchoolID != nil && *q.SchoolID != uuid.Nil {
			db = db.Where("quiz_question_school_id = ?", *q.SchoolID)
		}

		switch strings.ToLower(strings.TrimSpace(q.QuestionsOrder)) {
		case "created_at":
			db = db.Order("quiz_question_created_at ASC")
		default:
			db = db.Order("quiz_question_created_at DESC")
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

func resolveSchoolForDKMOrTeacher(c *fiber.Ctx, db *gorm.DB) (uuid.UUID, error) {
	// injek DB agar GetSchoolIDBySlug bisa jalan
	c.Locals("DB", db)

	// 1) Coba ambil school dari TOKEN dulu (prefer teacher)
	if sid, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return uuid.Nil, fe
		}
		return uuid.Nil, err
	} else if sid != uuid.Nil {
		mid := sid

		// Pastikan user ini DKM/Admin/Teacher di school tsb
		if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil {
			return uuid.Nil, err
		}
		return mid, nil
	}

	// 2) Fallback: pakai school context (id / slug)
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return uuid.Nil, helperAuth.ErrSchoolContextMissing
	}

	// 3) Pastikan user DKM/Admin/Teacher di school ini
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil {
		return uuid.Nil, err
	}

	return mid, nil
}

/* =======================
   Handlers
======================= */

// POST /quizzes (WRITE — DKM/Teacher/Admin)
func (ctrl *QuizController) Create(c *fiber.Ctx) error {
	var body dto.CreateQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force school scope (DKM/Teacher)
	mid, err := resolveSchoolForDKMOrTeacher(c, ctrl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	body.QuizSchoolID = mid

	// Build model dari DTO (sudah termasuk snapshot behaviour & scoring)
	m := body.ToModel()

	// Generate slug (pakai body jika ada; else dari title) → pastikan unik per tenant (alive only)
	base := ""
	if body.QuizSlug != nil && strings.TrimSpace(*body.QuizSlug) != "" {
		base = helper.Slugify(*body.QuizSlug, 160)
	} else {
		base = helper.Slugify(body.QuizTitle, 160)
	}
	uniq, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		ctrl.DB,
		"quizzes",
		"quiz_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("quiz_school_id = ? AND quiz_deleted_at IS NULL", mid)
		},
		160,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyiapkan slug")
	}
	m.QuizSlug = &uniq

	// Simpan
	if err := ctrl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Quiz berhasil dibuat", dto.FromModel(m))
}

// PATCH /quizzes/:id (WRITE — DKM/Teacher/Admin)
func (ctrl *QuizController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := resolveSchoolForDKMOrTeacher(c, ctrl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.QuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&m, "quiz_id = ? AND quiz_school_id = ?", id, mid).Error; err != nil {
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

	// Jika slug di-update (dan bukan null), enforce uniqueness per tenant (alive-only, exclude self)
	if body.QuizSlug.ShouldUpdate() && !body.QuizSlug.IsNull() {
		raw := strings.TrimSpace(body.QuizSlug.Val())
		if raw == "" {
			updates["quiz_slug"] = gorm.Expr("NULL")
		} else {
			base := helper.Slugify(raw, 160)
			uniq, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				ctrl.DB,
				"quizzes",
				"quiz_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where("quiz_school_id = ? AND quiz_deleted_at IS NULL AND quiz_id <> ?", mid, id)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyiapkan slug")
			}
			updates["quiz_slug"] = uniq
		}
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.FromModel(&m))
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Model(&m).
		Where("quiz_id = ? AND quiz_school_id = ?", id, mid).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// reload
	if err := ctrl.DB.WithContext(c.Context()).
		First(&m, "quiz_id = ? AND quiz_school_id = ?", id, mid).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Quiz diperbarui", dto.FromModel(&m))
}

// DELETE /quizzes/:id (WRITE — DKM/Teacher/Admin)
func (ctrl *QuizController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := resolveSchoolForDKMOrTeacher(c, ctrl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.QuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Select("quiz_id").
		First(&m, "quiz_id = ? AND quiz_school_id = ?", id, mid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctrl.DB.WithContext(c.Context()).Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Quiz dihapus", fiber.Map{
		"quiz_id": id,
	})
}
