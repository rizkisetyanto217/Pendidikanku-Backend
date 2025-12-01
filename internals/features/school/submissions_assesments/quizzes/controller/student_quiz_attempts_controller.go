// file: internals/features/school/submissions_assesments/quizzes/controller/student_quiz_attempts_controller.go
package controller

import (
	"errors"
	"log"
	"strings"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	qdto "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	qservice "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/service"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

type StudentQuizAttemptsController struct {
	DB        *gorm.DB
	validator *validator.Validate
}

func NewStudentQuizAttemptsController(db *gorm.DB) *StudentQuizAttemptsController {
	return &StudentQuizAttemptsController{
		DB: db,
	}
}

func (ctl *StudentQuizAttemptsController) ensureValidator() {
	if ctl.validator == nil {
		ctl.validator = validator.New()
	}
}

/* =========================================================
   Helpers — scope & relasi
========================================================= */

// Ambil school_id dari quizzes (kolom: quiz_school_id / quiz_id)
func (ctl *StudentQuizAttemptsController) getQuizSchoolID(quizID uuid.UUID) (uuid.UUID, error) {
	if quizID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "quiz_id wajib")
	}

	var schoolIDStr string
	if err := ctl.DB.
		Raw(`SELECT quiz_school_id::text
			   FROM quizzes
			  WHERE quiz_id = ? AND quiz_deleted_at IS NULL`,
			quizID).
		Scan(&schoolIDStr).Error; err != nil {
		return uuid.Nil, err
	}

	if strings.TrimSpace(schoolIDStr) == "" {
		return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "Quiz tidak ditemukan / sudah dihapus")
	}

	mid, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "School ID quiz tidak valid")
	}
	return mid, nil
}

// Validasi status (pakai enum dari model)
func validAttemptStatus(s qmodel.StudentQuizAttemptStatus) bool {
	switch s {
	case qmodel.StudentQuizAttemptInProgress,
		qmodel.StudentQuizAttemptSubmitted,
		qmodel.StudentQuizAttemptFinished,
		qmodel.StudentQuizAttemptAbandoned:
		return true
	default:
		return false
	}
}

// balikin (schoolID, studentID, isStudent)
func (ctl *StudentQuizAttemptsController) resolveScopeForCreate(
	c *fiber.Ctx,
	req *qdto.CreateStudentQuizAttemptRequest,
) (uuid.UUID, uuid.UUID, bool, error) {
	// 1) derive school dari quiz
	qMid, err := ctl.getQuizSchoolID(req.StudentQuizAttemptQuizID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}

	// 2) siswa → wajib terdaftar sebagai student di school quiz & gunakan student_id dari token
	if helperAuth.IsStudent(c) {
		// pastikan benar-benar student di school quiz
		if err := helperAuth.EnsureStudentSchool(c, qMid); err != nil {
			return uuid.Nil, uuid.Nil, true, err
		}
		sid, err := helperAuth.GetSchoolStudentIDForSchool(c, qMid)
		if err != nil {
			return uuid.Nil, uuid.Nil, true, err
		}
		return qMid, sid, true, nil
	}

	// 3) non-student → harus DKM/Teacher (Owner juga diizinkan)
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, qMid); err != nil && !helperAuth.IsOwner(c) {
		return uuid.Nil, uuid.Nil, false, err
	}

	// 4) admin/dkm/teacher/owner → student_id wajib dikirim
	if req.StudentQuizAttemptStudentID == nil || *req.StudentQuizAttemptStudentID == uuid.Nil {
		return uuid.Nil, uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "student_quiz_attempt_student_id wajib untuk admin/dkm/teacher")
	}

	// validasi student milik school tsb
	var ok bool
	if err := ctl.DB.Raw(`
		SELECT EXISTS(
			SELECT 1 FROM school_students
			 WHERE school_student_id = ? AND school_id = ?
		)
	`, *req.StudentQuizAttemptStudentID, qMid).Scan(&ok).Error; err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}
	if !ok {
		return uuid.Nil, uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "student tidak terdaftar di school quiz")
	}

	return qMid, *req.StudentQuizAttemptStudentID, false, nil
}

/* =========================================================
   Handlers
========================================================= */
// POST /student-quiz-attempts
// Bisa:
// - hanya bikin attempt summary kosong (tanpa jawaban) → items = []
// - atau create/reuse + submit attempt (pertama / berikutnya) → items diisi
func (ctl *StudentQuizAttemptsController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	var req qdto.CreateStudentQuizAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[StudentQuizAttemptsController] BodyParser error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	log.Printf("[StudentQuizAttemptsController] Create called. quiz_id=%s started_at=%v finished_at=%v items=%d",
		req.StudentQuizAttemptQuizID, req.AttemptStartedAt, req.AttemptFinishedAt, len(req.Items))

	// Validasi DTO
	if err := ctl.validator.Struct(&req); err != nil {
		log.Printf("[StudentQuizAttemptsController] Validation error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}

	// Resolve scope (school + student) dari quiz & token
	mid, sid, _, err := ctl.resolveScopeForCreate(c, &req)
	if err != nil {
		log.Printf("[StudentQuizAttemptsController] resolveScopeForCreate error: %v", err)
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	log.Printf("[StudentQuizAttemptsController] Scope resolved. school_id=%s student_id=%s", mid, sid)

	// Override anti-spoof
	req.StudentQuizAttemptSchoolID = &mid
	req.StudentQuizAttemptStudentID = &sid

	// =========================================
	// 1) Cek dulu: sudah ada summary row atau belum?
	//    (1 row = 1 student × 1 quiz)
	// =========================================
	var existing qmodel.StudentQuizAttemptModel
	err = ctl.DB.WithContext(c.Context()).
		Where(`
			student_quiz_attempt_school_id = ?
			AND student_quiz_attempt_quiz_id = ?
			AND student_quiz_attempt_student_id = ?
		`, mid, req.StudentQuizAttemptQuizID, sid).
		First(&existing).Error

	isNew := false
	var m *qmodel.StudentQuizAttemptModel

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Belum ada → bikin baru
			log.Printf("[StudentQuizAttemptsController] No existing attempt summary. Creating new row...")
			isNew = true
			m = req.ToModel()

			now := time.Now().UTC()
			// Kalau FE kirim AttemptStartedAt → pakai itu, kalau tidak pakai now
			if req.AttemptStartedAt != nil {
				m.StudentQuizAttemptStartedAt = req.AttemptStartedAt
			} else {
				m.StudentQuizAttemptStartedAt = &now
			}

			// Pastikan history dan status aman
			if len(m.StudentQuizAttemptHistory) == 0 {
				m.StudentQuizAttemptHistory = datatypes.JSON([]byte("[]"))
			}
			if !validAttemptStatus(m.StudentQuizAttemptStatus) {
				m.StudentQuizAttemptStatus = qmodel.StudentQuizAttemptInProgress
			}

			if err := ctl.DB.Create(m).Error; err != nil {
				log.Printf("[StudentQuizAttemptsController] DB Create error: %v", err)
				if isUniqueViolation(err) {
					// Kalau kejadian race condition aneh, balikin conflict
					return helper.JsonError(c, fiber.StatusConflict, "Duplikat / melanggar unique index")
				}
				if isCheckViolation(err) {
					return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar CHECK constraint")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan attempt")
			}

			log.Printf("[StudentQuizAttemptsController] New summary created. attempt_id=%s", m.StudentQuizAttemptID)
		} else {
			// Error DB lain
			log.Printf("[StudentQuizAttemptsController] DB error when checking existing attempt: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek attempt existing")
		}
	} else {
		// Sudah ada summary row → pakai existing
		m = &existing
		log.Printf("[StudentQuizAttemptsController] Existing summary found. attempt_id=%s count=%d",
			m.StudentQuizAttemptID, m.StudentQuizAttemptCount)

		// Optional: kalau mau update started_at kalau belum pernah di-set
		if m.StudentQuizAttemptStartedAt == nil && req.AttemptStartedAt != nil {
			m.StudentQuizAttemptStartedAt = req.AttemptStartedAt
			if err := ctl.DB.Save(m).Error; err != nil {
				log.Printf("[StudentQuizAttemptsController] Failed to update started_at on existing attempt: %v", err)
			}
		}
	}

	// =========================================
	// 2) Kalau items kosong → hanya memastikan row ada (mulai attempt)
	// =========================================
	if len(req.Items) == 0 {
		msg := "Berhasil memulai attempt"
		if !isNew {
			msg = "Attempt sudah ada, menggunakan attempt yang sama"
		}
		log.Printf("[StudentQuizAttemptsController] No items submitted. Returning summary only. isNew=%v attempt_id=%s",
			isNew, m.StudentQuizAttemptID)
		return helper.JsonCreated(c, msg, qdto.FromModelStudentQuizAttempt(m))
	}

	// =========================================
	// 3) MODE: create / reuse + submit attempt
	//    → kita konversi items → map answers,
	//      lalu panggil service.SubmitAttempt (append history)
	// =========================================

	answers := make(map[uuid.UUID]string, len(req.Items))
	for _, it := range req.Items {
		var v string
		if it.AnswerSingle != nil {
			v = strings.TrimSpace(*it.AnswerSingle)
		}
		if it.AnswerEssay != nil && v == "" {
			v = strings.TrimSpace(*it.AnswerEssay)
		}
		if v != "" {
			answers[it.QuizQuestionID] = v
		}
	}

	log.Printf("[StudentQuizAttemptsController] Submitting attempt. attempt_id=%s answers_count=%d finished_at=%v",
		m.StudentQuizAttemptID, len(answers), req.AttemptFinishedAt)

	svc := qservice.NewStudentQuizAttemptService(ctl.DB)
	submitIn := &qservice.SubmitQuizAttemptInput{
		AttemptID:  m.StudentQuizAttemptID,
		FinishedAt: req.AttemptFinishedAt, // boleh nil → service pakai now
		Answers:    answers,
	}

	finalAttempt, err := svc.SubmitAttempt(c.Context(), submitIn)
	if err != nil {
		log.Printf("[StudentQuizAttemptsController] SubmitAttempt error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memproses submit attempt")
	}

	log.Printf("[StudentQuizAttemptsController] SubmitAttempt success. attempt_id=%s total_history=%d last_percent=%v",
		finalAttempt.StudentQuizAttemptID,
		finalAttempt.StudentQuizAttemptCount,
		func() *float64 {
			return finalAttempt.StudentQuizAttemptLastPercent
		}(),
	)

	msg := "Berhasil membuat dan mensubmit attempt"
	if !isNew {
		msg = "Berhasil mensubmit attempt baru ke history"
	}
	return helper.JsonCreated(c, msg, qdto.FromModelStudentQuizAttempt(finalAttempt))
}

// PATCH /student-quiz-attempts/:id
func (ctl *StudentQuizAttemptsController) Patch(c *fiber.Ctx) error {
	ctl.ensureValidator()

	// Student dilarang patch (patch ini buat admin/dkm/teacher / internal)
	if helperAuth.IsStudent(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya admin/dkm/teacher yang diizinkan mengubah attempt")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var m qmodel.StudentQuizAttemptModel
	if err := ctl.DB.First(&m, "student_quiz_attempt_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Attempt tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// scope: DKM/Teacher (Owner juga diizinkan)
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, m.StudentQuizAttemptSchoolID); err != nil && !helperAuth.IsOwner(c) {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return err
	}

	var req qdto.UpdateStudentQuizAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}

	if err := req.ApplyToModel(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Patch tidak valid")
	}

	// Jaga-jaga: status tetap valid, history nggak null
	if !validAttemptStatus(m.StudentQuizAttemptStatus) {
		m.StudentQuizAttemptStatus = qmodel.StudentQuizAttemptInProgress
	}
	if len(m.StudentQuizAttemptHistory) == 0 {
		m.StudentQuizAttemptHistory = datatypes.JSON([]byte("[]"))
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

	return helper.JsonUpdated(c, "Berhasil memperbarui attempt", qdto.FromModelStudentQuizAttempt(&m))
}

// DELETE /student-quiz-attempts/:id
func (ctl *StudentQuizAttemptsController) Delete(c *fiber.Ctx) error {
	// Student dilarang delete
	if helperAuth.IsStudent(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya admin/dkm/teacher yang diizinkan menghapus attempt")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var m qmodel.StudentQuizAttemptModel
	if err := ctl.DB.
		Select("student_quiz_attempt_id, student_quiz_attempt_school_id").
		First(&m, "student_quiz_attempt_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Attempt tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// scope: DKM/Teacher (Owner juga diizinkan)
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, m.StudentQuizAttemptSchoolID); err != nil && !helperAuth.IsOwner(c) {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return err
	}

	if err := ctl.DB.Delete(&qmodel.StudentQuizAttemptModel{}, "student_quiz_attempt_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus")
	}
	return helper.JsonDeleted(c, "Berhasil menghapus", fiber.Map{"deleted_id": id})
}

// GET /student-quiz-attempts?quiz_id=&student_id=&status=&active_only=true&school_id=&all=1
func (ctl *StudentQuizAttemptsController) List(c *fiber.Ctx) error {
	quizIDStr := strings.TrimSpace(c.Query("quiz_id"))
	studentIDStr := strings.TrimSpace(c.Query("student_id"))
	statusStr := strings.TrimSpace(c.Query("status"))
	activeOnly := strings.EqualFold(strings.TrimSpace(c.Query("active_only")), "true")
	schoolIDStr := strings.TrimSpace(c.Query("school_id"))
	all := parseBool(c.Query("all"))

	q := ctl.DB.WithContext(c.Context()).Model(&qmodel.StudentQuizAttemptModel{})

	// ===== Role-based scoping =====
	if helperAuth.IsStudent(c) {
		// Student: lock ke school aktif + student_id sendiri
		mid, err := helperAuth.GetActiveSchoolID(c)
		if err != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
		}
		if err := helperAuth.EnsureStudentSchool(c, mid); err != nil {
			return err
		}
		sid, err := helperAuth.GetSchoolStudentIDForSchool(c, mid)
		if err != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
		}
		q = q.Where("student_quiz_attempt_school_id = ? AND student_quiz_attempt_student_id = ?", mid, sid)
	} else {
		// Admin/DKM/Teacher (Owner juga diizinkan)
		var mid uuid.UUID
		var err error
		if schoolIDStr != "" {
			mid, err = uuid.Parse(schoolIDStr)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak valid")
			}
			if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil && !helperAuth.IsOwner(c) {
				if fe, ok := err.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return err
			}
		} else {
			mid, err = helperAuth.GetSchoolIDFromTokenPreferTeacher(c)
			if err != nil {
				return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
			}
			if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil && !helperAuth.IsOwner(c) {
				if fe, ok := err.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return err
			}
		}
		q = q.Where("student_quiz_attempt_school_id = ?", mid)

		// teacher/dkm boleh filter student_id tertentu
		if studentIDStr != "" {
			studentID, err := uuid.Parse(studentIDStr)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "student_id tidak valid")
			}
			q = q.Where("student_quiz_attempt_student_id = ?", studentID)
		}
	}

	// ===== Filters =====
	if quizIDStr != "" {
		quizID, err := uuid.Parse(quizIDStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
		q = q.Where("student_quiz_attempt_quiz_id = ?", quizID)
	}

	if statusStr != "" {
		st := qmodel.StudentQuizAttemptStatus(statusStr)
		if !validAttemptStatus(st) {
			return helper.JsonError(c, fiber.StatusBadRequest, "status tidak valid (in_progress|submitted|finished|abandoned)")
		}
		q = q.Where("student_quiz_attempt_status = ?", st)
	}

	if activeOnly {
		q = q.Where("student_quiz_attempt_status IN (?)",
			[]string{
				string(qmodel.StudentQuizAttemptInProgress),
				string(qmodel.StudentQuizAttemptSubmitted),
			})
	}

	// ===== Count total (sebelum limit/offset) =====
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ===== Paging & Sort =====
	pg := helper.ResolvePaging(c, 20, 100) // default per_page=20, max=100
	q = q.Order("student_quiz_attempt_started_at DESC")
	if !all {
		q = q.Offset(pg.Offset).Limit(pg.Limit)
	}

	// ===== Fetch =====
	var rows []qmodel.StudentQuizAttemptModel
	if err := q.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== Build pagination object =====
	var pagination helper.Pagination
	if all {
		per := int(total)
		if per == 0 {
			per = 1
		}
		pagination = helper.BuildPaginationFromPage(total, 1, per)
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	// ===== JSON =====
	return helper.JsonList(c, "OK", qdto.FromModelsStudentQuizAttempts(rows), pagination)
}

// util kecil
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
