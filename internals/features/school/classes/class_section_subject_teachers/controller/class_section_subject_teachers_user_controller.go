// file: internals/features/school/academics/subject/controller/class_section_subject_teachers_user_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"

	modelCSST "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ============================= Query params ============================== */

type listQuery struct {
	IsActive    *bool `query:"is_active"`
	WithDeleted *bool `query:"with_deleted"`
	Limit       *int  `query:"limit"`
	Offset      *int  `query:"offset"`
	// created_at|updated_at|subject_name|section_name|teacher_name|book_title|slug
	OrderBy *string `query:"order_by"`
	// asc|desc
	Sort *string `query:"sort"`
}

/* ============ Helper: parse list UUID (buat query param id, dst) ============ */

func parseUUIDList(s string) ([]uuid.UUID, error) {
	parts := strings.Split(s, ",")
	out := make([]uuid.UUID, 0, len(parts))
	seen := make(map[uuid.UUID]struct{}, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}

	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "daftar id kosong")
	}
	return out, nil
}

/* ================================ Handler (NO-JOIN) ================================ */

// Satu endpoint saja:
//
//	GET /api/u/class-section-subject-teachers/list
//
// Filter via query:
//   - ?id=uuid,uuid2
//   - ?class_section_id=uuid,uuid2 atau ?section_id=uuid,uuid2
//   - ?teacher_id=uuid
//   - ?subject_id=uuid
//   - ?q=...
//   - ?is_active=...
//   - ?with_deleted=...
//   - ?order_by=created_at|updated_at|subject_name|section_name|teacher_name|book_title|slug
//   - ?sort=asc|desc
//   - paging: ?page=&per_page= (ResolvePaging) atau ?limit=&offset=
func (ctl *ClassSectionSubjectTeacherController) List(c *fiber.Ctx) error {
	// === School context ===
	var schoolID uuid.UUID

	// 1) Prioritas: token / middleware UseSchoolScope (active_school_id)
	if sid, err := helperAuth.ResolveSchoolIDFromContext(c); err == nil && sid != uuid.Nil {
		schoolID = sid
	}

	// (opsional) fallback: kalau suatu saat dipakai di route lain yg pakai :school_id atau :school_slug
	if schoolID == uuid.Nil {
		if raw := strings.TrimSpace(c.Params("school_id")); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil || id == uuid.Nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "school_id path tidak valid")
			}
			schoolID = id
		}
	}
	if schoolID == uuid.Nil {
		if slug := strings.TrimSpace(c.Params("school_slug")); slug != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, slug)
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id
		}
	}

	if schoolID == uuid.Nil {
		return helper.JsonError(
			c,
			helperAuth.ErrSchoolContextMissing.Code,
			helperAuth.ErrSchoolContextMissing.Message,
		)
	}

	// query params umum (limit/offset/order_by/sort/is_active/with_deleted)
	var q listQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ===== Paging (standar jsonresponse) =====
	p := helper.ResolvePaging(c, 20, 200)
	if q.Offset != nil && *q.Offset >= 0 {
		p.Offset = *q.Offset
	}
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 {
		p.Limit = *q.Limit
	}
	limit, offset := p.Limit, p.Offset

	// ==== Query params khusus ====
	// Dukung dua nama section_id via query:
	rawSectionID := strings.TrimSpace(c.Query("section_id"))
	rawClassSectionID := strings.TrimSpace(c.Query("class_section_id"))

	teacherIDStr := strings.TrimSpace(c.Query("teacher_id"))
	subjectIDStr := strings.TrimSpace(c.Query("subject_id"))
	qtext := strings.TrimSpace(strings.ToLower(c.Query("q")))

	// filter by id via query param (bisa multi)
	var filterIDs []uuid.UUID
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		ids, err := parseUUIDList(s)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid: "+err.Error())
		}
		filterIDs = ids
	}

	// multi filter untuk class_section_id / section_id
	var sectionIDs []uuid.UUID
	combinedSection := strings.TrimSpace(rawSectionID)
	if combinedSection != "" && rawClassSectionID != "" {
		combinedSection = combinedSection + "," + strings.TrimSpace(rawClassSectionID)
	} else if combinedSection == "" && rawClassSectionID != "" {
		combinedSection = strings.TrimSpace(rawClassSectionID)
	}
	if combinedSection != "" {
		ids, err := parseUUIDList(combinedSection)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "section_id/class_section_id tidak valid: "+err.Error())
		}
		sectionIDs = ids
	}

	// parse teacher & subject ID (opsional)
	var teacherID *uuid.UUID
	if teacherIDStr != "" {
		if id, err := uuid.Parse(teacherIDStr); err == nil {
			teacherID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}

	var subjectID *uuid.UUID
	if subjectIDStr != "" {
		if id, err := uuid.Parse(subjectIDStr); err == nil {
			subjectID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "subject_id tidak valid")
		}
	}

	// ===== Sorting =====
	orderCol := "class_section_subject_teacher_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "created_at":
			orderCol = "class_section_subject_teacher_created_at"
		case "updated_at":
			orderCol = "class_section_subject_teacher_updated_at"
		case "subject_name":
			orderCol = "class_section_subject_teacher_subject_name_snapshot"
		case "section_name":
			orderCol = "class_section_subject_teacher_class_section_name_snapshot"
		case "teacher_name":
			orderCol = "class_section_subject_teacher_school_teacher_name_snapshot"
		case "book_title":
			orderCol = "class_section_subject_teacher_book_title_snapshot"
		case "slug":
			orderCol = "class_section_subject_teacher_slug"
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "order_by tidak dikenal (gunakan: created_at, updated_at, subject_name, section_name, teacher_name, book_title, slug)")
		}
	}
	sortDir := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sortDir = "DESC"
	}
	orderExpr := fmt.Sprintf("%s %s", orderCol, sortDir)

	// ===== BASE QUERY (pakai model, tanpa join) =====
	tx := ctl.DB.
		Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teacher_school_id = ?", schoolID)

	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_section_subject_teacher_deleted_at IS NULL")
	}
	if q.IsActive != nil {
		tx = tx.Where("class_section_subject_teacher_is_active = ?", *q.IsActive)
	}

	if len(filterIDs) > 0 {
		tx = tx.Where("class_section_subject_teacher_id IN ?", filterIDs)
	}
	if len(sectionIDs) > 0 {
		tx = tx.Where("class_section_subject_teacher_class_section_id IN ?", sectionIDs)
	}
	if teacherID != nil {
		tx = tx.Where("class_section_subject_teacher_school_teacher_id = ?", *teacherID)
	}
	if subjectID != nil {
		tx = tx.Where("class_section_subject_teacher_subject_id_snapshot = ?", *subjectID)
	}

	if qtext != "" {
		like := "%" + qtext + "%"
		tx = tx.Where(`
			LOWER(class_section_subject_teacher_slug) LIKE ? OR
			LOWER(class_section_subject_teacher_class_section_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_subject_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_school_teacher_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_book_title_snapshot) LIKE ?`,
			like, like, like, like, like,
		)
	}

	// COUNT
	countTx := ctl.DB.
		Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teacher_school_id = ?", schoolID)

	if q.WithDeleted == nil || !*q.WithDeleted {
		countTx = countTx.Where("class_section_subject_teacher_deleted_at IS NULL")
	}
	if q.IsActive != nil {
		countTx = countTx.Where("class_section_subject_teacher_is_active = ?", *q.IsActive)
	}
	if len(filterIDs) > 0 {
		countTx = countTx.Where("class_section_subject_teacher_id IN ?", filterIDs)
	}
	if len(sectionIDs) > 0 {
		countTx = countTx.Where("class_section_subject_teacher_class_section_id IN ?", sectionIDs)
	}
	if teacherID != nil {
		countTx = countTx.Where("class_section_subject_teacher_school_teacher_id = ?", *teacherID)
	}
	if subjectID != nil {
		countTx = countTx.Where("class_section_subject_teacher_subject_id_snapshot = ?", *subjectID)
	}
	if qtext != "" {
		like := "%" + qtext + "%"
		countTx = countTx.Where(`
			LOWER(class_section_subject_teacher_slug) LIKE ? OR
			LOWER(class_section_subject_teacher_class_section_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_subject_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_school_teacher_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_book_title_snapshot) LIKE ?`,
			like, like, like, like, like,
		)
	}

	var total int64
	if err := countTx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// LIST
	var rows []modelCSST.ClassSectionSubjectTeacherModel
	if err := tx.
		Order(orderExpr).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	pg := helper.BuildPaginationFromOffset(total, offset, limit)
	return helper.JsonList(c, "ok", rows, pg)
}
