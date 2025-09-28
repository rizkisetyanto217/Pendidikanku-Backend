// file: internals/features/school/academics/classes/controller/class_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// Services & helpers
	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
	termmodel "masjidku_backend/internals/features/school/academics/academic_terms/model"
	dto "masjidku_backend/internals/features/school/classes/classes/dto"
	classmodel "masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"
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

// Tambahkan di atas (helpers kecil)
func slugifySafe(s string, maxLen int) string {
	return helper.Slugify(strings.TrimSpace(s), maxLen)
}


/* ================= Slug builder (parent + optional term) ================= */

func buildClassBaseSlug(
	ctx context.Context,
	db *gorm.DB,
	masjidID uuid.UUID,
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

	// 1) Ambil data class_parent (kolom sesuai model)
	type parentRow struct {
		Slug *string `gorm:"column:class_parent_slug"`
		Name string  `gorm:"column:class_parent_name"`
	}
	var pr parentRow
	if err := db.WithContext(ctx).
		Table("class_parents").
		Select("class_parent_slug, class_parent_name").
		Where("class_parent_id = ? AND class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL",
			classParentID, masjidID).
		Take(&pr).Error; err != nil {
		return "", fmt.Errorf("class_parent tidak ditemukan / db error: %w", err)
	}
	rawParent := coalesceStr(ptrToStr(pr.Slug), pr.Name)
	parentPart := strings.TrimSpace(helper.Slugify(rawParent, maxLen))
	log.Printf("[CLASSES][SLUG] parent: slug_db=%v name_db='%s' → parentPart='%s'",
		pr.Slug, pr.Name, parentPart)

	// 2) Ambil data academic_term (jika ada) — kolom sesuai model
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
			Where("academic_term_id = ? AND academic_term_masjid_id = ? AND academic_term_deleted_at IS NULL",
				*academicTermID, masjidID).
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

/* ================= Snapshot hydrator (parent & term) ================= */

func hydrateClassSnapshots(ctx context.Context, tx *gorm.DB, masjidID uuid.UUID, m *classmodel.ClassModel) error {
	// Parent (wajib)
	var cp classmodel.ClassParentModel
	if err := tx.WithContext(ctx).
		Select("class_parent_name", "class_parent_code", "class_parent_slug", "class_parent_level").
		Where("class_parent_id = ? AND class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL",
			m.ClassParentID, masjidID).
		Take(&cp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "Class parent tidak ditemukan di masjid ini")
		}
		return err
	}
	// Set snapshots dari parent
	m.ClassNameParentSnapshot = &cp.ClassParentName
	m.ClassCodeParentSnapshot = cp.ClassParentCode // *string
	m.ClassSlugParentSnapshot = cp.ClassParentSlug // *string
	if cp.ClassParentLevel != nil {
		lv := int16(*cp.ClassParentLevel)
		m.ClassLevelParentSnapshot = &lv
	} else {
		m.ClassLevelParentSnapshot = nil
	}

	// Term (opsional)
	if m.ClassTermID != nil {
		var t termmodel.AcademicTermModel
		if err := tx.WithContext(ctx).
			Select(
				"academic_term_academic_year",
				"academic_term_name",
				"academic_term_slug",
				"academic_term_angkatan",
			).
			Where("academic_term_id = ? AND academic_term_masjid_id = ? AND academic_term_deleted_at IS NULL",
				*m.ClassTermID, masjidID).
			Take(&t).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Academic term tidak ditemukan di masjid ini")
			}
			return err
		}
		m.ClassAcademicYearTermSnapshot = &t.AcademicTermAcademicYear
		m.ClassNameTermSnapshot = &t.AcademicTermName
		m.ClassSlugTermSnapshot = t.AcademicTermSlug // *string
		if t.AcademicTermAngkatan != nil {
			s := strconv.Itoa(*t.AcademicTermAngkatan)
			m.ClassAngkatanTermSnapshot = &s
		} else {
			m.ClassAngkatanTermSnapshot = nil
		}
	} else {
		m.ClassAcademicYearTermSnapshot = nil
		m.ClassNameTermSnapshot = nil
		m.ClassSlugTermSnapshot = nil
		m.ClassAngkatanTermSnapshot = nil
	}
	return nil
}

/* =========================== CREATE =========================== */

// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	start := time.Now()
	log.Printf("[CLASSES][CREATE] ▶️ incoming request")

	/* ---- Resolve Masjid Context + Staff Guard ---- */
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		log.Printf("[CLASSES][CREATE] ❌ resolve masjid ctx error: %v", err)
		return err
	}
	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
		log.Printf("[CLASSES][CREATE] 🕌 masjid_id from ctx.ID=%s", masjidID)
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			log.Printf("[CLASSES][CREATE] ❌ masjid by slug(%s) not found: %v", mc.Slug, er)
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
		log.Printf("[CLASSES][CREATE] 🕌 masjid_id from slug=%s → %s", mc.Slug, masjidID)
	default:
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			log.Printf("[CLASSES][CREATE] ❌ masjid context not found via token: %v", er)
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
		}
		masjidID = id
		log.Printf("[CLASSES][CREATE] 🕌 masjid_id from token=%s", masjidID)
	}

	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ ensure staff masjid failed: %v", err)
		return err
	}

	/* ---- Parse request & paksa tenant ---- */
	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ body parse error: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassMasjidID = masjidID
	req.Normalize()
	log.Printf("[CLASSES][CREATE] 📩 req: parent_id=%s term_id=%v delivery=%v status=%v slug_in='%s'",
		req.ClassParentID, req.ClassTermID, req.ClassDeliveryMode, req.ClassStatus, req.ClassSlug)

	/* ---- Validasi ---- */
	if err := req.Validate(); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ req validate error: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(req); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ struct validate error: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	/* ---- Bentuk model awal ---- */
	m := req.ToModel() // *classmodel.ClassModel
	log.Printf("[CLASSES][CREATE] 🔧 model init: parent_id=%s term_id=%v billing=%s status=%s",
		m.ClassParentID, m.ClassTermID, m.ClassBillingCycle, m.ClassStatus)

	/* ---- TX ---- */
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		log.Printf("[CLASSES][CREATE] ❌ begin tx error: %v", tx.Error)
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] 💥 panic recovered: %+v", r)
			panic(r)
		}
	}()

	/* ---- Slug komposit (parent + term) → CI-unique per masjid ---- */
	effectiveTermID := m.ClassTermID // *uuid.UUID (boleh nil)
	baseSlug, err := buildClassBaseSlug(
		c.Context(), tx, masjidID,
		m.ClassParentID,
		effectiveTermID,
		"", // paksa komposit
		160,
	)
	if err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ build base slug error: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Gagal membentuk slug dasar: "+err.Error())
	}
	log.Printf("[CLASSES][CREATE] 🧩 base_slug='%s' (parent+term)", baseSlug)

	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(), tx,
		"classes", "class_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_masjid_id = ? AND class_deleted_at IS NULL", masjidID)
		},
		160,
	)
	if err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ ensure unique slug error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	m.ClassSlug = uniqueSlug
	log.Printf("[CLASSES][CREATE] ✅ unique_slug='%s'", m.ClassSlug)

	/* ---- SNAPSHOT (parent+term) sebelum insert ---- */
	if err := hydrateClassSnapshots(c.Context(), tx, masjidID, m); err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ hydrate snapshots error: %v", err)
		return err // sudah fiber.Error(400) jika not found
	}

	/* ---- Insert ---- */
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		log.Printf("[CLASSES][CREATE] ❌ insert error: %v", err)
		if strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")) {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}
	log.Printf("[CLASSES][CREATE] 💾 created class_id=%s", m.ClassID)

	/* ---- Optional upload image ---- */
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		log.Printf("[CLASSES][CREATE] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("masjids/%s/classes", masjidID.String())
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
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] ❌ ensure stats error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveClasses(tx, masjidID, +1); err != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] ❌ inc active classes error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	/* ---- Commit ---- */
	if err := tx.Commit().Error; err != nil {
		log.Printf("[CLASSES][CREATE] ❌ commit error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	log.Printf("[CLASSES][CREATE] ✅ done in %s", time.Since(start))
	return helper.JsonCreated(c, "Kelas berhasil dibuat", fiber.Map{
		"class":              dto.FromModel(m),
		"uploaded_image_url": uploadedURL,
	})
}

/* =========================== PATCH =========================== */
// PATCH /admin/classes/:id
func (ctrl *ClassController) PatchClass(c *fiber.Ctx) error {
	// ---- Path param ----
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
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

	// ---- Ambil existing + lock ----
	var existing classmodel.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {

		_ = tx.Rollback().Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---- Guard staff pada masjid terkait ----
	if err := helperAuth.EnsureStaffMasjid(c, existing.ClassMasjidID); err != nil {
		_ = tx.Rollback().Error
		return err
	}

	// ---- Snapshot sebelum apply (untuk deteksi perubahan & stats) ----
	prevParentID := existing.ClassParentID
	prevTermID := existing.ClassTermID
	wasActive := (existing.ClassStatus == classmodel.ClassStatusActive)

	// ---- Apply patch ke entity (selain slug, tapi aman karena slug kita proses setelah ini) ----
	req.Apply(&existing)

	// ---- Track perubahan status active (setelah apply) ----
	newActive := (existing.ClassStatus == classmodel.ClassStatusActive)

	// ==== SLUG HANDLING ====
	// Helper bandingkan pointer UUID (termasuk perubahan ke/dari NULL)
	uuidPtrChanged := func(a, b *uuid.UUID) bool {
		if a == nil && b == nil {
			return false
		}
		if (a == nil) != (b == nil) {
			return true
		}
		return *a != *b
	}

	// 1) Kalau user PATCH slug manual → hormati, tapi CI-unique per masjid.
	if req.ClassSlug.Present && req.ClassSlug.Value != nil {
		exp := slugifySafe(existing.ClassSlug, 160) // existing.ClassSlug sudah berisi nilai dari apply()
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
					"class_masjid_id = ? AND class_id <> ? AND class_deleted_at IS NULL",
					existing.ClassMasjidID, existing.ClassID,
				)
			},
			160,
		)
		if gErr != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		// Kalau user minta spesifik & hasil unik beda → 409 (konsisten dgn ClassParent)
		if uniq != exp {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		existing.ClassSlug = uniq
	} else {
		// 2) Slug tidak dipatch → regen jika parent/term berubah (termasuk term di-clear jadi NULL)
		parentChanged := (existing.ClassParentID != prevParentID)
		termChanged := uuidPtrChanged(existing.ClassTermID, prevTermID)
		if parentChanged || termChanged {
			baseSlug, gErr := buildClassBaseSlug(
				c.Context(), tx,
				existing.ClassMasjidID,
				existing.ClassParentID,
				existing.ClassTermID,
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
						"class_masjid_id = ? AND class_id <> ? AND class_deleted_at IS NULL",
						existing.ClassMasjidID, existing.ClassID,
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

	// ---- Simpan ----
	if err := tx.Model(&classmodel.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {

		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		switch {
		case strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")):
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}
	}

	// ---- Optional: upload gambar baru → pindahkan lama ke spam ----
	uploadedURL := ""
	movedOld := ""

	if fh := pickImageFile(c, "image", "file"); fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("masjids/%s/classes", existing.ClassMasjidID.String())
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
					}).Error
			}
		}
	}

	// ---- Update lembaga_stats jika transisi active berubah ----
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, existing.ClassMasjidID); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveClasses(tx, existing.ClassMasjidID, delta); err != nil {
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
func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Lock row + cek masjid_id untuk guard
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
		Where("class_id = ? AND class_deleted_at IS NULL", classID).
		First(&m).Error; err != nil {

		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard staff pada masjid terkait
	if err := helperAuth.EnsureStaffMasjid(c, m.ClassMasjidID); err != nil {
		_ = tx.Rollback()
		return err
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
		if err := stats.EnsureForMasjid(tx, m.ClassMasjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveClasses(tx, m.ClassMasjidID, -1); err != nil {
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
