package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	qdto "masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* =========================================================
   Controller
========================================================= */

type QuizQuestionsController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewQuizQuestionsController(db *gorm.DB) *QuizQuestionsController {
	return &QuizQuestionsController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* =========================================================
   Tenant helpers
========================================================= */

// Admin/Teacher only (untuk write)
func (ctl *QuizQuestionsController) tenantMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	if id, _ := helperAuth.GetMasjidIDFromToken(c); id != uuid.Nil {
		return id, nil
	}
	if id, _ := helperAuth.GetTeacherMasjidIDFromToken(c); id != uuid.Nil {
		return id, nil
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Hanya admin/guru yang diizinkan")
}

// ANY (admin/teacher/student) — untuk read
func (ctl *QuizQuestionsController) tenantMasjidIDAny(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		return id, nil
	}
	if id, err := helperAuth.GetActiveMasjidID(c); err == nil && id != uuid.Nil {
		return id, nil
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Butuh autentikasi")
}

/* =========================================================
   Filters & sort helpers
========================================================= */




func (ctl *QuizQuestionsController) applyFilters(db *gorm.DB, masjidID uuid.UUID, quizID *uuid.UUID, qType string, q string) *gorm.DB {
	db = db.Where("quiz_questions_masjid_id = ? AND quiz_questions_deleted_at IS NULL", masjidID)
	if quizID != nil && *quizID != uuid.Nil {
		db = db.Where("quiz_questions_quiz_id = ?", *quizID)
	}
	if t := strings.ToLower(strings.TrimSpace(qType)); t == "single" || t == "essay" {
		db = db.Where("quiz_questions_type = ?", t)
	}
	if s := strings.TrimSpace(q); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		db = db.Where("(LOWER(quiz_questions_text) LIKE ? OR LOWER(COALESCE(quiz_questions_explanation,'')) LIKE ?)", like, like)
	}
	return db
}

func (ctl *QuizQuestionsController) applySort(db *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
	case "created_at":
		return db.Order("quiz_questions_created_at ASC")
	case "desc_created_at", "":
		return db.Order("quiz_questions_created_at DESC")
	case "points":
		return db.Order("quiz_questions_points ASC")
	case "desc_points":
		return db.Order("quiz_questions_points DESC")
	case "type":
		return db.Order("quiz_questions_type ASC")
	case "desc_type":
		return db.Order("quiz_questions_type DESC")
	default:
		return db.Order("quiz_questions_created_at DESC")
	}
}

/* =========================================================
   READ (User/Admin/Teacher)
========================================================= */

// GET /quiz-questions
// Query: quiz_id, type, q, page, per_page, sort
func (ctl *QuizQuestionsController) List(c *fiber.Ctx) error {
	masjidID, err := ctl.tenantMasjidIDAny(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var (
		quizID *uuid.UUID
	)
	if s := strings.TrimSpace(c.Query("quiz_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			quizID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
	}
	qType := c.Query("type") // "single"|"essay"|empty
	q := c.Query("q")
	sort := c.Query("sort")

	// pagination (simple)
	limit := atoiOr(20, c.Query("per_page"), c.Query("limit"))
	offset := pageOffset(atoiOr(0, c.Query("page")), limit)

	dbq := ctl.DB.Model(&qmodel.QuizQuestionModel{})
	dbq = ctl.applyFilters(dbq, masjidID, quizID, qType, q)

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	dbq = ctl.applySort(dbq, sort)
	if limit > 0 {
		dbq = dbq.Offset(offset).Limit(limit)
	}

	var rows []qmodel.QuizQuestionModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	out := qdto.FromModelsQuizQuestions(rows)

	meta := fiber.Map{
		"total":    total,
		"page":     atoiOr(0, c.Query("page")),
		"per_page": limit,
	}
	return helper.JsonList(c, out, meta)
}

// GET /quiz-questions/public
// Sama seperti List tapi hanya yang kuisnya published
func (ctl *QuizQuestionsController) ListPublic(c *fiber.Ctx) error {
	masjidID, err := ctl.tenantMasjidIDAny(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var quizID *uuid.UUID
	if s := strings.TrimSpace(c.Query("quiz_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			quizID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
	}
	qType := c.Query("type")
	q := c.Query("q")
	sort := c.Query("sort")
	limit := atoiOr(20, c.Query("per_page"), c.Query("limit"))
	offset := pageOffset(atoiOr(0, c.Query("page")), limit)

	// join ke quizzes untuk filter published
	dbq := ctl.DB.Table("quiz_questions qqq").
		Select("qqq.*").
		Joins("JOIN quizzes qq ON qq.quizzes_id = qqq.quiz_questions_quiz_id").
		Where("qqq.quiz_questions_masjid_id = ? AND qqq.quiz_questions_deleted_at IS NULL", masjidID).
		Where("qq.quizzes_is_published = ?", true)

	if quizID != nil && *quizID != uuid.Nil {
		dbq = dbq.Where("qqq.quiz_questions_quiz_id = ?", *quizID)
	}
	if t := strings.ToLower(strings.TrimSpace(qType)); t == "single" || t == "essay" {
		dbq = dbq.Where("qqq.quiz_questions_type = ?", t)
	}
	if s := strings.TrimSpace(q); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		dbq = dbq.Where("(LOWER(qqq.quiz_questions_text) LIKE ? OR LOWER(COALESCE(qqq.quiz_questions_explanation,'')) LIKE ?)", like, like)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	dbq = ctl.applySort(dbq, sort)
	if limit > 0 {
		dbq = dbq.Offset(offset).Limit(limit)
	}

	var rows []qmodel.QuizQuestionModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	out := qdto.FromModelsQuizQuestions(rows)

	meta := fiber.Map{
		"total":    total,
		"page":     atoiOr(0, c.Query("page")),
		"per_page": limit,
	}
	return helper.JsonList(c, out, meta)
}

// GET /quiz-questions/:id
func (ctl *QuizQuestionsController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	masjidID, err := ctl.tenantMasjidIDAny(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var m qmodel.QuizQuestionModel
	if err := ctl.DB.
		First(&m, "quiz_questions_id = ? AND quiz_questions_masjid_id = ? AND quiz_questions_deleted_at IS NULL", id, masjidID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Soal tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "OK", qdto.FromModelQuizQuestion(&m))
}

/* =========================================================
   WRITE (Admin/Teacher)
========================================================= */

// POST /quiz-questions
func (ctl *QuizQuestionsController) Create(c *fiber.Ctx) error {
	var req qdto.CreateQuizQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	tenantMasjidID, err := ctl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Force masjid_id dari tenant
	req.QuizQuestionsMasjidID = tenantMasjidID

	// Safety: pastikan quiz_id milik masjid tenant
	var ok bool
	if err := ctl.DB.Raw(`
		SELECT EXISTS(
		  SELECT 1 FROM quizzes
		  WHERE quizzes_id = ? AND quizzes_masjid_id = ?
		)
	`, req.QuizQuestionsQuizID, tenantMasjidID).Scan(&ok).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !ok {
		return helper.JsonError(c, fiber.StatusForbidden, "Quiz tidak milik tenant aktif")
	}

	m, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Create(m).Error; err != nil {
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar aturan bentuk data (CHECK)")
		}
		if isForeignKeyViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Relasi tidak valid (quiz/masjid)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "Soal berhasil dibuat", qdto.FromModelQuizQuestion(m))
}

// PATCH /quiz-questions/:id
func (ctl *QuizQuestionsController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	tenantMasjidID, err := ctl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var m qmodel.QuizQuestionModel
	if err := ctl.DB.
		First(&m, "quiz_questions_id = ? AND quiz_questions_masjid_id = ? AND quiz_questions_deleted_at IS NULL", id, tenantMasjidID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Soal tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var req qdto.PatchQuizQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	if err := req.ApplyToModel(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Jika quiz_id berubah, validasi quiz baru milik tenant
	if req.QuizQuestionsQuizID.ShouldUpdate() && !req.QuizQuestionsQuizID.IsNull() {
		newQID := req.QuizQuestionsQuizID.Val()
		var ok bool
		if err := ctl.DB.Raw(`
			SELECT EXISTS(SELECT 1 FROM quizzes WHERE quizzes_id = ? AND quizzes_masjid_id = ?)
		`, newQID, tenantMasjidID).Scan(&ok).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if !ok {
			return helper.JsonError(c, fiber.StatusForbidden, "Quiz tidak milik tenant aktif")
		}
	}

	// Save all changed fields (optimistic)
	if err := ctl.DB.
		Model(&qmodel.QuizQuestionModel{}).
		Where("quiz_questions_id = ?", m.QuizQuestionsID).
		// gunakan Select(clause.Associations) jika ingin ikut associations (tidak perlu di sini)
		Select("*").
		Updates(&m).Error; err != nil {
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar aturan bentuk data (CHECK)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Reload
	if err := ctl.DB.First(&m, "quiz_questions_id = ?", m.QuizQuestionsID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "Soal diperbarui", qdto.FromModelQuizQuestion(&m))
}

// DELETE /quiz-questions/:id (soft delete: set deleted_at)
func (ctl *QuizQuestionsController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	tenantMasjidID, err := ctl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// pastikan exist dan milik tenant
	var m qmodel.QuizQuestionModel
	if err := ctl.DB.Select("quiz_questions_id").
		First(&m, "quiz_questions_id = ? AND quiz_questions_masjid_id = ? AND quiz_questions_deleted_at IS NULL", id, tenantMasjidID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Soal tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	now := time.Now()
	if err := ctl.DB.Model(&qmodel.QuizQuestionModel{}).
		Where("quiz_questions_id = ?", id).
		Update("quiz_questions_deleted_at", now).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonDeleted(c, "Soal dihapus", nil)
}

/* =========================================================
   Small utils
========================================================= */

func atoiOr(def int, vals ...string) int {
	for _, s := range vals {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		var n int
		_, err := fmtSscanf(s, &n)
		if err == nil {
			return n
		}
	}
	return def
}

// very small, avoids strconv import; expects pure integer string
func fmtSscanf(s string, out *int) (int, error) {
	var n int
	sign := 1
	i := 0
	if s != "" && (s[0] == '-' || s[0] == '+') {
		if s[0] == '-' {
			sign = -1
		}
		i++
	}
	if i >= len(s) {
		return 0, errors.New("invalid")
	}
	for ; i < len(s); i++ {
		ch := s[i]
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid")
		}
		n = n*10 + int(ch-'0')
	}
	*out = n * sign
	return 1, nil
}

func pageOffset(page, perPage int) int {
	if page <= 0 {
		return 0
	}
	return page * perPage
}

/* =========================================================
   DB error helpers (reuse pattern dari controller lain)
========================================================= */

func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "check constraint")
}

func isForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "foreign key")
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unique")
}
