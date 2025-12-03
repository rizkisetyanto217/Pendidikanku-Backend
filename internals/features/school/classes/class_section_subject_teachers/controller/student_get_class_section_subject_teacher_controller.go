// file: internals/features/school/classes/class_section_subject_teachers/controller/student_csst_list_controller.go
package controller

import (
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	model "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   Types untuk include=csst
========================================================= */

type CSSTIncluded struct {
	ID                  uuid.UUID `json:"class_section_subject_teacher_id"`
	Slug                *string   `json:"class_section_subject_teacher_slug,omitempty"`
	SubjectName         *string   `json:"class_section_subject_teacher_subject_name_cache,omitempty"`
	SubjectCode         *string   `json:"class_section_subject_teacher_subject_code_cache,omitempty"`
	SubjectSlug         *string   `json:"class_section_subject_teacher_subject_slug_cache,omitempty"`
	TeacherNameCache *string   `json:"class_section_subject_teacher_school_teacher_name_cache,omitempty"`
	ClassSectionID      uuid.UUID `json:"class_section_subject_teacher_class_section_id"`
	ClassSectionName    *string   `json:"class_section_subject_teacher_class_section_name_cache,omitempty"`
	ClassSectionCode    *string   `json:"class_section_subject_teacher_class_section_code_cache,omitempty"`
	ClassSectionSlug    *string   `json:"class_section_subject_teacher_class_section_slug_cache,omitempty"`
	DeliveryMode        string    `json:"class_section_subject_teacher_delivery_mode"`
	EnrolledCount       int       `json:"class_section_subject_teacher_enrolled_count"`
	MinPassingScore     *int      `json:"class_section_subject_teacher_min_passing_score,omitempty"`
	ClassRoomName       *string   `json:"class_section_subject_teacher_class_room_name_cache,omitempty"`

	// Tambahan info CSST
	TotalBooks int       `json:"class_section_subject_teacher_total_books"`
	CreatedAt  string    `json:"class_section_subject_teacher_created_at"`
	UpdatedAt  string    `json:"class_section_subject_teacher_updated_at"`
	IsActive   bool      `json:"class_section_subject_teacher_is_active"`
	DeletedAt  *string   `json:"class_section_subject_teacher_deleted_at,omitempty"`
	SchoolID   uuid.UUID `json:"class_section_subject_teacher_school_id"`
}

// CSST di atas + murid-murid di bawahnya
type CSSTWithStudents struct {
	CSSTIncluded
	Students []dto.StudentCSSTItem `json:"students"`
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
		// hapus dari query args supaya QueryParser nggak error
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

	// 4) Base query (join table)
	tx := ctl.DB.WithContext(c.Context()).
		Model(&model.StudentClassSectionSubjectTeacher{}).
		Where("student_class_section_subject_teacher_school_id = ?", schoolID)

	// Filters
	if q.StudentID != nil {
		tx = tx.Where("student_class_section_subject_teacher_student_id = ?", *q.StudentID)
	}
	if q.CSSTID != nil {
		tx = tx.Where("student_class_section_subject_teacher_csst_id = ?", *q.CSSTID)
	}
	if q.IsActive != nil {
		tx = tx.Where("student_class_section_subject_teacher_is_active = ?", *q.IsActive)
	}
	if !q.IncludeDeleted {
		tx = tx.Where("student_class_section_subject_teacher_deleted_at IS NULL")
	}

	// Pencarian ringan (sementara by slug)
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		like := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where("student_class_section_subject_teacher_slug ILIKE ?", like)
	}

	// Sorting
	sortBy := strings.ToLower(strings.TrimSpace(q.SortBy))
	order := strings.ToLower(strings.TrimSpace(q.Order))
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	switch sortBy {
	case dto.StudentCSSTSortUpdatedAt:
		tx = tx.Order("student_class_section_subject_teacher_updated_at " + order)
	case dto.StudentCSSTSortStudent:
		tx = tx.Order("student_class_section_subject_teacher_student_id " + order)
	case dto.StudentCSSTSortCSST:
		tx = tx.Order("student_class_section_subject_teacher_csst_id " + order)
	default:
		tx = tx.Order("student_class_section_subject_teacher_created_at " + order)
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

		// include=csst / view=csst → kembalikan struktur konsisten
		return helper.JsonList(c, "ok", []any{}, pagination)
	}

	// 6) Ambil page
	var rows []model.StudentClassSectionSubjectTeacher
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
			if rows[i].StudentClassSectionSubjectTeacherCSSTID != uuid.Nil {
				csstSet[rows[i].StudentClassSectionSubjectTeacherCSSTID] = struct{}{}
			}
		}

		if len(csstSet) == 0 {
			items := []CSSTIncluded{}
			return helper.JsonList(c, "ok", items, pagination)
		}

		csstIDs := make([]uuid.UUID, 0, len(csstSet))
		for id := range csstSet {
			csstIDs = append(csstIDs, id)
		}

		// 2) query tabel class_section_subject_teachers
		var csstRows []model.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&model.ClassSectionSubjectTeacherModel{}).
			Where("class_section_subject_teacher_school_id = ?", schoolID).
			Where("class_section_subject_teacher_id IN ?", csstIDs).
			Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data csst")
		}

		items := make([]CSSTIncluded, 0, len(csstRows))
		for i := range csstRows {
			cs := csstRows[i]
			item := CSSTIncluded{
				ID:                  cs.ClassSectionSubjectTeacherID,
				Slug:                cs.ClassSectionSubjectTeacherSlug,
				SubjectName:         cs.ClassSectionSubjectTeacherSubjectNameCache,
				SubjectCode:         cs.ClassSectionSubjectTeacherSubjectCodeCache,
				SubjectSlug:         cs.ClassSectionSubjectTeacherSubjectSlugCache,
				TeacherNameCache: cs.ClassSectionSubjectTeacherSchoolTeacherNameCache,
				ClassSectionID:      cs.ClassSectionSubjectTeacherClassSectionID,
				ClassSectionName:    cs.ClassSectionSubjectTeacherClassSectionNameCache,
				ClassSectionCode:    cs.ClassSectionSubjectTeacherClassSectionCodeCache,
				ClassSectionSlug:    cs.ClassSectionSubjectTeacherClassSectionSlugCache,
				DeliveryMode:        string(cs.ClassSectionSubjectTeacherDeliveryMode),
				EnrolledCount:       cs.ClassSectionSubjectTeacherEnrolledCount,
				MinPassingScore:     cs.ClassSectionSubjectTeacherMinPassingScore,
				ClassRoomName:       cs.ClassSectionSubjectTeacherClassRoomNameCache,
				IsActive:            cs.ClassSectionSubjectTeacherIsActive,
				SchoolID:            cs.ClassSectionSubjectTeacherSchoolID,
				CreatedAt:           cs.ClassSectionSubjectTeacherCreatedAt.Format(time.RFC3339),
				UpdatedAt:           cs.ClassSectionSubjectTeacherUpdatedAt.Format(time.RFC3339),
				TotalBooks:          cs.ClassSectionSubjectTeacherTotalBooks,
			}
			if cs.ClassSectionSubjectTeacherDeletedAt.Valid {
				s := cs.ClassSectionSubjectTeacherDeletedAt.Time.Format(time.RFC3339)
				item.DeletedAt = &s
			}
			items = append(items, item)
		}

		// (opsional) kalau mau, bisa override pagination ke jumlah CSST unik.
		// totalCSST := int64(len(items))
		// pagination = helper.BuildPaginationFromPage(totalCSST, q.Page, q.PageSize)

		return helper.JsonList(c, "ok", items, pagination)
	}

	/* ============================================
	   MODE default: TANPA include=csst
	   ============================================ */
	if !wantCSST {
		items := make([]dto.StudentCSSTItem, 0, len(rows))
		for i := range rows {
			items = append(items, toStudentCSSTItem(&rows[i]))
		}
		return helper.JsonList(c, "ok", items, pagination)
	}

	/* ============================================
	   MODE include=csst → satu CSST + students[]
	   ============================================ */

	// 1) kumpulkan csst_id unik dari rows
	csstSet := make(map[uuid.UUID]struct{})
	for i := range rows {
		if rows[i].StudentClassSectionSubjectTeacherCSSTID != uuid.Nil {
			csstSet[rows[i].StudentClassSectionSubjectTeacherCSSTID] = struct{}{}
		}
	}

	csstIDs := make([]uuid.UUID, 0, len(csstSet))
	for id := range csstSet {
		csstIDs = append(csstIDs, id)
	}

	// 2) query tabel class_section_subject_teachers → bentuk csstMap
	csstMap := make(map[uuid.UUID]*CSSTIncluded)

	if len(csstIDs) > 0 {
		var csstRows []model.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&model.ClassSectionSubjectTeacherModel{}).
			Where("class_section_subject_teacher_school_id = ?", schoolID).
			Where("class_section_subject_teacher_id IN ?", csstIDs).
			Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data csst")
		}

		for i := range csstRows {
			cs := csstRows[i]
			item := &CSSTIncluded{
				ID:                  cs.ClassSectionSubjectTeacherID,
				Slug:                cs.ClassSectionSubjectTeacherSlug,
				SubjectName:         cs.ClassSectionSubjectTeacherSubjectNameCache,
				SubjectCode:         cs.ClassSectionSubjectTeacherSubjectCodeCache,
				SubjectSlug:         cs.ClassSectionSubjectTeacherSubjectSlugCache,
				TeacherNameCache: cs.ClassSectionSubjectTeacherSchoolTeacherNameCache,
				ClassSectionID:      cs.ClassSectionSubjectTeacherClassSectionID,
				ClassSectionName:    cs.ClassSectionSubjectTeacherClassSectionNameCache,
				ClassSectionCode:    cs.ClassSectionSubjectTeacherClassSectionCodeCache,
				ClassSectionSlug:    cs.ClassSectionSubjectTeacherClassSectionSlugCache,
				DeliveryMode:        string(cs.ClassSectionSubjectTeacherDeliveryMode),
				EnrolledCount:       cs.ClassSectionSubjectTeacherEnrolledCount,
				MinPassingScore:     cs.ClassSectionSubjectTeacherMinPassingScore,
				ClassRoomName:       cs.ClassSectionSubjectTeacherClassRoomNameCache,
				IsActive:            cs.ClassSectionSubjectTeacherIsActive,
				SchoolID:            cs.ClassSectionSubjectTeacherSchoolID,
				CreatedAt:           cs.ClassSectionSubjectTeacherCreatedAt.Format(time.RFC3339),
				UpdatedAt:           cs.ClassSectionSubjectTeacherUpdatedAt.Format(time.RFC3339),
				TotalBooks:          cs.ClassSectionSubjectTeacherTotalBooks,
			}
			if cs.ClassSectionSubjectTeacherDeletedAt.Valid {
				s := cs.ClassSectionSubjectTeacherDeletedAt.Time.Format(time.RFC3339)
				item.DeletedAt = &s
			}
			csstMap[cs.ClassSectionSubjectTeacherID] = item
		}
	}

	// 3) pilih satu CSST utama (kontrak sekarang: 1 CSST per respons nested)
	var mainCSST *CSSTIncluded
	var mainCSSTID uuid.UUID
	for id, v := range csstMap {
		mainCSST = v
		mainCSSTID = id
		break
	}

	// 4) kumpulkan students HANYA untuk CSST utama
	students := make([]dto.StudentCSSTItem, 0, len(rows))
	for i := range rows {
		if rows[i].StudentClassSectionSubjectTeacherCSSTID == mainCSSTID {
			students = append(students, toStudentCSSTItem(&rows[i]))
		}
	}

	// kalau entah bagaimana CSST tidak ada, tapi students ada → balikin students saja
	if mainCSST == nil {
		return helper.JsonList(c, "ok", fiber.Map{
			"students": students,
		}, pagination)
	}

	wrapped := &CSSTWithStudents{
		CSSTIncluded: *mainCSST,
		Students:     students,
	}

	return helper.JsonList(c, "ok", fiber.Map{
		"class_section_subject_teacher": wrapped,
	}, pagination)
}
