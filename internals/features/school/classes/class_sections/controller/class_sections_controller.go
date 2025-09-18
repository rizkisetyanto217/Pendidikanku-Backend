// file: internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"errors"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	semstats "masjidku_backend/internals/features/lembaga/stats/semester_stats/service"
	ucsDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	classModel "masjidku_backend/internals/features/school/classes/classes/model"
)

type ClassSectionController struct {
	DB *gorm.DB
}

func NewClassSectionController(db *gorm.DB) *ClassSectionController {
	return &ClassSectionController{DB: db}
}

/* ================= Handlers (ADMIN) ================= */

// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	log.Printf("[SECTIONS][CREATE] ▶️ incoming request")

	// ---- Masjid context (konsisten dgn ClassParent) ----
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	default:
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
		}
		masjidID = id
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}
	log.Printf("[SECTIONS][CREATE] 🕌 masjid_id=%s", masjidID)

	// ---- Parse req ----
	var req ucsDTO.ClassSectionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassSectionsMasjidID = masjidID // paksa tenant
	log.Printf("[SECTIONS][CREATE] 📩 req: class_id=%s teacher_id=%v room_id=%v name='%s' slug_in='%s'",
		req.ClassSectionsClassID, req.ClassSectionsTeacherID, req.ClassSectionsClassRoomID,
		req.ClassSectionsName, req.ClassSectionsSlug)

	// ---- Sanity ringan ----
	if strings.TrimSpace(req.ClassSectionsName) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	if req.ClassSectionsCapacity != nil && *req.ClassSectionsCapacity < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// ---- TX ----
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// ---- Validasi class se-masjid ----
	{
		var cls classModel.ClassModel
		if err := tx.
			Select("class_id, class_masjid_id").
			Where("class_id = ? AND class_deleted_at IS NULL", req.ClassSectionsClassID).
			First(&cls).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassMasjidID != masjidID {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Class bukan milik masjid Anda")
		}
	}

	// ---- Validasi teacher se-masjid (jika ada) ----
	if req.ClassSectionsTeacherID != nil {
		var tMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT masjid_teacher_masjid_id
			FROM masjid_teachers
			WHERE masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL
		`, *req.ClassSectionsTeacherID).Scan(&tMasjid).Error; err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if tMasjid == uuid.Nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusBadRequest, "Pengajar tidak ditemukan")
		}
		if tMasjid != masjidID {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Pengajar bukan milik masjid Anda")
		}
	}

	// ---- Validasi room se-masjid (jika ada) ----
	if req.ClassSectionsClassRoomID != nil {
		var rMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT class_rooms_masjid_id
			FROM class_rooms
			WHERE class_room_id = ? AND class_rooms_deleted_at IS NULL
		`, *req.ClassSectionsClassRoomID).Scan(&rMasjid).Error; err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		if rMasjid == uuid.Nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
		}
		if rMasjid != masjidID {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Ruang kelas bukan milik masjid Anda")
		}
	}

	// ---- Slug unik (CI) per masjid (pola ClassParent) ----
	var baseSlug string
	if s := strings.TrimSpace(req.ClassSectionsSlug); s != "" {
		baseSlug = helper.Slugify(s, 160)
	} else {
		baseSlug = helper.Slugify(strings.TrimSpace(req.ClassSectionsName), 160)
		if baseSlug == "" {
			baseSlug = "section"
		}
	}
	log.Printf("[SECTIONS][SLUG] baseSlug='%s'", baseSlug)

	uniqueSlug, uErr := helper.EnsureUniqueSlugCI(
		c.Context(), tx,
		"class_sections", "class_sections_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_sections_masjid_id = ? AND class_sections_deleted_at IS NULL", masjidID)
		},
		160,
	)
	if uErr != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	log.Printf("[SECTIONS][CREATE] ✅ unique_slug='%s'", uniqueSlug)

	// ---- Map ke model & simpan ----
	m := req.ToModel()
	m.ClassSectionsMasjidID = masjidID
	m.ClassSectionsSlug = uniqueSlug

	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}
	log.Printf("[SECTIONS][CREATE] 💾 created section_id=%s", m.ClassSectionsID)

	// ---- Optional upload image (pakai helperOSS seperti Class Parent) ----
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		log.Printf("[SECTIONS][CREATE] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		url, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "classes/sections", fh)
		if upErr == nil && strings.TrimSpace(url) != "" {
			uploadedURL = url

			// derive object key
			objKey := ""
			if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
				objKey = k
			} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
				objKey = k2
			}

			// ✅ persist di DB
			_ = tx.WithContext(c.Context()).
				Table("class_sections").
				Where("class_sections_id = ?", m.ClassSectionsID).
				Updates(map[string]any{
					"class_sections_image_url":        uploadedURL,
					"class_sections_image_object_key": objKey,
				}).Error

			// ✅ sinkronkan struct utk response
			m.ClassSectionsImageURL = &uploadedURL
			m.ClassSectionsImageObjectKey = &objKey

			log.Printf("[SECTIONS][CREATE] ✅ image set url=%s key=%s", uploadedURL, objKey)
		}
	}


	// ---- Update lembaga_stats bila active ----
	if m.ClassSectionsIsActive {
		log.Printf("[SECTIONS][CREATE] 📊 updating lembaga_stats (active +1)")
		statsSvc := semstats.NewLembagaStatsService()
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveSections(tx, masjidID, +1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	log.Printf("[SECTIONS][CREATE] ✅ done")

	return helper.JsonCreated(c, "Section berhasil dibuat", fiber.Map{
		"section":            ucsDTO.FromModelClassSection(m),
		"uploaded_image_url": uploadedURL,
	})
}

// PATCH /admin/class-sections/:id   (PATCH semantics)
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req ucsDTO.ClassSectionPatchRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	var existing secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).
		First(&existing).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---- Guard staff pada masjid terkait (pola ClassParent) ----
	if err := helperAuth.EnsureStaffMasjid(c, existing.ClassSectionsMasjidID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// ---- Normalisasi req ringan ----
	if req.ClassSectionsName.Present && req.ClassSectionsName.Value != nil {
		name := strings.TrimSpace(*req.ClassSectionsName.Value)
		if name == "" {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Nama section wajib diisi")
		}
	}
	if req.ClassSectionsCapacity.Present && req.ClassSectionsCapacity.Value != nil && *req.ClassSectionsCapacity.Value < 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// ---- Validasi teacher kalau diubah ----
	if req.ClassSectionsTeacherID.Present && req.ClassSectionsTeacherID.Value != nil {
		var tMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT masjid_teacher_masjid_id
			FROM masjid_teachers
			WHERE masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL
		`, *req.ClassSectionsTeacherID.Value).Scan(&tMasjid).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if tMasjid == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Pengajar tidak ditemukan")
		}
		if tMasjid != existing.ClassSectionsMasjidID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Pengajar bukan milik masjid Anda")
		}
	}

	// ---- Validasi room kalau diubah ----
	if req.ClassSectionsClassRoomID.Present && req.ClassSectionsClassRoomID.Value != nil {
		var rMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT class_rooms_masjid_id
			FROM class_rooms
			WHERE class_room_id = ? AND class_rooms_deleted_at IS NULL
		`, *req.ClassSectionsClassRoomID.Value).Scan(&rMasjid).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		if rMasjid == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
		}
		if rMasjid != existing.ClassSectionsMasjidID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Ruang kelas bukan milik masjid Anda")
		}
	}

	// ---- SLUG handling (pola ClassParent) ----
	// Jika slug dipatch → generate & unique
	if req.ClassSectionsSlug.Present && req.ClassSectionsSlug.Value != nil {
		base := helper.Slugify(strings.TrimSpace(*req.ClassSectionsSlug.Value), 160)
		if base == "" {
			// kalau kosong, coba dari name (final)
			n := existing.ClassSectionsName
			if req.ClassSectionsName.Present && req.ClassSectionsName.Value != nil {
				n = strings.TrimSpace(*req.ClassSectionsName.Value)
			}
			base = helper.Slugify(n, 160)
			if base == "" {
				base = "section"
			}
		}
		uniq, e := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"class_sections", "class_sections_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_sections_masjid_id = ? AND class_sections_id <> ? AND class_sections_deleted_at IS NULL",
					existing.ClassSectionsMasjidID, existing.ClassSectionsID,
				)
			},
			160,
		)
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		// set ke request agar Apply() ikut menulis
		req.ClassSectionsSlug.Value = &uniq
	} else if req.ClassSectionsName.Present && req.ClassSectionsName.Value != nil {
		// Slug tidak dipatch, tapi NAME berubah → regen dari NAME
		base := helper.Slugify(strings.TrimSpace(*req.ClassSectionsName.Value), 160)
		if base == "" {
			base = "section"
		}
		uniq, e := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"class_sections", "class_sections_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_sections_masjid_id = ? AND class_sections_id <> ? AND class_sections_deleted_at IS NULL",
					existing.ClassSectionsMasjidID, existing.ClassSectionsID,
				)
			},
			160,
		)
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		req.ClassSectionsSlug.Present = true
		req.ClassSectionsSlug.Value = &uniq
	}

	// ---- Track perubahan status aktif ----
	wasActive := existing.ClassSectionsIsActive
	newActive := wasActive
	if req.ClassSectionsIsActive.Present && req.ClassSectionsIsActive.Value != nil {
		newActive = *req.ClassSectionsIsActive.Value
	}

	// ---- Apply & save ----
	req.Normalize()
	req.Apply(&existing)
	if err := tx.Save(&existing).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui section")
	}

	// ---- Optional upload image (mirip ClassParent Patch; best-effort) ----
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		log.Printf("[SECTIONS][PATCH] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		url, upErr := helperOSS.UploadImageToOSSScoped(existing.ClassSectionsMasjidID, "classes/sections", fh)
		if upErr == nil && strings.TrimSpace(url) != "" {
			uploadedURL = url

			newObjKey := ""
			if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
				newObjKey = k
			} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
				newObjKey = k2
			}

			// Ambil lama dari DB (best-effort)
			var oldURL, oldObjKey string
			{
				type row struct {
					URL string `gorm:"column:class_sections_image_url"`
					Key string `gorm:"column:class_sections_image_object_key"`
				}
				var r row
				_ = tx.Table("class_sections").
					Select("class_sections_image_url, class_sections_image_object_key").
					Where("class_sections_id = ?", existing.ClassSectionsID).
					Take(&r).Error
				oldURL = strings.TrimSpace(r.URL)
				oldObjKey = strings.TrimSpace(r.Key)
			}

			// Pindah lama ke spam (jika ada)
			movedURL := ""
			if oldURL != "" {
				if mv, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 0); mvErr == nil {
					movedURL = mv
					// sinkronkan key lama
					if k, e := helperOSS.ExtractKeyFromPublicURL(movedURL); e == nil {
						oldObjKey = k
					} else if k2, e2 := helperOSS.KeyFromPublicURL(movedURL); e2 == nil {
						oldObjKey = k2
					}
				}
			}

			deletePendingUntil := time.Now().Add(30 * 24 * time.Hour)

			_ = tx.Table("class_sections").
				Where("class_sections_id = ?", existing.ClassSectionsID).
				Updates(map[string]any{
					"class_sections_image_url":                  uploadedURL,
					"class_sections_image_object_key":           newObjKey,
					"class_sections_image_url_old":              func() any { if movedURL == "" { return gorm.Expr("NULL") }; return movedURL }(),
					"class_sections_image_object_key_old":       func() any { if oldObjKey == "" { return gorm.Expr("NULL") }; return oldObjKey }(),
					"class_sections_image_delete_pending_until": deletePendingUntil,
				}).Error
		}
	}

	// ---- Update stats jika status aktif berubah ----
	if wasActive != newActive {
		stats := semstats.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, existing.ClassSectionsMasjidID); err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveSections(tx, existing.ClassSectionsMasjidID, delta); err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "Section berhasil diperbarui", ucsDTO.FromModelClassSection(&existing))
}

// DELETE /admin/class-sections/:id (soft delete)
func (ctrl *ClassSectionController) SoftDeleteClassSection(c *fiber.Ctx) error {
	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	var m secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard akses staff pada masjid terkait (pola ClassParent)
	if err := helperAuth.EnsureStaffMasjid(c, m.ClassSectionsMasjidID); err != nil {
		_ = tx.Rollback()
		return err
	}

	wasActive := m.ClassSectionsIsActive
	now := time.Now()

	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", m.ClassSectionsID).
		Updates(map[string]any{
			"class_sections_deleted_at": now,
			"class_sections_is_active":  false,
			"class_sections_updated_at": now,
		}).Error; err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus section")
	}

	if wasActive {
		stats := semstats.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, m.ClassSectionsMasjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveSections(tx, m.ClassSectionsMasjidID, -1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Section berhasil dihapus", fiber.Map{
		"class_sections_id": m.ClassSectionsID,
	})
}

func pickImageFile(c *fiber.Ctx, names ...string) *multipart.FileHeader {
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil && fh.Size > 0 {
			return fh
		}
	}
	return nil
}