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

	mid, err := helperAuth.GetMasjidIDFromToken(c) // ini prefer DKM/Admin (bukan teacher)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMMasjid(c, mid); err != nil {
		return err
	}


	// Force masjid_id dari tenant
	req.QuizQuestionsMasjidID = mid

	// Safety: pastikan quiz_id milik masjid tenant
	var ok bool
	if err := ctl.DB.Raw(`
		SELECT EXISTS(
		  SELECT 1 FROM quizzes
		  WHERE quizzes_id = ? AND quizzes_masjid_id = ?
		)
	`, req.QuizQuestionsQuizID, mid).Scan(&ok).Error; err != nil {
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
	mid, err := helperAuth.GetMasjidIDFromToken(c) // ini prefer DKM/Admin (bukan teacher)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMMasjid(c, mid); err != nil {
		return err
	}


	var m qmodel.QuizQuestionModel
	if err := ctl.DB.
		First(&m, "quiz_questions_id = ? AND quiz_questions_masjid_id = ? AND quiz_questions_deleted_at IS NULL", id, mid).
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
		`, newQID, mid).Scan(&ok).Error; err != nil {
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
	mid, err := helperAuth.GetMasjidIDFromToken(c) // ini prefer DKM/Admin (bukan teacher)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMMasjid(c, mid); err != nil {
		return err
	}


	// pastikan exist dan milik tenant
	var m qmodel.QuizQuestionModel
	if err := ctl.DB.Select("quiz_questions_id").
		First(&m, "quiz_questions_id = ? AND quiz_questions_masjid_id = ? AND quiz_questions_deleted_at IS NULL", id, mid).
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
