package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"masjidku_backend/internals/features/certificates/certificate_versions/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CertificateVersionController struct {
	DB *gorm.DB
}

func NewCertificateVersionController(db *gorm.DB) *CertificateVersionController {
	return &CertificateVersionController{DB: db}
}

func (ctrl *CertificateVersionController) GetAll(c *fiber.Ctx) error {
	// 🔍 Ambil semua data certificate_versions dari database
	var versions []model.CertificateVersionModel
	if err := ctrl.DB.Find(&versions).Error; err != nil {
		// ❌ Jika gagal ambil data, kirim error 500
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal ambil data versi sertifikat",
		})
	}

	// ✅ Berhasil, kirim data versi dalam bentuk JSON array
	return c.JSON(fiber.Map{
		"data": versions,
	})
}

func (ctrl *CertificateVersionController) GetByID(c *fiber.Ctx) error {
	// 🔹 Ambil parameter id dari URL
	id := c.Params("id")

	// 🔍 Query satu data berdasarkan primary key
	var version model.CertificateVersionModel
	if err := ctrl.DB.First(&version, id).Error; err != nil {
		// ❌ Jika tidak ditemukan, kirim error 404
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Versi sertifikat tidak ditemukan",
		})
	}

	// ✅ Kirim data versi dalam bentuk JSON
	return c.JSON(version)
}

func (ctrl *CertificateVersionController) Create(c *fiber.Ctx) error {
	// 🔄 Parsing payload dari body request ke dalam struct model
	var payload model.CertificateVersionModel
	if err := c.BodyParser(&payload); err != nil {
		// ❌ Jika parsing gagal, kirim error 400
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Payload tidak valid",
		})
	}

	// 🕒 Set created_at (timestamp sekarang)
	payload.CreatedAt = time.Now()

	// ⛔️ Optional: Jika ingin otomatis updatedAt juga
	// payload.UpdatedAt = time.Now()

	// 🔍 Simpan ke database
	if err := ctrl.DB.Create(&payload).Error; err != nil {
		// ❌ Jika gagal insert ke database
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat versi sertifikat",
		})
	}

	// ✅ Kirim kembali data yang telah dibuat
	return c.JSON(payload)
}

func (ctrl *CertificateVersionController) Update(c *fiber.Ctx) error {
	// 🔹 Ambil ID dari parameter URL
	id := c.Params("id")

	// 🔍 Cek apakah data dengan ID tersebut ada di database
	var version model.CertificateVersionModel
	if err := ctrl.DB.First(&version, id).Error; err != nil {
		// ❌ Jika tidak ditemukan, kirim error 404
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Versi tidak ditemukan",
		})
	}

	// 🔄 Parsing data dari body ke map agar fleksibel update
	var updateData map[string]interface{}
	if err := c.BodyParser(&updateData); err != nil {
		// ❌ Jika payload tidak valid
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Payload tidak valid",
		})
	}

	// 🧩 Siapkan map kosong untuk data yang akan diupdate
	updates := map[string]interface{}{}

	// 🔍 Update field "cert_version_note" jika ada
	if cert_version_note, ok := updateData["cert_version_note"].(string); ok {
		updates["cert_version_note"] = cert_version_note
	}

	// 🔍 Update field "certificate_total_themes" jika ada
	if totalThemes, ok := updateData["cert_total_themes"].(float64); ok {
		// ❗ Karena nilai number dari JSON dibaca sebagai float64, kita konversi ke int
		updates["cert_versions_total_themes"] = int(totalThemes)
	}

	// 🕒 Tambahkan updated_at untuk tracking
	now := time.Now()
	updates["updated_at"] = now

	// ⚠️ Jika yang diupdate hanya updated_at saja, anggap tidak ada perubahan signifikan
	if len(updates) == 1 {
		return c.JSON(fiber.Map{
			"message": "Tidak ada perubahan data",
		})
	}

	// 🔧 Lakukan update ke database
	if err := ctrl.DB.Model(&version).Updates(updates).Error; err != nil {
		// ❌ Gagal update ke DB
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal update versi",
		})
	}

	// ✅ Kirim kembali data versi yang telah diupdate
	return c.JSON(version)
}

func (ctrl *CertificateVersionController) Delete(c *fiber.Ctx) error {
	// 🔹 Ambil ID dari parameter URL
	id := c.Params("id")

	// 🧮 Konversi ID ke integer (karena primary key biasanya int)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		// ❌ Jika bukan angka, kirim error 400
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "ID tidak valid",
		})
	}

	// 🔍 Cek apakah versi dengan ID tersebut ada
	var version model.CertificateVersionModel
	if err := ctrl.DB.First(&version, idInt).Error; err != nil {
		// ❌ Tidak ditemukan
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("Versi sertifikat dengan ID %s tidak ditemukan", id),
		})
	}

	// 🗑️ Hapus dari database
	if err := ctrl.DB.Delete(&version).Error; err != nil {
		// ❌ Gagal menghapus dari database
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Gagal menghapus versi dengan ID %s", id),
		})
	}

	// ✅ Berhasil dihapus, kirim konfirmasi
	return c.JSON(fiber.Map{
		"message":    fmt.Sprintf("Versi sertifikat dengan ID %s berhasil dihapus", id),
		"deleted_id": id,
	})
}
