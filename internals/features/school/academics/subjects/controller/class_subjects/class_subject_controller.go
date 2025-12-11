// file: internals/features/lembaga/class_subjects/controller/class_subject_controller.go
package controller

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	classSubjectBookModel "madinahsalam_backend/internals/features/school/academics/books/model"
	csDTO "madinahsalam_backend/internals/features/school/academics/subjects/dto"
	csModel "madinahsalam_backend/internals/features/school/academics/subjects/model"
	cacheSubject "madinahsalam_backend/internals/features/school/academics/subjects/service"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

type ClassSubjectController struct {
	DB *gorm.DB
}

/* ====== Helper kecil ====== */
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ===== Cache class_parent (ringan, tenant-aware) =====
type classParentSnap struct {
	Code  *string
	Slug  *string
	Level *int16 // samakan dengan tipe di model/DB kamu
	Name  *string
}

// Ambil cache Class Parent — TANPA SELECT kolom yang tak ada.
func fetchClassParentCache(db *gorm.DB, schoolID, parentID uuid.UUID) (classParentSnap, error) {
	var snap classParentSnap
	err := db.
		Table("class_parents").
		Select(`
			class_parent_code  AS code,
			class_parent_slug  AS slug,
			class_parent_level AS level,
			class_parent_name  AS name
		`).
		Where("class_parent_id = ? AND class_parent_school_id = ?", parentID, schoolID).
		Take(&snap).
		Error
	return snap, err
}

// Derive URL dari slug (opsional). Balikkan *string atau nil.
func parentURLFromSlug(slug *string) *string {
	if slug == nil {
		return nil
	}
	s := strings.TrimSpace(*slug)
	if s == "" {
		return nil
	}
	u := "/class-parents/" + s
	return &u
}

// setParentURLCacheIfExists: set field "ClassSubjectClassParentURLCache" jika ada di model.
// Aman: kalau field nggak ada, fungsi ini no-op.
func setParentURLCacheIfExists(m *csModel.ClassSubjectModel, url *string) {
	v := reflect.ValueOf(m).Elem()
	f := v.FieldByName("ClassSubjectClassParentURLCache")
	if !f.IsValid() || !f.CanSet() {
		return
	}
	// cocokkan tipe (*string)
	if f.Type() == reflect.TypeOf((*string)(nil)) {
		f.Set(reflect.ValueOf(url))
	}
}

/*
=========================================================

	CREATE
	POST /admin/:school_id/class-subjects
	(atau /admin/:school_slug/class-subjects)

=========================================================
*/
func (h *ClassSubjectController) Create(c *fiber.Ctx) error {
	c.Locals("DB", h.DB)

	// 🔐 Resolve school_id: PRIORITAS dari token, fallback ke context (ID/slug)
	var schoolID uuid.UUID

	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "Konteks sekolah tidak ditemukan")
		}
	}

	// 🔒 Role: HANYA DKM/Admin di school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// 📦 Parse payload
	var req csDTO.CreateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.SchoolID = schoolID // force tenant

	// 🧼 Normalisasi ringan
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}
	if req.Slug != nil {
		s := helper.Slugify(strings.TrimSpace(*req.Slug), 160)
		if s == "" {
			req.Slug = nil
		} else {
			req.Slug = &s
		}
	}

	// ✅ Validasi
	if err := validator.New().Struct(req); err != nil {
		// bisa di-upgrade ke JsonValidationError kalau mau field-level
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🧾 Transaksi
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// === Slug dasar ===
		baseSlug := ""
		if req.Slug != nil {
			baseSlug = *req.Slug
		} else {
			var subjName, parentSlug string
			_ = tx.Table("subjects").
				Select("subject_name").
				Where("subject_id = ? AND subject_school_id = ?", req.SubjectID, req.SchoolID).
				Scan(&subjName).Error

			_ = tx.Table("class_parents").
				Select("class_parent_slug").
				Where("class_parent_id = ? AND class_parent_school_id = ?", req.ClassParentID, req.SchoolID).
				Scan(&parentSlug).Error

			switch {
			case strings.TrimSpace(parentSlug) != "" && strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(parentSlug+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(parentSlug) != "":
				baseSlug = helper.Slugify(parentSlug, 160)
			default:
				baseSlug = helper.Slugify(
					fmt.Sprintf("cs-%s-%s",
						strings.Split(req.ClassParentID.String(), "-")[0],
						strings.Split(req.SubjectID.String(), "-")[0],
					), 160)
			}
		}

		// === Slug unik (tenant-safe) ===
		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			"class_subjects",
			"class_subject_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("class_subject_school_id = ? AND class_subject_deleted_at IS NULL", req.SchoolID)
			},
			160,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}

		// === Cache Subject (tenant-aware) ===
		subjSnap, err := cacheSubject.BuildSubjectCache(c.Context(), tx, req.SchoolID, req.SubjectID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				return fiber.NewError(fiber.StatusNotFound, "Subject tidak ditemukan")
			case errors.Is(err, cacheSubject.ErrSchoolMismatch):
				return fiber.NewError(fiber.StatusForbidden, "Subject bukan milik school ini")
			default:
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil cache subject")
			}
		}

		// === Cache Class Parent (tanpa kolom URL) ===
		parentSnap, err := fetchClassParentCache(tx, req.SchoolID, req.ClassParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Class parent tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil cache class parent")
		}

		// === Build model + isi caches ===
		m := req.ToModel()
		m.ClassSubjectSlug = &uniqueSlug

		// Subject caches
		m.ClassSubjectSubjectNameCache = &subjSnap.Name
		m.ClassSubjectSubjectCodeCache = &subjSnap.Code
		m.ClassSubjectSubjectSlugCache = &subjSnap.Slug
		m.ClassSubjectSubjectURLCache = subjSnap.URL // asumsi subject memang punya URL cache

		// Class Parent caches
		m.ClassSubjectClassParentCodeCache = parentSnap.Code
		m.ClassSubjectClassParentSlugCache = parentSnap.Slug
		m.ClassSubjectClassParentLevelCache = parentSnap.Level
		m.ClassSubjectClassParentNameCache = parentSnap.Name

		// Opsional: derive URL dari slug parent—DISET hanya jika field-nya ada di model.
		setParentURLCacheIfExists(&m, parentURLFromSlug(parentSnap.Slug))

		// === Upsert race-safe: DO NOTHING ===
		res := tx.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&m)

		if res.Error != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat class subject")
		}

		if res.RowsAffected == 0 {
			// Sudah ada — ambil existing (idempotent)
			var existing csModel.ClassSubjectModel
			if err := tx.
				Where(`
					class_subject_school_id = ?
					AND class_subject_class_parent_id = ?
					AND class_subject_subject_id = ?
					AND class_subject_deleted_at IS NULL
				`, req.SchoolID, req.ClassParentID, req.SubjectID).
				Take(&existing).Error; err != nil {

				if errors.Is(err, gorm.ErrRecordNotFound) {
					// race ekstrem — retry sekali
					if er2 := tx.Create(&m).Error; er2 != nil {
						return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat class subject (retry)")
					}
					c.Locals("created_class_subject", m)
					c.Locals("http_status", fiber.StatusCreated)
					return nil
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil class subject yang sudah ada")
			}
			c.Locals("created_class_subject", existing)
			c.Locals("http_status", fiber.StatusOK)
			return nil
		}

		// Insert baru
		c.Locals("created_class_subject", m)
		c.Locals("http_status", fiber.StatusCreated)
		return nil
	}); err != nil {
		// Semua error dari transaksi dibungkus ke JsonError standar
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// === Response ===
	m := c.Locals("created_class_subject").(csModel.ClassSubjectModel)
	status := fiber.StatusCreated
	if v := c.Locals("http_status"); v != nil {
		if s, ok := v.(int); ok {
			status = s
		}
	}

	if status == fiber.StatusCreated {
		return helper.JsonCreated(
			c,
			"Class subject berhasil diproses",
			csDTO.FromClassSubjectModelWithSchoolTime(c, m),
		)
	}
	return helper.JsonOK(
		c,
		"Class subject sudah ada (idempotent)",
		csDTO.FromClassSubjectModelWithSchoolTime(c, m),
	)
}

/*
=========================================================

	UPDATE (partial)
	PUT /admin/:school_id/class-subjects/:id

=========================================================
*/
func (h *ClassSubjectController) Update(c *fiber.Ctx) error {
	c.Locals("DB", h.DB)

	// 🔐 Resolve school_id: PRIORITAS dari token, fallback ke context
	var schoolID uuid.UUID

	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "Konteks sekolah tidak ditemukan")
		}
	}

	// 🔒 Role: HANYA DKM/Admin di school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// Param ID
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Parse payload
	var req csDTO.UpdateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.SchoolID = &schoolID

	// Normalisasi ringan
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}
	if req.Slug != nil {
		s := strings.TrimSpace(*req.Slug)
		if s == "" {
			req.Slug = nil
		} else {
			s = helper.Slugify(s, 160)
			req.Slug = &s
		}
	}

	// Validasi DTO
	if err := validator.New().Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Transaksi
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 🔒 Ambil record lama + kunci baris (race-safe)
		var m csModel.ClassSubjectModel
		if err := tx.
			Set("gorm:query_option", "FOR UPDATE").
			Where("class_subject_id = ?", id).
			First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectSchoolID != schoolID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah data milik school lain")
		}
		if m.ClassSubjectDeletedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// Simpan nilai lama untuk deteksi perubahan
		oldParentID := m.ClassSubjectClassParentID
		oldSubjectID := m.ClassSubjectSubjectID
		oldSlugEmpty := (m.ClassSubjectSlug == nil || strings.TrimSpace(ptrStr(m.ClassSubjectSlug)) == "")

		// Terapkan perubahan dari req ke model
		req.Apply(&m)

		// === Jika SubjectID berubah → refresh SubjectCache ===
		if m.ClassSubjectSubjectID != oldSubjectID {
			subjSnap, err := cacheSubject.BuildSubjectCache(c.Context(), tx, schoolID, m.ClassSubjectSubjectID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusNotFound, "Subject tidak ditemukan")
				}
				if errors.Is(err, cacheSubject.ErrSchoolMismatch) {
					return fiber.NewError(fiber.StatusForbidden, "Subject bukan milik school ini")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil cache subject")
			}
			m.ClassSubjectSubjectNameCache = &subjSnap.Name
			m.ClassSubjectSubjectCodeCache = &subjSnap.Code
			m.ClassSubjectSubjectSlugCache = &subjSnap.Slug
			m.ClassSubjectSubjectURLCache = subjSnap.URL
		}

		// === Jika ClassParentID berubah → refresh ClassParentCache ===
		if m.ClassSubjectClassParentID != oldParentID {
			parentSnap, err := fetchClassParentCache(tx, schoolID, m.ClassSubjectClassParentID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusNotFound, "Class parent tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil cache class parent")
			}
			m.ClassSubjectClassParentCodeCache = parentSnap.Code
			m.ClassSubjectClassParentSlugCache = parentSnap.Slug
			m.ClassSubjectClassParentLevelCache = parentSnap.Level
			m.ClassSubjectClassParentNameCache = parentSnap.Name
		}

		// === Slug handling (regen jika kosong / parent berubah / subject berubah / user set slug manual) ===
		needSetSlug := false
		var baseSlug string

		if req.Slug != nil {
			// User provide slug manual
			baseSlug = *req.Slug
			needSetSlug = true
		} else if oldSlugEmpty || m.ClassSubjectClassParentID != oldParentID || m.ClassSubjectSubjectID != oldSubjectID {
			needSetSlug = true

			var subjName, parentSlug string
			_ = tx.Table("subjects").
				Select("subject_name").
				Where("subject_id = ? AND subject_school_id = ?", m.ClassSubjectSubjectID, schoolID).
				Scan(&subjName).Error

			_ = tx.Table("class_parents").
				Select("class_parent_slug").
				Where("class_parent_id = ? AND class_parent_school_id = ?", m.ClassSubjectClassParentID, schoolID).
				Scan(&parentSlug).Error

			switch {
			case strings.TrimSpace(parentSlug) != "" && strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(parentSlug+" "+subjName, 160)
			case strings.TrimSpace(subjName) != "":
				baseSlug = helper.Slugify(subjName, 160)
			case strings.TrimSpace(parentSlug) != "":
				baseSlug = helper.Slugify(parentSlug, 160)
			default:
				baseSlug = helper.Slugify(
					fmt.Sprintf("cs-%s-%s",
						strings.Split(m.ClassSubjectClassParentID.String(), "-")[0],
						strings.Split(m.ClassSubjectSubjectID.String(), "-")[0],
					), 160)
			}
		}

		if needSetSlug {
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				tx,
				"class_subjects",
				"class_subject_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(`
						class_subject_school_id = ?
						AND class_subject_deleted_at IS NULL
						AND class_subject_id <> ?
					`, schoolID, m.ClassSubjectID)
				},
				160,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			m.ClassSubjectSlug = &uniqueSlug
		}

		// === Persist (biarkan DB enforce unique) ===
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where("class_subject_id = ?", m.ClassSubjectID).
			Updates(map[string]any{
				"class_subject_school_id":              m.ClassSubjectSchoolID,
				"class_subject_class_parent_id":        m.ClassSubjectClassParentID,
				"class_subject_subject_id":             m.ClassSubjectSubjectID,
				"class_subject_slug":                   m.ClassSubjectSlug,
				"class_subject_order_index":            m.ClassSubjectOrderIndex,
				"class_subject_hours_per_week":         m.ClassSubjectHoursPerWeek,
				"class_subject_min_passing_score":      m.ClassSubjectMinPassingScore,
				"class_subject_weight_on_report":       m.ClassSubjectWeightOnReport,
				"class_subject_is_core":                m.ClassSubjectIsCore,
				"class_subject_desc":                   m.ClassSubjectDesc,
				"class_subject_weight_assignment":      m.ClassSubjectWeightAssignment,
				"class_subject_weight_quiz":            m.ClassSubjectWeightQuiz,
				"class_subject_weight_mid":             m.ClassSubjectWeightMid,
				"class_subject_weight_final":           m.ClassSubjectWeightFinal,
				"class_subject_min_attendance_percent": m.ClassSubjectMinAttendancePercent,
				// caches subject (mungkin unchanged)
				"class_subject_subject_name_cache": m.ClassSubjectSubjectNameCache,
				"class_subject_subject_code_cache": m.ClassSubjectSubjectCodeCache,
				"class_subject_subject_slug_cache": m.ClassSubjectSubjectSlugCache,
				"class_subject_subject_url_cache":  m.ClassSubjectSubjectURLCache,
				// caches class_parent (mungkin unchanged)
				"class_subject_class_parent_code_cache":  m.ClassSubjectClassParentCodeCache,
				"class_subject_class_parent_slug_cache":  m.ClassSubjectClassParentSlugCache,
				"class_subject_class_parent_level_cache": m.ClassSubjectClassParentLevelCache,
				"class_subject_class_parent_url_cache":   m.ClassSubjectClassParentURLCache,
				"class_subject_class_parent_name_cache":  m.ClassSubjectClassParentNameCache,

				"class_subject_is_active": m.ClassSubjectIsActive,
			}).Error; err != nil {

			msg := strings.ToLower(err.Error())
			// Tangkap unik constraint (slug atau kombinasi parent+subject alive)
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") || strings.Contains(msg, "uq_class_subject") {
				return fiber.NewError(fiber.StatusConflict, "Slug atau kombinasi parent+subject sudah terdaftar")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}

		c.Locals("updated_class_subject", m)
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Response
	m := c.Locals("updated_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonUpdated(
		c,
		"Class subject berhasil diperbarui",
		csDTO.FromClassSubjectModelWithSchoolTime(c, m),
	)
}

/*
=========================================================

	DELETE
	DELETE /admin/:school_id/class-subjects/:id?force=true

=========================================================
*/
func (h *ClassSubjectController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", h.DB)

	// 🔐 Resolve school_id: PRIORITAS dari token, fallback ke context
	var schoolID uuid.UUID

	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "Konteks sekolah tidak ditemukan")
		}
	}

	// 🔒 Role: DKM/Admin di school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// Hanya Admin (bukan sekadar DKM) yang boleh hard delete
	adminSchoolID, _ := helperAuth.GetSchoolIDFromToken(c)
	isAdmin := adminSchoolID != uuid.Nil && adminSchoolID == schoolID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var m csModel.ClassSubjectModel
		if err := tx.First(&m, "class_subject_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectSchoolID != schoolID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik school lain")
		}

		// ===== GUARD: masih dipakai di class_subject_books? =====
		var usedCount int64
		if err := tx.Model(&classSubjectBookModel.ClassSubjectBookModel{}).
			Where(`
				class_subject_book_school_id = ?
				AND class_subject_book_class_subject_id = ?
				AND class_subject_book_deleted_at IS NULL
			`, schoolID, m.ClassSubjectID).
			Count(&usedCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengecek relasi buku")
		}

		if usedCount > 0 {
			// Blokir baik soft maupun hard delete
			return fiber.NewError(
				fiber.StatusBadRequest,
				"Class subject tidak dapat dihapus karena masih digunakan pada relasi buku (class_subject_books). Hapus/ubah relasi tersebut terlebih dahulu.",
			)
		}

		// ===== Lanjut hapus (kalau sudah tidak dipakai) =====
		if force {
			// hard delete benar-benar hapus row
			if err := tx.Unscoped().Delete(&csModel.ClassSubjectModel{}, "class_subject_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			if m.ClassSubjectDeletedAt.Valid {
				return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
			}
			// soft delete
			if err := tx.Delete(&csModel.ClassSubjectModel{}, "class_subject_id = ?", id).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		}

		c.Locals("deleted_class_subject", m)
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	m := c.Locals("deleted_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonDeleted(
		c,
		"Class subject berhasil dihapus",
		csDTO.FromClassSubjectModelWithSchoolTime(c, m),
	)
}
