// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_questions_controller.go
package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	qdto "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	dbtime "madinahsalam_backend/internals/helpers/dbtime"
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
   Tenant helpers (list)
========================================================= */

func (ctl *QuizQuestionsController) applyFilters(db *gorm.DB, schoolID uuid.UUID, quizID *uuid.UUID, qType string, q string) *gorm.DB {
	db = db.Where("quiz_question_school_id = ? AND quiz_question_deleted_at IS NULL", schoolID)
	if quizID != nil && *quizID != uuid.Nil {
		db = db.Where("quiz_question_quiz_id = ?", *quizID)
	}
	if t := strings.ToLower(strings.TrimSpace(qType)); t == "single" || t == "essay" {
		db = db.Where("quiz_question_type = ?", t)
	}
	if s := strings.TrimSpace(q); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		db = db.Where("(LOWER(quiz_question_text) LIKE ? OR LOWER(COALESCE(quiz_question_explanation,'')) LIKE ?)", like, like)
	}
	return db
}

func (ctl *QuizQuestionsController) applySort(db *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
	case "created_at":
		return db.Order("quiz_question_created_at ASC")
	case "desc_created_at", "":
		return db.Order("quiz_question_created_at DESC")
	case "points":
		return db.Order("quiz_question_points ASC")
	case "desc_points":
		return db.Order("quiz_question_points DESC")
	case "type":
		return db.Order("quiz_question_type ASC")
	case "desc_type":
		return db.Order("quiz_question_type DESC")
	default:
		return db.Order("quiz_question_created_at DESC")
	}
}

/* =========================================================
   WRITE (DKM/Admin/Teacher)
========================================================= */

// POST /quiz-questions
// Bisa terima:
//   - Single object  : { ... }
//   - Bulk (array)   : [ { ... }, { ... } ]
func (ctl *QuizQuestionsController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// ========================================
	// 1) Parse body: dukung object / array
	// ========================================

	body := c.Body()
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload kosong")
	}

	var reqs []qdto.CreateQuizQuestionRequest

	switch trim[0] {
	case '[': // array
		if err := json.Unmarshal(trim, &reqs); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (array): "+err.Error())
		}
	case '{': // single object
		var single qdto.CreateQuizQuestionRequest
		if err := json.Unmarshal(trim, &single); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (object): "+err.Error())
		}
		reqs = append(reqs, single)
	default:
		return helper.JsonError(c, fiber.StatusBadRequest, "Format JSON tidak dikenali (harus object atau array)")
	}

	if len(reqs) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Tidak ada soal yang dikirim")
	}

	// ========================================
	// 2) Tenant & role:
	//    gunakan helperAuth.ResolveSchoolForDKMOrTeacher
	// ========================================

	schoolID, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		// Resolver ini boleh saja mengembalikan *fiber.Error atau error lain.
		// Kalau dia sudah tulis response sendiri (mis. lewat helper.JsonError),
		// cukup return err apa adanya.
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return err
	}

	// ========================================
	// 3) Validasi & konversi ke model
	//    - Force school_id
	//    - Validasi struct
	//    - Safety: cek quiz_id milik tenant
	// ========================================

	models := make([]*qmodel.QuizQuestionModel, 0, len(reqs))

	// Cache hasil cek quiz per quiz_id biar nggak nembak DB berulang
	quizChecked := make(map[uuid.UUID]bool)

	for i := range reqs {
		reqs[i].QuizQuestionSchoolID = schoolID

		// Validasi DTO per item
		if err := ctl.Validator.Struct(&reqs[i]); err != nil {
			return helper.JsonError(
				c,
				fiber.StatusBadRequest,
				"Validasi gagal pada item ke-"+strconv.Itoa(i+1)+": "+err.Error(),
			)
		}

		qid := reqs[i].QuizQuestionQuizID

		// Cek quiz_id milik tenant (sekali per quiz_id)
		if _, already := quizChecked[qid]; !already {
			var exists bool
			if err := ctl.DB.Raw(`
				SELECT EXISTS(
				  SELECT 1 FROM quizzes
				  WHERE quiz_id = ? AND quiz_school_id = ?
				)
			`, qid, schoolID).Scan(&exists).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}
			if !exists {
				return helper.JsonError(
					c,
					fiber.StatusForbidden,
					"Quiz tidak milik tenant aktif (item ke-"+strconv.Itoa(i+1)+")",
				)
			}
			quizChecked[qid] = true
		}

		m, err := reqs[i].ToModel()
		if err != nil {
			return helper.JsonError(
				c,
				fiber.StatusBadRequest,
				"Konversi gagal pada item ke-"+strconv.Itoa(i+1)+": "+err.Error(),
			)
		}
		models = append(models, m)
	}

	// ========================================
	// 4) Simpan dalam transaksi
	// ========================================

	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		for _, m := range models {
			if err := tx.Create(m).Error; err != nil {
				if isCheckViolation(err) {
					return fiber.NewError(fiber.StatusBadRequest, "Melanggar aturan bentuk data (CHECK)")
				}
				if isForeignKeyViolation(err) {
					return fiber.NewError(fiber.StatusBadRequest, "Relasi tidak valid (quiz/school)")
				}
				if isUniqueViolation(err) {
					return fiber.NewError(fiber.StatusBadRequest, "Duplikasi data (UNIQUE)")
				}
				return err
			}
		}
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ========================================
	// 5) Build response DTO
	// ========================================

	if len(models) == 1 {
		return helper.JsonCreated(c, "Soal berhasil dibuat", qdto.FromModelQuizQuestion(models[0]))
	}

	out := make([]*qdto.QuizQuestionResponse, 0, len(models))
	for _, m := range models {
		out = append(out, qdto.FromModelQuizQuestion(m))
	}

	return helper.JsonCreated(c, "Soal berhasil dibuat (multiple)", out)
}

// PATCH /quiz-questions/:id
func (ctl *QuizQuestionsController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 🔒 Tentukan school_id lewat helperAuth (DKM/Admin/Teacher)
	schoolID, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return err
	}

	var m qmodel.QuizQuestionModel
	if err := ctl.DB.
		First(&m, "quiz_question_id = ? AND quiz_question_school_id = ? AND quiz_question_deleted_at IS NULL", id, schoolID).
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

	// Normalisasi change_kind dari user
	userKind := strings.ToLower(strings.TrimSpace(req.ChangeKind))
	if userKind != "" && userKind != "major" && userKind != "minor" {
		return helper.JsonError(c, fiber.StatusBadRequest, "change_kind harus salah satu dari: major, minor, atau kosong")
	}

	// Deteksi apakah ada field "sensitif penilaian" yang berubah.
	// Kalau iya, dan user tidak set "major", kita paksa jadi major.
	majorFieldChanged :=
		(req.QuizQuestionCorrect.ShouldUpdate() && !req.QuizQuestionCorrect.IsNull()) ||
			req.QuizQuestionAnswers.ShouldUpdate() ||
			req.QuizQuestionType.ShouldUpdate() ||
			req.QuizQuestionPoints.ShouldUpdate()

	if majorFieldChanged && userKind != "major" {
		req.ChangeKind = "major"
	}

	// Terapkan patch + (jika major) simpan history + increment version
	if err := req.ApplyToModel(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Jika quiz_id berubah, validasi quiz baru milik tenant
	if req.QuizQuestionQuizID.ShouldUpdate() && !req.QuizQuestionQuizID.IsNull() {
		newQID := req.QuizQuestionQuizID.Val()
		var ok bool
		if err := ctl.DB.Raw(`
			SELECT EXISTS(SELECT 1 FROM quizzes WHERE quiz_id = ? AND quiz_school_id = ?)
		`, newQID, schoolID).Scan(&ok).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if !ok {
			return helper.JsonError(c, fiber.StatusForbidden, "Quiz tidak milik tenant aktif")
		}
	}

	// Save all changed fields (optimistic)
	if err := ctl.DB.
		Model(&qmodel.QuizQuestionModel{}).
		Where("quiz_question_id = ?", m.QuizQuestionID).
		Select("*").
		Updates(&m).Error; err != nil {
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar aturan bentuk data (CHECK)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Reload
	if err := ctl.DB.First(&m, "quiz_question_id = ?", m.QuizQuestionID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "Soal diperbarui", qdto.FromModelQuizQuestion(&m))
}

// DELETE /quiz-questions/:id (soft delete: set deleted_at)
func (ctl *QuizQuestionsController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 🔒 Tentukan school_id lewat helperAuth (DKM/Admin/Teacher)
	schoolID, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return err
	}

	// pastikan exist dan milik tenant
	var m qmodel.QuizQuestionModel
	if err := ctl.DB.Select("quiz_question_id").
		First(&m, "quiz_question_id = ? AND quiz_question_school_id = ? AND quiz_question_deleted_at IS NULL", id, schoolID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Soal tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	now, err := dbtime.GetDBTime(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mendapatkan waktu server")
	}

	if err := ctl.DB.Model(&qmodel.QuizQuestionModel{}).
		Where("quiz_question_id = ?", id).
		Update("quiz_question_deleted_at", now).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonDeleted(c, "Soal dihapus", nil)
}

/* =========================================================
   DB error helpers
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
