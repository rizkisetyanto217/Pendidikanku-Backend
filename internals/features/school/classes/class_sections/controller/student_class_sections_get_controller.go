// file: internals/features/school/classes/class_sections/controller/student_class_section_controller.go
package controller

import (
	"strings"

	dto "schoolku_backend/internals/features/school/classes/class_sections/dto"
	model "schoolku_backend/internals/features/school/classes/class_sections/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/u/student-class-sections/list
// ?school_student_id=me|<uuid,uuid2,...>
// ?section_id=<uuid,uuid2,...>
// ?status=active|inactive|completed
// ?q=...
// ?page=1&size=20
func (ctl *StudentClassSectionController) List(c *fiber.Ctx) error {
	// 1) school dari TOKEN
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// 2) cek apakah caller staff (guru/dkm/admin/bendahara)
	isStaff := (helperAuth.EnsureStaffSchool(c, schoolID) == nil)

	// 3) ambil user_id dari token (perlu untuk "me")
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	tx := ctl.DB.WithContext(c.Context())

	// ----------------- RESOLVE school_student_id -----------------
	rawSchoolStudent := strings.TrimSpace(c.Query("school_student_id"))

	var schoolStudentIDs []uuid.UUID

	if rawSchoolStudent == "" {
		// kalau kosong:
		// - staff  → boleh lihat semua (tanpa filter)
		// - non-staff → auto "me"
		if !isStaff {
			rawSchoolStudent = "me"
		}
	}

	if rawSchoolStudent == "me" {
		// ==== MODE "ME" → resolve dari user_id ====
		usersProfileID, err := getUsersProfileID(tx, userID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
		}

		msID, err := getOrCreateSchoolStudentWithSnapshots(c.Context(), tx, schoolID, usersProfileID, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mendapatkan status student")
		}
		schoolStudentIDs = []uuid.UUID{msID}

	} else if rawSchoolStudent != "" {
		// ==== MODE FILTER EXPLICIT UUID LIST ====
		ids, err := parseUUIDList(rawSchoolStudent)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid: "+err.Error())
		}
		schoolStudentIDs = ids

		// kalau bukan staff, pastikan id-id ini memang milik dia
		if !isStaff && len(ids) > 0 {
			usersProfileID, err := getUsersProfileID(tx, userID)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
			}

			var cnt int64
			if err := tx.Table("school_students").
				Where(`
					school_student_id IN ?
					AND school_student_school_id = ?
					AND school_student_user_profile_id = ?
					AND school_student_deleted_at IS NULL
				`, ids, schoolID, usersProfileID).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi school_student")
			}
			if cnt != int64(len(ids)) {
				return helper.JsonError(c, fiber.StatusForbidden, "Beberapa school_student_id bukan milik Anda / beda tenant")
			}
		}
	}

	// ----------------- FILTER SECTION & STATUS & SEARCH -----------------
	var (
		secIDs     []uuid.UUID
		status     string
		searchTerm = strings.TrimSpace(c.Query("q"))
	)

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

	// pagination
	page, size := getPageSize(c)
	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}

	// BASE QUERY
	q := tx.Model(&model.StudentClassSection{}).
		Where(`
			student_class_section_school_id = ?
			AND student_class_section_deleted_at IS NULL
		`, schoolID)

	if len(schoolStudentIDs) > 0 {
		q = q.Where("student_class_section_school_student_id IN ?", schoolStudentIDs)
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
			LOWER(COALESCE(student_class_section_user_profile_name_snapshot, '')) LIKE ?
			OR LOWER(student_class_section_section_slug_snapshot) LIKE ?
		`, s, s)
	}

	// COUNT
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// DATA
	var rows []model.StudentClassSection
	if err := q.
		Order("student_class_section_created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// MAP ke DTO
	out := make([]dto.StudentClassSectionResp, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModel(&rows[i]))
	}

	// pagination style standar
	pagination := helper.BuildPaginationFromOffset(total, offset, size)

	// ⬅️ sekarang tanpa "items"/"meta", langsung array di "data"
	return helper.JsonList(c, "OK", out, pagination)
}
