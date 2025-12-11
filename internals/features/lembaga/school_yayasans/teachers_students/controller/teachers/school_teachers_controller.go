// internals/features/lembaga/teachers_students/controller/school_teacher_controller.go
package controller

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	teacherDTO "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	teacherModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	statsSvc "madinahsalam_backend/internals/features/lembaga/stats/lembaga_stats/service"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperDbTime "madinahsalam_backend/internals/helpers/dbtime"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SchoolTeacherController struct {
	DB    *gorm.DB
	Stats *statsSvc.LembagaStatsService
}

func NewSchoolTeacherController(db *gorm.DB) *SchoolTeacherController {
	return &SchoolTeacherController{
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

func grantTeacherRole(tx *gorm.DB, userID, schoolID uuid.UUID) error {
	// revive jika ada baris yang soft-deleted untuk kombinasi yang sama
	res := tx.Exec(`
		UPDATE user_roles
		   SET deleted_at = NULL,
		       assigned_at = now()
		 WHERE user_id = ?
		   AND school_id = ?
		   AND role_id = (SELECT role_id FROM roles WHERE role_name = 'teacher' LIMIT 1)
		   AND deleted_at IS NOT NULL
	`, userID, schoolID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}

	// insert baru jika belum ada baris alive
	return tx.Exec(`
		INSERT INTO user_roles (user_id, role_id, school_id, assigned_at)
		SELECT ?, r.role_id, ?, now()
		  FROM roles r
		 WHERE r.role_name = 'teacher'
		   AND NOT EXISTS (
			 SELECT 1
			   FROM user_roles ur
			  WHERE ur.user_id   = ?
			    AND ur.role_id   = r.role_id
			    AND ur.school_id = ?
			    AND ur.deleted_at IS NULL
		   )
	`, userID, schoolID, userID, schoolID).Error
}

func revokeTeacherRole(tx *gorm.DB, userID, schoolID uuid.UUID) error {
	// soft delete baris role teacher yang masih alive pada school ini
	return tx.Exec(`
		UPDATE user_roles ur
		   SET deleted_at = now()
		 WHERE ur.user_id = ?
		   AND ur.school_id = ?
		   AND ur.role_id = (SELECT role_id FROM roles WHERE role_name = 'teacher' LIMIT 1)
		   AND ur.deleted_at IS NULL
	`, userID, schoolID).Error
}

// ===================== CREATE =====================
// POST /api/a/schools/:school_id/school-teachers
func (ctrl *SchoolTeacherController) Create(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	// 🔒 Ambil school_id dari token + pastikan role DKM/Admin
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// body (tanpa school_id; gunakan user_teacher_id + optional fields)
	var body teacherDTO.CreateSchoolTeacherRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request")
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(body); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	// DTO -> model (paksa school dari context/token)
	rec, err := body.ToModel(schoolID.String())
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var created teacherModel.SchoolTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// pastikan user_teacher exists (+ambil user_id untuk role)
		var userIDStr string
		if err := tx.Raw(`
			SELECT user_teacher_user_id::text
			  FROM user_teachers
			 WHERE user_teacher_id = ?
			   AND user_teacher_deleted_at IS NULL
			LIMIT 1
		`, rec.SchoolTeacherUserTeacherID).Scan(&userIDStr).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca data user_teacher")
		}
		if strings.TrimSpace(userIDStr) == "" {
			return fiber.NewError(fiber.StatusNotFound, "Profil pengajar (user_teacher) tidak ditemukan")
		}
		userID, perr := uuid.Parse(userIDStr)
		if perr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "user_id pada user_teacher tidak valid")
		}

		// cek duplikat alive (per school + user_teacher)
		var dup int64
		if err := tx.Model(&teacherModel.SchoolTeacherModel{}).
			Where(`
				school_teacher_school_id = ?
				AND school_teacher_user_teacher_id = ?
				AND school_teacher_deleted_at IS NULL
			`, rec.SchoolTeacherSchoolID, rec.SchoolTeacherUserTeacherID).
			Count(&dup).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if dup > 0 {
			return fiber.NewError(fiber.StatusConflict, "Pengajar sudah terdaftar")
		}

		// set created_at / updated_at pakai timezone sekolah
		if now, _ := helperDbTime.GetDBTime(c); !now.IsZero() {
			if rec.SchoolTeacherCreatedAt.IsZero() {
				rec.SchoolTeacherCreatedAt = now
			}
			rec.SchoolTeacherUpdatedAt = now
		}

		// insert guru
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		created = *rec

		// statistik lembaga (bukan stats per guru — itu dari snapshot lain)
		if err := ctrl.Stats.EnsureForSchool(tx, rec.SchoolTeacherSchoolID); err == nil {
			if rec.SchoolTeacherIsActive {
				_ = ctrl.Stats.IncActiveTeachers(tx, rec.SchoolTeacherSchoolID, +1)
			}
		}

		// sinkron role teacher (best-effort) → pakai userID hasil lookup
		if err := grantTeacherRole(tx, userID, rec.SchoolTeacherSchoolID); err != nil {
			log.Printf("[WARN] grant teacher role failed: %v", err)
			// tidak fatal
		}

		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonCreated(
		c,
		"Pengajar berhasil ditambahkan",
		teacherDTO.NewSchoolTeacherResponse(c, &created),
	)
}

// ===================== UPDATE =====================
// PATCH /api/a/schools/:school_id/school-teachers/:id
func (ctrl *SchoolTeacherController) Update(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	// 🔒 Ambil school_id dari token + pastikan role DKM/Admin
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
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

	var req teacherDTO.UpdateSchoolTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	var before teacherModel.SchoolTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&before,
			"school_teacher_id = ? AND school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL",
			rowID, schoolID,
		).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Pengajar tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	wasActive := before.SchoolTeacherIsActive

	// apply tri-state
	if err := req.ApplyToModel(&before); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// updated_at pakai timezone sekolah
	if now, _ := helperDbTime.GetDBTime(c); !now.IsZero() {
		before.SchoolTeacherUpdatedAt = now
	} else {
		before.SchoolTeacherUpdatedAt = time.Now()
	}

	// save
	if err := ctrl.DB.WithContext(c.Context()).Save(&before).Error; err != nil {
		return toJSONErr(c, err)
	}

	// statistik & role sinkron jika status aktif berubah
	if wasActive != before.SchoolTeacherIsActive {
		// statistik lembaga
		if err := ctrl.Stats.EnsureForSchool(ctrl.DB, schoolID); err == nil {
			delta := -1
			if before.SchoolTeacherIsActive {
				delta = +1
			}
			_ = ctrl.Stats.IncActiveTeachers(ctrl.DB, schoolID, delta)
		}

		// ambil user_id dari user_teachers berdasar user_teacher_id sekarang
		var userID uuid.UUID
		if err := ctrl.DB.WithContext(c.Context()).Raw(`
			SELECT user_teacher_user_id
			  FROM user_teachers
			 WHERE user_teacher_id = ?
			   AND user_teacher_deleted_at IS NULL
			LIMIT 1
		`, before.SchoolTeacherUserTeacherID).Scan(&userID).Error; err != nil {
			log.Printf("[WARN] lookup user_id for role sync failed: %v", err)
		}

		// role teacher
		if userID != uuid.Nil {
			if before.SchoolTeacherIsActive {
				if err := grantTeacherRole(ctrl.DB, userID, before.SchoolTeacherSchoolID); err != nil {
					log.Printf("[WARN] grant teacher role (update) failed: %v", err)
				}
			} else {
				if err := revokeTeacherRole(ctrl.DB, userID, before.SchoolTeacherSchoolID); err != nil {
					log.Printf("[WARN] revoke teacher role (update) failed: %v", err)
				}
			}
		}
	}

	return helper.JsonUpdated(
		c,
		"Pengajar diperbarui",
		teacherDTO.NewSchoolTeacherResponse(c, &before),
	)
}

// ===================== DELETE =====================
// DELETE /api/a/schools/:school_id/school-teachers/:id
func (ctrl *SchoolTeacherController) Delete(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	// 🔒 Ambil school_id dari token + pastikan role DKM/Admin
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var rows int64
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Lock row guru yang mau dihapus
		var teacher teacherModel.SchoolTeacherModel
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&teacher,
				"school_teacher_id = ? AND school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL",
				id, schoolID,
			).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Pengajar tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
		}

		// lookup user_id dari user_teachers
		var userID uuid.UUID
		if err := tx.Raw(`
			SELECT user_teacher_user_id
			  FROM user_teachers
			 WHERE user_teacher_id = ?
			   AND user_teacher_deleted_at IS NULL
			LIMIT 1
		`, teacher.SchoolTeacherUserTeacherID).Scan(&userID).Error; err != nil {
			log.Printf("[WARN] lookup user_id before delete failed: %v", err)
		}

		// Soft-delete guru
		res := tx.Where("school_teacher_id = ?", teacher.SchoolTeacherID).
			Delete(&teacherModel.SchoolTeacherModel{})
		if res.Error != nil {
			log.Println("[ERROR] delete school teacher:", res.Error)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus pengajar")
		}
		rows = res.RowsAffected

		// Update statistik aktif (jika perlu)
		if rows > 0 && teacher.SchoolTeacherIsActive {
			if err := ctrl.Stats.EnsureForSchool(tx, schoolID); err == nil {
				_ = ctrl.Stats.IncActiveTeachers(tx, schoolID, -1)
			}
		}

		// Cek apakah masih ada record guru lain (alive) utk user_teacher ini pada school yang sama
		var remain int64
		if err := tx.Model(&teacherModel.SchoolTeacherModel{}).
			Where(`
				school_teacher_user_teacher_id = ?
				AND school_teacher_school_id = ?
				AND school_teacher_deleted_at IS NULL
			`, teacher.SchoolTeacherUserTeacherID, teacher.SchoolTeacherSchoolID).
			Count(&remain).Error; err != nil {
			log.Printf("[WARN] count remaining teachers failed: %v", err)
			return nil
		}

		// Jika sudah tidak ada lagi → cabut role "teacher" pada school ini
		if remain == 0 && userID != uuid.Nil {
			if err := revokeTeacherRole(tx, userID, teacher.SchoolTeacherSchoolID); err != nil {
				log.Printf("[WARN] revoke teacher role (delete) failed: %v", err)
			}
		}

		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonDeleted(c, "Pengajar berhasil dihapus", fiber.Map{
		"school_teacher_id": id,
		"affected":          rows,
	})
}
