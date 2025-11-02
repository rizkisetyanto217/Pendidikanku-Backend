package controller

import (
	"time"

	questionModel "schoolku_backend/internals/features/schools/lecture_sessions/questions/model"
	resp "schoolku_backend/internals/helpers"

	lectureSessionModel "schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	"schoolku_backend/internals/features/schools/lecture_sessions/quizzes/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/quizzes/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// =============================
// 🌐 Get Quiz By School Slug (Public, full GORM)
// =============================
func (ctrl *LectureSessionsQuizController) GetQuizzesBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Slug school diperlukan")
	}

	// Cari school berdasarkan slug
	var school struct {
		SchoolID string `gorm:"column:school_id"`
	}
	if err := ctrl.DB.WithContext(c.Context()).
		Table("schools").
		Select("school_id").
		Where("school_slug = ?", slug).
		First(&school).Error; err != nil || school.SchoolID == "" {
		return resp.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	// Ambil quiz berdasarkan school_id
	var quizzes []model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_school_id = ?", school.SchoolID).
		Find(&quizzes).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil quiz")
	}

	// Konversi ke DTO
	result := make([]dto.LectureSessionsQuizDTO, 0, len(quizzes))
	for _, quiz := range quizzes {
		result = append(result, dto.ToLectureSessionsQuizDTO(quiz))
	}

	return resp.JsonOK(c, "OK", result)
}

// ✅ GET /api/a/lecture-sessions-quiz/by-session/:id
func (ctrl *LectureSessionsQuizController) GetByLectureSessionID(c *fiber.Ctx) error {
	lectureSessionID := c.Params("id")
	if lectureSessionID == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Lecture session ID tidak ditemukan di URL")
	}
	if _, err := uuid.Parse(lectureSessionID); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Lecture session ID tidak valid")
	}

	// Ambil satu quiz berdasarkan lecture_session_id
	var quiz model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_lecture_session_id = ?", lectureSessionID).
		First(&quiz).Error; err != nil {
		return resp.JsonError(c, fiber.StatusNotFound, "Quiz untuk sesi kajian ini tidak ditemukan")
	}

	// Ambil soal-soal terkait quiz
	var questions []questionModel.LectureSessionsQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_question_quiz_id = ?", quiz.LectureSessionsQuizID).
		Order("lecture_sessions_question_created_at ASC").
		Find(&questions).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil soal-soal quiz")
	}

	return resp.JsonOK(c, "Quiz dan soal berhasil ditemukan", fiber.Map{
		"quiz":      quiz,
		"questions": questions,
	})
}

// ✅ GET /api/a/lecture-sessions-quiz/by-lecture-slug/:lecture_slug
// Opsional: ?only_with_quiz=true  (hanya sesi yang punya quiz)
func (ctrl *LectureSessionsQuizController) GetQuizzesByLectureSlug(c *fiber.Ctx) error {
	lectureSlug := c.Params("lecture_slug")
	if lectureSlug == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "lecture_slug wajib diisi")
	}

	// user_id opsional (middleware / header)
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		userID = c.Get("X-User-Id")
	}
	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
		}
	}
	onlyWithQuiz := c.Query("only_with_quiz") == "true"

	// 1) resolve lecture_slug -> lecture_id
	var lectureID string
	if err := ctrl.DB.WithContext(c.Context()).
		Table("lectures").Select("lecture_id").
		Where("lecture_slug = ?", lectureSlug).
		Scan(&lectureID).Error; err != nil || lectureID == "" {
		return resp.JsonError(c, fiber.StatusNotFound, "Lecture tidak ditemukan")
	}

	// 2) sesi + quiz + (nilai quiz user jika ada)
	type Row struct {
		LectureSessionID        string     `json:"lecture_session_id"`
		LectureSessionSlug      string     `json:"lecture_session_slug"`
		LectureSessionTitle     string     `json:"lecture_session_title"`
		LectureSessionStartTime *time.Time `json:"lecture_session_start_time"`

		LectureSessionsQuizID          *string `json:"lecture_sessions_quiz_id,omitempty"`
		LectureSessionsQuizTitle       *string `json:"lecture_sessions_quiz_title,omitempty"`
		LectureSessionsQuizDescription *string `json:"lecture_sessions_quiz_description,omitempty"`

		UserQuizGradeResult     *float64 `json:"user_quiz_grade_result,omitempty"`
		UserQuizAttemptCount    *int     `json:"user_quiz_attempt_count,omitempty"`
		UserQuizDurationSeconds *int     `json:"user_quiz_duration_seconds,omitempty"`
	}

	baseSelect := `
		ls.lecture_session_id,
		ls.lecture_session_slug,
		ls.lecture_session_title,
		ls.lecture_session_start_time,
		lsq.lecture_sessions_quiz_id,
		lsq.lecture_sessions_quiz_title,
		lsq.lecture_sessions_quiz_description
	`

	q := ctrl.DB.WithContext(c.Context()).
		Table("lecture_sessions AS ls").
		Select(baseSelect).
		Joins(`LEFT JOIN lecture_sessions_quiz AS lsq
		       ON lsq.lecture_sessions_quiz_lecture_session_id = ls.lecture_session_id`).
		Where("ls.lecture_session_lecture_id = ? AND ls.lecture_session_deleted_at IS NULL", lectureID)

	if onlyWithQuiz {
		q = q.Where("lsq.lecture_sessions_quiz_id IS NOT NULL")
	}

	// join nilai QUIZ kalau user_id ada
	if userID != "" {
		q = q.Select(baseSelect+`,
			ulsq.user_lecture_sessions_quiz_grade_result     AS user_quiz_grade_result,
			ulsq.user_lecture_sessions_quiz_attempt_count    AS user_quiz_attempt_count,
			ulsq.user_lecture_sessions_quiz_duration_seconds AS user_quiz_duration_seconds
		`).Joins(`LEFT JOIN user_lecture_sessions_quiz AS ulsq
		          ON ulsq.user_lecture_sessions_quiz_quiz_id = lsq.lecture_sessions_quiz_id
		          AND ulsq.user_lecture_sessions_quiz_user_id = ?`, userID)
	}

	q = q.Order("ls.lecture_session_start_time ASC NULLS LAST")

	var rows []Row
	if err := q.Scan(&rows).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil quiz berdasarkan lecture_slug")
	}

	return resp.JsonOK(c, "Berhasil mengambil quiz berdasarkan lecture_slug", rows)
}

// ✅ GET /api/a/lecture-sessions-quiz/by-session-slug/:slug
func (ctrl *LectureSessionsQuizController) GetByLectureSessionSlug(c *fiber.Ctx) error {
	lectureSessionSlug := c.Params("slug")
	if lectureSessionSlug == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Slug sesi kajian tidak ditemukan di URL")
	}

	// Ambil sesi kajian berdasarkan slug
	var session lectureSessionModel.LectureSessionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_session_slug = ?", lectureSessionSlug).
		First(&session).Error; err != nil {
		return resp.JsonError(c, fiber.StatusNotFound, "Sesi kajian dengan slug tersebut tidak ditemukan")
	}

	// Ambil quiz berdasarkan lecture_session_id dari sesi yang ditemukan
	var quiz model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_lecture_session_id = ?", session.LectureSessionID).
		First(&quiz).Error; err != nil {
		return resp.JsonError(c, fiber.StatusNotFound, "Quiz untuk sesi kajian ini tidak ditemukan")
	}

	// Ambil soal-soal terkait quiz
	var questions []questionModel.LectureSessionsQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_question_quiz_id = ?", quiz.LectureSessionsQuizID).
		Order("lecture_sessions_question_created_at ASC").
		Find(&questions).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil soal-soal quiz")
	}

	return resp.JsonOK(c, "Quiz dan soal berhasil ditemukan", fiber.Map{
		"quiz":      quiz,
		"questions": questions,
	})
}

// ✅ GET /api/a/lecture-sessions-quiz/by-lecture/:id
func (ctrl *LectureSessionsQuizController) GetByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	if lectureID == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Lecture ID tidak ditemukan di URL")
	}
	if _, err := uuid.Parse(lectureID); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Lecture ID tidak valid")
	}

	// Ambil semua sesi kajian dari lecture ini
	var sessionIDs []string
	if err := ctrl.DB.WithContext(c.Context()).
		Table("lecture_sessions").
		Where("lecture_session_lecture_id = ?", lectureID).
		Pluck("lecture_session_id", &sessionIDs).Error; err != nil || len(sessionIDs) == 0 {
		return resp.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan untuk lecture ini")
	}

	// Ambil semua quiz dari sesi-sesi tersebut
	var quizzes []model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_lecture_session_id IN ?", sessionIDs).
		Find(&quizzes).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil quiz dari lecture")
	}

	return resp.JsonOK(c, "Quiz berhasil diambil berdasarkan lecture", quizzes)
}
