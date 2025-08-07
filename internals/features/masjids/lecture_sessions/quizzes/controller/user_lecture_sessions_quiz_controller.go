package controller

import (
	"fmt"
	"log"
	"masjidku_backend/internals/constants"
	modelUserLectureSession "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/model"
	modelLecture "masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLectureSessionsQuizController struct {
	DB *gorm.DB
}

func NewUserLectureSessionsQuizController(db *gorm.DB) *UserLectureSessionsQuizController {
	return &UserLectureSessionsQuizController{DB: db}
}


func (ctrl *UserLectureSessionsQuizController) CreateUserLectureSessionsQuiz(c *fiber.Ctx) error {
	var body dto.CreateUserLectureSessionsQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validate.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	userUUID := helper.GetUserUUID(c)
	userID := userUUID.String()
	isAnonymous := userUUID == constants.DummyUserID

	// ✅ Ambil lecture_session_slug dari route param
	lectureSessionSlug := c.Params("lecture_session_slug")
	if lectureSessionSlug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Lecture session slug is required")
	}

	// ✅ Ambil lecture_session_id dan masjid_id dari slug
	var session struct {
		ID        string `gorm:"column:lecture_session_id"`
		MasjidID  string `gorm:"column:lecture_session_masjid_id"`
	}
	if err := ctrl.DB.
		Table("lecture_sessions").
		Select("lecture_session_id, lecture_session_masjid_id").
		Where("lecture_session_slug = ?", lectureSessionSlug).
		First(&session).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Lecture session not found for given slug")
	}

	// === ✅ USER LOGIN ===
	if !isAnonymous {
		var existing model.UserLectureSessionsQuizModel
		err := ctrl.DB.Where("user_lecture_sessions_quiz_user_id = ? AND user_lecture_sessions_quiz_quiz_id = ?", userID, body.UserLectureSessionsQuizQuizID).
			First(&existing).Error

		if err == nil {
			existing.UserLectureSessionsQuizAttemptCount += 1
			if body.UserLectureSessionsQuizGrade > existing.UserLectureSessionsQuizGrade {
				existing.UserLectureSessionsQuizGrade = body.UserLectureSessionsQuizGrade
			}
			if err := ctrl.DB.Save(&existing).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update quiz result")
			}
			_ = ctrl.RecalculateLectureSessionsGradeBySlug(userID, session.ID, session.MasjidID)
			return c.Status(fiber.StatusOK).JSON(dto.ToUserLectureSessionsQuizDTO(existing))
		}
	}

	newData := model.UserLectureSessionsQuizModel{
		UserLectureSessionsQuizGrade:            body.UserLectureSessionsQuizGrade,
		UserLectureSessionsQuizQuizID:           body.UserLectureSessionsQuizQuizID,
		UserLectureSessionsQuizUserID:           userID,
		UserLectureSessionsQuizMasjidID:         session.MasjidID,
		UserLectureSessionsQuizAttemptCount:     1,
		UserLectureSessionsQuizDurationSeconds:  body.UserLectureSessionsQuizDurationSeconds,
		UserLectureSessionsQuizLectureSessionID: session.ID,
	}

	if err := ctrl.DB.Create(&newData).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to save quiz result")
	}

	if !isAnonymous {
		_ = ctrl.RecalculateLectureSessionsGradeBySlug(userID, session.ID, session.MasjidID)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureSessionsQuizDTO(newData))
}



func (ctrl *UserLectureSessionsQuizController) RecalculateLectureSessionsGradeBySlug(userID, lectureSessionSlug, masjidID string) error {
	// ✅ Ambil lecture_session_id & lecture_id dari slug
	var session struct {
		ID        string
		LectureID string
	}
	if err := ctrl.DB.
		Table("lecture_sessions").
		Select("lecture_session_id AS id, lecture_session_lecture_id AS lecture_id").
		Where("lecture_session_slug = ?", lectureSessionSlug).
		Scan(&session).Error; err != nil || session.ID == "" || session.LectureID == "" {
		return fmt.Errorf("failed to find session by slug: %w", err)
	}

	// ✅ Hitung rata-rata nilai quiz user di sesi
	var avg float64
	if err := ctrl.DB.
		Table("user_lecture_sessions_quiz").
		Select("AVG(user_lecture_sessions_quiz_grade_result)").
		Where("user_lecture_sessions_quiz_user_id = ? AND user_lecture_sessions_quiz_lecture_session_id = ?", userID, session.ID).
		Scan(&avg).Error; err != nil {
		return fmt.Errorf("failed to calculate quiz average: %w", err)
	}

	// ✅ Cek apakah user_lecture_session sudah ada
	var existing modelUserLectureSession.UserLectureSessionModel
	err := ctrl.DB.
		Where("user_lecture_session_user_id = ? AND user_lecture_session_lecture_session_id = ?", userID, session.ID).
		First(&existing).Error

	if err != nil {
		// ❗ Belum ada → insert baru
		newData := modelUserLectureSession.UserLectureSessionModel{
			UserLectureSessionUserID:           userID,
			UserLectureSessionLectureSessionID: session.ID,
			UserLectureSessionLectureID:        session.LectureID,
			UserLectureSessionMasjidID:         masjidID,
			UserLectureSessionGradeResult:      &avg,
		}
		if err := ctrl.DB.Create(&newData).Error; err != nil {
			return fmt.Errorf("failed to create user_lecture_session: %w", err)
		}
	} else {
		// ✅ Update nilai saja
		if err := ctrl.DB.
			Model(&existing).
			Update("user_lecture_session_grade_result", avg).
			Error; err != nil {
			return fmt.Errorf("failed to update grade result: %w", err)
		}
	}

	// ✅ Update progres user ke tabel user_lectures
	return ctrl.UpdateUserLectureProgressBySlug(userID, session.LectureID, masjidID)
}


func (ctrl *UserLectureSessionsQuizController) UpdateUserLectureProgressBySlug(userID, lectureSlug, masjidID string) error {
	// ✅ Cari lecture_id dari slug
	var lecture struct {
		ID string
	}
	if err := ctrl.DB.
		Table("lectures").
		Select("lecture_id AS id").
		Where("lecture_slug = ?", lectureSlug).
		Scan(&lecture).Error; err != nil || lecture.ID == "" {
		return fmt.Errorf("failed to find lecture by slug: %w", err)
	}

	// ✅ Hitung rata-rata nilai semua sesi user di satu lecture
	var avg float64
	err := ctrl.DB.
		Table("user_lecture_sessions").
		Select("AVG(user_lecture_session_grade_result)").
		Joins("JOIN lecture_sessions ON user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id").
		Where("user_lecture_session_user_id = ? AND lecture_sessions.lecture_session_lecture_id = ?", userID, lecture.ID).
		Scan(&avg).Error
	if err != nil {
		return fmt.Errorf("failed to calculate lecture avg: %w", err)
	}

	// ✅ Hitung jumlah sesi yang sudah selesai
	var count int64
	err = ctrl.DB.
		Table("user_lecture_sessions").
		Joins("JOIN lecture_sessions ON user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id").
		Where("user_lecture_session_user_id = ? AND lecture_sessions.lecture_session_lecture_id = ? AND user_lecture_session_grade_result IS NOT NULL", userID, lecture.ID).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to count completed sessions: %w", err)
	}

	// ✅ Cek apakah user_lecture sudah ada
	var existing modelLecture.UserLectureModel
	err = ctrl.DB.
		Where("user_lecture_user_id = ? AND user_lecture_lecture_id = ?", userID, lecture.ID).
		First(&existing).Error

	if err != nil {
		// ❗ Insert baru jika belum ada
		newData := modelLecture.UserLectureModel{
			UserLectureUserID:                 uuid.MustParse(userID),
			UserLectureLectureID:              uuid.MustParse(lecture.ID),
			UserLectureMasjidID:               uuid.MustParse(masjidID),
			UserLectureGradeResult:            intPtr(int(avg)),
			UserLectureTotalCompletedSessions: int(count),
		}
		return ctrl.DB.Create(&newData).Error
	}

	// ✅ Update jika sudah ada
	return ctrl.DB.
		Model(&existing).
		Updates(map[string]interface{}{
			"user_lecture_grade_result":             int(avg),
			"user_lecture_total_completed_sessions": int(count),
		}).Error
}

func intPtr(v int) *int {
	return &v
}




// =============================
// 📄 Get All Quiz Results
// =============================
func (ctrl *UserLectureSessionsQuizController) GetAllUserLectureSessionsQuiz(c *fiber.Ctx) error {
	var results []model.UserLectureSessionsQuizModel

	if err := ctrl.DB.Find(&results).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch quiz results")
	}

	var dtos []dto.UserLectureSessionsQuizDTO
	for _, r := range results {
		dtos = append(dtos, dto.ToUserLectureSessionsQuizDTO(r))
	}

	return c.JSON(dtos)
}

// =============================
// 🔍 Get By Quiz ID or User ID
// =============================
func (ctrl *UserLectureSessionsQuizController) GetUserLectureSessionsQuizFiltered(c *fiber.Ctx) error {
	quizID := c.Query("quiz_id")
	userID := c.Query("user_id")

	query := ctrl.DB.Model(&model.UserLectureSessionsQuizModel{})
	if quizID != "" {
		query = query.Where("user_lecture_sessions_quiz_quiz_id = ?", quizID)
	}
	if userID != "" {
		query = query.Where("user_lecture_sessions_quiz_user_id = ?", userID)
	}

	var results []model.UserLectureSessionsQuizModel
	if err := query.Find(&results).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch filtered quiz results")
	}

	var dtos []dto.UserLectureSessionsQuizDTO
	for _, r := range results {
		dtos = append(dtos, dto.ToUserLectureSessionsQuizDTO(r))
	}

	return c.JSON(dtos)
}

// =============================
// ❌ Delete Quiz Result by ID
// =============================
func (ctrl *UserLectureSessionsQuizController) DeleteUserLectureSessionsQuizByID(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.UserLectureSessionsQuizModel{}, "user_lecture_sessions_quiz_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete quiz result")
	}

	return c.JSON(fiber.Map{
		"message": "Quiz result deleted successfully",
	})
}

func (ctrl *UserLectureSessionsQuizController) GetUserQuizWithDetail(c *fiber.Ctx) error {
	start := time.Now()
	log.Println("[INFO] GetUserQuizWithDetail called")

	// ========== 1. Ambil User ID jika login ==========
	userID := ""
	userIDRaw := c.Locals("user_id")
	if userIDRaw != nil {
		userID = userIDRaw.(string)
		log.Printf("[SUCCESS] User ID stored: %s\n", userID)
	} else {
		log.Println("[INFO] Tidak ada user_id (anonim), ambil kuis tanpa progress")
	}

	// ========== 2. Ambil Query Params ==========
	lectureID := c.Query("lecture_id")
	lectureSessionID := c.Query("lecture_session_id")
	log.Printf("[DEBUG] Query Params => lecture_id: %s | lecture_session_id: %s\n", lectureID, lectureSessionID)

	if lectureID == "" && lectureSessionID == "" {
		log.Println("[ERROR] Minimal salah satu parameter lecture_id atau lecture_session_id harus diisi")
		return fiber.NewError(fiber.StatusBadRequest, "Minimal salah satu parameter: lecture_id atau lecture_session_id harus diisi")
	}

	// ========== 3. Struct Hasil ==========
	type UserQuizWithDetail struct {
		LectureSessionsQuizID               string    `json:"lecture_sessions_quiz_id" gorm:"column:lecture_sessions_quiz_id"`
		LectureSessionsQuizTitle            string    `json:"lecture_sessions_quiz_title" gorm:"column:lecture_sessions_quiz_title"`
		LectureSessionsQuizDescription      string    `json:"lecture_sessions_quiz_description" gorm:"column:lecture_sessions_quiz_description"`
		LectureSessionsQuizLectureSessionID string    `json:"lecture_sessions_quiz_lecture_session_id" gorm:"column:lecture_sessions_quiz_lecture_session_id"`
		LectureSessionsQuizCreatedAt        time.Time `json:"lecture_sessions_quiz_created_at" gorm:"column:lecture_sessions_quiz_created_at"`

		UserLectureSessionsQuizID        *string    `json:"user_lecture_sessions_quiz_id,omitempty" gorm:"column:user_lecture_sessions_quiz_id"`
		UserLectureSessionsQuizGrade     *float64   `json:"user_lecture_sessions_quiz_grade_result,omitempty" gorm:"column:user_lecture_sessions_quiz_grade_result"`
		UserLectureSessionsQuizUserID    *string    `json:"user_lecture_sessions_quiz_user_id,omitempty" gorm:"column:user_lecture_sessions_quiz_user_id"`
		UserLectureSessionsQuizCreatedAt *time.Time `json:"user_lecture_sessions_quiz_created_at,omitempty" gorm:"column:user_lecture_sessions_quiz_created_at"`
	}

	var results []UserQuizWithDetail

	// ========== 4. Query Dasar ==========
	query := ctrl.DB.
		Table("lecture_sessions_quiz AS q").
		Select(`
			q.lecture_sessions_quiz_id,
			q.lecture_sessions_quiz_title,
			q.lecture_sessions_quiz_description,
			q.lecture_sessions_quiz_lecture_session_id,
			q.lecture_sessions_quiz_created_at`)

	// ========== 5. Tambahkan LEFT JOIN hanya jika user login ==========
	if userID != "" {
		query = query.Select(`
			q.lecture_sessions_quiz_id,
			q.lecture_sessions_quiz_title,
			q.lecture_sessions_quiz_description,
			q.lecture_sessions_quiz_lecture_session_id,
			q.lecture_sessions_quiz_created_at,
			uq.user_lecture_sessions_quiz_id,
			uq.user_lecture_sessions_quiz_grade_result,
			uq.user_lecture_sessions_quiz_user_id,
			uq.user_lecture_sessions_quiz_created_at`).
			Joins("LEFT JOIN user_lecture_sessions_quiz AS uq ON uq.user_lecture_sessions_quiz_quiz_id = q.lecture_sessions_quiz_id AND uq.user_lecture_sessions_quiz_user_id = ?", userID)
	}

	query = query.
		Joins("JOIN lecture_sessions AS ls ON ls.lecture_session_id = q.lecture_sessions_quiz_lecture_session_id")

	if lectureID != "" {
		query = query.Where("ls.lecture_session_lecture_id = ?", lectureID)
	}
	if lectureSessionID != "" {
		query = query.Where("q.lecture_sessions_quiz_lecture_session_id = ?", lectureSessionID)
	}

	// ========== 6. Eksekusi Query ==========
	if err := query.Scan(&results).Error; err != nil {
		log.Printf("[ERROR] Gagal ambil data kuis: %v\n", err)
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// ========== 7. Debug Output ==========
	for i, r := range results {
		log.Printf("[DEBUG] #%d | QuizID: %s | Title: %s | Grade: %v | UserQuizID: %v\n",
			i+1,
			r.LectureSessionsQuizID,
			r.LectureSessionsQuizTitle,
			r.UserLectureSessionsQuizGrade,
			r.UserLectureSessionsQuizID,
		)
	}

	// ========== 8. Return ==========
	duration := time.Since(start)
	log.Printf("[SUCCESS] Berhasil ambil %d kuis dalam %s\n", len(results), duration)

	return c.JSON(fiber.Map{
		"message": "Berhasil ambil kuis",
		"data":    results,
	})
}
