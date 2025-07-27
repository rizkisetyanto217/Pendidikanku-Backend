package controller

import (
	"encoding/json"
	"log"
	lectureSessionModel "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"

	"github.com/gofiber/fiber/v2"
)

// ✅ GET /public/lectures/by-masjid-slug/:slug
func (ctrl *LectureController) GetLectureByMasjidSlug(c *fiber.Ctx) error {
	// Ambil slug dari parameter URL
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Slug masjid tidak ditemukan di URL",
		})
	}

	// Ambil masjid_id berdasarkan slug
	var masjid struct {
		MasjidID string `gorm:"column:masjid_id"`
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Scan(&masjid).Error; err != nil || masjid.MasjidID == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Masjid dengan slug tersebut tidak ditemukan",
		})
	}

	// Ambil lectures berdasarkan masjid_id
	var lectures []model.LectureModel
	if err := ctrl.DB.
		Where("lecture_masjid_id = ?", masjid.MasjidID).
		Order("lecture_created_at DESC").
		Find(&lectures).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data lecture",
		})
	}

	if len(lectures) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Belum ada tema kajian untuk masjid ini",
		})
	}

	// Sukses
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Daftar tema kajian berhasil ditemukan",
		"data":    dto.ToLectureResponseList(lectures),
	})
}

// ✅ GET /api/a/lecture-sessions/by-lecture/:id
func (ctrl *LectureController) GetLectureSessionsByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	if lectureID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Lecture ID tidak ditemukan di URL",
		})
	}

	type Result struct {
		lectureSessionModel.LectureSessionModel
		UserName     *string `gorm:"column:user_name"`
		LectureTitle string  `gorm:"column:lecture_title"`
	}

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
		Scan(&sessions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Daftar sesi kajian berhasil ditemukan",
		"data":    sessions,
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
	userID := c.Cookies("user_id")
	if userID == "" {
		userID = c.Get("X-User-Id")
	}
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
