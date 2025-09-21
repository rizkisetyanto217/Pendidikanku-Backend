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
	// Force tenant
	req.MasjidID = masjidID

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
			// ratakan dulu ke bentuk slug dasar
			s = helper.Slugify(s, 160)
			req.Slug = &s
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// === Cek duplikasi kombinasi (soft delete aware) ===
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
			return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+term/semester sudah terdaftar")
		}

		// === Generate slug unik (kalau kosong, auto-suggest) ===
		baseSlug := ""
		if req.Slug != nil {
			baseSlug = *req.Slug
		} else {
			// Coba rakit dari nama subject & class (fallback ke UUID pendek)
			var subjName, className string
			_ = tx.Table("subjects").
				Select("subjects_name").
				Where("subjects_id = ? AND subjects_masjid_id = ?", req.SubjectID, req.MasjidID).
				Scan(&subjName).Error
			_ = tx.Table("classes").
				Select("classes_name").
				Where("classes_id = ? AND classes_masjid_id = ?", req.ClassID, req.MasjidID).
				Scan(&className).Error

			switch {
			case strings.TrimSpace(subjName) != "" && strings.TrimSpace(className) != "":
				baseSlug = helper.Slugify(className+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(className) != "":
				baseSlug = helper.Slugify(className, 160)
			default:
				// fallback: potong UUID biar singkat
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
				// unik per masjid, abaikan soft-deleted
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

/*
=========================================================

	UPDATE (partial)
	PUT /admin/:masjid_id/class-subjects/:id

=========================================================
*/
/*
=========================================================

  UPDATE (partial)
  PUT /admin/:masjid_id/class-subjects/:id

=========================================================
*/
func (h *ClassSubjectController) Update(c *fiber.Ctx) error {
	// 🔐 Context + role check
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

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
	if req.Slug != nil {
		s := strings.TrimSpace(*req.Slug)
		if s == "" {
			// kosongkan agar dianggap "minta auto-generate jika perlu"
			req.Slug = nil
		} else {
			// ratakan ke slug dasar
			s = helper.Slugify(s, 160)
			req.Slug = &s
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Ambil record
		var m csModel.ClassSubjectModel
		if err := tx.
			Where("class_subjects_id = ?", id).
			First(&m).Error; err != nil {
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

		if req.ClassID != nil && *req.ClassID != m.ClassSubjectsClassID {
			shouldCheckDup = true
			newClassID = *req.ClassID
		}
		if req.SubjectID != nil && *req.SubjectID != m.ClassSubjectsSubjectID {
			shouldCheckDup = true
			newSubjectID = *req.SubjectID
		}

		if shouldCheckDup {
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

		// ==== Slug handling ====
		// 1) Apply dulu perubahan lain ke model (kita belum set slug di sini)
		//    supaya jika class/subject berubah, nama untuk auto-slug bisa mengacu kondisi baru.
		oldSlugNil := (m.ClassSubjectsSlug == nil || strings.TrimSpace(ptrStr(m.ClassSubjectsSlug)) == "")
		req.Apply(&m)

		// 2) Tentukan apakah perlu set/generate slug
		needSetSlug := false
		var baseSlug string

		if req.Slug != nil {
			// User explicitly memberikan slug → gunakan sebagai base
			baseSlug = *req.Slug
			needSetSlug = true
		} else if oldSlugNil {
			// sebelumnya belum punya slug; jika class/subject berubah atau awalnya kosong → generate
			needSetSlug = true
			// coba rakit dari nama terbaru (post-Apply)
			var subjName, className string
			_ = tx.Table("subjects").
				Select("subjects_name").
				Where("subjects_id = ? AND subjects_masjid_id = ?", m.ClassSubjectsSubjectID, masjidID).
				Scan(&subjName).Error
			_ = tx.Table("classes").
				Select("classes_name").
				Where("classes_id = ? AND classes_masjid_id = ?", m.ClassSubjectsClassID, masjidID).
				Scan(&className).Error

			switch {
			case strings.TrimSpace(className) != "" && strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(className+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(className) != "":
				baseSlug = helper.Slugify(className, 160)
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
					// unik per masjid, abaikan soft-deleted, exclude diri sendiri
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

		// ==== Persist ====
		// Gunakan Updates(map) agar aman untuk partial nullable; sertakan kolom yang relevan.
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where("class_subjects_id = ?", m.ClassSubjectsID).
			Updates(map[string]interface{}{
				"class_subjects_masjid_id":  m.ClassSubjectsMasjidID,
				"class_subjects_class_id":   m.ClassSubjectsClassID,
				"class_subjects_subject_id": m.ClassSubjectsSubjectID,
				"class_subjects_slug":       m.ClassSubjectsSlug,

				"class_subjects_order_index":       m.ClassSubjectsOrderIndex,
				"class_subjects_hours_per_week":    m.ClassSubjectsHoursPerWeek,
				"class_subjects_min_passing_score": m.ClassSubjectsMinPassingScore,
				"class_subjects_weight_on_report":  m.ClassSubjectsWeightOnReport,
				"class_subjects_is_core":           m.ClassSubjectsIsCore,
				"class_subjects_desc":              m.ClassSubjectsDesc,

				// bobot (smallint di model → pointer)
				"class_subjects_weight_assignment":  m.ClassSubjectsWeightAssignment,
				"class_subjects_weight_quiz":        m.ClassSubjectsWeightQuiz,
				"class_subjects_weight_mid":         m.ClassSubjectsWeightMid,
				"class_subjects_weight_final":       m.ClassSubjectsWeightFinal,
				"class_subjects_min_attendance_pct": m.ClassSubjectsMinAttendancePct,

				// image
				"class_subjects_image_url":        m.ClassSubjectsImageURL,
				"class_subjects_image_object_key": m.ClassSubjectsImageObjectKey,

				// status
				"class_subjects_is_active": m.ClassSubjectsIsActive,
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
