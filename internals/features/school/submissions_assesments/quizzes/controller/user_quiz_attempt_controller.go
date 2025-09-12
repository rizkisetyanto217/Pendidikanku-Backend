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
	helperAuth "masjidku_backend/internals/helpers/auth"
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


// file: .../controller/user_quiz_attempts_controller.go


// Ambil masjid_id dari quizzes, aman di-scan sebagai string
func (ctl *UserQuizAttemptsController) getQuizMasjidID(quizID uuid.UUID) (uuid.UUID, error) {
    if quizID == uuid.Nil {
        return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "quiz_id wajib")
    }

    var masjidIDStr string
    if err := ctl.DB.
        Raw(`SELECT quizzes_masjid_id::text FROM quizzes WHERE quizzes_id = ? AND quizzes_deleted_at IS NULL`, quizID).
        Scan(&masjidIDStr).Error; err != nil {
        return uuid.Nil, err
    }
    if strings.TrimSpace(masjidIDStr) == "" {
        return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "Quiz tidak ditemukan / sudah dihapus")
    }

    mid, err := uuid.Parse(masjidIDStr)
    if err != nil {
        return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Masjid ID quiz tidak valid")
    }
    return mid, nil
}

// balikin (masjidID, studentID, isStudent)
func (ctl *UserQuizAttemptsController) resolveScopeForCreate(c *fiber.Ctx, req *qdto.CreateUserQuizAttemptRequest) (uuid.UUID, uuid.UUID, bool, error) {
	// 1) derive masjid dari quiz
	qMid, err := ctl.getQuizMasjidID(req.UserQuizAttemptsQuizID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}

	// 2) siswa → ambil student_id dari token untuk masjid quiz
	if helperAuth.IsStudent(c) {
		sid, err := helperAuth.GetMasjidStudentIDForMasjid(c, qMid)
		if err != nil {
			return uuid.Nil, uuid.Nil, true, err
		}
		return qMid, sid, true, nil
	}

	// 3) admin/dkm/teacher → wajib punya akses ke masjid quiz
	if err := ctl.ensureMasjidScope(c, qMid); err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}

	// 4) admin/dkm/teacher → student_id wajib dikirim
	if req.UserQuizAttemptsStudentID == nil || *req.UserQuizAttemptsStudentID == uuid.Nil {
		return uuid.Nil, uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "user_quiz_attempts_student_id wajib untuk admin/dkm/teacher")
	}

	// validasi student milik masjid tsb (recommended)
	var ok bool
	if err := ctl.DB.Raw(`
		SELECT EXISTS(
			SELECT 1 FROM masjid_students
			WHERE masjid_student_id = ? AND masjid_id = ?
		)
	`, *req.UserQuizAttemptsStudentID, qMid).Scan(&ok).Error; err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}
	if !ok {
		return uuid.Nil, uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "student tidak terdaftar di masjid quiz")
	}

	return qMid, *req.UserQuizAttemptsStudentID, false, nil
}


// scope helper: cek user punya akses ke masjid tertentu
func (ctl *UserQuizAttemptsController) ensureMasjidScope(c *fiber.Ctx, masjidID uuid.UUID) error {
	if masjidID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib")
	}
	ids, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == masjidID {
			return nil
		}
	}
	return fiber.NewError(fiber.StatusForbidden, "Masjid ID tidak sesuai scope pengguna")
}


// POST /user-quiz-attempts
func (ctl *UserQuizAttemptsController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	var req qdto.CreateUserQuizAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Validasi sesuai DTO baru (hanya quiz_id wajib)
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}

	mid, sid, _, err := ctl.resolveScopeForCreate(c, &req)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Override anti-spoof
	req.UserQuizAttemptsMasjidID = &mid
	req.UserQuizAttemptsStudentID = &sid

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

	if helperAuth.IsStudent(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya admin/dkm/teacher yang diizinkan mengubah attempt")
	}

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

	// scope check: admin/dkm/teacher harus punya akses ke masjid attempt ini
	if err := ctl.ensureMasjidScope(c, m.UserQuizAttemptsMasjidID); err != nil {
		return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
	}

	var req qdto.UpdateUserQuizAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
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

// DELETE /user-quiz-attempts/:id
func (ctl *UserQuizAttemptsController) Delete(c *fiber.Ctx) error {
	if helperAuth.IsStudent(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya admin/dkm/teacher yang diizinkan menghapus attempt")
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var m qmodel.UserQuizAttemptModel
	if err := ctl.DB.Select("user_quiz_attempts_id, user_quiz_attempts_masjid_id").
		First(&m, "user_quiz_attempts_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Attempt tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if err := ctl.ensureMasjidScope(c, m.UserQuizAttemptsMasjidID); err != nil {
	return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
	}

	if err := ctl.DB.Delete(&qmodel.UserQuizAttemptModel{}, "user_quiz_attempts_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus")
	}
	return helper.JsonDeleted(c, "Berhasil menghapus", fiber.Map{"deleted_id": id})
}

// GET /user-quiz-attempts?quiz_id=&student_id=&status=&active_only=true
// GET /user-quiz-attempts?quiz_id=&student_id=&status=&active_only=true&masjid_id=
func (ctl *UserQuizAttemptsController) List(c *fiber.Ctx) error {
	quizIDStr := strings.TrimSpace(c.Query("quiz_id"))
	studentIDStr := strings.TrimSpace(c.Query("student_id"))
	statusStr := strings.TrimSpace(c.Query("status"))
	activeOnly := strings.EqualFold(strings.TrimSpace(c.Query("active_only")), "true")
	masjidIDStr := strings.TrimSpace(c.Query("masjid_id"))

	q := ctl.DB.Model(&qmodel.UserQuizAttemptModel{})

	// Role-based scoping
	if helperAuth.IsStudent(c) {
		// Student: lock ke masjid aktif + student_id sendiri
		mid, err := helperAuth.GetActiveMasjidID(c)
		if err != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
		}
		sid, err := helperAuth.GetMasjidStudentIDForMasjid(c, mid)
		if err != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
		}
		q = q.Where("user_quiz_attempts_masjid_id = ? AND user_quiz_attempts_student_id = ?", mid, sid)
	} else {
		// Admin/DKM/Teacher: pakai masjid_id jika dikirim, else prefer teacher
		var mid uuid.UUID
		var err error
		if masjidIDStr != "" {
			mid, err = uuid.Parse(masjidIDStr)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id tidak valid")
			}
			if err := ctl.ensureMasjidScope(c, mid); err != nil {
				return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
			}
		} else {
			mid, err = helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
			if err != nil {
				return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
			}
		}
		q = q.Where("user_quiz_attempts_masjid_id = ?", mid)

		// teacher/dkm boleh filter student_id tertentu
		if studentIDStr != "" {
			studentID, err := uuid.Parse(studentIDStr)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "student_id tidak valid")
			}
			q = q.Where("user_quiz_attempts_student_id = ?", studentID)
		}
	}

	// filter quiz
	if quizIDStr != "" {
		quizID, err := uuid.Parse(quizIDStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
		q = q.Where("user_quiz_attempts_quiz_id = ?", quizID)
	}
	// filter status
	if statusStr != "" {
		st := qmodel.UserQuizAttemptStatus(statusStr)
		if !st.Valid() {
			return helper.JsonError(c, fiber.StatusBadRequest, "status tidak valid (in_progress|submitted|finished|abandoned)")
		}
		q = q.Where("user_quiz_attempts_status = ?", st)
	}
	// filter active_only
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