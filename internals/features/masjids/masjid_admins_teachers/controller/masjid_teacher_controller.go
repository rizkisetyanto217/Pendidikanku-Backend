package controller

import (
	"errors"
	"log"
	"time"

	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/dto"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	helper "masjidku_backend/internals/helpers"

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

/* ============================================
   POST /api/a/masjid-teachers
   Body: { "masjid_teacher_user_id": "<uuid>" }
   (masjid didapat dari token)
   ============================================ */
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
		return helper.FromFiberError(c, err)
	}

	// Parse user_id (validated as uuid by validator)
	userUUID, err := uuid.Parse(body.MasjidTeacherUserID)
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "masjid_teacher_user_id tidak valid")
	}

	var created model.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// idempotent: cek baris hidup
		var exists int64
		if err := tx.Model(&model.MasjidTeacherModel{}).
			Where(
				"masjid_teacher_masjid_id = ? AND masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL",
				masjidUUID, userUUID,
			).
			Count(&exists).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if exists > 0 {
			return fiber.NewError(fiber.StatusConflict, "Pengajar sudah terdaftar")
		}

		rec := model.MasjidTeacherModel{
			MasjidTeacherMasjidID: masjidUUID,
			MasjidTeacherUserID:   userUUID,
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
		MasjidTeacherID:        created.MasjidTeacherID.String(),
		MasjidTeacherMasjidID:  created.MasjidTeacherMasjidID.String(),
		MasjidTeacherUserID:    created.MasjidTeacherUserID.String(),
		MasjidTeacherCreatedAt: created.MasjidTeacherCreatedAt,
		MasjidTeacherUpdatedAt: created.MasjidTeacherUpdatedAt,
	}
	return helper.Success(c, "Pengajar berhasil ditambahkan", resp)
}

/* ============================================
   GET /api/a/masjid-teachers/by-masjid
   (masjid diambil dari token prefer TEACHER)
   ============================================ */
func (ctrl *MasjidTeacherController) GetByMasjid(c *fiber.Ctx) error {
	// 👥 Prefer TEACHER -> UNION masjid_ids -> ADMIN
	masjidUUID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.FromFiberError(c, err)
	}

	type MasjidTeacherWithName struct {
		MasjidTeacherID        string    `json:"masjid_teacher_id"`
		MasjidTeacherMasjidID  string    `json:"masjid_teacher_masjid_id"`
		MasjidTeacherUserID    string    `json:"masjid_teacher_user_id"`
		UserName               string    `json:"user_name"`
		MasjidTeacherCreatedAt time.Time `json:"masjid_teacher_created_at"`
		MasjidTeacherUpdatedAt time.Time `json:"masjid_teacher_updated_at"`
	}
	var result []MasjidTeacherWithName

	if err := ctrl.DB.WithContext(c.Context()).
		Table("masjid_teachers").
		Select(`
			masjid_teachers.masjid_teacher_id        AS masjid_teacher_id,
			masjid_teachers.masjid_teacher_masjid_id AS masjid_teacher_masjid_id,
			masjid_teachers.masjid_teacher_user_id   AS masjid_teacher_user_id,
			users.user_name                           AS user_name,
			masjid_teachers.masjid_teacher_created_at AS masjid_teacher_created_at,
			masjid_teachers.masjid_teacher_updated_at AS masjid_teacher_updated_at
		`).
		Joins("JOIN users ON users.id = masjid_teachers.masjid_teacher_user_id").
		Where("masjid_teachers.masjid_teacher_masjid_id = ? AND masjid_teachers.masjid_teacher_deleted_at IS NULL", masjidUUID).
		Order("masjid_teachers.masjid_teacher_created_at DESC").
		Scan(&result).Error; err != nil {
		log.Println("[ERROR] Gagal join masjid_teachers ke users:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
	}

	return helper.Success(c, "Daftar pengajar ditemukan", fiber.Map{
		"total":    len(result),
		"teachers": result,
	})
}

/* ============================================
   DELETE /api/a/masjid-teachers/:id
   Soft delete + update statistik
   ============================================ */
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

	var rows int64
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var teacher model.MasjidTeacherModel
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&teacher,
				"masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
				id, masjidUUID,
			).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Pengajar tidak ditemukan atau sudah dihapus")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
		}

		res := tx.Where("masjid_teacher_id = ?", teacher.MasjidTeacherID).
			Delete(&model.MasjidTeacherModel{}) // soft delete
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
		"masjid_teacher_id": id,
		"affected":          rows,
	})
}
