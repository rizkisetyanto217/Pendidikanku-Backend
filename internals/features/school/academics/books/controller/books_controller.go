// file: internals/features/lembaga/class_books/controller/books_controller.go
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

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	dto "masjidku_backend/internals/features/school/academics/books/dto"
	model "masjidku_backend/internals/features/school/academics/books/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"
)

type BooksController struct {
	DB *gorm.DB
}

func strPtrIfNotEmpty(s string) *string {
	v := strings.TrimSpace(s)
	if v == "" {
		return nil
	}
	return &v
}

// cari file dengan prioritas beberapa key
func pickImageFile(c *fiber.Ctx, keys ...string) *multipart.FileHeader {
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	log.Printf("[BOOKS][CREATE] pickImageFile(): Content-Type=%q", ct)

	for _, k := range keys {
		fh, err := c.FormFile(k)
		if err == nil && fh != nil {
			log.Printf("[BOOKS][CREATE] FormFile(%q) OK: name=%q size=%d", k, fh.Filename, fh.Size)
			return fh
		}
		log.Printf("[BOOKS][CREATE] FormFile(%q) miss: err=%T %v", k, err, err)
	}

	form, err := c.MultipartForm()
	if err != nil || form == nil {
		log.Printf("[BOOKS][CREATE] MultipartForm() miss: err=%T %v", err, err)
		return nil
	}
	for _, k := range keys {
		if arr := form.File[k]; len(arr) > 0 {
			log.Printf("[BOOKS][CREATE] MultipartForm[\"%s\"][0] OK: name=%q size=%d", k, arr[0].Filename, arr[0].Size)
			return arr[0]
		}
		log.Printf("[BOOKS][CREATE] MultipartForm[\"%s\"] empty", k)
	}
	for k, arr := range form.File {
		if len(arr) > 0 {
			log.Printf("[BOOKS][CREATE] Fallback first file: key=%q name=%q size=%d", k, arr[0].Filename, arr[0].Size)
			return arr[0]
		}
	}
	log.Printf("[BOOKS][CREATE] No file found in multipart")
	return nil
}

// debug: list semua file keys + jumlahnya
func dumpMultipartKeys(c *fiber.Ctx) {
	if form, err := c.MultipartForm(); err == nil && form != nil {
		keys := make([]string, 0, len(form.File))
		for k, arr := range form.File {
			keys = append(keys, fmt.Sprintf("%s(len=%d)", k, len(arr)))
		}
		log.Printf("[BOOKS][CREATE] multipart keys: %v", keys)
	} else {
		log.Printf("[BOOKS][CREATE] no MultipartForm: err=%v", err)
	}
}

// POST /api/books
func (h *BooksController) Create(c *fiber.Ctx) error {
	log.Printf("[BOOKS][CREATE] ▶ incoming request %s %s", c.Method(), c.OriginalURL())
	c.Locals("DB", h.DB)

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	isMultipart := strings.HasPrefix(ct, "multipart/form-data")
	log.Printf("[BOOKS][CREATE] Content-Type=%q isMultipart=%v", ct, isMultipart)

	// 1) Parse payload
	var p dto.BookCreateRequest
	if isMultipart {
		// ✅ JANGAN BodyParser di sini
		p.BookTitle = strings.TrimSpace(c.FormValue("book_title"))
		p.BookAuthor = strPtrIfNotEmpty(c.FormValue("book_author"))
		p.BookDesc = strPtrIfNotEmpty(c.FormValue("book_desc"))
		if v := strings.TrimSpace(c.FormValue("book_slug")); v != "" {
			s := helper.Slugify(v, 160)
			p.BookSlug = &s
		}
	} else {
		// JSON saja yang pakai BodyParser
		if err := c.BodyParser(&p); err != nil {
			log.Printf("[BOOKS][CREATE] BodyParser error: %T %v", err, err)
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}
	p.Normalize()
	log.Printf("[BOOKS][CREATE] Parsed: title=%q author=%q slug=%v", p.BookTitle, derefStr(p.BookAuthor), p.BookSlug)

	// 2) Masjid context + guard
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}
	p.BookMasjidID = masjidID
	log.Printf("[BOOKS][CREATE] masjid_id=%s", masjidID)

	// 3) Slug unik
	baseSlug := ""
	if p.BookSlug != nil && strings.TrimSpace(*p.BookSlug) != "" {
		baseSlug = helper.Slugify(*p.BookSlug, 160)
	} else {
		baseSlug = helper.SuggestSlugFromName(p.BookTitle)
		if baseSlug == "" {
			baseSlug = "book"
		}
	}
	scope := func(q *gorm.DB) *gorm.DB {
		return q.Where("book_masjid_id = ? AND book_deleted_at IS NULL", masjidID)
	}
	uniqueSlug, err := helper.EnsureUniqueSlugCI(c.Context(), h.DB, "books", "book_slug", baseSlug, scope, 160)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	log.Printf("[BOOKS][CREATE] uniqueSlug=%q", uniqueSlug)

	// 4) Create entity
	ent := p.ToModel() // *model.BookModel
	ent.BookMasjidID = masjidID
	ent.BookSlug = &uniqueSlug
	if err := h.DB.Create(ent).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_books_slug_per_masjid_alive") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan buku")
	}
	log.Printf("[BOOKS][CREATE] Created book_id=%s", ent.BookID)

	// 5) (Opsional) Dump keys dan upload image
	uploadedURL := ""
	if isMultipart {
		dumpMultipartKeys(c) // log semua key file yang kebaca

		if fh := pickImageFile(c, "image", "file", "cover"); fh != nil {
			log.Printf("[BOOKS][CREATE] will upload file: name=%q size=%d", fh.Filename, fh.Size)
			keyPrefix := fmt.Sprintf("masjids/%s/library/books", masjidID.String())
			if svc, er := helperOSS.NewOSSServiceFromEnv(""); er != nil {
				log.Printf("[BOOKS][CREATE] OSS init error: %T %v", er, er)
			} else {
				ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
				defer cancel()

				url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix) // atau UploadAnyToOSS
				if upErr != nil {
					log.Printf("[BOOKS][CREATE] upload error: %T %v", upErr, upErr)
				} else {
					uploadedURL = url
					objKey := ""
					if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
						objKey = k
					} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
						objKey = k2
					}
					log.Printf("[BOOKS][CREATE] upload OK url=%s key=%q", uploadedURL, objKey)

					if err := h.DB.WithContext(c.Context()).
						Model(&model.BookModel{}).
						Where("book_id = ?", ent.BookID).
						Updates(map[string]any{
							"book_image_url":        uploadedURL,
							"book_image_object_key": objKey,
						}).Error; err != nil {
						log.Printf("[BOOKS][CREATE] DB.Updates image err: %T %v", err, err)
					} else {
						ent.BookImageURL = &uploadedURL
						if objKey != "" {
							ent.BookImageObjectKey = &objKey
						} else {
							ent.BookImageObjectKey = nil
						}
						log.Printf("[BOOKS][CREATE] image fields updated")
					}
				}
			}
		} else {
			log.Printf("[BOOKS][CREATE] no image file found after parsing multipart")
		}
	} else {
		log.Printf("[BOOKS][CREATE] not a multipart request; skipping upload")
	}

	// 6) Reload (best-effort)
	_ = h.DB.WithContext(c.Context()).First(ent, "book_id = ?", ent.BookID).Error

	// 7) Response
	resp := dto.ToBookResponse(ent)
	log.Printf("[BOOKS][CREATE] respond book_id=%s image_url=%v", ent.BookID, resp.BookImageURL)

	return helper.JsonCreated(c, "Buku berhasil dibuat", fiber.Map{
		"book":               resp,
		"uploaded_image_url": uploadedURL,
	})
}

// helper kecil buat log
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

/*
=========================================================

	PATCH URL - /api/a/:masjid_id/book-urls/:id
	Body: JSON / multipart (partial update)

=========================================================
*/
func (h *BooksController) Patch(c *fiber.Ctx) error {
	// Inject DB utk helper (konsisten)
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// Masjid context + guard
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Param ID
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil || urlID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "book_url_id tidak valid")
	}

	// Ambil row + cek tenant
	var u model.BookURLModel
	if err := h.DB.First(&u, "book_url_id = ?", urlID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data URL")
	}
	if u.BookURLMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	// ================= Parse request: JSON ATAU multipart =================
	type patchReq struct {
		BookURLLabel     *string `json:"book_url_label"     form:"book_url_label"`
		BookURLOrder     *int    `json:"book_url_order"     form:"book_url_order"`
		BookURLIsPrimary *bool   `json:"book_url_is_primary" form:"book_url_is_primary"`
		BookURLKind      *string `json:"book_url_kind"      form:"book_url_kind"`
		BookURLHref      *string `json:"book_url_href"      form:"book_url_href"`
		BookURLObjectKey *string `json:"book_url_object_key" form:"book_url_object_key"`
	}
	var req patchReq

	ct := strings.ToLower(c.Get("content-type"))
	if strings.HasPrefix(ct, "multipart/form-data") {
		// --- multipart/form-data ---
		trim := func(v string) *string {
			v = strings.TrimSpace(v)
			if v == "" {
				return nil
			}
			return &v
		}
		if v := c.FormValue("book_url_label"); v != "" || c.FormValue("book_url_label") != "" {
			req.BookURLLabel = trim(c.FormValue("book_url_label"))
		}
		if v := c.FormValue("book_url_kind"); v != "" || c.FormValue("book_url_kind") != "" {
			req.BookURLKind = trim(c.FormValue("book_url_kind"))
		}
		if v := c.FormValue("book_url_href"); v != "" || c.FormValue("book_url_href") != "" {
			req.BookURLHref = trim(c.FormValue("book_url_href"))
		}
		if v := c.FormValue("book_url_object_key"); v != "" || c.FormValue("book_url_object_key") != "" {
			req.BookURLObjectKey = trim(c.FormValue("book_url_object_key"))
		}
		if s := strings.TrimSpace(c.FormValue("book_url_order")); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				req.BookURLOrder = &n
			}
		}
		if s := strings.TrimSpace(c.FormValue("book_url_is_primary")); s != "" {
			if b, err := strconv.ParseBool(s); err == nil {
				req.BookURLIsPrimary = &b
			}
		}
	} else {
		// --- default: JSON / x-www-form-urlencoded ---
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		// normalisasi string ke trim(nil jika kosong)
		trimPtr := func(p **string) {
			if *p == nil {
				return
			}
			v := strings.TrimSpace(**p)
			if v == "" {
				*p = nil
			} else {
				*p = &v
			}
		}
		trimPtr(&req.BookURLLabel)
		trimPtr(&req.BookURLKind)
		trimPtr(&req.BookURLHref)
		trimPtr(&req.BookURLObjectKey)
	}

	// ================= TX =================
	tx := h.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
		}
	}()

	// Re-lock row
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&u, "book_url_id = ?", urlID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data URL")
	}

	// Apply perubahan
	if req.BookURLLabel != nil {
		u.BookURLLabel = req.BookURLLabel
	}
	if req.BookURLOrder != nil {
		u.BookURLOrder = *req.BookURLOrder
	}
	if req.BookURLKind != nil {
		k := strings.TrimSpace(*req.BookURLKind)
		if k != "" {
			u.BookURLKind = k
		}
	}
	// Rotasi object key/href → simpan lama ke *_old kalau belum ada
	if req.BookURLObjectKey != nil && (u.BookURLObjectKey == nil || *req.BookURLObjectKey != *u.BookURLObjectKey) {
		if u.BookURLObjectKey != nil && u.BookURLObjectKeyOld == nil {
			old := *u.BookURLObjectKey
			u.BookURLObjectKeyOld = &old
		}
		u.BookURLObjectKey = req.BookURLObjectKey
	}
	if req.BookURLHref != nil && (u.BookURLHref == nil || *req.BookURLHref != *u.BookURLHref) {
		u.BookURLHref = req.BookURLHref
	}

	// Simpan dasar
	if err := tx.Save(&u).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan URL")
	}

	// Primary unik per (book_id, kind)
	if req.BookURLIsPrimary != nil && *req.BookURLIsPrimary {
		if err := tx.Model(&model.BookURLModel{}).
			Where(`book_url_masjid_id = ? AND book_url_book_id = ? AND book_url_kind = ? AND book_url_id <> ?`,
				u.BookURLMasjidID, u.BookURLBookID, u.BookURLKind, u.BookURLID).
			Update("book_url_is_primary", false).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal reset primary URL lain")
		}
		u.BookURLIsPrimary = true
		if err := tx.Save(&u).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal set primary URL")
		}
	} else if req.BookURLIsPrimary != nil && !*req.BookURLIsPrimary {
		u.BookURLIsPrimary = false
		if err := tx.Save(&u).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal unset primary URL")
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	// Response (row terbaru)
	return helper.JsonOK(c, "URL buku berhasil diperbarui", fiber.Map{
		"book_url_id":             u.BookURLID,
		"book_url_masjid_id":      u.BookURLMasjidID,
		"book_url_book_id":        u.BookURLBookID,
		"book_url_kind":           u.BookURLKind,
		"book_url_label":          u.BookURLLabel,
		"book_url_href":           u.BookURLHref,
		"book_url_object_key":     u.BookURLObjectKey,
		"book_url_object_key_old": u.BookURLObjectKeyOld,
		"book_url_is_primary":     u.BookURLIsPrimary,
		"book_url_order":          u.BookURLOrder,
		"book_url_updated_at":     u.BookURLUpdatedAt,
	})
}

/*
=========================================================

	DELETE (soft) - /api/a/:masjid_id/book-urls/:id

=========================================================
*/
func (h *BooksController) Delete(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	rawID := strings.TrimSpace(c.Params("id"))
	urlID, err := uuid.Parse(rawID)
	if err != nil || urlID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "book_url_id tidak valid")
	}

	var u model.BookURLModel
	if err := h.DB.First(&u, "book_url_id = ?", urlID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data URL")
	}
	if u.BookURLMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	// Soft delete
	if err := h.DB.Delete(&u).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus URL")
	}

	// (Opsional) tandai pending window untuk cleanup OSS oleh worker (mis. 7 hari)
	if u.BookURLObjectKey != nil {
		_ = h.DB.Model(&model.BookURLModel{}).
			Where("book_url_id = ?", urlID).
			Update("book_url_delete_pending_until", time.Now().Add(7*24*time.Hour)).Error
	}

	return helper.JsonDeleted(c, "URL buku berhasil dihapus", fiber.Map{
		"book_url_id": urlID,
	})
}
