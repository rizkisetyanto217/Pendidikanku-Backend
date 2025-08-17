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

func (h *UserClassController) ensureClassBelongsToMasjid(classID, masjidID uuid.UUID) error {
	var cls classModel.ClassModel
	if err := h.DB.Select("class_id, class_masjid_id, class_deleted_at").
		First(&cls, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
	}
	if cls.ClassMasjidID == nil || *cls.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Class bukan milik masjid Anda")
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

func (h *UserClassController) checkActiveEnrollmentConflict(userID, classID, excludeID, masjidID uuid.UUID) error {
	var cnt int64
	tx := h.DB.Model(&classModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where("user_classes_user_id = ? AND user_classes_class_id = ? AND user_classes_status = 'active' AND user_classes_ended_at IS NULL",
			userID, classID).
		Where("classes.class_masjid_id = ? AND classes.class_deleted_at IS NULL", masjidID)

	if excludeID != uuid.Nil {
		tx = tx.Where("user_classes_id <> ?", excludeID)
	}

	if err := tx.Count(&cnt).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi enrolment aktif")
	}
	if cnt > 0 {
		return fiber.NewError(fiber.StatusConflict, "Sudah ada enrolment aktif untuk user & class ini")
	}
	return nil
}


// internals/features/lembaga/classes/user_classes/main/controller/user_class_controller.go

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
		// Jika class diganti, pastikan class milik masjid yang sama
		targetClass := existing.UserClassesClassID
		if req.UserClassesClassID != nil {
			if err := h.ensureClassBelongsToMasjid(*req.UserClassesClassID, masjidID); err != nil {
				return err
			}
			targetClass = *req.UserClassesClassID
		}

		// Hitung status/ended_at target (untuk validasi unik enrolment aktif)
		status := existing.UserClassesStatus
		if req.UserClassesStatus != nil {
			status = *req.UserClassesStatus
		}
		endedAt := existing.UserClassesEndedAt
		if req.UserClassesEndedAt != nil {
			endedAt = req.UserClassesEndedAt
		}

		// Jika akan menjadi "aktif" tanpa ended_at, pastikan tidak duplikat enrolment aktif
		if strings.EqualFold(status, classModel.UserClassStatusActive) && endedAt == nil {
			targetUser := existing.UserClassesUserID
			if req.UserClassesUserID != nil {
				targetUser = *req.UserClassesUserID
			}
			if err := h.checkActiveEnrollmentConflict(targetUser, targetClass, existing.UserClassesID, masjidID); err != nil {
				return err
			}
		}

		// ===== Aturan started_at =====
		// 1) Jangan izinkan payload mengubah started_at langsung
		req.UserClassesStartedAt = nil

		// 2) Jika TRANSISI: non-aktif -> aktif, dan sebelumnya started_at masih kosong, isi sekarang
		shouldSetStart := false
		var startAt time.Time
		if existing.UserClassesStatus != classModel.UserClassStatusActive &&
			strings.EqualFold(status, classModel.UserClassStatusActive) &&
			existing.UserClassesStartedAt == nil {
			startAt = time.Now()
			shouldSetStart = true
		}

		// --- Deteksi state aktif sebelum update ---
		wasActive := isActive(existing.UserClassesStatus, existing.UserClassesEndedAt)

		// Terapkan perubahan lain ke model (bisa mengubah user_id/class_id/status/ended_at)
		req.ApplyToModel(existing)
		if shouldSetStart {
			existing.UserClassesStartedAt = &startAt
		}

		// Simpan enrolment
		if err := tx.Model(&classModel.UserClassesModel{}).
			Where("user_classes_id = ?", existing.UserClassesID).
			Updates(existing).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui enrolment")
		}

		// --- Deteksi state aktif sesudah update ---
		nowActive := isActive(existing.UserClassesStatus, existing.UserClassesEndedAt)

		// Jika ada perubahan state aktif/non-aktif → update stats
		delta := 0
		if !wasActive && nowActive {
			delta = +1
		} else if wasActive && !nowActive {
			delta = -1
		}

		if delta != 0 {
			// Pastikan baris stats ada
			if err := h.Stats.EnsureForMasjid(tx, masjidID); err != nil {
				return err
			}
			// Update counter students atomik
			if err := h.Stats.IncActiveStudents(tx, masjidID, delta); err != nil {
				return err
			}
		}

		// =========================
		// NEW: Promosi role → student
		// =========================
		// Hanya saat transisi non-aktif → aktif.
		if !wasActive && nowActive {
			// Ambil user target setelah ApplyToModel (bisa jadi berubah dari payload)
			userID := existing.UserClassesUserID

			// Update role hanya jika saat ini 'user'
			if err := tx.Model(&userModel.UserModel{}).
				Where("id = ? AND role = ?", userID, "user").
				Update("role", "student").Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah role user menjadi student")
			}
		}

		return helper.JsonUpdated(c, "Enrolment berhasil diperbarui", ucDTO.NewUserClassResponse(existing))
	})
}

// helper kecil
func isActive(status string, endedAt *time.Time) bool {
	return strings.EqualFold(status, classModel.UserClassStatusActive) && endedAt == nil
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
    masjidID, err := helper.GetMasjidIDFromToken(c) // <-- ini yang benar
    if err != nil {
        return err
    }

    var q ucDTO.ListUserClassQuery
    q.Limit = 20
    if err := c.QueryParser(&q); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
    }

    tx := h.DB.Model(&classModel.UserClassesModel{}).
        Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
        Where("classes.class_masjid_id = ? AND classes.class_deleted_at IS NULL", masjidID)

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

    if q.Limit > 0 {
        tx = tx.Limit(q.Limit)
    }
    if q.Offset > 0 {
        tx = tx.Offset(q.Offset)
    }

    var rows []classModel.UserClassesModel
    if err := tx.Find(&rows).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
    }

    resp := make([]*ucDTO.UserClassResponse, 0, len(rows))
    for i := range rows {
        resp = append(resp, ucDTO.NewUserClassResponse(&rows[i]))
    }
    return helper.JsonOK(c, "OK", resp)
}

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

    // Jalankan atomik
    return h.DB.Transaction(func(tx *gorm.DB) error {
        // Cek state sebelum update
        wasActive := isActive(m.UserClassesStatus, m.UserClassesEndedAt)

        now := time.Now()
        updates := map[string]any{
            "user_classes_status":     classModel.UserClassStatusEnded,
            "user_classes_ended_at":   now,
            "user_classes_updated_at": now,
        }

        if err := tx.Model(&classModel.UserClassesModel{}).
            Where("user_classes_id = ?", m.UserClassesID).
            Updates(updates).Error; err != nil {
            return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengakhiri enrolment")
        }

        // Jika sebelumnya aktif → sekarang tidak aktif ⇒ decrement
        if wasActive {
            if err := h.Stats.EnsureForMasjid(tx, masjidID); err != nil {
                return err
            }
            if err := h.Stats.IncActiveStudents(tx, masjidID, -1); err != nil {
                return err
            }
        }

        // Balas setelah sukses
        return helper.JsonUpdated(c, "Enrolment diakhiri", fiber.Map{
            "user_classes_id":       m.UserClassesID,
            "user_classes_status":   classModel.UserClassStatusEnded,
            "user_classes_ended_at": now,
        })
    })
}


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

	// Default rule: hanya boleh hapus kalau masih pending/inactive dan belum punya ended_at
	if !force {
		if !(m.UserClassesStatus == classModel.UserClassStatusInactive && m.UserClassesEndedAt == nil) {
			return fiber.NewError(
				fiber.StatusConflict,
				"Enrolment sudah aktif/berakhir. Gunakan endpoint end terlebih dahulu atau set status ke inactive.",
			)
		}
	}

	// Hard delete. FK turunan (user_class_sections, user_class_invoices) sebaiknya ON DELETE CASCADE.
	if err := h.DB.Where("user_classes_id = ?", m.UserClassesID).
		Delete(&classModel.UserClassesModel{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus enrolment")
	}

	return helper.JsonDeleted(c, "Enrolment dihapus", fiber.Map{
		"user_classes_id": m.UserClassesID,
	})
}
