// file: internals/features/school/submissions_assesments/quizzes/controller/user_quiz_attempts_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	validator "github.com/go-playground/validator/v10"

	qdto "masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers" // <— pakai helper kamu
)

type UserQuizAttemptsController struct {
	DB        *gorm.DB
	validator *validator.Validate
}

func NewUserQuizAttemptsController(db *gorm.DB) *UserQuizAttemptsController {
	return &UserQuizAttemptsController{DB: db}
}

func (ctl *UserQuizAttemptsController) ensureValidator() {
	if ctl.validator == nil {
		ctl.validator = validator.New()
	}
}

// POST /user-quiz-attempts
func (ctl *UserQuizAttemptsController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	var req qdto.CreateUserQuizAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}

	m := req.ToModel()
	if err := ctl.DB.Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Duplikat / melanggar unique index")
		}
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar CHECK constraint")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan attempt")
	}
	return helper.JsonCreated(c, "Berhasil memulai attempt", qdto.FromModelUserQuizAttempt(m))
}

// PATCH /user-quiz-attempts/:id
func (ctl *UserQuizAttemptsController) Patch(c *fiber.Ctx) error {
	ctl.ensureValidator()

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req qdto.UpdateUserQuizAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}

	var m qmodel.UserQuizAttemptModel
	if err := ctl.DB.First(&m, "user_quiz_attempts_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Attempt tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := req.ApplyToModel(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Patch tidak valid")
	}

	if err := ctl.DB.Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Melanggar aturan unik")
		}
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar CHECK constraint")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	return helper.JsonUpdated(c, "Berhasil memperbarui attempt", qdto.FromModelUserQuizAttempt(&m))
}

// GET /user-quiz-attempts?quiz_id=&student_id=&status=&active_only=true
func (ctl *UserQuizAttemptsController) List(c *fiber.Ctx) error {
	quizIDStr := strings.TrimSpace(c.Query("quiz_id"))
	studentIDStr := strings.TrimSpace(c.Query("student_id"))
	statusStr := strings.TrimSpace(c.Query("status"))
	activeOnly := strings.EqualFold(strings.TrimSpace(c.Query("active_only")), "true")

	q := ctl.DB.Model(&qmodel.UserQuizAttemptModel{})

	// filter
	if quizIDStr != "" {
		quizID, err := uuid.Parse(quizIDStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
		q = q.Where("user_quiz_attempts_quiz_id = ?", quizID)
	}
	if studentIDStr != "" {
		studentID, err := uuid.Parse(studentIDStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "student_id tidak valid")
		}
		q = q.Where("user_quiz_attempts_student_id = ?", studentID)
	}
	if statusStr != "" {
		st := qmodel.UserQuizAttemptStatus(statusStr)
		if !st.Valid() {
			return helper.JsonError(c, fiber.StatusBadRequest, "status tidak valid (in_progress|submitted|finished|abandoned)")
		}
		q = q.Where("user_quiz_attempts_status = ?", st)
	}
	if activeOnly {
		q = q.Where("user_quiz_attempts_status IN (?)",
			[]string{string(qmodel.UserAttemptInProgress), string(qmodel.UserAttemptSubmitted)})
	}

	// total
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// pagination
	p := helper.ParseFiber(c, "user_quiz_attempts_started_at", "desc", helper.DefaultOpts)

	var rows []*qmodel.UserQuizAttemptModel
	if err := q.Order("user_quiz_attempts_started_at DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, qdto.FromModelsUserQuizAttempts(rows), meta)
}

// GET /user-quiz-attempts/:id
func (ctl *UserQuizAttemptsController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}
	var m qmodel.UserQuizAttemptModel
	if err := ctl.DB.First(&m, "user_quiz_attempts_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Attempt tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return helper.JsonOK(c, "OK", qdto.FromModelUserQuizAttempt(&m))
}

// DELETE /user-quiz-attempts/:id
func (ctl *UserQuizAttemptsController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}
	if err := ctl.DB.Delete(&qmodel.UserQuizAttemptModel{}, "user_quiz_attempts_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus")
	}
	return helper.JsonDeleted(c, "Berhasil menghapus", fiber.Map{"deleted_id": id})
}

