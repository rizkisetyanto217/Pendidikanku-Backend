// internals/features/lembaga/classes/user_classes/main/controller/user_my_class_controller.go
package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/school/classes/classes/dto"
	"masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

// GET /admin/classes/slug/:slug
func (ctrl *ClassController) GetClassBySlug(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// slugify ringan (lower + trim + safe)
	slug := helper.GenerateSlug(c.Params("slug"))

	var m model.ClassModel
	if err := ctrl.DB.
		Where(`
			class_masjid_id = ?
			AND lower(class_slug) = lower(?)
			AND class_deleted_at IS NULL
			AND class_delete_pending_until IS NULL
		`, masjidID, slug).
		First(&m).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Data diterima", dto.FromModel(&m))
}

// GET /admin/classes  (public-friendly; auth optional utk list)
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	// 1) Ambil tenant scope dari token jika ada
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil || len(masjidIDs) == 0 {
		masjidIDs = []uuid.UUID{}

		// ?masjid_id=... (comma-separated)
		if rawIDs := strings.TrimSpace(c.Query("masjid_id")); rawIDs != "" {
			for _, part := range strings.Split(rawIDs, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				id, e := uuid.Parse(part)
				if e != nil {
					return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id tidak valid")
				}
				masjidIDs = append(masjidIDs, id)
			}
		}

		// ?masjid_slug=... (comma-separated)
		if len(masjidIDs) == 0 {
			if rawSlugs := strings.TrimSpace(c.Query("masjid_slug")); rawSlugs != "" {
				slugs := make([]string, 0, 4)
				for _, s := range strings.Split(rawSlugs, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						slugs = append(slugs, s)
					}
				}
				if len(slugs) > 0 {
					var ids []uuid.UUID
					if err := ctrl.DB.
						Table("masjids").
						Select("masjid_id").
						Where("masjid_slug IN ?", slugs).
						Where("masjid_deleted_at IS NULL").
						Scan(&ids).Error; err != nil {
						return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membaca masjid_slug")
					}
					masjidIDs = append(masjidIDs, ids...)
				}
			}
		}

		if len(masjidIDs) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Tanpa auth, wajib kirim masjid_id atau masjid_slug")
		}
	}

	// 2) Parse filter (sesuai DTO baru)
	var q dto.ListClassQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	q.Normalize()

	// 3) Pagination & sorting via helper.ParseWith (AdminOpts)
	rawQuery := string(c.Request().URI().QueryString())
	httpReq, _ := http.NewRequest("GET", "http://local?"+rawQuery, nil)
	p := helper.ParseWith(httpReq, "created_at", "desc", helper.AdminOpts)

	// 4) Base query (tenant-safe, alive only)
	tx := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_masjid_id IN ?", masjidIDs).
		Where("class_deleted_at IS NULL").
		Where("class_delete_pending_until IS NULL")

	// 5) Apply filters (DDL baru)
	if q.ParentID != nil {
		tx = tx.Where("class_parent_id = ?", *q.ParentID)
	}
	if q.TermID != nil {
		tx = tx.Where("class_term_id = ?", *q.TermID)
	}
	if q.IsOpen != nil {
		tx = tx.Where("class_is_open = ?", *q.IsOpen)
	}
	if q.IsActive != nil {
		tx = tx.Where("class_is_active = ?", *q.IsActive)
	}
	if q.DeliveryMode != nil && strings.TrimSpace(*q.DeliveryMode) != "" {
		tx = tx.Where("LOWER(class_delivery_mode) = LOWER(?)", strings.TrimSpace(*q.DeliveryMode))
	}
	if q.Slug != nil && strings.TrimSpace(*q.Slug) != "" {
		tx = tx.Where("LOWER(class_slug) = LOWER(?)", strings.TrimSpace(*q.Slug))
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where(`
			LOWER(COALESCE(class_notes, '')) LIKE ?
			OR LOWER(class_slug) LIKE ?
		`, s, s)
	}
	if q.StartGe != nil {
		tx = tx.Where("class_start_date >= ?", *q.StartGe)
	}
	if q.StartLe != nil {
		tx = tx.Where("class_start_date <= ?", *q.StartLe)
	}
	if q.RegOpenGe != nil {
		tx = tx.Where("class_registration_opens_at >= ?", *q.RegOpenGe)
	}
	if q.RegCloseLe != nil {
		tx = tx.Where("class_registration_closes_at <= ?", *q.RegCloseLe)
	}

	// 6) total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// 7) Sorting whitelist (sesuai kolom baru)
	sortBy := strings.ToLower(strings.TrimSpace(p.SortBy))
	order := strings.ToLower(strings.TrimSpace(p.SortOrder))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	switch sortBy {
	case "slug":
		tx = tx.Order("LOWER(class_slug) " + strings.ToUpper(order)).
			Order("class_created_at DESC")
	case "start_date":
		tx = tx.Order("class_start_date " + strings.ToUpper(order)).
			Order("class_created_at DESC")
	case "is_open":
		tx = tx.Order("class_is_open " + strings.ToUpper(order)).
			Order("class_created_at DESC")
	case "is_active":
		tx = tx.Order("class_is_active " + strings.ToUpper(order)).
			Order("class_created_at DESC")
	case "delivery_mode":
		tx = tx.Order("LOWER(class_delivery_mode) " + strings.ToUpper(order)).
			Order("class_created_at DESC")
	case "reg_open":
		tx = tx.Order("class_registration_opens_at " + strings.ToUpper(order)).
			Order("class_created_at DESC")
	case "reg_close":
		tx = tx.Order("class_registration_closes_at " + strings.ToUpper(order)).
			Order("class_created_at DESC")
	case "created_at":
		fallthrough
	default:
		tx = tx.Order("class_created_at " + strings.ToUpper(order))
	}

	// 8) data + paging
	var rows []model.ClassModel
	if err := tx.
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]dto.ClassResponse, 0, len(rows))
	for i := range rows {
		items = append(items, dto.FromModel(&rows[i]))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}

// GET /admin/classes/search-with-subjects?q=...&limit=&offset=
func (ctl *ClassController) SearchWithSubjects(c *fiber.Ctx) error {
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	q := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	like := "%" + strings.ToLower(q) + "%"

	// ----- base filter: classes x class_subjects x subjects -----
	filter := ctl.DB.Table("classes AS c").
		Joins(`
			JOIN class_subjects AS cs
			  ON cs.class_subjects_class_id = c.class_id
			 AND cs.class_subjects_masjid_id = c.class_masjid_id
			 AND cs.class_subjects_is_active = TRUE
			 AND cs.class_subjects_deleted_at IS NULL
		`).
		Joins(`JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id`).
		Where(`
			c.class_masjid_id IN ?
			AND c.class_deleted_at IS NULL
			AND c.class_delete_pending_until IS NULL
		`, masjidIDs)

	if q != "" {
		filter = filter.Where(`
			LOWER(c.class_slug) LIKE ? OR
			LOWER(COALESCE(c.class_notes,'')) LIKE ? OR
			LOWER(s.subjects_name) LIKE ?
		`, like, like, like)
	}

	// ----- total kelas unik -----
	var total int64
	if err := filter.Session(&gorm.Session{}).
		Distinct("c.class_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	// ----- page of class_ids (urut slug biar stabil)
	type idRow struct {
		ClassID  uuid.UUID `gorm:"column:class_id"`
		ClassSlug string   `gorm:"column:class_slug"`
	}
	var idRows []idRow
	if err := filter.
		Select("c.class_id, c.class_slug").
		Group("c.class_id, c.class_slug").
		Order("LOWER(c.class_slug) ASC").
		Limit(limit).Offset(offset).
		Scan(&idRows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar kelas")
	}
	if len(idRows) == 0 {
		return helper.JsonList(c, []any{}, fiber.Map{"limit": limit, "offset": offset, "total": int(total)})
	}
	classIDs := make([]uuid.UUID, 0, len(idRows))
	for _, r := range idRows {
		classIDs = append(classIDs, r.ClassID)
	}

	// ----- detail kelas untuk page IDs -----
	type classRow struct {
		ClassID          uuid.UUID  `gorm:"column:class_id" json:"class_id"`
		ClassMasjidID    uuid.UUID  `gorm:"column:class_masjid_id" json:"class_masjid_id"`
		ClassSlug        string     `gorm:"column:class_slug" json:"class_slug"`
		ClassNotes       *string    `gorm:"column:class_notes" json:"class_notes,omitempty"`
		ClassImageURL    *string    `gorm:"column:class_image_url" json:"class_image_url,omitempty"`
		ClassDeliveryMode string    `gorm:"column:class_delivery_mode" json:"class_delivery_mode"`
		ClassIsActive    bool       `gorm:"column:class_is_active" json:"class_is_active"`
		ClassCreatedAt   time.Time  `gorm:"column:class_created_at" json:"class_created_at"`
	}
	var clsRows []classRow
	if err := ctl.DB.Table("classes AS c").
		Where("c.class_id IN ?", classIDs).
		Where(`
			c.class_masjid_id IN ?
			AND c.class_deleted_at IS NULL
			AND c.class_delete_pending_until IS NULL
		`, masjidIDs).
		Select(`
			c.class_id, c.class_masjid_id, c.class_slug,
			c.class_notes, c.class_image_url, c.class_delivery_mode,
			c.class_is_active, c.class_created_at
		`).
		Order("LOWER(c.class_slug) ASC").
		Scan(&clsRows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil detail kelas")
	}

	// ----- subjects aktif per class (untuk page IDs) -----
	type subjRow struct {
		ClassID         uuid.UUID `gorm:"column:class_id"`
		SubjectsID      uuid.UUID `gorm:"column:subjects_id"`
		SubjectsName    string    `gorm:"column:subjects_name"`
		ClassSubjectsID uuid.UUID `gorm:"column:class_subjects_id"`
	}
	var sjRows []subjRow
	if err := ctl.DB.Table("class_subjects AS cs").
		Joins(`JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id`).
		Where(`
			cs.class_subjects_masjid_id IN ?
			AND cs.class_subjects_is_active = TRUE
			AND cs.class_subjects_deleted_at IS NULL
		`, masjidIDs).
		Where("cs.class_subjects_class_id IN ?", classIDs).
		Select(`cs.class_subjects_class_id AS class_id, s.subjects_id, s.subjects_name, cs.class_subjects_id`).
		Order("s.subjects_name ASC").
		Scan(&sjRows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil subject kelas")
	}

	// ----- compose output -----
	type SubjectLite struct {
		SubjectsID      uuid.UUID `json:"subjects_id"`
		SubjectsName    string    `json:"subjects_name"`
		ClassSubjectsID uuid.UUID `json:"class_subjects_id"`
	}
	type ClassHit struct {
		ClassID          uuid.UUID     `json:"class_id"`
		ClassMasjidID    uuid.UUID     `json:"class_masjid_id"`
		ClassSlug        string        `json:"class_slug"`
		ClassNotes       *string       `json:"class_notes,omitempty"`
		ClassImageURL    *string       `json:"class_image_url,omitempty"`
		ClassDeliveryMode string       `json:"class_delivery_mode"`
		ClassIsActive    bool          `json:"class_is_active"`
		ClassCreatedAt   time.Time     `json:"class_created_at"`
		Subjects         []SubjectLite `json:"subjects"`
	}

	byClass := make(map[uuid.UUID]*ClassHit, len(clsRows))
	orderIDs := make([]uuid.UUID, 0, len(clsRows))
	for _, cr := range clsRows {
		byClass[cr.ClassID] = &ClassHit{
			ClassID:          cr.ClassID,
			ClassMasjidID:    cr.ClassMasjidID,
			ClassSlug:        cr.ClassSlug,
			ClassNotes:       cr.ClassNotes,
			ClassImageURL:    cr.ClassImageURL,
			ClassDeliveryMode: cr.ClassDeliveryMode,
			ClassIsActive:    cr.ClassIsActive,
			ClassCreatedAt:   cr.ClassCreatedAt,
			Subjects:         []SubjectLite{},
		}
		orderIDs = append(orderIDs, cr.ClassID)
	}
	for _, sr := range sjRows {
		if hit := byClass[sr.ClassID]; hit != nil {
			hit.Subjects = append(hit.Subjects, SubjectLite{
				SubjectsID:      sr.SubjectsID,
				SubjectsName:    sr.SubjectsName,
				ClassSubjectsID: sr.ClassSubjectsID,
			})
		}
	}

	out := make([]ClassHit, 0, len(orderIDs))
	for _, id := range orderIDs {
		out = append(out, *byClass[id])
	}

	return helper.JsonList(c, out, fiber.Map{
		"limit": limit, "offset": offset, "total": int(total),
	})
}
