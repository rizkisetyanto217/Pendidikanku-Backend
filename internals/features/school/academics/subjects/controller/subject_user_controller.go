// file: internals/features/lembaga/subjects/main/controller/subjects_controller_list.go
package controller

import (
	"errors"
	subjectDTO "schoolku_backend/internals/features/school/academics/subjects/dto"
	subjectModel "schoolku_backend/internals/features/school/academics/subjects/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================
LIST
GET /admin/subjects?q=&is_active=&order_by=&sort=&limit=&offset=&with_deleted=&id=&ids=
order_by: code|name|created_at|updated_at
sort: asc|desc
=========================================================
*/
func (h *SubjectsController) List(c *fiber.Ctx) error {
	// Kalau ada helper lain yang butuh DB di Locals
	c.Locals("DB", h.DB)

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
	tx := h.DB.Model(&subjectModel.SubjectModel{}).
		Where("subject_school_id = ?", schoolID)

	// exclude soft-deleted by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("subject_deleted_at IS NULL")
	}

	// ========== filter by id / ids (comma-separated) ==========
	if ids, ok, errResp := uuidListFromQuery(c, "id", "ids"); errResp != nil {
		// errResp sudah berupa fiber.Error; bungkus ke JsonError
		if fe, ok := errResp.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, errResp.Error())
	} else if ok {
		tx = tx.Where("subject_id IN ?", ids)
	}

	// filters lain
	if q.IsActive != nil {
		tx = tx.Where("subject_is_active = ?", *q.IsActive)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("(LOWER(subject_code) LIKE ? OR LOWER(subject_name) LIKE ?)", kw, kw)
	}

	// order by whitelist
	orderBy := "subject_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "code":
			orderBy = "subject_code"
		case "name":
			orderBy = "subject_name"
		case "created_at":
			orderBy = "subject_created_at"
		case "updated_at":
			orderBy = "subject_updated_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// --- total (sebelum limit/offset) ---
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// --- data ---
	var rows []subjectModel.SubjectModel
	if err := tx.
		Select(`
			subject_id,
			subject_school_id,
			subject_code,
			subject_name,
			subject_desc,
			subject_slug,
			subject_image_url,
			subject_image_object_key,
			subject_image_url_old,
			subject_image_object_key_old,
			subject_image_delete_pending_until,
			subject_is_active,
			subject_created_at,
			subject_updated_at,
			subject_deleted_at
		`).
		Order(orderBy + " " + sort).
		Limit(*q.Limit).Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// --- pagination meta konsisten ---
	meta := helper.BuildPaginationFromOffset(total, *q.Offset, *q.Limit)

	// --- response standar ---
	return helper.JsonList(c, "ok", subjectDTO.FromSubjectModels(rows), meta)
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
