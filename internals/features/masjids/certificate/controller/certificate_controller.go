package controllers

import (
	"masjidku_backend/internals/features/masjids/certificate/dto"
	certificateModel "masjidku_backend/internals/features/masjids/certificate/model"
	lectureExamModel "masjidku_backend/internals/features/masjids/lectures/exams/model"
	lectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
	masjidModel "masjidku_backend/internals/features/masjids/masjids/model"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CertificateController struct {
	DB *gorm.DB
}

func NewCertificateController(db *gorm.DB) *CertificateController {
	return &CertificateController{DB: db}
}

// ✅ CREATE
func (ctrl *CertificateController) Create(c *fiber.Ctx) error {
	var body dto.CreateCertificateDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid body",
			"error":   err.Error(),
		})
	}

	cert := certificateModel.CertificateModel{
		CertificateID:          uuid.New(),
		CertificateTitle:       body.CertificateTitle,
		CertificateDescription: body.CertificateDescription,
		CertificateLectureID:   body.CertificateLectureID,
		CertificateTemplateURL: body.CertificateTemplateURL,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	// Mulai transaksi
	tx := ctrl.DB.Begin()

	// Simpan certificate
	if err := tx.Create(&cert).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create certificate",
			"error":   err.Error(),
		})
	}

	// Update Lecture: set LectureIsCerticateGenerated = true
	if err := tx.Model(&lectureModel.LectureModel{}).
		Where("lecture_id = ?", body.CertificateLectureID).
		Update("lecture_is_certificate_generated", true).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Certificate created but failed to update lecture",
			"error":   err.Error(),
		})
	}

	tx.Commit()

	// Return format dengan message + data
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Certificate created successfully",
		"data":    cert,
	})
}


// ✅ GET ALL
func (ctrl *CertificateController) GetAll(c *fiber.Ctx) error {
	var certificates []certificateModel.CertificateModel
	if err := ctrl.DB.Find(&certificates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to fetch data", "error": err.Error()})
	}
	return c.JSON(certificates)
}

// ✅ GET BY ID
func (ctrl *CertificateController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var cert certificateModel.CertificateModel

	if err := ctrl.DB.First(&cert, "certificate_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Certificate not found", "error": err.Error()})
	}

	return c.JSON(cert)
}

func (ctrl *CertificateController) GetByUserExamID(c *fiber.Ctx) error {
	userExamIDParam := c.Params("user_exam_id")
	userExamID, err := uuid.Parse(userExamIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user_exam_id",
			"error":   err.Error(),
		})
	}

	// STEP 1: Ambil data ujian user
	var userExam lectureExamModel.UserLectureExamModel
	if err := ctrl.DB.First(&userExam, "user_lecture_exam_id = ?", userExamID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User exam not found",
			"error":   err.Error(),
		})
	}

	// STEP 2: Ambil exam → untuk mendapatkan lecture_id
	var exam lectureExamModel.LectureExamModel
	if err := ctrl.DB.First(&exam, "lecture_exam_id = ?", userExam.UserLectureExamExamID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Lecture exam not found",
			"error":   err.Error(),
		})
	}

	// STEP 3: Ambil lecture
	var lecture lectureModel.LectureModel
	if err := ctrl.DB.First(&lecture, "lecture_id = ?", exam.LectureExamLectureID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Lecture not found",
			"error":   err.Error(),
		})
	}

	// STEP 4: Ambil certificate berdasarkan lecture_id
	var certificate certificateModel.CertificateModel
	if err := ctrl.DB.First(&certificate, "certificate_lecture_id = ?", lecture.LectureID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Certificate not found for this lecture",
			"error":   err.Error(),
		})
	}

	// STEP 5: Ambil masjid
	var masjid masjidModel.MasjidModel
	if err := ctrl.DB.First(&masjid, "masjid_id = ?", lecture.LectureMasjidID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Masjid not found",
			"error":   err.Error(),
		})
	}

	// STEP 6: Konversi nilai
	var gradeResult *int
	if userExam.UserLectureExamGrade != nil {
		tmp := int(*userExam.UserLectureExamGrade)
		gradeResult = &tmp
	}

	// STEP 7: Bangun response
	response := dto.CertificateDetailResponse{
		CertificateID:                 certificate.CertificateID,
		CertificateTitle:              certificate.CertificateTitle,
		CertificateDescription:        certificate.CertificateDescription,
		CertificateTemplateURL:        certificate.CertificateTemplateURL,
		LectureTitle:                  lecture.LectureTitle,
		LectureIsCertificateGenerated: lecture.LectureIsCertificateGenerated,
		MasjidID:                      masjid.MasjidID,
		MasjidName:                    masjid.MasjidName,
		MasjidImageURL:                &masjid.MasjidImageURL,
		UserLectureExamUserName:      userExam.UserLectureExamUserName,
		UserLectureExamGradeResult:   gradeResult,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}





// ✅ UPDATE
func (ctrl *CertificateController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var cert certificateModel.CertificateModel

	if err := ctrl.DB.First(&cert, "certificate_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Certificate not found", "error": err.Error()})
	}

	var body dto.UpdateCertificateDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid body", "error": err.Error()})
	}

	if body.CertificateTitle != nil {
		cert.CertificateTitle = *body.CertificateTitle
	}
	if body.CertificateDescription != nil {
		cert.CertificateDescription = *body.CertificateDescription
	}
	if body.CertificateLectureID != nil {
		cert.CertificateLectureID = *body.CertificateLectureID
	}
	if body.CertificateTemplateURL != nil {
		cert.CertificateTemplateURL = *body.CertificateTemplateURL
	}
	cert.UpdatedAt = time.Now()

	if err := ctrl.DB.Save(&cert).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update certificate", "error": err.Error()})
	}

	return c.JSON(cert)
}

// ✅ DELETE
func (ctrl *CertificateController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.Delete(&certificateModel.CertificateModel{}, "certificate_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete certificate", "error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Certificate deleted"})
}
