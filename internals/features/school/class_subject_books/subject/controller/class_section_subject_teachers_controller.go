// internals/features/lembaga/class_section_subject_teachers/controller/csst_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	modelMasjidTeacher "masjidku_backend/internals/features/lembaga/masjid_admins_teachers/admins_teachers/model"
	modelCSST "masjidku_backend/internals/features/school/class_subject_books/subject/model"
	modelClassSection "masjidku_backend/internals/features/school/classes/class_sections/model"

	dto "masjidku_backend/internals/features/school/class_subject_books/subject/dto"
	helper "masjidku_backend/internals/helpers"
)

type ClassSectionSubjectTeacherController struct {
	DB *gorm.DB
}




// ===============================
// CREATE (force masjid_id dari token)
// POST /admin/class-section-subject-teachers
// ===============================
func (ctl *ClassSectionSubjectTeacherController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var req dto.CreateClassSectionSubjectTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return ctl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) SECTION harus ada & tenant cocok (ambil class_id dari section)
		var sec modelClassSection.ClassSectionModel
		if err := tx.
			Where("class_sections_id = ? AND class_sections_masjid_id = ?",
				req.ClassSectionSubjectTeachersSectionID, masjidID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek section")
		}

		// 2) CLASS_SUBJECTS harus ada; tenant sama; dan class_id (cs) == class_id (section)
		var cs struct {
			ClassSubjectsID       uuid.UUID `gorm:"column:class_subjects_id"`
			ClassSubjectsMasjidID uuid.UUID `gorm:"column:class_subjects_masjid_id"`
			ClassSubjectsClassID  uuid.UUID `gorm:"column:class_subjects_class_id"`
		}
		if err := tx.Table("class_subjects").
			Select("class_subjects_id, class_subjects_masjid_id, class_subjects_class_id").
			Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL",
				req.ClassSectionSubjectTeachersClassSubjectsID).
			Take(&cs).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "class_subjects tidak ditemukan / sudah dihapus")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek class_subjects")
		}
		if cs.ClassSubjectsMasjidID != masjidID {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid mismatch: class_subjects milik masjid lain")
		}
		if cs.ClassSubjectsClassID != sec.ClassSectionsClassID {
			return helper.JsonError(c, fiber.StatusBadRequest,
				"Class mismatch: class_subjects.class_id != class_sections.class_id")
		}

		// 3) TEACHER harus ada & tenant cocok
		if err := tx.
			Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ?",
				req.ClassSectionSubjectTeachersTeacherID, masjidID).
			First(&modelMasjidTeacher.MasjidTeacherModel{}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Guru tidak ditemukan / bukan guru masjid ini")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek guru")
		}

		// 4) Build row + force tenant + default aktif
		row := req.ToModel()
		row.ClassSectionSubjectTeachersMasjidID = masjidID
		if !row.ClassSectionSubjectTeachersIsActive {
			row.ClassSectionSubjectTeachersIsActive = true
		}

		if err := tx.Create(&row).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csst_active_by_cs"),
				strings.Contains(msg, "uq_csst_active_unique"),
				strings.Contains(msg, "duplicate"),
				strings.Contains(msg, "unique"):
				return helper.JsonError(c, fiber.StatusConflict, "Penugasan guru untuk class_subjects ini sudah aktif (duplikat).")

			case strings.Contains(msg, "23503"), strings.Contains(msg, "foreign key"):
				switch {
				case strings.Contains(msg, "fk_csst_section_masjid"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_subjects"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECTS): tidak ditemukan / beda tenant")
				case strings.Contains(msg, "masjid_teachers"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
				default:
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/class_subjects/guru valid")
				}
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Insert gagal: "+err.Error())
		}

		return helper.JsonCreated(c, "Penugasan guru berhasil dibuat", dto.FromClassSectionSubjectTeacherModel(row))
	})
}
// ===============================
// LIST
// GET /admin/class-section-subject-teachers?is_active=&with_deleted=&limit=&offset=&order_by=&sort=
// order_by: created_at|updated_at
// sort: asc|desc
// ===============================
type listQuery struct {
	IsActive    *bool   `query:"is_active"`
	WithDeleted *bool   `query:"with_deleted"`
	Limit       *int    `query:"limit"`
	Offset      *int    `query:"offset"`
	OrderBy     *string `query:"order_by"`
	Sort        *string `query:"sort"`
}

func (ctl *ClassSectionSubjectTeacherController) List(c *fiber.Ctx) error {
	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	var q listQuery
	q.Limit, q.Offset = intPtr(20), intPtr(0)
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 {
		q.Limit = intPtr(20)
	}
	if q.Offset == nil || *q.Offset < 0 {
		q.Offset = intPtr(0)
	}

	tx := ctl.DB.Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teachers_masjid_id IN ?", masjidIDs)

	// exclude soft-deleted by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_section_subject_teachers_deleted_at IS NULL")
	}
	if q.IsActive != nil {
		tx = tx.Where("class_section_subject_teachers_is_active = ?", *q.IsActive)
	}

	orderBy := "class_section_subject_teachers_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "created_at":
			orderBy = "class_section_subject_teachers_created_at"
		case "updated_at":
			orderBy = "class_section_subject_teachers_updated_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	var rows []modelCSST.ClassSectionSubjectTeacherModel
	if err := tx.
		Order(orderBy + " " + sort).
		Limit(*q.Limit).
		Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonList(c,
		dto.FromClassSectionSubjectTeacherModels(rows),
		fiber.Map{
			"limit":  *q.Limit,
			"offset": *q.Offset,
			"total":  int(total),
		},
	)
}

// ===============================
// GET BY ID
// GET /admin/class-section-subject-teachers/:id[?with_deleted=true]
// ===============================
func (ctl *ClassSectionSubjectTeacherController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID tidak valid")
	}

	withDeleted := strings.EqualFold(c.Query("with_deleted"), "true")

	var row modelCSST.ClassSectionSubjectTeacherModel
	if err := ctl.DB.
		Where("class_section_subject_teachers_id = ?", id).
		First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// tenant guard
	if row.ClassSectionSubjectTeachersMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	// soft-delete guard
	if !withDeleted && row.ClassSectionSubjectTeachersDeletedAt.Valid {
		return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
	}

	return helper.JsonOK(c, "Detail penugasan guru", dto.FromClassSectionSubjectTeacherModel(row))
}

// ===============================
// UPDATE (partial)
// PUT /admin/class-section-subject-teachers/:id
// ===============================
func (ctl *ClassSectionSubjectTeacherController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateClassSectionSubjectTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "Payload tidak valid")
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var row modelCSST.ClassSectionSubjectTeacherModel
	if err := ctl.DB.
		Where("class_section_subject_teachers_id = ? AND class_section_subject_teachers_deleted_at IS NULL", id).
		First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// tenant guard
	if row.ClassSectionSubjectTeachersMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	// (opsional) precheck konsistensi jika section_id / class_subjects_id berubah
	if req.ClassSectionSubjectTeachersSectionID != nil || req.ClassSectionSubjectTeachersClassSubjectsID != nil {
		sectionID := row.ClassSectionSubjectTeachersSectionID
		if req.ClassSectionSubjectTeachersSectionID != nil {
			sectionID = *req.ClassSectionSubjectTeachersSectionID
		}
		classSubjectsID := row.ClassSectionSubjectTeachersClassSubjectsID
		if req.ClassSectionSubjectTeachersClassSubjectsID != nil {
			classSubjectsID = *req.ClassSectionSubjectTeachersClassSubjectsID
		}

		// cek section milik tenant
		if err := ctl.DB.
			Where("class_sections_id = ? AND class_sections_masjid_id = ?", sectionID, masjidID).
			First(&modelClassSection.ClassSectionModel{}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek section")
		}

		// cek class_subjects cocok
		var cs struct {
			ClassSubjectsMasjidID uuid.UUID `gorm:"column:class_subjects_masjid_id"`
			ClassSubjectsClassID  uuid.UUID `gorm:"column:class_subjects_class_id"`
		}
		if err := ctl.DB.Table("class_subjects").
			Select("class_subjects_masjid_id, class_subjects_class_id").
			Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL", classSubjectsID).
			Take(&cs).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "class_subjects tidak ditemukan / sudah dihapus")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek class_subjects")
		}
		if cs.ClassSubjectsMasjidID != masjidID {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid mismatch: class_subjects milik masjid lain")
		}
		if cs.ClassSubjectsClassID != sectionID {
			return helper.JsonError(c, fiber.StatusBadRequest, "Section mismatch: class_subjects.class_id != section_id yang dikirim")
		}
	}

	// partial update via DTO
	req.Apply(&row)

	if err := ctl.DB.Save(&row).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_csst_active_by_cs") ||
			strings.Contains(msg, "uq_csst_active_unique") ||
			strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Penugasan guru untuk class_subjects ini sudah aktif (duplikat).")
		}
		if strings.Contains(msg, "sqlstate 23503") || strings.Contains(msg, "foreign key") {
			switch {
			case strings.Contains(msg, "fk_csst_section_masjid"):
				return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
			case strings.Contains(msg, "fk_csst_to_class_subjects"), strings.Contains(msg, "class_subjects"):
				return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECTS): tidak ditemukan / beda tenant")
			case strings.Contains(msg, "masjid_teachers"):
				return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
			default:
				return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/class_subjects/guru valid")
			}
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Penugasan guru berhasil diperbarui", dto.FromClassSectionSubjectTeacherModel(row))
}

// ===============================
// DELETE (soft delete)
// DELETE /admin/class-section-subject-teachers/:id
// ===============================
func (ctl *ClassSectionSubjectTeacherController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID tidak valid")
	}

	var row modelCSST.ClassSectionSubjectTeacherModel
	if err := ctl.DB.First(&row, "class_section_subject_teachers_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	if row.ClassSectionSubjectTeachersMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	if row.ClassSectionSubjectTeachersDeletedAt.Valid {
		// idempotent
		return helper.JsonDeleted(c, "Sudah terhapus", fiber.Map{"id": id})
	}

	if err := ctl.DB.
		Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teachers_id = ?", id).
		Update("class_section_subject_teachers_deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Penugasan guru berhasil dihapus", fiber.Map{"id": id})
}
