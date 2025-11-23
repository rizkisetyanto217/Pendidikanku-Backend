// file: internals/features/school/classes/class_sections/controller/student_class_section_controller.go
package controller

import (
	"strings"
	"time"

	dto "schoolku_backend/internals/features/school/classes/class_sections/dto"
	classSectionModel "schoolku_backend/internals/features/school/classes/class_sections/model"
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
// ?include=class_sections
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

	// ----------------- PARSE INCLUDE -----------------
	includeRaw := strings.TrimSpace(c.Query("include"))
	var includeClassSections bool
	if includeRaw != "" {
		parts := strings.Split(includeRaw, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) == "class_sections" {
				includeClassSections = true
				break
			}
		}
	}

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
	q := tx.Model(&classSectionModel.StudentClassSection{}).
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
	var rows []classSectionModel.StudentClassSection
	if err := q.
		Order("student_class_section_created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// pagination style standar
	pagination := helper.BuildPaginationFromOffset(total, offset, size)

	// =====================================================================
	//  MODE TANPA INCLUDE → balikkan seperti sebelumnya (backward compatible)
	// =====================================================================
	if !includeClassSections {
		out := make([]dto.StudentClassSectionResp, 0, len(rows))
		for i := range rows {
			out = append(out, dto.FromModel(&rows[i]))
		}
		return helper.JsonList(c, "OK", out, pagination)
	}

	// =====================================================================
	//  MODE include=class_sections
	//  - Ambil semua section_id
	//  - Query ke tabel class_sections
	//  - Embed ke tiap item di field "class_section"
	// =====================================================================

	// 1) Kumpulkan section_id unik
	secIDSet := make(map[uuid.UUID]struct{})
	for i := range rows {
		secIDSet[rows[i].StudentClassSectionSectionID] = struct{}{}
	}

	secIDs = make([]uuid.UUID, 0, len(secIDSet))
	for id := range secIDSet {
		secIDs = append(secIDs, id)
	}

	// 2) Query ke tabel class_sections
	type ClassSectionIncluded struct {
		ID            uuid.UUID  `json:"class_section_id"`
		SchoolID      uuid.UUID  `json:"class_section_school_id"`
		ClassID       *uuid.UUID `json:"class_section_class_id,omitempty"`
		Slug          string     `json:"class_section_slug"`
		Name          string     `json:"class_section_name"`
		Code          *string    `json:"class_section_code,omitempty"`
		Schedule      *string    `json:"class_section_schedule,omitempty"`
		Capacity      *int       `json:"class_section_capacity,omitempty"`
		TotalStudents int        `json:"class_section_total_students"`
		GroupURL      *string    `json:"class_section_group_url,omitempty"`
		IsActive      bool       `json:"class_section_is_active"`

		// 🔹 Tambahan: image fields
		ImageURL                *string    `json:"class_section_image_url,omitempty"`
		ImageObjectKey          *string    `json:"class_section_image_object_key,omitempty"`
		ImageURLOld             *string    `json:"class_section_image_url_old,omitempty"`
		ImageObjectKeyOld       *string    `json:"class_section_image_object_key_old,omitempty"`
		ImageDeletePendingUntil *time.Time `json:"class_section_image_delete_pending_until,omitempty"`

		ClassNameSnapshot        *string    `json:"class_section_class_name_snapshot,omitempty"`
		ClassSlugSnapshot        *string    `json:"class_section_class_slug_snapshot,omitempty"`
		ClassParentID            *uuid.UUID `json:"class_section_class_parent_id,omitempty"`
		ClassParentNameSnapshot  *string    `json:"class_section_class_parent_name_snapshot,omitempty"`
		ClassParentSlugSnapshot  *string    `json:"class_section_class_parent_slug_snapshot,omitempty"`
		ClassParentLevelSnapshot *int16     `json:"class_section_class_parent_level_snapshot,omitempty"`
		SchoolTeacherID          *uuid.UUID `json:"class_section_school_teacher_id,omitempty"`
		ClassRoomID              *uuid.UUID `json:"class_section_class_room_id,omitempty"`
		AcademicTermID           *uuid.UUID `json:"class_section_academic_term_id,omitempty"`
	}

	classSectionMap := make(map[uuid.UUID]*ClassSectionIncluded)

	if len(secIDs) > 0 {
		var secRows []classSectionModel.ClassSectionModel
		if err := tx.
			Where(`
                class_section_id IN ?
                AND class_section_school_id = ?
                AND class_section_deleted_at IS NULL
            `, secIDs, schoolID).
			Find(&secRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data class sections")
		}

		for i := range secRows {
			cs := secRows[i]

			item := &ClassSectionIncluded{
				ID:            cs.ClassSectionID,
				SchoolID:      cs.ClassSectionSchoolID,
				ClassID:       cs.ClassSectionClassID,
				Slug:          cs.ClassSectionSlug,
				Name:          cs.ClassSectionName,
				Code:          cs.ClassSectionCode,
				Schedule:      cs.ClassSectionSchedule,
				Capacity:      cs.ClassSectionCapacity,
				TotalStudents: cs.ClassSectionTotalStudents,
				GroupURL:      cs.ClassSectionGroupURL,
				IsActive:      cs.ClassSectionIsActive,

				ClassNameSnapshot:        cs.ClassSectionClassNameSnapshot,
				ClassSlugSnapshot:        cs.ClassSectionClassSlugSnapshot,
				ClassParentID:            cs.ClassSectionClassParentID,
				ClassParentNameSnapshot:  cs.ClassSectionClassParentNameSnapshot,
				ClassParentSlugSnapshot:  cs.ClassSectionClassParentSlugSnapshot,
				ClassParentLevelSnapshot: cs.ClassSectionClassParentLevelSnapshot,
				SchoolTeacherID:          cs.ClassSectionSchoolTeacherID,
				ClassRoomID:              cs.ClassSectionClassRoomID,
				AcademicTermID:           cs.ClassSectionAcademicTermID,
			}

			// 🔹 Set field image di sini
			item.ImageURL = cs.ClassSectionImageURL
			item.ImageObjectKey = cs.ClassSectionImageObjectKey
			item.ImageURLOld = cs.ClassSectionImageURLOld
			item.ImageObjectKeyOld = cs.ClassSectionImageObjectKeyOld
			item.ImageDeletePendingUntil = cs.ClassSectionImageDeletePendingUntil

			classSectionMap[cs.ClassSectionID] = item
		}

	}

	// 3) Bentuk DTO nested
	type StudentClassSectionWithClassSectionResp struct {
		dto.StudentClassSectionResp
		ClassSection *ClassSectionIncluded `json:"class_section,omitempty"`
	}

	out := make([]StudentClassSectionWithClassSectionResp, 0, len(rows))
	for i := range rows {
		base := dto.FromModel(&rows[i])

		var included *ClassSectionIncluded
		if cs, ok := classSectionMap[rows[i].StudentClassSectionSectionID]; ok {
			included = cs
		}

		out = append(out, StudentClassSectionWithClassSectionResp{
			StudentClassSectionResp: base,
			ClassSection:            included,
		})
	}

	return helper.JsonList(c, "OK", out, pagination)

}
