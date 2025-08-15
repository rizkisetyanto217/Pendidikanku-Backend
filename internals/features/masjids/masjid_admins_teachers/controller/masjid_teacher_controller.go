package controller

import (
	"errors"
	"log"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/dto"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	helper "masjidku_backend/internals/helpers"
	"time"

	statsSvc "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MasjidTeacherController struct {
	DB    *gorm.DB
	Stats *statsSvc.LembagaStatsService
}

func NewMasjidTeacherController(db *gorm.DB) *MasjidTeacherController {
	return &MasjidTeacherController{
		DB:    db,
		Stats: statsSvc.NewLembagaStatsService(),
	}
}
func (ctrl *MasjidTeacherController) Create(c *fiber.Ctx) error {
	var body dto.CreateMasjidTeacherRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request")
	}
	if err := validator.New().Struct(body); err != nil {
		return helper.ValidationError(c, err)
	}

	masjidIDStr, ok := c.Locals("masjid_id").(string)
	if !ok || masjidIDStr == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan atau tidak valid")
	}
	masjidUUID, err := uuid.Parse(masjidIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak valid (UUID)")
	}

	return ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// Cek duplikasi (idempotent behavior)
		var exists int64
		if err := tx.Model(&model.MasjidTeacher{}).
			Where("masjid_teachers_masjid_id = ? AND masjid_teachers_user_id = ?", masjidIDStr, body.MasjidTeachersUserID).
			Count(&exists).Error; err != nil {
			return helper.Error(c, fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if exists > 0 {
			// Sudah terdaftar → kembalikan 409 supaya tak menggandakan counter
			return helper.Error(c, fiber.StatusConflict, "Pengajar sudah terdaftar")
		}

		// Insert
		data := model.MasjidTeacher{
			MasjidTeachersMasjidID: masjidIDStr,
			MasjidTeachersUserID:   body.MasjidTeachersUserID,
		}
		if err := tx.Create(&data).Error; err != nil {
			return helper.Error(c, fiber.StatusInternalServerError, "Gagal menambahkan pengajar")
		}

		// Pastikan baris stats ada, lalu increment guru aktif
		if err := ctrl.Stats.EnsureForMasjid(tx, masjidUUID); err != nil {
			return helper.Error(c, fiber.StatusInternalServerError, "Gagal memastikan baris statistik")
		}
		if err := ctrl.Stats.IncActiveTeachers(tx, masjidUUID, +1); err != nil {
			return helper.Error(c, fiber.StatusInternalServerError, "Gagal memperbarui statistik guru")
		}

		return helper.Success(c, "Pengajar berhasil ditambahkan",
			dto.ToMasjidTeacherResponse(dto.MasjidTeacher(data)),
		)
	})
}



func (ctrl *MasjidTeacherController) GetByMasjid(c *fiber.Ctx) error {
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}

	type MasjidTeacherWithName struct {
		MasjidTeachersID        string    `json:"masjid_teachers_id"`
		MasjidTeachersMasjidID  string    `json:"masjid_teachers_masjid_id"`
		MasjidTeachersUserID    string    `json:"masjid_teachers_user_id"`
		UserName                string    `json:"user_name"`
		MasjidTeachersCreatedAt time.Time `json:"masjid_teachers_created_at"`
		MasjidTeachersUpdatedAt time.Time `json:"masjid_teachers_updated_at"`
	}
	var result []MasjidTeacherWithName

	if err := ctrl.DB.
		Table("masjid_teachers").
		Select("masjid_teachers.*, users.user_name").
		Joins("JOIN users ON users.id = masjid_teachers.masjid_teachers_user_id").
		Where("masjid_teachers.masjid_teachers_masjid_id = ?", masjidID).
		Scan(&result).Error; err != nil {
		log.Println("[ERROR] Gagal join masjid_teachers ke users:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
	}

	return helper.Success(c, "Daftar pengajar ditemukan", fiber.Map{
		"total":    len(result),
		"teachers": result,
	})
}



func (ctrl *MasjidTeacherController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	masjidIDStr, ok := c.Locals("masjid_id").(string)
	if !ok || masjidIDStr == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak ditemukan")
	}
	masjidUUID, err := uuid.Parse(masjidIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Masjid ID tidak valid (UUID)")
	}

	return ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// Ambil row milik tenant (lock opsional untuk keamanan concurrent)
		var teacher model.MasjidTeacher
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&teacher, "masjid_teachers_id = ? AND masjid_teachers_masjid_id = ?", id, masjidIDStr).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.Error(c, fiber.StatusNotFound, "Pengajar tidak ditemukan atau bukan milik masjid kamu")
			}
			return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
		}

		// Hapus
		res := tx.Where("masjid_teachers_id = ?", teacher.MasjidTeachersID).
			Delete(&model.MasjidTeacher{})
		if res.Error != nil {
			log.Println("[ERROR] Failed to delete masjid teacher:", res.Error)
			return helper.Error(c, fiber.StatusInternalServerError, "Gagal menghapus pengajar")
		}

		// Jika benar-benar terhapus → decrement statistik guru aktif
		if res.RowsAffected > 0 {
			// Pastikan baris stats ada dulu
			if err := ctrl.Stats.EnsureForMasjid(tx, masjidUUID); err != nil {
				return helper.Error(c, fiber.StatusInternalServerError, "Gagal memastikan baris statistik")
			}
			if err := ctrl.Stats.IncActiveTeachers(tx, masjidUUID, -1); err != nil {
				return helper.Error(c, fiber.StatusInternalServerError, "Gagal memperbarui statistik guru")
			}
		}

		return helper.JsonDeleted(c, "Pengajar berhasil dihapus", fiber.Map{
			"masjid_teachers_id": id,
		})
	})
}