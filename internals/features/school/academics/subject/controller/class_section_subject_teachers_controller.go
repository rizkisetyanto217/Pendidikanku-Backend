// internals/features/lembaga/class_section_subject_teachers/controller/csst_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	modelMasjidTeacher "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"
	modelClassSection "masjidku_backend/internals/features/school/classes/class_sections/model"

	// === pakai model & dto CSST terbaru (sectionsubjectteachers) ===
	dto "masjidku_backend/internals/features/school/academics/subject/dto"
	modelCSST "masjidku_backend/internals/features/school/academics/subject/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type ClassSectionSubjectTeacherController struct {
	DB *gorm.DB
}

func NewClassSectionSubjectTeacherController(db *gorm.DB) *ClassSectionSubjectTeacherController {
	return &ClassSectionSubjectTeacherController{DB: db}
}

/*
===============================

	CREATE (admin/DKM via masjid context)
	POST /admin/:masjid_id/class-section-subject-teachers
	/admin/:masjid_slug/class-section-subject-teachers
	===============================
*/
func (ctl *ClassSectionSubjectTeacherController) Create(c *fiber.Ctx) error {
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
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
		// 1) SECTION exists & same tenant
		var sec modelClassSection.ClassSectionModel
		if err := tx.
			Where("class_sections_id = ? AND class_sections_masjid_id = ? AND class_sections_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherSectionID, masjidID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek section")
		}

		// 2) CLASS_SUBJECTS exists, same tenant, matches section.class_id
		var cs struct {
			ClassSubjectsID       uuid.UUID
			ClassSubjectsMasjidID uuid.UUID
			ClassSubjectsClassID  uuid.UUID
		}
		if err := tx.Table("class_subjects").
			Select("class_subjects_id, class_subjects_masjid_id, class_subjects_class_id").
			Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherClassSubjectID).
			Take(&cs).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "class_subjects tidak ditemukan / sudah dihapus")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek class_subjects")
		}
		if cs.ClassSubjectsMasjidID != masjidID {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid mismatch: class_subjects milik masjid lain")
		}
		if cs.ClassSubjectsClassID != sec.ClassSectionClassID {
			return helper.JsonError(c, fiber.StatusBadRequest, "Class mismatch: class_subjects.class_id != class_sections.class_id")
		}

		// 3) TEACHER exists & same tenant (cek saja; tak ambil nama)
		if err := tx.
			Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherTeacherID, masjidID).
			First(&modelMasjidTeacher.MasjidTeacherModel{}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Guru tidak ditemukan / bukan guru masjid ini")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek guru")
		}

		// 4) Build row
		row := req.ToModel()
		row.ClassSectionSubjectTeacherMasjidID = masjidID
		if !row.ClassSectionSubjectTeacherIsActive {
			row.ClassSectionSubjectTeacherIsActive = true
		}

		// ========== SLUG ==========
		if row.ClassSectionSubjectTeacherSlug != nil {
			s := helper.Slugify(*row.ClassSectionSubjectTeacherSlug, 160)
			row.ClassSectionSubjectTeacherSlug = &s
		}
		if row.ClassSectionSubjectTeacherSlug == nil || strings.TrimSpace(*row.ClassSectionSubjectTeacherSlug) == "" {
			var sectionName, subjectName string

			_ = tx.Table("class_sections").
				Select("class_sections_name").
				Where("class_sections_id = ? AND class_sections_masjid_id = ?", req.ClassSectionSubjectTeacherSectionID, masjidID).
				Scan(&sectionName).Error

			_ = tx.Table("class_subjects AS cs").
				Select("s.subjects_name").
				Joins("JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id AND s.subjects_deleted_at IS NULL").
				Where("cs.class_subjects_id = ? AND cs.class_subjects_masjid_id = ? AND cs.class_subjects_deleted_at IS NULL",
					req.ClassSectionSubjectTeacherClassSubjectID, masjidID).
				Scan(&subjectName).Error

			parts := make([]string, 0, 2)
			if strings.TrimSpace(sectionName) != "" {
				parts = append(parts, sectionName)
			}
			if strings.TrimSpace(subjectName) != "" {
				parts = append(parts, subjectName)
			}

			base := "csst"
			if len(parts) > 0 {
				base = strings.Join(parts, " ")
			} else {
				base = fmt.Sprintf("csst-%s-%s-%s",
					strings.Split(req.ClassSectionSubjectTeacherSectionID.String(), "-")[0],
					strings.Split(req.ClassSectionSubjectTeacherClassSubjectID.String(), "-")[0],
					strings.Split(req.ClassSectionSubjectTeacherTeacherID.String(), "-")[0],
				)
			}
			base = helper.Slugify(base, 160)

			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"class_section_subject_teachers", "class_section_subject_teacher_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(`
						class_section_subject_teacher_masjid_id = ?
						AND class_section_subject_teacher_deleted_at IS NULL
					`, masjidID)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			row.ClassSectionSubjectTeacherSlug = &uniqueSlug
		} else {
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"class_section_subject_teachers", "class_section_subject_teacher_slug",
				*row.ClassSectionSubjectTeacherSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(`
						class_section_subject_teacher_masjid_id = ?
						AND class_section_subject_teacher_deleted_at IS NULL
					`, masjidID)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			row.ClassSectionSubjectTeacherSlug = &uniqueSlug
		}
		// ========== END SLUG ==========

		// 5) INSERT
		if err := tx.Create(&row).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csst_one_active_per_section_subject_alive"),
				strings.Contains(msg, "uq_csst_unique_alive"),
				strings.Contains(msg, "duplicate"),
				strings.Contains(msg, "unique"):
				return helper.JsonError(c, fiber.StatusConflict, "Penugasan atau slug sudah terdaftar (duplikat).")
			case strings.Contains(msg, "uq_csst_slug_per_tenant_alive"):
				return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
			case strings.Contains(msg, "23503"), strings.Contains(msg, "foreign key"):
				switch {
				case strings.Contains(msg, "class_sections"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_subjects"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECTS): tidak ditemukan / beda tenant")
				case strings.Contains(msg, "masjid_teachers"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_rooms"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (ROOM): ruangan tidak ditemukan / beda tenant")
				default:
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/class_subjects/guru/room valid")
				}
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Insert gagal: "+err.Error())
		}

		return helper.JsonCreated(c, "Penugasan guru berhasil dibuat", dto.FromClassSectionSubjectTeacherModel(row))
	})
}

/*
===============================

	UPDATE (partial)
	PUT /admin/:masjid_id/class-section-subject-teachers/:id

===============================
*/
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

	return ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Ambil row
		var row modelCSST.ClassSectionSubjectTeacherModel
		if err := tx.
			Where("class_section_subject_teacher_id = ? AND class_section_subject_teacher_deleted_at IS NULL", id).
			First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		// tenant guard
		if row.ClassSectionSubjectTeacherMasjidID != masjidID {
			return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
		}

		// (opsional) precheck konsistensi jika section_id / class_subject_id berubah
		if req.ClassSectionSubjectTeacherSectionID != nil || req.ClassSectionSubjectTeacherClassSubjectID != nil {
			sectionID := row.ClassSectionSubjectTeacherSectionID
			if req.ClassSectionSubjectTeacherSectionID != nil {
				sectionID = *req.ClassSectionSubjectTeacherSectionID
			}
			classSubjectID := row.ClassSectionSubjectTeacherClassSubjectID
			if req.ClassSectionSubjectTeacherClassSubjectID != nil {
				classSubjectID = *req.ClassSectionSubjectTeacherClassSubjectID
			}

			// cek section milik tenant + ambil class_id dari section
			var sec modelClassSection.ClassSectionModel
			if err := tx.
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
			if err := tx.
				Table("class_subjects").
				Select("class_subjects_masjid_id, class_subjects_class_id").
				Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL", classSubjectID).
				Take(&cs).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusBadRequest, "class_subjects tidak ditemukan / sudah dihapus")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek class_subjects")
			}
			if cs.ClassSubjectsMasjidID != masjidID {
				return helper.JsonError(c, fiber.StatusBadRequest, "Masjid mismatch: class_subjects milik masjid lain")
			}
			if cs.ClassSubjectsClassID != sec.ClassSectionClassID {
				return helper.JsonError(c, fiber.StatusBadRequest, "Class mismatch: class_subjects.class_id != class_sections.class_id")
			}
		}

		// Catat apakah ada perubahan yang memengaruhi slug
		sectionChanged := req.ClassSectionSubjectTeacherSectionID != nil &&
			*req.ClassSectionSubjectTeacherSectionID != row.ClassSectionSubjectTeacherSectionID
		csChanged := req.ClassSectionSubjectTeacherClassSubjectID != nil &&
			*req.ClassSectionSubjectTeacherClassSubjectID != row.ClassSectionSubjectTeacherClassSubjectID
		teacherChanged := req.ClassSectionSubjectTeacherTeacherID != nil &&
			*req.ClassSectionSubjectTeacherTeacherID != row.ClassSectionSubjectTeacherTeacherID

		// Apply perubahan lain (belum sentuh slug)
		req.Apply(&row)

		// ===== SLUG handling =====
		// Normalisasi slug jika user mengirimkan
		if req.ClassSectionSubjectTeacherSlug != nil {
			if s := strings.TrimSpace(*req.ClassSectionSubjectTeacherSlug); s != "" {
				norm := helper.Slugify(s, 160)
				row.ClassSectionSubjectTeacherSlug = &norm
			} else {
				// "" → nil
				row.ClassSectionSubjectTeacherSlug = nil
			}
		}

		// Perlu generate/cek unik jika:
		// - user set slug baru (di-normalisasi di atas), atau
		// - slug masih kosong dan ada perubahan section/class_subject/teacher (atau awalnya kosong)
		needEnsureUnique := false
		baseSlug := ""

		if req.ClassSectionSubjectTeacherSlug != nil {
			needEnsureUnique = true
			if row.ClassSectionSubjectTeacherSlug != nil {
				baseSlug = *row.ClassSectionSubjectTeacherSlug
			}
		} else if row.ClassSectionSubjectTeacherSlug == nil || strings.TrimSpace(ptrStr(row.ClassSectionSubjectTeacherSlug)) == "" {
			if sectionChanged || csChanged || teacherChanged || row.ClassSectionSubjectTeacherSlug == nil {
				needEnsureUnique = true

				// Rakitan base dari entity terkait (best-effort)
				var sectionName, className, subjectName, teacherName string

				_ = tx.Table("class_sections").
					Select("class_sections_name").
					Where("class_sections_id = ? AND class_sections_masjid_id = ?", row.ClassSectionSubjectTeacherSectionID, masjidID).
					Scan(&sectionName).Error

				_ = tx.Table("class_subjects cs").
					Select("c.classes_name, s.subjects_name").
					Joins("JOIN classes c ON c.classes_id = cs.class_subjects_class_id AND c.classes_deleted_at IS NULL").
					Joins("JOIN subjects s ON s.subjects_id = cs.class_subjects_subject_id AND s.subjects_deleted_at IS NULL").
					Where("cs.class_subjects_id = ? AND cs.class_subjects_masjid_id = ?", row.ClassSectionSubjectTeacherClassSubjectID, masjidID).
					Scan(&struct {
						ClassesName  *string
						SubjectsName *string
					}{&className, &subjectName}).Error

				_ = tx.Table("masjid_teachers").
					Select("masjid_teacher_name").
					Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ?", row.ClassSectionSubjectTeacherTeacherID, masjidID).
					Scan(&teacherName).Error

				parts := []string{}
				if strings.TrimSpace(className) != "" {
					parts = append(parts, className)
				}
				if strings.TrimSpace(sectionName) != "" {
					parts = append(parts, sectionName)
				}
				if strings.TrimSpace(subjectName) != "" {
					parts = append(parts, subjectName)
				}
				if strings.TrimSpace(teacherName) != "" {
					parts = append(parts, teacherName)
				}

				if len(parts) == 0 {
					baseSlug = fmt.Sprintf("csst-%s-%s-%s",
						strings.Split(row.ClassSectionSubjectTeacherSectionID.String(), "-")[0],
						strings.Split(row.ClassSectionSubjectTeacherClassSubjectID.String(), "-")[0],
						strings.Split(row.ClassSectionSubjectTeacherTeacherID.String(), "-")[0],
					)
				} else {
					baseSlug = strings.Join(parts, " ")
				}
				baseSlug = helper.Slugify(baseSlug, 160)
			}
		}

		if needEnsureUnique {
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				tx,
				"class_section_subject_teachers",
				"class_section_subject_teacher_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					// Unik per tenant, soft-delete aware, EXCLUDE diri sendiri
					return q.Where(`
						class_section_subject_teacher_masjid_id = ?
						AND class_section_subject_teacher_deleted_at IS NULL
						AND class_section_subject_teacher_id <> ?
					`, masjidID, row.ClassSectionSubjectTeacherID)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, http.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			// Jika baseSlug kosong (mis. user set slug "" → nil), uniqueSlug akan dibuat dari fallback di helper.
			if strings.TrimSpace(uniqueSlug) != "" {
				row.ClassSectionSubjectTeacherSlug = &uniqueSlug
			} else {
				row.ClassSectionSubjectTeacherSlug = nil
			}
		}
		// ===== END SLUG =====

		// Persist
		if err := tx.Save(&row).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "uq_csst_one_active_per_section_subject_alive") ||
				strings.Contains(msg, "uq_csst_unique_alive") ||
				strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				// bisa karena duplikat kombinasi assignment, atau slug bentrok
				// cek slug index spesifik
				if strings.Contains(msg, "uq_csst_slug_per_tenant_alive") {
					return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
				}
				return helper.JsonError(c, fiber.StatusConflict, "Penugasan guru untuk class_subjects ini sudah aktif (duplikat).")
			}
			if strings.Contains(msg, "sqlstate 23503") || strings.Contains(msg, "foreign key") {
				switch {
				case strings.Contains(msg, "class_sections"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_subjects"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECTS): tidak ditemukan / beda tenant")
				case strings.Contains(msg, "masjid_teachers"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_rooms"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (ROOM): ruangan tidak ditemukan / beda tenant")
				default:
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/class_subjects/guru/room valid")
				}
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		return helper.JsonUpdated(c, "Penugasan guru berhasil diperbarui", dto.FromClassSectionSubjectTeacherModel(row))
	})
}

/*
===============================

	DELETE (soft delete)
	DELETE /admin/:masjid_id/class-section-subject-teachers/:id

===============================
*/
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
		First(&row, "class_section_subject_teacher_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	if row.ClassSectionSubjectTeacherMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	if row.ClassSectionSubjectTeacherDeletedAt.Valid {
		// idempotent
		return helper.JsonDeleted(c, "Sudah terhapus", fiber.Map{"id": id})
	}

	if err := ctl.DB.WithContext(c.Context()).
		Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teacher_id = ?", id).
		Update("class_section_subject_teacher_deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Penugasan guru berhasil dihapus", fiber.Map{"id": id})
}
