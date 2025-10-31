package controller

import (
	"errors"
	"strings"

	secDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* ================= Helpers ================= */

// parseUUIDList: "a,b,c" → []uuid.UUID (dedupe + validasi)
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
			return nil, errors.New("UUID tidak valid: " + p)
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("daftar kosong")
	}
	return out, nil
}

/* ================= List (simple, snapshot-first) ================= */

// GET /api/{a|u}/:masjid_id/class-sections/list
// Params ringan:
//   - q atau search         : keyword (min 2 char bila pakai q)
//   - is_active             : bool
//   - id                    : comma-separated UUIDs (opsional)
//
// Sort & paging via helper.ParseFiber (sortBy: name|created_at)
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	/* ---------- Masjid context dari path/slug (PUBLIC) ---------- */
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil {
			return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
		return helperAuth.ErrMasjidContextMissing
	}

	/* ---------- Search term ---------- */
	rawQ := strings.TrimSpace(c.Query("q"))
	rawSearch := strings.TrimSpace(c.Query("search"))
	searchTerm := rawSearch
	if rawQ != "" {
		searchTerm = rawQ
		if len([]rune(searchTerm)) < 2 {
			return fiber.NewError(fiber.StatusBadRequest, "Parameter q minimal 2 karakter")
		}
	}

	/* ---------- Paging & sorting (default dinamis) ---------- */
	defaultSortBy := "created_at"
	defaultSortOrder := "desc"
	if searchTerm != "" {
		defaultSortBy = "name"
		defaultSortOrder = "asc"
	}
	p := helper.ParseFiber(c, defaultSortBy, defaultSortOrder, helper.AdminOpts)

	// whitelist kolom → nama kolom DB
	allowed := map[string]string{
		"name":       "class_section_name",
		"created_at": "class_section_created_at",
	}
	orderClause, _ := (helper.Params{SortBy: p.SortBy, SortOrder: p.SortOrder}).SafeOrderClause(allowed, defaultSortBy)
	orderClause = strings.TrimPrefix(orderClause, "ORDER BY ")

	/* ---------- Filters ringan ---------- */
	var (
		sectionIDs []uuid.UUID
		activeOnly *bool
	)
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		ids, e := parseUUIDList(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "id tidak valid: "+e.Error())
		}
		sectionIDs = ids
	}
	if s := strings.TrimSpace(c.Query("is_active")); s != "" {
		v := c.QueryBool("is_active")
		activeOnly = &v
	}

	/* ---------- Base query (single-tenant via masjid context) ---------- */
	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_section_deleted_at IS NULL").
		Where("class_section_masjid_id = ?", masjidID)

	if len(sectionIDs) > 0 {
		tx = tx.Where("class_section_id IN ?", sectionIDs)
	}
	if activeOnly != nil {
		tx = tx.Where("class_section_is_active = ?", *activeOnly)
	}
	if searchTerm != "" {
		s := "%" + strings.ToLower(searchTerm) + "%"
		tx = tx.Where(`
			LOWER(class_section_name) LIKE ?
			OR LOWER(class_section_code) LIKE ?
			OR LOWER(class_section_slug) LIKE ?`,
			s, s, s)
	}

	/* ---------- Total ---------- */
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	/* ---------- Data ---------- */
	if orderClause != "" {
		tx = tx.Order(orderClause)
	}
	if !p.All {
		tx = tx.Limit(p.Limit()).Offset(p.Offset())
	}

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	/* ---------- Build response dari snapshot DTO ---------- */
	/* ---------- Build response dari snapshot DTO (tanpa "data") ---------- */
	items := make([]secDTO.ClassSectionResponse, 0, len(rows))
	for i := range rows {
		items = append(items, secDTO.FromModelClassSection(&rows[i]))
	}

	meta := helper.BuildMeta(total, p)

	// langsung kirim tanpa wrapper "data"
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": items,
		"meta": meta,
	})

}
