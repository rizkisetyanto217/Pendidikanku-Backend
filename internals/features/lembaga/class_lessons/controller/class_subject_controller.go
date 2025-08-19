// internals/features/lembaga/class_subjects/controller/class_subject_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	csDTO "masjidku_backend/internals/features/lembaga/class_lessons/dto"
	csModel "masjidku_backend/internals/features/lembaga/class_lessons/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectController struct {
	DB *gorm.DB
}

/* =========================================================
   CREATE
   POST /admin/class-subjects
   Body: CreateClassSubjectRequest
   ========================================================= */
func (h *ClassSubjectController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
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
	if req.AcademicYear != nil {
		ay := strings.TrimSpace(*req.AcademicYear)
		req.AcademicYear = &ay
	}
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Cek duplikat kombinasi unik (abaikan soft-deleted)
		var cnt int64
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where(`
				class_subjects_masjid_id = ?
				AND class_subjects_class_id = ?
				AND class_subjects_subject_id = ?
				AND COALESCE(class_subjects_academic_year, '') = COALESCE(?, '')
				AND class_subjects_deleted_at IS NULL
			`,
				req.MasjidID, req.ClassID, req.SubjectID, req.AcademicYear,
			).Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
		}

		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
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
   GET BY ID
   GET /admin/class-subjects/:id[?with_deleted=true]
   ========================================================= */
func (h *ClassSubjectController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	withDeleted := strings.EqualFold(c.Query("with_deleted"), "true")

	var m csModel.ClassSubjectModel
	if err := h.DB.First(&m, "class_subjects_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Tenant guard
	if m.ClassSubjectsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Akses ditolak")
	}

	// Soft delete guard
	if !withDeleted && m.ClassSubjectsDeletedAt != nil {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	return helper.JsonOK(c, "Detail class subject", csDTO.FromClassSubjectModel(m))
}

/* =========================================================
   LIST
   GET /admin/class-subjects?q=&is_active=&order_by=&sort=&limit=&offset=&with_deleted=
   order_by: order_index|created_at|updated_at
   sort: asc|desc
   ========================================================= */
func (h *ClassSubjectController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var q csDTO.ListClassSubjectQuery
	// default pagination
	q.Limit, q.Offset = intPtr(20), intPtr(0)

	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	// normalisasi limit/offset
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 {
		q.Limit = intPtr(20)
	}
	if q.Offset == nil || *q.Offset < 0 {
		q.Offset = intPtr(0)
	}

	tx := h.DB.Model(&csModel.ClassSubjectModel{}).
		Where("class_subjects_masjid_id = ?", masjidID)

	// exclude soft-deleted by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_subjects_deleted_at IS NULL")
	}

	if q.IsActive != nil {
		tx = tx.Where("class_subjects_is_active = ?", *q.IsActive)
	}

	// simple search
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("LOWER(COALESCE(class_subjects_academic_year,'')) LIKE ?", kw)
	}

	// whitelist order by
	orderBy := "class_subjects_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "order_index":
			orderBy = "class_subjects_order_index"
		case "created_at":
			orderBy = "class_subjects_created_at"
		case "updated_at":
			orderBy = "class_subjects_updated_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// count total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ambil data
	var rows []csModel.ClassSubjectModel
	if err := tx.
		Select(`
			class_subjects_id,
			class_subjects_masjid_id,
			class_subjects_class_id,
			class_subjects_subject_id,
			class_subjects_order_index,
			class_subjects_hours_per_week,
			class_subjects_min_passing_score,
			class_subjects_weight_on_report,
			class_subjects_is_core,
			class_subjects_academic_year,
			class_subjects_desc,
			class_subjects_is_active,
			class_subjects_created_at,
			class_subjects_updated_at,
			class_subjects_deleted_at
		`).
		Order(orderBy + " " + sort).
		Limit(*q.Limit).
		Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonList(
		c,
		csDTO.FromClassSubjectModels(rows),
		csDTO.Pagination{
			Limit:  *q.Limit,
			Offset: *q.Offset,
			Total:  int(total),
		},
	)
}

/* =========================================================
   UPDATE (partial)
   PUT /admin/class-subjects/:id
   ========================================================= */
func (h *ClassSubjectController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
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
	if req.AcademicYear != nil {
		ay := strings.TrimSpace(*req.AcademicYear)
		req.AcademicYear = &ay
	}
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
		if m.ClassSubjectsDeletedAt != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// Jika kombinasi unik berpotensi berubah → cek duplikat
		shouldCheckDup := false
		newClassID := m.ClassSubjectsClassID
		newSubjectID := m.ClassSubjectsSubjectID
		var newAcademicYear *string = m.ClassSubjectsAcademicYear

		if req.ClassID != nil && *req.ClassID != m.ClassSubjectsClassID {
			shouldCheckDup = true
			newClassID = *req.ClassID
		}
		if req.SubjectID != nil && *req.SubjectID != m.ClassSubjectsSubjectID {
			shouldCheckDup = true
			newSubjectID = *req.SubjectID
		}
		if req.AcademicYear != nil {
			ay := strings.TrimSpace(*req.AcademicYear)
			curr := ""
			if m.ClassSubjectsAcademicYear != nil {
				curr = strings.TrimSpace(*m.ClassSubjectsAcademicYear)
			}
			if ay != curr {
				shouldCheckDup = true
			}
			if ay == "" {
				newAcademicYear = nil
			} else {
				newAcademicYear = &ay
			}
		}

		if shouldCheckDup {
			var cnt int64
			if err := tx.Model(&csModel.ClassSubjectModel{}).
				Where(`
					class_subjects_masjid_id = ?
					AND class_subjects_class_id = ?
					AND class_subjects_subject_id = ?
					AND COALESCE(class_subjects_academic_year, '') = COALESCE(?, '')
					AND class_subjects_id <> ?
					AND class_subjects_deleted_at IS NULL
				`,
					masjidID, newClassID, newSubjectID, newAcademicYear, m.ClassSubjectsID,
				).Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
			}
		}

		// Apply & update timestamp
		req.Apply(&m)
		now := time.Now()
		m.ClassSubjectsUpdatedAt = &now

		patch := map[string]interface{}{
			"class_subjects_masjid_id":         m.ClassSubjectsMasjidID,
			"class_subjects_class_id":          m.ClassSubjectsClassID,
			"class_subjects_subject_id":        m.ClassSubjectsSubjectID,
			"class_subjects_order_index":       m.ClassSubjectsOrderIndex,
			"class_subjects_hours_per_week":    m.ClassSubjectsHoursPerWeek,
			"class_subjects_min_passing_score": m.ClassSubjectsMinPassingScore,
			"class_subjects_weight_on_report":  m.ClassSubjectsWeightOnReport,
			"class_subjects_is_core":           m.ClassSubjectsIsCore,
			"class_subjects_academic_year":     m.ClassSubjectsAcademicYear,
			"class_subjects_desc":              m.ClassSubjectsDesc,
			"class_subjects_is_active":         m.ClassSubjectsIsActive,
			"class_subjects_updated_at":        m.ClassSubjectsUpdatedAt,
		}

		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where("class_subjects_id = ?", m.ClassSubjectsID).
			Updates(patch).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
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
   - force=true (admin saja): hard delete
   - default: soft delete (set deleted_at)
   ========================================================= */
func (h *ClassSubjectController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// Hanya admin yang boleh hard delete
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
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
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik masjid lain")
		}

		if force {
			if err := tx.Delete(&csModel.ClassSubjectModel{}, "class_subjects_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			if m.ClassSubjectsDeletedAt != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
			}
			now := time.Now()
			if err := tx.Model(&csModel.ClassSubjectModel{}).
				Where("class_subjects_id = ?", id).
				Updates(map[string]interface{}{
					"class_subjects_deleted_at": &now,
					"class_subjects_updated_at": &now,
				}).Error; err != nil {
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
