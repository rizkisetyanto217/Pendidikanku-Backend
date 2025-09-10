// file: internals/features/school/assessments/controller/assessment_controller.go
package controller

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* ========================================================
   Controller
======================================================== */
type AssessmentController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentController(db *gorm.DB) *AssessmentController {
	return &AssessmentController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ========================================================
   Helpers
======================================================== */

// Ambil masjid_id dari token dengan preferensi TEACHER (konsisten di semua route)
func requireMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return uuid.Nil, helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	if mid == uuid.Nil {
		return uuid.Nil, helper.JsonError(c, fiber.StatusForbidden, "Tidak ada akses masjid")
	}
	return mid, nil
}

// helpers kecil lokal
func atoiOr(def int, s string) int {
	if s == "" {
		return def
	}
	n := 0
	sign := 1
	for i := 0; i < len(s); i++ {
		if i == 0 && s[i] == '-' {
			sign = -1
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return def
		}
		n = n*10 + int(s[i]-'0')
	}
	n *= sign
	if n <= 0 {
		return def
	}
	return n
}

func eqTrue(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true"
}




// toResponse memetakan model -> DTO respons
func toResponse(m *model.AssessmentModel) dto.AssessmentResponse {
	var deletedAt *time.Time
	if m.AssessmentsDeletedAt.Valid {
		t := m.AssessmentsDeletedAt.Time
		deletedAt = &t
	}

	return dto.AssessmentResponse{
		AssessmentsID:                           m.AssessmentsID,
		AssessmentsMasjidID:                     m.AssessmentsMasjidID,
		AssessmentsClassSectionSubjectTeacherID: m.AssessmentsClassSectionSubjectTeacherID,
		AssessmentsTypeID:                       m.AssessmentsTypeID,

		AssessmentsTitle:       m.AssessmentsTitle,
		AssessmentsDescription: m.AssessmentsDescription,

		AssessmentsStartAt: m.AssessmentsStartAt,
		AssessmentsDueAt:   m.AssessmentsDueAt,

		AssessmentsMaxScore: m.AssessmentsMaxScore,

		AssessmentsIsPublished:     m.AssessmentsIsPublished,
		AssessmentsAllowSubmission: m.AssessmentsAllowSubmission,

		AssessmentsCreatedByTeacherID: m.AssessmentsCreatedByTeacherID,

		AssessmentsCreatedAt: m.AssessmentsCreatedAt,
		AssessmentsUpdatedAt: m.AssessmentsUpdatedAt,
		AssessmentsDeletedAt: deletedAt,
	}
}

// Validasi opsional: created_by_teacher_id (jika ada) memang milik masjid
func (ctl *AssessmentController) assertTeacherBelongsToMasjid(
    ctx context.Context,
    masjidID uuid.UUID,
    teacherID *uuid.UUID,
) error {
    if teacherID == nil || *teacherID == uuid.Nil {
        return nil
    }
    var n int64
    // Pastikan nama kolom sesuai skema mu:
    // umumnya: masjid_teacher_id, masjid_teacher_masjid_id, masjid_teacher_deleted_at
    if err := ctl.DB.WithContext(ctx).
        Table("masjid_teachers").
        Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
            *teacherID, masjidID).
        Count(&n).Error; err != nil {
        return err
    }
    if n == 0 {
        return errors.New("assessments_created_by_teacher_id bukan milik masjid ini")
    }
    return nil
}

/* ========================================================
   Handlers
======================================================== */
// GET /assessments
// Query (opsional):
//   type_id, csst_id, is_published, q, limit, offset, sort_by, sort_dir
//   with_urls, urls_published_only, urls_limit_per, urls_order
//   include=types (untuk embed object type per item)
func (ctl *AssessmentController) List(c *fiber.Ctx) error {
	masjidID, err := requireMasjidID(c)
	if err != nil {
		return err
	}

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
		Where("assessments_masjid_id = ?", masjidID)

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
			Where("assessment_types_id IN ? AND assessment_types_masjid_id = ?", typeIDs, masjidID).
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


// GET /assessments/:id
func (ctl *AssessmentController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessments_id tidak valid")
	}

	masjidID, err := requireMasjidID(c)
	if err != nil {
		return err
	}

	var row model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "OK", toResponse(&row))
}

// POST /assessments
func (ctl *AssessmentController) Create(c *fiber.Ctx) error {
	var req dto.CreateAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	masjidID, err := requireMasjidID(c)
	if err != nil { return err }
	req.AssessmentsMasjidID = masjidID

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// validasi creator (opsional)
	if err := ctl.assertTeacherBelongsToMasjid(c.Context(), req.AssessmentsMasjidID, req.AssessmentsCreatedByTeacherID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// validasi waktu: due >= start (kalau keduanya ada)
	if req.AssessmentsStartAt != nil && req.AssessmentsDueAt != nil &&
		req.AssessmentsDueAt.Before(*req.AssessmentsStartAt) {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessments_due_at harus setelah atau sama dengan assessments_start_at")
	}

	now := time.Now()

	row := model.AssessmentModel{
		// ❌ JANGAN set AssessmentsID ke uuid.Nil, biarkan DB atau app yang generate
		// ✅ Opsi A (biar DB generate): pastikan kolom punya DEFAULT gen_random_uuid() dan tag gorm default
		// ✅ Opsi B (app-generate): aktifkan baris di bawah ini dan pastikan kolom tidak override
		AssessmentsID: uuid.New(),

		AssessmentsMasjidID:                     req.AssessmentsMasjidID,
		AssessmentsClassSectionSubjectTeacherID: req.AssessmentsClassSectionSubjectTeacherID,
		AssessmentsTypeID:                       req.AssessmentsTypeID,

		AssessmentsTitle:       strings.TrimSpace(req.AssessmentsTitle),
		AssessmentsDescription: nil,

		AssessmentsMaxScore:        100,
		AssessmentsIsPublished:     true,
		AssessmentsAllowSubmission: true,

		AssessmentsCreatedByTeacherID: req.AssessmentsCreatedByTeacherID,

		AssessmentsCreatedAt: now,
		AssessmentsUpdatedAt: now,
	}

	// optional fields
	if req.AssessmentsDescription != nil {
		if d := strings.TrimSpace(*req.AssessmentsDescription); d != "" {
			row.AssessmentsDescription = &d
		}
	}
	if req.AssessmentsMaxScore != nil {
		row.AssessmentsMaxScore = *req.AssessmentsMaxScore
	}
	if req.AssessmentsIsPublished != nil {
		row.AssessmentsIsPublished = *req.AssessmentsIsPublished
	}
	if req.AssessmentsAllowSubmission != nil {
		row.AssessmentsAllowSubmission = *req.AssessmentsAllowSubmission
	}
	// ⬇️ pointer → value
	if req.AssessmentsStartAt != nil {
		row.AssessmentsStartAt = req.AssessmentsStartAt
	}
	if req.AssessmentsDueAt != nil {
		row.AssessmentsDueAt = req.AssessmentsDueAt
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat assessment")
	}

	// Safety: kalau masih nol (misal default DB tidak terset), fallback generate di app (opsional)
	if row.AssessmentsID == uuid.Nil {
		row.AssessmentsID = uuid.New()
		_ = ctl.DB.WithContext(c.Context()).
			Model(&model.AssessmentModel{}).
			Where("assessments_created_at = ? AND assessments_title = ? AND assessments_masjid_id = ?",
				row.AssessmentsCreatedAt, row.AssessmentsTitle, row.AssessmentsMasjidID).
			Update("assessments_id", row.AssessmentsID).Error
	}

	return helper.JsonCreated(c, "Assessment berhasil dibuat", toResponse(&row))
}


// PATCH /assessments/:id (partial)
// PATCH /assessments/:id (partial)
func (ctl *AssessmentController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessments_id tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	masjidID, err := requireMasjidID(c)
	if err != nil {
		return err
	}

	var existing model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Validasi guru jika diubah
	if req.AssessmentsCreatedByTeacherID != nil {
		if err := ctl.assertTeacherBelongsToMasjid(
			c.Context(),
			masjidID,
			req.AssessmentsCreatedByTeacherID,
		); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// ===== Validasi waktu bila diubah =====
	switch {
	case req.AssessmentsStartAt != nil && req.AssessmentsDueAt != nil:
		// Keduanya diubah → due >= start
		if req.AssessmentsDueAt.Before(*req.AssessmentsStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessments_due_at harus setelah atau sama dengan assessments_start_at")
		}
	case req.AssessmentsStartAt != nil && req.AssessmentsDueAt == nil:
		// Start diubah saja → pastikan due existing (jika ada) tidak sebelum start baru
		if existing.AssessmentsDueAt != nil && existing.AssessmentsDueAt.Before(*req.AssessmentsStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Tanggal due saat ini lebih awal dari start baru")
		}
	case req.AssessmentsStartAt == nil && req.AssessmentsDueAt != nil:
		// Due diubah saja → pastikan due baru tidak sebelum start existing
		if existing.AssessmentsStartAt != nil && req.AssessmentsDueAt.Before(*existing.AssessmentsStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessments_due_at tidak boleh sebelum assessments_start_at")
		}
	}

	// ===== Build updates =====
	updates := map[string]interface{}{}

	if req.AssessmentsTitle != nil {
		updates["assessments_title"] = strings.TrimSpace(*req.AssessmentsTitle)
	}
	if req.AssessmentsDescription != nil {
		updates["assessments_description"] = strings.TrimSpace(*req.AssessmentsDescription)
	}
	if req.AssessmentsStartAt != nil {
		// model pakai *time.Time, tapi gorm bisa terima value juga
		updates["assessments_start_at"] = *req.AssessmentsStartAt
	}
	if req.AssessmentsDueAt != nil {
		updates["assessments_due_at"] = *req.AssessmentsDueAt
	}
	if req.AssessmentsMaxScore != nil {
		updates["assessments_max_score"] = *req.AssessmentsMaxScore
	}
	if req.AssessmentsIsPublished != nil {
		updates["assessments_is_published"] = *req.AssessmentsIsPublished
	}
	if req.AssessmentsAllowSubmission != nil {
		updates["assessments_allow_submission"] = *req.AssessmentsAllowSubmission
	}
	if req.AssessmentsTypeID != nil {
		updates["assessments_type_id"] = *req.AssessmentsTypeID
	}
	if req.AssessmentsClassSectionSubjectTeacherID != nil {
		updates["assessments_class_section_subject_teacher_id"] = *req.AssessmentsClassSectionSubjectTeacherID
	}
	if req.AssessmentsCreatedByTeacherID != nil {
		updates["assessments_created_by_teacher_id"] = *req.AssessmentsCreatedByTeacherID
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", toResponse(&existing))
	}
	updates["assessments_updated_at"] = time.Now()

	if err := ctl.DB.WithContext(c.Context()).
		Model(&model.AssessmentModel{}).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidID).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
	}

	var after model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidID).
		First(&after).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat ulang assessment")
	}

	return helper.JsonUpdated(c, "Assessment berhasil diperbarui", toResponse(&after))
}

// DELETE /assessments/:id (soft delete)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessments_id tidak valid")
	}

	masjidID, err := requireMasjidID(c)
	if err != nil {
		return err
	}

	var row model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus assessment")
	}

	return helper.JsonDeleted(c, "Assessment dihapus", fiber.Map{
		"assessments_id": id,
	})
}
