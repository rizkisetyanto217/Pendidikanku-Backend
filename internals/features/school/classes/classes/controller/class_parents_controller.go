// file: internals/features/school/classes/classes/controller/class_parent_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"strings"
	"time"

	cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
	classModel "masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClassParentController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewClassParentController(db *gorm.DB, v interface{ Struct(any) error }) *ClassParentController {
	return &ClassParentController{DB: db, Validator: v}
}

// Ambil file dari multipart (nama yang didukung)
func getImageFormFile(c *fiber.Ctx) (*multipart.FileHeader, error) {
	names := []string{"image", "avatar", "photo", "file", "picture"}
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh, nil
		}
	}
	return nil, errors.New("gambar tidak ditemukan")
}

/*
=========================================================

	CREATE (staff only) — slug unik + optional upload (save to DB)
	=========================================================
*/
func (ctl *ClassParentController) Create(c *fiber.Ctx) error {
	// 1) Parse payload
	var p cpdto.ClassParentCreateRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	p.Normalize()

	if len(p.ClassParentRequirements.ToJSONMap()) == 0 {
		// BodyParser pada multipart kadang tidak memanggil UnmarshalText → parse manual dari form-data
		raw := strings.TrimSpace(c.FormValue("class_parent_requirements"))
		if raw != "" {
			var tmp datatypes.JSONMap
			if err := json.Unmarshal([]byte(raw), &tmp); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest,
					"class_parent_requirements harus JSON object yang valid: "+err.Error())
			}
			// set kembali ke DTO
			p.ClassParentRequirements = cpdto.JSONMapFlexible(tmp)
		}
	}

	// 2) Resolve masjid context + staff guard
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
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
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
	if p.ClassParentMasjidID == uuid.Nil {
		p.ClassParentMasjidID = masjidID
	} else if p.ClassParentMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusConflict, "class_parent_masjid_id pada body tidak cocok dengan konteks masjid")
	}

	// 3) Uniqueness: code (opsional)
	if p.ClassParentCode != nil {
		code := strings.TrimSpace(*p.ClassParentCode)
		if code != "" {
			var cnt int64
			if err := ctl.DB.Model(&classModel.ClassParentModel{}).
				Where("class_parent_masjid_id = ? AND class_parent_code = ? AND class_parent_deleted_at IS NULL",
					masjidID, code).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kode")
			}
			if cnt > 0 {
				return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
			}
		}
	}

	// 4) Slug unik (CI) per masjid
	var baseSlug string
	if p.ClassParentSlug != nil && strings.TrimSpace(*p.ClassParentSlug) != "" {
		baseSlug = helper.Slugify(*p.ClassParentSlug, 160)
	} else {
		baseSlug = helper.Slugify(p.ClassParentName, 160)
		if baseSlug == "" {
			baseSlug = "item"
		}
	}
	scope := func(q *gorm.DB) *gorm.DB {
		return q.Where("class_parent_masjid_id = ?", masjidID).
			Where("class_parent_deleted_at IS NULL")
	}
	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(), ctl.DB,
		"class_parents", "class_parent_slug",
		baseSlug, scope, 160,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug")
	}

	// 5) Build entity & simpan
	ent := p.ToModel()
	ent.ClassParentMasjidID = masjidID
	entSlug := uniqueSlug
	ent.ClassParentSlug = &entSlug

	if err := ctl.DB.Create(ent).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	// 6) Optional upload file → simpan ke DB (image_url + object_key)
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		// Pastikan kategori benar: "classes/class-parents" (pakai slash)
		url, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "classes/class-parents", fh)
		if upErr == nil {
			uploadedURL = url

			// Ambil object key dari public URL (pakai helper yang sudah ada)
			objKey := ""
			if k, er := helperOSS.ExtractKeyFromPublicURL(uploadedURL); er == nil {
				objKey = k
			} else if k2, er2 := helperOSS.KeyFromPublicURL(uploadedURL); er2 == nil {
				objKey = k2
			}

			// Tulis ke DB (kolom existing pada model kamu)
			_ = ctl.DB.WithContext(c.Context()).
				Model(&classModel.ClassParentModel{}).
				Where("class_parent_id = ?", ent.ClassParentID).
				Updates(map[string]any{
					"class_parent_image_url":        uploadedURL,
					"class_parent_image_object_key": objKey,
					// kolom *_old dan *_delete_pending_until biarkan NULL saat create
				}).Error
		}
		// Jika upload error → abaikan (CREATE tetap sukses)
	}

	// 7) Reload entity (agar response memuat field ter-update)
	_ = ctl.DB.WithContext(c.Context()).
		First(&ent, "class_parent_id = ?", ent.ClassParentID).Error

	// 8) Response
	return helper.JsonCreated(c, "Berhasil membuat parent kelas", fiber.Map{
		"class_parent":       cpdto.FromModelClassParent(ent),
		"uploaded_image_url": uploadedURL,
	})
} 

// PATCH /api/a/:masjid_id/class-parents/:id
func (ctl *ClassParentController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	var ent classModel.ClassParentModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_parent_id = ? AND class_parent_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {

		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard staff/tenant
	if err := helperAuth.EnsureStaffMasjid(c, ent.ClassParentMasjidID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// === Parse payload (JSON / multipart) → tri-state
	var p cpdto.ClassParentPatchRequest
	if err := cpdto.DecodePatchClassParentFromRequest(c, &p); err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// === Uniqueness: code (jika diubah & non-empty)
	if p.ClassParentCode.Present && p.ClassParentCode.Value != nil {
		if v := strings.TrimSpace(**p.ClassParentCode.Value); v != "" {
			var cnt int64
			if err := tx.Model(&classModel.ClassParentModel{}).
				Where(`class_parent_masjid_id = ? AND class_parent_code = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL`,
					ent.ClassParentMasjidID, v, ent.ClassParentID).
				Count(&cnt).Error; err != nil {

				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kode")
			}
			if cnt > 0 {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
			}
		}
	}

	// === SIMPAN NILAI LAMA untuk deteksi refresh snapshot
	oldCode := ""
	if ent.ClassParentCode != nil {
		oldCode = *ent.ClassParentCode
	}
	oldName := ent.ClassParentName

	var oldSlug *string
	if ent.ClassParentSlug != nil {
		s := *ent.ClassParentSlug
		oldSlug = &s
	}

	var oldLevel *int16
	if ent.ClassParentLevel != nil {
		lv := *ent.ClassParentLevel
		oldLevel = &lv
	}

	// === Apply patch ke entity in-memory
	p.Apply(&ent)

	// === Slug handling (unique per masjid)
	if p.ClassParentSlug.Present {
		if p.ClassParentSlug.Value != nil {
			base := helper.Slugify(strings.TrimSpace(**p.ClassParentSlug.Value), 100)
			if base == "" {
				base = helper.SuggestSlugFromName(ent.ClassParentName)
			}
			uniq, err := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"class_parents", "class_parent_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(
						"class_parent_masjid_id = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL",
						ent.ClassParentMasjidID, ent.ClassParentID,
					)
				},
				100,
			)
			if err != nil {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			ent.ClassParentSlug = &uniq
		}
	} else if p.ClassParentName.Present && ent.ClassParentName != "" {
		base := helper.Slugify(ent.ClassParentName, 100)
		if base == "" {
			base = "item"
		}
		uniq, err := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"class_parents", "class_parent_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_parent_masjid_id = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL",
					ent.ClassParentMasjidID, ent.ClassParentID,
				)
			},
			100,
		)
		if err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		ent.ClassParentSlug = &uniq
	}

	// === Simpan perubahan utama (tanpa image dulu)
	if err := tx.Model(&classModel.ClassParentModel{}).
		Where("class_parent_id = ?", ent.ClassParentID).
		Updates(&ent).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// === Upload image (multipart optional) → 2-slot + retensi
	uploadedURL := ""
	movedOld := ""

	if fh, ferr := getImageFormFile(c); ferr == nil && fh != nil {
		svc, e := helperOSS.NewOSSServiceFromEnv("")
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusServiceUnavailable, "OSS belum terkonfigurasi")
		}

		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()

		// folder: classes/class-parents  (rapi & konsisten)
		url, upErr := helperOSS.UploadImageToOSS(ctx, svc, ent.ClassParentMasjidID, "classes/class-parents", fh)
		if upErr != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, upErr.Error())
		}
		key, kerr := helperOSS.KeyFromPublicURL(url)
		if kerr != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (image)")
		}

		imap := map[string]any{
			"class_parent_image_url":        url,
			"class_parent_image_object_key": key,
			"class_parent_updated_at":       time.Now(),
		}

		if ent.ClassParentImageURL != nil && *ent.ClassParentImageURL != "" {
			due := time.Now().Add(helperOSS.GetRetentionDuration())
			imap["class_parent_image_url_old"] = ent.ClassParentImageURL
			imap["class_parent_image_object_key_old"] = ent.ClassParentImageObjectKey
			imap["class_parent_image_delete_pending_until"] = &due
			movedOld = strings.TrimSpace(*ent.ClassParentImageURL)
		}

		if err := tx.Model(&classModel.ClassParentModel{}).
			Where("class_parent_id = ?", ent.ClassParentID).
			Updates(imap).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan image")
		}

		// sinkron ent untuk response
		ent.ClassParentImageURL = &url
		ent.ClassParentImageObjectKey = &key
		uploadedURL = url
	}

	// === Tentukan perlu REFRESH SNAPSHOT classes (denormalized fields)
	newCode := ""
	if ent.ClassParentCode != nil {
		newCode = *ent.ClassParentCode
	}
	newName := ent.ClassParentName
	newSlug := ent.ClassParentSlug
	newLevel := ent.ClassParentLevel

	needRefresh := false
	if p.ClassParentCode.Present && (strings.TrimSpace(newCode) != strings.TrimSpace(oldCode)) {
		needRefresh = true
	}
	if p.ClassParentName.Present && newName != oldName {
		needRefresh = true
	}
	if p.ClassParentSlug.Present {
		needRefresh = true
	}
	if !needRefresh {
		switch {
		case (oldSlug == nil && newSlug != nil),
			(oldSlug != nil && newSlug == nil):
			needRefresh = true
		case (oldSlug != nil && newSlug != nil) && (*oldSlug != *newSlug):
			needRefresh = true
		}
	}
	if p.ClassParentLevel.Present {
		switch {
		case (oldLevel == nil && newLevel != nil),
			(oldLevel != nil && newLevel == nil):
			needRefresh = true
		case (oldLevel != nil && newLevel != nil) && (*oldLevel != *newLevel):
			needRefresh = true
		}
	}

	if needRefresh {
		type classmodel = classModel.ClassModel
		if err := tx.Model(&classmodel{}).
			Where("class_masjid_id = ? AND class_parent_id = ?", ent.ClassParentMasjidID, ent.ClassParentID).
			Updates(map[string]any{
				"class_parent_code_snapshot": func() any {
					if ent.ClassParentCode == nil {
						return gorm.Expr("NULL")
					}
					return *ent.ClassParentCode
				}(),
				"class_parent_name_snapshot":  newName,
				"class_parent_slug_snapshot":  newSlug,
				"class_parent_level_snapshot": newLevel,
				"class_updated_at":            time.Now(),
			}).Error; err != nil {

			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyegarkan snapshot classes")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "Berhasil memperbarui parent kelas", fiber.Map{
		"class_parent":        cpdto.FromModelClassParent(&ent),
		"uploaded_image_url":  uploadedURL,
		"moved_old_image_url": movedOld,
	})
}

/*
=========================================================

	DELETE (soft delete, staff only) + optional file cleanup
	=========================================================
*/
func (ctl *ClassParentController) Delete(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var ent classModel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard akses staff pada masjid terkait
	if err := helperAuth.EnsureStaffMasjid(c, ent.ClassParentMasjidID); err != nil {
		return err
	}

	now := time.Now()
	if err := ctl.DB.WithContext(c.Context()).
		Model(&classModel.ClassParentModel{}).
		Where("class_parent_id = ?", ent.ClassParentID).
		Updates(map[string]any{
			"class_parent_deleted_at": &now,
			"class_parent_updated_at": now,
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// OPSIONAL: soft-delete file terkait jika image_url disertakan
	imageURL := strings.TrimSpace(c.Query("image_url"))
	if imageURL == "" {
		if v := strings.TrimSpace(c.FormValue("image_url")); v != "" {
			imageURL = v
		}
	}
	if imageURL != "" {
		if _, err := helperOSS.MoveToSpamByPublicURLENV(imageURL, 0); err != nil {
			// optional: log.Print(err)
		}
		// Kalau mau hard-delete:
		// _, _ = helperOSS.DeleteByPublicURLENV(imageURL, 15*time.Second)
	}

	return helper.JsonOK(c, "Berhasil menghapus parent kelas", fiber.Map{"class_parent_id": ent.ClassParentID})
}

/*
=========================================================

	Util
	=========================================================
*/
func clampLimit(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}
