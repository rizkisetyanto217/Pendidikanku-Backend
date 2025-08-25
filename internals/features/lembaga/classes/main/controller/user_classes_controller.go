package controller

import (
	"log"
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

	openingSvc "masjidku_backend/internals/features/lembaga/academics/academic_terms/services"
	statsSvc "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
)

type UserClassController struct {
    DB       *gorm.DB
    Stats    *statsSvc.LembagaStatsService
    QuotaSvc openingSvc.OpeningQuotaService // <— add
}

func NewUserClassController(db *gorm.DB) *UserClassController {
    return &UserClassController{
        DB:       db,
        Stats:    statsSvc.NewLembagaStatsService(),
        QuotaSvc: openingSvc.NewOpeningQuotaService(), // <— add
    }
}

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

	log.Printf("[UserClass] 🔥 UpdateUserClass START ucID=%s masjidID=%s", ucID, masjidID)

	// Ambil enrolment + tenant guard
	existing, err := h.findUserClassWithTenantGuard(ucID, masjidID)
	if err != nil {
		log.Printf("[UserClass] ❌ findUserClassWithTenantGuard gagal ucID=%s masjidID=%s err=%v", ucID, masjidID, err)
		return err
	}

	var req ucDTO.UpdateUserClassRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[UserClass] ❌ BodyParser gagal err=%v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := validateUserClasses.Struct(req); err != nil {
		log.Printf("[UserClass] ❌ Validasi gagal err=%v", err)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ==== Mulai transaksi agar semua atomik ====
	return h.DB.Transaction(func(tx *gorm.DB) error {
		log.Printf("[UserClass] ➡️ Mulai Transaction ucID=%s", ucID)

		targetUser := existing.UserClassesUserID
		if req.UserClassesUserID != nil {
			targetUser = *req.UserClassesUserID
		}

		targetClass := existing.UserClassesClassID
		if req.UserClassesClassID != nil {
			if err := h.ensureClassBelongsToMasjid(tx, *req.UserClassesClassID, masjidID); err != nil {
				log.Printf("[UserClass] ❌ ensureClassBelongsToMasjid gagal classID=%s masjidID=%s err=%v", *req.UserClassesClassID, masjidID, err)
				return err
			}
			targetClass = *req.UserClassesClassID
		}

		if req.UserClassesMasjidID != nil && *req.UserClassesMasjidID != masjidID {
			log.Printf("[UserClass] ❌ UserClassesMasjidID tidak boleh diubah masjidID=%s", *req.UserClassesMasjidID)
			return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak boleh diubah")
		}

		targetTerm := existing.UserClassesTermID
		if req.UserClassesTermID != nil {
			if err := h.ensureTermBelongsToMasjid(tx, *req.UserClassesTermID, masjidID); err != nil {
				log.Printf("[UserClass] ❌ ensureTermBelongsToMasjid gagal termID=%s masjidID=%s err=%v", *req.UserClassesTermID, masjidID, err)
				return err
			}
			targetTerm = *req.UserClassesTermID
		}

		targetStatus := existing.UserClassesStatus
		if req.UserClassesStatus != nil {
			targetStatus = *req.UserClassesStatus
		}

		// Cegah duplikasi enrolment aktif
		if strings.EqualFold(targetStatus, classModel.UserClassStatusActive) {
			if err := h.checkActiveEnrollmentConflict(tx, targetUser, targetClass, targetTerm, existing.UserClassesID, masjidID); err != nil {
				log.Printf("[UserClass] ❌ Conflict enrolment aktif user=%s class=%s term=%s err=%v", targetUser, targetClass, targetTerm, err)
				return err
			}
		}

		wasActive := strings.EqualFold(existing.UserClassesStatus, classModel.UserClassStatusActive)
		willBeActive := strings.EqualFold(targetStatus, classModel.UserClassStatusActive)
		log.Printf("[UserClass] Status flip? wasActive=%t willBeActive=%t", wasActive, willBeActive)

		// Tentukan opening yang akan dipakai / berubah
		var openingToUse *uuid.UUID
		if req.UserClassesOpeningID != nil {
			openingToUse = req.UserClassesOpeningID
		} else if existing.UserClassesOpeningID != nil {
			openingToUse = existing.UserClassesOpeningID
		}
		log.Printf("[UserClass] openingToUse=%v", openingToUse)

		// Simpan nilai final yang harus dipersist (mencegah ketimpa ApplyToModel)
		finalOpeningID := existing.UserClassesOpeningID

		// (1) inactive -> active  => CLAIM kuota (jika ada opening)
		if !wasActive && willBeActive {
			if openingToUse != nil {
				log.Printf("[UserClass] 👉 Claim quota (inactive→active) openingID=%s", *openingToUse)
				if err := h.QuotaSvc.EnsureOpeningBelongsToMasjid(tx, *openingToUse, masjidID); err != nil {
					return err
				}
				if err := h.QuotaSvc.Claim(tx, *openingToUse); err != nil {
					return err
				}
				finalOpeningID = openingToUse
			}
		}

		// (2) active -> inactive  => RELEASE kuota (pakai opening yang lama/tercatat)
		if wasActive && !willBeActive {
			if existing.UserClassesOpeningID != nil {
				log.Printf("[UserClass] 👉 Release quota (active→inactive) openingID=%s", *existing.UserClassesOpeningID)
				if err := h.QuotaSvc.Release(tx, *existing.UserClassesOpeningID); err != nil {
					return err
				}
			}
			// saat non-aktif, boleh biarkan opening tetap tercatat atau null-kan sesuai kebijakan
		}

		// (3) active -> active, tapi opening berubah / baru ditambahkan
		if wasActive && willBeActive {
			// Tambah opening pertama kali (sebelumnya nil, sekarang ada di req)
			if existing.UserClassesOpeningID == nil && req.UserClassesOpeningID != nil {
				oid := *req.UserClassesOpeningID
				log.Printf("[UserClass] 👉 Active→Active ADD opening, Claim openingID=%s", oid)
				if err := h.QuotaSvc.EnsureOpeningBelongsToMasjid(tx, oid, masjidID); err != nil {
					return err
				}
				if err := h.QuotaSvc.Claim(tx, oid); err != nil {
					return err
				}
				finalOpeningID = &oid
			}

			// Ganti opening A → B
			if existing.UserClassesOpeningID != nil && req.UserClassesOpeningID != nil &&
				existing.UserClassesOpeningID.String() != req.UserClassesOpeningID.String() {

				oldID := *existing.UserClassesOpeningID
				newID := *req.UserClassesOpeningID
				log.Printf("[UserClass] 👉 Active→Active SWITCH opening old=%s new=%s", oldID, newID)

				// Release kuota lama
				if err := h.QuotaSvc.Release(tx, oldID); err != nil {
					return err
				}
				// Claim kuota baru
				if err := h.QuotaSvc.EnsureOpeningBelongsToMasjid(tx, newID, masjidID); err != nil {
					return err
				}
				if err := h.QuotaSvc.Claim(tx, newID); err != nil {
					return err
				}
				finalOpeningID = &newID
			}
		}

		// Terapkan perubahan field lain dari request
		req.ApplyToModel(existing)

		// Pastikan opening ID yang sudah diputuskan tidak ketimpa oleh ApplyToModel
		if finalOpeningID != nil {
			existing.UserClassesOpeningID = finalOpeningID
		}

		if err := tx.Model(&classModel.UserClassesModel{}).
			Where("user_classes_id = ?", existing.UserClassesID).
			Updates(existing).Error; err != nil {
			log.Printf("[UserClass] ❌ Gagal update enrolment ucID=%s err=%v", existing.UserClassesID, err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui enrolment")
		}

		// Statistik & role promotion
		nowActive := strings.EqualFold(existing.UserClassesStatus, classModel.UserClassStatusActive)
		delta := 0
		if !wasActive && nowActive {
			delta = +1
		} else if wasActive && !nowActive {
			delta = -1
		}

		if delta != 0 {
			log.Printf("[UserClass] Update statistik delta=%d masjidID=%s", delta, masjidID)
			if err := h.Stats.EnsureForMasjid(tx, masjidID); err != nil {
				return err
			}
			if err := h.Stats.IncActiveStudents(tx, masjidID, delta); err != nil {
				return err
			}
		}

		if !wasActive && nowActive {
			log.Printf("[UserClass] Promote user=%s to role=student", existing.UserClassesUserID)
			userID := existing.UserClassesUserID
			if err := tx.Model(&userModel.UserModel{}).
				Where("id = ? AND role = ?", userID, "user").
				Update("role", "student").Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah role user menjadi student")
			}
		}

		log.Printf("[UserClass] ✅ UpdateUserClass DONE ucID=%s openingID=%v", existing.UserClassesID, existing.UserClassesOpeningID)
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
	default:
		tx = tx.Order("user_classes_created_at ASC")
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
