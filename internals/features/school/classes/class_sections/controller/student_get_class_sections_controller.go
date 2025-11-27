package controller

import (
	"strings"
	"time"

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	dto "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/u/student-class-sections/list
// ?school_student_id=me|<uuid,uuid2,...>
// ?section_id=<uuid,uuid2,...>        // alias lama
// ?class_section_id=<uuid,uuid2,...>  // alias baru
// ?status=active|inactive|completed
// ?q=...
// ?include=class_sections,csst
// ?view=compact|full
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

	// ----------------- PARSE VIEW -----------------
	view := strings.ToLower(strings.TrimSpace(c.Query("view"))) // "", "compact", "full"

	// ----------------- PARSE INCLUDE -----------------
	includeRaw := strings.TrimSpace(c.Query("include"))
	var includeClassSections bool
	var includeCSST bool

	if includeRaw != "" {
		parts := strings.Split(includeRaw, ",")
		for _, p := range parts {
			switch strings.TrimSpace(p) {
			case "class_sections":
				includeClassSections = true
			case "csst", "cssts", "class_section_subject_teachers":
				includeCSST = true
			}
		}
	}

	// kalau minta CSST, otomatis butuh class_sections juga
	if includeCSST {
		includeClassSections = true
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

	// 🔹 section_id (lama)
	if raw := strings.TrimSpace(c.Query("section_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "section_id tidak valid: "+e.Error())
		}
		secIDs = append(secIDs, ids...)
	}

	// 🔹 class_section_id (alias baru)
	if raw := strings.TrimSpace(c.Query("class_section_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_section_id tidak valid: "+e.Error())
		}
		secIDs = append(secIDs, ids...)
	}

	// (opsional) hilangkan duplikat secIDs
	if len(secIDs) > 1 {
		tmpSet := make(map[uuid.UUID]struct{}, len(secIDs))
		uniq := make([]uuid.UUID, 0, len(secIDs))
		for _, id := range secIDs {
			if _, ok := tmpSet[id]; !ok {
				tmpSet[id] = struct{}{}
				uniq = append(uniq, id)
			}
		}
		secIDs = uniq
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
			OR LOWER(COALESCE(student_class_section_student_code_snapshot, '')) LIKE ?
		`, s, s, s)
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
	//  MODE VIEW=COMPACT → selalu balikin compact DTO,
	//  abaikan include=class_sections,csst (biar ringan, konsisten kayak enrollment)
	// =====================================================================
	if view == "compact" {
		out := dto.FromModelsStudentClassSectionCompact(rows)
		return helper.JsonList(c, "OK", out, pagination)
	}

	// =====================================================================
	//  MODE TANPA INCLUDE → balikkan seperti sebelumnya (full, tanpa nested)
	//  (StudentClassSectionResp sekarang sudah include gender + student_code)
	// =====================================================================
	if !includeClassSections && !includeCSST {
		out := make([]dto.StudentClassSectionResp, 0, len(rows))
		for i := range rows {
			out = append(out, dto.FromModel(&rows[i]))
		}
		return helper.JsonList(c, "OK", out, pagination)
	}

	// =====================================================================
	//  MODE include=class_sections / csst → FULL + nested
	// =====================================================================

	// 1) Kumpulkan section_id unik dari hasil query
	secIDSet := make(map[uuid.UUID]struct{})
	for i := range rows {
		secIDSet[rows[i].StudentClassSectionSectionID] = struct{}{}
	}

	secIDs = make([]uuid.UUID, 0, len(secIDSet))
	for id := range secIDSet {
		secIDs = append(secIDs, id)
	}

	// ---- Tipe nested untuk CSST ----
	type CSSTIncluded struct {
		ID                                 uuid.UUID                   `json:"class_section_subject_teacher_id"`
		Slug                               *string                     `json:"class_section_subject_teacher_slug,omitempty"`
		Description                        *string                     `json:"class_section_subject_teacher_description,omitempty"`
		GroupURL                           *string                     `json:"class_section_subject_teacher_group_url,omitempty"`
		TotalAttendance                    int                         `json:"class_section_subject_teacher_total_attendance"`
		TotalMeetingsTarget                *int                        `json:"class_section_subject_teacher_total_meetings_target,omitempty"`
		Capacity                           *int                        `json:"class_section_subject_teacher_capacity,omitempty"`
		EnrolledCount                      int                         `json:"class_section_subject_teacher_enrolled_count"`
		TotalAssessments                   int                         `json:"class_section_subject_teacher_total_assessments"`
		TotalAssessmentsGraded             int                         `json:"class_section_subject_teacher_total_assessments_graded"`
		TotalAssessmentsUngraded           int                         `json:"class_section_subject_teacher_total_assessments_ungraded"`
		TotalStudentsPassed                int                         `json:"class_section_subject_teacher_total_students_passed"`
		DeliveryMode                       csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode"`
		ClassSectionID                     uuid.UUID                   `json:"class_section_subject_teacher_class_section_id"`
		ClassRoomNameSnapshot              *string                     `json:"class_section_subject_teacher_class_room_name_snapshot,omitempty"`
		SchoolTeacherNameSnapshot          *string                     `json:"class_section_subject_teacher_school_teacher_name_snapshot,omitempty"`
		AssistantSchoolTeacherNameSnapshot *string                     `json:"class_section_subject_teacher_assistant_school_teacher_name_snapshot,omitempty"`
		SubjectNameSnapshot                *string                     `json:"class_section_subject_teacher_subject_name_snapshot,omitempty"`
		SubjectCodeSnapshot                *string                     `json:"class_section_subject_teacher_subject_code_snapshot,omitempty"`
		SubjectSlugSnapshot                *string                     `json:"class_section_subject_teacher_subject_slug_snapshot,omitempty"`
		MinPassingScore                    *int                        `json:"class_section_subject_teacher_min_passing_score,omitempty"`
		IsActive                           bool                        `json:"class_section_subject_teacher_is_active"`
		CreatedAt                          time.Time                   `json:"class_section_subject_teacher_created_at"`
		UpdatedAt                          time.Time                   `json:"class_section_subject_teacher_updated_at"`
	}

	// 2) Tipe nested untuk ClassSection + CSST list
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

		// list CSST
		SubjectTeachers []*CSSTIncluded `json:"class_section_subject_teachers,omitempty"`
	}

	classSectionMap := make(map[uuid.UUID]*ClassSectionIncluded)

	// 3) Query class_sections
	if len(secIDs) > 0 {
		var secRows []classSectionModel.ClassSectionModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&classSectionModel.ClassSectionModel{}).
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

				ImageURL:                cs.ClassSectionImageURL,
				ImageObjectKey:          cs.ClassSectionImageObjectKey,
				ImageURLOld:             cs.ClassSectionImageURLOld,
				ImageObjectKeyOld:       cs.ClassSectionImageObjectKeyOld,
				ImageDeletePendingUntil: cs.ClassSectionImageDeletePendingUntil,

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

			classSectionMap[cs.ClassSectionID] = item
		}
	}

	// 4) Query CSST & attach ke masing-masing section
	if includeCSST && len(secIDs) > 0 {
		var csstRows []csstModel.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where(`
				class_section_subject_teacher_school_id = ?
				AND class_section_subject_teacher_deleted_at IS NULL
				AND class_section_subject_teacher_class_section_id IN ?
			`, schoolID, secIDs).
			Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data subject teachers")
		}

		for i := range csstRows {
			r := csstRows[i]

			ci := &CSSTIncluded{
				ID:                                 r.ClassSectionSubjectTeacherID,
				Slug:                               r.ClassSectionSubjectTeacherSlug,
				Description:                        r.ClassSectionSubjectTeacherDescription,
				GroupURL:                           r.ClassSectionSubjectTeacherGroupURL,
				TotalAttendance:                    r.ClassSectionSubjectTeacherTotalAttendance,
				TotalMeetingsTarget:                r.ClassSectionSubjectTeacherTotalMeetingsTarget,
				Capacity:                           r.ClassSectionSubjectTeacherCapacity,
				EnrolledCount:                      r.ClassSectionSubjectTeacherEnrolledCount,
				TotalAssessments:                   r.ClassSectionSubjectTeacherTotalAssessments,
				TotalAssessmentsGraded:             r.ClassSectionSubjectTeacherTotalAssessmentsGraded,
				TotalAssessmentsUngraded:           r.ClassSectionSubjectTeacherTotalAssessmentsUngraded,
				TotalStudentsPassed:                r.ClassSectionSubjectTeacherTotalStudentsPassed,
				DeliveryMode:                       r.ClassSectionSubjectTeacherDeliveryMode,
				ClassSectionID:                     r.ClassSectionSubjectTeacherClassSectionID,
				ClassRoomNameSnapshot:              r.ClassSectionSubjectTeacherClassRoomNameSnapshot,
				SchoolTeacherNameSnapshot:          r.ClassSectionSubjectTeacherSchoolTeacherNameSnapshot,
				AssistantSchoolTeacherNameSnapshot: r.ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot,
				SubjectNameSnapshot:                r.ClassSectionSubjectTeacherSubjectNameSnapshot,
				SubjectCodeSnapshot:                r.ClassSectionSubjectTeacherSubjectCodeSnapshot,
				SubjectSlugSnapshot:                r.ClassSectionSubjectTeacherSubjectSlugSnapshot,
				MinPassingScore:                    r.ClassSectionSubjectTeacherMinPassingScore,
				IsActive:                           r.ClassSectionSubjectTeacherIsActive,
				CreatedAt:                          r.ClassSectionSubjectTeacherCreatedAt,
				UpdatedAt:                          r.ClassSectionSubjectTeacherUpdatedAt,
			}

			if sec, ok := classSectionMap[r.ClassSectionSubjectTeacherClassSectionID]; ok {
				sec.SubjectTeachers = append(sec.SubjectTeachers, ci)
			}
		}
	}

	// 5) Bentuk DTO nested
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
