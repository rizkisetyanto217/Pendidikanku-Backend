// file: internals/features/school/classes/class_section_subject_teachers/controller/student_csst_list_controller.go
package controller

import (
	"strings"
	"time"

	dto "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	model "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   Types untuk include=csst
   ========================================================= */

type CSSTIncluded struct {
	ID                  uuid.UUID `json:"class_section_subject_teacher_id"`
	Slug                *string   `json:"class_section_subject_teacher_slug,omitempty"`
	SubjectName         *string   `json:"class_section_subject_teacher_subject_name_snapshot,omitempty"`
	SubjectCode         *string   `json:"class_section_subject_teacher_subject_code_snapshot,omitempty"`
	SubjectSlug         *string   `json:"class_section_subject_teacher_subject_slug_snapshot,omitempty"`
	TeacherNameSnapshot *string   `json:"class_section_subject_teacher_school_teacher_name_snapshot,omitempty"`
	ClassSectionID      uuid.UUID `json:"class_section_subject_teacher_class_section_id"`
	ClassSectionName    *string   `json:"class_section_subject_teacher_class_section_name_snapshot,omitempty"`
	ClassSectionCode    *string   `json:"class_section_subject_teacher_class_section_code_snapshot,omitempty"`
	ClassSectionSlug    *string   `json:"class_section_subject_teacher_class_section_slug_snapshot,omitempty"`
	DeliveryMode        string    `json:"class_section_subject_teacher_delivery_mode"`
	EnrolledCount       int       `json:"class_section_subject_teacher_enrolled_count"`
	MinPassingScore     *int      `json:"class_section_subject_teacher_min_passing_score,omitempty"`
	ClassRoomName       *string   `json:"class_section_subject_teacher_class_room_name_snapshot,omitempty"`

	// ⬇️ NEW
	TotalBooks int       `json:"class_section_subject_teacher_total_books"`
	CreatedAt  string    `json:"class_section_subject_teacher_created_at"`
	UpdatedAt  string    `json:"class_section_subject_teacher_updated_at"`
	IsActive   bool      `json:"class_section_subject_teacher_is_active"`
	DeletedAt  *string   `json:"class_section_subject_teacher_deleted_at,omitempty"`
	SchoolID   uuid.UUID `json:"class_section_subject_teacher_school_id"`
}

type StudentCSSTWithCSST struct {
	dto.StudentCSSTItem
	CSST *CSSTIncluded `json:"class_section_subject_teacher,omitempty"`
}

/* =========================================================
   LIST
   ========================================================= */

// GET /api/a/student-csst
// ?student_id=<uuid>
// ?csst_id=<uuid>   // alias, diisi ke q.CSSTID
// ?is_active=true|false
// ?include=csst
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

	// 3) Parse query ke struct
	var q dto.StudentCSSTListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "query params tidak valid")
	}

	// 🔹 Alias: ?csst_id=<uuid> → isi q.CSSTID (override kalau ada)
	if raw := strings.TrimSpace(c.Query("csst_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "csst_id tidak valid")
		}
		q.CSSTID = &id
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

	// default paging (pakai q.Page & q.PageSize sesuai DTO kamu)
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 200 {
		q.PageSize = 20
	}

	// 4) Base query
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

	// Kalau kosong, langsung balikin list kosong + pagination standar
	if total == 0 {
		pagination := helper.BuildPaginationFromPage(total, q.Page, q.PageSize)
		// list kosong, nggak masalah mau include csst atau nggak → tetap []
		empty := []dto.StudentCSSTItem{}
		return helper.JsonList(c, "ok", empty, pagination)
	}

	// 6) Ambil page
	var rows []model.StudentClassSectionSubjectTeacher
	if err := tx.
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	pagination := helper.BuildPaginationFromPage(total, q.Page, q.PageSize)

	// ---------- mode default: TANPA include=csst ----------
	if !wantCSST {
		items := make([]dto.StudentCSSTItem, 0, len(rows))
		for i := range rows {
			items = append(items, toStudentCSSTItem(&rows[i]))
		}

		// ✅ format: { success, message, data: [...], pagination: {...} }
		return helper.JsonList(c, "ok", items, pagination)
	}

	// =====================================================
	//  MODE include=csst → embed detail CSST di tiap item
	// =====================================================

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

	// 2) query tabel class_section_subject_teachers
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
				SubjectName:         cs.ClassSectionSubjectTeacherSubjectNameSnapshot,
				SubjectCode:         cs.ClassSectionSubjectTeacherSubjectCodeSnapshot,
				SubjectSlug:         cs.ClassSectionSubjectTeacherSubjectSlugSnapshot,
				TeacherNameSnapshot: cs.ClassSectionSubjectTeacherSchoolTeacherNameSnapshot,
				ClassSectionID:      cs.ClassSectionSubjectTeacherClassSectionID,
				ClassSectionName:    cs.ClassSectionSubjectTeacherClassSectionNameSnapshot,
				ClassSectionCode:    cs.ClassSectionSubjectTeacherClassSectionCodeSnapshot,
				ClassSectionSlug:    cs.ClassSectionSubjectTeacherClassSectionSlugSnapshot,
				DeliveryMode:        string(cs.ClassSectionSubjectTeacherDeliveryMode),
				EnrolledCount:       cs.ClassSectionSubjectTeacherEnrolledCount,
				MinPassingScore:     cs.ClassSectionSubjectTeacherMinPassingScore,
				ClassRoomName:       cs.ClassSectionSubjectTeacherClassRoomNameSnapshot,
				IsActive:            cs.ClassSectionSubjectTeacherIsActive,
				SchoolID:            cs.ClassSectionSubjectTeacherSchoolID,
				CreatedAt:           cs.ClassSectionSubjectTeacherCreatedAt.Format(time.RFC3339),
				UpdatedAt:           cs.ClassSectionSubjectTeacherUpdatedAt.Format(time.RFC3339),
				// ⬇️ NEW
				TotalBooks: cs.ClassSectionSubjectTeacherTotalBooks,
			}
			if cs.ClassSectionSubjectTeacherDeletedAt.Valid {
				s := cs.ClassSectionSubjectTeacherDeletedAt.Time.Format(time.RFC3339)
				item.DeletedAt = &s
			}
			csstMap[cs.ClassSectionSubjectTeacherID] = item
		}
	}

	// 3) bentuk payload: item + csst nested
	out := make([]StudentCSSTWithCSST, 0, len(rows))
	for i := range rows {
		base := toStudentCSSTItem(&rows[i])

		var included *CSSTIncluded
		if cs, ok := csstMap[rows[i].StudentClassSectionSubjectTeacherCSSTID]; ok {
			included = cs
		}

		out = append(out, StudentCSSTWithCSST{
			StudentCSSTItem: base,
			CSST:            included,
		})
	}

	// ✅ Response final: { success, message, data: [...], pagination: {...} }
	return helper.JsonList(c, "ok", out, pagination)
}
