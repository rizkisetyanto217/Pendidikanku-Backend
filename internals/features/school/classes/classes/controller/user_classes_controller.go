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

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	memsvc "masjidku_backend/internals/features/school/classes/classes/service"
)

/* ================== Controller ================== */

type UserClassController struct {
	DB            *gorm.DB
	MembershipSvc memsvc.Service
}

func NewUserClassController(db *gorm.DB) *UserClassController {
	return &UserClassController{
		DB:            db,
		MembershipSvc: memsvc.New(),
	}
}

var validateUserClasses = validator.New()

/* ================= Helpers ================= */

// pastikan class milik tenant
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

// ambil user_id dari masjid_students
func (h *UserClassController) getUserIDFromMasjidStudent(tx *gorm.DB, masjidStudentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	if err := tx.Raw(`
		SELECT masjid_student_user_id
		FROM masjid_students
		WHERE masjid_student_id = ? AND masjid_student_deleted_at IS NULL
	`, masjidStudentID).Scan(&userID).Error; err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil user dari masjid_students")
	}
	if userID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_student tidak valid atau sudah dihapus")
	}
	return userID, nil
}

// cek konflik enrolment aktif per (masjid_student, class, masjid) selain baris yang diupdate
func (h *UserClassController) checkActiveEnrollmentConflict(
	tx *gorm.DB,
	masjidStudentID, classID, excludeID, masjidID uuid.UUID,
) error {
	var exists bool
	sql := `
		SELECT EXISTS (
			SELECT 1 FROM user_classes
			WHERE user_classes_deleted_at IS NULL
			  AND user_classes_status = 'active'
			  AND user_classes_masjid_student_id = ?
			  AND user_classes_class_id = ?
			  AND user_classes_masjid_id = ?
			  AND user_classes_id <> ?
			LIMIT 1
		)
	`
	if err := tx.Raw(sql, masjidStudentID, classID, masjidID, excludeID).Scan(&exists).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa duplikasi enrolment aktif")
	}
	if exists {
		return fiber.NewError(fiber.StatusConflict, "Santri sudah memiliki enrolment aktif pada kelas ini")
	}
	return nil
}

func (h *UserClassController) findUserClassWithTenantGuard(userClassID, masjidID uuid.UUID) (*ucModel.UserClassesModel, error) {
	var m ucModel.UserClassesModel
	if err := h.DB.Model(&ucModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where(`
			user_classes_id = ?
			AND user_classes.user_classes_masjid_id = ?
			AND classes.class_masjid_id = ?
			AND classes.class_deleted_at IS NULL
			AND classes.class_delete_pending_until IS NULL
			AND user_classes_deleted_at IS NULL
		`, userClassID, masjidID, masjidID).
		First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Enrolment tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil enrolment")
	}
	return &m, nil
}

/* ================== UPDATE (PATCH-like) ================== */

// PUT/PATCH /admin/user-classes/:id
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

		// calon target nilai setelah update
		targetMasjidStudent := existing.UserClassesMasjidStudentID
		if req.UserClassesMasjidStudentID != nil {
			targetMasjidStudent = *req.UserClassesMasjidStudentID
		}

		targetClass := existing.UserClassesClassID
		if req.UserClassesClassID != nil {
			if err := h.ensureClassBelongsToMasjid(tx, *req.UserClassesClassID, masjidID); err != nil {
				return err
			}
			targetClass = *req.UserClassesClassID
		}

		// Masjid ID tidak boleh lintas tenant
		if req.UserClassesMasjidID != nil && *req.UserClassesMasjidID != masjidID {
			return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak boleh diubah")
		}

		targetStatus := existing.UserClassesStatus
		if req.UserClassesStatus != nil {
			targetStatus = *req.UserClassesStatus
		}

		// Cegah duplikasi enrolment aktif
		if strings.EqualFold(targetStatus, ucModel.UserClassStatusActive) {
			if err := h.checkActiveEnrollmentConflict(tx, targetMasjidStudent, targetClass, existing.UserClassesID, masjidID); err != nil {
				return err
			}
		}

		// Simpan status awal sebelum ApplyToModel (untuk deteksi transisi)
		origStatus := existing.UserClassesStatus

		// Terapkan perubahan field dari request (pointer-aware)
		req.ApplyToModel(existing)

		// Persist
		if err := tx.Model(&ucModel.UserClassesModel{}).
			Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", existing.UserClassesID).
			Updates(existing).Error; err != nil {
			log.Printf("[UserClass] ❌ Gagal update enrolment ucID=%s err=%v", existing.UserClassesID, err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui enrolment")
		}

		// Hook membership: transisi status
		origActive := strings.EqualFold(origStatus, ucModel.UserClassStatusActive)
		nowActive := strings.EqualFold(existing.UserClassesStatus, ucModel.UserClassStatusActive)

		if !origActive && nowActive {
			// enrolment baru menjadi aktif → grant role student + ensure masjid_students aktif
			assignedBy, err := helperAuth.GetUserIDFromToken(c)
			if err != nil {
				return err
			}
			// ambil user_id dari masjid_students yang (mungkin) baru
			userID, err := h.getUserIDFromMasjidStudent(tx, existing.UserClassesMasjidStudentID)
			if err != nil {
				return err
			}
			if err := h.MembershipSvc.OnEnrollmentActivated(tx, userID, masjidID, assignedBy); err != nil {
				return err
			}
		} else if origActive && !nowActive {
			// turun dari aktif → set masjid_students jadi inactive (tanpa revoke role, kebijakan minimal)
			userID, err := h.getUserIDFromMasjidStudent(tx, existing.UserClassesMasjidStudentID)
			if err != nil {
				return err
			}
			_ = userID // reserve jika nanti ada kebijakan revoke role
			if err := h.MembershipSvc.OnEnrollmentDeactivated(tx, userID, masjidID); err != nil {
				return err
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
	if q.ClassID != nil {
		tx = tx.Where("user_classes_class_id = ?", *q.ClassID)
	}
	if q.MasjidID != nil {
		tx = tx.Where("user_classes_masjid_id = ?", *q.MasjidID)
	}
	if q.MasjidStudentID != nil {
		tx = tx.Where("user_classes_masjid_student_id = ?", *q.MasjidStudentID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_classes_status = ?", strings.TrimSpace(*q.Status))
	}
	if q.Result != nil && strings.TrimSpace(*q.Result) != "" {
		tx = tx.Where("user_classes_result = ?", strings.TrimSpace(*q.Result))
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
	case "completed_at_desc":
		tx = tx.Order("user_classes_completed_at DESC NULLS LAST").Order("user_classes_created_at DESC")
	case "completed_at_asc":
		tx = tx.Order("user_classes_completed_at ASC NULLS LAST").Order("user_classes_created_at ASC")
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

/* ================== COMPLETE (status=completed) ================== */

// Body opsional untuk set result & completed_at
type completeBody struct {
	Result      *string    `json:"result" validate:"omitempty,oneof=passed failed"`
	CompletedAt *time.Time `json:"completed_at" validate:"omitempty"`
}

// POST /admin/user-classes/:id/complete
func (h *UserClassController) CompleteUserClass(c *fiber.Ctx) error {
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

	var body completeBody
	_ = c.BodyParser(&body)
	if err := validateUserClasses.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Idempotent + merge data
	if strings.EqualFold(m.UserClassesStatus, ucModel.UserClassStatusCompleted) {
		// jika sudah completed tapi ingin update result/ts, izinkan via update cepat
		now := time.Now()
		updates := map[string]any{
			"user_classes_updated_at": now,
		}
		if body.Result != nil {
			updates["user_classes_result"] = *body.Result
		}
		if body.CompletedAt != nil {
			updates["user_classes_completed_at"] = *body.CompletedAt
		}

		if len(updates) > 1 { // ada sesuatu selain updated_at
			if err := h.DB.Model(&ucModel.UserClassesModel{}).
				Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", m.UserClassesID).
				Updates(updates).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data kelulusan")
			}
			// refresh lokal
			if body.Result != nil {
				m.UserClassesResult = body.Result
			}
			if body.CompletedAt != nil {
				m.UserClassesCompletedAt = body.CompletedAt
			}
			m.UserClassesUpdatedAt = now
		}

		return helper.JsonUpdated(c, "Enrolment sudah berstatus completed", ucDTO.NewUserClassResponse(m))
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		origStatus := m.UserClassesStatus

		now := time.Now()
		completedAt := now
		if body.CompletedAt != nil {
			completedAt = *body.CompletedAt
		}
		updates := map[string]any{
			"user_classes_status":         ucModel.UserClassStatusCompleted,
			"user_classes_completed_at":   completedAt,
			"user_classes_updated_at":     now,
		}
		if body.Result != nil {
			updates["user_classes_result"] = *body.Result
		} else {
			// result boleh kosong saat completed (belum diputuskan lulus/gagal)
			updates["user_classes_result"] = gorm.Expr("NULL")
		}

		if err := tx.Model(&ucModel.UserClassesModel{}).
			Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", m.UserClassesID).
			Updates(updates).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyelesaikan enrolment")
		}

		// refresh struct
		m.UserClassesStatus = ucModel.UserClassStatusCompleted
		m.UserClassesCompletedAt = &completedAt
		m.UserClassesResult = body.Result
		m.UserClassesUpdatedAt = now

		// Hook membership jika turun dari active → non-active
		if strings.EqualFold(origStatus, ucModel.UserClassStatusActive) {
			userID, err := h.getUserIDFromMasjidStudent(tx, m.UserClassesMasjidStudentID)
			if err != nil {
				return err
			}
			if err := h.MembershipSvc.OnEnrollmentDeactivated(tx, userID, masjidID); err != nil {
				return err
			}
		}

		return helper.JsonUpdated(c, "Enrolment diset selesai (completed)", ucDTO.NewUserClassResponse(m))
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
			"Enrolment masih aktif. Nonaktifkan/selesaikan terlebih dahulu atau gunakan ?force=true.",
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
