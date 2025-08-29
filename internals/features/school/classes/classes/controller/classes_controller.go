package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
	"masjidku_backend/internals/features/school/classes/classes/dto"
	"masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
)

/* ================= Controller & Constructor ================= */

type ClassController struct {
	DB *gorm.DB
}

func NewClassController(db *gorm.DB) *ClassController {
	return &ClassController{DB: db}
}

// single validator instance for this package (tidak perlu di-inject)
var validate = validator.New()



func (ctl *ClassController) SearchWithSubjects(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

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
			c.class_masjid_id = ?
			AND c.class_deleted_at IS NULL
			AND c.class_delete_pending_until IS NULL
		`, masjidID)

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
			c.class_masjid_id = ?
			AND c.class_deleted_at IS NULL
			AND c.class_delete_pending_until IS NULL
		`, masjidID).
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
			cs.class_subjects_masjid_id = ?
			AND cs.class_subjects_is_active = TRUE
			AND cs.class_subjects_deleted_at IS NULL
		`, masjidID).
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



// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// 🔐 Paksa tenant
	req.ClassMasjidID = masjidID

	// 🧹 Normalisasi & slug dasar
	req.ClassName = strings.TrimSpace(req.ClassName)
	req.ClassSlug = strings.TrimSpace(req.ClassSlug)
	baseSlug := req.ClassSlug
	if baseSlug == "" {
		baseSlug = req.ClassName
	}

	// ✅ Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 🖼️ (Opsional) upload gambar → otomatis konversi ke WebP
	if fh, ferr := c.FormFile("class_image_url"); ferr == nil && fh != nil {
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Init OSS gagal: "+err.Error())
		}
		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		// path: masjids/{masjidID}/classes
		dir := fmt.Sprintf("masjids/%s/classes", masjidID.String())

		publicURL, upErr := svc.UploadAsWebP(ctx, fh, dir)
		if upErr != nil {
			low := strings.ToLower(upErr.Error())
			if strings.Contains(low, "format tidak didukung") {
				return fiber.NewError(fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg/png/webp)")
			}
			return fiber.NewError(fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	m := req.ToModel() // -> *model.ClassModel

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			panic(r)
		}
	}()

	// 🏷️ Generate slug unik per masjid
	slugOpts := helper.SlugOptions{
		Table:            "classes",
		SlugColumn:       "class_slug",
		SoftDeleteColumn: "class_deleted_at",
		Filters:          map[string]any{"class_masjid_id": masjidID},
		MaxLen:           160,
		DefaultBase:      "kelas",
	}
	uniqueSlug, err := helper.GenerateUniqueSlug(tx, slugOpts, baseSlug)
	if err != nil {
		_ = tx.Rollback().Error
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik: "+err.Error())
	}
	m.ClassSlug = uniqueSlug

	// 💾 Simpan
	if err := tx.Create(m).Error; err != nil {
		low := strings.ToLower(err.Error())

		// Konflik slug
		if strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")) {
			if reSlug, rErr := helper.GenerateUniqueSlug(tx, slugOpts, baseSlug); rErr == nil {
				m.ClassSlug = reSlug
				if e2 := tx.Create(m).Error; e2 == nil {
					goto SAVE_OK
				}
			}
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}

		// Konflik code
		if strings.Contains(low, "uq_classes_code_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_code")) {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Kode kelas sudah digunakan di masjid ini")
		}

		_ = tx.Rollback().Error
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}

SAVE_OK:
	// 📈 Update lembaga_stats
	if m.ClassIsActive {
		statsSvc := service.NewLembagaStatsService()
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveClasses(tx, masjidID, +1); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Kelas berhasil dibuat", dto.NewClassResponse(m))
}




// UPDATE /admin/classes/:id
// UPDATE /admin/classes/:id
func (ctrl *ClassController) UpdateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// --- Parse path param
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// --- Parse payload
	var req dto.UpdateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Normalisasi slug: jika user kirim slug → slugify; jika tidak & user kirim name → turunkan slug dari name
	if req.ClassSlug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSlug))
		req.ClassSlug = &s
	} else if req.ClassName != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassName))
		req.ClassSlug = &s
	}

	// --- Tenant: tidak boleh dipindah masjid
	// NOTE: sesuaikan tipe DTO kamu:
	// - jika *uuid.UUID -> req.ClassMasjidID = &masjidID
	// - jika uuid.UUID  -> req.ClassMasjidID = masjidID
	req.ClassMasjidID = &masjidID

	// --- Upload gambar baru (field: class_image_url) → konversi WebP via helper
	if fh, ferr := c.FormFile("class_image_url"); ferr == nil && fh != nil {
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Init OSS gagal: "+err.Error())
		}
		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		dir := fmt.Sprintf("masjids/%s/classes", masjidID.String())
		publicURL, upErr := svc.UploadAsWebP(ctx, fh, dir) // jpg/png → .webp; webp passthrough
		if upErr != nil {
			low := strings.ToLower(upErr.Error())
			if strings.Contains(low, "format tidak didukung") {
				return fiber.NewError(fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg/png/webp)")
			}
			return fiber.NewError(fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	// --- Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// --- TX
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			panic(r)
		}
	}()

	// --- Ambil existing (FOR UPDATE)
	var existing model.ClassModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {

		_ = tx.Rollback().Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// --- Tenant guard
	if existing.ClassMasjidID != masjidID {
		_ = tx.Rollback().Error
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah kelas di masjid lain")
	}

	// --- Track perubahan aktif → update stats
	wasActive := existing.ClassIsActive
	newActive := wasActive
	if req.ClassIsActive != nil {
		newActive = *req.ClassIsActive
	}

	// --- Slug unik per masjid (abaikan soft-deleted; pending delete difilter unique partial index)
	if req.ClassSlug != nil && *req.ClassSlug != existing.ClassSlug {
		opts := helper.SlugOptions{
			Table:            "classes",
			SlugColumn:       "class_slug",
			SoftDeleteColumn: "class_deleted_at",
			Filters:          map[string]any{"class_masjid_id": masjidID},
			MaxLen:           160,
			DefaultBase:      "kelas",
		}

		uni, gErr := helper.GenerateUniqueSlug(tx, opts, *req.ClassSlug)
		if gErr != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik: "+gErr.Error())
		}

		// Jika user benar-benar mengirim slug eksplisit (bukan hasil turunan dari name) dan bentrok → tolak
		if formSlug := strings.TrimSpace(c.FormValue("class_slug")); formSlug != "" && formSlug != uni {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		req.ClassSlug = &uni
	}

	// --- Jika ada gambar baru & berbeda → pindahkan yang lama ke spam/
	if req.ClassImageURL != nil && existing.ClassImageURL != nil && *existing.ClassImageURL != *req.ClassImageURL {
		if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(*existing.ClassImageURL, 0); mvErr == nil {
			// Catat ke trash_url bila belum diisi user
			if req.ClassTrashURL == nil {
				req.ClassTrashURL = &spamURL
			}
		}
		// best-effort: kalau gagal pindah ke spam, lanjutkan update data
	}

	// --- Apply & Save
	req.ApplyToModel(&existing)

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {

		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())

		switch {
		case strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")):
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")

		case strings.Contains(low, "uq_classes_code_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_code")):
			return fiber.NewError(fiber.StatusConflict, "Kode kelas sudah digunakan di masjid ini")

		default:
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}
	}

	// --- Update statistik jika status aktif berubah
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveClasses(tx, masjidID, delta); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// --- Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", dto.NewClassResponse(&existing))
}



// GET /admin/classes/:id
func (ctrl *ClassController) GetClassByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.
		Where("class_id = ? AND class_masjid_id = ? AND class_deleted_at IS NULL", classID, masjidID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}


// GET /admin/classes
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// --- parse filter ringan via Fiber (mode, code, active, search)
	var q dto.ListClassQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	// --- pagination + sorting via helper.ParseWith (pakai AdminOpts)
	rawQuery := string(c.Request().URI().QueryString())
	httpReq, _ := http.NewRequest("GET", "http://local?"+rawQuery, nil)
	p := helper.ParseWith(httpReq, "created_at", "desc", helper.AdminOpts)

	// base query
	tx := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_masjid_id = ?", masjidID).
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
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
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

	// data + paging (p.All sudah di-handle dalam PerPage; Limit/Offset tetap aman)
	var rows []model.ClassModel
	if err := tx.
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
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


// DELETE /admin/classes/:id (soft delete)
func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(); panic(r)
		}
	}()

	// Lock row
	var m model.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_id = ? AND class_masjid_id = ? AND class_deleted_at IS NULL", classID, masjidID).
		First(&m).Error; err != nil {

		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	wasActive := m.ClassIsActive

	// Optional: pindahkan gambar ke spam/ (OSS) jika diminta ?delete_image=true
	deletedImage := false
	newTrashURL := ""
	if strings.EqualFold(c.Query("delete_image"), "true") && m.ClassImageURL != nil && *m.ClassImageURL != "" {
		if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(*m.ClassImageURL, 0); mvErr == nil {
			newTrashURL = spamURL
			deletedImage = true
		}
		// best-effort walau gagal memindahkan
	}

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": now,
		"class_is_active":  false,
		"class_updated_at": now,
	}
	if deletedImage {
		updates["class_image_url"] = nil
		// simpan jejak spam url jika ada
		if newTrashURL != "" {
			updates["class_trash_url"] = newTrashURL
		}
	}

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {

		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// Decrement stats jika sebelumnya aktif
	if wasActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveClasses(tx, masjidID, -1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Kelas berhasil dihapus", fiber.Map{
		"class_id":      m.ClassID,
		"deleted_image": deletedImage,
		"trash_url":     newTrashURL,
	})
}
