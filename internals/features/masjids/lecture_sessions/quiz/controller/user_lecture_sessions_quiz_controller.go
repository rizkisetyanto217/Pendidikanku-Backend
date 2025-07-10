package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/model"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserLectureSessionsQuizController struct {
	DB *gorm.DB
}

func NewUserLectureSessionsQuizController(db *gorm.DB) *UserLectureSessionsQuizController {
	return &UserLectureSessionsQuizController{DB: db}
}

// =============================
// ➕ Create User Quiz Result (from token)
// =============================
func (ctrl *UserLectureSessionsQuizController) CreateUserLectureSessionsQuiz(c *fiber.Ctx) error {
	var body dto.CreateUserLectureSessionsQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validate.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil user_id dari token (diset oleh middleware)
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID not found in token")
	}
	userID := userIDRaw.(string)

	data := model.UserLectureSessionsQuizModel{
		UserLectureSessionsQuizGrade:  body.UserLectureSessionsQuizGrade,
		UserLectureSessionsQuizQuizID: body.UserLectureSessionsQuizQuizID,
		UserLectureSessionsQuizUserID: userID,
	}

	if err := ctrl.DB.Create(&data).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to save quiz result")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureSessionsQuizDTO(data))
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
