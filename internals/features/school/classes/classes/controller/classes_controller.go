// file: internals/features/school/academics/classes/controller/class_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// Services & helpers
	"madinahsalam_backend/internals/features/lembaga/stats/lembaga_stats/service"
	academicTermsSnapshot "madinahsalam_backend/internals/features/school/academics/academic_terms/snapshot"
	dto "madinahsalam_backend/internals/features/school/classes/classes/dto"
	classmodel "madinahsalam_backend/internals/features/school/classes/classes/model"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	classParentSnapshot "madinahsalam_backend/internals/features/school/classes/classes/snapshot"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"
)

/* ================= Controller & Constructor ================= */

type ClassController struct {
	DB *gorm.DB
}

func NewClassController(db *gorm.DB) *ClassController {
	return &ClassController{DB: db}
}

var validate = validator.New()

/* ================= Helpers kecil ================= */

func ptrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func coalesceStr(a, b string) string {
	a = strings.TrimSpace(a)
	if a != "" {
		return a
	}
	return strings.TrimSpace(b)
}

func slugifySafe(s string, maxLen int) string {
	return helper.Slugify(strings.TrimSpace(s), maxLen)
}

// ClassName di model bertipe *string → bungkus string jadi *string (kosong → nil)
func strPtrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

/* ================= Slug builder (parent + optional term) ================= */

func buildClassBaseSlug(
	ctx context.Context,
	db *gorm.DB,
	schoolID uuid.UUID,
	classParentID uuid.UUID,
	academicTermID *uuid.UUID,
	explicitBase string,
	maxLen int,
) (string, error) {
	// 0) Jika explicitBase ada → pakai langsung
	if s := strings.TrimSpace(explicitBase); s != "" {
		b := helper.Slugify(s, maxLen)
		if b == "" {
			b = "kelas"
		}
		log.Printf("[CLASSES][SLUG] explicit base='%s' → '%s'", s, b)
		return b, nil
	}

	// 1) Ambil data class_parent (pakai row kecil agar loose-coupling)
	type parentRow struct {
		Slug *string `gorm:"column:class_parent_slug"`
		Name string  `gorm:"column:class_parent_name"`
	}
	var pr parentRow
	if err := db.WithContext(ctx).
		Table("class_parents").
		Select("class_parent_slug, class_parent_name").
		Where("class_parent_id = ? AND class_parent_school_id = ? AND class_parent_deleted_at IS NULL",
			classParentID, schoolID).
		Take(&pr).Error; err != nil {
		return "", fmt.Errorf("class_parent tidak ditemukan / db error: %w", err)
	}
	rawParent := coalesceStr(ptrToStr(pr.Slug), pr.Name)
	parentPart := strings.TrimSpace(helper.Slugify(rawParent, maxLen))
	log.Printf("[CLASSES][SLUG] parent: slug_db=%v name_db='%s' → parentPart='%s'",
		pr.Slug, pr.Name, parentPart)

	// 2) Ambil data academic_term (jika ada)
	termPart := ""
	if academicTermID != nil && *academicTermID != uuid.Nil {
		type termRow struct {
			Slug  *string `gorm:"column:academic_term_slug"`
			Year  string  `gorm:"column:academic_term_academic_year"`
			TName string  `gorm:"column:academic_term_name"`
		}
		var tr termRow
		if err := db.WithContext(ctx).
			Table("academic_terms").
			Select("academic_term_slug, academic_term_academic_year, academic_term_name").
			Where("academic_term_id = ? AND academic_term_school_id = ? AND academic_term_deleted_at IS NULL",
				*academicTermID, schoolID).
			Take(&tr).Error; err == nil {
			if s := strings.TrimSpace(ptrToStr(tr.Slug)); s != "" {
				termPart = helper.Slugify(s, maxLen)
			} else {
				termPart = helper.Slugify(strings.TrimSpace(tr.Year+" "+tr.TName), maxLen)
			}
			log.Printf("[CLASSES][SLUG] term: slug_db=%v year='%s' name='%s' → termPart='%s'",
				tr.Slug, tr.Year, tr.TName, termPart)
		} else {
			log.Printf("[CLASSES][SLUG] term fetch error (ignored, lanjut tanpa term): %v", err)
		}
	} else {
		log.Printf("[CLASSES][SLUG] no academic term (nil)")
	}

	// 3) Gabungkan parentPart + termPart
	base := parentPart
	if termPart != "" {
		if base != "" {
			base += "-" + termPart
		} else {
			base = termPart
		}
	}
	if base == "" {
		base = "kelas"
	}
	base = helper.Slugify(base, maxLen)
	log.Printf("[CLASSES][SLUG] baseSlug='%s'", base)

	return base, nil
}

/* =========================== CREATE =========================== */

// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	start := time.Now()
	log.Printf("[CLASSES][CREATE] ▶️ incoming request")

	/* ---- Resolve School Context via helper ---- */
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// ResolveSchoolIDFromContext sudah balikin JsonError yang rapi
		return err
	}
	log.Printf("[CLASSES][CREATE] 🕌 school_id=%s (from context)", schoolID)

	// 🔒 Hanya DKM/Admin yang boleh bikin kelas
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ ensure DKM school failed: %v", err)
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya DKM/Admin yang diizinkan untuk mengelola kelas")
	}

	/* ---- Parse request & paksa tenant ---- */
	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ body parse error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassSchoolID = schoolID
	req.Normalize()
	log.Printf("[CLASSES][CREATE] 📩 req: parent_id=%s term_id=%v delivery=%v status=%v slug_in='%s'",
		req.ClassClassParentID, req.ClassAcademicTermID, req.ClassDeliveryMode, req.ClassStatus, req.ClassSlug)

	/* ---- Validasi ---- */
	if err := req.Validate(); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ req validate error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(req); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ struct validate error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi data kelas gagal: "+err.Error())
	}

	/* ---- Bentuk model awal ---- */
	m := req.ToModel()
	log.Printf("[CLASSES][CREATE] 🔧 model init: parent_id=%s term_id=%v status=%s",
		m.ClassClassParentID, m.ClassAcademicTermID, m.ClassStatus)

	/* ---- TX ---- */
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		log.Printf("[CLASSES][CREATE] ❌ begin tx error: %v", tx.Error)
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] 💥 panic recovered: %+v", r)
			panic(r)
		}
	}()

	/* ---- Slug komposit (parent + term) → CI-unique per school ---- */
	baseSlug, err := buildClassBaseSlug(
		c.Context(), tx, schoolID,
		m.ClassClassParentID,
		m.ClassAcademicTermID, // boleh nil
		"",                    // paksa komposit
		160,
	)
	if err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ build base slug error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Gagal membentuk slug dasar: "+err.Error())
	}

	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(), tx,
		"classes", "class_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_school_id = ? AND class_deleted_at IS NULL", schoolID)
		},
		160,
	)
	if err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ ensure unique slug error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	m.ClassSlug = uniqueSlug
	log.Printf("[CLASSES][CREATE] ✅ unique_slug='%s'", m.ClassSlug)

	/* ---- SNAPSHOT (parent + term) ---- */
	if err := classParentSnapshot.HydrateClassParentSnapshot(c.Context(), tx, schoolID, m); err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ parent snapshot error: %v", err)
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot parent: "+err.Error())
	}
	if err := academicTermsSnapshot.HydrateAcademicTermSnapshot(c.Context(), tx, schoolID, m); err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ term snapshot error: %v", err)
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot term: "+err.Error())
	}

	/* ---- class_name (gabungan parent + term) ---- */
	parent := ""
	if m.ClassParentNameSnapshot != nil {
		parent = *m.ClassParentNameSnapshot
	}
	m.ClassName = strPtrOrNil(dto.ComposeClassNameSpace(parent, m.ClassAcademicTermNameSnapshot))

	/* ---- Insert ---- */
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		log.Printf("[CLASSES][CREATE] ❌ insert error: %v", err)

		if strings.Contains(low, "uq_classes_slug_per_school_alive") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di school ini")
		}

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}
	log.Printf("[CLASSES][CREATE] 💾 created class_id=%s", m.ClassID)

	/* ---- Optional upload image ---- */
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file", "class_image"); fh != nil {
		log.Printf("[CLASSES][CREATE] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("schools/%s/classes", schoolID.String())
			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
				uploadedURL = url

				objKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
					objKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
					objKey = k2
				}

				m.ClassImageURL = &uploadedURL
				m.ClassImageObjectKey = &objKey
				if err := tx.Model(&classmodel.ClassModel{}).
					Where("class_id = ?", m.ClassID).
					Updates(&classmodel.ClassModel{
						ClassImageURL:       m.ClassImageURL,
						ClassImageObjectKey: m.ClassImageObjectKey,
					}).Error; err != nil {
					log.Printf("[CLASSES][CREATE] ⚠️ persist image fields failed: %v", err)
				} else {
					log.Printf("[CLASSES][CREATE] ✅ image set url=%s key=%s", uploadedURL, objKey)
				}
			} else {
				log.Printf("[CLASSES][CREATE] ❌ upload error: %v", upErr)
			}
		} else {
			log.Printf("[CLASSES][CREATE] ❌ init OSS svc error: %v", er)
		}
	}

	/* ---- Update lembaga_stats bila active ---- */
	if m.ClassStatus == classmodel.ClassStatusActive {
		log.Printf("[CLASSES][CREATE] 📊 updating lembaga_stats (active +1)")
		statsSvc := service.NewLembagaStatsService()
		if err := statsSvc.EnsureForSchool(tx, schoolID); err != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] ❌ ensure stats error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveClasses(tx, schoolID, +1); err != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] ❌ inc active classes error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	/* ---- Commit ---- */
	if err := tx.Commit().Error; err != nil {
		log.Printf("[CLASSES][CREATE] ❌ commit error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	log.Printf("[CLASSES][CREATE] ✅ done in %s", time.Since(start))

	// ⬇️ balikin 1 object class
	return helper.JsonCreated(c, "Kelas berhasil dibuat", dto.FromModel(m))
}

/* =========================== PATCH =========================== */

// PATCH /admin/classes/:id
func (ctrl *ClassController) PatchClass(c *fiber.Ctx) error {
	// ---- Path param ----
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// ---- Resolve school dari context ----
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// ---- Parse payload tri-state (JSON / multipart) ----
	var req dto.PatchClassRequest
	if err := dto.DecodePatchClassFromRequest(c, &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ---- TX ----
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

	// ---- Ambil existing + lock (tenant-safe) ----
	var existing classmodel.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing,
			"class_id = ? AND class_school_id = ? AND class_deleted_at IS NULL",
			classID, schoolID,
		).Error; err != nil {

		_ = tx.Rollback().Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 🔒 Guard: hanya DKM/Admin di school terkait
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		_ = tx.Rollback().Error
		return err
	}

	// ---- Snapshot sebelum apply (untuk deteksi perubahan & stats) ----
	prevParentID := existing.ClassClassParentID
	prevTermID := existing.ClassAcademicTermID
	wasActive := (existing.ClassStatus == classmodel.ClassStatusActive)

	// ---- Apply patch ke entity ----
	req.Apply(&existing)

	// ---- Track perubahan status active (setelah apply) ----
	newActive := (existing.ClassStatus == classmodel.ClassStatusActive)

	// ==== SLUG HANDLING ====
	uuidPtrChanged := func(a, b *uuid.UUID) bool {
		if a == nil && b == nil {
			return false
		}
		if (a == nil) != (b == nil) {
			return true
		}
		return *a != *b
	}

	// Hitung perubahan parent/term SEKALI
	parentChanged := (existing.ClassClassParentID != prevParentID)
	termChanged := uuidPtrChanged(existing.ClassAcademicTermID, prevTermID)

	// 1) Kalau user PATCH slug manual → hormati, tapi CI-unique per school.
	if req.ClassSlug.Present && req.ClassSlug.Value != nil {
		exp := slugifySafe(existing.ClassSlug, 160) // existing.ClassSlug sudah dari apply()
		if exp == "" {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusBadRequest, "class_slug tidak boleh kosong")
		}
		uniq, gErr := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"classes", "class_slug",
			exp,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_school_id = ? AND class_id <> ? AND class_deleted_at IS NULL",
					existing.ClassSchoolID, existing.ClassID,
				)
			},
			160,
		)
		if gErr != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		if uniq != exp {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di school ini")
		}
		existing.ClassSlug = uniq
	} else {
		// 2) Slug tidak dipatch → regen jika parent/term berubah
		if parentChanged || termChanged {
			baseSlug, gErr := buildClassBaseSlug(
				c.Context(), tx,
				existing.ClassSchoolID,
				existing.ClassClassParentID,
				existing.ClassAcademicTermID,
				"", // paksa komposit
				160,
			)
			if gErr != nil {
				_ = tx.Rollback().Error
				return fiber.NewError(fiber.StatusBadRequest, "Gagal membentuk slug dasar: "+gErr.Error())
			}
			uniq, gErr := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"classes", "class_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(
						"class_school_id = ? AND class_id <> ? AND class_deleted_at IS NULL",
						existing.ClassSchoolID, existing.ClassID,
					)
				},
				160,
			)
			if gErr != nil {
				_ = tx.Rollback().Error
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			existing.ClassSlug = uniq
		}
	}

	// ---- Refresh snapshot (parent + term) & class_name jika parent/term berubah ----
	if parentChanged || termChanged {
		if err := classParentSnapshot.HydrateClassParentSnapshot(c.Context(), tx, existing.ClassSchoolID, &existing); err != nil {
			_ = tx.Rollback().Error
			return err
		}
		if err := academicTermsSnapshot.HydrateAcademicTermSnapshot(c.Context(), tx, existing.ClassSchoolID, &existing); err != nil {
			_ = tx.Rollback().Error
			return err
		}
		// recompute class_name: "<Parent> — <Term>" (atau hanya parent jika term nil/empty)
		parent := ""
		if existing.ClassParentNameSnapshot != nil {
			parent = *existing.ClassParentNameSnapshot
		}
		existing.ClassName = strPtrOrNil(dto.ComposeClassNameSpace(parent, existing.ClassAcademicTermNameSnapshot))
	}

	// ---- Simpan ----
	if err := tx.Model(&classmodel.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {

		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		switch {
		case strings.Contains(low, "uq_classes_slug_per_school_alive") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")):
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di school ini")
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}
	}

	// ---- Optional: upload gambar baru → pindahkan lama ke spam ----
	uploadedURL := ""
	movedOld := ""

	if fh := pickImageFile(c, "image", "file", "class_image"); fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("schools/%s/classes", existing.ClassSchoolID.String())
			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
				uploadedURL = url

				// object key baru
				newObjKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
					newObjKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
					newObjKey = k2
				}

				// ambil url lama dari DB (best effort)
				var oldURL, oldObjKey string
				{
					type row struct {
						URL string `gorm:"column:class_image_url"`
						Key string `gorm:"column:class_image_object_key"`
					}
					var r row
					_ = tx.Table("classes").
						Select("class_image_url, class_image_object_key").
						Where("class_id = ?", existing.ClassID).
						Take(&r).Error
					oldURL = strings.TrimSpace(r.URL)
					oldObjKey = strings.TrimSpace(r.Key)
				}

				movedURL := ""
				if oldURL != "" {
					if mv, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 0); mvErr == nil {
						movedURL = mv
						movedOld = mv
						// sinkronkan key lama ke lokasi baru
						if k, e := helperOSS.ExtractKeyFromPublicURL(movedURL); e == nil {
							oldObjKey = k
						} else if k2, e2 := helperOSS.KeyFromPublicURL(movedURL); e2 == nil {
							oldObjKey = k2
						}
					}
				}

				deletePendingUntil := time.Now().Add(30 * 24 * time.Hour)

				_ = tx.Model(&classmodel.ClassModel{}).
					Where("class_id = ?", existing.ClassID).
					Updates(map[string]any{
						"class_image_url":        uploadedURL,
						"class_image_object_key": newObjKey,
						"class_image_url_old": func() any {
							if movedURL == "" {
								return gorm.Expr("NULL")
							}
							return movedURL
						}(),
						"class_image_object_key_old": func() any {
							if oldObjKey == "" {
								return gorm.Expr("NULL")
							}
							return oldObjKey
						}(),
						"class_image_delete_pending_until": deletePendingUntil,
					})
			}
		}
	}

	// ---- Update lembaga_stats jika transisi active berubah ----
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForSchool(tx, existing.ClassSchoolID); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveClasses(tx, existing.ClassSchoolID, delta); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// ---- Commit ----
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", fiber.Map{
		"class":               dto.FromModel(&existing),
		"uploaded_image_url":  uploadedURL,
		"moved_old_image_url": movedOld,
	})
}

/* =========================== DELETE (soft) =========================== */

// DELETE /admin/classes/:id
func (ctrl *ClassController) DeleteClass(c *fiber.Ctx) error {
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// ---- Resolve school dari context ----
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Lock row + cek school_id untuk guard
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	var m classmodel.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"class_id = ? AND class_school_id = ? AND class_deleted_at IS NULL",
			classID, schoolID,
		).
		First(&m).Error; err != nil {

		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 🔒 Guard: hanya DKM/Admin pada school terkait
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// 🔒 GUARD: masih dipakai di class_sections?
	var sectionCount int64
	if err := tx.WithContext(c.Context()).
		Model(&classSectionModel.ClassSectionModel{}).
		Where(`
			class_section_school_id = ?
			AND class_section_class_id = ?
			AND class_section_deleted_at IS NULL
		`, m.ClassSchoolID, m.ClassID).
		Count(&sectionCount).Error; err != nil {

		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengecek relasi class sections")
	}
	if sectionCount > 0 {
		_ = tx.Rollback()
		return fiber.NewError(
			fiber.StatusBadRequest,
			"Kelas tidak dapat dihapus karena masih digunakan oleh rombongan belajar (class sections). Mohon hapus/ubah class section yang terkait terlebih dahulu.",
		)
	}

	wasActive := (m.ClassStatus == classmodel.ClassStatusActive)

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": &now,
		"class_updated_at": now,
		"class_status":     "inactive", // opsional
	}
	if err := tx.Model(&classmodel.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {

		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// Decrement stats jika sebelumnya ACTIVE
	if wasActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForSchool(tx, m.ClassSchoolID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveClasses(tx, m.ClassSchoolID, -1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Kelas berhasil dihapus", fiber.Map{
		"class_id": m.ClassID,
	})
}

/* =========================== Util =========================== */

func pickImageFile(c *fiber.Ctx, names ...string) *multipart.FileHeader {
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh
		}
	}
	return nil
}
