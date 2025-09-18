// internals/features/lembaga/subjects/main/controller/subjects_controller.go
package controller

import (
	"errors"
	"log" // ⬅️ tambahkan
	"strings"
	"time"

	subjectDTO "masjidku_backend/internals/features/school/subject_books/subject/dto"
	subjectModel "masjidku_backend/internals/features/school/subject_books/subject/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubjectsController struct {
	DB *gorm.DB
}

/* =========================
   CREATE — POST /admin/subjects
   ========================= */
func (h *SubjectsController) CreateSubject(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][CREATE] ▶️ incoming request")
	// pastikan DB tersedia utk resolver slug→id jika perlu
	c.Locals("DB", h.DB)

	// 1) Masjid context (path/header/cookie/query/host → fallback token)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	default:
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
		}
		masjidID = id
	}
	// 2) Staff guard (admin DKM untuk endpoint /admin)
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}
	log.Printf("[SUBJECTS][CREATE] 🕌 masjid_id=%s", masjidID)

	// 3) Parse + normalize
	var req subjectDTO.CreateSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.MasjidID = masjidID

	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		if d == "" {
			req.Desc = nil
		} else {
			req.Desc = &d
		}
	}
	// slug: dari body → normalize; kalau kosong generate dari name
	if req.Slug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.Slug))
		if s == "" {
			req.Slug = nil
		} else {
			req.Slug = &s
		}
	} else {
		if s := helper.GenerateSlug(req.Name); s != "" {
			req.Slug = &s
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 4) TX + DB.WithContext
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// cek unique code per masjid (alive)
		var cnt int64
		if err := tx.Model(&subjectModel.SubjectsModel{}).
			Where(`
				subjects_masjid_id = ?
				AND lower(subjects_code) = lower(?)
				AND subjects_deleted_at IS NULL
			`, req.MasjidID, req.Code).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Kode mapel sudah digunakan")
		}

		// cek unique slug per masjid (alive) jika ada
		if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
			cnt = 0
			if err := tx.Model(&subjectModel.SubjectsModel{}).
				Where(`
					subjects_masjid_id = ?
					AND lower(subjects_slug) = lower(?)
					AND subjects_deleted_at IS NULL
				`, req.MasjidID, *req.Slug).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi slug")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
			}
		}

		// simpan
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_subjects_code_per_masjid"),
				strings.Contains(msg, "duplicate"), strings.Contains(msg, "unique"):
				if req.Slug != nil {
					return fiber.NewError(fiber.StatusConflict, "Kode/Slug sudah digunakan di masjid ini")
				}
				return fiber.NewError(fiber.StatusConflict, "Kode mapel sudah digunakan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat subject")
		}
		c.Locals("created_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("created_subject").(subjectModel.SubjectsModel)
	log.Printf("[SUBJECTS][CREATE] ✅ created subjects_id=%s", m.SubjectsID)
	return helper.JsonCreated(c, "Subject berhasil dibuat", subjectDTO.FromSubjectModel(m))
}

/* =========================
   UPDATE — PUT /admin/subjects/:id
   ========================= */
func (h *SubjectsController) UpdateSubject(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][UPDATE] ▶️ incoming request")
	c.Locals("DB", h.DB)

	// Resolve masjid context + guard
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	default:
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
		}
		masjidID = id
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}
	log.Printf("[SUBJECTS][UPDATE] 🕌 masjid_id=%s", masjidID)

	// Params
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Body
	var req subjectDTO.UpdateSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.MasjidID = &masjidID

	// normalize string
	if req.Code != nil {
		s := strings.TrimSpace(*req.Code)
		req.Code = &s
	}
	if req.Name != nil {
		s := strings.TrimSpace(*req.Name)
		req.Name = &s
	}
	if req.Desc != nil {
		s := strings.TrimSpace(*req.Desc)
		req.Desc = &s
	}
	// slug normalize/generate
	if req.Slug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.Slug))
		if s == "" {
			req.Slug = nil
		} else {
			req.Slug = &s
		}
	} else if req.Name != nil {
		if s := helper.GenerateSlug(*req.Name); s != "" {
			req.Slug = &s
		}
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// ambil existing
		var m subjectModel.SubjectsModel
		if err := tx.First(&m, "subjects_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Subject tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		// tenant guard: record harus milik masjid context
		if m.SubjectsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah subject milik masjid lain")
		}
		if m.SubjectsDeletedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Subject sudah dihapus")
		}

		// cek duplikat code jika berubah
		if req.Code != nil && *req.Code != m.SubjectsCode {
			var cnt int64
			if err := tx.Model(&subjectModel.SubjectsModel{}).
				Where(`
					subjects_masjid_id = ?
					AND lower(subjects_code) = lower(?)
					AND subjects_id <> ?
					AND subjects_deleted_at IS NULL
				`, masjidID, *req.Code, m.SubjectsID).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Kode mapel sudah digunakan")
			}
		}

		// cek duplikat slug jika dikirim & berubah
		if req.Slug != nil {
			needCheck := m.SubjectsSlug == nil || !strings.EqualFold(*m.SubjectsSlug, *req.Slug)
			if needCheck {
				var cnt int64
				if err := tx.Model(&subjectModel.SubjectsModel{}).
					Where(`
						subjects_masjid_id = ?
						AND subjects_id <> ?
						AND subjects_deleted_at IS NULL
						AND subjects_slug IS NOT NULL
						AND lower(subjects_slug) = lower(?)
					`, masjidID, m.SubjectsID, *req.Slug).
					Count(&cnt).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi slug")
				}
				if cnt > 0 {
					return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
				}
			}
		}

		// apply & save
		req.Apply(&m)
		m.SubjectsUpdatedAt = time.Now()

		patch := map[string]any{
			"subjects_masjid_id":  m.SubjectsMasjidID,
			"subjects_code":       m.SubjectsCode,
			"subjects_name":       m.SubjectsName,
			"subjects_desc":       m.SubjectsDesc,
			"subjects_is_active":  m.SubjectsIsActive,
			"subjects_updated_at": m.SubjectsUpdatedAt,
		}
		if req.Slug != nil {
			patch["subjects_slug"] = m.SubjectsSlug
		}

		if err := tx.Model(&subjectModel.SubjectsModel{}).
			Where("subjects_id = ?", m.SubjectsID).
			Updates(patch).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_subjects_code_per_masjid"):
				return fiber.NewError(fiber.StatusConflict, "Kode mapel sudah digunakan")
			case strings.Contains(msg, "uq_subjects_slug_per_masjid"):
				return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
			case strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique"):
				return fiber.NewError(fiber.StatusConflict, "Duplikasi data (kode/slug)")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui subject")
		}

		c.Locals("updated_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("updated_subject").(subjectModel.SubjectsModel)
	log.Printf("[SUBJECTS][UPDATE] ✅ updated subjects_id=%s", m.SubjectsID)
	return helper.JsonUpdated(c, "Subject berhasil diperbarui", subjectDTO.FromSubjectModel(m))
}

/* =========================
   DELETE — DELETE /admin/subjects/:id?force=true
   ========================= */
func (h *SubjectsController) DeleteSubject(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][DELETE] ▶️ incoming request")
	c.Locals("DB", h.DB)

	// context + guard
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	default:
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
		}
		masjidID = id
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}
	log.Printf("[SUBJECTS][DELETE] 🕌 masjid_id=%s", masjidID)

	// only admin (DKM) boleh force
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
		var m subjectModel.SubjectsModel
		if err := tx.First(&m, "subjects_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Subject tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.SubjectsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus subject milik masjid lain")
		}

		if force {
			if err := tx.Unscoped().
				Delete(&subjectModel.SubjectsModel{}, "subjects_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus subject")
			}
		} else {
			if m.SubjectsDeletedAt.Valid {
				return fiber.NewError(fiber.StatusBadRequest, "Subject sudah dihapus")
			}
			if err := tx.Delete(&subjectModel.SubjectsModel{}, "subjects_id = ?", id).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus subject")
			}
		}

		c.Locals("deleted_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("deleted_subject").(subjectModel.SubjectsModel)
	log.Printf("[SUBJECTS][DELETE] ✅ deleted subjects_id=%s force=%v", m.SubjectsID, force)
	return helper.JsonDeleted(c, "Subject berhasil dihapus", subjectDTO.FromSubjectModel(m))
}
