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

// ===== Contoh registrasi route =====
// func RegisterQuizRoutes(app *fiber.App, db *gorm.DB) {
// 	ctrl := NewQuizController(db)
// 	g := app.Group("/api/a/quizzes")
// 	g.Get("/", ctrl.List)
// 	g.Get("/:id", ctrl.GetByID)
// 	g.Post("/", ctrl.Create)
// 	g.Patch("/:id", ctrl.Patch)
// 	g.Delete("/:id", ctrl.Delete)
// }

// ============ Helpers ============
func (ctrl *QuizController) tenantMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)

	switch {
	case adminMasjidID != uuid.Nil:
		return adminMasjidID, nil
	case teacherMasjidID != uuid.Nil:
		return teacherMasjidID, nil
	default:
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
}

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
		db = db.Where("(LOWER(quizzes_title) LIKE ? OR LOWER(COALESCE(quizzes_description,'')) LIKE ?)", like, like)
	}
	return db
}

func applySort(db *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
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

// ============ Handlers ============

// GET /
func (ctrl *QuizController) List(c *fiber.Ctx) error {
	var q dto.ListQuizzesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
	// default filter by tenant
	if q.MasjidID == nil {
		q.MasjidID = &tenantMasjidID
	} else if *q.MasjidID != tenantMasjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Masjid ID tidak sesuai tenant")
	}

	// Pagination via helper.ParseFiber (respect per_page=all preset if you want; here use DefaultOpts)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Base query + filters
	dbq := ctrl.DB.Model(&model.QuizModel{})
	dbq = applyFilters(dbq, &q)

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

	// Fetch
	var rows []model.QuizModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Map DTO
	out := make([]dto.QuizResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModel(&rows[i]))
	}

	// Pagination meta
	meta := helper.BuildMeta(total, p)

	return helper.JsonList(c, out, meta)
}

// GET /:id
func (ctrl *QuizController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	var m model.QuizModel
	if err := ctrl.DB.
		First(&m, "quizzes_id = ? AND quizzes_masjid_id = ?", id, tenantMasjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", dto.FromModel(&m))
}

// POST /
func (ctrl *QuizController) Create(c *fiber.Ctx) error {
	var body dto.CreateQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
	if body.QuizzesMasjidID != tenantMasjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Masjid ID tidak sesuai tenant")
	}

	m := body.ToModel()
	if err := ctrl.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Quiz berhasil dibuat", dto.FromModel(m))
}

// PATCH /:id
func (ctrl *QuizController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	var m model.QuizModel
	if err := ctrl.DB.
		First(&m, "quizzes_id = ? AND quizzes_masjid_id = ?", id, tenantMasjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var body dto.PatchQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// validator untuk Patch biasanya optional; tambahkan rule custom bila perlu
	if err := ctrl.Validator.Struct(&body); err != nil {
		// abaikan kalau kamu tidak set tag validate di Patch; atau kirim errornya
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

// DELETE /:id (soft delete)
func (ctrl *QuizController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	var m model.QuizModel
	if err := ctrl.DB.Select("quizzes_id").
		First(&m, "quizzes_id = ? AND quizzes_masjid_id = ?", id, tenantMasjidID).Error; err != nil {
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
