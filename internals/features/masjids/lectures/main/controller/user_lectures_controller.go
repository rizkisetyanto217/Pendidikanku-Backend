package controller

import (
	"encoding/json"
	"log"
	"strconv"

	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==============================
// UserLectureController
// ==============================

type UserLectureController struct {
	DB *gorm.DB
}

func NewUserLectureController(db *gorm.DB) *UserLectureController {
	return &UserLectureController{DB: db}
}

// 🟢 POST /api/a/user-lectures
func (ctrl *UserLectureController) CreateUserLecture(c *fiber.Ctx) error {
	var req dto.UserLectureRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Validasi: lecture & user harus ada
	var cnt int64
	if err := ctrl.DB.Table("lectures").Where("lecture_id = ?", req.UserLectureLectureID).Count(&cnt).Error; err != nil || cnt == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture tidak ditemukan atau tidak valid")
	}
	if err := ctrl.DB.Table("users").Where("id = ?", req.UserLectureUserID).Count(&cnt).Error; err != nil || cnt == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "User tidak ditemukan atau tidak valid")
	}

	newUserLecture := req.ToModel()
	if err := ctrl.DB.Create(newUserLecture).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan partisipasi")
	}

	return helper.JsonCreated(c, "Partisipasi berhasil dicatat", dto.ToUserLectureResponse(newUserLecture))
}

// 🟢 GET /api/a/user-lectures?lecture_id=...   (atau POST /api/u/user-lectures/by-lecture)
func (ctrl *UserLectureController) GetUsersByLecture(c *fiber.Ctx) error {
	// Ambil lecture_id dari query atau body
	lectureIDStr := c.Query("lecture_id")
	if lectureIDStr == "" {
		var payload struct {
			LectureID string `json:"lecture_id"`
		}
		if err := c.BodyParser(&payload); err == nil {
			lectureIDStr = payload.LectureID
		}
	}
	if lectureIDStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "lecture_id wajib dikirim")
	}

	lectureID, err := uuid.Parse(lectureIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "lecture_id tidak valid")
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
	if err := ctrl.DB.Model(&model.UserLectureModel{}).
		Where("user_lecture_lecture_id = ?", lectureID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung peserta")
	}

	// Data
	var participants []model.UserLectureModel
	if err := ctrl.DB.
		Where("user_lecture_lecture_id = ?", lectureID).
		Order("user_lecture_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&participants).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil peserta")
	}

	// Map ke DTO
	res := make([]dto.UserLectureResponse, 0, len(participants))
	for i := range participants {
		res = append(res, *dto.ToUserLectureResponse(&participants[i]))
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"lecture_id":  lectureIDStr,
	}
	return helper.JsonList(c, res, pagination)
}

// ==============================
// LectureController (progress user)
// ==============================

// ✅ GET /api/a/lectures/:id/progress
func (ctrl *LectureController) GetLectureByIDProgressUser(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	var lecture model.LectureModel

	if err := ctrl.DB.First(&lecture, "lecture_id = ?", lectureID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Kajian tidak ditemukan")
	}

	// Lengkapi nama pengajar (jika kosong)
	if lecture.LectureTeachers != nil {
		var teacherList []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(lecture.LectureTeachers, &teacherList); err == nil {
			changed := false
			for i, t := range teacherList {
				if t.ID != "" && t.Name == "" {
					var user struct{ UserName string }
					if err := ctrl.DB.
						Table("users").
						Select("user_name").
						Where("id = ?", t.ID).
						Scan(&user).Error; err == nil && user.UserName != "" {
						teacherList[i].Name = user.UserName
						changed = true
					}
				}
			}
			if changed {
				if updated, err := json.Marshal(teacherList); err == nil {
					lecture.LectureTeachers = updated
				}
			}
		}
	}

	// Ambil user_id dari helper (cookie/header)
	userUUID := helper.GetUserUUID(c)
	userID := userUUID.String()
	log.Println("[INFO] user_id dari request:", userID)

	// Ambil progress
	var userLecture model.UserLectureModel
	err := ctrl.DB.Where("user_lecture_lecture_id = ? AND user_lecture_user_id = ?", lectureID, userID).
		First(&userLecture).Error

	var userProgress map[string]interface{}
	if err == nil {
		userProgress = map[string]interface{}{
			"grade_result":             userLecture.UserLectureGradeResult,
			"total_completed_sessions": userLecture.UserLectureTotalCompletedSessions,
			"is_registered":            userLecture.UserLectureIsRegistered,
			"has_paid":                 userLecture.UserLectureHasPaid,
			"paid_amount":              userLecture.UserLecturePaidAmount,
			"payment_time":             userLecture.UserLecturePaymentTime,
		}
	} else {
		userProgress = nil
	}

	payload := fiber.Map{
		"lecture":       dto.ToLectureResponse(&lecture),
		"user_progress": userProgress,
	}
	return helper.JsonOK(c, "Berhasil mengambil detail kajian", payload)
}

// ✅ GET /api/a/lectures/by-slug/:slug/progress
func (ctrl *LectureController) GetLectureBySlugProgressUser(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var lecture model.LectureModel

	if err := ctrl.DB.First(&lecture, "lecture_slug = ?", slug).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Kajian tidak ditemukan")
	}

	// Lengkapi nama pengajar (jika kosong)
	if lecture.LectureTeachers != nil {
		var teacherList []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(lecture.LectureTeachers, &teacherList); err == nil {
			changed := false
			for i, t := range teacherList {
				if t.ID != "" && t.Name == "" {
					var user struct{ UserName string }
					if err := ctrl.DB.
						Table("users").
						Select("user_name").
						Where("id = ?", t.ID).
						Scan(&user).Error; err == nil && user.UserName != "" {
						teacherList[i].Name = user.UserName
						changed = true
					}
				}
			}
			if changed {
				if updated, err := json.Marshal(teacherList); err == nil {
					lecture.LectureTeachers = updated
				}
			}
		}
	}

	// Ambil user_id
	userUUID := helper.GetUserUUID(c)
	userID := userUUID.String()
	log.Println("[INFO] user_id dari request:", userID)

	// Progress user
	var userLecture model.UserLectureModel
	err := ctrl.DB.Where("user_lecture_lecture_id = ? AND user_lecture_user_id = ?", lecture.LectureID, userID).
		First(&userLecture).Error

	var userProgress map[string]interface{}
	if err == nil {
		userProgress = map[string]interface{}{
			"grade_result":             userLecture.UserLectureGradeResult,
			"total_completed_sessions": userLecture.UserLectureTotalCompletedSessions,
			"is_registered":            userLecture.UserLectureIsRegistered,
			"has_paid":                 userLecture.UserLectureHasPaid,
			"paid_amount":              userLecture.UserLecturePaidAmount,
			"payment_time":             userLecture.UserLecturePaymentTime,
		}
	} else {
		userProgress = nil
	}

	payload := fiber.Map{
		"lecture":       dto.ToLectureResponse(&lecture),
		"user_progress": userProgress,
	}
	return helper.JsonOK(c, "Berhasil mengambil detail kajian", payload)
}
