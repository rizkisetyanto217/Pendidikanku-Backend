// internals/features/school/classes/class_sections/controller/student_class_section_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	model "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

type StudentClassSectionController struct {
	DB *gorm.DB
}

func NewStudentClassSectionController(db *gorm.DB) *StudentClassSectionController {
	return &StudentClassSectionController{DB: db}
}

// ---- helpers ----
func parseSchoolIDFromPath(c *fiber.Ctx) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Params("school_id"))
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id path tidak valid")
	}
	return id, nil
}

func getPageSize(c *fiber.Ctx) (page, size int) {
	page, _ = strconv.Atoi(c.Query("page", "1"))
	size, _ = strconv.Atoi(c.Query("size", "20"))
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 200 {
		size = 20
	}
	return
}

// ========== CREATE ==========
// Role: DKM / Guru / Admin (akademik)
// Endpoint admin, bikin relasi murid ↔ section (penempatan kelas).
func (ctl *StudentClassSectionController) Create(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Hanya DKM / Guru / Admin di sekolah ini yang boleh create
	if e := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); e != nil {
		return e
	}

	var req dto.StudentClassSectionCreateReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	// Paksa tenant dari path (jangan percaya payload)
	req.StudentClassSectionSchoolID = schoolID

	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ========= ensure slug snapshot dari class_sections (tenant-safe) =========
	// Jika client tidak kirim snapshot, kita ambil dari sumbernya.
	if req.StudentClassSectionSectionSlugSnapshot == nil {
		var row struct {
			Slug string `gorm:"column:class_section_slug"`
		}
		if err := ctl.DB.Table("class_sections").
			Select("class_section_slug").
			Where(`
				class_section_id = ?
				AND class_section_school_id = ?
				AND class_section_deleted_at IS NULL
			`, req.StudentClassSectionSectionID, schoolID).
			First(&row).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil slug section")
		}
		req.StudentClassSectionSectionSlugSnapshot = &row.Slug
	}

	m := req.ToModel() // *model.StudentClassSection

	now := time.Now()
	m.StudentClassSectionCreatedAt = now
	m.StudentClassSectionUpdatedAt = now
	m.StudentClassSectionSchoolID = schoolID // hard-guard tenant

	if err := ctl.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat student_class_section")
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": dto.FromModel(m),
	})
}

// ========== GET DETAIL ==========
// Role: Staff akademik (Guru / DKM / Admin / Bendahara)
// Detail satu row relasi murid-section. Murid lihat list via ListMine.
func (ctl *StudentClassSectionController) GetDetail(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Hanya staff yang boleh lihat detail arbitrary
	if e := helperAuth.EnsureStaffSchool(c, schoolID); e != nil {
		return e
	}

	rawID := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(rawID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.StudentClassSection
	if err := ctl.DB.
		Where("student_class_section_school_id = ? AND student_class_section_id = ? AND student_class_section_deleted_at IS NULL", schoolID, id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"item": dto.FromModel(&m),
	})
}


// ========== PATCH ==========
// Role: DKM / Guru / Admin (akademik)
// Mengubah status/kolom lain di relasi murid ↔ section.
func (ctl *StudentClassSectionController) Patch(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// hanya DKM / Guru / Admin di sekolah ini
	if e := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); e != nil {
		return e
	}

	rawID := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(rawID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.StudentClassSectionPatchReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.StudentClassSection
	if err := ctl.DB.
		Where("student_class_section_school_id = ? AND student_class_section_id = ? AND student_class_section_deleted_at IS NULL", schoolID, id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	req.Apply(&m)
	m.StudentClassSectionSchoolID = schoolID // hard-guard tenant
	m.StudentClassSectionUpdatedAt = time.Now()

	if err := ctl.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan patch")
	}

	return helper.JsonOK(c, "Berhasil patch", fiber.Map{
		"item": dto.FromModel(&m),
	})
}

// ========== DELETE (soft) ==========
// Role: DKM / Guru / Admin (akademik)
// Menghapus (soft) relasi murid ↔ section.
func (ctl *StudentClassSectionController) Delete(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// hanya DKM / Guru / Admin di sekolah ini
	if e := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); e != nil {
		return e
	}

	rawID := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(rawID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.StudentClassSection
	if err := ctl.DB.
		Where("student_class_section_school_id = ? AND student_class_section_id = ? AND student_class_section_deleted_at IS NULL", schoolID, id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	now := time.Now()
	m.StudentClassSectionDeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	m.StudentClassSectionUpdatedAt = now

	if err := ctl.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonOK(c, "Berhasil hapus", fiber.Map{
		"item": dto.FromModel(&m),
	})
}
