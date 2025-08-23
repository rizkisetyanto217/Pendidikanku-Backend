package controller

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	ucDTO "masjidku_backend/internals/features/lembaga/classes/main/dto"
	classModel "masjidku_backend/internals/features/lembaga/classes/main/model"
	userModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"

	statsSvc "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
)

type UserClassController struct {
	DB *gorm.DB
	Stats *statsSvc.LembagaStatsService

}

func NewUserClassController(db *gorm.DB) *UserClassController {
	return &UserClassController{
        DB:    db,
        Stats: statsSvc.NewLembagaStatsService(),
    }}

var validateUserClasses = validator.New()

/* ================= Helpers ================= */

func (h *UserClassController) ensureClassBelongsToMasjid(tx *gorm.DB, classID, masjidID uuid.UUID) error {
	var count int64
	if err := tx.Table("classes").
		Where("class_id = ? AND class_masjid_id = ?", classID, masjidID).
		Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa kelas")
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Kelas tidak ditemukan di masjid ini")
	}
	return nil
}

func (h *UserClassController) ensureTermBelongsToMasjid(tx *gorm.DB, termID, masjidID uuid.UUID) error {
	var count int64
	if err := tx.Table("academic_terms").
		Where("academic_terms_id = ? AND academic_terms_masjid_id = ?", termID, masjidID).
		Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa term akademik")
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Term akademik tidak ditemukan di masjid ini")
	}
	return nil
}


func (h *UserClassController) findUserClassWithTenantGuard(userClassID, masjidID uuid.UUID) (*classModel.UserClassesModel, error) {
	var m classModel.UserClassesModel
	if err := h.DB.Model(&classModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where("user_classes_id = ? AND classes.class_masjid_id = ? AND classes.class_deleted_at IS NULL",
			userClassID, masjidID).
		First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Enrolment tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil enrolment")
	}
	return &m, nil
}

// Cek konflik enrolment aktif pada kombinasi (user,class,term,masjid) selain baris yang sedang diupdate
func (h *UserClassController) checkActiveEnrollmentConflict(
	tx *gorm.DB,
	userID, classID, termID, excludeID, masjidID uuid.UUID,
) error {
	var exists bool
	sql := `
		SELECT EXISTS (
			SELECT 1 FROM user_classes
			WHERE user_classes_deleted_at IS NULL
			  AND user_classes_status = 'active'
			  AND user_classes_user_id = ?
			  AND user_classes_class_id = ?
			  AND user_classes_term_id  = ?
			  AND user_classes_masjid_id = ?
			  AND user_classes_id <> ?
			LIMIT 1
		)
	`
	if err := tx.Raw(sql, userID, classID, termID, masjidID, excludeID).Scan(&exists).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa duplikasi enrolment aktif")
	}
	if exists {
		return fiber.NewError(fiber.StatusConflict, "Pengguna sudah memiliki enrolment aktif pada kelas & term ini")
	}
	return nil
}

// internals/features/lembaga/classes/user_classes/main/controller/user_class_controller.go
// file: internals/features/lembaga/classes/user_classes/main/controller/user_class_controller.go

func (h *UserClassController) UpdateUserClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	ucID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil enrolment + tenant guard
	existing, err := h.findUserClassWithTenantGuard(ucID, masjidID)
	if err != nil {
		return err
	}

	var req ucDTO.UpdateUserClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := validateUserClasses.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ==== Mulai transaksi agar update enrolment & stats atomik ====
	return h.DB.Transaction(func(tx *gorm.DB) error {
		// =========================
		// Validasi FK komposit & target nilai
		// =========================
		targetUser := existing.UserClassesUserID
		if req.UserClassesUserID != nil {
			targetUser = *req.UserClassesUserID
		}

		targetClass := existing.UserClassesClassID
		if req.UserClassesClassID != nil {
			// pastikan class milik masjid ini
			if err := h.ensureClassBelongsToMasjid(tx, *req.UserClassesClassID, masjidID); err != nil {
				return err
			}
			targetClass = *req.UserClassesClassID
		}

		// tenant id — sebaiknya tidak boleh berubah; guard jika ada payload
		if req.UserClassesMasjidID != nil && *req.UserClassesMasjidID != masjidID {
			return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak boleh diubah")
		}

		targetTerm := existing.UserClassesTermID
		if req.UserClassesTermID != nil {
			// pastikan term milik masjid ini
			if err := h.ensureTermBelongsToMasjid(tx, *req.UserClassesTermID, masjidID); err != nil {
				return err
			}
			targetTerm = *req.UserClassesTermID
		}

		// status target
		targetStatus := existing.UserClassesStatus
		if req.UserClassesStatus != nil {
			targetStatus = *req.UserClassesStatus
		}

		// =========================
		// Cegah duplikasi enrolment aktif
		// (mengikuti UNIQUE PARTIAL di DB:
		//   uq_uc_active_per_user_class_term(user_id,class_id,term_id,masjid_id) WHERE status='active' AND deleted_at IS NULL)
		// =========================
		if strings.EqualFold(targetStatus, classModel.UserClassStatusActive) {
			if err := h.checkActiveEnrollmentConflict(tx, targetUser, targetClass, targetTerm, existing.UserClassesID, masjidID); err != nil {
				return err
			}
		}

		// --- Deteksi state aktif sebelum/sesudah update (tanpa ended_at)
		wasActive := strings.EqualFold(existing.UserClassesStatus, classModel.UserClassStatusActive)

		// Terapkan perubahan ke model
		req.ApplyToModel(existing)

		// Simpan enrolment
		if err := tx.Model(&classModel.UserClassesModel{}).
			Where("user_classes_id = ?", existing.UserClassesID).
			Updates(existing).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui enrolment")
		}

		nowActive := strings.EqualFold(existing.UserClassesStatus, classModel.UserClassStatusActive)

		// =========================
		// Update stats bila berubah aktif/non-aktif
		// =========================
		delta := 0
		if !wasActive && nowActive {
			delta = +1
		} else if wasActive && !nowActive {
			delta = -1
		}

		if delta != 0 {
			if err := h.Stats.EnsureForMasjid(tx, masjidID); err != nil {
				return err
			}
			if err := h.Stats.IncActiveStudents(tx, masjidID, delta); err != nil {
				return err
			}
		}

		// =========================
		// Promosi role → student saat berubah ke aktif
		// =========================
		if !wasActive && nowActive {
			userID := existing.UserClassesUserID
			if err := tx.Model(&userModel.UserModel{}).
				Where("id = ? AND role = ?", userID, "user").
				Update("role", "student").Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah role user menjadi student")
			}
		}

		return helper.JsonUpdated(c, "Enrolment berhasil diperbarui", ucDTO.NewUserClassResponse(existing))
	})
}


func (h *UserClassController) GetUserClassByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	ucID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findUserClassWithTenantGuard(ucID, masjidID)
	if err != nil {
		return err
	}
	return helper.JsonOK(c, "OK", ucDTO.NewUserClassResponse(m))
}


func (h *UserClassController) ListUserClasses(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c) // benar: tenant guard
	if err != nil {
		return err
	}

	var q ucDTO.ListUserClassQuery
	// default paging
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	// guard pagination
	if q.Limit <= 0 { q.Limit = 20 }
	if q.Limit > 200 { q.Limit = 200 }
	if q.Offset < 0 { q.Offset = 0 }

	tx := h.DB.Model(&classModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where("classes.class_masjid_id = ? AND classes.class_deleted_at IS NULL", masjidID)

	// filters
	if q.UserID != nil {
		tx = tx.Where("user_classes_user_id = ?", *q.UserID)
	}
	if q.ClassID != nil {
		tx = tx.Where("user_classes_class_id = ?", *q.ClassID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_classes_status = ?", strings.TrimSpace(*q.Status))
	}
	if q.ActiveNow != nil && *q.ActiveNow {
		tx = tx.Where("user_classes_status = 'active' AND user_classes_ended_at IS NULL")
	}

	// total (sebelum limit/offset)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// sorting whitelist
	sort := "started_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "started_at_asc":
		tx = tx.Order("user_classes_started_at ASC")
	case "created_at_asc":
		tx = tx.Order("user_classes_created_at ASC")
	case "created_at_desc":
		tx = tx.Order("user_classes_created_at DESC")
	default:
		tx = tx.Order("user_classes_started_at DESC")
	}

	// fetch data
	var rows []classModel.UserClassesModel
	if err := tx.
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]*ucDTO.UserClassResponse, 0, len(rows))
	for i := range rows {
		items = append(items, ucDTO.NewUserClassResponse(&rows[i]))
	}

	// gunakan JsonList agar konsisten: { data, pagination }
	return helper.JsonList(c, items, fiber.Map{
		"limit":  q.Limit,
		"offset": q.Offset,
		"total":  int(total),
	})
}


// file: internals/features/lembaga/classes/user_classes/main/controller/user_class_controller.go

func (h *UserClassController) EndUserClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	ucID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Tenant guard
	m, err := h.findUserClassWithTenantGuard(ucID, masjidID)
	if err != nil {
		return err
	}

	// Jika sudah ended, buat idempotent response
	if strings.EqualFold(m.UserClassesStatus, classModel.UserClassStatusEnded) {
		return helper.JsonUpdated(c, "Enrolment sudah berstatus ended", fiber.Map{
			"user_classes_id":     m.UserClassesID,
			"user_classes_status": classModel.UserClassStatusEnded,
		})
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		// State sebelum update
		wasActive := strings.EqualFold(m.UserClassesStatus, classModel.UserClassStatusActive)

		now := time.Now()
		updates := map[string]any{
			"user_classes_status":     classModel.UserClassStatusEnded,
			"user_classes_updated_at": now,
		}

		if err := tx.Model(&classModel.UserClassesModel{}).
			Where("user_classes_id = ?", m.UserClassesID).
			Updates(updates).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengakhiri enrolment")
		}

		// Jika sebelumnya aktif → sekarang tidak aktif ⇒ decrement counter
		if wasActive {
			if err := h.Stats.EnsureForMasjid(tx, masjidID); err != nil {
				return err
			}
			if err := h.Stats.IncActiveStudents(tx, masjidID, -1); err != nil {
				return err
			}
		}

		return helper.JsonUpdated(c, "Enrolment diakhiri", fiber.Map{
			"user_classes_id":     m.UserClassesID,
			"user_classes_status": classModel.UserClassStatusEnded,
			"updated_at":          now,
		})
	})
}


// file: internals/features/lembaga/classes/user_classes/main/controller/user_class_controller.go

// DELETE /admin/user-classes/:id
func (h *UserClassController) DeleteUserClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	ucID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Tenant guard + pastikan enrolment ada
	m, err := h.findUserClassWithTenantGuard(ucID, masjidID)
	if err != nil {
		return err
	}

	// Opsional: dukung force delete via query ?force=true
	force := strings.EqualFold(c.Query("force"), "true")

	// Default rule: hanya boleh hapus jika status != active (yakni inactive/ended).
	// Untuk menghapus enrolment active, wajib pakai ?force=true atau set non‑aktif via endpoint lain.
	if !force && strings.EqualFold(m.UserClassesStatus, classModel.UserClassStatusActive) {
		return fiber.NewError(
			fiber.StatusConflict,
			"Enrolment masih aktif. Nonaktifkan/akhiri terlebih dahulu atau gunakan ?force=true.",
		)
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		// Track apakah sebelumnya aktif untuk update statistik
		wasActive := strings.EqualFold(m.UserClassesStatus, classModel.UserClassStatusActive)

		// Soft delete (default) atau hard delete (force)
		var delErr error
		if force {
			delErr = tx.Unscoped().
				Where("user_classes_id = ?", m.UserClassesID).
				Delete(&classModel.UserClassesModel{}).Error
		} else {
			delErr = tx.
				Where("user_classes_id = ?", m.UserClassesID).
				Delete(&classModel.UserClassesModel{}).Error
		}
		if delErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus enrolment")
		}

		// Jika menghapus enrolment yang aktif → turunkan counter active students
		if wasActive {
			if err := h.Stats.EnsureForMasjid(tx, masjidID); err != nil {
				return err
			}
			if err := h.Stats.IncActiveStudents(tx, masjidID, -1); err != nil {
				return err
			}
		}

		return helper.JsonDeleted(c, "Enrolment dihapus", fiber.Map{
			"user_classes_id": m.UserClassesID,
			"force":           force,
		})
	})
}
