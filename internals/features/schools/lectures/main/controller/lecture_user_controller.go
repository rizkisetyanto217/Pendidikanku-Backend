package controller

import (
	lectureSessionModel "schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	"schoolku_backend/internals/features/schools/lectures/main/dto"
	"schoolku_backend/internals/features/schools/lectures/main/model"
	helper "schoolku_backend/internals/helpers"

	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ✅ GET /public/lectures/by-school-slug/:slug  (pakai JsonList + pagination)
func (ctrl *LectureController) GetLectureBySchoolSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug school tidak boleh kosong")
	}

	// Ambil school_id by slug
	var school struct {
		SchoolID string `gorm:"column:school_id"`
	}
	if err := ctrl.DB.
		Table("schools").
		Select("school_id").
		Where("school_slug = ?", slug).
		Scan(&school).Error; err != nil || school.SchoolID == "" {
		return helper.JsonError(c, fiber.StatusNotFound, "School dengan slug tersebut tidak ditemukan")
	}

	// Pagination
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

	// Count total
	var total int64
	if err := ctrl.DB.
		Model(&model.LectureModel{}).
		Where("lecture_school_id = ?", school.SchoolID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// Fetch data
	var lectures []model.LectureModel
	if err := ctrl.DB.
		Where("lecture_school_id = ?", school.SchoolID).
		Order("lecture_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&lectures).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data lecture")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
	}
	return helper.JsonList(c, dto.ToLectureResponseList(lectures), pagination)
}

// ✅ GET /api/a/lecture-sessions/by-lecture/:id  (pakai JsonList + pagination)
func (ctrl *LectureController) GetLectureSessionsByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	if lectureID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture ID tidak ditemukan di URL")
	}

	type Result struct {
		lectureSessionModel.LectureSessionModel
		UserName     *string `gorm:"column:user_name"`
		LectureTitle string  `gorm:"column:lecture_title"`
	}

	// Pagination
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

	// Count
	var total int64
	if err := ctrl.DB.
		Table("lecture_sessions").
		Where("lecture_session_lecture_id = ?", lectureID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// Data
	var sessions []Result
	if err := ctrl.DB.
		Table("lecture_sessions").
		Select(`
			lecture_sessions.*,
			users.user_name AS user_name,
			lectures.lecture_title AS lecture_title
		`).
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id").
		Joins("LEFT JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lecture_sessions.lecture_session_lecture_id = ?", lectureID).
		Order("lecture_sessions.lecture_session_created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&sessions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian")
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
	return helper.JsonList(c, sessions, pagination)
}

// ✅ GET /api/a/lecture-sessions/by-lecture-slug/:slug  (pakai JsonList + pagination)
func (ctrl *LectureController) GetLectureSessionsByLectureSlug(c *fiber.Ctx) error {
	lectureSlug := c.Params("slug")
	if lectureSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture slug tidak ditemukan di URL")
	}

	type Result struct {
		lectureSessionModel.LectureSessionModel
		UserName     *string `gorm:"column:user_name"`
		LectureTitle string  `gorm:"column:lecture_title"`
	}

	// Pagination
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

	// Count
	var total int64
	if err := ctrl.DB.
		Table("lecture_sessions").
		Joins("LEFT JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_slug = ?", lectureSlug).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// Data
	var sessions []Result
	if err := ctrl.DB.
		Table("lecture_sessions").
		Select(`
			lecture_sessions.*,
			users.user_name AS user_name,
			lectures.lecture_title AS lecture_title
		`).
		Joins("LEFT JOIN users ON users.id = lecture_sessions.lecture_session_teacher_id").
		Joins("LEFT JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_slug = ?", lectureSlug).
		Order("lecture_sessions.lecture_session_created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&sessions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian")
	}

	pagination := fiber.Map{
		"page":         page,
		"limit":        limit,
		"total":        total,
		"total_pages":  int((total + int64(limit) - 1) / int64(limit)),
		"has_next":     int64(page*limit) < total,
		"has_prev":     page > 1,
		"lecture_slug": lectureSlug,
	}
	return helper.JsonList(c, sessions, pagination)
}
