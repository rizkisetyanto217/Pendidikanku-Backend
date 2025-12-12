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

	studentModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
)

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unique")
}

func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "check constraint")
}

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

// Isi snapshot/caches dari school_students
func (ctl *StudentQuizAttemptsController) fillStudentSnapshots(
	c *fiber.Ctx,
	schoolID uuid.UUID,
	studentID uuid.UUID,
	m *qmodel.StudentQuizAttemptModel,
) {
	var stu studentModel.SchoolStudentModel

	if err := ctl.DB.WithContext(c.Context()).
		Select(
			"school_student_code",
			"school_student_user_profile_name_cache",
			"school_student_user_profile_avatar_url_cache",
			"school_student_user_profile_whatsapp_url_cache",
			"school_student_user_profile_gender_cache",
		).
		Where("school_student_id = ? AND school_student_school_id = ?", studentID, schoolID).
		First(&stu).Error; err != nil {

		log.Printf("[StudentQuizAttemptsController] fillStudentSnapshots: gagal load school_student: %v", err)
		return
	}

	m.StudentQuizAttemptSchoolStudentCodeCache = stu.SchoolStudentCode
	m.StudentQuizAttemptUserProfileNameSnapshot = stu.SchoolStudentUserProfileNameCache
	m.StudentQuizAttemptUserProfileAvatarURLSnapshot = stu.SchoolStudentUserProfileAvatarURLCache
	m.StudentQuizAttemptUserProfileWhatsappURLSnapshot = stu.SchoolStudentUserProfileWhatsappURLCache
	m.StudentQuizAttemptUserProfileGenderSnapshot = stu.SchoolStudentUserProfileGenderCache
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
			 WHERE school_student_id = ?
			   AND school_student_school_id = ?
		)
	`, *req.StudentQuizAttemptStudentID, qMid).Scan(&ok).Error; err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}
	if !ok {
		return uuid.Nil, uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "student tidak terdaftar di school quiz")
	}

	return qMid, *req.StudentQuizAttemptStudentID, false, nil
}

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

	// DEBUG: cek apa BodyParser ngisi AnswerSingle / AnswerEssay
	for i, it := range req.Items {
		log.Printf("[StudentQuizAttemptsController][DEBUG] item[%d] qid=%s answer_single=%v answer_essay=%v",
			i,
			it.QuizQuestionID,
			it.AnswerSingle,
			it.AnswerEssay,
		)
	}

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

	// Override anti-spoof dari token
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

			// Isi snapshot dari school_students
			ctl.fillStudentSnapshots(c, mid, sid, m)

			if err := ctl.DB.Create(m).Error; err != nil {
				log.Printf("[StudentQuizAttemptsController] DB Create error: %v", err)
				if isUniqueViolation(err) {
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
		return helper.JsonCreated(c, msg, qdto.FromModelStudentQuizAttemptWithCtx(c, m))
	}

	// =========================================
	// 3) MODE: create / reuse + submit attempt
	//    → konversi items → map answers,
	//      lalu panggil service.SubmitAttempt (append history)
	// =========================================

	answers := make(map[uuid.UUID]string, len(req.Items))

	for _, it := range req.Items {
		var v string

		// SINGLE
		if it.AnswerSingle != nil {
			v = strings.TrimSpace(*it.AnswerSingle)
		}

		// ESSAY (fallback kalau single kosong)
		if it.AnswerEssay != nil && v == "" {
			v = strings.TrimSpace(*it.AnswerEssay)
		}

		if v != "" {
			answers[it.QuizQuestionID] = v
		}
	}

	if len(answers) == 0 {
		log.Printf("[StudentQuizAttemptsController][WARN] answers map kosong padahal items=%d. Cek apakah FE mengirim field 'answer_single' / 'answer_essay' sesuai JSON tag.", len(req.Items))
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
		func() interface{} {
			if finalAttempt.StudentQuizAttemptLastPercent == nil {
				return "nil"
			}
			return *finalAttempt.StudentQuizAttemptLastPercent
		}(),
	)

	msg := "Berhasil membuat dan mensubmit attempt"
	if !isNew {
		msg = "Berhasil mensubmit attempt baru ke history"
	}
	return helper.JsonCreated(c, msg, qdto.FromModelStudentQuizAttemptWithCtx(c, finalAttempt))
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

	return helper.JsonUpdated(c, "Berhasil memperbarui attempt", qdto.FromModelStudentQuizAttemptWithCtx(c, &m))
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



// util kecil
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
