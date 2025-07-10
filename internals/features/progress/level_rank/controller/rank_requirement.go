package controller

import (
	"masjidku_backend/internals/features/progress/level_rank/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type RankRequirementController struct {
	DB *gorm.DB
}

func NewRankRequirementController(db *gorm.DB) *RankRequirementController {
	return &RankRequirementController{DB: db}
}

// 🟡 POST /api/rank-requirements
// Menambahkan banyak data rank sekaligus (batch insert).
// Wajib dikirim dalam bentuk array JSON: [{ rank, name, min_level, max_level }, ...]
func (ctrl *RankRequirementController) Create(c *fiber.Ctx) error {
	var inputs []model.RankRequirement

	// ✅ Validasi format input
	if err := c.BodyParser(&inputs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}
	if len(inputs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak boleh kosong"})
	}

	// 🔄 Simpan semua data
	if err := ctrl.DB.Create(&inputs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Data berhasil ditambahkan",
		"data":    inputs,
	})
}

// 🟢 GET /api/rank-requirements
// Mengambil seluruh daftar rank yang tersedia, urut berdasarkan nilai `rank` naik (asc).
// Berguna untuk mapping otomatis saat user naik level.
func (ctrl *RankRequirementController) GetAll(c *fiber.Ctx) error {
	var ranks []model.RankRequirement

	if err := ctrl.DB.Order("rank_req_rank ASC").Find(&ranks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(ranks)
}

// 🟢 GET /api/rank-requirements/:id
// Mengambil satu data rank berdasarkan ID.
// Biasanya digunakan untuk halaman detail atau edit.
func (ctrl *RankRequirementController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var rank model.RankRequirement

	if err := ctrl.DB.First(&rank, "rank_req_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data tidak ditemukan"})
	}

	return c.JSON(rank)
}

// 🟠 PUT /api/rank-requirements/:id
// Mengupdate data rank berdasarkan ID tertentu.
// ID di body akan dikunci dan tidak bisa diubah untuk menjaga konsistensi.
func (ctrl *RankRequirementController) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var input model.RankRequirement
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format tidak valid"})
	}

	var existing model.RankRequirement
	if err := ctrl.DB.First(&existing, "rank_req_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data tidak ditemukan"})
	}

	input.RankReqID = existing.RankReqID

	if err := ctrl.DB.Save(&input).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Data berhasil diupdate",
		"data":    input,
	})
}

// 🔴 DELETE /api/rank-requirements/:id
// Menghapus satu data rank berdasarkan ID dari database.
// Gunakan dengan hati-hati karena ini operasi permanen.
func (ctrl *RankRequirementController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.RankRequirement{}, "rank_req_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Data berhasil dihapus",
	})
}
