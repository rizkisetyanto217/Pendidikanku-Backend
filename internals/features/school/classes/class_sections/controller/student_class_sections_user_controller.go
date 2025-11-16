package controller

import (
	dto "schoolku_backend/internals/features/school/classes/class_sections/dto"
	model "schoolku_backend/internals/features/school/classes/class_sections/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ========== LIST ALL (by school, untuk staff/admin) ==========
// Role: Staff (teacher | dkm | admin | bendahara)
// GET /api/a/:school_id/student-class-sections/list
func (ctl *StudentClassSectionController) ListAll(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c) // gunakan helper export
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
			OR LOWER(student_class_section_section_slug_snapshot) LIKE ?
		`, s, s)
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
// Role: Student (atau user yang profilnya terhubung ke school_student di sekolah itu).
// GET /api/u/:school_id/student-class-sections/my
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

	// opsional: tetap boleh cek member (bisa diaktifkan kalau mau strict)
	if e := helperAuth.EnsureMemberSchool(c, schoolID); e != nil {
		// return e
		_ = e
	}

	// --- mulai TX ---
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

	// Ambil/resolve school_student_id
	var schoolStudentID uuid.UUID
	if raw := strings.TrimSpace(c.Query("school_student_id", "")); raw != "" {
		msID, e := uuid.Parse(raw)
		if e != nil || msID == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid")
		}
		// validasi kepemilikan (tenant + profile)
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
		// auto-resolve / auto-create student untuk user ini di school tersebut
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
		"school_student_id": schoolStudentID,
		"items":             out,
		"meta": fiber.Map{
			"page":  page,
			"size":  size,
			"total": total,
		},
	})
}
