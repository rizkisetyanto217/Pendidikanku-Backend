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
// ========== LIST MINE (auto-resolve masjid_student) ==========
func (ctl *UserClassSectionController) ListMine(c *fiber.Ctx) error {
	masjidID, err := parseMasjidIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// wajib login
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	// opsional: tetap boleh cek member (DKM/teacher/student). Biarkan kalau kamu mau harden.
	if e := helperAuth.EnsureMemberMasjid(c, masjidID); e != nil {
		// Jika kamu tidak ingin memaksa membership strict, comment return-nya dan lanjutkan.
		// return e
	}

	// --- mulai TX (perlu kalau nanti create masjid_student) ---
	tx := ctl.DB.WithContext(c.Context()).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Resolve users_profile_id dari user
	usersProfileID, err := getUsersProfileID(tx, userID)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
	}

	// Ambil masjid_student_id:
	// - jika query param ada, validasi milik user & tenant
	// - kalau kosong, buat/ambil otomatis berdasarkan profile
	var masjidStudentID uuid.UUID

	if raw := strings.TrimSpace(c.Query("masjid_student_id", "")); raw != "" {
		msID, e := uuid.Parse(raw)
		if e != nil || msID == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "masjid_student_id tidak valid")
		}
		// validasi kepemilikan (tenant + user profile)
		var cnt int64
		if err := tx.Table("masjid_students").
			Where(`
				masjid_student_id = ?
				AND masjid_student_masjid_id = ?
				AND masjid_student_user_profile_id = ?
				AND masjid_student_deleted_at IS NULL
			`, msID, masjidID, usersProfileID).
			Count(&cnt).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi masjid_student")
		}
		if cnt == 0 {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "masjid_student_id bukan milik Anda / beda tenant")
		}
		masjidStudentID = msID
	} else {
		// auto resolve (dan buat jika belum ada)
		msID, e := getOrCreateMasjidStudentByProfile(tx, masjidID, usersProfileID)
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mendapatkan status student")
		}
		masjidStudentID = msID
	}

	// pagination
	page, size := getPageSize(c)

	// query data
	var total int64
	q := tx.Model(&model.UserClassSection{}).
		Where(`
			user_class_section_masjid_id = ?
			AND user_class_section_masjid_student_id = ?
			AND user_class_section_deleted_at IS NULL
		`, masjidID, masjidStudentID)

	if err := q.Count(&total).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var items []model.UserClassSection
	if err := q.
		Order("user_class_section_created_at DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&items).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
	}

	// mapping ke resp
	out := make([]dto.UserClassSectionResp, 0, len(items))
	for i := range items {
		out = append(out, dto.FromModel(&items[i]))
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"masjid_student_id": masjidStudentID, // biar klien tahu ID yang dipakai
		"items":             out,
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
