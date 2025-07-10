package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validate2 = validator.New() // ✅ Buat instance validator

type LectureSessionsMaterialController struct {
	DB *gorm.DB
}

func NewLectureSessionsMaterialController(db *gorm.DB) *LectureSessionsMaterialController {
	return &LectureSessionsMaterialController{DB: db}
}

// =============================
// ➕ Create Lecture Session Material
// =============================
func (ctrl *LectureSessionsMaterialController) CreateLectureSessionsMaterial(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsMaterialRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if err := validate2.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	material := model.LectureSessionsMaterialModel{
		LectureSessionsMaterialTitle:            body.LectureSessionsMaterialTitle,
		LectureSessionsMaterialSummary:          body.LectureSessionsMaterialSummary,
		LectureSessionsMaterialTranscriptFull:   body.LectureSessionsMaterialTranscriptFull,
		LectureSessionsMaterialLectureSessionID: body.LectureSessionsMaterialLectureSessionID,
	}

	if err := ctrl.DB.Create(&material).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create material")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionsMaterialDTO(material))
}

// =============================
// 📄 Get All Materials
// =============================
func (ctrl *LectureSessionsMaterialController) GetAllLectureSessionsMaterials(c *fiber.Ctx) error {
	var materials []model.LectureSessionsMaterialModel

	if err := ctrl.DB.Find(&materials).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve materials")
	}

	var response []dto.LectureSessionsMaterialDTO
	for _, m := range materials {
		response = append(response, dto.ToLectureSessionsMaterialDTO(m))
	}

	return c.JSON(response)
}

// =============================
// 🔍 Get Material by ID
// =============================
func (ctrl *LectureSessionsMaterialController) GetLectureSessionsMaterialByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var material model.LectureSessionsMaterialModel
	if err := ctrl.DB.First(&material, "lecture_sessions_material_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Material not found")
	}

	return c.JSON(dto.ToLectureSessionsMaterialDTO(material))
}

// =============================
// ❌ Delete Material
// =============================
func (ctrl *LectureSessionsMaterialController) DeleteLectureSessionsMaterial(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureSessionsMaterialModel{}, "lecture_sessions_material_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete material")
	}

	return c.SendStatus(fiber.StatusNoContent)
}



func (ctrl *LectureSessionsMaterialController) GetContentByLectureID(c *fiber.Ctx) error {
	lectureID := c.Query("lecture_id")
	if lectureID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "lecture_id wajib diisi",
		})
	}

	var sessionIDs []string
	if err := ctrl.DB.
		Table("lecture_sessions").
		Where("lecture_session_lecture_id = ?", lectureID).
		Pluck("lecture_session_id", &sessionIDs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian",
			"error":   err.Error(),
		})
	}

	var materials []model.LectureSessionsMaterialModel
	if len(sessionIDs) > 0 {
		ctrl.DB.
			Where("lecture_sessions_material_lecture_session_id IN ?", sessionIDs).
			Find(&materials)
	}

	var assets []model.LectureSessionsAssetModel
	if len(sessionIDs) > 0 {
		ctrl.DB.
			Where("lecture_sessions_asset_lecture_session_id IN ?", sessionIDs).
			Find(&assets)
	}

	var content []map[string]interface{}
	for _, m := range materials {
		content = append(content, map[string]interface{}{
			"type":       "material",
			"id":         m.LectureSessionsMaterialID,
			"title":      m.LectureSessionsMaterialTitle,
			"summary":    m.LectureSessionsMaterialSummary,
			"transcript": m.LectureSessionsMaterialTranscriptFull,
			"session_id": m.LectureSessionsMaterialLectureSessionID,
			"created_at": m.LectureSessionsMaterialCreatedAt,
		})
	}
	for _, a := range assets {
		content = append(content, map[string]interface{}{
			"type":       "asset",
			"id":         a.LectureSessionsAssetID,
			"title":      a.LectureSessionsAssetTitle,
			"file_url":   a.LectureSessionsAssetFileURL,
			"file_type":  a.LectureSessionsAssetFileType,
			"session_id": a.LectureSessionsAssetLectureSessionID,
			"created_at": a.LectureSessionsAssetCreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil seluruh konten kajian berdasarkan lecture_id",
		"data":    content,
	})
}
