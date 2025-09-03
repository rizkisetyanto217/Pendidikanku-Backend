// internals/features/lembaga/classes/user_classes/main/controller/user_my_class_controller.go
package controller

import (
	"masjidku_backend/internals/features/school/classes/classes/dto"
	"masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /admin/classes/slug/:slug
func (ctrl *ClassController) GetClassBySlug(c *fiber.Ctx) error {
	// Ambil masjid dari token (ganti ke GetUserIDFromToken jika itu yang tersedia di project)
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

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
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Tidak perlu cek masjid lagi karena sudah difilter di WHERE
	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}


// GET /admin/classes
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c) // ✅ teacher / dkm / admin / student (union)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// --- parse filter ringan via Fiber (mode, code, active, search)
	var q dto.ListClassQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// --- pagination + sorting via helper.ParseWith (pakai AdminOpts)
	rawQuery := string(c.Request().URI().QueryString())
	httpReq, _ := http.NewRequest("GET", "http://local?"+rawQuery, nil)
	p := helper.ParseWith(httpReq, "created_at", "desc", helper.AdminOpts)

	// base query
	tx := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_masjid_id IN ?", masjidIDs). // ✅ support multi-masjid
		Where("class_deleted_at IS NULL")

	// filters
	if q.ActiveOnly != nil {
		tx = tx.Where("class_is_active = ?", *q.ActiveOnly)
	}
	if q.Mode != nil && strings.TrimSpace(*q.Mode) != "" {
		tx = tx.Where("LOWER(class_mode) = LOWER(?)", strings.TrimSpace(*q.Mode))
	}
	if q.Code != nil && strings.TrimSpace(*q.Code) != "" {
		tx = tx.Where("LOWER(class_code) = LOWER(?)", strings.TrimSpace(*q.Code))
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where(`
			LOWER(class_name) LIKE ? OR
			LOWER(COALESCE(class_level, '')) LIKE ? OR
			LOWER(COALESCE(class_description, '')) LIKE ?
		`, s, s, s)
	}

	// total sebelum paging
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// sorting whitelist (pakai p.SortBy + p.SortOrder)
	sortBy := strings.ToLower(strings.TrimSpace(p.SortBy))
	order := strings.ToLower(strings.TrimSpace(p.SortOrder))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	switch sortBy {
	case "name":
		tx = tx.Order("class_name " + strings.ToUpper(order))
	case "mode":
		tx = tx.Order("LOWER(class_mode) " + strings.ToUpper(order)).Order("class_created_at DESC")
	case "created_at":
		fallthrough
	default:
		tx = tx.Order("class_created_at " + strings.ToUpper(order))
	}

	// data + paging
	var rows []model.ClassModel
	if err := tx.
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]*dto.ClassResponse, 0, len(rows))
	for i := range rows {
		items = append(items, dto.NewClassResponse(&rows[i]))
	}

	// meta dari helper
	meta := helper.BuildMeta(total, p)

	// respons standar
	return helper.JsonList(c, items, meta)
}



func (ctl *ClassController) SearchWithSubjects(c *fiber.Ctx) error {
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c) // ✅ teacher / dkm / admin / student
	if err != nil { 
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	q := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	like := "%" + q + "%"

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
		`, masjidIDs) // ✅ IN ? untuk multi masjid

	if q != "" {
		filter = filter.Where(
			`(c.class_name ILIKE ? OR c.class_slug ILIKE ? OR s.subjects_name ILIKE ?)`,
			like, like, like,
		)
	}

	// ----- total kelas unik -----
	var total int64
	if err := filter.Session(&gorm.Session{}).
		Distinct("c.class_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	// ----- page of class_ids (pakai GROUP BY agar ORDER BY sah) -----
	type idRow struct {
		ClassID   uuid.UUID `gorm:"column:class_id"`
		ClassName string    `gorm:"column:class_name"`
	}
	var idRows []idRow
	if err := filter.
		Select("c.class_id, c.class_name").
		Group("c.class_id, c.class_name").
		Order("c.class_name ASC").
		Limit(limit).Offset(offset).
		Scan(&idRows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar kelas")
	}
	if len(idRows) == 0 {
		return helper.JsonList(c, []any{}, fiber.Map{"limit": limit, "offset": offset, "total": int(total)})
	}
	classIDs := make([]uuid.UUID, 0, len(idRows))
	for _, r := range idRows { classIDs = append(classIDs, r.ClassID) }

	// ----- detail kelas untuk page IDs -----
	type classRow struct {
		ClassID          uuid.UUID  `gorm:"column:class_id" json:"class_id"`
		ClassMasjidID    uuid.UUID  `gorm:"column:class_masjid_id" json:"class_masjid_id"`
		ClassName        string     `gorm:"column:class_name" json:"class_name"`
		ClassSlug        *string    `gorm:"column:class_slug" json:"class_slug,omitempty"`
		ClassDescription *string    `gorm:"column:class_description" json:"class_description,omitempty"`
		ClassLevel       *string    `gorm:"column:class_level" json:"class_level,omitempty"`
		ClassImageURL    *string    `gorm:"column:class_image_url" json:"class_image_url,omitempty"`
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
			c.class_id, c.class_masjid_id, c.class_name, c.class_slug,
			c.class_description, c.class_level, c.class_image_url, c.class_is_active, c.class_created_at
		`).
		Order("c.class_name ASC").
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
		ClassName        string        `json:"class_name"`
		ClassSlug        *string       `json:"class_slug,omitempty"`
		ClassDescription *string       `json:"class_description,omitempty"`
		ClassLevel       *string       `json:"class_level,omitempty"`
		ClassImageURL    *string       `json:"class_image_url,omitempty"`
		ClassIsActive    bool          `json:"class_is_active"`
		ClassCreatedAt   time.Time     `json:"class_created_at"`
		Subjects         []SubjectLite `json:"subjects"`
	}

	byClass := make(map[uuid.UUID]*ClassHit, len(clsRows))
	order := make([]uuid.UUID, 0, len(clsRows))
	for _, cr := range clsRows {
		byClass[cr.ClassID] = &ClassHit{
			ClassID: cr.ClassID, ClassMasjidID: cr.ClassMasjidID,
			ClassName: cr.ClassName, ClassSlug: cr.ClassSlug,
			ClassDescription: cr.ClassDescription, ClassLevel: cr.ClassLevel,
			ClassImageURL: cr.ClassImageURL,
			ClassIsActive: cr.ClassIsActive, ClassCreatedAt: cr.ClassCreatedAt,
			Subjects: []SubjectLite{},
		}
		order = append(order, cr.ClassID)
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
	out := make([]ClassHit, 0, len(order))
	for _, id := range order { out = append(out, *byClass[id]) }

	return helper.JsonList(c, out, fiber.Map{
		"limit": limit, "offset": offset, "total": int(total),
	})
}
