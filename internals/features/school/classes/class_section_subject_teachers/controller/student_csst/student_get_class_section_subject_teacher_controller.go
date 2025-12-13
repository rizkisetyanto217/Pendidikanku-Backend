package controller

import (
	"strings"

	dto "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	modelCSST "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   Wrappers include=csst
========================================================= */

type CSSTWithStudentsFull struct {
	dto.CSSTCompactResponse
	Students []dto.StudentCSSTItem `json:"student_class_section_subject_teachers"`
}

type CSSTWithStudentsCompact struct {
	dto.CSSTCompactResponse
	Students []dto.StudentCSSTCompactItem `json:"student_class_section_subject_teachers"`
}

/* =========================================================
   Helpers
========================================================= */

func parseMode(modeRaw string) (wantCompact bool) {
	mode := strings.ToLower(strings.TrimSpace(modeRaw))
	switch mode {
	case "compact", "mini", "light":
		return true
	default:
		return false // default full
	}
}

func parseIncludeCSST(c *fiber.Ctx) bool {
	includeRaw := strings.ToLower(strings.TrimSpace(c.Query("include")))
	if includeRaw == "" {
		return false
	}
	for _, p := range strings.Split(includeRaw, ",") {
		switch strings.TrimSpace(p) {
		case "csst", "class_section_subject_teacher", "class_section_subject_teachers":
			return true
		}
	}
	return false
}

func parseViewCSST(c *fiber.Ctx) bool {
	viewRaw := strings.ToLower(strings.TrimSpace(c.Query("view")))
	return viewRaw == "csst" || viewRaw == "csst_only" || viewRaw == "csst_list"
}

/* =========================================================
   GET: LIST or DETAIL (single endpoint)
========================================================= */

// GET /api/u/student-class-section-subject-teachers/list
//
// LIST (default):
//
//	?student_id=<uuid>|me
//	?csst_id=<uuid>
//	?is_active=true|false
//	?include=csst
//	?view=csst
//	?mode=full|compact (default full)
//	?q=...
//	?page=1&page_size=20
//
// DETAIL (kalau id ada):
//
//	?id=<student_csst_id>
//	+ optional: ?include=csst & ?mode=compact
func (ctl *StudentCSSTController) List(c *fiber.Ctx) error {
	// 1) School context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// 2) Guard staff
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	// mode
	wantCompact := parseMode(c.Query("mode"))

	// include/view
	wantCSST := parseIncludeCSST(c)
	viewCSST := parseViewCSST(c)

	// ==== DETAIL MODE (id di query) ====
	if idRaw := strings.TrimSpace(c.Query("id")); idRaw != "" {
		id, err := uuid.Parse(idRaw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}

		var row modelCSST.StudentClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&modelCSST.StudentClassSectionSubjectTeacherModel{}).
			Where("student_csst_school_id = ?", schoolID).
			Where("student_csst_id = ?", id).
			Where("student_csst_deleted_at IS NULL").
			First(&row).Error; err != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "data tidak ditemukan")
		}

		// no include
		if !wantCSST {
			if wantCompact {
				item := dto.FromStudentCSSTModelCompactWithSchoolTime(c, &row)
				return helper.JsonOK(c, "ok", item)
			}
			item := dto.FromStudentCSSTModelWithSchoolTime(c, &row)
			return helper.JsonOK(c, "ok", item)
		}

		// include csst
		var csst modelCSST.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
			Where("csst_school_id = ?", schoolID).
			Where("csst_deleted_at IS NULL").
			Where("csst_id = ?", row.StudentCSSTCSSTID).
			First(&csst).Error; err != nil {
			// csst missing -> return student only
			if wantCompact {
				item := dto.FromStudentCSSTModelCompactWithSchoolTime(c, &row)
				return helper.JsonOK(c, "ok", fiber.Map{"student": item})
			}
			item := dto.FromStudentCSSTModelWithSchoolTime(c, &row)
			return helper.JsonOK(c, "ok", fiber.Map{"student": item})
		}

		csstCompact := dto.FromCSSTModelsCompactWithSchoolTime(c, []modelCSST.ClassSectionSubjectTeacherModel{csst})
		var csstItem dto.CSSTCompactResponse
		if len(csstCompact) > 0 {
			csstItem = csstCompact[0]
		}

		if wantCompact {
			stu := dto.FromStudentCSSTModelCompactWithSchoolTime(c, &row)
			return helper.JsonOK(c, "ok", fiber.Map{
				"class_section_subject_teacher": csstItem,
				"student":                       stu,
			})
		}

		stu := dto.FromStudentCSSTModelWithSchoolTime(c, &row)
		return helper.JsonOK(c, "ok", fiber.Map{
			"class_section_subject_teacher": csstItem,
			"student":                       stu,
		})
	}

	// ==== LIST MODE ====

	// normalize student_id=me sebelum QueryParser
	studentIDRaw := strings.TrimSpace(c.Query("student_id"))
	studentIDIsMe := false
	if strings.EqualFold(studentIDRaw, "me") {
		studentIDIsMe = true
		c.Context().QueryArgs().Del("student_id")
	}

	var q dto.StudentCSSTListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "query params tidak valid")
	}

	// alias csst_id manual override (biar aman)
	if raw := strings.TrimSpace(c.Query("csst_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "csst_id tidak valid")
		}
		q.CSSTID = &id
	}

	// student_id=me -> resolve token
	if studentIDIsMe {
		studentID, err := helperAuth.ResolveStudentIDFromContext(c, schoolID)
		if err != nil {
			return err
		}
		q.StudentID = &studentID
	}

	// paging default
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 200 {
		q.PageSize = 20
	}

	tx := ctl.DB.WithContext(c.Context()).
		Model(&modelCSST.StudentClassSectionSubjectTeacherModel{}).
		Where("student_csst_school_id = ?", schoolID)

	if q.StudentID != nil {
		tx = tx.Where("student_csst_student_id = ?", *q.StudentID)
	}
	if q.CSSTID != nil {
		tx = tx.Where("student_csst_csst_id = ?", *q.CSSTID)
	}
	if q.IsActive != nil {
		tx = tx.Where("student_csst_is_active = ?", *q.IsActive)
	}
	if !q.IncludeDeleted {
		tx = tx.Where("student_csst_deleted_at IS NULL")
	}

	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		like := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where("student_csst_slug ILIKE ?", like)
	}

	sortBy := strings.ToLower(strings.TrimSpace(q.SortBy))
	order := strings.ToLower(strings.TrimSpace(q.Order))
	if order != "asc" && order != "desc" {
		order = "asc"
	}
	switch sortBy {
	case dto.StudentCSSTSortUpdatedAt:
		tx = tx.Order("student_csst_updated_at " + order)
	case dto.StudentCSSTSortStudent:
		tx = tx.Order("student_csst_student_id " + order)
	case dto.StudentCSSTSortCSST:
		tx = tx.Order("student_csst_csst_id " + order)
	default:
		tx = tx.Order("student_csst_created_at " + order)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal menghitung total")
	}
	pagination := helper.BuildPaginationFromPage(total, q.Page, q.PageSize)

	if total == 0 {
		if !wantCSST && !viewCSST {
			if wantCompact {
				return helper.JsonList(c, "ok", []dto.StudentCSSTCompactItem{}, pagination)
			}
			return helper.JsonList(c, "ok", []dto.StudentCSSTItem{}, pagination)
		}
		return helper.JsonList(c, "ok", []any{}, pagination)
	}

	var rows []modelCSST.StudentClassSectionSubjectTeacherModel
	if err := tx.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	// view=csst
	if viewCSST {
		csstSet := make(map[uuid.UUID]struct{})
		for i := range rows {
			if rows[i].StudentCSSTCSSTID != uuid.Nil {
				csstSet[rows[i].StudentCSSTCSSTID] = struct{}{}
			}
		}
		if len(csstSet) == 0 {
			return helper.JsonList(c, "ok", []dto.CSSTCompactResponse{}, pagination)
		}
		csstIDs := make([]uuid.UUID, 0, len(csstSet))
		for id := range csstSet {
			csstIDs = append(csstIDs, id)
		}

		var csstRows []modelCSST.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
			Where("csst_school_id = ?", schoolID).
			Where("csst_deleted_at IS NULL").
			Where("csst_id IN ?", csstIDs).
			Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data csst")
		}

		items := dto.FromCSSTModelsCompactWithSchoolTime(c, csstRows)
		return helper.JsonList(c, "ok", items, pagination)
	}

	// default no include=csst
	if !wantCSST {
		if wantCompact {
			items := dto.FromStudentCSSTModelsCompactWithSchoolTime(c, rows)
			return helper.JsonList(c, "ok", items, pagination)
		}
		items := dto.FromStudentCSSTModelsWithSchoolTime(c, rows)
		return helper.JsonList(c, "ok", items, pagination)
	}

	// include=csst -> 1 csst + students[]
	csstSet := make(map[uuid.UUID]struct{})
	for i := range rows {
		if rows[i].StudentCSSTCSSTID != uuid.Nil {
			csstSet[rows[i].StudentCSSTCSSTID] = struct{}{}
		}
	}
	csstIDs := make([]uuid.UUID, 0, len(csstSet))
	for id := range csstSet {
		csstIDs = append(csstIDs, id)
	}

	csstMap := make(map[uuid.UUID]dto.CSSTCompactResponse)
	if len(csstIDs) > 0 {
		var csstRows []modelCSST.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
			Where("csst_school_id = ?", schoolID).
			Where("csst_deleted_at IS NULL").
			Where("csst_id IN ?", csstIDs).
			Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data csst")
		}

		compactList := dto.FromCSSTModelsCompactWithSchoolTime(c, csstRows)
		for i := range compactList {
			item := compactList[i]
			csstMap[item.CSSTID] = item
		}
	}

	var mainCSST dto.CSSTCompactResponse
	var mainCSSTID uuid.UUID
	hasMain := false
	for id, v := range csstMap {
		mainCSST = v
		mainCSSTID = id
		hasMain = true
		break
	}

	if wantCompact {
		students := make([]dto.StudentCSSTCompactItem, 0, len(rows))
		for i := range rows {
			if hasMain && rows[i].StudentCSSTCSSTID == mainCSSTID {
				students = append(students, dto.FromStudentCSSTModelCompactWithSchoolTime(c, &rows[i]))
			}
		}
		if !hasMain {
			return helper.JsonList(c, "ok", fiber.Map{"students": students}, pagination)
		}
		wrapped := &CSSTWithStudentsCompact{CSSTCompactResponse: mainCSST, Students: students}
		return helper.JsonList(c, "ok", fiber.Map{"class_section_subject_teacher": wrapped}, pagination)
	}

	students := make([]dto.StudentCSSTItem, 0, len(rows))
	for i := range rows {
		if hasMain && rows[i].StudentCSSTCSSTID == mainCSSTID {
			students = append(students, dto.FromStudentCSSTModelWithSchoolTime(c, &rows[i]))
		}
	}
	if !hasMain {
		return helper.JsonList(c, "ok", fiber.Map{"students": students}, pagination)
	}
	wrapped := &CSSTWithStudentsFull{CSSTCompactResponse: mainCSST, Students: students}
	return helper.JsonList(c, "ok", fiber.Map{"class_section_subject_teacher": wrapped}, pagination)
}
