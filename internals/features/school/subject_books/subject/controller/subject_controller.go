// file: internals/features/lembaga/subjects/main/controller/subjects_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strings"
	"time"

	subjectDTO "masjidku_backend/internals/features/school/subject_books/subject/dto"
	subjectModel "masjidku_backend/internals/features/school/subject_books/subject/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =======================================================
   CONTROLLER
   ======================================================= */

type SubjectsController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewSubjectsController(db *gorm.DB, v interface{ Struct(any) error }) *SubjectsController {
	return &SubjectsController{DB: db, Validator: v}
}

/*
=========================================================

	CREATE (staff only) — slug unik + optional upload
	=========================================================
*/

/*
=========================================================

	CREATE (staff only) — slug unik + optional upload
	=========================================================
*/
func (h *SubjectsController) Create(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][CREATE] ▶️ incoming request")
	c.Locals("DB", h.DB)

	// 1) Parse payload
	var p subjectDTO.CreateSubjectRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Normalisasi ringan
	p.Code = strings.TrimSpace(p.Code)
	p.Name = strings.TrimSpace(p.Name)
	if p.Desc != nil {
		d := strings.TrimSpace(*p.Desc)
		if d == "" {
			p.Desc = nil
		} else {
			p.Desc = &d
		}
	}

	// 2) Resolve masjid context + staff guard
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

	// Paksa body sesuai context
	p.MasjidID = masjidID

	// 3) Uniqueness: code (opsional tapi jika ada wajib unik per masjid & alive)
	if strings.TrimSpace(p.Code) != "" {
		var cnt int64
		if err := h.DB.Model(&subjectModel.SubjectsModel{}).
			Where(`
				subjects_masjid_id = ?
				AND lower(subjects_code) = lower(?)
				AND subjects_deleted_at IS NULL
			`, masjidID, p.Code).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
		}
		if cnt > 0 {
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		}
	}

	// 4) Slug unik (CI) per masjid — pakai helpers baru
	var baseSlug string
	if p.Slug != nil && strings.TrimSpace(*p.Slug) != "" {
		baseSlug = helper.Slugify(*p.Slug, 160)
	} else {
		baseSlug = helper.Slugify(p.Name, 160)
		if baseSlug == "" {
			baseSlug = "subject"
		}
	}
	scope := func(q *gorm.DB) *gorm.DB {
		return q.Where("subjects_masjid_id = ? AND subjects_deleted_at IS NULL", masjidID)
	}
	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		h.DB,
		"subjects",
		"subjects_slug",
		baseSlug,
		scope,
		160,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}

	// 5) Build entity & simpan
	ent := p.ToModel()
	ent.SubjectsMasjidID = masjidID
	ent.SubjectsSlug = uniqueSlug

	if err := h.DB.Create(&ent).Error; err != nil {
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "uq_subjects_code_per_masjid"):
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		case strings.Contains(msg, "uq_subjects_slug_per_masjid"):
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan subject")
	}

	// 6) Optional upload image → update kolom image di DB
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		// gunakan folder yang konsisten
		keyPrefix := fmt.Sprintf("masjids/%s/classes/subjects", masjidID.String())
		if svc, er := helperOSS.NewOSSServiceFromEnv(""); er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
				uploadedURL = url

				objKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
					objKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
					objKey = k2
				}

				_ = h.DB.WithContext(c.Context()).
					Model(&subjectModel.SubjectsModel{}).
					Where("subjects_id = ?", ent.SubjectsID).
					Updates(map[string]any{
						"subjects_image_url":        uploadedURL,
						"subjects_image_object_key": objKey,
					}).Error
				// sinkron untuk response
				ent.SubjectsImageURL = &uploadedURL
				if objKey != "" {
					ent.SubjectsImageObjectKey = &objKey
				}
			}
		}
	}

	// 7) Reload (best effort)
	_ = h.DB.WithContext(c.Context()).
		First(&ent, "subjects_id = ?", ent.SubjectsID).Error

	return helper.JsonCreated(c, "Berhasil membuat subject", fiber.Map{
		"subject":            subjectDTO.FromSubjectModel(ent),
		"uploaded_image_url": uploadedURL,
	})
}

/*
=========================================================

	PATCH (staff only) — tri-state + slug unique + optional upload
	=========================================================
*/
func (h *SubjectsController) Patch(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][PATCH] ▶️ incoming request")
	c.Locals("DB", h.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil record lama (alive)
	var ent subjectModel.SubjectsModel
	if err := h.DB.WithContext(c.Context()).
		Where("subjects_id = ? AND subjects_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Subject tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard staff pada masjid terkait
	if err := helperAuth.EnsureStaffMasjid(c, ent.SubjectsMasjidID); err != nil {
		return err
	}

	// Parse payload (gunakan DTO UpdateSubjectRequest sebagai patch-friendly)
	var req subjectDTO.UpdateSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Paksa context
	req.MasjidID = &ent.SubjectsMasjidID

	// ====== Normalisasi tri-state (aman tipe) ======
	// code
	if req.Code.Present && req.Code.Value != nil {
		s := strings.TrimSpace(*req.Code.Value)
		req.Code.Value = &s
	}
	// name
	if req.Name.Present && req.Name.Value != nil {
		s := strings.TrimSpace(*req.Name.Value)
		req.Name.Value = &s
	}
	// desc (nullable; PatchField[*string] ⇒ **string)
	if req.Desc.Present && req.Desc.Value != nil {
		v := strings.TrimSpace(**req.Desc.Value)
		if v == "" {
			// empty → NULL
			req.Desc.Value = nil
		} else {
			ns := v
			ps := &ns
			req.Desc.Value = &ps // **string: pointer ke *string
		}
	}
	// slug (pakai slug helpers terbaru)
	if req.Slug.Present {
		if req.Slug.Value != nil {
			s := helper.Slugify(strings.TrimSpace(*req.Slug.Value), 160)
			if s == "" {
				req.Slug.Present = false
				req.Slug.Value = nil
			} else {
				req.Slug.Value = &s
			}
		} else {
			// present tapi nil → abaikan perubahan slug
			req.Slug.Present = false
		}
	} else if req.Name.Present && req.Name.Value != nil {
		// auto-regenerate slug ketika name berubah dan slug tidak diset eksplisit
		if s := helper.Slugify(*req.Name.Value, 160); s != "" {
			req.Slug.Present = true
			req.Slug.Value = &s
		}
	}

	// ====== Uniqueness checks bila berubah ======
	// code
	if req.Code.Present && req.Code.Value != nil && !strings.EqualFold(ent.SubjectsCode, *req.Code.Value) {
		var cnt int64
		if err := h.DB.Model(&subjectModel.SubjectsModel{}).
			Where(`
				subjects_masjid_id = ?
				AND subjects_id <> ?
				AND subjects_deleted_at IS NULL
				AND lower(subjects_code) = lower(?)
			`, ent.SubjectsMasjidID, ent.SubjectsID, *req.Code.Value).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
		}
		if cnt > 0 {
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		}
	}
	// slug
	if req.Slug.Present && req.Slug.Value != nil && !strings.EqualFold(ent.SubjectsSlug, *req.Slug.Value) {
		var cnt int64
		if err := h.DB.Model(&subjectModel.SubjectsModel{}).
			Where(`
				subjects_masjid_id = ?
				AND subjects_id <> ?
				AND subjects_deleted_at IS NULL
				AND lower(subjects_slug) = lower(?)
			`, ent.SubjectsMasjidID, ent.SubjectsID, *req.Slug.Value).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi slug")
		}
		if cnt > 0 {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
	}

	// Terapkan patch ke entity (agar dapat nilai final) + timestamp
	req.Apply(&ent)
	ent.SubjectsUpdatedAt = time.Now()

	// Jika slug tidak dikirim tetapi name berubah, regen slug yang unik (pakai helpers baru)
	if !req.Slug.Present && req.Name.Present && ent.SubjectsName != "" {
		base := helper.Slugify(ent.SubjectsName, 160)
		if base == "" {
			base = "subject"
		}
		uniq, er := helper.EnsureUniqueSlugCI(
			c.Context(),
			h.DB,
			"subjects",
			"subjects_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"subjects_masjid_id = ? AND subjects_id <> ? AND subjects_deleted_at IS NULL",
					ent.SubjectsMasjidID, ent.SubjectsID,
				)
			},
			160,
		)
		if er != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		ent.SubjectsSlug = uniq
	}

	// Bangun patch map (ikutkan kolom image hanya jika Present)
	patch := map[string]any{
		"subjects_masjid_id":  ent.SubjectsMasjidID,
		"subjects_code":       ent.SubjectsCode,
		"subjects_name":       ent.SubjectsName,
		"subjects_desc":       ent.SubjectsDesc,
		"subjects_is_active":  ent.SubjectsIsActive,
		"subjects_slug":       ent.SubjectsSlug,
		"subjects_updated_at": ent.SubjectsUpdatedAt,
	}
	if req.ImageURL.Present {
		patch["subjects_image_url"] = ent.SubjectsImageURL
	}
	if req.ImageObjectKey.Present {
		patch["subjects_image_object_key"] = ent.SubjectsImageObjectKey
	}
	if req.ImageURLOld.Present {
		patch["subjects_image_url_old"] = ent.SubjectsImageURLOld
	}
	if req.ImageObjectKeyOld.Present {
		patch["subjects_image_object_key_old"] = ent.SubjectsImageObjectKeyOld
	}
	if req.ImageDeletePendingUntil.Present {
		patch["subjects_image_delete_pending_until"] = ent.SubjectsImageDeletePendingUntil
	}

	// Simpan patch dasar
	if err := h.DB.WithContext(c.Context()).
		Model(&subjectModel.SubjectsModel{}).
		Where("subjects_id = ?", ent.SubjectsID).
		Updates(patch).Error; err != nil {
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "uq_subjects_code_per_masjid"):
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		case strings.Contains(msg, "uq_subjects_slug_per_masjid"):
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		case strings.Contains(msg, "duplicate"), strings.Contains(msg, "unique"):
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi data (kode/slug)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// ===== Optional: upload image jika ada file =====
	uploadedURL := ""
	movedOld := ""

	if fh := pickImageFile(c, "image", "file"); fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("masjids/%s/classes/subjects", ent.SubjectsMasjidID.String())
			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
				uploadedURL = url

				// object key baru
				newObjKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
					newObjKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
					newObjKey = k2
				}

				// --- ambil url & key lama dari DB (best effort) ---
				var oldURL, oldObjKey string
				{
					type row struct {
						URL string `gorm:"column:subjects_image_url"`
						Key string `gorm:"column:subjects_image_object_key"`
					}
					var r row
					_ = h.DB.WithContext(c.Context()).
						Table("subjects").
						Select("subjects_image_url, subjects_image_object_key").
						Where("subjects_id = ?", ent.SubjectsID).
						Take(&r).Error
					oldURL = strings.TrimSpace(r.URL)
					oldObjKey = strings.TrimSpace(r.Key)
				}

				// --- pindahkan lama ke spam (kalau ada) ---
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

				// --- update kolom image di DB ---
				_ = h.DB.WithContext(c.Context()).
					Model(&subjectModel.SubjectsModel{}).
					Where("subjects_id = ?", ent.SubjectsID).
					Updates(map[string]any{
						"subjects_image_url":        uploadedURL,
						"subjects_image_object_key": newObjKey,
						"subjects_image_url_old": func() any {
							if movedURL == "" {
								return gorm.Expr("NULL")
							}
							return movedURL
						}(),
						"subjects_image_object_key_old": func() any {
							if oldObjKey == "" {
								return gorm.Expr("NULL")
							}
							return oldObjKey
						}(),
						"subjects_image_delete_pending_until": deletePendingUntil,
					}).Error

				// --- sinkron struct untuk response (pakai &var / nil) ---
				ent.SubjectsImageURL = &uploadedURL
				if newObjKey != "" {
					ent.SubjectsImageObjectKey = &newObjKey
				} else {
					ent.SubjectsImageObjectKey = nil
				}
				if movedURL != "" {
					ent.SubjectsImageURLOld = &movedURL
				} else {
					ent.SubjectsImageURLOld = nil
				}
				if oldObjKey != "" {
					ent.SubjectsImageObjectKeyOld = &oldObjKey
				} else {
					ent.SubjectsImageObjectKeyOld = nil
				}
				ent.SubjectsImageDeletePendingUntil = &deletePendingUntil
			}
		}
	}

	// Reload (best effort)
	_ = h.DB.WithContext(c.Context()).
		First(&ent, "subjects_id = ?", ent.SubjectsID).Error

	return helper.JsonOK(c, "Berhasil memperbarui subject", fiber.Map{
		"subject":             subjectDTO.FromSubjectModel(ent),
		"uploaded_image_url":  uploadedURL,
		"moved_old_image_url": movedOld,
	})
}

/*
=========================================================

	DELETE (soft delete, staff only) + optional file cleanup
	- Idempotent: kalau sudah deleted, tetap boleh lanjut cleanup
	- image_url: diambil dari query/form, fallback ke ent.SubjectsImageURL
	=========================================================
*/
func (h *SubjectsController) Delete(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][DELETE] ▶️ incoming request")
	c.Locals("DB", h.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil record (alive atau sudah soft-deleted)
	var ent subjectModel.SubjectsModel
	if err := h.DB.WithContext(c.Context()).
		First(&ent, "subjects_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Subject tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard akses staff pada masjid terkait
	if err := helperAuth.EnsureStaffMasjid(c, ent.SubjectsMasjidID); err != nil {
		return err
	}

	// Soft delete bila belum
	now := time.Now()
	justDeleted := false
	if !ent.SubjectsDeletedAt.Valid {
		if err := h.DB.WithContext(c.Context()).
			Model(&subjectModel.SubjectsModel{}).
			Where("subjects_id = ?", ent.SubjectsID).
			Updates(map[string]any{
				"subjects_deleted_at": &now,
				"subjects_updated_at": now,
			}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus subject")
		}
		justDeleted = true
	}

	// === OPSIONAL: cleanup file terkait ===
	// Ambil dari query/form dulu, kalau kosong fallback ke ent.SubjectsImageURL
	imageURL := strings.TrimSpace(c.Query("image_url"))
	if imageURL == "" {
		if v := strings.TrimSpace(c.FormValue("image_url")); v != "" {
			imageURL = v
		}
	}
	if imageURL == "" && ent.SubjectsImageURL != nil && strings.TrimSpace(*ent.SubjectsImageURL) != "" {
		imageURL = strings.TrimSpace(*ent.SubjectsImageURL)
	}

	if imageURL != "" {
		// Abaikan error seperti semula
		_, _ = helperOSS.MoveToSpamByPublicURLENV(imageURL, 0)
	}

	msg := "Subject sudah dihapus"
	if justDeleted {
		msg = "Berhasil menghapus subject"
	}
	return helper.JsonOK(c, msg, fiber.Map{
		"subjects_id": ent.SubjectsID,
		"image_url":   imageURL, // kirim balik biar kelihatan yang diproses
	})
}

/* =======================================================
   Util
   ======================================================= */

// Ambil *multipart.FileHeader dari beberapa kemungkinan field name.
// Return nil bila tidak ada file.
func pickImageFile(c *fiber.Ctx, keys ...string) *multipart.FileHeader {
	if form, err := c.MultipartForm(); err == nil && form != nil {
		for _, k := range keys {
			if files, ok := form.File[k]; ok && len(files) > 0 {
				if files[0] != nil && files[0].Size > 0 {
					return files[0]
				}
			}
		}
	}
	// fallback single key
	if len(keys) > 0 {
		if fh, err := c.FormFile(keys[0]); err == nil && fh != nil && fh.Size > 0 {
			return fh
		}
	}
	return nil
}
