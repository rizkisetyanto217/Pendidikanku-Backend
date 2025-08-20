package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"masjidku_backend/internals/features/lembaga/classes/main/dto"
	"masjidku_backend/internals/features/lembaga/classes/main/model"
	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
	helper "masjidku_backend/internals/helpers"

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
		Where("c.class_masjid_id = ? AND c.class_deleted_at IS NULL", masjidID)

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

	// ----- page of class_ids (FIX: pakai GROUP BY agar ORDER BY sah) -----
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
		ClassID            uuid.UUID  `gorm:"column:class_id" json:"class_id"`
		ClassMasjidID      uuid.UUID  `gorm:"column:class_masjid_id" json:"class_masjid_id"`
		ClassName          string     `gorm:"column:class_name" json:"class_name"`
		ClassSlug          *string    `gorm:"column:class_slug" json:"class_slug,omitempty"`
		ClassDescription   *string    `gorm:"column:class_description" json:"class_description,omitempty"`
		ClassLevel         *string    `gorm:"column:class_level" json:"class_level,omitempty"`
		ClassImageURL      *string    `gorm:"column:class_image_url" json:"class_image_url,omitempty"`
		ClassFeeMonthlyIDR *int64     `gorm:"column:class_fee_monthly_idr" json:"class_fee_monthly_idr,omitempty"`
		ClassIsActive      bool       `gorm:"column:class_is_active" json:"class_is_active"`
		ClassCreatedAt     time.Time  `gorm:"column:class_created_at" json:"class_created_at"`
	}
	var clsRows []classRow
	if err := ctl.DB.Table("classes AS c").
		Where("c.class_id IN ?", classIDs).
		Where("c.class_masjid_id = ? AND c.class_deleted_at IS NULL", masjidID).
		Select(`
			c.class_id, c.class_masjid_id, c.class_name, c.class_slug,
			c.class_description, c.class_level, c.class_image_url,
			c.class_fee_monthly_idr, c.class_is_active, c.class_created_at
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
		Where("cs.class_subjects_masjid_id = ? AND cs.class_subjects_is_active = TRUE AND cs.class_subjects_deleted_at IS NULL", masjidID).
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
		ClassID            uuid.UUID    `json:"class_id"`
		ClassMasjidID      uuid.UUID    `json:"class_masjid_id"`
		ClassName          string       `json:"class_name"`
		ClassSlug          *string      `json:"class_slug,omitempty"`
		ClassDescription   *string      `json:"class_description,omitempty"`
		ClassLevel         *string      `json:"class_level,omitempty"`
		ClassImageURL      *string      `json:"class_image_url,omitempty"`
		ClassFeeMonthlyIDR *int64       `json:"class_fee_monthly_idr,omitempty"`
		ClassIsActive      bool         `json:"class_is_active"`
		ClassCreatedAt     time.Time    `json:"class_created_at"`
		Subjects           []SubjectLite `json:"subjects"`
	}

	byClass := make(map[uuid.UUID]*ClassHit, len(clsRows))
	order := make([]uuid.UUID, 0, len(clsRows))
	for _, cr := range clsRows {
		byClass[cr.ClassID] = &ClassHit{
			ClassID: cr.ClassID, ClassMasjidID: cr.ClassMasjidID,
			ClassName: cr.ClassName, ClassSlug: cr.ClassSlug,
			ClassDescription: cr.ClassDescription, ClassLevel: cr.ClassLevel,
			ClassImageURL: cr.ClassImageURL, ClassFeeMonthlyIDR: cr.ClassFeeMonthlyIDR,
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


/* ================= Handlers ================= */
// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// paksa tenant
	req.ClassMasjidID = &masjidID

	// normalisasi & slug
	req.ClassName = strings.TrimSpace(req.ClassName)
	req.ClassSlug = strings.TrimSpace(req.ClassSlug)
	if req.ClassSlug == "" {
		req.ClassSlug = helper.GenerateSlug(req.ClassName)
	} else {
		req.ClassSlug = helper.GenerateSlug(req.ClassSlug)
	}

	// validasi
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// (opsional) upload gambar dari form field "class_image_url"
	if fh, ferr := c.FormFile("class_image_url"); ferr == nil && fh != nil {
		publicURL, upErr := helper.UploadImageToSupabase("classes", fh)
		if upErr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	m := req.ToModel()

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil { tx.Rollback(); panic(r) }
	}()

	// Cek slug unik PER MASJID (case-insensitive, soft-delete aware)
	var exists model.ClassModel
	findErr := tx.
		Where(
			"class_masjid_id = ? AND lower(class_slug) = lower(?) AND class_deleted_at IS NULL",
			masjidID, m.ClassSlug,
		).
		Take(&exists).Error
	if findErr == nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
	}
	if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, findErr.Error())
	}

	// Simpan
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}

	// Update lembaga_stats
	statsSvc := service.NewLembagaStatsService()
	if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
	}
	if err := statsSvc.IncActiveClasses(tx, masjidID, +1); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Kelas berhasil dibuat", dto.NewClassResponse(m))
}



// UPDATE /admin/classes/:id  (multipart/form-data ATAU JSON)
func (ctrl *ClassController) UpdateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Parse payload (JSON / form)
	var req dto.UpdateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Normalize name/slug ---
	if req.ClassSlug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSlug))
		req.ClassSlug = &s
	} else if req.ClassName != nil {
		// Regen slug dari name hanya kalau slug tidak dikirim
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassName))
		req.ClassSlug = &s
	}

	// Paksa tenant (tidak bisa diganti dari klien)
	req.ClassMasjidID = &masjidID

	// === (Opsional) Upload file (coba "class_image", fallback "class_image_url") ===
	if fh, err := c.FormFile("class_image"); err == nil && fh != nil {
		if publicURL, upErr := helper.UploadImageToSupabase("classes", fh); upErr == nil {
			req.ClassImageURL = &publicURL
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
	} else if fh, err := c.FormFile("class_image_url"); err == nil && fh != nil {
		if publicURL, upErr := helper.UploadImageToSupabase("classes", fh); upErr == nil {
			req.ClassImageURL = &publicURL
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
	}

	// Validasi DTO
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ===== TRANSACTION START =====
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Lock row + cek tenant
	var existing model.ClassModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if existing.ClassMasjidID == nil || *existing.ClassMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah kelas di masjid lain")
	}

	// Track perubahan status aktif (untuk lembaga_stats)
	wasActive := existing.ClassIsActive
	newActive := wasActive
	if req.ClassIsActive != nil {
		newActive = *req.ClassIsActive
	}

	// Cek unik slug PER MASJID saat slug berubah (case-insensitive, soft-delete aware)
	if req.ClassSlug != nil && *req.ClassSlug != existing.ClassSlug {
		var cnt int64
		if err := tx.Model(&model.ClassModel{}).
			Where(`
				class_masjid_id = ?
				AND lower(class_slug) = lower(?)
				AND class_id <> ?
				AND class_deleted_at IS NULL
			`, masjidID, *req.ClassSlug, existing.ClassID).
			Count(&cnt).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if cnt > 0 {
			tx.Rollback()
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
	}

	// Jika URL gambar diganti manual, hapus file lama (best effort)
	if req.ClassImageURL != nil && existing.ClassImageURL != nil && *existing.ClassImageURL != *req.ClassImageURL {
		if bucket, path, exErr := helper.ExtractSupabasePath(*existing.ClassImageURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path)
		}
	}

	// Apply perubahan ke model & simpan
	req.ApplyToModel(&existing)

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {
		tx.Rollback()
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	// Sinkronkan lembaga_stats jika status aktif berubah
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveClasses(tx, masjidID, delta); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// ===== TRANSACTION END =====

	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", dto.NewClassResponse(&existing))
}


// GET /admin/classes/:id
func (ctrl *ClassController) GetClassByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	// tenant check
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses kelas di masjid lain")
	}
	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}



// GET /admin/classes
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q dto.ListClassQuery
	// default paging
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	// guard pagination
	if q.Limit <= 0 { q.Limit = 20 }
	if q.Limit > 200 { q.Limit = 200 }
	if q.Offset < 0 { q.Offset = 0 }

	tx := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_masjid_id = ?", masjidID).
		Where("class_deleted_at IS NULL")

	// filters
	if q.ActiveOnly != nil {
		tx = tx.Where("class_is_active = ?", *q.ActiveOnly)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where("(LOWER(class_name) LIKE ? OR LOWER(class_level) LIKE ?)", s, s)
	}

	// total (sebelum limit/offset)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// sorting whitelist
	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sortVal {
	case "name_asc":
		tx = tx.Order("class_name ASC")
	case "name_desc":
		tx = tx.Order("class_name DESC")
	case "created_at_asc":
		tx = tx.Order("class_created_at ASC")
	default:
		tx = tx.Order("class_created_at DESC")
	}

	// data
	var rows []model.ClassModel
	if err := tx.
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]*dto.ClassResponse, 0, len(rows))
	for i := range rows {
		items = append(items, dto.NewClassResponse(&rows[i]))
	}

	// gunakan JsonList agar konsisten: { data, pagination }
	return helper.JsonList(c, items, fiber.Map{
		"limit":  q.Limit,
		"offset": q.Offset,
		"total":  int(total),
	})
}




func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Lock row untuk hindari race
	var m model.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus kelas di masjid lain")
	}

	// Cek apakah sebelum dihapus dia aktif → nanti dipakai untuk decrement
	wasActive := m.ClassIsActive

	// Optional: hapus gambar
	deletedImage := false
	if strings.EqualFold(c.Query("delete_image"), "true") && m.ClassImageURL != nil && *m.ClassImageURL != "" {
		if bucket, path, exErr := helper.ExtractSupabasePath(*m.ClassImageURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path)
			deletedImage = true
		}
		m.ClassImageURL = nil
	}

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": now,
		"class_is_active":  false,
		"class_updated_at": now,
	}
	if deletedImage {
		updates["class_image_url"] = nil
	}

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// Decrement stats jika sebelumnya aktif
	if wasActive {
		stats := service.NewLembagaStatsService()
		// pastikan baris stats ada (idempotent)
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		// -1 kelas aktif
		if err := stats.IncActiveClasses(tx, masjidID, -1); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Kelas berhasil dihapus", fiber.Map{
		"class_id":      m.ClassID,
		"deleted_image": deletedImage,
	})
}