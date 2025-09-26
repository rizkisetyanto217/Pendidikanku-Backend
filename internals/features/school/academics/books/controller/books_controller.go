// file: internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
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

var validate = validator.New()

func strPtrIfNotEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// =========================================================
// CREATE  - POST /admin/:masjid/books
// Body: JSON atau multipart tanpa upload file (upload file juga didukung)
// =========================================================
func (h *BooksController) Create(c *fiber.Ctx) error {
	// Pastikan DB tersedia untuk helper slug→id
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// Ambil masjid context dari path/header/query/host/token
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	// Hanya DKM/Admin masjid ini yang boleh create
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req dto.BookCreateRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ================== Parse body ==================
	if strings.HasPrefix(ct, "multipart/form-data") {
		// Text fields minimal
		req.BookTitle = strings.TrimSpace(c.FormValue("book_title"))
		req.BookDesc = strPtrIfNotEmpty(c.FormValue("book_desc"))
		req.BookAuthor = strPtrIfNotEmpty(c.FormValue("book_author"))

		// (opsional) slug dari FE
		if v := strings.TrimSpace(c.FormValue("book_slug")); v != "" {
			s := helper.Slugify(v, 100)
			req.BookSlug = &s
		}

		// (opsional) urls_json
		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			if err := json.Unmarshal([]byte(uj), &req.URLs); err != nil {
				log.Printf("[books.create] urls_json unmarshal err: %v raw=%s", err, uj)
				return helper.JsonError(c, fiber.StatusBadRequest, "urls_json tidak valid: "+err.Error())
			}
		}

		// (opsional) bracket/array style → parse jika URLs belum ada
		if len(req.URLs) == 0 {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				ups := helperOSS.ParseURLUpsertsFromMultipart(form, nil) // gunakan defaults dari helper
				if len(ups) > 0 {
					for _, u := range ups {
						req.URLs = append(req.URLs, dto.BookURLUpsert{
							BookURLKind:      u.Kind,
							BookURLLabel:     u.Label,
							BookURLHref:      u.Href,
							BookURLObjectKey: u.ObjectKey,
							BookURLOrder:     u.Order,
							BookURLIsPrimary: u.IsPrimary,
						})
					}
				}
			}
		}
	} else {
		// JSON body
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Isi masjid ID + normalisasi + slug
	req.BookMasjidID = masjidID
	req.Normalize()

	if req.BookSlug == nil || strings.TrimSpace(*req.BookSlug) == "" {
		gen := helper.SuggestSlugFromName(req.BookTitle)
		req.BookSlug = &gen
	} else {
		s := helper.Slugify(*req.BookSlug, 100)
		req.BookSlug = &s
	}

	// ================== Validasi ==================
	if err := validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if req.BookSlug == nil || *req.BookSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak valid (judul terlalu kosong untuk dibentuk slug)")
	}

	// Cek unik slug per masjid
	var cnt int64
	if err := h.DB.Model(&model.BookModel{}).
		Where(`
			book_masjid_id = ?
			AND lower(book_slug) = lower(?)
			AND book_deleted_at IS NULL
		`, masjidID, *req.BookSlug).
		Count(&cnt).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi slug")
	}
	if cnt > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
	}

	// ================== TX ==================
	tx := h.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
		}
	}()

	// Simpan buku
	m := req.ToModel()
	if err := tx.Create(m).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_book_slug_per_masjid") ||
			strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}

	// ================== Siapkan book_urls (metadata dari body) ==================
	var urlItems []model.BookURLModel
	for _, it := range req.URLs {
		row := model.BookURLModel{
			BookURLMasjidID:  masjidID,
			BookURLBookID:    m.BookID, // PK dari model.Book
			BookURLKind:      strings.TrimSpace(it.BookURLKind),
			BookURLHref:      it.BookURLHref,
			BookURLObjectKey: it.BookURLObjectKey,
			BookURLLabel:     it.BookURLLabel,
			BookURLOrder:     it.BookURLOrder,
			BookURLIsPrimary: it.BookURLIsPrimary,
		}
		if row.BookURLKind == "" {
			row.BookURLKind = "attachment"
		}
		urlItems = append(urlItems, row)
	}

	// ================== Upload files (kalau multipart) ==================
	if strings.HasPrefix(ct, "multipart/form-data") {
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			// Kumpulkan file dari berbagai key
			fhs, usedKeys := helperOSS.CollectUploadFiles(form, nil)
			log.Printf("[books.create] collected files=%d via keys=%v", len(fhs), usedKeys)

			if len(fhs) > 0 {
				oss, oerr := helperOSS.NewOSSServiceFromEnv("")
				if oerr != nil {
					tx.Rollback()
					log.Printf("[books.create] OSS init error: %v", oerr)
					return helper.JsonError(c, fiber.StatusBadGateway, "OSS tidak siap")
				}
				ctx := context.Background()

				for idx, fh := range fhs {
					log.Printf("[books.create] uploading file #%d name=%q size=%d", idx+1, fh.Filename, fh.Size)

					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "books", fh)
					if uerr != nil {
						tx.Rollback()
						log.Printf("[books.create] upload error for %q: %v", fh.Filename, uerr)
						return helper.JsonError(c, fiber.StatusBadRequest, uerr.Error())
					}

					row := model.BookURLModel{
						BookURLMasjidID: masjidID,
						BookURLBookID:   m.BookID,
						BookURLKind:     "attachment",
					}
					// set href + object_key
					row.BookURLHref = &publicURL
					if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						row.BookURLObjectKey = &key
					}
					// order default: append di belakang
					row.BookURLOrder = len(urlItems) + 1

					urlItems = append(urlItems, row)
				}
			}
		}
	}

	// ================== Simpan book_urls (jika ada) ==================
	if len(urlItems) > 0 {
		if err := tx.Create(&urlItems).Error; err != nil {
			tx.Rollback()
			log.Printf("[books.create] insert book_urls error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan lampiran buku")
		}

		// Jaga hanya 1 primary per kind (opsional)
		for _, it := range urlItems {
			if it.BookURLIsPrimary {
				if err := tx.Model(&model.BookURLModel{}).
					Where(`
						book_url_masjid_id = ? AND
						book_url_book_id   = ? AND
						book_url_kind      = ? AND
						book_url_id       <> ?
					`, masjidID, m.BookID, it.BookURLKind, it.BookURLID).
					Update("book_url_is_primary", false).Error; err != nil {
					tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal set primary lampiran")
				}
			}
		}
	}

	// ================== Commit & Response ==================
	if err := tx.Commit().Error; err != nil {
		log.Printf("[books.create] tx commit error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	// (opsional) isi URLs ringkas untuk response
	var rows []model.BookURLModel
	_ = h.DB.Where("book_url_book_id = ?", m.BookID).
		Order("book_url_order ASC, book_url_created_at ASC").
		Find(&rows)

	resp := dto.ToBookResponse(m)
	for _, r := range rows {
		if r.BookURLHref == nil {
			continue
		}
		resp.URLs = append(resp.URLs, dto.BookURLLite{
			BookURLID:    r.BookURLID,
			BookURLLabel: r.BookURLLabel,
			BookURLHref:  *r.BookURLHref,
			BookURLKind:  r.BookURLKind,
			BookURLIsPrimary:    r.BookURLIsPrimary,
			BookURLOrder:        r.BookURLOrder,
		})
	}

	return helper.JsonCreated(c, "Buku berhasil dibuat", resp)
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
