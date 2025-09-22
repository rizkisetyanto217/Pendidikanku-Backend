// internals/features/lembaga/teachers_students/controller/masjid_teacher_controller.go
package controller

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	yDTO "masjidku_backend/internals/features/lembaga/teachers_students/dto"
	yModel "masjidku_backend/internals/features/lembaga/teachers_students/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

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

// --- tiny helpers

func toJSONErr(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return helper.JsonError(c, fe.Code, fe.Message)
	}
	return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
}

func parseDateYYYYMMDD(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("format tanggal harus YYYY-MM-DD")
	}
	return &t, nil
}

// ==== role helpers (idempotent, aware soft-delete) ====

func grantTeacherRole(tx *gorm.DB, userID, masjidID uuid.UUID) error {
	// revive jika ada baris yang soft-deleted untuk kombinasi yang sama
	res := tx.Exec(`
		UPDATE user_roles
		   SET deleted_at = NULL,
		       assigned_at = now()
		 WHERE user_id = ?
		   AND masjid_id = ?
		   AND role_id = (SELECT role_id FROM roles WHERE role_name = 'teacher' LIMIT 1)
		   AND deleted_at IS NOT NULL
	`, userID, masjidID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}

	// insert baru jika belum ada baris alive
	return tx.Exec(`
		INSERT INTO user_roles (user_id, role_id, masjid_id, assigned_at)
		SELECT ?, r.role_id, ?, now()
		  FROM roles r
		 WHERE r.role_name = 'teacher'
		   AND NOT EXISTS (
			 SELECT 1
			   FROM user_roles ur
			  WHERE ur.user_id   = ?
			    AND ur.role_id   = r.role_id
			    AND ur.masjid_id = ?
			    AND ur.deleted_at IS NULL
		   )
	`, userID, masjidID, userID, masjidID).Error
}

func revokeTeacherRole(tx *gorm.DB, userID, masjidID uuid.UUID) error {
	// soft delete baris role teacher yang masih alive pada masjid ini
	return tx.Exec(`
		UPDATE user_roles ur
		   SET deleted_at = now()
		 WHERE ur.user_id = ?
		   AND ur.masjid_id = ?
		   AND ur.role_id = (SELECT role_id FROM roles WHERE role_name = 'teacher' LIMIT 1)
		   AND ur.deleted_at IS NULL
	`, userID, masjidID).Error
}

/* ===================== CREATE ===================== */
// POST /api/a/masjids/:masjid_id/masjid-teachers
func (ctrl *MasjidTeacherController) Create(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	// 🔒 resolve context masjid
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// body (tanpa masjid_id, hanya user_id + optional fields)
	var body yDTO.CreateMasjidTeacherRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request")
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(body); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	// DTO -> model (paksa masjid dari context/params)
	rec, err := body.ToModel(masjidID.String(), body.MasjidTeacherUserID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var created yModel.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// pastikan user ada
		var exists bool
		if err := tx.Raw(`
			SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND deleted_at IS NULL)
		`, rec.MasjidTeacherUserID).Scan(&exists).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca data user")
		}
		if !exists {
			return fiber.NewError(fiber.StatusNotFound, "User tidak ditemukan")
		}

		// cek duplikat alive
		var dup int64
		if err := tx.Model(&yModel.MasjidTeacherModel{}).
			Where("masjid_teacher_masjid_id = ? AND masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL",
				rec.MasjidTeacherMasjidID, rec.MasjidTeacherUserID).
			Count(&dup).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if dup > 0 {
			return fiber.NewError(fiber.StatusConflict, "Pengajar sudah terdaftar")
		}

		// insert guru
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		created = *rec

		// statistik
		if err := ctrl.Stats.EnsureForMasjid(tx, rec.MasjidTeacherMasjidID); err == nil {
			if rec.MasjidTeacherIsActive {
				_ = ctrl.Stats.IncActiveTeachers(tx, rec.MasjidTeacherMasjidID, +1)
			}
		}

		// sinkron role teacher (best-effort)
		if err := grantTeacherRole(tx, rec.MasjidTeacherUserID, rec.MasjidTeacherMasjidID); err != nil {
			log.Printf("[WARN] grant teacher role failed: %v", err)
			// tidak fatal
		}

		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonCreated(c, "Pengajar berhasil ditambahkan", yDTO.NewMasjidTeacherResponse(&created))
}

/* ===================== UPDATE ===================== */
// PATCH /api/a/masjids/:masjid_id/masjid-teachers/:id
func (ctrl *MasjidTeacherController) Update(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}
	rowID, err := uuid.Parse(id)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req yDTO.UpdateMasjidTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	var before yModel.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&before, "masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
			rowID, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Pengajar tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	wasActive := before.MasjidTeacherIsActive

	// apply tri-state
	if err := req.ApplyToModel(&before); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	before.MasjidTeacherUpdatedAt = time.Now()

	// save
	if err := ctrl.DB.WithContext(c.Context()).Save(&before).Error; err != nil {
		return toJSONErr(c, err)
	}

	// statistik & role sinkron jika status aktif berubah
	if wasActive != before.MasjidTeacherIsActive {
		// statistik
		if err := ctrl.Stats.EnsureForMasjid(ctrl.DB, masjidID); err == nil {
			delta := -1
			if before.MasjidTeacherIsActive {
				delta = +1
			}
			_ = ctrl.Stats.IncActiveTeachers(ctrl.DB, masjidID, delta)
		}

		// role teacher
		if before.MasjidTeacherIsActive {
			if err := grantTeacherRole(ctrl.DB, before.MasjidTeacherUserID, before.MasjidTeacherMasjidID); err != nil {
				log.Printf("[WARN] grant teacher role (update) failed: %v", err)
			}
		} else {
			if err := revokeTeacherRole(ctrl.DB, before.MasjidTeacherUserID, before.MasjidTeacherMasjidID); err != nil {
				log.Printf("[WARN] revoke teacher role (update) failed: %v", err)
			}
		}
	}

	return helper.JsonUpdated(c, "Pengajar diperbarui", yDTO.NewMasjidTeacherResponse(&before))
}

/* ===================== DELETE ===================== */
// DELETE /api/a/masjids/:masjid_id/masjid-teachers/:id
func (ctrl *MasjidTeacherController) Delete(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	// 🔒 masjid context + DKM/Admin guard
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var rows int64
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Lock row guru yang mau dihapus
		var teacher yModel.MasjidTeacherModel
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&teacher,
				"masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
				id, masjidID,
			).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Pengajar tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
		}

		// Soft-delete guru
		res := tx.Where("masjid_teacher_id = ?", teacher.MasjidTeacherID).
			Delete(&yModel.MasjidTeacherModel{})
		if res.Error != nil {
			log.Println("[ERROR] delete masjid teacher:", res.Error)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus pengajar")
		}
		rows = res.RowsAffected

		// Update statistik aktif (jika perlu)
		if rows > 0 && teacher.MasjidTeacherIsActive {
			if err := ctrl.Stats.EnsureForMasjid(tx, masjidID); err == nil {
				_ = ctrl.Stats.IncActiveTeachers(tx, masjidID, -1)
			}
		}

		// Cek apakah masih ada record guru lain (alive) utk user & masjid ini
		var remain int64
		if err := tx.Model(&yModel.MasjidTeacherModel{}).
			Where("masjid_teacher_user_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
				teacher.MasjidTeacherUserID, teacher.MasjidTeacherMasjidID).
			Count(&remain).Error; err != nil {
			log.Printf("[WARN] count remaining teachers failed: %v", err)
			return nil
		}

		// Jika sudah tidak ada lagi → cabut role "teacher" pada masjid ini
		if remain == 0 {
			if err := revokeTeacherRole(tx, teacher.MasjidTeacherUserID, teacher.MasjidTeacherMasjidID); err != nil {
				log.Printf("[WARN] revoke teacher role (delete) failed: %v", err)
			}
		}

		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonDeleted(c, "Pengajar berhasil dihapus", fiber.Map{
		"masjid_teacher_id": id,
		"affected":          rows,
	})
}
