// file: internals/features/lembaga/class_section_subject_teachers/controller/csst_list_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"

	csstDTO "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	modelCSST "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

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
	// created_at|updated_at|subject_name|section_name|teacher_name|slug|academic_term_name|academic_year
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

// GET /api/u/class-section-subject-teachers/list
func (ctl *ClassSectionSubjectTeacherController) List(c *fiber.Ctx) error {
	// === School context ===
	var schoolID uuid.UUID

	if sid, err := helperAuth.ResolveSchoolIDFromContext(c); err == nil && sid != uuid.Nil {
		schoolID = sid
	}

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

	// ============ Query params umum ============
	var q listQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// Paging
	p := helper.ResolvePaging(c, 20, 200)
	if q.Offset != nil && *q.Offset >= 0 {
		p.Offset = *q.Offset
	}
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 {
		p.Limit = *q.Limit
	}
	limit, offset := p.Limit, p.Offset

	// ============ Query params khusus ============
	rawSectionID := strings.TrimSpace(c.Query("section_id"))
	rawClassSectionID := strings.TrimSpace(c.Query("class_section_id"))

	teacherIDStr := strings.TrimSpace(c.Query("teacher_id"))
	subjectIDStr := strings.TrimSpace(c.Query("subject_id"))

	// 🆕 multi academic_term_id (comma separated)
	rawAcademicTermID := strings.TrimSpace(c.Query("academic_term_id"))

	// 🆕 multi class_room_id / room_id (comma separated)
	rawClassRoomID := strings.TrimSpace(c.Query("class_room_id"))
	rawRoomID := strings.TrimSpace(c.Query("room_id"))
	if rawClassRoomID == "" {
		rawClassRoomID = rawRoomID
	}

	// 🆕 multi class_parent_id / parent_id (comma separated)
	rawClassParentID := strings.TrimSpace(c.Query("class_parent_id"))
	if rawClassParentID == "" {
		rawClassParentID = strings.TrimSpace(c.Query("parent_id"))
	}

	var classParentIDs []uuid.UUID
	if rawClassParentID != "" {
		ids, err := parseUUIDList(rawClassParentID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_parent_id/parent_id tidak valid: "+err.Error())
		}
		classParentIDs = ids
	}

	qtext := strings.TrimSpace(strings.ToLower(c.Query("q")))

	// =============== INCLUDE & NESTED TOKENS ===============
	includeRaw := strings.TrimSpace(c.Query("include"))
	includeTokens := map[string]bool{}
	if includeRaw != "" {
		for _, tok := range strings.Split(includeRaw, ",") {
			t := strings.TrimSpace(strings.ToLower(tok))
			if t != "" {
				includeTokens[t] = true
			}
		}
	}

	// nested=academic_term → nested di tiap item
	nestedRaw := strings.TrimSpace(c.Query("nested"))
	nestedTokens := map[string]bool{}
	if nestedRaw != "" {
		for _, tok := range strings.Split(nestedRaw, ",") {
			t := strings.TrimSpace(strings.ToLower(tok))
			if t != "" {
				nestedTokens[t] = true
			}
		}
	}

	// include=academic_term → list unik di include.academic_terms
	includeAcademicTermList := includeTokens["academic_term"] || includeTokens["academic_terms"]
	nestedAcademicTerm := nestedTokens["academic_term"]

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

	// parse teacher & subject ID
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

	// 🆕 parse academic_term_id as list
	var academicTermIDs []uuid.UUID
	if rawAcademicTermID != "" {
		ids, err := parseUUIDList(rawAcademicTermID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "academic_term_id tidak valid: "+err.Error())
		}
		academicTermIDs = ids
	}

	// 🆕 parse class_room_id / room_id as list
	var classRoomIDs []uuid.UUID
	if rawClassRoomID != "" {
		ids, err := parseUUIDList(rawClassRoomID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_room_id/room_id tidak valid: "+err.Error())
		}
		classRoomIDs = ids
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
		case "slug":
			orderCol = "class_section_subject_teacher_slug"
		case "academic_term_name":
			orderCol = "class_section_subject_teacher_academic_term_name_snapshot"
		case "academic_year":
			orderCol = "class_section_subject_teacher_academic_year_snapshot"
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "order_by tidak dikenal (gunakan: created_at, updated_at, subject_name, section_name, teacher_name, slug, academic_term_name, academic_year)")
		}
	}
	sortDir := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sortDir = "DESC"
	}
	orderExpr := fmt.Sprintf("%s %s", orderCol, sortDir)

	// ================= BASE QUERY =================
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
	if len(academicTermIDs) > 0 {
		tx = tx.Where("class_section_subject_teacher_academic_term_id IN ?", academicTermIDs)
	}

	// 🆕 filter by class_parent_id / parent_id via class_sections (pakai cache di section)
	if len(classParentIDs) > 0 {
		subq := ctl.DB.
			Table("class_sections").
			Select("class_section_id").
			Where("class_section_school_id = ?", schoolID).
			Where("class_section_deleted_at IS NULL").
			Where("class_section_class_parent_id IN ?", classParentIDs)

		tx = tx.Where("class_section_subject_teacher_class_section_id IN (?)", subq)
	}

	// 🆕 filter by class_room_id / room_id via subquery ke class_sections
	if len(classRoomIDs) > 0 {
		subq := ctl.DB.
			Table("class_sections").
			Select("class_section_id").
			Where("class_section_school_id = ?", schoolID).
			Where("class_section_deleted_at IS NULL").
			Where("class_section_class_room_id IN ?", classRoomIDs)

		tx = tx.Where("class_section_subject_teacher_class_section_id IN (?)", subq)
	}

	if qtext != "" {
		like := "%" + qtext + "%"
		tx = tx.Where(`
			LOWER(class_section_subject_teacher_slug) LIKE ? OR
			LOWER(class_section_subject_teacher_class_section_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_subject_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_school_teacher_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_academic_term_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_academic_year_snapshot) LIKE ?`,
			like, like, like, like, like, like,
		)
	}

	// ================= COUNT QUERY =================
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
	if len(academicTermIDs) > 0 {
		countTx = countTx.Where("class_section_subject_teacher_academic_term_id IN ?", academicTermIDs)
	}

	// 🆕 filter by class_room_id di COUNT juga (harus sama dengan tx)
	if len(classRoomIDs) > 0 {
		subq := ctl.DB.
			Table("class_sections").
			Select("class_section_id").
			Where("class_section_school_id = ?", schoolID).
			Where("class_section_deleted_at IS NULL").
			Where("class_section_class_room_id IN ?", classRoomIDs)

		countTx = countTx.Where("class_section_subject_teacher_class_section_id IN (?)", subq)
	}

	// 🆕 filter by class_parent_id di COUNT juga (pakai cache di section)
	if len(classParentIDs) > 0 {
		subq := ctl.DB.
			Table("class_sections").
			Select("class_section_id").
			Where("class_section_school_id = ?", schoolID).
			Where("class_section_deleted_at IS NULL").
			Where("class_section_class_parent_id IN ?", classParentIDs)

		countTx = countTx.Where("class_section_subject_teacher_class_section_id IN (?)", subq)
	}

	if qtext != "" {
		like := "%" + qtext + "%"
		countTx = countTx.Where(`
			LOWER(class_section_subject_teacher_slug) LIKE ? OR
			LOWER(class_section_subject_teacher_class_section_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_subject_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_school_teacher_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_academic_term_name_snapshot) LIKE ? OR
			LOWER(class_section_subject_teacher_academic_year_snapshot) LIKE ?`,
			like, like, like, like, like, like,
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

	// ================= DTO MAPPING =================
	resp := csstDTO.FromClassSectionSubjectTeacherModelsWithOptions(
		rows,
		csstDTO.FromCSSTOptions{
			// ❌ DULU: IncludeAcademicTerm: nestedAcademicTerm || includeAcademicTermList,
			// ✅ SEKARANG: nested academic_term HANYA kalau query ?nested=academic_term
			IncludeAcademicTerm: nestedAcademicTerm,
		},
	)

	pg := helper.BuildPaginationFromOffset(total, offset, limit)

	// ================= BUILD INCLUDE PAYLOAD =================
	includePayload := fiber.Map{}

	if includeAcademicTermList {
		// map academic_term_id → lite
		type AcademicTermLite struct {
			ID       uuid.UUID `json:"id"`
			Name     *string   `json:"name,omitempty"`
			Slug     *string   `json:"slug,omitempty"`
			Year     *string   `json:"year,omitempty"`
			Angkatan *int      `json:"angkatan,omitempty"`
		}

		byTerm := make(map[uuid.UUID]AcademicTermLite)

		for i := range rows {
			r := rows[i]
			// field ID pointer → cek nil dulu
			if r.ClassSectionSubjectTeacherAcademicTermID == nil {
				continue
			}
			termUUID := *r.ClassSectionSubjectTeacherAcademicTermID
			if termUUID == uuid.Nil {
				continue
			}

			item, ok := byTerm[termUUID]
			if !ok {
				item.ID = termUUID
			}
			// isi hanya kalau masih kosong, biar nggak sering override
			if item.Name == nil {
				item.Name = r.ClassSectionSubjectTeacherAcademicTermNameCache
			}
			if item.Slug == nil {
				item.Slug = r.ClassSectionSubjectTeacherAcademicTermSlugCache
			}
			if item.Year == nil {
				item.Year = r.ClassSectionSubjectTeacherAcademicYearCache
			}
			if item.Angkatan == nil {
				item.Angkatan = r.ClassSectionSubjectTeacherAcademicTermAngkatanCache
			}

			byTerm[termUUID] = item
		}

		terms := make([]AcademicTermLite, 0, len(byTerm))
		for _, v := range byTerm {
			terms = append(terms, v)
		}

		includePayload["academic_terms"] = terms
	}

	// pakai JsonListWithInclude supaya selalu ada "include": {...}
	return helper.JsonListWithInclude(c, "ok", resp, includePayload, pg)
}
