// internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	dto "masjidku_backend/internals/features/school/subject_books/books/dto"
	model "masjidku_backend/internals/features/school/subject_books/books/model"
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
// CREATE  - POST /admin/
// Body: JSON (atau form sederhana, tanpa upload file)
// =========================================================
// file: internals/features/books/controller/books_create.go

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

	var req dto.BooksCreateRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ================== Parse body ==================
	if strings.HasPrefix(ct, "multipart/form-data") {
		// Text fields minimal
		req.BooksTitle = strings.TrimSpace(c.FormValue("books_title"))
		req.BooksDesc = strPtrIfNotEmpty(c.FormValue("books_desc")) // <-- sesuai DTO

		// (opsional) slug dikirim FE
		if v := strings.TrimSpace(c.FormValue("books_slug")); v != "" {
			s := helper.Slugify(v, 100)
			req.BooksSlug = &s
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
	req.BooksMasjidID = masjidID
	req.Normalize()

	if req.BooksSlug == nil || strings.TrimSpace(*req.BooksSlug) == "" {
		gen := helper.SuggestSlugFromName(req.BooksTitle)
		req.BooksSlug = &gen
	} else {
		s := helper.Slugify(*req.BooksSlug, 100)
		req.BooksSlug = &s
	}

	// ================== Validasi ==================
	if err := validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if req.BooksSlug == nil || *req.BooksSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak valid (judul terlalu kosong untuk dibentuk slug)")
	}

	// Cek unik slug per masjid
	var cnt int64
	if err := h.DB.Model(&model.BooksModel{}).
		Where(`
			books_masjid_id = ?
			AND lower(books_slug) = lower(?)
			AND books_deleted_at IS NULL
		`, masjidID, *req.BooksSlug).
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
		if strings.Contains(msg, "uq_books_slug_per_masjid") ||
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
			BookURLBookID:    m.BooksID, // sesuaikan field PK model buku kamu
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
						BookURLBookID:   m.BooksID,
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
					`, masjidID, m.BooksID, it.BookURLKind, it.BookURLID).
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

	// (opsional) isi URLs ringkas utk response
	// Jika DTO response sudah menampung URLs, load cepat:
	// isi URLs ringkas utk response
	var rows []model.BookURLModel
	_ = h.DB.Where("book_url_book_id = ?", m.BooksID).
		Order("book_url_order ASC, book_url_created_at ASC").
		Find(&rows)

	resp := dto.ToBooksResponse(m)
	for _, r := range rows {
		if r.BookURLHref == nil {
			continue
		}
		resp.URLs = append(resp.URLs, dto.BookURLLiteBook{
			ID:        r.BookURLID,
			Label:     r.BookURLLabel,
			Href:      *r.BookURLHref,
			Kind:      r.BookURLKind,
			IsPrimary: r.BookURLIsPrimary,
			Order:     r.BookURLOrder,
		})
	}

	return helper.JsonCreated(c, "Buku berhasil dibuat", resp)

}

/*
	=========================================================
	  PATCH - /api/a/:masjid_id/book-urls/:id
	  Body: JSON (partial update, tanpa upload)
	  Field yang boleh diubah: label, order, is_primary, kind, href, object_key
	  - Jika is_primary=true → unset primary lain untuk (book_id, kind) yang sama
	  - Jika ganti object_key/href → simpan yang lama ke object_key_old (bila belum ada)

=========================================================
*/
func (h *BooksController) Patch(c *fiber.Ctx) error {
	// Inject DB utk helper slug->id dsb (konsisten dgn controller lain)
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// Masjid context + guard DKM/Admin
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Param ID URL
	rawID := strings.TrimSpace(c.Params("id"))
	urlID, err := uuid.Parse(rawID)
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

	// Parse body
	var req struct {
		BookURLLabel     *string `json:"book_url_label"`
		BookURLOrder     *int    `json:"book_url_order"`
		BookURLIsPrimary *bool   `json:"book_url_is_primary"`
		BookURLKind      *string `json:"book_url_kind"`
		BookURLHref      *string `json:"book_url_href"`
		BookURLObjectKey *string `json:"book_url_object_key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi ringan
	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		t := strings.TrimSpace(*p)
		if t == "" {
			return nil
		}
		return &t
	}
	req.BookURLLabel = trimPtr(req.BookURLLabel)
	req.BookURLKind = trimPtr(req.BookURLKind)
	req.BookURLHref = trimPtr(req.BookURLHref)
	req.BookURLObjectKey = trimPtr(req.BookURLObjectKey)

	// TX biar aman saat set primary / rotasi object key
	tx := h.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
		}
	}()

	// LOCK row saat update
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&u, "book_url_id = ?", urlID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data URL")
	}

	// Terapkan perubahan bidang-bidang
	if req.BookURLLabel != nil {
		u.BookURLLabel = req.BookURLLabel
	}
	if req.BookURLOrder != nil {
		u.BookURLOrder = *req.BookURLOrder
	}
	if req.BookURLKind != nil {
		kind := strings.TrimSpace(*req.BookURLKind)
		if kind != "" {
			u.BookURLKind = kind
		}
	}
	// Rotasi object key/href jika diubah → simpan yang lama ke object_key_old
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

	// Simpan perubahan dasar
	if err := tx.Save(&u).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan URL")
	}

	// Atur primary unik per (book_id, kind) jika diminta
	if req.BookURLIsPrimary != nil && *req.BookURLIsPrimary {
		// unset primary lainnya untuk (masjid, book, kind) yang sama
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

	// Response sederhana
	return helper.JsonOK(c, "URL buku berhasil diperbarui", fiber.Map{
		"book_url_id":         u.BookURLID,
		"book_url_kind":       u.BookURLKind,
		"book_url_label":      u.BookURLLabel,
		"book_url_href":       u.BookURLHref,
		"book_url_object_key": u.BookURLObjectKey,
		"book_url_is_primary": u.BookURLIsPrimary,
		"book_url_order":      u.BookURLOrder,
	})
}

/*
	=========================================================
	  DELETE (soft) - /api/a/:masjid_id/book-urls/:id
	  - Soft-delete: pakai DeletedAt
	  - Optional: tandai delete pending window agar worker bisa
	    cleanup object di OSS (jika kamu punya mekanisme itu)

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
