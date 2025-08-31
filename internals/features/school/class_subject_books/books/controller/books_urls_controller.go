// file: internals/features/books/controller/book_url_controller.go
package controller

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	bookdto "masjidku_backend/internals/features/school/class_subject_books/books/dto"
	bookmodel "masjidku_backend/internals/features/school/class_subject_books/books/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
)

type BookURLController struct {
	DB        *gorm.DB
	validator *validator.Validate
}

func NewBookURLController(db *gorm.DB) *BookURLController {
	return &BookURLController{
		DB:        db,
		validator: validator.New(),
	}
}

/* =========================================================
 * CREATE (JSON)
 * POST /api/a/book-urls
 * Body: CreateBookURLRequest
 * ========================================================= */
func (ctl *BookURLController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	// ============== MULTIPART: upload + convert to WebP ==============
	if isMultipart {
		// Wajib: book_id
		bookIDStr := strings.TrimSpace(c.FormValue("book_url_book_id"))
		if bookIDStr == "" {
			return fiber.NewError(fiber.StatusBadRequest, "book_url_book_id wajib diisi")
		}
		bookID, e := uuid.Parse(bookIDStr)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "book_url_book_id tidak valid")
		}

		// Pastikan book milik masjid yang aktif
		var ok bool
		if err := ctl.DB.
			Raw(`
				SELECT EXISTS (
					SELECT 1 FROM books
					WHERE books_id = ? AND books_masjid_id = ? AND books_deleted_at IS NULL
				)`, bookID, masjidID).
			Scan(&ok).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa buku")
		}
		if !ok {
			return fiber.NewError(fiber.StatusNotFound, "Buku tidak ditemukan / berbeda masjid")
		}

		// Optional: label & type
		var lbl *string
		if v := strings.TrimSpace(c.FormValue("book_url_label")); v != "" {
			lbl = &v
		}
		typ := strings.TrimSpace(c.FormValue("book_url_type"))

		// --- Ambil file (prioritas: book_url_href -> file -> book_url_file)
		var (
			href string
		)
		fh, ferr := c.FormFile("book_url_href") // <- dukung "book_url_href" sebagai FILE
		if ferr != nil || fh == nil || fh.Size == 0 {
			fh, ferr = c.FormFile("file") // kompat lama
		}
		if ferr != nil || fh == nil || fh.Size == 0 {
			fh, ferr = c.FormFile("book_url_file") // alternatif
		}

		if ferr == nil && fh != nil && fh.Size > 0 {
			// --- Mode FILE ---
			if fh.Size > 5*1024*1024 {
				return fiber.NewError(fiber.StatusRequestEntityTooLarge, "Ukuran gambar maksimal 5MB")
			}

			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return fiber.NewError(fiber.StatusBadGateway, "OSS init gagal")
			}

			dir := fmt.Sprintf("masjids/%s/book-urls/%s", masjidID.String(), bookID.String())
			newURL, upErr := svc.UploadAsWebP(c.Context(), fh, dir)
			if upErr != nil {
				low := strings.ToLower(upErr.Error())
				if strings.Contains(low, "format tidak didukung") {
					return fiber.NewError(fiber.StatusUnsupportedMediaType, "Unsupported image format (pakai jpg/png/webp)")
				}
				return fiber.NewError(fiber.StatusBadGateway, "Gagal upload file")
			}
			href = newURL

			// default type saat ada file tapi type kosong → cover
			if strings.TrimSpace(typ) == "" {
				typ = bookdto.BookURLTypeCover
			}
		} else {
			// --- Mode URL TEXT ---
			h := strings.TrimSpace(c.FormValue("book_url_href"))
			if h == "" {
				return fiber.NewError(fiber.StatusBadRequest, "Wajib mengirim file atau book_url_href")
			}
			href = h
			if strings.TrimSpace(typ) == "" {
				typ = bookdto.BookURLTypeDesc // default aman saat non-file
			}
		}

		mdl := bookmodel.BookURLModel{
			BookURLMasjidID:           masjidID,
			BookURLBookID:             bookID,
			BookURLLabel:              lbl,
			BookURLType:               bookdto.NormalizeBookURLType(typ),
			BookURLHref:               href, // sesuai kolom SQL
		}

		if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fiber.NewError(fiber.StatusConflict, "URL sudah ada untuk buku ini")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan URL buku")
		}

		resp := bookdto.NewBookURLResponse(mdl)
		return c.JSON(fiber.Map{
			"message": "Berhasil membuat URL buku",
			"data":    resp,
		})
	}

	// ============== JSON: perilaku lama (tanpa file) ==============
	var req bookdto.CreateBookURLRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Pastikan book milik masjid yang aktif
	var ok bool
	if err := ctl.DB.
		Raw(`
			SELECT EXISTS (
				SELECT 1 FROM books
				WHERE books_id = ? AND books_masjid_id = ? AND books_deleted_at IS NULL
			)`, req.BookURLBookID, masjidID).
		Scan(&ok).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa buku")
	}
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Buku tidak ditemukan / berbeda masjid")
	}

	mdl := req.ToModel(masjidID)

	if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fiber.NewError(fiber.StatusConflict, "URL sudah ada untuk buku ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan URL buku")
	}

	resp := bookdto.NewBookURLResponse(mdl)
	return c.JSON(fiber.Map{
		"message": "Berhasil membuat URL buku",
		"data":    resp,
	})
}


/* =========================================================
 * UPDATE (partial JSON)
 * PATCH /api/a/book-urls/:id
 * Body: UpdateBookURLRequest
 * ========================================================= */
/* =========================================================
 * UPDATE (JSON or MULTIPART, partial + file rotate)
 * PATCH /api/a/book-urls/:id
 * ========================================================= */
func (ctl *BookURLController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl bookmodel.BookURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("book_url_id = ? AND book_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	if isMultipart {
		// ====== MULTIPART ======
		// Label (opsional)
		if v := strings.TrimSpace(c.FormValue("book_url_label")); v != "" {
			mdl.BookURLLabel = &v
		}
		// Type (opsional)
		if t := strings.TrimSpace(c.FormValue("book_url_type")); t != "" {
			mdl.BookURLType = bookdto.NormalizeBookURLType(t)
		}

		// Jika ada file baru → upload + rotasi file lama ke spam/
		if fh, ferr := c.FormFile("file"); ferr == nil && fh != nil && fh.Size > 0 {
			// batas ukuran contoh 5MB
			if fh.Size > 5*1024*1024 {
				return fiber.NewError(fiber.StatusRequestEntityTooLarge, "Ukuran gambar maksimal 5MB")
			}

			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return fiber.NewError(fiber.StatusBadGateway, "OSS init gagal")
			}

			// simpan di masjids/<masjid_id>/book-urls/<book_id>
			dir := fmt.Sprintf("masjids/%s/book-urls/%s", masjidID.String(), mdl.BookURLBookID.String())
			newURL, upErr := svc.UploadAsWebP(c.Context(), fh, dir)
			if upErr != nil {
				low := strings.ToLower(upErr.Error())
				if strings.Contains(low, "format tidak didukung") {
					return fiber.NewError(fiber.StatusUnsupportedMediaType, "Unsupported image format (pakai jpg/png/webp)")
				}
				return fiber.NewError(fiber.StatusBadGateway, "Gagal upload file")
			}

			// Pindahkan file lama (jika ada) ke spam/ + jadwalkan hapus
			if old := strings.TrimSpace(mdl.BookURLHref); old != "" {
				if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(old, 15*time.Second); mvErr == nil {
					mdl.BookURLTrashURL = &spamURL
				} else {
					// kalau gagal pindah, tetap catat old sebagai trash_url agar reaper bisa mengambil aksi
					mdl.BookURLTrashURL = &old
				}
				due := time.Now().Add(7 * 24 * time.Hour)
				mdl.BookURLDeletePendingUntil = &due
			}

			// Set href ke URL baru
			mdl.BookURLHref = newURL
		} else {
			// Tanpa file → boleh update href manual (opsional)
			// Ambil seluruh form supaya bisa cek "apakah field dikirim?"
			form, _ := c.MultipartForm()

			if h := strings.TrimSpace(c.FormValue("book_url_href")); h != "" {
				mdl.BookURLHref = h
			}

			// Optional: trash_url manual
			if form != nil {
				if vals, exists := form.Value["book_url_trash_url"]; exists {
					// field dikirim (meski kosong)
					if len(vals) == 0 || strings.TrimSpace(vals[0]) == "" {
						mdl.BookURLTrashURL = nil
					} else {
						tr := strings.TrimSpace(vals[0])
						mdl.BookURLTrashURL = &tr
					}
				}
			}

			// Optional: delete_pending_until manual (RFC3339)
			if d := strings.TrimSpace(c.FormValue("book_url_delete_pending_until")); d != "" {
				if t, e := time.Parse(time.RFC3339, d); e == nil {
					mdl.BookURLDeletePendingUntil = &t
				}
			}
		}
	} else {
		// ====== JSON ======
		var req bookdto.UpdateBookURLRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
		}
		if err := ctl.validator.Struct(req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Terapkan perubahan
		if req.BookURLLabel != nil {
			v := strings.TrimSpace(*req.BookURLLabel)
			if v == "" {
				mdl.BookURLLabel = nil
			} else {
				mdl.BookURLLabel = &v
			}
		}
		if req.BookURLType != nil {
			mdl.BookURLType = bookdto.NormalizeBookURLType(*req.BookURLType)
		}
		if req.BookURLHref != nil {
			mdl.BookURLHref = strings.TrimSpace(*req.BookURLHref)
		}
		if req.BookURLTrashURL != nil {
			tr := strings.TrimSpace(*req.BookURLTrashURL)
			if tr == "" {
				mdl.BookURLTrashURL = nil
			} else {
				mdl.BookURLTrashURL = &tr
			}
		}
		if req.BookURLDeletePendingUntil != nil {
			mdl.BookURLDeletePendingUntil = req.BookURLDeletePendingUntil
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fiber.NewError(fiber.StatusConflict, "URL sudah ada untuk buku ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	resp := bookdto.NewBookURLResponse(mdl)
	return c.JSON(fiber.Map{
		"message": "Berhasil memperbarui",
		"data":    resp,
	})
}


/* =========================================================
 * GET BY ID
 * GET /api/a/book-urls/:id
 * ========================================================= */
func (ctl *BookURLController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl bookmodel.BookURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("book_url_id = ? AND book_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := bookdto.NewBookURLResponse(mdl)
	return c.JSON(fiber.Map{
		"data": resp,
	})
}

/* =========================================================
 * FILTER / LIST
 * GET /api/a/book-urls/filter?book_id=&type=&search=&only_alive=&page=&limit=&sort=
 * ========================================================= */
func (ctl *BookURLController) Filter(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	var q bookdto.FilterBookURLRequest
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctl.validator.Struct(q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	page := 1
	limit := 20
	if q.Page != nil && *q.Page > 0 {
		page = *q.Page
	}
	if q.Limit != nil && *q.Limit > 0 {
		limit = *q.Limit
	}
	offset := (page - 1) * limit

	dbq := ctl.DB.WithContext(c.Context()).
		Model(&bookmodel.BookURLModel{}).
		Where("book_url_masjid_id = ?", masjidID)

	// only alive (default true)
	onlyAlive := true
	if q.OnlyAlive != nil {
		onlyAlive = *q.OnlyAlive
	}
	if onlyAlive {
		dbq = dbq.Where("book_url_deleted_at IS NULL")
	}

	if q.BookID != nil {
		dbq = dbq.Where("book_url_book_id = ?", *q.BookID)
	}
	if q.Type != nil && strings.TrimSpace(*q.Type) != "" {
		dbq = dbq.Where("book_url_type = ?", bookdto.NormalizeBookURLType(*q.Type))
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.TrimSpace(*q.Search) + "%"
		dbq = dbq.Where("(book_url_label ILIKE ? OR book_url_href ILIKE ?)", s, s)
	}

	// Sorting
	order := "book_url_created_at DESC"
	if q.Sort != nil {
		switch *q.Sort {
		case "created_at_asc":
			order = "book_url_created_at ASC"
		case "label_asc":
			order = "book_url_label ASC NULLS LAST, book_url_created_at DESC"
		case "label_desc":
			order = "book_url_label DESC NULLS LAST, book_url_created_at DESC"
		}
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var rows []bookmodel.BookURLModel
	if err := dbq.Order(order).Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resps := make([]bookdto.BookURLResponse, 0, len(rows))
	for _, m := range rows {
		resps = append(resps, bookdto.NewBookURLResponse(m))
	}

	return c.JSON(fiber.Map{
		"data": resps,
		"meta": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}


/* =========================================================
 * DELETE (soft) + move file to spam/
 * DELETE /api/a/book-urls/:id
 * ========================================================= */
func (ctl *BookURLController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl bookmodel.BookURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("book_url_id = ? AND book_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 1) Coba pindahkan file aktif ke spam/ (best-effort)
	var spamURL string
	if h := strings.TrimSpace(mdl.BookURLHref); h != "" {
		if s, mvErr := helperOSS.MoveToSpamByPublicURLENV(h, 15*time.Second); mvErr == nil {
			spamURL = s
		} else {
			// gagal pindah → tetap catat href lama agar reaper bisa menangani nanti
			spamURL = h
		}

		due := time.Now().Add(7 * 24 * time.Hour)
		mdl.BookURLTrashURL = &spamURL
		mdl.BookURLDeletePendingUntil = &due

		// Simpan status trash sebelum soft-delete (biar ikut terekam di row)
		if err := ctl.DB.WithContext(c.Context()).Save(&mdl).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui status trash")
		}
	}

	// 2) Soft delete row
	if err := ctl.DB.WithContext(c.Context()).Delete(&mdl).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil menghapus",
		"data": fiber.Map{
			"book_url_id":              mdl.BookURLID,
			"moved_to_spam_url":        spamURL,                          // bisa kosong kalau href kosong
			"delete_pending_until":     mdl.BookURLDeletePendingUntil,    // diisi jika href ada
			"deleted_at":               time.Now(),
		},
	})
}
