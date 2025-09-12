// internals/features/lembaga/class_subjects/controller/class_subject_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	csDTO "masjidku_backend/internals/features/school/subject_books/subject/dto"
	csModel "masjidku_backend/internals/features/school/subject_books/subject/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type ClassSubjectController struct {
	DB *gorm.DB
}


/* =========================================================
   CREATE
   POST /admin/class-subjects
   ========================================================= */
func (h *ClassSubjectController) Create(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var req csDTO.CreateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant
	req.MasjidID = masjidID

	// Normalisasi ringan
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// === Cek duplikasi kombinasi (soft delete aware) ===
		// Unik pada: masjid_id, class_id, subject_id, (term_id nullable)
		termStr := ""
		if req.TermID != nil {
			termStr = req.TermID.String()
		}

		var cnt int64
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where(`
				class_subjects_masjid_id = ?
				AND class_subjects_class_id = ?
				AND class_subjects_subject_id = ?
				AND COALESCE(class_subjects_term_id::text, '') = COALESCE(?, '')
				AND class_subjects_deleted_at IS NULL
			`, req.MasjidID, req.ClassID, req.SubjectID, termStr).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+term/semester sudah terdaftar")
		}

		// Create
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+term/semester sudah terdaftar")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat class subject")
		}
		c.Locals("created_class_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("created_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonCreated(c, "Class subject berhasil dibuat", csDTO.FromClassSubjectModel(m))
}


/* =========================================================
   UPDATE (partial)
   PUT /admin/class-subjects/:id
   ========================================================= */
func (h *ClassSubjectController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	var req csDTO.UpdateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Force tenant
	req.MasjidID = &masjidID

	// Normalisasi ringan
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var m csModel.ClassSubjectModel
		if err := tx.First(&m, "class_subjects_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah data milik masjid lain")
		}
		if m.ClassSubjectsDeletedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// ==== Cek duplikat jika kombinasi berubah ====
		shouldCheckDup := false
		newClassID := m.ClassSubjectsClassID
		newSubjectID := m.ClassSubjectsSubjectID
		var newTermID *uuid.UUID = m.ClassSubjectsTermID

		if req.ClassID != nil && *req.ClassID != m.ClassSubjectsClassID {
			shouldCheckDup = true
			newClassID = *req.ClassID
		}
		if req.SubjectID != nil && *req.SubjectID != m.ClassSubjectsSubjectID {
			shouldCheckDup = true
			newSubjectID = *req.SubjectID
		}
		if req.TermID != nil {
			// beda nilai?
			curr := ""
			if m.ClassSubjectsTermID != nil { curr = m.ClassSubjectsTermID.String() }
			if req.TermID.String() != curr {
				shouldCheckDup = true
			}
			if req.TermID == nil {
				newTermID = nil
			} else {
				t := *req.TermID
				newTermID = &t
			}
		}

		if shouldCheckDup {
			termStr := ""
			if newTermID != nil { termStr = newTermID.String() }

			var cnt int64
			if err := tx.Model(&csModel.ClassSubjectModel{}).
				Where(`
					class_subjects_masjid_id = ?
					AND class_subjects_class_id  = ?
					AND class_subjects_subject_id= ?
					AND COALESCE(class_subjects_term_id::text,'') = COALESCE(?, '')
					AND class_subjects_id <> ?
					AND class_subjects_deleted_at IS NULL
				`, masjidID, newClassID, newSubjectID, termStr, m.ClassSubjectsID).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+term/semester sudah terdaftar")
			}
		}

		// Apply ke model lalu update
		req.Apply(&m)

		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where("class_subjects_id = ?", m.ClassSubjectsID).
			Updates(map[string]interface{}{
				"class_subjects_masjid_id":         m.ClassSubjectsMasjidID,
				"class_subjects_class_id":          m.ClassSubjectsClassID,
				"class_subjects_subject_id":        m.ClassSubjectsSubjectID,
				"class_subjects_term_id":           m.ClassSubjectsTermID,
				"class_subjects_order_index":       m.ClassSubjectsOrderIndex,
				"class_subjects_hours_per_week":    m.ClassSubjectsHoursPerWeek,
				"class_subjects_min_passing_score": m.ClassSubjectsMinPassingScore,
				"class_subjects_weight_on_report":  m.ClassSubjectsWeightOnReport,
				"class_subjects_is_core":           m.ClassSubjectsIsCore,
				"class_subjects_desc":              m.ClassSubjectsDesc,
				// updated_at akan diisi trigger
				"class_subjects_is_active": m.ClassSubjectsIsActive,
			}).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+term/semester sudah terdaftar")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}

		c.Locals("updated_class_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("updated_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonUpdated(c, "Class subject berhasil diperbarui", csDTO.FromClassSubjectModel(m))
}

/* =========================================================
   DELETE
   DELETE /admin/class-subjects/:id?force=true
   ========================================================= */
func (h *ClassSubjectController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	// Hanya admin yang boleh hard delete
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var m csModel.ClassSubjectModel
		if err := tx.First(&m, "class_subjects_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik masjid lain")
		}

		if force {
			// hard delete benar-benar hapus row
			if err := tx.Unscoped().Delete(&csModel.ClassSubjectModel{}, "class_subjects_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			if m.ClassSubjectsDeletedAt.Valid {
				return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
			}
			// soft delete → GORM akan UPDATE deleted_at; trigger akan set updated_at
			if err := tx.Delete(&csModel.ClassSubjectModel{}, "class_subjects_id = ?", id).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		}

		c.Locals("deleted_class_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("deleted_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonDeleted(c, "Class subject berhasil dihapus", csDTO.FromClassSubjectModel(m))
}