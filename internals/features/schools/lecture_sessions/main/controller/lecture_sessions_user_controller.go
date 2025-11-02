package controller

import (
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"schoolku_backend/internals/features/schools/lecture_sessions/main/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	lectureModel "schoolku_backend/internals/features/schools/lectures/main/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/*
	=========================================================
	  GET by School ID (param)  -> JSONList + pagination

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionsBySchoolIDParam(c *fiber.Ctx) error {
	schoolID := c.Params("id")
	if schoolID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "School ID tidak ditemukan di parameter URL")
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
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	// count
	var total int64
	if err := ctrl.DB.Table("lecture_sessions").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_school_id = ?", schoolID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian berdasarkan school")
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
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian berdasarkan school ID")
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
	  GET by Session Slug (single) -> JsonOK

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak ditemukan")
	}

	// Ambil user_id opsional
	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}

	type joinedResult struct {
		model.LectureSessionModel
		LectureTitle    string   `gorm:"column:lecture_title"`
		UserName        *string  `gorm:"column:user_name"`
		UserGradeResult *float64 `gorm:"column:user_grade_result"`
	}

	baseSelect := `
		lecture_sessions.*,
		lectures.lecture_title,
		users.user_name
	`

	db := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id").
		Where("lecture_sessions.lecture_session_slug = ?", slug)

	if userID != "" {
		baseSelect += `,
		(
			SELECT u.user_lecture_session_grade_result
			FROM user_lecture_sessions u
			WHERE u.user_lecture_session_user_id = ?
			  AND u.user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id
			ORDER BY u.user_lecture_session_created_at DESC
			LIMIT 1
		) AS user_grade_result`
		db = db.Select(baseSelect, userID)
	} else {
		db = db.Select(baseSelect)
	}

	var result joinedResult
	if err := db.Take(&result).Error; err != nil {
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
	  Grouped by Month (map) -> JsonOK

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionsGroupedByMonth(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug school tidak ditemukan di URL")
	}

	var school struct {
		SchoolID uuid.UUID `gorm:"column:school_id"`
	}
	if err := ctrl.DB.Table("schools").Select("school_id").Where("school_slug = ?", slug).Scan(&school).Error; err != nil || school.SchoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusNotFound, "School dengan slug tersebut tidak ditemukan")
	}

	type MonthlyResult struct {
		Month string `gorm:"column:month"` // "YYYY-MM"
		model.LectureSessionModel
		LectureTitle string  `gorm:"column:lecture_title"`
		UserName     *string `gorm:"column:user_name"`
	}
	var results []MonthlyResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select(`
			TO_CHAR(lecture_sessions.lecture_session_start_time, 'YYYY-MM') AS month,
			lecture_sessions.*,
			lectures.lecture_title,
			users.user_name`).
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id").
		Where("lectures.lecture_school_id = ?", school.SchoolID).
		Order("month DESC, lecture_sessions.lecture_session_start_time ASC").
		Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian per bulan")
	}

	grouped := make(map[string][]dto.LectureSessionDTO)
	for _, r := range results {
		item := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
		if item.LectureSessionTeacherName == "" && r.UserName != nil {
			item.LectureSessionTeacherName = *r.UserName
		}
		grouped[r.Month] = append(grouped[r.Month], item)
	}

	return helper.JsonOK(c, "Berhasil mengambil sesi kajian dikelompokkan per bulan", grouped)
}

/*
	=========================================================
	  Public: by Lecture ID (list) -> JsonList + pagination

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionsByLectureID(c *fiber.Ctx) error {
	lectureIDParam := c.Params("lecture_id")
	lectureID, err := uuid.Parse(lectureIDParam)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture ID tidak valid")
	}

	var lecture lectureModel.LectureModel
	if err := ctrl.DB.Where("lecture_id = ?", lectureID).First(&lecture).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Lecture tidak ditemukan")
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

	var total int64
	if err := ctrl.DB.Model(&model.LectureSessionModel{}).
		Where("lecture_session_lecture_id = ? AND lecture_session_deleted_at IS NULL", lectureID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian")
	}

	var sessions []model.LectureSessionModel
	if err := ctrl.DB.
		Where("lecture_session_lecture_id = ? AND lecture_session_deleted_at IS NULL", lectureID).
		Order("lecture_session_start_time ASC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data sesi kajian")
	}

	resp := make([]dto.LectureSessionDTO, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, dto.ToLectureSessionDTO(s))
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
	return helper.JsonList(c, resp, pagination)
}

/*
	=========================================================
	  by Month (list) -> JsonList + pagination

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionsByMonth(c *fiber.Ctx) error {
	slug := c.Params("slug")
	month := c.Params("month") // "YYYY-MM"
	if slug == "" || month == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug atau bulan tidak ditemukan di URL")
	}

	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}

	var school struct {
		SchoolID uuid.UUID `gorm:"column:school_id"`
	}
	if err := ctrl.DB.Table("schools").Select("school_id").Where("school_slug = ?", slug).Scan(&school).Error; err != nil || school.SchoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	if limit < 1 {
		limit = 30
	}
	if limit > 300 {
		limit = 300
	}
	offset := (page - 1) * limit

	// count
	var total int64
	countQ := ctrl.DB.Table("lecture_sessions").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_school_id = ?", school.SchoolID).
		Where("TO_CHAR(lecture_sessions.lecture_session_start_time, 'YYYY-MM') = ?", month)
	if err := countQ.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian per bulan")
	}

	type Result struct {
		model.LectureSessionModel
		LectureTitle     string   `gorm:"column:lecture_title"`
		UserName         *string  `gorm:"column:user_name"`
		UserGradeResult  *float64 `gorm:"column:user_grade_result"`
		AttendanceStatus *int     `gorm:"column:attendance_status"`
	}
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
	}

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
	}

	var results []Result
	if err := query.
		Select(strings.Join(selectFields, ", ")).
		Where("lectures.lecture_school_id = ?", school.SchoolID).
		Where("TO_CHAR(lecture_sessions.lecture_session_start_time, 'YYYY-MM') = ?", month).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Limit(limit).Offset(offset).
		Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian")
	}

	dtoList := make([]dto.LectureSessionDTO, 0, len(results))
	for _, r := range results {
		item := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
		if item.LectureSessionTeacherName == "" && r.UserName != nil {
			item.LectureSessionTeacherName = *r.UserName
		}
		if r.UserGradeResult != nil {
			item.UserGradeResult = r.UserGradeResult
		}
		if r.AttendanceStatus != nil {
			item.UserAttendanceStatus = r.AttendanceStatus
		}
		dtoList = append(dtoList, item)
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"slug":        slug,
		"month":       month,
	}
	return helper.JsonList(c, dtoList, pagination)
}

/*
	=========================================================
	  by School Slug (list) -> JsonList + pagination

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionsBySchoolSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug school tidak ditemukan di URL")
	}

	var school struct {
		SchoolID uuid.UUID `gorm:"column:school_id"`
	}
	if err := ctrl.DB.Table("schools").Select("school_id").Where("school_slug = ?", slug).Scan(&school).Error; err != nil || school.SchoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusNotFound, "School dengan slug tersebut tidak ditemukan")
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
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	// count
	var total int64
	if err := ctrl.DB.Table("lecture_sessions").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_school_id = ?", school.SchoolID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian")
	}

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
		Where("lectures.lecture_school_id = ?", school.SchoolID).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Limit(limit).Offset(offset).
		Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian berdasarkan slug school")
	}

	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		dtoItem := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
		if dtoItem.LectureSessionTeacherName == "" && r.UserName != nil {
			dtoItem.LectureSessionTeacherName = *r.UserName
		}
		response[i] = dtoItem
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"slug":        slug,
	}
	return helper.JsonList(c, response, pagination)
}

/*
	=========================================================
	  Upcoming by School Slug (list) -> JsonList + pagination

=========================================================
*/
func (ctrl *LectureSessionController) GetUpcomingLectureSessionsBySchoolSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug school tidak ditemukan di URL")
	}

	var school struct {
		SchoolID uuid.UUID `gorm:"column:school_id"`
	}
	if err := ctrl.DB.Table("schools").Select("school_id").Where("school_slug = ?", slug).Scan(&school).Error; err != nil || school.SchoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusNotFound, "School dengan slug tersebut tidak ditemukan")
	}

	now := time.Now()

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	// count
	var total int64
	if err := ctrl.DB.Table("lecture_sessions").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_school_id = ? AND lecture_sessions.lecture_session_start_time > ?", school.SchoolID, now).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian mendatang")
	}

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
		Where("lectures.lecture_school_id = ? AND lecture_sessions.lecture_session_start_time > ?", school.SchoolID, now).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Limit(limit).Offset(offset).
		Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian mendatang berdasarkan slug school")
	}

	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		dtoItem := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
		if dtoItem.LectureSessionTeacherName == "" && r.UserName != nil {
			dtoItem.LectureSessionTeacherName = *r.UserName
		}
		response[i] = dtoItem
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"slug":        slug,
	}
	return helper.JsonList(c, response, pagination)
}

/*
	=========================================================
	  Finished by School Slug (list) -> JsonList + pagination

=========================================================
*/
func (ctrl *LectureSessionController) GetFinishedLectureSessionsBySchoolSlug(c *fiber.Ctx) error {
	log.Println("🟢 GET /api/u/schools/:slug/finished-lecture-sessions")

	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug school tidak ditemukan di URL")
	}

	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}

	var school struct{ SchoolID uuid.UUID }
	if err := ctrl.DB.Table("schools").Select("school_id").Where("school_slug = ?", slug).Scan(&school).Error; err != nil || school.SchoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusNotFound, "School dengan slug tersebut tidak ditemukan")
	}

	now := time.Now()

	attendanceOnly := c.Query("attendance_only") == "true"

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	// count
	countQ := ctrl.DB.Table("lecture_sessions").
		Where("lecture_session_school_id = ? AND lecture_session_end_time < ?", school.SchoolID, now)
	if attendanceOnly && userID != "" {
		countQ = countQ.Joins(`LEFT JOIN user_lecture_sessions_attendance 
			ON user_lecture_sessions_attendance.user_lecture_sessions_attendance_lecture_session_id = lecture_sessions.lecture_session_id 
			AND user_lecture_sessions_attendance.user_lecture_sessions_attendance_user_id = ?`, userID).
			Where("user_lecture_sessions_attendance.user_lecture_sessions_attendance_status IS NOT NULL")
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi kajian")
	}

	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle     string   `gorm:"column:lecture_title"`
		UserName         *string  `gorm:"column:user_name"`
		UserGradeResult  *float64 `gorm:"column:user_grade_result"`
		AttendanceStatus *int     `gorm:"column:attendance_status"`
	}
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
	}
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
		if attendanceOnly {
			query = query.Where("user_lecture_sessions_attendance.user_lecture_sessions_attendance_status IS NOT NULL")
		}
	}

	var results []JoinedResult
	if err := query.
		Select(strings.Join(selectFields, ", ")).
		Where("lecture_sessions.lecture_session_school_id = ? AND lecture_sessions.lecture_session_end_time < ?", school.SchoolID, now).
		Order("lecture_sessions.lecture_session_start_time DESC").
		Limit(limit).Offset(offset).
		Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian")
	}

	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		dtoItem := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
		if dtoItem.LectureSessionTeacherName == "" && r.UserName != nil {
			dtoItem.LectureSessionTeacherName = *r.UserName
		}
		if r.UserGradeResult != nil {
			dtoItem.UserGradeResult = r.UserGradeResult
		}
		if r.AttendanceStatus != nil {
			dtoItem.UserAttendanceStatus = r.AttendanceStatus
		}
		response[i] = dtoItem
	}

	pagination := fiber.Map{
		"page":            page,
		"limit":           limit,
		"total":           total,
		"total_pages":     int((total + int64(limit) - 1) / int64(limit)),
		"has_next":        int64(page*limit) < total,
		"has_prev":        page > 1,
		"slug":            slug,
		"attendance_only": attendanceOnly,
	}
	return helper.JsonList(c, response, pagination)
}

/*
	=========================================================
	  All Sessions by Lecture Slug (split upcoming/finished) -> JsonOK

=========================================================
*/
func (ctrl *LectureSessionController) GetAllLectureSessionsByLectureSlug(c *fiber.Ctx) error {
	lectureSlug := c.Params("lecture_slug")
	if lectureSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture slug tidak ditemukan di URL")
	}
	attendanceOnly := c.Query("attendance_only") == "true"

	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}

	type LectureLite struct {
		LectureID    uuid.UUID `gorm:"column:lecture_id"`
		LectureTitle string    `gorm:"column:lecture_title"`
	}
	var lec LectureLite
	if err := ctrl.DB.Table("lectures").
		Select("lecture_id, lecture_title").
		Where("lecture_slug = ?", lectureSlug).
		Scan(&lec).Error; err != nil || lec.LectureID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Lecture dengan slug tsb tidak ditemukan")
	}

	now := time.Now()
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle     string   `gorm:"column:lecture_title"`
		UserName         *string  `gorm:"column:user_name"`
		UserGradeResult  *float64 `gorm:"column:user_grade_result"`
		AttendanceStatus *int     `gorm:"column:attendance_status"`
		Status           string   `gorm:"column:status"`
	}

	selectFields := []string{
		"lecture_sessions.*",
		"lectures.lecture_title",
		"users.user_name",
		"CASE WHEN lecture_sessions.lecture_session_end_time < ? THEN 'finished' ELSE 'upcoming' END AS status",
	}
	args := []any{now}

	q := ctrl.DB.Model(&model.LectureSessionModel{}).
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id").
		Where("lecture_sessions.lecture_session_lecture_id = ?", lec.LectureID)

	if userID != "" {
		selectFields = append(selectFields,
			"user_lecture_sessions.user_lecture_session_grade_result AS user_grade_result",
			"user_lecture_sessions_attendance.user_lecture_sessions_attendance_status AS attendance_status",
		)
		q = q.
			Joins(`LEFT JOIN user_lecture_sessions 
				ON user_lecture_sessions.user_lecture_session_lecture_session_id = lecture_sessions.lecture_session_id 
				AND user_lecture_sessions.user_lecture_session_user_id = ?`, userID).
			Joins(`LEFT JOIN user_lecture_sessions_attendance 
				ON user_lecture_sessions_attendance.user_lecture_sessions_attendance_lecture_session_id = lecture_sessions.lecture_session_id 
				AND user_lecture_sessions_attendance.user_lecture_sessions_attendance_user_id = ?`, userID)
		if attendanceOnly {
			q = q.Where("user_lecture_sessions_attendance.user_lecture_sessions_attendance_status IS NOT NULL")
		}
	}

	var rows []JoinedResult
	if err := q.Select(strings.Join(selectFields, ", "), args...).
		Order("CASE WHEN lecture_sessions.lecture_session_end_time < NOW() THEN 1 ELSE 0 END ASC").
		Order("lecture_sessions.lecture_session_start_time ASC").
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi by lecture slug")
	}

	upcoming := make([]dto.LectureSessionDTO, 0, len(rows))
	finished := make([]dto.LectureSessionDTO, 0, len(rows))
	for _, r := range rows {
		item := dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
		if item.LectureSessionTeacherName == "" && r.UserName != nil {
			item.LectureSessionTeacherName = *r.UserName
		}
		if r.UserGradeResult != nil {
			item.UserGradeResult = r.UserGradeResult
		}
		if r.AttendanceStatus != nil {
			item.UserAttendanceStatus = r.AttendanceStatus
		}
		if r.Status == "finished" {
			finished = append(finished, item)
		} else {
			upcoming = append(upcoming, item)
		}
	}
	// finished DESC
	sort.SliceStable(finished, func(i, j int) bool {
		return finished[i].LectureSessionStartTime.After(finished[j].LectureSessionStartTime)
	})

	return helper.JsonOK(c, "Berhasil ambil semua sesi berdasarkan lecture slug", fiber.Map{
		"lecture":  fiber.Map{"lecture_id": lec.LectureID, "lecture_title": lec.LectureTitle, "lecture_slug": lectureSlug},
		"upcoming": upcoming,
		"finished": finished,
	})
}

/*
	=========================================================
	  GET by ID + user progress (single) -> JsonOK

=========================================================
*/
func (ctrl *LectureSessionController) GetLectureSessionByIDProgressUser(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

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

	if err := query.Where("lecture_sessions.lecture_session_id = ?", sessionID).Scan(&result).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
	}

	dtoItem := dto.ToLectureSessionDTOWithLectureTitle(result.LectureSessionModel, result.LectureTitle)
	if dtoItem.LectureSessionTeacherName == "" && result.UserName != nil {
		dtoItem.LectureSessionTeacherName = *result.UserName
	}
	if result.UserGradeResult != nil {
		dtoItem.UserGradeResult = result.UserGradeResult
	}

	return helper.JsonOK(c, "Berhasil mengambil detail sesi kajian", dtoItem)
}
