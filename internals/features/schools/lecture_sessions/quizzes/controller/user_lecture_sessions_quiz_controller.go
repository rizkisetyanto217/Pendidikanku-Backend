package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"schoolku_backend/internals/constants"
	modelUserLectureSession "schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	"schoolku_backend/internals/features/schools/lecture_sessions/quizzes/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/quizzes/model"
	modelLecture "schoolku_backend/internals/features/schools/lectures/main/model"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

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

// =============================
// ➕ Create / Upsert User Quiz Result (by session slug)
// =============================
// =============================
// ➕ Create / Upsert User Quiz Result (by session slug)
// =============================
func (ctrl *UserLectureSessionsQuizController) CreateUserLectureSessionsQuiz(c *fiber.Ctx) error {
	var body dto.CreateUserLectureSessionsQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	// validasi payload
	if err := validate.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// user dari token (bisa anonim)
	userUUID := helper.GetUserUUID(c) // tetap pakai utilmu yang ada
	userID := userUUID.String()
	isAnonymous := userUUID == constants.DummyUserID

	// param slug
	lectureSessionSlug := c.Params("lecture_session_slug")
	if lectureSessionSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture session slug is required")
	}

	// resolve slug -> (session_id, school_id)
	var session struct {
		ID       string `gorm:"column:lecture_session_id"`
		SchoolID string `gorm:"column:lecture_session_school_id"`
	}
	if err := ctrl.DB.WithContext(c.Context()).
		Table("lecture_sessions").
		Select("lecture_session_id, lecture_session_school_id").
		Where("lecture_session_slug = ?", lectureSessionSlug).
		First(&session).Error; err != nil || session.ID == "" {
		return helper.JsonError(c, fiber.StatusNotFound, "Lecture session not found for given slug")
	}

	// 🔒 (opsional tapi disarankan) enforce akses school dari token
	//    — lewati untuk pengguna anonim
	if !isAnonymous {
		allowedSchoolUUIDs, err := helperAuth.GetSchoolIDsFromToken(c)
		if err == nil && len(allowedSchoolUUIDs) > 0 {
			var ok bool
			if sid, e := uuid.Parse(session.SchoolID); e == nil {
				for _, a := range allowedSchoolUUIDs {
					if a == sid {
						ok = true
						break
					}
				}
			}
			if !ok {
				return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak memiliki akses ke school ini")
			}
		}
	}

	// upsert: jika user login, update attempt & best grade
	if !isAnonymous {
		var existing model.UserLectureSessionsQuizModel
		err := ctrl.DB.WithContext(c.Context()).
			Where("user_lecture_sessions_quiz_user_id = ? AND user_lecture_sessions_quiz_quiz_id = ?",
				userID, body.UserLectureSessionsQuizQuizID).
			First(&existing).Error

		if err == nil {
			existing.UserLectureSessionsQuizAttemptCount += 1
			if body.UserLectureSessionsQuizGrade > existing.UserLectureSessionsQuizGrade {
				existing.UserLectureSessionsQuizGrade = body.UserLectureSessionsQuizGrade
			}
			existing.UserLectureSessionsQuizDurationSeconds = body.UserLectureSessionsQuizDurationSeconds

			if err := ctrl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update quiz result")
			}
			_ = ctrl.RecalculateLectureSessionsGradeByID(userID, session.ID, session.SchoolID)
			return helper.JsonUpdated(c, "Quiz result updated", dto.ToUserLectureSessionsQuizDTO(existing))
		}
		// record belum ada → lanjut create
	}

	newData := model.UserLectureSessionsQuizModel{
		UserLectureSessionsQuizGrade:            body.UserLectureSessionsQuizGrade,
		UserLectureSessionsQuizQuizID:           body.UserLectureSessionsQuizQuizID,
		UserLectureSessionsQuizUserID:           userID,
		UserLectureSessionsQuizSchoolID:         session.SchoolID,
		UserLectureSessionsQuizAttemptCount:     1,
		UserLectureSessionsQuizDurationSeconds:  body.UserLectureSessionsQuizDurationSeconds,
		UserLectureSessionsQuizLectureSessionID: session.ID,
	}

	if err := ctrl.DB.WithContext(c.Context()).Create(&newData).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to save quiz result")
	}

	if !isAnonymous {
		_ = ctrl.RecalculateLectureSessionsGradeByID(userID, session.ID, session.SchoolID)
	}

	return helper.JsonCreated(c, "Quiz result created", dto.ToUserLectureSessionsQuizDTO(newData))
}

// =======================================================
// Helpers (tetap return error; dipanggil internal)
// =======================================================
func (ctrl *UserLectureSessionsQuizController) RecalculateLectureSessionsGradeByID(
	userID, lectureSessionID, schoolID string,
) error {
	// ---- Parse IDs ke uuid (validasi lebih dini) ----
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid userID: %w", err)
	}
	sid, err := uuid.Parse(lectureSessionID)
	if err != nil {
		return fmt.Errorf("invalid lectureSessionID: %w", err)
	}
	mid, err := uuid.Parse(schoolID)
	if err != nil {
		return fmt.Errorf("invalid schoolID: %w", err)
	}

	// ---- Resolve lecture_session -> lecture_id ----
	var session struct {
		LectureID uuid.UUID
	}
	if err := ctrl.DB.
		Table("lecture_sessions").
		Select("lecture_session_lecture_id AS lecture_id").
		Where("lecture_session_id = ?", sid).
		Scan(&session).Error; err != nil {
		return fmt.Errorf("failed to find session by ID: %w", err)
	}
	if session.LectureID == uuid.Nil {
		return fmt.Errorf("failed to find session by ID: lecture_id is nil")
	}

	// ---- AVG nilai quiz user di sesi tsb (hasil bisa NULL) ----
	var avg sql.NullFloat64
	if err := ctrl.DB.
		Table("user_lecture_sessions_quiz").
		Select("AVG(user_lecture_sessions_quiz_grade_result)").
		Where("user_lecture_sessions_quiz_user_id = ? AND user_lecture_sessions_quiz_lecture_session_id = ?", uid, sid).
		Scan(&avg).Error; err != nil {
		return fmt.Errorf("failed to calculate quiz average: %w", err)
	}

	// ---- Upsert ke user_lecture_sessions (idempotent by (user_id, session_id)) ----
	var existing modelUserLectureSession.UserLectureSessionModel
	findErr := ctrl.DB.
		Where("user_lecture_session_user_id = ? AND user_lecture_session_lecture_session_id = ?", uid, sid).
		First(&existing).Error

	switch {
	case errors.Is(findErr, gorm.ErrRecordNotFound):
		// Create baru
		newData := modelUserLectureSession.UserLectureSessionModel{
			UserLectureSessionUserID:           uid,
			UserLectureSessionLectureSessionID: sid,
			UserLectureSessionLectureID:        session.LectureID,
			UserLectureSessionSchoolID:         mid,
		}
		if avg.Valid {
			v := avg.Float64
			newData.UserLectureSessionGradeResult = &v
		}
		if err := ctrl.DB.Create(&newData).Error; err != nil {
			return fmt.Errorf("failed to create user_lecture_session: %w", err)
		}

	case findErr != nil:
		// Error lain saat mencari
		return fmt.Errorf("failed to get user_lecture_session: %w", findErr)

	default:
		// Update kolom grade saja (NULL jika avg tidak valid)
		if avg.Valid {
			if err := ctrl.DB.Model(&existing).
				Update("user_lecture_session_grade_result", avg.Float64).Error; err != nil {
				return fmt.Errorf("failed to update grade result: %w", err)
			}
		} else {
			if err := ctrl.DB.Model(&existing).
				Update("user_lecture_session_grade_result", gorm.Expr("NULL")).Error; err != nil {
				return fmt.Errorf("failed to nullify grade result: %w", err)
			}
		}
	}

	// ---- Update progress lecture user (pakai string lagi jika fungsi target pakai string) ----
	return ctrl.UpdateUserLectureProgressByID(uid.String(), session.LectureID.String(), mid.String())
}

func (ctrl *UserLectureSessionsQuizController) UpdateUserLectureProgressByID(userID, lectureID, schoolID string) error {
	// avg nilai seluruh sesi untuk lecture tsb
	var avg float64
	if err := ctrl.DB.
		Table("user_lecture_sessions").
		Select("AVG(user_lecture_session_grade_result)").
		Joins("JOIN lecture_sessions ON user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id").
		Where("user_lecture_session_user_id = ? AND lecture_sessions.lecture_session_lecture_id = ?", userID, lectureID).
		Scan(&avg).Error; err != nil {
		return fmt.Errorf("failed to calculate lecture avg: %w", err)
	}

	// jumlah sesi yang sudah punya nilai
	var count int64
	if err := ctrl.DB.
		Table("user_lecture_sessions").
		Joins("JOIN lecture_sessions ON user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id").
		Where(`user_lecture_session_user_id = ? 
		       AND lecture_sessions.lecture_session_lecture_id = ? 
		       AND user_lecture_session_grade_result IS NOT NULL`, userID, lectureID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count completed sessions: %w", err)
	}

	// upsert ke user_lectures
	var existing modelLecture.UserLectureModel
	err := ctrl.DB.
		Where("user_lecture_user_id = ? AND user_lecture_lecture_id = ?", userID, lectureID).
		First(&existing).Error

	if err != nil {
		newData := modelLecture.UserLectureModel{
			UserLectureUserID:                 uuid.MustParse(userID),
			UserLectureLectureID:              uuid.MustParse(lectureID),
			UserLectureSchoolID:               uuid.MustParse(schoolID),
			UserLectureGradeResult:            intPtr(int(avg)),
			UserLectureTotalCompletedSessions: int(count),
		}
		return ctrl.DB.Create(&newData).Error
	}

	return ctrl.DB.
		Model(&existing).
		Updates(map[string]any{
			"user_lecture_grade_result":             int(avg),
			"user_lecture_total_completed_sessions": int(count),
		}).Error
}

func intPtr(v int) *int { return &v }

// =============================
// 📄 Get All Quiz Results
// =============================
func (ctrl *UserLectureSessionsQuizController) GetAllUserLectureSessionsQuiz(c *fiber.Ctx) error {
	var results []model.UserLectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).Find(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch quiz results")
	}

	dtos := make([]dto.UserLectureSessionsQuizDTO, 0, len(results))
	for _, r := range results {
		dtos = append(dtos, dto.ToUserLectureSessionsQuizDTO(r))
	}
	return helper.JsonOK(c, "OK", dtos)
}

// =============================
// 🔍 Get By Quiz ID or User ID (query params)
// =============================
func (ctrl *UserLectureSessionsQuizController) GetUserLectureSessionsQuizFiltered(c *fiber.Ctx) error {
	quizID := c.Query("quiz_id")
	userID := c.Query("user_id")

	q := ctrl.DB.WithContext(c.Context()).Model(&model.UserLectureSessionsQuizModel{})
	if quizID != "" {
		if _, err := uuid.Parse(quizID); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
		q = q.Where("user_lecture_sessions_quiz_quiz_id = ?", quizID)
	}
	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
		}
		q = q.Where("user_lecture_sessions_quiz_user_id = ?", userID)
	}

	var results []model.UserLectureSessionsQuizModel
	if err := q.Find(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch filtered quiz results")
	}

	dtos := make([]dto.UserLectureSessionsQuizDTO, 0, len(results))
	for _, r := range results {
		dtos = append(dtos, dto.ToUserLectureSessionsQuizDTO(r))
	}
	return helper.JsonOK(c, "OK", dtos)
}

// =============================
// ❌ Delete Quiz Result by ID
// =============================
func (ctrl *UserLectureSessionsQuizController) DeleteUserLectureSessionsQuizByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Delete(&model.UserLectureSessionsQuizModel{}, "user_lecture_sessions_quiz_id = ?", idStr).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete quiz result")
	}

	return helper.JsonDeleted(c, "Quiz result deleted successfully", fiber.Map{"id": idStr})
}

// =============================
// 📊 GetUserQuizWithDetail (opsional gabung progress user)
// =============================
func (ctrl *UserLectureSessionsQuizController) GetUserQuizWithDetail(c *fiber.Ctx) error {
	start := time.Now()

	// user (opsional)
	userID := ""
	if v := c.Locals("user_id"); v != nil {
		userID, _ = v.(string)
		if userID != "" {
			if _, err := uuid.Parse(userID); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
			}
		}
	}

	lectureID := c.Query("lecture_id")
	lectureSessionID := c.Query("lecture_session_id")
	if lectureID == "" && lectureSessionID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Minimal salah satu parameter: lecture_id atau lecture_session_id harus diisi")
	}
	if lectureID != "" {
		if _, err := uuid.Parse(lectureID); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "lecture_id tidak valid")
		}
	}
	if lectureSessionID != "" {
		if _, err := uuid.Parse(lectureSessionID); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "lecture_session_id tidak valid")
		}
	}

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

	// base select
	baseSelect := `
		q.lecture_sessions_quiz_id,
		q.lecture_sessions_quiz_title,
		q.lecture_sessions_quiz_description,
		q.lecture_sessions_quiz_lecture_session_id,
		q.lecture_sessions_quiz_created_at`

	query := ctrl.DB.WithContext(c.Context()).
		Table("lecture_sessions_quiz AS q").
		Select(baseSelect).
		Joins("JOIN lecture_sessions AS ls ON ls.lecture_session_id = q.lecture_sessions_quiz_lecture_session_id")

	// join progress user jika login
	if userID != "" {
		query = query.Select(baseSelect+`,
			uq.user_lecture_sessions_quiz_id,
			uq.user_lecture_sessions_quiz_grade_result,
			uq.user_lecture_sessions_quiz_user_id,
			uq.user_lecture_sessions_quiz_created_at`).
			Joins(`LEFT JOIN user_lecture_sessions_quiz AS uq 
			       ON uq.user_lecture_sessions_quiz_quiz_id = q.lecture_sessions_quiz_id 
			      AND uq.user_lecture_sessions_quiz_user_id = ?`, userID)
	}

	if lectureID != "" {
		query = query.Where("ls.lecture_session_lecture_id = ?", lectureID)
	}
	if lectureSessionID != "" {
		query = query.Where("q.lecture_sessions_quiz_lecture_session_id = ?", lectureSessionID)
	}

	if err := query.Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	log.Printf("[INFO] fetched %d quizzes in %s", len(results), time.Since(start))
	return helper.JsonOK(c, "Berhasil ambil kuis", results)
}
