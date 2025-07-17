package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	LectureModel "masjidku_backend/internals/features/masjids/lectures/model"
	helper "masjidku_backend/internals/helpers"

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
	// Validasi user login
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}

	// Ambil masjid_id dari token
	masjidIDs, ok := c.Locals("masjid_admin_ids").([]string)
	if !ok || len(masjidIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}
	masjidID, err := uuid.Parse(masjidIDs[0])
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak valid")
	}

	// Ambil semua field dari form-data
	title := c.FormValue("lecture_session_title")
	description := c.FormValue("lecture_session_description")
	teacherIDStr := c.FormValue("lecture_session_teacher_id")
	teacherName := c.FormValue("lecture_session_teacher_name")
	startTimeStr := c.FormValue("lecture_session_start_time")
	endTimeStr := c.FormValue("lecture_session_end_time")
	place := c.FormValue("lecture_session_place")
	lectureIDStr := c.FormValue("lecture_session_lecture_id")
	approvedAtStr := c.FormValue("lecture_session_approved_by_teacher_at") // optional

	// Validasi dan parsing UUID & waktu
	teacherID, err := uuid.Parse(teacherIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID guru tidak valid")
	}
	lectureID, err := uuid.Parse(lectureIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tema kajian tidak valid")
	}
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Format waktu mulai tidak valid")
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Format waktu selesai tidak valid")
	}

	// Upload gambar jika ada
	var imageURL *string
	if file, err := c.FormFile("lecture_session_image_url"); err == nil && file != nil {
		url, err := helper.UploadImageAsWebPToSupabase("lecture_sessions", file)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar")
		}
		imageURL = &url
	} else if val := c.FormValue("lecture_session_image_url"); val != "" {
		imageURL = &val
	}

	// Buat objek session
	newSession := model.LectureSessionModel{
		LectureSessionTitle:       title,
		LectureSessionDescription: description,
		LectureSessionTeacherID:   teacherID,
		LectureSessionTeacherName: teacherName,
		LectureSessionStartTime:   startTime,
		LectureSessionEndTime:     endTime,
		LectureSessionPlace:       &place,
		LectureSessionLectureID:   &lectureID,
		LectureSessionMasjidID:    masjidID,
		LectureSessionIsActive:    true,
		LectureSessionImageURL:    imageURL,
	}

	// Jika waktu verifikasi oleh guru dikirim
	if approvedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, approvedAtStr); err == nil {
			newSession.LectureSessionApprovedByTeacherID = &teacherID
			newSession.LectureSessionApprovedByTeacherAt = &t
		}
	}

	// Simpan ke DB
	if err := ctrl.DB.Create(&newSession).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kajian")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionDTO(newSession))
}



// =============================
// 📥 GET All Lecture Sessions by Lecture ID
// =============================
func (ctrl *LectureSessionController) GetLectureSessionsByLectureID(c *fiber.Ctx) error {
	log.Println("📥 MASUK GetLectureSessionsByLectureID (Public)")

	// Ambil lecture_id dari URL
	lectureIDParam := c.Params("lecture_id")
	lectureID, err := uuid.Parse(lectureIDParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Lecture ID tidak valid")
	}

	// ✅ Cek apakah lecture_id valid dan milik masjid yang ada
	var lecture LectureModel.LectureModel
	if err := ctrl.DB.
		Where("lecture_id = ?", lectureID).
		First(&lecture).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Lecture tidak ditemukan")
	}

	// ✅ Ambil semua sesi berdasarkan lecture_id
	var sessions []model.LectureSessionModel
	if err := ctrl.DB.
		Where("lecture_session_lecture_id = ? AND lecture_session_deleted_at IS NULL", lectureID).
		Find(&sessions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data sesi kajian")
	}

	// Konversi ke DTO
	response := make([]dto.LectureSessionDTO, 0, len(sessions))
	for _, s := range sessions {
		response = append(response, dto.ToLectureSessionDTO(s))
	}

	return c.JSON(fiber.Map{
		"message": "Daftar sesi kajian berhasil diambil",
		"data":    response,
	})
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
