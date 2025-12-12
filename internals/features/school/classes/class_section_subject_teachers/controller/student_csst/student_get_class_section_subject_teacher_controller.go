package controller

import (
	"strings"

	dto "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"

	modelCSST "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	studentCSSTModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   Types untuk include=csst (pakai compact DTO baru csst_*)
========================================================= */

// CSST compact + murid-murid di bawahnya
type CSSTWithStudents struct {
	dto.CSSTCompactResponse
	Students []dto.StudentCSSTItem `json:"student_class_section_subject_teachers"`
}

/* =========================================================
   LIST
========================================================= */

// GET /api/u/student-class-section-subject-teachers/list
// ?student_id=<uuid> | me
// ?csst_id=<uuid>   // alias, diisi ke q.CSSTID
// ?is_active=true|false
// ?include=csst     // nested: satu CSST + students[]
// ?view=csst        // list CSST yang diikuti student (tanpa rows student)
// ?q=...
// ?page=1&page_size=20
func (ctl *StudentCSSTController) List(c *fiber.Ctx) error {
	// 1) School context dari token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// 2) Guard: staff (teacher/dkm/admin/bendahara)
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	// 🔹 Normalize student_id=me sebelum QueryParser (supaya nggak ke-parse sebagai UUID)
	studentIDRaw := strings.TrimSpace(c.Query("student_id"))
	studentIDIsMe := false
	if strings.EqualFold(studentIDRaw, "me") {
		studentIDIsMe = true
		c.Context().QueryArgs().Del("student_id")
	}

	// 3) Parse query ke struct DTO
	var q dto.StudentCSSTListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "query params tidak valid")
	}

	// Alias: ?csst_id=<uuid> → isi q.CSSTID (override kalau ada)
	if raw := strings.TrimSpace(c.Query("csst_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "csst_id tidak valid")
		}
		q.CSSTID = &id
	}

	// 🔹 Handle student_id=me → isi q.StudentID dari token
	if studentIDIsMe {
		studentID, err := helperAuth.ResolveStudentIDFromContext(c, schoolID)
		if err != nil {
			return err
		}
		q.StudentID = &studentID
	}

	// --- parse include ---
	includeRaw := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantCSST := false
	if includeRaw != "" {
		for _, p := range strings.Split(includeRaw, ",") {
			switch strings.TrimSpace(p) {
			case "csst", "class_section_subject_teacher", "class_section_subject_teachers":
				wantCSST = true
			}
		}
	}

	// --- parse view ---
	viewRaw := strings.ToLower(strings.TrimSpace(c.Query("view")))
	viewCSST := viewRaw == "csst" || viewRaw == "csst_only" || viewRaw == "csst_list"

	// Paging default
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 200 {
		q.PageSize = 20
	}

	// 4) Base query (student_csst_*)
	tx := ctl.DB.WithContext(c.Context()).
		Model(&studentCSSTModel.StudentClassSectionSubjectTeacherModel{}).
		Where("student_csst_school_id = ?", schoolID)

	// Filters
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

	// Pencarian ringan (sementara by slug)
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		like := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where("student_csst_slug ILIKE ?", like)
	}

	// Sorting
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

	// 5) Hitung total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal menghitung total")
	}

	pagination := helper.BuildPaginationFromPage(total, q.Page, q.PageSize)

	// Kalau kosong
	if total == 0 {
		if !wantCSST && !viewCSST {
			empty := []dto.StudentCSSTItem{}
			return helper.JsonList(c, "ok", empty, pagination)
		}
		return helper.JsonList(c, "ok", []any{}, pagination)
	}

	// 6) Ambil page
	var rows []studentCSSTModel.StudentClassSectionSubjectTeacherModel
	if err := tx.
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	/* ============================================
	   MODE view=csst → daftar CSST yang diikuti student
	   ============================================ */
	if viewCSST {
		// 1) kumpulkan csst_id unik dari rows
		csstSet := make(map[uuid.UUID]struct{})
		for i := range rows {
			if rows[i].StudentCSSTCSSTID != uuid.Nil {
				csstSet[rows[i].StudentCSSTCSSTID] = struct{}{}
			}
		}

		if len(csstSet) == 0 {
			items := []dto.CSSTCompactResponse{}
			return helper.JsonList(c, "ok", items, pagination)
		}

		csstIDs := make([]uuid.UUID, 0, len(csstSet))
		for id := range csstSet {
			csstIDs = append(csstIDs, id)
		}

		// 2) query tabel csst (csst_*)
		var csstRows []modelCSST.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
			Where("csst_school_id = ?", schoolID).
			Where("csst_deleted_at IS NULL").
			Where("csst_id IN ?", csstIDs).
			Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data csst")
		}

		// compact + timezone sekolah
		items := dto.FromCSSTModelsCompactWithSchoolTime(c, csstRows)
		return helper.JsonList(c, "ok", items, pagination)
	}

	/* ============================================
	   MODE default: TANPA include=csst
	   ============================================ */
	if !wantCSST {
		items := dto.FromStudentCSSTModelsWithSchoolTime(c, rows)
		return helper.JsonList(c, "ok", items, pagination)
	}

	/* ============================================
	   MODE include=csst → satu CSST + students[]
	   ============================================ */

	// 1) kumpulkan csst_id unik dari rows
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

	// 2) query tabel csst → bentuk csstMap (pakai compact)
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

	// 3) pilih satu CSST utama (kontrak sekarang: 1 CSST per respons nested)
	var mainCSST dto.CSSTCompactResponse
	var mainCSSTID uuid.UUID
	hasMain := false

	for id, v := range csstMap {
		mainCSST = v
		mainCSSTID = id
		hasMain = true
		break
	}

	// 4) kumpulkan students HANYA untuk CSST utama
	students := make([]dto.StudentCSSTItem, 0, len(rows))
	for i := range rows {
		if hasMain && rows[i].StudentCSSTCSSTID == mainCSSTID {
			students = append(students, dto.FromStudentCSSTModelWithSchoolTime(c, &rows[i]))
		}
	}

	// kalau entah bagaimana CSST tidak ada, tapi students ada → balikin students saja
	if !hasMain {
		return helper.JsonList(c, "ok", fiber.Map{
			"students": students,
		}, pagination)
	}

	wrapped := &CSSTWithStudents{
		CSSTCompactResponse: mainCSST,
		Students:            students,
	}

	// key biarin sama biar backward compatible
	return helper.JsonList(c, "ok", fiber.Map{
		"class_section_subject_teacher": wrapped,
	}, pagination)
}
