package controllers

import (
	"time"

	helper "masjidku_backend/internals/helpers"

	masjidModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"
	"masjidku_backend/internals/features/masjids/certificate/dto"
	certificateModel "masjidku_backend/internals/features/masjids/certificate/model"
	lectureExamModel "masjidku_backend/internals/features/masjids/lectures/exams/model"
	lectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"

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
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid body")
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
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}

	// Simpan certificate
	if err := tx.Create(&cert).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create certificate")
	}

	// Update Lecture: set LectureIsCertificateGenerated = true
	if err := tx.Model(&lectureModel.LectureModel{}).
		Where("lecture_id = ?", body.CertificateLectureID).
		Update("lecture_is_certificate_generated", true).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Certificate created but failed to update lecture")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to commit transaction")
	}

	return helper.JsonCreated(c, "Certificate created successfully", cert)
}

// ✅ GET ALL
func (ctrl *CertificateController) GetAll(c *fiber.Ctx) error {
	var certificates []certificateModel.CertificateModel
	if err := ctrl.DB.Find(&certificates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch certificates")
	}
	// Tidak pakai pagination → kirim nil
	return helper.JsonList(c, certificates, nil)
}

// ✅ GET BY ID
func (ctrl *CertificateController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var cert certificateModel.CertificateModel

	if err := ctrl.DB.First(&cert, "certificate_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Certificate not found")
	}

	return helper.JsonOK(c, "OK", cert)
}

// ✅ GET BY USER EXAM ID → detail certificate + lecture + masjid + grade
func (ctrl *CertificateController) GetByUserExamID(c *fiber.Ctx) error {
	userExamIDParam := c.Params("user_exam_id")
	userExamID, err := uuid.Parse(userExamIDParam)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid user_exam_id")
	}

	// 1) user exam
	var userExam lectureExamModel.UserLectureExamModel
	if err := ctrl.DB.First(&userExam, "user_lecture_exam_id = ?", userExamID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "User exam not found")
	}

	// 2) exam
	var exam lectureExamModel.LectureExamModel
	if err := ctrl.DB.First(&exam, "lecture_exam_id = ?", userExam.UserLectureExamExamID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Lecture exam not found")
	}

	// 3) lecture
	var lecture lectureModel.LectureModel
	if err := ctrl.DB.First(&lecture, "lecture_id = ?", exam.LectureExamLectureID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Lecture not found")
	}

	// 4) certificate
	var certificate certificateModel.CertificateModel
	if err := ctrl.DB.First(&certificate, "certificate_lecture_id = ?", lecture.LectureID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Certificate not found for this lecture")
	}

	// 5) masjid
	var masjid masjidModel.MasjidModel
	if err := ctrl.DB.First(&masjid, "masjid_id = ?", lecture.LectureMasjidID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid not found")
	}

	// Nil-safe conversions
	var (
		userName    string
		gradeResult *int
	)
	if userExam.UserLectureExamUserName != nil {
		userName = *userExam.UserLectureExamUserName
	}
	if userExam.UserLectureExamGrade != nil {
		v := int(*userExam.UserLectureExamGrade)
		gradeResult = &v
	}

	// Response
	response := dto.CertificateDetailResponse{
		CertificateID:                 certificate.CertificateID,
		CertificateTitle:              certificate.CertificateTitle,
		CertificateDescription:        certificate.CertificateDescription,
		CertificateTemplateURL:        certificate.CertificateTemplateURL,
		LectureTitle:                  lecture.LectureTitle,
		LectureIsCertificateGenerated: lecture.LectureIsCertificateGenerated,
		MasjidID:                      masjid.MasjidID,
		MasjidName:                    masjid.MasjidName,
		// MasjidImageURL:                &masjid.MasjidImageURL, // pastikan tipe field ini *string di DTO
		UserLectureExamUserName:    userName,    // <- sudah string
		UserLectureExamGradeResult: gradeResult, // <- *int (boleh nil)
	}

	return helper.JsonOK(c, "OK", response)
}

// ✅ UPDATE
func (ctrl *CertificateController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var cert certificateModel.CertificateModel

	if err := ctrl.DB.First(&cert, "certificate_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Certificate not found")
	}

	var body dto.UpdateCertificateDTO
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid body")
	}

	// Partial update
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
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update certificate")
	}

	return helper.JsonUpdated(c, "Certificate updated", cert)
}

// ✅ DELETE
func (ctrl *CertificateController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	// Optional: cek eksistensi
	var exists int64
	if err := ctrl.DB.Model(&certificateModel.CertificateModel{}).
		Where("certificate_id = ?", id).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete certificate")
	}
	if exists == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Certificate not found")
	}

	if err := ctrl.DB.Delete(&certificateModel.CertificateModel{}, "certificate_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete certificate")
	}
	return helper.JsonDeleted(c, "Certificate deleted", fiber.Map{"id": id})
}
