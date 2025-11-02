package controller

import (
	"log"
	"strconv"
	"time"

	certificateModel "schoolku_backend/internals/features/schools/certificate/model"
	"schoolku_backend/internals/features/schools/lectures/exams/dto"
	"schoolku_backend/internals/features/schools/lectures/exams/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLectureExamController struct {
	DB *gorm.DB
}

func NewUserLectureExamController(db *gorm.DB) *UserLectureExamController {
	return &UserLectureExamController{DB: db}
}

// ➕ POST /api/u/user-lecture-exams
func (ctrl *UserLectureExamController) CreateUserLectureExam(c *fiber.Ctx) error {
	var body dto.CreateUserLectureExamRequest
	if err := c.BodyParser(&body); err != nil {
		log.Printf("[ERROR] Failed to parse request body: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	log.Printf("[DEBUG] Payload diterima: %+v", body)

	// Resolve school_id dari slug
	var school struct{ SchoolID string }
	if err := ctrl.DB.Table("schools").
		Select("school_id").
		Where("school_slug = ?", body.UserLectureExamSchoolSlug).
		Scan(&school).Error; err != nil || school.SchoolID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "School tidak ditemukan")
	}
	parsedSchoolID, err := uuid.Parse(school.SchoolID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "School ID invalid")
	}

	// user_id dari token (wajib sesuai migrasi)
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User ID tidak ditemukan di token")
	}

	// Simpan hasil exam user
	newExam := model.UserLectureExamModel{
		UserLectureExamGrade:    body.UserLectureExamGrade,
		UserLectureExamExamID:   body.UserLectureExamExamID,
		UserLectureExamUserID:   userID,
		UserLectureExamSchoolID: parsedSchoolID,
		UserLectureExamUserName: body.UserLectureExamUserName,
	}
	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		log.Printf("[ERROR] Gagal simpan exam: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal simpan data ujian")
	}
	log.Println("[INFO] Progress ujian berhasil disimpan")

	// -------- Sertifikat (best-effort) --------
	var lectureIDStr string
	if err := ctrl.DB.
		Table("lecture_exams").
		Select("lecture_exam_lecture_id").
		Where("lecture_exam_id = ?", body.UserLectureExamExamID).
		Scan(&lectureIDStr).Error; err == nil && lectureIDStr != "" {
		if lectureID, err := uuid.Parse(lectureIDStr); err == nil {
			var cert certificateModel.CertificateModel
			if err := ctrl.DB.Where("certificate_lecture_id = ?", lectureID).First(&cert).Error; err == nil {
				if newExam.UserLectureExamGrade != nil && *newExam.UserLectureExamGrade >= 70 {
					userCert := certificateModel.UserCertificateModel{
						UserCertCertificateID: cert.CertificateID,
						UserCertScore:         toIntPointer(newExam.UserLectureExamGrade),
						UserCertSlugURL:       uuid.New().String(),
						UserCertIsUpToDate:    true,
						UserCertIssuedAt:      time.Now(),
					}
					userCert.UserCertUserID = userID
					_ = ctrl.DB.Create(&userCert).Error
				}
			}
		}
	}

	return helper.JsonCreated(c, "User lecture exam created successfully", dto.ToUserLectureExamDTO(newExam))
}

func toIntPointer(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(*f)
	return &v
}

// 📄 GET /api/a/user-lecture-exams (support pagination)
func (ctrl *UserLectureExamController) GetAllUserLectureExams(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := ctrl.DB.Model(&model.UserLectureExamModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count exams")
	}

	var records []model.UserLectureExamModel
	if err := ctrl.DB.
		Order("user_lecture_exam_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&records).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch exam data")
	}

	resp := make([]dto.UserLectureExamDTO, len(records))
	for i, r := range records {
		resp[i] = dto.ToUserLectureExamDTO(r)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	pagination := fiber.Map{
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
	}

	return helper.JsonList(c, resp, pagination)
}

// 🔍 GET /api/u/user-lecture-exams/:id
func (ctrl *UserLectureExamController) GetUserLectureExamByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID is required")
	}

	var record model.UserLectureExamModel
	if err := ctrl.DB.First(&record, "user_lecture_exam_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Exam record not found")
	}

	return helper.JsonOK(c, "Exam record fetched successfully", dto.ToUserLectureExamDTO(record))
}

// ❌ DELETE /api/u/user-lecture-exams/:id (soft delete)
func (ctrl *UserLectureExamController) DeleteUserLectureExam(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID is required")
	}

	if err := ctrl.DB.Delete(&model.UserLectureExamModel{}, "user_lecture_exam_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete exam record")
	}

	return helper.JsonDeleted(c, "Exam record deleted successfully", fiber.Map{"id": id})
}

// ♻️ (Opsional) Restore exam yg dihapus
func (ctrl *UserLectureExamController) RestoreUserLectureExam(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID is required")
	}

	if err := ctrl.DB.Unscoped().
		Model(&model.UserLectureExamModel{}).
		Where("user_lecture_exam_id = ?", id).
		Update("user_lecture_exam_deleted_at", nil).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to restore exam record")
	}

	return helper.JsonOK(c, "Exam record restored successfully", fiber.Map{"id": id})
}
