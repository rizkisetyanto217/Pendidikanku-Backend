package controllers

import (
	"time"

	"masjidku_backend/internals/features/masjids/certificate/dto"
	userCertModel "masjidku_backend/internals/features/masjids/certificate/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserCertificateController struct {
	DB *gorm.DB
}

func NewUserCertificateController(db *gorm.DB) *UserCertificateController {
	return &UserCertificateController{DB: db}
}

// ✅ CREATE
func (ctrl *UserCertificateController) Create(c *fiber.Ctx) error {
	var body dto.CreateUserCertificateDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid body", "error": err.Error()})
	}

	newCert := userCertModel.UserCertificateModel{
		UserCertUserID:        body.UserCertUserID,
		UserCertCertificateID: body.UserCertCertificateID,
		UserCertScore:         body.UserCertScore,
		UserCertSlugURL:       body.UserCertSlugURL,
		UserCertIsUpToDate:    true,
		UserCertIssuedAt:      time.Now(),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := ctrl.DB.Create(&newCert).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create user certificate", "error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(newCert)
}

// ✅ GET ALL
func (ctrl *UserCertificateController) GetAll(c *fiber.Ctx) error {
	var certs []userCertModel.UserCertificateModel
	if err := ctrl.DB.Find(&certs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to fetch data", "error": err.Error()})
	}
	return c.JSON(certs)
}

// ✅ GET BY ID
func (ctrl *UserCertificateController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var cert userCertModel.UserCertificateModel

	if err := ctrl.DB.First(&cert, "user_cert_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User Certificate not found", "error": err.Error()})
	}

	return c.JSON(cert)
}

// ✅ UPDATE
func (ctrl *UserCertificateController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var cert userCertModel.UserCertificateModel

	if err := ctrl.DB.First(&cert, "user_cert_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User Certificate not found", "error": err.Error()})
	}

	var body dto.UpdateUserCertificateDTO
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid body", "error": err.Error()})
	}

	if body.UserCertScore != nil {
		cert.UserCertScore = body.UserCertScore
	}
	if body.UserCertSlugURL != nil {
		cert.UserCertSlugURL = *body.UserCertSlugURL
	}
	if body.UserCertIsUpToDate != nil {
		cert.UserCertIsUpToDate = *body.UserCertIsUpToDate
	}

	cert.UpdatedAt = time.Now()

	if err := ctrl.DB.Save(&cert).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update", "error": err.Error()})
	}

	return c.JSON(cert)
}

// ✅ DELETE
func (ctrl *UserCertificateController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.Delete(&userCertModel.UserCertificateModel{}, "user_cert_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete", "error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "User Certificate deleted"})
}
