package controller

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"

	lectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"
	"net/url"
	"strings"

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

	// JSON guru untuk lecture_teachers
	teacherObj := map[string]string{
		"id":   teacherID.String(),
		"name": teacherName,
	}
	teacherJSON, err := json.Marshal(teacherObj)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal konversi teacher ke JSON")
	}

	// Buat objek session (tanpa gambar dulu)
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
	}

	// Jika waktu verifikasi oleh guru dikirim
	if approvedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, approvedAtStr); err == nil {
			newSession.LectureSessionApprovedByTeacherID = &teacherID
			newSession.LectureSessionApprovedByTeacherAt = &t
		}
	}

	err = ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// Simpan sesi dulu (tanpa gambar)
		if err := tx.Create(&newSession).Error; err != nil {
			return err
		}

		// Tambah 1 sesi ke tema kajian
		if err := tx.Model(&lectureModel.LectureModel{}).
			Where("lecture_id = ?", lectureID).
			UpdateColumn("total_lecture_sessions", gorm.Expr("COALESCE(total_lecture_sessions, 0) + 1")).Error; err != nil {
			return err
		}

		// Tambahkan pengajar ke lecture_teachers (JSONB array)
		if err := tx.Exec(`
			UPDATE lectures
			SET lecture_teachers = CASE
				WHEN lecture_teachers IS NULL THEN jsonb_build_array(?::jsonb)
				WHEN NOT (lecture_teachers @> ?::jsonb) THEN lecture_teachers || ?::jsonb
				ELSE lecture_teachers
			END
			WHERE lecture_id = ?
		`, string(teacherJSON), string(teacherJSON), string(teacherJSON), lectureID).Error; err != nil {
			return err
		}

		// Upload gambar hanya setelah semua DB logic aman
		if file, err := c.FormFile("lecture_session_image_url"); err == nil && file != nil {
			url, err := helper.UploadImageToSupabase("lecture_sessions", file)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar")
			}
			// Update field image di sesi yang sudah dibuat
			if err := tx.Model(&newSession).Update("lecture_session_image_url", url).Error; err != nil {
				return err
			}
			newSession.LectureSessionImageURL = &url
		} else if val := c.FormValue("lecture_session_image_url"); val != "" {
			if err := tx.Model(&newSession).Update("lecture_session_image_url", val).Error; err != nil {
				return err
			}
			newSession.LectureSessionImageURL = &val
		}

		return nil
	})

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kajian")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionDTO(newSession))
}



func (ctrl *LectureSessionController) GetLectureSessionByID(c *fiber.Ctx) error {
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


// ✅ PUT /api/a/lecture-sessions/:id
func (ctrl *LectureSessionController) UpdateLectureSession(c *fiber.Ctx) error {
	// Ambil ID dari param
	idParam := c.Params("id")
	sessionID, err := uuid.Parse(idParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Cari sesi yang ada
	var existing model.LectureSessionModel
	if err := ctrl.DB.First(&existing, "lecture_session_id = ?", sessionID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	// Update field jika ada
	if val := c.FormValue("lecture_session_title"); val != "" {
		existing.LectureSessionTitle = val
		existing.LectureSessionSlug = dto.GenerateSlug(val)
	}

	if val := c.FormValue("lecture_session_description"); val != "" {
		existing.LectureSessionDescription = val
	}
	if val := c.FormValue("lecture_session_teacher_id"); val != "" {
		if id, err := uuid.Parse(val); err == nil {
			existing.LectureSessionTeacherID = id
		}
	}
	if val := c.FormValue("lecture_session_teacher_name"); val != "" {
		existing.LectureSessionTeacherName = val
	}
	if val := c.FormValue("lecture_session_start_time"); val != "" {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			existing.LectureSessionStartTime = t
		}
	}
	if val := c.FormValue("lecture_session_end_time"); val != "" {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			existing.LectureSessionEndTime = t
		}
	}
	if val := c.FormValue("lecture_session_place"); val != "" {
		existing.LectureSessionPlace = &val
	}
	if val := c.FormValue("lecture_session_lecture_id"); val != "" {
		if id, err := uuid.Parse(val); err == nil {
			existing.LectureSessionLectureID = &id
		}
	}

	// 🖼️ Handle gambar jika ada file baru
	if file, err := c.FormFile("lecture_session_image_url"); err == nil && file != nil {
		// 🔁 Hapus gambar lama dari Supabase jika ada
		if existing.LectureSessionImageURL != nil {
			parsed, err := url.Parse(*existing.LectureSessionImageURL)
			if err == nil {
				rawPath := parsed.Path // /storage/v1/object/public/image/lecture_sessions%2Fxxx.png
				prefix := "/storage/v1/object/public/"
				cleaned := strings.TrimPrefix(rawPath, prefix)

				unescaped, err := url.QueryUnescape(cleaned)
				if err == nil {
					parts := strings.SplitN(unescaped, "/", 2)
					if len(parts) == 2 {
						bucket := parts[0]      // "image"
						objectPath := parts[1]  // "lecture_sessions/xxx.png"
						_ = helper.DeleteFromSupabase(bucket, objectPath)
					}
				}
			}
		}

		// ⬆️ Upload gambar baru
		newURL, err := helper.UploadImageToSupabase("lecture_sessions", file)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar")
		}
		existing.LectureSessionImageURL = &newURL
	}

	// 💾 Simpan perubahan
	if err := ctrl.DB.Save(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update sesi kajian")
	}

	return c.JSON(dto.ToLectureSessionDTO(existing))
}


func (ctrl *LectureSessionController) ApproveLectureSessionByDKM(c *fiber.Ctx) error {
	// 🔎 Ambil dan validasi ID sesi dari URL
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		log.Printf("[ERROR] Invalid session ID: %s", c.Params("id"))
		return fiber.NewError(fiber.StatusBadRequest, "ID sesi tidak valid")
	}

	// 🔐 Validasi role DKM
	role, ok := c.Locals("role").(string)
	if !ok || role != "dkm" {
		log.Printf("[ERROR] Role bukan DKM: %#v", c.Locals("role"))
		return fiber.NewError(fiber.StatusForbidden, "Hanya DKM yang dapat menyetujui sesi")
	}

	// 📦 Ambil data sesi
	var session model.LectureSessionModel
	if err := ctrl.DB.First(&session, "lecture_session_id = ?", sessionID).Error; err != nil {
		log.Printf("[ERROR] Sesi tidak ditemukan: %s", sessionID)
		return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
	}

	// ⏱ Tandai approval by DKM
	now := time.Now()
	session.LectureSessionApprovedByDkmAt = &now

	if err := ctrl.DB.Save(&session).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan approval DKM: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan approval")
	}

	log.Printf("[SUCCESS] DKM menyetujui sesi %s", sessionID)
	return c.JSON(dto.ToLectureSessionDTO(session))
}



func (ctrl *LectureSessionController) ApproveLectureSession(c *fiber.Ctx) error {
	// 🔎 Ambil & validasi UUID sesi
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		log.Printf("[ERROR] Invalid session ID: %s", c.Params("id"))
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// 🔐 Ambil role dari JWT context
	role, ok := c.Locals("role").(string)
	if !ok || role == "" {
		log.Printf("[ERROR] Role tidak valid: %#v", c.Locals("role"))
		return fiber.NewError(fiber.StatusForbidden, "Role tidak valid")
	}

	// 👤 Ambil user_id dari JWT context
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		log.Printf("[ERROR] User ID tidak valid: %#v", c.Locals("user_id"))
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak valid")
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("[ERROR] Gagal parsing UUID dari user_id: %s", userIDStr)
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak valid")
	}

	log.Printf("[INFO] User %s dengan role '%s' meng-approve sesi %s", userUUID, role, sessionID)

	// 🗃 Ambil sesi kajian dari DB
	var session model.LectureSessionModel
	if err := ctrl.DB.First(&session, "lecture_session_id = ?", sessionID).Error; err != nil {
		log.Printf("[ERROR] Sesi kajian tidak ditemukan: %s", sessionID)
		return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	// 🕒 Set approval berdasarkan role
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
	case "dkm":
		session.LectureSessionApprovedByDkmAt = &now
	default:
		log.Printf("[ERROR] Role '%s' tidak memiliki hak approval", role)
		return fiber.NewError(fiber.StatusForbidden, "Role tidak diizinkan untuk melakukan approval")
	}

	// 💾 Simpan ke DB
	if err := ctrl.DB.Save(&session).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan approval: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan approval")
	}

	log.Println("[SUCCESS] Approval berhasil disimpan")
	return c.JSON(dto.ToLectureSessionDTO(session))
}



// 🔴 DELETE /api/a/lecture-sessions/:id
func (ctrl *LectureSessionController) DeleteLectureSession(c *fiber.Ctx) error {
	sessionID := c.Params("id")

	// 🔍 Ambil sesi kajian
	var existing model.LectureSessionModel
	if err := ctrl.DB.First(&existing, "lecture_session_id = ?", sessionID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 🗑️ Hapus gambar dari Supabase jika ada
		if existing.LectureSessionImageURL != nil {
			parsed, err := url.Parse(*existing.LectureSessionImageURL)
			if err == nil {
				rawPath := parsed.Path // /storage/v1/object/public/image/lecture_sessions%2Fxxx.png
				prefix := "/storage/v1/object/public/"
				cleaned := strings.TrimPrefix(rawPath, prefix)

				unescaped, err := url.QueryUnescape(cleaned)
				if err == nil {
					parts := strings.SplitN(unescaped, "/", 2)
					if len(parts) == 2 {
						bucket := parts[0]
						objectPath := parts[1]
						_ = helper.DeleteFromSupabase(bucket, objectPath)
					}
				}
			}
		}

		// ❌ Hapus sesi dari database
		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}

		// 🔢 Kurangi total_lecture_sessions di lectures
		if existing.LectureSessionLectureID != nil {
			if err := tx.Model(&lectureModel.LectureModel{}).
				Where("lecture_id = ?", *existing.LectureSessionLectureID).
				UpdateColumn("total_lecture_sessions", gorm.Expr("GREATEST(COALESCE(total_lecture_sessions, 1) - 1, 0)")).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus sesi kajian")
	}

	return c.JSON(fiber.Map{
		"message": "Sesi kajian berhasil dihapus",
	})
}
