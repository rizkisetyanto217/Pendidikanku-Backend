package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserLectureSessionController struct {
	DB *gorm.DB
}

func NewUserLectureSessionController(db *gorm.DB) *UserLectureSessionController {
	return &UserLectureSessionController{DB: db}
}

// CREATE
func (ctrl *UserLectureSessionController) CreateUserLectureSession(c *fiber.Ctx) error {
	var req dto.CreateUserLectureSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	newRecord := model.UserLectureSessionModel{
		UserLectureSessionAttendanceStatus: req.UserLectureSessionAttendanceStatus,
		UserLectureSessionGradeResult:      req.UserLectureSessionGradeResult,
		UserLectureSessionLectureSessionID: req.UserLectureSessionLectureSessionID,
		UserLectureSessionUserID:           req.UserLectureSessionUserID,
		UserLectureSessionIsRegistered:     req.UserLectureSessionIsRegistered,
		UserLectureSessionHasPaid:          req.UserLectureSessionHasPaid,
		UserLectureSessionPaidAmount:       req.UserLectureSessionPaidAmount,
		UserLectureSessionPaymentTime:      req.UserLectureSessionPaymentTime,
	}

	if err := ctrl.DB.Create(&newRecord).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat user lecture session")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureSessionDTO(newRecord))
}

// GET ALL
func (ctrl *UserLectureSessionController) GetAllUserLectureSessions(c *fiber.Ctx) error {
	var records []model.UserLectureSessionModel
	if err := ctrl.DB.Find(&records).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve records")
	}

	var result []dto.UserLectureSessionDTO
	for _, record := range records {
		result = append(result, dto.ToUserLectureSessionDTO(record))
	}

	return c.JSON(result)
}

// GET BY ID
func (ctrl *UserLectureSessionController) GetUserLectureSessionByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var record model.UserLectureSessionModel
	if err := ctrl.DB.First(&record, "user_lecture_session_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Record not found")
	}

	return c.JSON(dto.ToUserLectureSessionDTO(record))
}

// UPDATE
func (ctrl *UserLectureSessionController) UpdateUserLectureSession(c *fiber.Ctx) error {
	id := c.Params("id")

	var record model.UserLectureSessionModel
	if err := ctrl.DB.First(&record, "user_lecture_session_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	var req dto.CreateUserLectureSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Update field
	record.UserLectureSessionAttendanceStatus = req.UserLectureSessionAttendanceStatus
	record.UserLectureSessionGradeResult = req.UserLectureSessionGradeResult
	record.UserLectureSessionLectureSessionID = req.UserLectureSessionLectureSessionID
	record.UserLectureSessionUserID = req.UserLectureSessionUserID
	record.UserLectureSessionIsRegistered = req.UserLectureSessionIsRegistered
	record.UserLectureSessionHasPaid = req.UserLectureSessionHasPaid
	record.UserLectureSessionPaidAmount = req.UserLectureSessionPaidAmount
	record.UserLectureSessionPaymentTime = req.UserLectureSessionPaymentTime

	if err := ctrl.DB.Save(&record).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data user lecture session")
	}

	return c.JSON(dto.ToUserLectureSessionDTO(record))
}

// DELETE
func (ctrl *UserLectureSessionController) DeleteUserLectureSession(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.UserLectureSessionModel{}, "user_lecture_session_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (ctrl *UserLectureSessionController) GetLectureSessionsWithUserProgress(c *fiber.Ctx) error {
	userIDRaw := c.Locals("user_id")
	userID := ""
	if userIDRaw != nil {
		userID = userIDRaw.(string)
	}

	masjidID := c.Query("masjid_id")
	if masjidID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Parameter masjid_id wajib diisi")
	}

	lectureID := c.Query("lecture_id") // ✅ Ambil parameter lecture_id

	type Result struct {
		LectureSessionID                     string     `json:"lecture_session_id"`
		LectureSessionTitle                  string     `json:"lecture_session_title"`
		LectureSessionDescription            string     `json:"lecture_session_description"`
		LectureSessionTeacherID              string     `json:"lecture_session_teacher_id"`
		LectureSessionTeacherName            string     `json:"lecture_session_teacher_name"`
		LectureSessionStartTime              time.Time  `json:"lecture_session_start_time"`
		LectureSessionEndTime                time.Time  `json:"lecture_session_end_time"`
		LectureSessionPlace                  string     `json:"lecture_session_place"`
		LectureSessionLectureID              string     `json:"lecture_session_lecture_id"`
		LectureSessionMasjidID               string     `json:"lecture_session_masjid_id"`
		LectureSessionCapacity               int        `json:"lecture_session_capacity"`
		LectureSessionIsPublic               bool       `json:"lecture_session_is_public"`
		LectureSessionIsRegistrationRequired bool       `json:"lecture_session_is_registration_required"`
		LectureSessionIsPaid                 bool       `json:"lecture_session_is_paid"`
		LectureSessionPrice                  *int       `json:"lecture_session_price,omitempty"`
		LectureSessionPaymentDeadline        *time.Time `json:"lecture_session_payment_deadline,omitempty"`
		LectureSessionCreatedAt              time.Time  `json:"lecture_session_created_at"`

		UserLectureSessionAttendanceStatus *int       `json:"user_lecture_session_attendance_status,omitempty"`
		UserLectureSessionGradeResult      *float64   `json:"user_lecture_session_grade_result,omitempty"`
		UserLectureSessionIsRegistered     *bool      `json:"user_lecture_session_is_registered,omitempty"`
		UserLectureSessionHasPaid          *bool      `json:"user_lecture_session_has_paid,omitempty"`
		UserLectureSessionPaidAmount       *int       `json:"user_lecture_session_paid_amount,omitempty"`
		UserLectureSessionPaymentTime      *time.Time `json:"user_lecture_session_payment_time,omitempty"`
		UserLectureSessionCreatedAt        *time.Time `json:"user_lecture_session_user_session_created_at,omitempty"`
	}

	var results []Result

	query := ctrl.DB.Table("lecture_sessions AS ls").
		Select([]string{
			"ls.lecture_session_id",
			"ls.lecture_session_title",
			"ls.lecture_session_description",
			"ls.lecture_session_teacher->>'id' AS lecture_session_teacher_id",
			"ls.lecture_session_teacher->>'name' AS lecture_session_teacher_name",
			"ls.lecture_session_start_time",
			"ls.lecture_session_end_time",
			"ls.lecture_session_place",
			"ls.lecture_session_lecture_id",
			"l.lecture_masjid_id AS lecture_session_masjid_id",
			"ls.lecture_session_capacity",
			"ls.lecture_session_is_public",
			"ls.lecture_session_is_registration_required",
			"ls.lecture_session_is_paid",
			"ls.lecture_session_price",
			"ls.lecture_session_payment_deadline",
			"ls.lecture_session_created_at",
			"uls.user_lecture_session_attendance_status",
			"uls.user_lecture_session_grade_result",
			"uls.user_lecture_session_is_registered",
			"uls.user_lecture_session_has_paid",
			"uls.user_lecture_session_paid_amount",
			"uls.user_lecture_session_payment_time",
			"uls.user_lecture_session_created_at",
		}).
		Joins("LEFT JOIN lectures l ON l.lecture_id = ls.lecture_session_lecture_id").
		Where("l.lecture_masjid_id = ?", masjidID).
		Order("ls.lecture_session_start_time ASC")

	// ✅ Tambahkan filter lecture_id jika ada
	if lectureID != "" {
		query = query.Where("ls.lecture_session_lecture_id = ?", lectureID)
	}

	// ✅ Join user progress jika login
	if userID != "" {
		query = query.Joins(`
			LEFT JOIN user_lecture_sessions uls 
			ON uls.user_lecture_session_lecture_session_id = ls.lecture_session_id 
			AND uls.user_lecture_session_user_id = ?
		`, userID)
	} else {
		query = query.Joins("LEFT JOIN user_lecture_sessions uls ON false")
	}

	if err := query.Scan(&results).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data sesi kajian")
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil daftar sesi kajian (dengan progress jika login)",
		"data":    results,
	})
}
