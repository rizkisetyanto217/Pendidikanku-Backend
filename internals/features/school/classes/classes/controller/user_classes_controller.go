// file: internals/features/lembaga/classes/user_classes/main/controller/user_class_controller.go
package controller

import (
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	ucDTO "masjidku_backend/internals/features/school/classes/classes/dto"
	ucModel "masjidku_backend/internals/features/school/classes/classes/model"

	userModel "masjidku_backend/internals/features/users/user/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* ================== Controller ================== */

type UserClassController struct {
	DB *gorm.DB
}

func NewUserClassController(db *gorm.DB) *UserClassController {
	return &UserClassController{DB: db}
}

var validateUserClasses = validator.New()

/* ================= Helpers ================= */

func (h *UserClassController) ensureClassBelongsToMasjid(tx *gorm.DB, classID, masjidID uuid.UUID) error {
	var count int64
	if err := tx.Table("classes").
		Where("class_id = ? AND class_masjid_id = ? AND class_deleted_at IS NULL AND class_delete_pending_until IS NULL", classID, masjidID).
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

func (h *UserClassController) findUserClassWithTenantGuard(userClassID, masjidID uuid.UUID) (*ucModel.UserClassesModel, error) {
	var m ucModel.UserClassesModel
	if err := h.DB.Model(&ucModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where(`
			user_classes_id = ?
			AND classes.class_masjid_id = ?
			AND classes.class_deleted_at IS NULL
			AND classes.class_delete_pending_until IS NULL
			AND user_classes_deleted_at IS NULL
		`, userClassID, masjidID).
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

/* ================== UPDATE (PATCH-like) ================== */

// PATCH /admin/user-classes/:id
func (h *UserClassController) UpdateUserClass(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	ucID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	log.Printf("[UserClass] 🔥 UpdateUserClass START ucID=%s masjidID=%s", ucID, masjidID)

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

	return h.DB.Transaction(func(tx *gorm.DB) error {
		log.Printf("[UserClass] ➡️ Mulai Transaction ucID=%s", ucID)

		targetUser := existing.UserClassesUserID
		if req.UserClassesUserID != nil {
			targetUser = *req.UserClassesUserID
		}

		targetClass := existing.UserClassesClassID
		if req.UserClassesClassID != nil {
			if err := h.ensureClassBelongsToMasjid(tx, *req.UserClassesClassID, masjidID); err != nil {
				return err
			}
			targetClass = *req.UserClassesClassID
		}

		// Masjid ID tidak boleh diubah lintas tenant
		if req.UserClassesMasjidID != nil && *req.UserClassesMasjidID != masjidID {
			return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak boleh diubah")
		}

		targetTerm := existing.UserClassesTermID
		if req.UserClassesTermID != nil {
			if err := h.ensureTermBelongsToMasjid(tx, *req.UserClassesTermID, masjidID); err != nil {
				return err
			}
			targetTerm = *req.UserClassesTermID
		}

		targetStatus := existing.UserClassesStatus
		if req.UserClassesStatus != nil {
			targetStatus = *req.UserClassesStatus
		}

		// Cegah duplikasi enrolment aktif
		if strings.EqualFold(targetStatus, ucModel.UserClassStatusActive) {
			if err := h.checkActiveEnrollmentConflict(tx, targetUser, targetClass, targetTerm, existing.UserClassesID, masjidID); err != nil {
				return err
			}
		}

		// Terapkan perubahan field dari request (pointer-aware)
		req.ApplyToModel(existing)

		// Persist
		if err := tx.Model(&ucModel.UserClassesModel{}).
			Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", existing.UserClassesID).
			Updates(existing).Error; err != nil {
			log.Printf("[UserClass] ❌ Gagal update enrolment ucID=%s err=%v", existing.UserClassesID, err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui enrolment")
		}

		// (Opsional) Promote user ke role "student" ketika berubah menjadi aktif
		wasActive := strings.EqualFold(existing.UserClassesStatus, ucModel.UserClassStatusActive) // setelah ApplyToModel
		if wasActive {
			if err := tx.Model(&userModel.UserModel{}).
				Where("id = ? AND role = ?", existing.UserClassesUserID, "user").
				Update("role", "student").Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah role user menjadi student")
			}
		}

		log.Printf("[UserClass] ✅ UpdateUserClass DONE ucID=%s", existing.UserClassesID)
		return helper.JsonUpdated(c, "Enrolment berhasil diperbarui", ucDTO.NewUserClassResponse(existing))
	})
}

/* ================== GET BY ID ================== */

// GET /admin/user-classes/:id
func (h *UserClassController) GetUserClassByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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

/* ================== LIST ================== */

// GET /admin/user-classes
func (h *UserClassController) ListUserClasses(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q ucDTO.ListUserClassQuery
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 200 {
		q.Limit = 200
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	tx := h.DB.Model(&ucModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where(`
			classes.class_masjid_id = ?
			AND classes.class_deleted_at IS NULL
			AND classes.class_delete_pending_until IS NULL
			AND user_classes_deleted_at IS NULL
		`, masjidID)

	// filters
	if q.UserID != nil {
		tx = tx.Where("user_classes_user_id = ?", *q.UserID)
	}
	if q.ClassID != nil {
		tx = tx.Where("user_classes_class_id = ?", *q.ClassID)
	}
	if q.TermID != nil {
		tx = tx.Where("user_classes_term_id = ?", *q.TermID)
	}
	if q.MasjidStudentID != nil {
		tx = tx.Where("user_classes_masjid_student_id = ?", *q.MasjidStudentID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_classes_status = ?", strings.TrimSpace(*q.Status))
	}
	if q.ActiveNow != nil && *q.ActiveNow {
		tx = tx.Where("user_classes_status = 'active' AND user_classes_left_at IS NULL")
	}
	if q.JoinedFrom != nil {
		tx = tx.Where("user_classes_joined_at IS NOT NULL AND user_classes_joined_at >= ?", *q.JoinedFrom)
	}
	if q.JoinedTo != nil {
		tx = tx.Where("user_classes_joined_at IS NOT NULL AND user_classes_joined_at <= ?", *q.JoinedTo)
	}

	// total
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// sorting
	sort := "created_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "created_at_asc":
		tx = tx.Order("user_classes_created_at ASC")
	case "joined_at_desc":
		tx = tx.Order("user_classes_joined_at DESC NULLS LAST").Order("user_classes_created_at DESC")
	case "joined_at_asc":
		tx = tx.Order("user_classes_joined_at ASC NULLS LAST").Order("user_classes_created_at ASC")
	case "created_at_desc":
		fallthrough
	default:
		tx = tx.Order("user_classes_created_at DESC")
	}

	// fetch data
	var rows []ucModel.UserClassesModel
	if err := tx.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]*ucDTO.UserClassResponse, 0, len(rows))
	for i := range rows {
		items = append(items, ucDTO.NewUserClassResponse(&rows[i]))
	}

	return helper.JsonList(c, items, fiber.Map{
		"limit":  q.Limit,
		"offset": q.Offset,
		"total":  int(total),
	})
}

/* ================== END (status=ended) ================== */

// POST /admin/user-classes/:id/end
func (h *UserClassController) EndUserClass(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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

	// Idempotent
	if strings.EqualFold(m.UserClassesStatus, ucModel.UserClassStatusEnded) {
		return helper.JsonUpdated(c, "Enrolment sudah berstatus ended", fiber.Map{
			"user_classes_id":     m.UserClassesID,
			"user_classes_status": ucModel.UserClassStatusEnded,
		})
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		updates := map[string]any{
			"user_classes_status":     ucModel.UserClassStatusEnded,
			"user_classes_left_at":    now, // set jejak keluar
			"user_classes_updated_at": now,
		}

		if err := tx.Model(&ucModel.UserClassesModel{}).
			Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", m.UserClassesID).
			Updates(updates).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengakhiri enrolment")
		}

		return helper.JsonUpdated(c, "Enrolment diakhiri", fiber.Map{
			"user_classes_id":     m.UserClassesID,
			"user_classes_status": ucModel.UserClassStatusEnded,
			"updated_at":          now,
		})
	})
}

/* ================== DELETE ================== */

// DELETE /admin/user-classes/:id
func (h *UserClassController) DeleteUserClass(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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

	force := strings.EqualFold(c.Query("force"), "true")

	// Default rule: cegah hapus enrolment aktif (kecuali force)
	if !force && strings.EqualFold(m.UserClassesStatus, ucModel.UserClassStatusActive) {
		return fiber.NewError(
			fiber.StatusConflict,
			"Enrolment masih aktif. Nonaktifkan/akhiri terlebih dahulu atau gunakan ?force=true.",
		)
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		var delErr error
		if force {
			delErr = tx.Unscoped().
				Where("user_classes_id = ?", m.UserClassesID).
				Delete(&ucModel.UserClassesModel{}).Error
		} else {
			delErr = tx.
				Where("user_classes_id = ?", m.UserClassesID).
				Delete(&ucModel.UserClassesModel{}).Error
		}
		if delErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus enrolment")
		}

		return helper.JsonDeleted(c, "Enrolment dihapus", fiber.Map{
			"user_classes_id": m.UserClassesID,
			"force":           force,
		})
	})
}
