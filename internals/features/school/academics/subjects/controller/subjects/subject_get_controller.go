// file: internals/features/lembaga/subjects/main/controller/subjects_controller_list.go
package controller

import (
	"errors"
	"strings"

	subjectDTO "madinahsalam_backend/internals/features/school/academics/subjects/dto"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================
LIST
GET /admin/subjects?q=&is_active=&order_by=&sort=&limit=&offset=&with_deleted=&id=&ids=&mode=
order_by: code|name|created_at|updated_at
sort: asc|desc
mode: full|compact (default: full)
=========================================================
*/
func (h *SubjectsController) List(c *fiber.Ctx) error {
	// Kalau ada helper lain yang butuh DB di Locals
	c.Locals("DB", h.DB)

	// =====================================================
	// 0) Mode response: full vs compact
	// =====================================================
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode", "full")))
	if mode != "compact" {
		mode = "full"
	}

	// =====================================================
	// 1) Tentukan schoolID:
	//    - Prioritas: dari token (GetSchoolIDFromTokenPreferTeacher)
	//    - Fallback: dari ResolveSchoolContext (id / slug)
	// =====================================================
	var schoolID uuid.UUID

	// 1. Coba dulu dari token
	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		// 2. Kalau nggak dapat dari token → pakai context multi-sumber
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			// balas dengan skema error JSON standar
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		if mc.ID != uuid.Nil {
			schoolID = mc.ID
		} else if s := strings.TrimSpace(mc.Slug); s != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, s)
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, helperAuth.ErrSchoolContextMissing.Error())
		}
	}

	// --- Query params & defaults ---
	var q subjectDTO.ListSubjectQuery
	q.Limit, q.Offset = intPtr(20), intPtr(0)

	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 {
		q.Limit = intPtr(20)
	}
	if q.Offset == nil || *q.Offset < 0 {
		q.Offset = intPtr(0)
	}

	// --- Base query (tenant + soft delete by default) ---
	tx := h.DB.WithContext(c.Context()).
		Table("subjects AS s").
		Where("s.subject_school_id = ?", schoolID)

	// exclude soft-deleted by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("s.subject_deleted_at IS NULL")
	}

	// ========== filter by id / ids (comma-separated) ==========
	if ids, ok, errResp := uuidListFromQuery(c, "id", "ids"); errResp != nil {
		// errResp sudah berupa fiber.Error; bungkus ke JsonError
		if fe, ok := errResp.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, errResp.Error())
	} else if ok {
		tx = tx.Where("s.subject_id IN ?", ids)
	}

	// filters lain
	if q.IsActive != nil {
		tx = tx.Where("s.subject_is_active = ?", *q.IsActive)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("(LOWER(s.subject_code) LIKE ? OR LOWER(s.subject_name) LIKE ?)", kw, kw)
	}
	// 🔍 filter spesifik by subject_name: ?name=
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Name)) + "%"
		tx = tx.Where("LOWER(s.subject_name) LIKE ?", kw)
	}

	// order by whitelist → map ke kolom fisik
	orderCol := "s.subject_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(strings.TrimSpace(*q.OrderBy)) {
		case "code":
			orderCol = "s.subject_code"
		case "name":
			orderCol = "s.subject_name"
		case "created_at":
			orderCol = "s.subject_created_at"
		case "updated_at":
			orderCol = "s.subject_updated_at"
		}
	}
	sortDir := "ASC"
	if q.Sort != nil && strings.ToLower(strings.TrimSpace(*q.Sort)) == "desc" {
		sortDir = "DESC"
	}
	orderExpr := orderCol + " " + sortDir

	// --- total (sebelum limit/offset) ---
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// --- data (pakai DTO low-level: SubjectRow + SubjectListSelectColumns) ---
	// prefix alias "s." ke setiap kolom
	selectCols := make([]string, 0, len(subjectDTO.SubjectListSelectColumns))
	for _, col := range subjectDTO.SubjectListSelectColumns {
		selectCols = append(selectCols, "s."+col)
	}

	var rows []subjectDTO.SubjectRow
	if err := tx.
		Select(strings.Join(selectCols, ",")).
		Order(orderExpr).
		Limit(*q.Limit).
		Offset(*q.Offset).
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// --- pagination meta konsisten ---
	meta := helper.BuildPaginationFromOffset(total, *q.Offset, *q.Limit)

	// --- response sesuai mode ---
	if mode == "compact" {
		return helper.JsonList(c, "ok", subjectDTO.SubjectRowsToCompactResponses(rows), meta)
	}

	// default: full
	return helper.JsonList(c, "ok", subjectDTO.SubjectRowsToResponsesWithSchoolTime(c, rows), meta)
}

/* ================= Helpers lokal ================= */

// baca "id" atau "ids" (comma-separated). Prioritas "id" lalu "ids".
// return: (ids, found, errorResponse)
func uuidListFromQuery(c *fiber.Ctx, keys ...string) ([]uuid.UUID, bool, error) {
	for _, k := range keys {
		raw := strings.TrimSpace(c.Query(k))
		if raw == "" {
			continue
		}
		ids, err := parseUUIDList(raw)
		if err != nil {
			return nil, false, fiber.NewError(fiber.StatusBadRequest, k+" tidak valid: "+err.Error())
		}
		return ids, true, nil
	}
	return nil, false, nil
}

// "a,b,c" -> []uuid.UUID (dedupe + validasi)
func parseUUIDList(s string) ([]uuid.UUID, error) {
	parts := strings.Split(s, ",")
	seen := make(map[uuid.UUID]struct{}, len(parts))
	out := make([]uuid.UUID, 0, len(parts))
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

func intPtr(v int) *int { return &v }
