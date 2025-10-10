// internals/features/school/classes/class_sections/controller/user_class_section_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/classes/class_sections/dto"
	model "masjidku_backend/internals/features/school/classes/class_sections/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type UserClassSectionController struct {
	DB *gorm.DB
}

func NewUserClassSectionController(db *gorm.DB) *UserClassSectionController {
	return &UserClassSectionController{DB: db}
}

// ---- helpers ----
func parseMasjidIDFromPath(c *fiber.Ctx) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Params("masjid_id"))
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id path tidak valid")
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
func (ctl *UserClassSectionController) Create(c *fiber.Ctx) error {
	masjidID, err := parseMasjidIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// contoh guard: anggota masjid
	// if e := helperAuth.EnsureMemberMasjid(c, masjidID); e != nil { return e }

	var req dto.UserClassSectionCreateReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	req.Normalize()

	// Paksa tenant dari path (jangan percaya payload)
	req.UserClassSectionMasjidID = masjidID

	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel() // *model.UserClassSection
	now := time.Now()
	m.UserClassSectionCreatedAt = now
	m.UserClassSectionUpdatedAt = now

	// Safety: hard-guard tenant
	m.UserClassSectionMasjidID = masjidID

	if err := ctl.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat user_class_section")
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": dto.FromModel(m),
	})
}

// ========== GET DETAIL ==========
func (ctl *UserClassSectionController) GetDetail(c *fiber.Ctx) error {
	masjidID, err := parseMasjidIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// anggota masjid saja boleh lihat
	if e := helperAuth.EnsureMemberMasjid(c, masjidID); e != nil {
		return e
	}

	rawID := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(rawID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UserClassSection
	if err := ctl.DB.
		Where("user_class_section_masjid_id = ? AND user_class_section_id = ? AND user_class_section_deleted_at IS NULL", masjidID, id).
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

// ========== LIST MINE (opsional) ==========
// ========== LIST MINE (opsional) ==========
func (ctl *UserClassSectionController) ListMine(c *fiber.Ctx) error {
	masjidID, err := parseMasjidIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	// guard: anggota masjid
	if e := helperAuth.EnsureMemberMasjid(c, masjidID); e != nil {
		return e
	}

	// === Penting ===
	// Skema kamu memakai MasjidStudent sebagai identitas "milikku".
	// Ambil dari query param: ?masjid_student_id=<uuid>
	msStr := strings.TrimSpace(c.Query("masjid_student_id", ""))
	if msStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_student_id wajib diisi pada query param")
	}
	masjidStudentID, err := uuid.Parse(msStr)
	if err != nil || masjidStudentID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_student_id tidak valid")
	}

	page, size := getPageSize(c)
	var items []model.UserClassSection
	var total int64

	q := ctl.DB.Model(&model.UserClassSection{}).
		Where(
			"user_class_section_masjid_id = ? AND user_class_section_masjid_student_id = ? AND user_class_section_deleted_at IS NULL",
			masjidID, masjidStudentID,
		)

	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}
	if err := q.
		Order("user_class_section_created_at DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// pakai RESP, bukan DTO
	out := make([]dto.UserClassSectionResp, 0, len(items))
	for i := range items {
		out = append(out, dto.FromModel(&items[i]))
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"items": out,
		"meta": fiber.Map{
			"page":  page,
			"size":  size,
			"total": total,
		},
	})
}

// ========== PATCH ==========
func (ctl *UserClassSectionController) Patch(c *fiber.Ctx) error {
	masjidID, err := parseMasjidIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// hanya anggota (atau kalau mau lebih ketat: EnsureDKMOrTeacherMasjid)
	if e := helperAuth.EnsureMemberMasjid(c, masjidID); e != nil {
		return e
	}

	rawID := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(rawID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UserClassSectionPatchReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.UserClassSection
	if err := ctl.DB.
		Where("user_class_section_masjid_id = ? AND user_class_section_id = ? AND user_class_section_deleted_at IS NULL", masjidID, id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	req.Apply(&m)
	m.UserClassSectionMasjidID = masjidID // hard-guard tenant
	m.UserClassSectionUpdatedAt = time.Now()

	if err := ctl.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan patch")
	}

	return helper.JsonOK(c, "Berhasil patch", fiber.Map{
		"item": dto.FromModel(&m),
	})
}

// ========== DELETE (soft) ==========
func (ctl *UserClassSectionController) Delete(c *fiber.Ctx) error {
	masjidID, err := parseMasjidIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// hanya anggota (atau set ke EnsureDKMOrTeacherMasjid jika perlu role lebih tinggi)
	if e := helperAuth.EnsureMemberMasjid(c, masjidID); e != nil {
		return e
	}

	rawID := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(rawID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UserClassSection
	if err := ctl.DB.
		Where("user_class_section_masjid_id = ? AND user_class_section_id = ? AND user_class_section_deleted_at IS NULL", masjidID, id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	now := time.Now()
	m.UserClassSectionDeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	m.UserClassSectionUpdatedAt = now

	if err := ctl.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonOK(c, "Berhasil hapus", fiber.Map{
		"item": dto.FromModel(&m),
	})
}
