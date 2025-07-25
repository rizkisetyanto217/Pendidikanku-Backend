package controllers

import (
	"masjidku_backend/internals/features/masjids/certificate/dto"
	certificateModel "masjidku_backend/internals/features/masjids/certificate/model"
	lectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
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

// ✅ GET BY LECTURE ID with JOIN (certificate + lecture + masjid)
func (ctrl *CertificateController) GetByLectureIDWithLectureAndMasjid(c *fiber.Ctx) error {
	lectureIDParam := c.Params("lecture_id")
	lectureID, err := uuid.Parse(lectureIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid lecture_id",
			"error":   err.Error(),
		})
	}

	type Result struct {
		certificateModel.CertificateModel
		LectureTitle                 string  `json:"lecture_title"`
		LectureIsCertificateGenerated bool   `json:"lecture_is_certificate_generated"`
		MasjidID                     uuid.UUID `json:"masjid_id"`
		MasjidName                   string  `json:"masjid_name"`
		MasjidImageURL               *string `json:"masjid_image_url"`
	}

	var result Result
	err = ctrl.DB.Table("certificates").
		Select(`certificates.*, 
				 lectures.lecture_title, lectures.lecture_is_certificate_generated,
				 lectures.lecture_masjid_id as masjid_id,
				 masjids.masjid_name, masjids.masjid_image_url`).
		Joins("join lectures on certificates.certificate_lecture_id = lectures.lecture_id").
		Joins("join masjids on lectures.lecture_masjid_id = masjids.masjid_id").
		Where("certificates.certificate_lecture_id = ?", lectureID).
		Scan(&result).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to fetch certificate with lecture and masjid",
			"error":   err.Error(),
		})
	}

	return c.JSON(result)
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
