package controller

import (
	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* ========================================================
   Handlers
======================================================== */
// GET /assessments
// Query (opsional):
//   type_id, csst_id, is_published, q, limit, offset, sort_by, sort_dir
//   with_urls, urls_published_only, urls_limit_per, urls_order
//   include=types (untuk embed object type per item)
func (ctl *AssessmentController) List(c *fiber.Ctx) error {

	// ambil masjid_id prefer teacher
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	// authorize: anggota masjid (semua role)
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil { return err }


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

	// --- include flags ---
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includes := map[string]bool{}
	for _, p := range strings.Split(includeStr, ",") {
		if x := strings.TrimSpace(p); x != "" {
			includes[x] = true
		}
	}
	wantTypes := includeAll || includes["type"] || includes["types"] || eqTrue(c.Query("with_types"))

	// --- opsi URL ---
	withURLs := eqTrue(c.Query("with_urls"))
	urlsPublishedOnly := eqTrue(c.Query("urls_published_only"))
	urlsLimitPer := atoiOr(0, c.Query("urls_limit_per")) // 0 = tanpa batas

	// whitelist order untuk URLs
	urlsOrderRaw := strings.ToLower(strings.TrimSpace(c.Query("urls_order")))
	urlsOrderCol := "assessment_urls_created_at"
	urlsOrderDir := "DESC"
	if urlsOrderRaw != "" {
		parts := strings.Fields(urlsOrderRaw) // ex: "published_at asc"
		if len(parts) >= 1 {
			switch parts[0] {
			case "created_at":
				urlsOrderCol = "assessment_urls_created_at"
			case "published_at":
				urlsOrderCol = "assessment_urls_published_at"
			}
		}
		if len(parts) >= 2 && (parts[1] == "asc" || parts[1] == "desc") {
			urlsOrderDir = strings.ToUpper(parts[1])
		}
	}
	urlsOrderClause := urlsOrderCol + " " + urlsOrderDir

	// --- parse filter id ---
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

	// --- filter boolean ---
	var isPublished *bool
	if isPubStr != "" {
		b := strings.EqualFold(isPubStr, "true") || isPubStr == "1"
		isPublished = &b
	}

	// --- sorting ---
	var sbPtr, sdPtr *string
	if sortBy != "" {
		sbPtr = &sortBy
	}
	if sortDir != "" {
		sdPtr = &sortDir
	}

	// --- base query ---
	qry := ctl.DB.WithContext(c.Context()).
		Model(&model.AssessmentModel{}).
		Where("assessments_masjid_id = ?", mid)

	if typeID != nil {
		qry = qry.Where("assessments_type_id = ?", *typeID)
	}
	if csstID != nil {
		qry = qry.Where("assessments_class_section_subject_teacher_id = ?", *csstID)
	}
	if isPublished != nil {
		qry = qry.Where("assessments_is_published = ?", *isPublished)
	}
	if qStr != "" {
		q := "%" + strings.ToLower(qStr) + "%"
		qry = qry.Where("(LOWER(assessments_title) LIKE ? OR LOWER(COALESCE(assessments_description, '')) LIKE ?)", q, q)
	}

	// --- total ---
	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// --- page data ---
	var rows []model.AssessmentModel
	if err := qry.
		Order(getSortClause(sbPtr, sdPtr)).
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// --- response skeleton (embed DTO + optional expand) ---
	type typeLite struct {
		ID            uuid.UUID `json:"id"             gorm:"column:assessment_types_id"`
		Key           string    `json:"key"            gorm:"column:assessment_types_key"`
		Name          string    `json:"name"           gorm:"column:assessment_types_name"`
		WeightPercent float64   `json:"weight_percent" gorm:"column:assessment_types_weight_percent"` // float64 supaya aman
		IsActive      bool      `json:"is_active"      gorm:"column:assessment_types_is_active"`
	}
	type assessmentWithExpand struct {
		dto.AssessmentResponse
		Type      *typeLite                   `json:"type,omitempty"`
		URLs      []model.AssessmentUrlsModel `json:"urls,omitempty"`
		URLsCount *int                        `json:"urls_count,omitempty"`
	}

	out := make([]assessmentWithExpand, 0, len(rows))
	for i := range rows {
		out = append(out, assessmentWithExpand{AssessmentResponse: toResponse(&rows[i])})
	}

	// --- kumpulkan TYPE unik dari page ini ---
	typeIDs := make([]uuid.UUID, 0, len(rows))
	seenType := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		if rows[i].AssessmentsTypeID == nil {
			continue
		}
		tid := *rows[i].AssessmentsTypeID
		if _, ok := seenType[tid]; ok {
			continue
		}
		seenType[tid] = struct{}{}
		typeIDs = append(typeIDs, tid)
	}

	// --- fetch TYPE batch (cast weight_percent → float8 agar scan → float64 mulus) ---
	typeMap := make(map[uuid.UUID]typeLite, len(typeIDs))
	if len(typeIDs) > 0 {
		var trows []typeLite
		if err := ctl.DB.WithContext(c.Context()).
			Table("assessment_types").
			Select(`
				assessment_types_id,
				assessment_types_key,
				assessment_types_name,
				(assessment_types_weight_percent)::float8 AS assessment_types_weight_percent,
				assessment_types_is_active`).
			Where("assessment_types_id IN ? AND assessment_types_masjid_id = ?", typeIDs, mid).
			Scan(&trows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil assessment types")
		}
		for _, t := range trows {
			typeMap[t.ID] = t
		}
	}

	// --- attach TYPE per item jika diminta ---
	if wantTypes {
		for i := range rows {
			if rows[i].AssessmentsTypeID == nil {
				continue
			}
			if t, ok := typeMap[*rows[i].AssessmentsTypeID]; ok {
				tc := t
				out[i].Type = &tc
			}
		}
	}

	// --- URLs (batch, tanpa N+1) ---
	if withURLs && len(rows) > 0 {
		ids := make([]uuid.UUID, len(rows))
		indexByID := make(map[uuid.UUID]int, len(rows))
		for i := range rows {
			ids[i] = rows[i].AssessmentsID
			indexByID[rows[i].AssessmentsID] = i
		}

		uqry := ctl.DB.WithContext(c.Context()).
			Model(&model.AssessmentUrlsModel{}).
			Where("assessment_urls_deleted_at IS NULL").
			Where("assessment_urls_assessment_id IN ?", ids)

		if urlsPublishedOnly {
			uqry = uqry.Where("assessment_urls_is_published = ? AND assessment_urls_is_active = ?", true, true)
		}
		uqry = uqry.Order(urlsOrderClause)

		var urlRows []model.AssessmentUrlsModel
		if err := uqry.Find(&urlRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil URL")
		}

		group := make(map[uuid.UUID][]model.AssessmentUrlsModel, len(ids))
		for i := range urlRows {
			aID := urlRows[i].AssessmentUrlsAssessmentID
			group[aID] = append(group[aID], urlRows[i])
		}

		if urlsLimitPer > 0 {
			for k, arr := range group {
				if len(arr) > urlsLimitPer {
					group[k] = arr[:urlsLimitPer]
				}
			}
		}

		for aID, arr := range group {
			if idx, ok := indexByID[aID]; ok {
				out[idx].URLs = arr
				cnt := len(arr)
				out[idx].URLsCount = &cnt
			}
		}
	}

	// --- ringkasan types untuk meta (unik per page) ---
	typeList := make([]typeLite, 0, len(typeMap))
	for _, t := range typeMap {
		typeList = append(typeList, t)
	}
	sort.Slice(typeList, func(i, j int) bool { return strings.ToLower(typeList[i].Name) < strings.ToLower(typeList[j].Name) })

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
