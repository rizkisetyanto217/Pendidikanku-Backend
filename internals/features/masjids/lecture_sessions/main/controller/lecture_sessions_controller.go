package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LectureSessionController struct {
	DB *gorm.DB
}

func NewLectureSessionController(db *gorm.DB) *LectureSessionController {
	return &LectureSessionController{DB: db}
}

func (ctrl *LectureSessionController) CreateLectureSession(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Validasi user login
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}

	// Konversi request ke model
	newSession := body.ToModel()
	newSession.LectureSessionIsActive = true
	
	// 💡 Jika waktu verifikasi teacher dikirim, berarti tidak diperiksa oleh guru
	if body.LectureSessionApprovedByTeacherAt != nil {
		newSession.LectureSessionApprovedByTeacherID = &newSession.LectureSessionTeacherID
		now := time.Now()
		newSession.LectureSessionApprovedByTeacherAt = &now
	}


	// Simpan ke DB
	if err := ctrl.DB.Create(&newSession).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kajian")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionDTO(newSession))
}



// ================================
// GET ALL
// ================================
func (ctrl *LectureSessionController) GetAllLectureSessions(c *fiber.Ctx) error {
	var sessions []model.LectureSessionModel

	if err := ctrl.DB.Find(&sessions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch lecture sessions")
	}

	var result []dto.LectureSessionDTO
	for _, s := range sessions {
		result = append(result, dto.ToLectureSessionDTO(s))
	}

	return c.JSON(result)
}


func (ctrl *LectureSessionController) GetLectureSessionsByMasjidID(c *fiber.Ctx) error {
	// ✅ Ambil dari token (middleware sudah pastikan valid dan admin)
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Masjid ID tidak valid atau tidak ditemukan di token",
		})
	}

	// Struct untuk hasil gabungan
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle string `gorm:"column:lecture_title"`
	}

	var results []JoinedResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select("lecture_sessions.*, lectures.lecture_title").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_masjid_id = ?", masjidID).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Scan(&results).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian berdasarkan masjid",
		})
	}

	// Map ke DTO
	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		response[i] = dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian berdasarkan masjid",
		"data":    response,
	})
}




// ✅ POST /api/a/lecture-sessions/by-lecture-id

// ✅ GET lecture sessions by lecture_id (adaptif: jika login, include user progress)
func (ctrl *LectureSessionController) GetByLectureID(c *fiber.Ctx) error {
	type RequestBody struct {
		LectureID string `json:"lecture_id"`
	}

	var body RequestBody
	if err := c.BodyParser(&body); err != nil || body.LectureID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Permintaan tidak valid, lecture_id wajib diisi",
		})
	}

	lectureID, err := uuid.Parse(body.LectureID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Lecture ID tidak valid",
		})
	}

	userIDRaw := c.Locals("user_id")

	// Jika tidak login, ambil data biasa
	if userIDRaw == nil {
		var sessions []model.LectureSessionModel
		if err := ctrl.DB.
			Where("lecture_session_lecture_id = ?", lectureID).
			Order("lecture_session_start_time ASC").
			Find(&sessions).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal mengambil data sesi kajian",
			})
		}

		response := make([]dto.LectureSessionDTO, len(sessions))
		for i, s := range sessions {
			response[i] = dto.ToLectureSessionDTO(s)
		}

		return c.JSON(fiber.Map{
			"message": "Berhasil mengambil sesi kajian",
			"data":    response,
		})
	}

	// Jika login → Ambil juga progress user
	userIDStr, ok := userIDRaw.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "User ID tidak valid",
		})
	}

	// Ambil data + progress via LEFT JOIN
	type JoinedResult struct {
		model.LectureSessionModel
		UserAttendanceStatus string   `json:"user_attendance_status"`
		UserGradeResult      *float64 `json:"user_grade_result"`
	}

	var joined []JoinedResult
	if err := ctrl.DB.Table("lecture_sessions as ls").
		Select(`
			ls.*, 
			uls.user_lecture_session_status_attendance as user_attendance_status, 
			uls.user_lecture_session_grade_result as user_grade_result
		`).
		Joins(`
			LEFT JOIN user_lecture_sessions uls 
			ON uls.user_lecture_session_lecture_session_id = ls.lecture_session_id 
			AND uls.user_lecture_session_user_id = ?
		`, userIDStr).
		Where("ls.lecture_session_lecture_id = ?", lectureID).
		Order("ls.lecture_session_start_time ASC").
		Scan(&joined).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data sesi + progres user",
		})
	}

	// Gabungkan ke response
	response := make([]fiber.Map, len(joined))
	for i, j := range joined {
		response[i] = fiber.Map{
			"lecture_session":        dto.ToLectureSessionDTO(j.LectureSessionModel),
			"user_attendance_status": j.UserAttendanceStatus,
			"user_grade_result":      j.UserGradeResult,
		}
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian + progres user",
		"data":    response,
	})
}


// ================================
// UPDATE
// ================================
func (ctrl *LectureSessionController) UpdateLectureSession(c *fiber.Ctx) error {
	idParam := c.Params("id")
	sessionID, err := uuid.Parse(idParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil data sesi yang ada
	var existing model.LectureSessionModel
	if err := ctrl.DB.First(&existing, "lecture_session_id = ?", sessionID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	// Parsing body
	var body dto.UpdateLectureSessionRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Validasi waktu
	if body.LectureSessionEndTime.Before(body.LectureSessionStartTime) {
		return fiber.NewError(fiber.StatusBadRequest, "Waktu selesai tidak boleh sebelum waktu mulai")
	}

	// Update field dari body
	existing.LectureSessionTitle = body.LectureSessionTitle
	existing.LectureSessionDescription = body.LectureSessionDescription
	existing.LectureSessionTeacherID = body.LectureSessionTeacherID
	existing.LectureSessionTeacherName = body.LectureSessionTeacherName
	existing.LectureSessionStartTime = body.LectureSessionStartTime
	existing.LectureSessionEndTime = body.LectureSessionEndTime
	existing.LectureSessionPlace = body.LectureSessionPlace
	existing.LectureSessionImageURL = body.LectureSessionImageURL
	existing.LectureSessionLectureID = body.LectureSessionLectureID

	// Simpan ke database
	if err := ctrl.DB.Save(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi kajian")
	}

	return c.JSON(dto.ToLectureSessionDTO(existing))
}


func (ctrl *LectureSessionController) ApproveLectureSession(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil role dan ID user dari context (misalnya dari JWT middleware)
	role := c.Locals("role")       // role: admin / author / teacher
	userID := c.Locals("user_id")  // UUID (pastikan sudah parse di middleware)

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak valid")
	}

	var session model.LectureSessionModel
	if err := ctrl.DB.First(&session, "lecture_session_id = ?", sessionID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	now := time.Now()

	switch role {
	case "admin":
		session.LectureSessionApprovedByAdminID = &userUUID
		session.LectureSessionApprovedByAdminAt = &now

	case "author":
		session.LectureSessionApprovedByAuthorID = &userUUID
		session.LectureSessionApprovedByAuthorAt = &now

	case "teacher":
		session.LectureSessionApprovedByTeacherID = &userUUID
		session.LectureSessionApprovedByTeacherAt = &now

	default:
		return fiber.NewError(fiber.StatusForbidden, "Role tidak diizinkan untuk melakukan approval")
	}

	if err := ctrl.DB.Save(&session).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan approval")
	}

	return c.JSON(dto.ToLectureSessionDTO(session))
}


// ================================
// DELETE
// ================================
func (ctrl *LectureSessionController) DeleteLectureSession(c *fiber.Ctx) error {
	idParam := c.Params("id")
	sessionID, err := uuid.Parse(idParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctrl.DB.Delete(&model.LectureSessionModel{}, "lecture_session_id = ?", sessionID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus sesi kajian")
	}

	return c.JSON(fiber.Map{
		"message": "Sesi kajian berhasil dihapus",
	})
}
