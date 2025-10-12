// file: internals/features/lembaga/class_subjects/controller/class_subject_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	csDTO "masjidku_backend/internals/features/school/academics/subject/dto"
	csModel "masjidku_backend/internals/features/school/academics/subject/model"

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
		// === Cek duplikasi kombinasi (tenant-aware, alive only) ===
		var cnt int64
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where(`
                class_subject_masjid_id = ?
                AND class_subject_parent_id = ?
                AND class_subject_subject_id = ?
                AND class_subject_deleted_at IS NULL
            `, req.MasjidID, req.ParentID, req.SubjectID).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Kombinasi parent+subject sudah terdaftar")
		}

		// === Generate slug unik ===
		baseSlug := ""
		if req.Slug != nil {
			baseSlug = *req.Slug
		} else {
			var subjName, parentSlug string
			_ = tx.Table("subjects").
				Select("subject_name").
				Where("subject_id = ? AND subject_masjid_id = ?", req.SubjectID, req.MasjidID).
				Scan(&subjName).Error

			_ = tx.Table("class_parents").
				Select("class_parent_slug").
				Where("class_parent_id = ? AND class_parent_masjid_id = ?", req.ParentID, req.MasjidID).
				Scan(&parentSlug).Error

			switch {
			case strings.TrimSpace(parentSlug) != "" && strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(parentSlug+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(parentSlug) != "":
				baseSlug = helper.Slugify(parentSlug, 160)
			default:
				baseSlug = helper.Slugify(
					fmt.Sprintf("cs-%s-%s",
						strings.Split(req.ParentID.String(), "-")[0],
						strings.Split(req.SubjectID.String(), "-")[0],
					), 160)
			}
		}

		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			"class_subjects",
			"class_subject_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("class_subject_masjid_id = ? AND class_subject_deleted_at IS NULL", req.MasjidID)
			},
			160,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}

		// Create
		m := req.ToModel()
		m.ClassSubjectSlug = &uniqueSlug

		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi parent+subject sudah terdaftar")
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
		if err := tx.Where("class_subject_id = ?", id).First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah data milik masjid lain")
		}
		if m.ClassSubjectDeletedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// Cek apakah parent/subject berubah → cek duplikasi
		newParentID := m.ClassSubjectParentID
		newSubjectID := m.ClassSubjectSubjectID
		if req.ParentID != nil {
			newParentID = *req.ParentID
		}
		if req.SubjectID != nil {
			newSubjectID = *req.SubjectID
		}

		if newParentID != m.ClassSubjectParentID || newSubjectID != m.ClassSubjectSubjectID {
			var cnt int64
			if err := tx.Model(&csModel.ClassSubjectModel{}).
				Where(`
					class_subject_masjid_id = ?
					AND class_subject_parent_id  = ?
					AND class_subject_subject_id = ?
					AND class_subject_id <> ?
					AND class_subject_deleted_at IS NULL
				`, masjidID, newParentID, newSubjectID, m.ClassSubjectID).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi parent+subject sudah terdaftar")
			}
		}

		// Flag untuk slug
		oldSlugNil := (m.ClassSubjectSlug == nil || strings.TrimSpace(ptrStr(m.ClassSubjectSlug)) == "")
		parentChanged := (req.ParentID != nil && *req.ParentID != m.ClassSubjectParentID)
		subjectChanged := (req.SubjectID != nil && *req.SubjectID != m.ClassSubjectSubjectID)

		// Apply perubahan dari req
		req.Apply(&m)

		// Slug handling
		needSetSlug := false
		var baseSlug string

		if req.Slug != nil {
			// User kasih slug manual
			baseSlug = *req.Slug
			needSetSlug = true
		} else if oldSlugNil || parentChanged || subjectChanged {
			// Slug kosong ATAU parent/subject berubah → regen
			needSetSlug = true

			var subjName, parentSlug string
			_ = tx.Table("subjects").
				Select("subject_name").
				Where("subject_id = ? AND subject_masjid_id = ?", m.ClassSubjectSubjectID, masjidID).
				Scan(&subjName).Error

			_ = tx.Table("class_parents").
				Select("class_parent_slug").
				Where("class_parent_id = ? AND class_parent_masjid_id = ?", m.ClassSubjectParentID, masjidID).
				Scan(&parentSlug).Error

			switch {
			case strings.TrimSpace(parentSlug) != "" && strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(parentSlug+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(parentSlug) != "":
				baseSlug = helper.Slugify(parentSlug, 160)
			default:
				baseSlug = helper.Slugify(
					fmt.Sprintf("cs-%s-%s",
						strings.Split(m.ClassSubjectParentID.String(), "-")[0],
						strings.Split(m.ClassSubjectSubjectID.String(), "-")[0],
					), 160)
			}
		}

		if needSetSlug {
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				tx,
				"class_subjects",
				"class_subject_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(`
						class_subject_masjid_id = ?
						AND class_subject_deleted_at IS NULL
						AND class_subject_id <> ?
					`, masjidID, m.ClassSubjectID)
				},
				160,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			m.ClassSubjectSlug = &uniqueSlug
		}

		// Persist ke DB
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where("class_subject_id = ?", m.ClassSubjectID).
			Updates(map[string]any{
				"class_subject_masjid_id":              m.ClassSubjectMasjidID,
				"class_subject_parent_id":              m.ClassSubjectParentID,
				"class_subject_subject_id":             m.ClassSubjectSubjectID,
				"class_subject_slug":                   m.ClassSubjectSlug,
				"class_subject_order_index":            m.ClassSubjectOrderIndex,
				"class_subject_hours_per_week":         m.ClassSubjectHoursPerWeek,
				"class_subject_min_passing_score":      m.ClassSubjectMinPassingScore,
				"class_subject_weight_on_report":       m.ClassSubjectWeightOnReport,
				"class_subject_is_core":                m.ClassSubjectIsCore,
				"class_subject_desc":                   m.ClassSubjectDesc,
				"class_subject_weight_assignment":      m.ClassSubjectWeightAssignment,
				"class_subject_weight_quiz":            m.ClassSubjectWeightQuiz,
				"class_subject_weight_mid":             m.ClassSubjectWeightMid,
				"class_subject_weight_final":           m.ClassSubjectWeightFinal,
				"class_subject_min_attendance_percent": m.ClassSubjectMinAttendancePercent,
				"class_subject_is_active":              m.ClassSubjectIsActive,
			}).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Slug atau kombinasi parent+subject sudah terdaftar")
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
		if err := tx.First(&m, "class_subject_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik masjid lain")
		}

		if force {
			// hard delete benar-benar hapus row
			if err := tx.Unscoped().Delete(&csModel.ClassSubjectModel{}, "class_subject_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			if m.ClassSubjectDeletedAt.Valid {
				return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
			}
			// soft delete
			if err := tx.Delete(&csModel.ClassSubjectModel{}, "class_subject_id = ?", id).Error; err != nil {
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
