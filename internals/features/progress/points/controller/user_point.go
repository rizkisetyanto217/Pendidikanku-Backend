package controllers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/progress/points/model" // sesuaikan path
)

type UserPointLogController struct {
	DB *gorm.DB
}

func NewUserPointLogController(db *gorm.DB) *UserPointLogController {
	return &UserPointLogController{DB: db}
}

// 🟢 GET /api/user-point-logs
// Mengambil seluruh riwayat poin milik user berdasarkan token JWT.
// Digunakan untuk menampilkan log aktivitas user seperti kuis, evaluasi, reading, dsb.
func (ctrl *UserPointLogController) GetByUserID(c *fiber.Ctx) error {
	// Ambil user_id dari token JWT (misalnya disimpan via middleware JWT)
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: token tidak valid atau tidak ditemukan",
		})
	}

	userID, ok := userIDRaw.(uuid.UUID)
	if !ok {
		// Fallback jika bentuknya string
		userIDStr, ok := userIDRaw.(string)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id tidak valid dalam token",
			})
		}

		parsedUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id tidak valid",
			})
		}
		userID = parsedUUID
	}

	var logs []model.UserPointLog
	if err := ctrl.DB.
		Where("user_point_log_user_id = ?", userID).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {

		log.Println("[ERROR] Gagal mengambil user_point_logs:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data poin user",
		})
	}

	return c.JSON(fiber.Map{
		"data": logs,
	})
}

// 🟡 POST /api/user-point-logs
// Menambahkan banyak log poin dalam sekali kirim (batch).
// Cocok untuk digunakan oleh service poin dari quiz, reading, exam, dsb.
func (ctrl *UserPointLogController) Create(c *fiber.Ctx) error {
	var input []model.UserPointLog

	if err := c.BodyParser(&input); err != nil {
		log.Println("[ERROR] Body parser gagal:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format input tidak valid",
		})
	}

	if len(input) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Data logs kosong",
		})
	}

	if err := ctrl.DB.Create(&input).Error; err != nil {
		log.Println("[ERROR] Gagal menyimpan logs:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan data poin",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil menambahkan log poin",
		"count":   len(input),
	})
}
