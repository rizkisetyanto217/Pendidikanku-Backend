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

	// 🔐 Admin-only
	masjidUUID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.FromFiberError(c, err) // sudah mengandung status+pesan dari helper
	}
	masjidIDStr := masjidUUID.String()

	var created model.MasjidTeacher
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// idempotent: cek baris hidup
		var exists int64
		if err := tx.Model(&model.MasjidTeacher{}).
			Where("masjid_teachers_masjid_id = ? AND masjid_teachers_user_id = ? AND masjid_teachers_deleted_at IS NULL",
				masjidIDStr, body.MasjidTeachersUserID).
			Count(&exists).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if exists > 0 {
			return fiber.NewError(fiber.StatusConflict, "Pengajar sudah terdaftar")
		}

		rec := model.MasjidTeacher{
			MasjidTeachersMasjidID: masjidIDStr,
			MasjidTeachersUserID:   body.MasjidTeachersUserID,
		}
		if err := tx.Create(&rec).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menambahkan pengajar")
		}
		created = rec

		// stats
		if err := ctrl.Stats.EnsureForMasjid(tx, masjidUUID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memastikan baris statistik")
		}
		if err := ctrl.Stats.IncActiveTeachers(tx, masjidUUID, +1); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui statistik guru")
		}
		return nil
	}); err != nil {
		return helper.FromFiberError(c, err)
	}

	resp := dto.MasjidTeacherResponse{
		MasjidTeachersID:        created.MasjidTeachersID,
		MasjidTeachersMasjidID:  created.MasjidTeachersMasjidID,
		MasjidTeachersUserID:    created.MasjidTeachersUserID,
		MasjidTeachersCreatedAt: created.MasjidTeachersCreatedAt,
		MasjidTeachersUpdatedAt: created.MasjidTeachersUpdatedAt,
	}
	return helper.Success(c, "Pengajar berhasil ditambahkan", resp)
}




func (ctrl *MasjidTeacherController) GetByMasjid(c *fiber.Ctx) error {
	// 👥 Prefer TEACHER -> UNION masjid_ids -> ADMIN
	masjidUUID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.FromFiberError(c, err)
	}
	masjidID := masjidUUID.String()

	type MasjidTeacherWithName struct {
		MasjidTeachersID        string    `json:"masjid_teachers_id"`
		MasjidTeachersMasjidID  string    `json:"masjid_teachers_masjid_id"`
		MasjidTeachersUserID    string    `json:"masjid_teachers_user_id"`
		UserName                string    `json:"user_name"`
		MasjidTeachersCreatedAt time.Time `json:"masjid_teachers_created_at"`
		MasjidTeachersUpdatedAt time.Time `json:"masjid_teachers_updated_at"`
	}
	var result []MasjidTeacherWithName

	if err := ctrl.DB.WithContext(c.Context()).
		Table("masjid_teachers").
		Select(`masjid_teachers.masjid_teachers_id,
		        masjid_teachers.masjid_teachers_masjid_id,
		        masjid_teachers.masjid_teachers_user_id,
		        users.user_name,
		        masjid_teachers.masjid_teachers_created_at,
		        masjid_teachers.masjid_teachers_updated_at`).
		Joins("JOIN users ON users.id = masjid_teachers.masjid_teachers_user_id").
		Where("masjid_teachers.masjid_teachers_masjid_id = ? AND masjid_teachers.masjid_teachers_deleted_at IS NULL", masjidID).
		Order("masjid_teachers.masjid_teachers_created_at DESC").
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

	// 🔐 Admin-only
	masjidUUID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.FromFiberError(c, err)
	}
	masjidIDStr := masjidUUID.String()

	var rows int64
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var teacher model.MasjidTeacher
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&teacher, "masjid_teachers_id = ? AND masjid_teachers_masjid_id = ? AND masjid_teachers_deleted_at IS NULL", id, masjidIDStr).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Pengajar tidak ditemukan atau sudah dihapus")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
		}

		res := tx.Where("masjid_teachers_id = ?", teacher.MasjidTeachersID).
			Delete(&model.MasjidTeacher{}) // soft delete
		if res.Error != nil {
			log.Println("[ERROR] Failed to delete masjid teacher:", res.Error)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus pengajar")
		}
		rows = res.RowsAffected

		if rows > 0 {
			if err := ctrl.Stats.EnsureForMasjid(tx, masjidUUID); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memastikan baris statistik")
			}
			if err := ctrl.Stats.IncActiveTeachers(tx, masjidUUID, -1); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui statistik guru")
			}
		}
		return nil
	}); err != nil {
		return helper.FromFiberError(c, err)
	}

	return helper.JsonDeleted(c, "Pengajar berhasil dihapus", fiber.Map{
		"masjid_teachers_id": id,
		"affected":           rows,
	})
}
