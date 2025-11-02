package controller

import (
	"encoding/json"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"schoolku_backend/internals/features/schools/lecture_sessions/main/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	lectureModel "schoolku_backend/internals/features/schools/lectures/main/model"
	helper "schoolku_backend/internals/helpers"

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

/*
	=========================================================
	  CREATE

=========================================================
*/
func (ctrl *LectureSessionController) CreateLectureSession(c *fiber.Ctx) error {
	// Validasi user login
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User belum login")
	}

	// Ambil school_id dari token
	schoolIDs, ok := c.Locals("school_admin_ids").([]string)
	if !ok || len(schoolIDs) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "School ID tidak ditemukan di token")
	}
	schoolID, err := uuid.Parse(schoolIDs[0])
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "School ID tidak valid")
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

	if title == "" || teacherIDStr == "" || startTimeStr == "" || endTimeStr == "" || lectureIDStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Field wajib ada: title, teacher_id, start_time, end_time, lecture_id")
	}

	teacherID, err := uuid.Parse(teacherIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID guru tidak valid")
	}
	lectureID, err := uuid.Parse(lectureIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tema kajian tidak valid")
	}
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format waktu mulai tidak valid (RFC3339)")
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format waktu selesai tidak valid (RFC3339)")
	}

	// JSON guru untuk lecture_teachers
	teacherObj := map[string]string{"id": teacherID.String(), "name": teacherName}
	teacherJSON, err := json.Marshal(teacherObj)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal konversi teacher ke JSON")
	}

	// Buat objek session (tanpa gambar dulu)
	newSession := model.LectureSessionModel{
		LectureSessionTitle:       title,
		LectureSessionSlug:        dto.GenerateSlug(title),
		LectureSessionDescription: description,
		LectureSessionTeacherID:   teacherID,
		LectureSessionTeacherName: teacherName,
		LectureSessionStartTime:   startTime,
		LectureSessionEndTime:     endTime,
		LectureSessionPlace:       &place,
		LectureSessionLectureID:   &lectureID,
		LectureSessionSchoolID:    schoolID,
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
		// Simpan sesi (tanpa gambar)
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

		// Upload gambar setelah DB ok
		if file, err := c.FormFile("lecture_session_image_url"); err == nil && file != nil {
			url, err := helper.UploadImageToSupabase("lecture_sessions", file)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar")
			}
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
		log.Printf("[ERROR] CreateLectureSession: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat sesi kajian")
	}

	return helper.JsonCreated(c, "Sesi kajian berhasil dibuat", dto.ToLectureSessionDTO(newSession))
}

/*
	=========================================================
	  GET BY ID (+user progress kalau ada)

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionByID(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil user_id dari cookie / header (opsional)
	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}

	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle    string   `gorm:"column:lecture_title"`
		UserName        *string  `gorm:"column:user_name"`
		UserGradeResult *float64 `gorm:"column:user_grade_result"`
	}

	var result JoinedResult
	query := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select(`lecture_sessions.*, lectures.lecture_title, users.user_name`).
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id")

	if userID != "" {
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

	if err := query.Where("lecture_sessions.lecture_session_id = ?", sessionID).
		Scan(&result).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	dtoItem := dto.ToLectureSessionDTOWithLectureTitle(result.LectureSessionModel, result.LectureTitle)
	if dtoItem.LectureSessionTeacherName == "" && result.UserName != nil {
		dtoItem.LectureSessionTeacherName = *result.UserName
	}
	if result.UserGradeResult != nil {
		dtoItem.UserGradeResult = result.UserGradeResult
	}

	return helper.JsonOK(c, "Berhasil mengambil sesi kajian", dtoItem)
}

/*
	=========================================================
	  GET ALL (pagination)

=========================================================
*/
func (ctrl *LectureSessionController) GetAllLectureSessions(c *fiber.Ctx) error {
	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var total int64
	if err := ctrl.DB.Model(&model.LectureSessionModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian")
	}

	var sessions []model.LectureSessionModel
	if err := ctrl.DB.Order("lecture_session_start_time DESC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch lecture sessions")
	}

	result := make([]dto.LectureSessionDTO, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, dto.ToLectureSessionDTO(s))
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
	}
	return helper.JsonList(c, result, pagination)
}

/*
	=========================================================
	  GET by School (pagination)

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionsBySchoolID(c *fiber.Ctx) error {
	schoolID, ok := c.Locals("school_id").(string)
	if !ok || schoolID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "School ID tidak valid atau tidak ditemukan di token")
	}

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// count
	var total int64
	if err := ctrl.DB.Table("lecture_sessions").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_school_id = ?", schoolID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian")
	}

	// data
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle string `gorm:"column:lecture_title"`
	}
	var results []JoinedResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select("lecture_sessions.*, lectures.lecture_title").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_school_id = ?", schoolID).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Limit(limit).Offset(offset).
		Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian berdasarkan school")
	}

	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		response[i] = dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"school_id":   schoolID,
	}
	return helper.JsonList(c, response, pagination)
}

/*
	=========================================================
	  GET by Lecture ID (adaptif: include user progress jika login)

=========================================================
*/
func (ctrl *LectureSessionController) GetByLectureID(c *fiber.Ctx) error {
	type RequestBody struct {
		LectureID string `json:"lecture_id"`
	}
	var body RequestBody
	if err := c.BodyParser(&body); err != nil || body.LectureID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid, lecture_id wajib diisi")
	}
	lectureID, err := uuid.Parse(body.LectureID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture ID tidak valid")
	}

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit < 1 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	userIDRaw := c.Locals("user_id")

	// Tidak login → data biasa
	if userIDRaw == nil {
		var total int64
		if err := ctrl.DB.Model(&model.LectureSessionModel{}).
			Where("lecture_session_lecture_id = ?", lectureID).
			Count(&total).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian")
		}

		var sessions []model.LectureSessionModel
		if err := ctrl.DB.
			Where("lecture_session_lecture_id = ?", lectureID).
			Order("lecture_session_start_time ASC").
			Limit(limit).Offset(offset).
			Find(&sessions).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data sesi kajian")
		}

		response := make([]dto.LectureSessionDTO, len(sessions))
		for i, s := range sessions {
			response[i] = dto.ToLectureSessionDTO(s)
		}

		pagination := fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": int((total + int64(limit) - 1) / int64(limit)),
			"has_next":    int64(page*limit) < total,
			"has_prev":    page > 1,
			"lecture_id":  lectureID,
		}
		return helper.JsonList(c, response, pagination)
	}

	// Login → gabung progress user
	userIDStr, ok := userIDRaw.(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User ID tidak valid")
	}

	// count
	var total int64
	if err := ctrl.DB.Table("lecture_sessions as ls").
		Where("ls.lecture_session_lecture_id = ?", lectureID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi + progres user")
	}

	// data
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
		Limit(limit).Offset(offset).
		Scan(&joined).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data sesi + progres user")
	}

	// response
	out := make([]fiber.Map, len(joined))
	for i, j := range joined {
		out[i] = fiber.Map{
			"lecture_session":        dto.ToLectureSessionDTO(j.LectureSessionModel),
			"user_attendance_status": j.UserAttendanceStatus,
			"user_grade_result":      j.UserGradeResult,
		}
	}
	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"lecture_id":  lectureID,
	}
	return helper.JsonList(c, out, pagination)
}

/*
	=========================================================
	  UPDATE (multipart form)

=========================================================
*/
func (ctrl *LectureSessionController) UpdateLectureSession(c *fiber.Ctx) error {
	idParam := c.Params("id")
	sessionID, err := uuid.Parse(idParam)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Cari sesi
	var existing model.LectureSessionModel
	if err := ctrl.DB.First(&existing, "lecture_session_id = ?", sessionID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	// Partial updates
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

	// Gambar baru?
	if file, err := c.FormFile("lecture_session_image_url"); err == nil && file != nil {
		// Hapus lama
		if existing.LectureSessionImageURL != nil {
			if parsed, err := url.Parse(*existing.LectureSessionImageURL); err == nil {
				rawPath := parsed.Path
				prefix := "/storage/v1/object/public/"
				cleaned := strings.TrimPrefix(rawPath, prefix)
				if unescaped, err := url.QueryUnescape(cleaned); err == nil {
					parts := strings.SplitN(unescaped, "/", 2)
					if len(parts) == 2 {
						_ = helper.DeleteFromSupabase(parts[0], parts[1])
					}
				}
			}
		}
		// Upload baru
		newURL, err := helper.UploadImageToSupabase("lecture_sessions", file)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload gambar")
		}
		existing.LectureSessionImageURL = &newURL
	} else if val := c.FormValue("lecture_session_image_url"); val != "" {
		existing.LectureSessionImageURL = &val
	}

	if err := ctrl.DB.Save(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update sesi kajian")
	}
	return helper.JsonUpdated(c, "Sesi kajian berhasil diperbarui", dto.ToLectureSessionDTO(existing))
}

/*
	=========================================================
	  APPROVALS

=========================================================
*/
func (ctrl *LectureSessionController) ApproveLectureSessionByDKM(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID sesi tidak valid")
	}

	role, ok := c.Locals("role").(string)
	if !ok || role != "dkm" {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya DKM yang dapat menyetujui sesi")
	}

	var session model.LectureSessionModel
	if err := ctrl.DB.First(&session, "lecture_session_id = ?", sessionID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Sesi tidak ditemukan")
	}

	now := time.Now()
	session.LectureSessionApprovedByDkmAt = &now

	if err := ctrl.DB.Save(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan approval")
	}
	return helper.JsonUpdated(c, "Sesi disetujui oleh DKM", dto.ToLectureSessionDTO(session))
}

func (ctrl *LectureSessionController) ApproveLectureSession(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	role, ok := c.Locals("role").(string)
	if !ok || role == "" {
		return helper.JsonError(c, fiber.StatusForbidden, "Role tidak valid")
	}

	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak valid")
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak valid")
	}

	var session model.LectureSessionModel
	if err := ctrl.DB.First(&session, "lecture_session_id = ?", sessionID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
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
	case "dkm":
		session.LectureSessionApprovedByDkmAt = &now
	default:
		return helper.JsonError(c, fiber.StatusForbidden, "Role tidak diizinkan untuk melakukan approval")
	}

	if err := ctrl.DB.Save(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan approval")
	}
	return helper.JsonUpdated(c, "Approval berhasil disimpan", dto.ToLectureSessionDTO(session))
}

/*
	=========================================================
	  DELETE

=========================================================
*/
func (ctrl *LectureSessionController) DeleteLectureSession(c *fiber.Ctx) error {
	sessionID := c.Params("id")

	var existing model.LectureSessionModel
	if err := ctrl.DB.First(&existing, "lecture_session_id = ?", sessionID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// Hapus gambar dari storage jika ada
		if existing.LectureSessionImageURL != nil {
			if parsed, err := url.Parse(*existing.LectureSessionImageURL); err == nil {
				rawPath := parsed.Path
				prefix := "/storage/v1/object/public/"
				cleaned := strings.TrimPrefix(rawPath, prefix)
				if unescaped, err := url.QueryUnescape(cleaned); err == nil {
					parts := strings.SplitN(unescaped, "/", 2)
					if len(parts) == 2 {
						_ = helper.DeleteFromSupabase(parts[0], parts[1])
					}
				}
			}
		}

		// Hapus sesi
		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}

		// Kurangi counter pada lecture
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
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus sesi kajian")
	}

	return helper.JsonDeleted(c, "Sesi kajian berhasil dihapus", fiber.Map{"id": sessionID})
}
