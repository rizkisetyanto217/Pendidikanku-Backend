package controller

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
		return c.Status(400).JSON(fiber.Map{
			"message": "Permintaan tidak valid",
			"error":   err.Error(),
		})
	}
 
	// 🔒 Validasi: pastikan Lecture dan User memang ada
	var count int64
	if err := ctrl.DB.Table("lectures").Where("lecture_id = ?", req.UserLectureLectureID).Count(&count).Error; err != nil || count == 0 {
		return c.Status(400).JSON(fiber.Map{"message": "Lecture tidak ditemukan atau tidak valid"})
	}
	if err := ctrl.DB.Table("users").Where("id = ?", req.UserLectureUserID).Count(&count).Error; err != nil || count == 0 {
		return c.Status(400).JSON(fiber.Map{"message": "User tidak ditemukan atau tidak valid"})
	}

	newUserLecture := req.ToModel()
	if err := ctrl.DB.Create(newUserLecture).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "Gagal menyimpan partisipasi",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Partisipasi berhasil dicatat",
		"data":    dto.ToUserLectureResponse(newUserLecture),
	})
}

// 🟢 GET /api/a/user-lectures?lecture_id=...
// 🟢 POST /api/u/user-lectures/by-lecture
func (ctrl *UserLectureController) GetUsersByLecture(c *fiber.Ctx) error {
	// Ambil dari JSON body
	var payload struct {
		LectureID string `json:"lecture_id"`
	}
	if err := c.BodyParser(&payload); err != nil || payload.LectureID == "" {
		return c.Status(400).JSON(fiber.Map{"message": "lecture_id wajib dikirim"})
	}

	// Validasi UUID
	lectureID, err := uuid.Parse(payload.LectureID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "lecture_id tidak valid", "error": err.Error()})
	}

	// Ambil data peserta dari DB
	var participants []model.UserLectureModel
	if err := ctrl.DB.Where("user_lecture_lecture_id = ?", lectureID).Find(&participants).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengambil peserta", "error": err.Error()})
	}

	// Konversi ke response DTO
	var result []dto.UserLectureResponse
	for _, p := range participants {
		result = append(result, *dto.ToUserLectureResponse(&p))
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil peserta kajian",
		"data":    result,
	})
}




func (ctrl *LectureController) GetLectureByIDProgressUser(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	var lecture model.LectureModel

	if err := ctrl.DB.First(&lecture, "lecture_id = ?", lectureID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"message": "Kajian tidak ditemukan",
			"error":   err.Error(),
		})
	}

	// 🔄 Lengkapi nama pengajar jika kosong
	if lecture.LectureTeachers != nil {
		var teacherList []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		if err := json.Unmarshal(lecture.LectureTeachers, &teacherList); err == nil {
			changed := false
			for i, t := range teacherList {
				if t.ID != "" && t.Name == "" {
					var user struct {
						UserName string
					}
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

	// 🔍 Ambil user_id dari cookie atau header
	userUUID := helper.GetUserUUID(c)
	userID := userUUID.String()
	log.Println("[INFO] user_id dari request:", userID)


	var userLecture model.UserLectureModel
	err := ctrl.DB.Where("user_lecture_lecture_id = ? AND user_lecture_user_id = ?", lectureID, userID).First(&userLecture).Error

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
		userProgress = nil // Atau kosongkan jika user belum ikut
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil detail kajian",
		"data": fiber.Map{
			"lecture":       dto.ToLectureResponse(&lecture),
			"user_progress": userProgress,
		},
	})
}


func (ctrl *LectureController) GetLectureBySlugProgressUser(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var lecture model.LectureModel

	// 🔍 Cari berdasarkan slug, bukan ID
	if err := ctrl.DB.First(&lecture, "lecture_slug = ?", slug).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"message": "Kajian tidak ditemukan",
			"error":   err.Error(),
		})
	}

	// 🔄 Lengkapi nama pengajar jika kosong
	if lecture.LectureTeachers != nil {
		var teacherList []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		if err := json.Unmarshal(lecture.LectureTeachers, &teacherList); err == nil {
			changed := false
			for i, t := range teacherList {
				if t.ID != "" && t.Name == "" {
					var user struct {
						UserName string
					}
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

	// 🔍 Ambil user_id dari cookie atau header
	userUUID := helper.GetUserUUID(c)
	userID := userUUID.String()
	log.Println("[INFO] user_id dari request:", userID)

	// 🔎 Ambil progres user berdasarkan lecture_id
	var userLecture model.UserLectureModel
	err := ctrl.DB.Where("user_lecture_lecture_id = ? AND user_lecture_user_id = ?", lecture.LectureID, userID).First(&userLecture).Error

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

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil detail kajian",
		"data": fiber.Map{
			"lecture":       dto.ToLectureResponse(&lecture),
			"user_progress": userProgress,
		},
	})
}
