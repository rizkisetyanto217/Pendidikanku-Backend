// internals/features/lembaga/class_section_subject_teachers/controller/csst_controller.go
package controller

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/lembaga/class_lessons/dto"
	model "masjidku_backend/internals/features/lembaga/class_lessons/model"
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

	// build dari DTO, lalu force tenant
	row := req.ToModel()
	row.ClassSectionSubjectTeacherModelMasjidID = masjidID

	if err := ctl.DB.Create(&row).Error; err != nil {
		msg := strings.ToLower(err.Error())
		// unique constraint
		if strings.Contains(msg, "uq_class_sec_subj_teachers_active") ||
			strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Penugasan guru untuk section+subject ini sudah aktif (duplikat).")
		}
		// foreign key
		if strings.Contains(msg, "sqlstate 23503") || strings.Contains(msg, "foreign key") {
			return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/subject/guru ada dan sesuai tenant.")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Insert gagal: "+err.Error())
	}

	return helper.JsonCreated(c, "Penugasan guru berhasil dibuat", dto.FromClassSectionSubjectTeacherModel(row))
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

	tx := ctl.DB.Model(&model.ClassSectionSubjectTeacherModel{}).
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

	var rows []model.ClassSectionSubjectTeacherModel
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

	var row model.ClassSectionSubjectTeacherModel
	if err := ctl.DB.
		Where("class_section_subject_teachers_id = ?", id).
		First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// tenant guard
	if row.ClassSectionSubjectTeacherModelMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	// soft-delete guard
	if !withDeleted && row.ClassSectionSubjectTeacherModelDeletedAt.Valid {
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

	var row model.ClassSectionSubjectTeacherModel
	if err := ctl.DB.
		Where("class_section_subject_teachers_id = ? AND class_section_subject_teachers_deleted_at IS NULL", id).
		First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// tenant guard
	if row.ClassSectionSubjectTeacherModelMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	// partial update via DTO
	req.Apply(&row)

	if err := ctl.DB.Save(&row).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_class_sec_subj_teachers_active") ||
			strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Penugasan guru untuk section+subject ini sudah aktif (duplikat).")
		}
		if strings.Contains(msg, "sqlstate 23503") || strings.Contains(msg, "foreign key") {
			return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/subject/guru ada dan sesuai tenant.")
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

	var row model.ClassSectionSubjectTeacherModel
	if err := ctl.DB.First(&row, "class_section_subject_teachers_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	if row.ClassSectionSubjectTeacherModelMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	if row.ClassSectionSubjectTeacherModelDeletedAt.Valid {
		// idempotent
		return helper.JsonDeleted(c, "Sudah terhapus", fiber.Map{"id": id})
	}

	if err := ctl.DB.
		Model(&model.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teachers_id = ?", id).
		Update("class_section_subject_teachers_deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Penugasan guru berhasil dihapus", fiber.Map{"id": id})
}
