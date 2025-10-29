package controller

import (
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"
	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================
   Small helpers (local)
========================= */

func getSortClauseAssessment(sortBy, sortDir *string) string {
	col := "assessment_created_at"
	if sortBy != nil {
		switch strings.ToLower(strings.TrimSpace(*sortBy)) {
		case "title":
			col = "assessment_title"
		case "start_at":
			col = "assessment_start_at"
		case "due_at":
			col = "assessment_due_at"
		case "created_at":
			col = "assessment_created_at"
		}
	}
	dir := "DESC"
	if sortDir != nil && strings.EqualFold(strings.TrimSpace(*sortDir), "asc") {
		dir = "ASC"
	}
	return col + " " + dir
}

// GET /assessments
// Query (opsional):
//
//	type_id, csst_id, is_published, q, limit, offset, sort_by, sort_dir
//	with_urls, urls_published_only, urls_limit_per, urls_order
//	include=types (untuk embed object type per item)
func (ctl *AssessmentController) List(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	// 1) Resolve masjid context
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// slug → id (jika perlu)
	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return helper.JsonError(c, helperAuth.ErrMasjidContextMissing.Code, helperAuth.ErrMasjidContextMissing.Message)
	}

	// 2) Authorize: minimal member masjid (any role)
	if !helperAuth.UserHasMasjid(c, mid) {
		return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak terdaftar pada masjid ini (membership).")
	}

	// 3) Query parameters
	var (
		typeIDStr = strings.TrimSpace(c.Query("type_id"))
		csstIDStr = strings.TrimSpace(c.Query("csst_id"))
		qStr      = strings.TrimSpace(c.Query("q"))
		isPubStr  = strings.TrimSpace(c.Query("is_published"))
		limit     = atoiOr(20, c.Query("limit"))
		offset    = atoiOr(0, c.Query("offset"))
		sortBy    = strings.TrimSpace(c.Query("sort_by"))
		sortDir   = strings.TrimSpace(c.Query("sort_dir"))
	)

	// include flags
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includes := map[string]bool{}
	for _, p := range strings.Split(includeStr, ",") {
		if x := strings.TrimSpace(p); x != "" {
			includes[x] = true
		}
	}
	wantTypes := includeAll || includes["type"] || includes["types"] || eqTrue(c.Query("with_types"))

	// opsi URL (metadata saja)
	withURLs := eqTrue(c.Query("with_urls"))
	urlsPublishedOnly := eqTrue(c.Query("urls_published_only"))
	urlsLimitPer := atoiOr(0, c.Query("urls_limit_per")) // 0 = tanpa batas

	// parse filter id
	var typeID, csstID *uuid.UUID
	if typeIDStr != "" {
		if u, e := uuid.Parse(typeIDStr); e == nil {
			typeID = &u
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "type_id tidak valid")
		}
	}
	if csstIDStr != "" {
		if u, e := uuid.Parse(csstIDStr); e == nil {
			csstID = &u
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "csst_id tidak valid")
		}
	}

	// filter boolean
	var isPublished *bool
	if isPubStr != "" {
		b := strings.EqualFold(isPubStr, "true") || isPubStr == "1"
		isPublished = &b
	}

	// sorting
	var sbPtr, sdPtr *string
	if sortBy != "" {
		sbPtr = &sortBy
	}
	if sortDir != "" {
		sdPtr = &sortDir
	}

	// 4) Base query (singular columns)
	qry := ctl.DB.WithContext(c.Context()).
		Model(&model.AssessmentModel{}).
		Where("assessment_masjid_id = ?", mid)

	if typeID != nil {
		qry = qry.Where("assessment_type_id = ?", *typeID)
	}
	if csstID != nil {
		qry = qry.Where("assessment_class_section_subject_teacher_id = ?", *csstID)
	}
	if isPublished != nil {
		qry = qry.Where("assessment_is_published = ?", *isPublished)
	}
	if qStr != "" {
		q := "%" + strings.ToLower(qStr) + "%"
		qry = qry.Where(
			"(LOWER(assessment_title) LIKE ? OR LOWER(COALESCE(assessment_description, '')) LIKE ?)",
			q, q,
		)
	}

	// total
	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// page data
	var rows []model.AssessmentModel
	if err := qry.
		Order(getSortClauseAssessment(sbPtr, sdPtr)).
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 5) Build response DTO
	type typeLite struct {
		ID            uuid.UUID `json:"id"             gorm:"column:assessment_type_id"`
		Key           string    `json:"key"            gorm:"column:assessment_type_key"`
		Name          string    `json:"name"           gorm:"column:assessment_type_name"`
		WeightPercent float64   `json:"weight_percent" gorm:"column:assessment_type_weight_percent"`
		IsActive      bool      `json:"is_active"      gorm:"column:assessment_type_is_active"`
	}
	type assessmentWithExpand struct {
		dto.AssessmentResponse
		Type      *typeLite `json:"type,omitempty"`
		URLsCount *int      `json:"urls_count,omitempty"`
	}

	out := make([]assessmentWithExpand, 0, len(rows))
	for i := range rows {
		out = append(out, assessmentWithExpand{
			AssessmentResponse: dto.FromModelAssesment(rows[i]),
		})
	}

	// kumpulkan type unik
	typeIDs := make([]uuid.UUID, 0, len(rows))
	seenType := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		if rows[i].AssessmentTypeID == nil {
			continue
		}
		tid := *rows[i].AssessmentTypeID
		if _, ok := seenType[tid]; ok {
			continue
		}
		seenType[tid] = struct{}{}
		typeIDs = append(typeIDs, tid)
	}

	// fetch type batch
	typeMap := make(map[uuid.UUID]typeLite, len(typeIDs))
	if len(typeIDs) > 0 {
		var trows []typeLite
		if err := ctl.DB.WithContext(c.Context()).
			Table("assessment_types").
			Select(`
				assessment_type_id,
				assessment_type_key,
				assessment_type_name,
				(assessment_type_weight_percent)::float8 AS assessment_type_weight_percent,
				assessment_type_is_active
			`).
			Where("assessment_type_id IN ? AND assessment_type_masjid_id = ?", typeIDs, mid).
			Scan(&trows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil assessment types")
		}
		for _, t := range trows {
			typeMap[t.ID] = t
		}
	}

	// attach TYPE per item jika diminta
	if wantTypes {
		for i := range rows {
			if rows[i].AssessmentTypeID == nil {
				continue
			}
			if t, ok := typeMap[*rows[i].AssessmentTypeID]; ok {
				tc := t
				out[i].Type = &tc
			}
		}
	}

	// 6) Return response
	return helper.JsonList(c, out, fiber.Map{
		"total":               total,
		"limit":               limit,
		"offset":              offset,
		"with_urls":           withURLs,
		"urls_published_only": urlsPublishedOnly,
		"urls_limit_per":      urlsLimitPer,
		"urls_order":          strings.ToLower(strings.TrimSpace(c.Query("urls_order"))),
	})
}
