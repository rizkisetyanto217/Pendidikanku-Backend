package controller

import (
	"log"
	"masjidku_backend/internals/features/donations/donation_questions/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type DonationQuestionController struct {
	DB *gorm.DB
}

func NewDonationQuestionController(db *gorm.DB) *DonationQuestionController {
	return &DonationQuestionController{DB: db}
}

// ✅ GET all donation_questions
func (ctrl *DonationQuestionController) GetAll(c *fiber.Ctx) error {
	// 🔍 Inisialisasi slice untuk menampung hasil query
	var items []model.DonationQuestionModel

	// 🔄 Query semua data dari tabel donation_questions
	if err := ctrl.DB.Find(&items).Error; err != nil {
		// ❌ Log error dan kirim response 500 jika gagal
		log.Println("[ERROR] Failed to fetch donation questions:", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch donation questions",
		})
	}

	// ✅ Kirim hasil data dalam format JSON
	return c.JSON(fiber.Map{
		"data": items,
	})
}

// ✅ GET donation_question by ID
func (ctrl *DonationQuestionController) GetByID(c *fiber.Ctx) error {
	// 🔹 Ambil parameter ID dari URL
	id := c.Params("id")

	// 🔍 Inisialisasi struct untuk hasil query
	var item model.DonationQuestionModel

	// 🔄 Query data berdasarkan kolom primary key eksplisit
	if err := ctrl.DB.First(&item, "donation_question_id = ?", id).Error; err != nil {
		// ❌ Jika data tidak ditemukan, log dan kirim error 404
		log.Println("[ERROR] Donation question not found:", err)
		return c.Status(404).JSON(fiber.Map{
			"error": "Donation question not found",
		})
	}

	// ✅ Kirim hasil data dalam format JSON
	return c.JSON(item)
}

// ✅ GET donation_questions by donation_id
func (ctrl *DonationQuestionController) GetByDonationID(c *fiber.Ctx) error {
	// 🔹 Ambil parameter donationId dari URL
	donationID := c.Params("donationId")

	// 🔍 Query semua soal yang terkait dengan donation_id
	var items []model.DonationQuestionModel
	if err := ctrl.DB.
		Where("donation_question_donation_id = ?", donationID).
		Find(&items).Error; err != nil {
		// ❌ Jika gagal query, log error dan kirim respons 500
		log.Println("[ERROR] Failed to fetch donation questions by donation_question_donation_id:", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch by donation ID",
		})
	}

	// ✅ Kirim hasil data dalam format JSON
	return c.JSON(fiber.Map{
		"data": items,
	})
}

// ✅ POST create new donation_question
func (ctrl *DonationQuestionController) Create(c *fiber.Ctx) error {
	// 🔄 Parsing request body ke struct model
	var input model.DonationQuestionModel
	if err := c.BodyParser(&input); err != nil {
		// ❌ Payload tidak valid
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// 🧩 Simpan data ke database
	if err := ctrl.DB.Create(&input).Error; err != nil {
		log.Println("[ERROR] Failed to create donation question:", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create donation question",
		})
	}

	// ✅ Kirim respons sukses dan data yang disimpan
	return c.Status(201).JSON(fiber.Map{
		"message": "Donation question created successfully",
		"data":    input,
	})
}

// ✅ PUT update donation_question
func (ctrl *DonationQuestionController) Update(c *fiber.Ctx) error {
	// 🔹 Ambil ID dari parameter URL
	id := c.Params("id")

	// 🔍 Cari data existing berdasarkan ID
	var item model.DonationQuestionModel
	if err := ctrl.DB.First(&item, id).Error; err != nil {
		// ❌ Data tidak ditemukan
		return c.Status(404).JSON(fiber.Map{
			"error": "Donation question not found",
		})
	}

	// 🔄 Update field berdasarkan body request
	if err := c.BodyParser(&item); err != nil {
		// ❌ Payload tidak valid
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	// 🔧 Simpan perubahan ke database
	if err := ctrl.DB.Save(&item).Error; err != nil {
		log.Println("[ERROR] Failed to update donation question:", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update donation question",
		})
	}

	// ✅ Kirim hasil update ke client
	return c.JSON(fiber.Map{
		"message": "Updated successfully",
		"data":    item,
	})
}

// ✅ DELETE donation_question by ID// ✅ DELETE donation_question by ID
func (ctrl *DonationQuestionController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.DonationQuestionModel{}, "donation_question_id = ?", id).Error; err != nil {
		log.Println("[ERROR] Failed to delete donation question:", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete donation question",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Donation question deleted successfully",
	})
}