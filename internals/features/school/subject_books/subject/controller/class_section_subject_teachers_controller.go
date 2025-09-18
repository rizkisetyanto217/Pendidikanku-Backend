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

	modelMasjidTeacher "masjidku_backend/internals/features/lembaga/teachers_students/model"
	modelClassSection "masjidku_backend/internals/features/school/classes/class_sections/model"
	modelCSST "masjidku_backend/internals/features/school/subject_books/subject/model"

	dto "masjidku_backend/internals/features/school/subject_books/subject/dto"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type ClassSectionSubjectTeacherController struct {
	DB *gorm.DB
}

func NewClassSectionSubjectTeacherController(db *gorm.DB) *ClassSectionSubjectTeacherController {
	return &ClassSectionSubjectTeacherController{DB: db}
}

/* ===============================
   CREATE (admin/DKM via masjid context)
   POST /admin/:masjid_id/class-section-subject-teachers
   /admin/:masjid_slug/class-section-subject-teachers
   =============================== */
func (ctl *ClassSectionSubjectTeacherController) Create(c *fiber.Ctx) error {
	// ✅ ambil konteks masjid dari path/header/query/host/token
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	// ✅ pastikan caller adalah DKM/Admin masjid tsb (gunakan helper baru)
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
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

	return ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) SECTION harus ada & tenant cocok
		var sec modelClassSection.ClassSectionModel
		if err := tx.
			Where("class_sections_id = ? AND class_sections_masjid_id = ? AND class_sections_deleted_at IS NULL",
				req.ClassSectionSubjectTeachersSectionID, masjidID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek section")
		}

		// 2) CLASS_SUBJECTS harus ada; tenant sama; dan class_id cocok dgn section.class_id
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
			Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
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

/* ===============================
   UPDATE (partial)
   PUT /admin/:masjid_id/class-section-subject-teachers/:id
   =============================== */
func (ctl *ClassSectionSubjectTeacherController) Update(c *fiber.Ctx) error {
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
	if err := ctl.DB.WithContext(c.Context()).
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

		// cek section milik tenant + ambil class_id dari section
		var sec modelClassSection.ClassSectionModel
		if err := ctl.DB.WithContext(c.Context()).
			Where("class_sections_id = ? AND class_sections_masjid_id = ? AND class_sections_deleted_at IS NULL", sectionID, masjidID).
			First(&sec).Error; err != nil {
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
		if err := ctl.DB.WithContext(c.Context()).
			Table("class_subjects").
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
		// ✅ perbaikan: bandingkan ke sec.ClassSectionsClassID (bukan sectionID)
		if cs.ClassSubjectsClassID != sec.ClassSectionsClassID {
			return helper.JsonError(c, fiber.StatusBadRequest, "Class mismatch: class_subjects.class_id != class_sections.class_id")
		}
	}

	// partial update via DTO
	req.Apply(&row)

	if err := ctl.DB.WithContext(c.Context()).Save(&row).Error; err != nil {
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

/* ===============================
   DELETE (soft delete)
   DELETE /admin/:masjid_id/class-section-subject-teachers/:id
   =============================== */
func (ctl *ClassSectionSubjectTeacherController) Delete(c *fiber.Ctx) error {
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
		return fiber.NewError(http.StatusBadRequest, "ID tidak valid")
	}

	var row modelCSST.ClassSectionSubjectTeacherModel
	if err := ctl.DB.WithContext(c.Context()).
		First(&row, "class_section_subject_teachers_id = ?", id).Error; err != nil {
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

	if err := ctl.DB.WithContext(c.Context()).
		Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teachers_id = ?", id).
		Update("class_section_subject_teachers_deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Penugasan guru berhasil dihapus", fiber.Map{"id": id})
}
