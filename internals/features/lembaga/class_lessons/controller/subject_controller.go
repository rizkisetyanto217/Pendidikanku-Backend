// internals/features/lembaga/subjects/main/controller/subjects_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	subjectDTO "masjidku_backend/internals/features/lembaga/class_lessons/dto"
	subjectModel "masjidku_backend/internals/features/lembaga/class_lessons/model" // mengikuti import di DTO
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubjectsController struct {
	DB *gorm.DB
}

/* =========================================================
   CREATE
   POST /admin/subjects
   Body: CreateSubjectRequest
   ========================================================= */
func (h *SubjectsController) CreateSubject(c *fiber.Ctx) error {
	// tenant guard (admin/teacher)
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	var req subjectDTO.CreateSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// force tenant
	req.MasjidID = masjidID

	// trim ringan (tambahan selain yang ada di ToModel)
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc); req.Desc = &d
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// transaksi kecil
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// duplikat code (per masjid) — case-insensitive
		var cnt int64
		if err := tx.Model(&subjectModel.SubjectsModel{}).
			Where("subjects_masjid_id = ? AND LOWER(subjects_code) = LOWER(?)", req.MasjidID, req.Code).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Kode mapel sudah digunakan")
		}

		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
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
	return helper.JsonCreated(c, "Subject berhasil dibuat", subjectDTO.FromSubjectModel(m))
}

/* =========================================================
   GET BY ID
   GET /admin/subjects/:id
   ========================================================= */
func (h *SubjectsController) GetSubject(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	var m subjectModel.SubjectsModel
	if err := h.DB.First(&m, "subjects_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Subject tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.SubjectsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Akses ditolak")
	}

	return helper.JsonOK(c, "Detail subject ditemukan", subjectDTO.FromSubjectModel(m))
}

/* =========================================================
   LIST
   GET /admin/subjects?q=&is_active=&order_by=&sort=&limit=&offset=
   order_by: code|name|created_at
   sort: asc|desc
   ========================================================= */
func (h *SubjectsController) ListSubjects(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	// Query params
	var q subjectDTO.ListSubjectQuery
	q.Limit, q.Offset = intPtr(20), intPtr(0)
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 { q.Limit = intPtr(20) }
	if q.Offset == nil || *q.Offset < 0 { q.Offset = intPtr(0) }

	tx := h.DB.Model(&subjectModel.SubjectsModel{}).
		Where("subjects_masjid_id = ?", masjidID)

	if q.IsActive != nil {
		tx = tx.Where("subjects_is_active = ?", *q.IsActive)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("LOWER(subjects_code) LIKE ? OR LOWER(subjects_name) LIKE ?", kw, kw)
	}

	// OrderBy whitelist
	orderBy := "subjects_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "code":
			orderBy = "subjects_code"
		case "name":
			orderBy = "subjects_name"
		case "created_at":
			orderBy = "subjects_created_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// data
	var rows []subjectModel.SubjectsModel
	if err := tx.
		Select(`
			subjects_id,
			subjects_masjid_id,
			subjects_code,
			subjects_name,
			subjects_desc,
			subjects_is_active,
			subjects_created_at,
			subjects_updated_at
		`).
		Order(orderBy + " " + sort).
		Limit(*q.Limit).Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Daftar subject berhasil diambil", subjectDTO.SubjectListResponse{
		Items: subjectDTO.FromSubjectModels(rows),
		Meta: subjectDTO.ListMeta{
			Limit:  *q.Limit,
			Offset: *q.Offset,
			Total:  int(total),
		},
	})
}

/* =========================================================
   UPDATE (partial)
   PUT /admin/subjects/:id
   ========================================================= */
func (h *SubjectsController) UpdateSubject(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	var req subjectDTO.UpdateSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// force tenant
	req.MasjidID = &masjidID

	// normalisasi string
	if req.Code != nil { s := strings.TrimSpace(*req.Code); req.Code = &s }
	if req.Name != nil { s := strings.TrimSpace(*req.Name); req.Name = &s }
	if req.Desc != nil { s := strings.TrimSpace(*req.Desc); req.Desc = &s }

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// transaksi
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// ambil existing
		var m subjectModel.SubjectsModel
		if err := tx.First(&m, "subjects_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Subject tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.SubjectsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah subject milik masjid lain")
		}

		// jika code berubah → cek duplikat
		if req.Code != nil && *req.Code != m.SubjectsCode {
			var cnt int64
			if err := tx.Model(&subjectModel.SubjectsModel{}).
				Where(`
					subjects_masjid_id = ?
					AND LOWER(subjects_code) = LOWER(?)
					AND subjects_id <> ?
				`, masjidID, *req.Code, m.SubjectsID).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Kode mapel sudah digunakan")
			}
		}

		// apply perubahan
		req.Apply(&m)
		now := time.Now()
		m.SubjectsUpdatedAt = &now

		// update field spesifik (hindari overwrite tak sengaja)
		patch := map[string]interface{}{
			"subjects_masjid_id":  m.SubjectsMasjidID,
			"subjects_code":       m.SubjectsCode,
			"subjects_name":       m.SubjectsName,
			"subjects_desc":       m.SubjectsDesc,
			"subjects_is_active":  m.SubjectsIsActive,
			"subjects_updated_at": m.SubjectsUpdatedAt,
		}

		if err := tx.Model(&subjectModel.SubjectsModel{}).
			Where("subjects_id = ?", m.SubjectsID).
			Updates(patch).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kode mapel sudah digunakan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui subject")
		}

		c.Locals("updated_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("updated_subject").(subjectModel.SubjectsModel)
	return helper.JsonUpdated(c, "Subject berhasil diperbarui", subjectDTO.FromSubjectModel(m))
}

/* =========================================================
   DELETE
   DELETE /admin/subjects/:id?force=true
   ========================================================= */
func (h *SubjectsController) DeleteSubject(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	// Hanya admin boleh force delete
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
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

		// TODO (opsional): cek relasi (kelas/mapel pakai subject ini). Blokir jika ada & force=false.

		var delErr error
		if force {
			delErr = tx.Delete(&subjectModel.SubjectsModel{}, "subjects_id = ?", id).Error
		} else {
			delErr = tx.Where("subjects_id = ?", id).Delete(&subjectModel.SubjectsModel{}).Error
		}
		if delErr != nil {
			msg := strings.ToLower(delErr.Error())
			if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
				return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus subject")
		}

		c.Locals("deleted_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("deleted_subject").(subjectModel.SubjectsModel)
	return helper.JsonDeleted(c, "Subject berhasil dihapus", subjectDTO.FromSubjectModel(m))
}

/* =========================================================
   Utils
   ========================================================= */
func intPtr(v int) *int { return &v }
