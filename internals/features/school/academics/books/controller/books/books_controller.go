// file: internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	dto "madinahsalam_backend/internals/features/school/academics/books/dto"
	bookModel "madinahsalam_backend/internals/features/school/academics/books/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"
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

// nilIfEmptyPtr: kalau pointer kosong → kembalikan gorm.Expr("NULL"),
// kalau ada isinya → kembalikan string-nya (biar bisa dipakai di map Updates).
func nilIfEmptyPtr(p *string) interface{} {
	if p == nil || strings.TrimSpace(*p) == "" {
		return gorm.Expr("NULL")
	}
	return *p
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

// helper kecil buat log
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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

	// 2) School context dari TOKEN + guard (khusus teacher / owner)
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak valid")
	}

	// Owner global → lolos; selain itu wajib teacher di school ini
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureTeacherSchool(c, schoolID); err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, err.Error())
		}
	}

	p.BookSchoolID = schoolID
	log.Printf("[BOOKS][CREATE] school_id=%s", schoolID)

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
		return q.Where("book_school_id = ? AND book_deleted_at IS NULL", schoolID)
	}
	uniqueSlug, err := helper.EnsureUniqueSlugCI(c.Context(), h.DB, "books", "book_slug", baseSlug, scope, 160)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	log.Printf("[BOOKS][CREATE] uniqueSlug=%q", uniqueSlug)

	// 4) Create entity
	ent := p.ToModel() // *bookModel.BookModel
	ent.BookSchoolID = schoolID
	ent.BookSlug = &uniqueSlug
	if err := h.DB.Create(ent).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_books_slug_per_school_alive") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di school ini")
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
			keyPrefix := fmt.Sprintf("schools/%s/library/books", schoolID.String())
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
						Model(&bookModel.BookModel{}).
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

	// 7) Response → pakai versi timezone sekolah
	resp := dto.ToBookResponseWithSchoolTime(c, ent)
	log.Printf("[BOOKS][CREATE] respond book_id=%s image_url=%v", ent.BookID, resp.BookImageURL)

	// data langsung = objek buku (tanpa wrapper "book", tanpa uploaded_image_url)
	return helper.JsonCreated(c, "Buku berhasil dibuat", resp)

}

// PATCH /api/a/books/:id
func (h *BooksController) Patch(c *fiber.Ctx) error {
	// inject DB utk helper
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// --- Tenant guard (teacher / owner) ---
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak valid")
	}
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureTeacherSchool(c, schoolID); err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, err.Error())
		}
	}

	// --- Param ---
	bookID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil || bookID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "book_id tidak valid")
	}

	// --- TX mulai ---
	tx := h.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
		}
	}()

	// --- Lock entity ---
	var m bookModel.BookModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "book_id = ?", bookID).Error; err != nil {
		_ = tx.Rollback().Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Buku tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil buku")
	}
	if m.BookSchoolID != schoolID {
		_ = tx.Rollback().Error
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	// --- Parse payload: JSON atau multipart ---
	var p dto.BookUpdateRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	isMultipart := strings.HasPrefix(ct, "multipart/form-data")

	if isMultipart {
		// form-data
		p.BookTitle = strPtrIfNotEmpty(c.FormValue("book_title"))
		p.BookAuthor = strPtrIfNotEmpty(c.FormValue("book_author"))
		p.BookDesc = strPtrIfNotEmpty(c.FormValue("book_desc"))
		if v := strings.TrimSpace(c.FormValue("book_slug")); v != "" {
			s := helper.Slugify(v, 160)
			p.BookSlug = &s
		}
	} else {
		// JSON / x-www-form-urlencoded
		if err := c.BodyParser(&p); err != nil {
			_ = tx.Rollback().Error
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}
	p.Normalize()

	// --- Auto-slug follow title (jika title berubah & client tidak kirim slug) ---
	titleChanged := p.BookTitle != nil &&
		strings.TrimSpace(*p.BookTitle) != "" &&
		strings.TrimSpace(*p.BookTitle) != strings.TrimSpace(m.BookTitle)

	if titleChanged && (p.BookSlug == nil || strings.TrimSpace(*p.BookSlug) == "") {
		base := helper.Slugify(*p.BookTitle, 160)
		if base == "" {
			base = helper.SuggestSlugFromName(*p.BookTitle)
			if base == "" {
				base = "book"
			}
		}
		scope := func(q *gorm.DB) *gorm.DB {
			// EXCLUDE diri sendiri saat cek unik
			return q.Where("book_school_id = ? AND book_deleted_at IS NULL AND book_id <> ?", schoolID, bookID)
		}
		uniq, err := helper.EnsureUniqueSlugCI(c.Context(), tx, "books", "book_slug", base, scope, 160)
		if err != nil {
			_ = tx.Rollback().Error
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik dari title")
		}
		p.BookSlug = &uniq
	}

	// --- Slug unik jika client kirim slug eksplisit berbeda ---
	if p.BookSlug != nil && (m.BookSlug == nil || *p.BookSlug != *m.BookSlug) {
		base := helper.Slugify(*p.BookSlug, 160)
		scope := func(q *gorm.DB) *gorm.DB {
			return q.Where("book_school_id = ? AND book_deleted_at IS NULL AND book_id <> ?", schoolID, bookID)
		}
		uniq, err := helper.EnsureUniqueSlugCI(c.Context(), tx, "books", "book_slug", base, scope, 160)
		if err != nil {
			_ = tx.Rollback().Error
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		p.BookSlug = &uniq
	}

	// --- Apply perubahan ke model buku ---
	p.ApplyToModel(&m)

	// --- Upload cover (opsional, multipart) ala Subject.Patch ---
	if isMultipart {
		// pakai key yang sama seperti create
		fh := pickImageFile(c, "image", "file", "cover")
		if fh != nil {
			svc, er := helperOSS.NewOSSServiceFromEnv("")
			if er == nil {
				ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
				defer cancel()

				keyPrefix := fmt.Sprintf("schools/%s/library/books", m.BookSchoolID.String())
				if uploadedURL, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {

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
							URL string `gorm:"column:book_image_url"`
							Key string `gorm:"column:book_image_object_key"`
						}
						var r row
						_ = tx.Table("books").
							Select("book_image_url, book_image_object_key").
							Where("book_id = ?", m.BookID).
							Take(&r).Error
						oldURL = strings.TrimSpace(r.URL)
						oldObjKey = strings.TrimSpace(r.Key)
					}

					// --- pindahkan lama ke spam (kalau ada) ---
					movedURL := ""
					if oldURL != "" {
						if mv, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 0); mvErr == nil {
							movedURL = mv
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
					_ = tx.Model(&bookModel.BookModel{}).
						Where("book_id = ?", m.BookID).
						Updates(map[string]any{
							"book_image_url":        uploadedURL,
							"book_image_object_key": newObjKey,
							"book_image_url_old": func() any {
								if movedURL == "" {
									return gorm.Expr("NULL")
								}
								return movedURL
							}(),
							"book_image_object_key_old": func() any {
								if oldObjKey == "" {
									return gorm.Expr("NULL")
								}
								return oldObjKey
							}(),
							"book_image_delete_pending_until": deletePendingUntil,
						}).Error

					// --- sinkron struct untuk response ---
					m.BookImageURL = &uploadedURL
					if newObjKey != "" {
						m.BookImageObjectKey = &newObjKey
					} else {
						m.BookImageObjectKey = nil
					}
					if movedURL != "" {
						m.BookImageURLOld = &movedURL
					} else {
						m.BookImageURLOld = nil
					}
					if oldObjKey != "" {
						m.BookImageObjectKeyOld = &oldObjKey
					} else {
						m.BookImageObjectKeyOld = nil
					}
					m.BookImageDeletePendingUntil = &deletePendingUntil
				}
			}
		}
	}

	// 🔹 Update timestamp manual
	m.BookUpdatedAt = time.Now()

	// --- Simpan buku (field non-image yang lain) ---
	if err := tx.Save(&m).Error; err != nil {
		_ = tx.Rollback().Error
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_books_slug_per_school_alive") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di school ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan buku")
	}

	// --- Sinkron cache ke class_subject_books ---
	upd := map[string]any{
		"class_subject_book_book_title_cache":     m.BookTitle,
		"class_subject_book_book_author_cache":    nilIfEmptyPtr(m.BookAuthor),
		"class_subject_book_book_slug_cache":      nilIfEmptyPtr(m.BookSlug),
		"class_subject_book_book_image_url_cache": nilIfEmptyPtr(m.BookImageURL),
	}
	if m.BookPublisher != nil {
		upd["class_subject_book_book_publisher_cache"] = *m.BookPublisher
	} else {
		upd["class_subject_book_book_publisher_cache"] = gorm.Expr("NULL")
	}
	if m.BookPublicationYear != nil {
		upd["class_subject_book_book_publication_year_cache"] = *m.BookPublicationYear
	} else {
		upd["class_subject_book_book_publication_year_cache"] = gorm.Expr("NULL")
	}

	if err := tx.Model(&bookModel.ClassSubjectBookModel{}).
		Where(`
			class_subject_book_school_id = ?
			AND class_subject_book_book_id = ?
			AND class_subject_book_deleted_at IS NULL
		`, schoolID, m.BookID).
		Updates(upd).Error; err != nil {
		_ = tx.Rollback().Error
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal sinkron cache pemakaian buku")
	}

	// --- Commit ---
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	// --- Response (pakai timezone sekolah) ---
	resp := dto.ToBookResponseWithSchoolTime(c, &m)
	return helper.JsonUpdated(c, "Buku berhasil diperbarui", resp)
}

/*
=========================================================

	DELETE (soft) - /api/a/:school_id/books/:id

=========================================================
*/
func (h *BooksController) Delete(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// --- Tenant guard (teacher / owner) ---
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak valid")
	}
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureTeacherSchool(c, schoolID); err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, err.Error())
		}
	}

	// --- Parse book_id ---
	rawID := strings.TrimSpace(c.Params("id"))
	bookID, err := uuid.Parse(rawID)
	if err != nil || bookID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "book_id tidak valid")
	}

	// --- Ambil entity buku ---
	var b bookModel.BookModel
	if err := h.DB.
		Where("book_school_id = ? AND book_id = ?", schoolID, bookID).
		First(&b).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data buku tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
	}

	// === GUARD 1: Masih dipakai di BookURL? ===
	var urlCount int64
	if err := h.DB.
		Model(&bookModel.BookURLModel{}).
		Where("book_url_school_id = ? AND book_url_book_id = ? AND book_url_deleted_at IS NULL", schoolID, bookID).
		Count(&urlCount).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek relasi URL buku")
	}

	// === GUARD 2: Masih dipakai di ClassSubjectBook? ===
	var csbCount int64
	if err := h.DB.
		Model(&bookModel.ClassSubjectBookModel{}).
		Where("class_subject_book_school_id = ? AND class_subject_book_book_id = ? AND class_subject_book_deleted_at IS NULL", schoolID, bookID).
		Count(&csbCount).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek relasi buku pada mata pelajaran")
	}

	if urlCount > 0 || csbCount > 0 {
		return helper.JsonError(
			c,
			fiber.StatusConflict,
			"Tidak dapat menghapus buku karena masih digunakan di URL buku atau terhubung dengan mata pelajaran. Silakan hapus/putuskan relasi tersebut terlebih dahulu.",
		)
	}

	// --- Aman → soft delete buku ---
	if err := h.DB.Delete(&b).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus buku")
	}

	return helper.JsonDeleted(c, "Buku berhasil dihapus", fiber.Map{
		"book_id": bookID,
	})
}
