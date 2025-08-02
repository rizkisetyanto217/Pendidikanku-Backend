package controller

import (
	"fmt"
	"log"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	lectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)


func (ctrl *LectureSessionController) GetLectureSessionsByMasjidIDParam(c *fiber.Ctx) error {
	masjidID := c.Params("id")
	if masjidID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Masjid ID tidak ditemukan di parameter URL",
		})
	}

	// Struct untuk hasil join
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
			"message": "Gagal mengambil sesi kajian berdasarkan masjid ID",
		})
	}

	// Map ke DTO
	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		response[i] = dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian berdasarkan masjid ID",
		"data":    response,
	})
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
	var lecture lectureModel.LectureModel
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


// ✅ GET lecture sessions by multiple lecture_session_ids (ringan, tanpa progress user)
func (ctrl *LectureSessionController) GetByIDs(c *fiber.Ctx) error {
	type RequestBody struct {
		LectureSessionIDs []string `json:"lecture_session_ids"`
	}

	var body RequestBody
	if err := c.BodyParser(&body); err != nil || len(body.LectureSessionIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Permintaan tidak valid, lecture_session_ids wajib diisi",
		})
	}

	// Parsing string UUID ke uuid.UUID
	var parsedIDs []uuid.UUID
	for _, idStr := range body.LectureSessionIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": fmt.Sprintf("Lecture session ID tidak valid: %s", idStr),
			})
		}
		parsedIDs = append(parsedIDs, id)
	}

	var sessions []model.LectureSessionModel
	if err := ctrl.DB.
		Where("lecture_session_id IN ?", parsedIDs).
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
		"message": "Berhasil mengambil data sesi kajian",
		"data":    response,
	})
}


func (ctrl *LectureSessionController) GetLectureSessionsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Slug masjid tidak ditemukan di URL",
		})
	}

	// Cari masjid berdasarkan slug
	var masjid struct {
		MasjidID uuid.UUID `gorm:"column:masjid_id"`
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Scan(&masjid).Error; err != nil || masjid.MasjidID == uuid.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Masjid dengan slug tersebut tidak ditemukan",
		})
	}

	// Join dengan lectures dan users
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle string  `gorm:"column:lecture_title"`
		UserName     *string `gorm:"column:user_name"`
	}

	var results []JoinedResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select("lecture_sessions.*, lectures.lecture_title, users.user_name").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id").
		Where("lectures.lecture_masjid_id = ?", masjid.MasjidID).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Scan(&results).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian berdasarkan slug masjid",
		})
	}

	// Mapping ke DTO
	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		dtoItem := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)

		// Fallback jika teacher_name kosong
		if dtoItem.LectureSessionTeacherName == "" && r.UserName != nil {
			dtoItem.LectureSessionTeacherName = *r.UserName
		}

		response[i] = dtoItem
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian berdasarkan slug masjid",
		"data":    response,
	})
}


func (ctrl *LectureSessionController) GetUpcomingLectureSessionsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Slug masjid tidak ditemukan di URL",
		})
	}

	// Cari masjid berdasarkan slug
	var masjid struct {
		MasjidID uuid.UUID `gorm:"column:masjid_id"`
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Scan(&masjid).Error; err != nil || masjid.MasjidID == uuid.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Masjid dengan slug tersebut tidak ditemukan",
		})
	}

	now := time.Now()

	// Join lectures + users
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle string  `gorm:"column:lecture_title"`
		UserName     *string `gorm:"column:user_name"`
	}

	var results []JoinedResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select("lecture_sessions.*, lectures.lecture_title, users.user_name").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id").
		Where("lectures.lecture_masjid_id = ? AND lecture_sessions.lecture_session_start_time > ?", masjid.MasjidID, now).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Scan(&results).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian mendatang berdasarkan slug masjid",
		})
	}

	// Map ke DTO
	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		dtoItem := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
		if dtoItem.LectureSessionTeacherName == "" && r.UserName != nil {
			dtoItem.LectureSessionTeacherName = *r.UserName
		}
		response[i] = dtoItem
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian mendatang berdasarkan slug masjid",
		"data":    response,
	})
}



func (ctrl *LectureSessionController) GetFinishedLectureSessionsByMasjidSlug(c *fiber.Ctx) error {
	log.Println("🟢 GET /api/u/masjids/:slug/finished-lecture-sessions")

	// --- Ambil slug dan user ID ---
	slug := c.Params("slug")
	if slug == "" {
		log.Println("[ERROR] Slug masjid kosong")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Slug masjid tidak ditemukan di URL",
		})
	}

	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}
	log.Println("[INFO] user_id dari request:", userID)

	// --- Ambil masjid_id berdasarkan slug ---
	var masjid struct {
		MasjidID uuid.UUID
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Scan(&masjid).Error; err != nil || masjid.MasjidID == uuid.Nil {
		log.Println("[ERROR] Gagal menemukan masjid dari slug:", slug)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Masjid dengan slug tersebut tidak ditemukan",
		})
	}
	log.Println("[INFO] masjid_id ditemukan:", masjid.MasjidID)

	now := time.Now()

	// --- Struct untuk hasil join ---
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle     string   `gorm:"column:lecture_title"`
		UserName         *string  `gorm:"column:user_name"`
		UserGradeResult  *float64 `gorm:"column:user_grade_result"`
		AttendanceStatus *int     `gorm:"column:attendance_status"`
	}

	// --- Select fields ---
	selectFields := []string{
		"lecture_sessions.*",
		"lectures.lecture_title",
		"users.user_name",
	}
	if userID != "" {
		selectFields = append(selectFields,
			"user_lecture_sessions.user_lecture_session_grade_result AS user_grade_result",
			"user_lecture_sessions_attendance.user_lecture_sessions_attendance_status AS attendance_status",
		)
		log.Println("[INFO] Menambahkan select field: grade_result dan attendance_status")
	}

	// --- Query builder ---
	query := ctrl.DB.Model(&model.LectureSessionModel{}).
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id")

	if userID != "" {
		query = query.
			Joins(`LEFT JOIN user_lecture_sessions 
				ON user_lecture_sessions.user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id 
				AND user_lecture_sessions.user_lecture_session_user_id = ?`, userID).
			Joins(`LEFT JOIN user_lecture_sessions_attendance 
				ON user_lecture_sessions_attendance.user_lecture_sessions_attendance_lecture_session_id = lecture_sessions.lecture_session_id 
				AND user_lecture_sessions_attendance.user_lecture_sessions_attendance_user_id = ?`, userID)
		log.Println("[INFO] Join ke user_lecture_sessions dan user_lecture_sessions_attendance berhasil")
	}

	// --- Eksekusi query ---
	var results []JoinedResult
	log.Println("[DEBUG] Menjalankan query untuk ambil sesi kajian selesai")
	if err := query.
		Select(strings.Join(selectFields, ", ")).
		Where("lecture_sessions.lecture_session_masjid_id = ? AND lecture_sessions.lecture_session_end_time < ?", masjid.MasjidID, now).
		Order("lecture_sessions.lecture_session_start_time DESC").
		Scan(&results).Error; err != nil {
		log.Println("[ERROR] Gagal mengeksekusi query:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian",
			"error":   err.Error(),
		})
	}

	log.Printf("[INFO] Berhasil ambil %d sesi kajian\n", len(results))

	// --- Mapping ke DTO ---
	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		dtoItem := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)

		if dtoItem.LectureSessionTeacherName == "" && r.UserName != nil {
			dtoItem.LectureSessionTeacherName = *r.UserName
		}
		if r.UserGradeResult != nil {
			dtoItem.UserGradeResult = r.UserGradeResult
			log.Printf("[DEBUG] Sesi %s: Grade = %.2f\n", r.LectureSessionID, *r.UserGradeResult)
		}
		if r.AttendanceStatus != nil {
			dtoItem.UserAttendanceStatus = r.AttendanceStatus
			log.Printf("[DEBUG] Sesi %s: AttendanceStatus = %d\n", r.LectureSessionID, *r.AttendanceStatus)
		}
		response[i] = dtoItem
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian yang telah lewat berdasarkan slug masjid",
		"data":    response,
	})
}


func (ctrl *LectureSessionController) GetLectureSessionByIDProgressUser(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		log.Println("[ERROR] Invalid session ID:", c.Params("id"))
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil user_id dari cookie / header
	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}
	log.Println("[INFO] user_id dari request:", userID)

	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle    string   `gorm:"column:lecture_title"`
		UserName        *string  `gorm:"column:user_name"`
		UserGradeResult *float64 `gorm:"column:user_grade_result"`
	}

	var result JoinedResult

	query := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select(`
			lecture_sessions.*, 
			lectures.lecture_title, 
			users.user_name
		`).
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id")

	if userID != "" {
		log.Println("[INFO] Menambahkan join ke user_lecture_sessions")
		query = query.Select(`
			lecture_sessions.*, 
			lectures.lecture_title, 
			users.user_name,
			user_lecture_sessions.user_lecture_session_grade_result AS user_grade_result
		`).Joins(`
			LEFT JOIN user_lecture_sessions 
			ON user_lecture_sessions.user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id 
			AND user_lecture_sessions.user_lecture_session_user_id = ?
		`, userID)
	}

	log.Println("[INFO] Eksekusi query untuk session ID:", sessionID)

	if err := query.
		Where("lecture_sessions.lecture_session_id = ?", sessionID).
		Scan(&result).Error; err != nil {
		log.Println("[ERROR] Gagal ambil data sesi kajian:", err)
		return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	log.Println("[INFO] Hasil query berhasil diambil")
	log.Printf("[DEBUG] UserGradeResult: %v\n", result.UserGradeResult)

	dtoItem := dto.ToLectureSessionDTOWithLectureTitle(result.LectureSessionModel, result.LectureTitle)

	if dtoItem.LectureSessionTeacherName == "" && result.UserName != nil {
		dtoItem.LectureSessionTeacherName = *result.UserName
		log.Println("[INFO] Fallback user_name digunakan sebagai teacher name:", *result.UserName)
	}

	if result.UserGradeResult != nil {
		dtoItem.UserGradeResult = result.UserGradeResult
	}

	return c.JSON(dtoItem)
}