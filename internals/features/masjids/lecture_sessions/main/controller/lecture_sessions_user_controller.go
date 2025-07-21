package controller

import (
	"fmt"
	"log"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	lectureModel "masjidku_backend/internals/features/masjids/lectures/model"
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

// =============================
// 🌐 GET Lecture Sessions by Masjid Slug (Public)
// =============================
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

	// Join lecture_sessions + lectures, filter berdasarkan masjid_id
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle string `gorm:"column:lecture_title"`
	}

	var results []JoinedResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select("lecture_sessions.*, lectures.lecture_title").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_masjid_id = ?", masjid.MasjidID).
		Order("lecture_sessions.lecture_session_start_time ASC").
		Scan(&results).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian berdasarkan slug masjid",
		})
	}

	// Map ke DTO
	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		response[i] = dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
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

	// Ambil waktu sekarang
	now := time.Now()

	// Join + filter hanya yang akan datang
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle string `gorm:"column:lecture_title"`
	}

	var results []JoinedResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select("lecture_sessions.*, lectures.lecture_title").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
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
		response[i] = dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian mendatang berdasarkan slug masjid",
		"data":    response,
	})
}


func (ctrl *LectureSessionController) GetFinishedLectureSessionsByMasjidSlug(c *fiber.Ctx) error {
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

	// Ambil waktu sekarang
	now := time.Now()

	// Join + filter hanya yang sudah lewat
	type JoinedResult struct {
		model.LectureSessionModel
		LectureTitle string `gorm:"column:lecture_title"`
	}

	var results []JoinedResult
	if err := ctrl.DB.
		Model(&model.LectureSessionModel{}).
		Select("lecture_sessions.*, lectures.lecture_title").
		Joins("JOIN lectures ON lectures.lecture_id = lecture_sessions.lecture_session_lecture_id").
		Where("lectures.lecture_masjid_id = ? AND lecture_sessions.lecture_session_end_time < ?", masjid.MasjidID, now).
		Order("lecture_sessions.lecture_session_start_time DESC").
		Scan(&results).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian yang telah lewat berdasarkan slug masjid",
		})
	}

	// Map ke DTO
	response := make([]dto.LectureSessionDTO, len(results))
	for i, r := range results {
		response[i] = dto.ToLectureSessionDTOWithLectureTitle(r.LectureSessionModel, r.LectureTitle)
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi kajian yang telah lewat berdasarkan slug masjid",
		"data":    response,
	})
}
