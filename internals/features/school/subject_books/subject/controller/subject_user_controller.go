package controller

import (
	subjectDTO "masjidku_backend/internals/features/school/subject_books/subject/dto"
	subjectModel "masjidku_backend/internals/features/school/subject_books/subject/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   LIST
   GET /admin/subjects?q=&is_active=&order_by=&sort=&limit=&offset=&with_deleted=&id=&ids=
   order_by: code|name|created_at|updated_at
   sort: asc|desc
   ========================================================= */
func (h *SubjectsController) ListSubjects(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// --- Query params & defaults ---
	var q subjectDTO.ListSubjectQuery
	q.Limit, q.Offset = intPtr(20), intPtr(0)

	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 {
		q.Limit = intPtr(20)
	}
	if q.Offset == nil || *q.Offset < 0 {
		q.Offset = intPtr(0)
	}

	// --- Base query (tenant + soft delete by default) ---
	tx := h.DB.Model(&subjectModel.SubjectsModel{}).
		Where("subjects_masjid_id = ?", masjidID)

	// exclude soft-deleted by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("subjects_deleted_at IS NULL")
	}

	// ========== NEW: filter by id / ids (comma-separated) ==========
	if ids, ok, errResp := uuidListFromQuery(c, "id", "ids"); errResp != nil {
		return errResp
	} else if ok {
		tx = tx.Where("subjects_id IN ?", ids)
	}

	// filters lain
	if q.IsActive != nil {
		tx = tx.Where("subjects_is_active = ?", *q.IsActive)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("(LOWER(subjects_code) LIKE ? OR LOWER(subjects_name) LIKE ?)", kw, kw)
	}

	// order by whitelist
	orderBy := "subjects_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "code":
			orderBy = "subjects_code"
		case "name":
			orderBy = "subjects_name"
		case "created_at":
			orderBy = "subjects_created_at"
		case "updated_at":
			orderBy = "subjects_updated_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// --- total (sebelum limit/offset) ---
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// --- data ---
	var rows []subjectModel.SubjectsModel
	if err := tx.
		Select(`
			subjects_id,
			subjects_masjid_id,
			subjects_code,
			subjects_name,
			subjects_desc,
			subjects_is_active,
			subjects_created_at,
			subjects_updated_at,
			subjects_deleted_at
		`).
		Order(orderBy + " " + sort).
		Limit(*q.Limit).Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// --- response konsisten: data[] + pagination ---
	return helper.JsonList(c,
		subjectDTO.FromSubjectModels(rows),
		fiber.Map{
			"limit":  *q.Limit,
			"offset": *q.Offset,
			"total":  int(total),
		},
	)
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
