// internals/features/lembaga/class_subjects/controller/class_subject_controller.go
package controller

import (
	"errors"
	"fmt"
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

/*
=========================================================

	CREATE
	POST /admin/:masjid_id/class-subjects
	(atau /admin/:masjid_slug/class-subjects)

=========================================================
*/
func (h *ClassSubjectController) Create(c *fiber.Ctx) error {
	// 🔐 Ambil konteks masjid & pastikan DKM/Admin
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req csDTO.CreateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.MasjidID = masjidID // force tenant

	// Normalisasi ringan
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}
	if req.Slug != nil {
		s := strings.TrimSpace(*req.Slug)
		if s == "" {
			req.Slug = nil
		} else {
			s = helper.Slugify(s, 160)
			req.Slug = &s
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// === Cek duplikasi kombinasi ===
		var cnt int64
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where(`
                class_subjects_masjid_id = ?
                AND class_subjects_class_id = ?
                AND class_subjects_subject_id = ?
                AND class_subjects_deleted_at IS NULL
            `, req.MasjidID, req.ClassID, req.SubjectID).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject sudah terdaftar")
		}

		// === Generate slug unik ===
		baseSlug := ""
		if req.Slug != nil {
			baseSlug = *req.Slug
		} else {
			var subjName, classSlug string
			_ = tx.Table("subjects").
				Select("subjects_name").
				Where("subjects_id = ? AND subjects_masjid_id = ?", req.SubjectID, req.MasjidID).
				Scan(&subjName).Error

			_ = tx.Table("classes").
				Select("class_slug").
				Where("class_id = ? AND class_masjid_id = ?", req.ClassID, req.MasjidID).
				Scan(&classSlug).Error

			switch {
			case strings.TrimSpace(subjName) != "" && strings.TrimSpace(classSlug) != "":
				baseSlug = helper.Slugify(classSlug+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(classSlug) != "":
				baseSlug = helper.Slugify(classSlug, 160)
			default:
				baseSlug = helper.Slugify(
					fmt.Sprintf("cs-%s-%s",
						strings.Split(req.ClassID.String(), "-")[0],
						strings.Split(req.SubjectID.String(), "-")[0],
					), 160)
			}
		}

		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			"class_subjects",
			"class_subjects_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("class_subjects_masjid_id = ? AND class_subjects_deleted_at IS NULL", req.MasjidID)
			},
			160,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}

		// Create
		m := req.ToModel()
		m.ClassSubjectsSlug = &uniqueSlug

		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject sudah terdaftar")
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

/*
=========================================================

	UPDATE (partial)
	PUT /admin/:masjid_id/class-subjects/:id

=========================================================
*/
func (h *ClassSubjectController) Update(c *fiber.Ctx) error {
	// 🔐 Context & role
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Param ID
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Parse payload
	var req csDTO.UpdateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.MasjidID = &masjidID

	// Normalisasi ringan
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}
	if req.Slug != nil {
		s := strings.TrimSpace(*req.Slug)
		if s == "" {
			req.Slug = nil
		} else {
			s = helper.Slugify(s, 160)
			req.Slug = &s
		}
	}

	// Validasi DTO
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Transaksi
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Ambil record lama
		var m csModel.ClassSubjectModel
		if err := tx.Where("class_subjects_id = ?", id).First(&m).Error; err != nil {
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

		// Cek apakah class/subject berubah → cek duplikasi
		newClassID := m.ClassSubjectsClassID
		newSubjectID := m.ClassSubjectsSubjectID
		if req.ClassID != nil {
			newClassID = *req.ClassID
		}
		if req.SubjectID != nil {
			newSubjectID = *req.SubjectID
		}

		if newClassID != m.ClassSubjectsClassID || newSubjectID != m.ClassSubjectsSubjectID {
			var cnt int64
			if err := tx.Model(&csModel.ClassSubjectModel{}).
				Where(`
					class_subjects_masjid_id = ?
					AND class_subjects_class_id  = ?
					AND class_subjects_subject_id= ?
					AND class_subjects_id <> ?
					AND class_subjects_deleted_at IS NULL
				`, masjidID, newClassID, newSubjectID, m.ClassSubjectsID).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject sudah terdaftar")
			}
		}

		// Flag untuk slug
		oldSlugNil := (m.ClassSubjectsSlug == nil || strings.TrimSpace(ptrStr(m.ClassSubjectsSlug)) == "")
		classChanged := (req.ClassID != nil && *req.ClassID != m.ClassSubjectsClassID)
		subjectChanged := (req.SubjectID != nil && *req.SubjectID != m.ClassSubjectsSubjectID)

		// Apply perubahan dari req
		req.Apply(&m)

		// Slug handling
		needSetSlug := false
		var baseSlug string

		if req.Slug != nil {
			// User kasih slug manual
			baseSlug = *req.Slug
			needSetSlug = true
		} else if oldSlugNil || classChanged || subjectChanged {
			// Slug kosong ATAU class/subject berubah → regen
			needSetSlug = true

			var subjName, classSlug string
			_ = tx.Table("subjects").
				Select("subjects_name").
				Where("subjects_id = ? AND subjects_masjid_id = ?", m.ClassSubjectsSubjectID, masjidID).
				Scan(&subjName).Error

			_ = tx.Table("classes").
				Select("class_slug").
				Where("class_id = ? AND class_masjid_id = ?", m.ClassSubjectsClassID, masjidID).
				Scan(&classSlug).Error

			switch {
			case strings.TrimSpace(classSlug) != "" && strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(classSlug+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(classSlug) != "":
				baseSlug = helper.Slugify(classSlug, 160)
			default:
				baseSlug = helper.Slugify(
					fmt.Sprintf("cs-%s-%s",
						strings.Split(m.ClassSubjectsClassID.String(), "-")[0],
						strings.Split(m.ClassSubjectsSubjectID.String(), "-")[0],
					), 160)
			}
		}

		if needSetSlug {
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				tx,
				"class_subjects",
				"class_subjects_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(`
						class_subjects_masjid_id = ?
						AND class_subjects_deleted_at IS NULL
						AND class_subjects_id <> ?
					`, masjidID, m.ClassSubjectsID)
				},
				160,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			m.ClassSubjectsSlug = &uniqueSlug
		}

		// Persist ke DB
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where("class_subjects_id = ?", m.ClassSubjectsID).
			Updates(map[string]any{
				"class_subjects_masjid_id":              m.ClassSubjectsMasjidID,
				"class_subjects_class_id":               m.ClassSubjectsClassID,
				"class_subjects_subject_id":             m.ClassSubjectsSubjectID,
				"class_subjects_slug":                   m.ClassSubjectsSlug,
				"class_subjects_order_index":            m.ClassSubjectsOrderIndex,
				"class_subjects_hours_per_week":         m.ClassSubjectsHoursPerWeek,
				"class_subjects_min_passing_score":      m.ClassSubjectsMinPassingScore,
				"class_subjects_weight_on_report":       m.ClassSubjectsWeightOnReport,
				"class_subjects_is_core":                m.ClassSubjectsIsCore,
				"class_subjects_desc":                   m.ClassSubjectsDesc,
				"class_subjects_weight_assignment":      m.ClassSubjectsWeightAssignment,
				"class_subjects_weight_quiz":            m.ClassSubjectsWeightQuiz,
				"class_subjects_weight_mid":             m.ClassSubjectsWeightMid,
				"class_subjects_weight_final":           m.ClassSubjectsWeightFinal,
				"class_subjects_min_attendance_percent": m.ClassSubjectsMinAttendancePct,
				"class_subjects_is_active":              m.ClassSubjectsIsActive,
			}).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Slug atau kombinasi kelas+subject sudah terdaftar")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}

		c.Locals("updated_class_subject", m)
		return nil
	}); err != nil {
		return err
	}

	// Response
	m := c.Locals("updated_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonUpdated(c, "Class subject berhasil diperbarui", csDTO.FromClassSubjectModel(m))
}

// util kecil
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

/*
=========================================================

	DELETE
	DELETE /admin/:masjid_id/class-subjects/:id?force=true
	=========================================================
*/
func (h *ClassSubjectController) Delete(c *fiber.Ctx) error {
	// 🔐 Context + role check (DKM/Admin)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Hanya Admin (bukan sekadar DKM) yang boleh hard delete
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
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
			// soft delete
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
