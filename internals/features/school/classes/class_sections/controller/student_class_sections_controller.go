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

	dto "schoolku_backend/internals/features/school/classes/class_sections/dto"
	model "schoolku_backend/internals/features/school/classes/class_sections/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
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
func (ctl *StudentClassSectionController) Create(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// contoh guard: anggota school
	// if e := helperAuth.EnsureMemberSchool(c, schoolID); e != nil { return e }

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

	m := req.ToModel() // *model.StudentClassSection
	now := time.Now()
	m.StudentClassSectionCreatedAt = now
	m.StudentClassSectionUpdatedAt = now

	// Safety: hard-guard tenant
	m.StudentClassSectionSchoolID = schoolID

	if err := ctl.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat student_class_section")
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": dto.FromModel(m),
	})
}

// ========== GET DETAIL ==========
func (ctl *StudentClassSectionController) GetDetail(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// anggota school saja boleh lihat
	if e := helperAuth.EnsureMemberSchool(c, schoolID); e != nil {
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

// ========== LIST ALL (by school, untuk staff/admin) ==========
// GET /api/a/:school_id/student-class-sections/list
func (ctl *StudentClassSectionController) ListAll(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c) // <-- use Exported name
	if err != nil {
		return err
	}

	// Guard: staff (teacher|dkm|admin|bendahara)
	if e := helperAuth.EnsureStaffSchool(c, schoolID); e != nil {
		return e
	}

	tx := ctl.DB.WithContext(c.Context())

	// --------- filters opsional ----------
	var (
		msIDs      []uuid.UUID
		secIDs     []uuid.UUID
		status     string
		searchTerm = strings.TrimSpace(c.Query("q"))
	)

	if raw := strings.TrimSpace(c.Query("school_student_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid: "+e.Error())
		}
		msIDs = ids
	}
	if raw := strings.TrimSpace(c.Query("section_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "section_id tidak valid: "+e.Error())
		}
		secIDs = ids
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		status = s
	}

	q := tx.Model(&model.StudentClassSection{}).
		Where(`
			student_class_section_school_id = ?
			AND student_class_section_deleted_at IS NULL
		`, schoolID)

	if len(msIDs) > 0 {
		q = q.Where("student_class_section_school_student_id IN ?", msIDs)
	}
	if len(secIDs) > 0 {
		q = q.Where("student_class_section_section_id IN ?", secIDs)
	}
	if status != "" {
		q = q.Where("student_class_section_status = ?", status)
	}
	if searchTerm != "" {
		s := "%" + strings.ToLower(searchTerm) + "%"
		q = q.Where(`
			LOWER(COALESCE(student_class_section_user_profile_name_snapshot,'')) LIKE ?
			OR LOWER(COALESCE(student_class_section_fee_snapshot->>'section_code','')) LIKE ?
			OR LOWER(COALESCE(student_class_section_fee_snapshot->>'section_slug','')) LIKE ?
		`, s, s, s)
	}

	page, size := getPageSize(c)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var rows []model.StudentClassSection
	if err := q.
		Order("student_class_section_created_at DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]dto.StudentClassSectionResp, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModel(&rows[i]))
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

// ========== LIST MINE (auto-resolve school_student) ==========
func (ctl *StudentClassSectionController) ListMine(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// wajib login
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	// opsional: tetap boleh cek member (DKM/teacher/student). Biarkan kalau kamu mau harden.
	if e := helperAuth.EnsureMemberSchool(c, schoolID); e != nil {
		// Jika kamu tidak ingin memaksa membership strict, comment return-nya dan lanjutkan.
		// return e
	}

	// --- mulai TX (perlu kalau nanti create school_student) ---
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

	// Ambil school_student_id:
	// - jika query param ada, validasi milik user & tenant
	// - kalau kosong, buat/ambil otomatis berdasarkan profile
	var schoolStudentID uuid.UUID

	if raw := strings.TrimSpace(c.Query("school_student_id", "")); raw != "" {
		msID, e := uuid.Parse(raw)
		if e != nil || msID == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid")
		}
		// validasi kepemilikan (tenant + user profile)
		var cnt int64
		if err := tx.Table("school_students").
			Where(`
				school_student_id = ?
				AND school_student_school_id = ?
				AND school_student_user_profile_id = ?
				AND school_student_deleted_at IS NULL
			`, msID, schoolID, usersProfileID).
			Count(&cnt).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi school_student")
		}
		if cnt == 0 {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "school_student_id bukan milik Anda / beda tenant")
		}
		schoolStudentID = msID
	} else {
		// auto resolve (dan buat jika belum ada)
		msID, e := getOrCreateSchoolStudentWithSnapshots(c.Context(), tx, schoolID, usersProfileID, nil)
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mendapatkan status student")
		}
		schoolStudentID = msID
	}

	// pagination
	page, size := getPageSize(c)

	// query data
	var total int64
	q := tx.Model(&model.StudentClassSection{}).
		Where(`
			student_class_section_school_id = ?
			AND student_class_section_school_student_id = ?
			AND student_class_section_deleted_at IS NULL
		`, schoolID, schoolStudentID)

	if err := q.Count(&total).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var items []model.StudentClassSection
	if err := q.
		Order("student_class_section_created_at DESC").
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
	out := make([]dto.StudentClassSectionResp, 0, len(items))
	for i := range items {
		out = append(out, dto.FromModel(&items[i]))
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"school_student_id": schoolStudentID, // biar klien tahu ID yang dipakai
		"items":             out,
		"meta": fiber.Map{
			"page":  page,
			"size":  size,
			"total": total,
		},
	})
}

// ========== PATCH ==========
func (ctl *StudentClassSectionController) Patch(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// hanya anggota (atau kalau mau lebih ketat: EnsureDKMOrTeacherSchool)
	if e := helperAuth.EnsureMemberSchool(c, schoolID); e != nil {
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
func (ctl *StudentClassSectionController) Delete(c *fiber.Ctx) error {
	schoolID, err := parseSchoolIDFromPath(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// hanya anggota (atau set ke EnsureDKMOrTeacherSchool jika perlu role lebih tinggi)
	if e := helperAuth.EnsureMemberSchool(c, schoolID); e != nil {
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
