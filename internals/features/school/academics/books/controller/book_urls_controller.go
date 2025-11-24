// file: internals/features/library/books/controller/book_url_controller.go
package controller

import (
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/school/academics/books/dto"
	model "madinahsalam_backend/internals/features/school/academics/books/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookURLController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewBookURLController(db *gorm.DB, v interface{ Struct(any) error }) *BookURLController {
	return &BookURLController{DB: db, Validator: v}
}

/*
=========================================================

	CREATE (staff only)

=========================================================
*/
func (ctl *BookURLController) Create(c *fiber.Ctx) error {
	// 1) Parse payload
	var p dto.CreateBookURLRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi ringan
	p.BookURLKind = strings.TrimSpace(p.BookURLKind)
	if p.BookURLHref != nil {
		t := strings.TrimSpace(*p.BookURLHref)
		if t == "" {
			p.BookURLHref = nil
		} else {
			p.BookURLHref = &t
		}
	}
	if p.BookURLObjectKey != nil {
		t := strings.TrimSpace(*p.BookURLObjectKey)
		if t == "" {
			p.BookURLObjectKey = nil
		} else {
			p.BookURLObjectKey = &t
		}
	}
	if p.BookURLLabel != nil {
		t := strings.TrimSpace(*p.BookURLLabel)
		if t == "" {
			p.BookURLLabel = nil
		} else {
			p.BookURLLabel = &t
		}
	}

	// 2) Validasi DTO
	if err := ctl.Validator.Struct(p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 3) Resolve school context + guard staff
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	var schoolID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		schoolID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		schoolID = id
	default:
		id, er := helperAuth.GetSchoolIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "School context tidak ditemukan")
		}
		schoolID = id
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	// 4) Validasi: book harus milik school & masih hidup
	var bk struct {
		SchoolID  uuid.UUID  `gorm:"column:school_id"`
		DeletedAt *time.Time `gorm:"column:deleted_at"`
	}
	if err := ctl.DB.Table("books").
		Select("books_school_id AS school_id, books_deleted_at AS deleted_at").
		Where("books_id = ?", p.BookURLBookID).
		Take(&bk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusBadRequest, "Buku tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
	}
	if bk.SchoolID != schoolID {
		return helper.JsonError(c, fiber.StatusForbidden, "Buku bukan milik school Anda")
	}
	if bk.DeletedAt != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Buku sudah dihapus")
	}

	// 5) Simpan (handle primary unik per (book, kind))
	var created model.BookURLModel
	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		// Enforce tenant ke payload sebelum buat model
		p.BookURLSchoolID = schoolID

		// Jika primary → nonaktifkan yang lain dulu agar tidak kena unique partial index
		if p.BookURLIsPrimary != nil && *p.BookURLIsPrimary {
			if err := tx.Model(&model.BookURLModel{}).
				Where(`
					book_url_school_id = ?
					AND book_url_book_id = ?
					AND book_url_kind = ?
					AND book_url_deleted_at IS NULL
				`, schoolID, p.BookURLBookID, p.BookURLKind).
				Updates(map[string]any{
					"book_url_is_primary": false,
					"book_url_updated_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}

		m := p.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_book_urls_primary_per_kind_alive"):
				return fiber.NewError(fiber.StatusConflict, "Hanya boleh satu URL primary per (buku, kind)")
			case strings.Contains(msg, "uq_book_urls_book_href_alive"):
				return fiber.NewError(fiber.StatusConflict, "URL sudah ada untuk buku ini (aktif)")
			case strings.Contains(msg, "unique"):
				return fiber.NewError(fiber.StatusConflict, "Duplikat data URL")
			default:
				return err
			}
		}
		created = *m
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 6) Response
	return helper.JsonCreated(c, "Berhasil menambahkan URL buku", dto.FromBookURLModel(&created))
}

/*
=========================================================

	PATCH (staff only) — partial update + handle primary & object_key rotation

=========================================================
*/
func (ctl *BookURLController) Patch(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	// Ambil existing (alive)
	var ex model.BookURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("book_url_id = ? AND book_url_deleted_at IS NULL", id).
		Take(&ex).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard staff pada school terkait
	if err := helperAuth.EnsureStaffSchool(c, ex.BookURLSchoolID); err != nil {
		return err
	}

	// Parse payload
	var p dto.PatchBookURLRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Normalisasi ringan
	if p.BookURLKind != nil {
		t := strings.TrimSpace(*p.BookURLKind)
		p.BookURLKind = &t
	}
	if p.BookURLHref != nil {
		t := strings.TrimSpace(*p.BookURLHref)
		if t == "" {
			p.BookURLHref = nil
		} else {
			p.BookURLHref = &t
		}
	}
	if p.BookURLObjectKey != nil {
		t := strings.TrimSpace(*p.BookURLObjectKey)
		if t == "" {
			p.BookURLObjectKey = nil
		} else {
			p.BookURLObjectKey = &t
		}
	}
	if p.BookURLLabel != nil {
		t := strings.TrimSpace(*p.BookURLLabel)
		if t == "" {
			p.BookURLLabel = nil
		} else {
			p.BookURLLabel = &t
		}
	}
	// Validasi DTO (omitempty -> partial)
	if err := ctl.Validator.Struct(p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Hitung "nilai efektif" (setelah patch) untuk keperluan set primary & scope unik
	effBookID := ex.BookURLBookID
	if p.BookURLBookID != nil {
		effBookID = *p.BookURLBookID
	}
	effKind := ex.BookURLKind
	if p.BookURLKind != nil {
		effKind = *p.BookURLKind
	}

	// Jika book_id diganti → validasi kepemilikan & alive
	if p.BookURLBookID != nil {
		var bk struct {
			SchoolID  uuid.UUID  `gorm:"column:school_id"`
			DeletedAt *time.Time `gorm:"column:deleted_at"`
		}
		if err := ctl.DB.Table("books").
			Select("books_school_id AS school_id, books_deleted_at AS deleted_at").
			Where("books_id = ?", *p.BookURLBookID).
			Take(&bk).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return helper.JsonError(c, fiber.StatusBadRequest, "Buku tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
		}
		if bk.SchoolID != ex.BookURLSchoolID {
			return helper.JsonError(c, fiber.StatusForbidden, "Buku bukan milik school Anda")
		}
		if bk.DeletedAt != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Buku sudah dihapus")
		}
	}

	// Siapkan patch map
	patch := map[string]any{
		"book_url_updated_at": time.Now(),
	}

	if p.BookURLBookID != nil {
		patch["book_url_book_id"] = *p.BookURLBookID
	}
	if p.BookURLKind != nil {
		patch["book_url_kind"] = *p.BookURLKind
	}
	if p.BookURLHref != nil || p.BookURLHref == nil {
		// Jika eksplisit kirim null, tetap apply null
		if p.BookURLHref == nil {
			patch["book_url_href"] = gorm.Expr("NULL")
		} else {
			patch["book_url_href"] = *p.BookURLHref
		}
	}
	// Object key rotation
	if p.BookURLObjectKey != nil {
		newKey := strings.TrimSpace(*p.BookURLObjectKey)
		oldKey := ""
		if ex.BookURLObjectKey != nil {
			oldKey = strings.TrimSpace(*ex.BookURLObjectKey)
		}
		if newKey != oldKey {
			patch["book_url_object_key"] = func() any {
				if newKey == "" {
					return gorm.Expr("NULL")
				}
				return newKey
			}()
			// Simpan old ke *_old (kalau ada nilai lama)
			if oldKey != "" {
				patch["book_url_object_key_old"] = oldKey
				// default retensi 30 hari
				patch["book_url_delete_pending_until"] = time.Now().Add(30 * 24 * time.Hour)
			} else {
				patch["book_url_object_key_old"] = gorm.Expr("NULL")
				patch["book_url_delete_pending_until"] = gorm.Expr("NULL")
			}
		}
	}
	if p.BookURLLabel != nil || p.BookURLLabel == nil {
		if p.BookURLLabel == nil {
			patch["book_url_label"] = gorm.Expr("NULL")
		} else {
			patch["book_url_label"] = *p.BookURLLabel
		}
	}
	if p.BookURLOrder != nil {
		patch["book_url_order"] = *p.BookURLOrder
	}
	if p.BookURLIsPrimary != nil {
		patch["book_url_is_primary"] = *p.BookURLIsPrimary
	}

	//  Transaksi: kalau set primary → matikan yang lain dulu (di scope efektif)
	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		if p.BookURLIsPrimary != nil && *p.BookURLIsPrimary {
			if err := tx.Model(&model.BookURLModel{}).
				Where(`
					book_url_school_id = ?
					AND book_url_book_id = ?
					AND book_url_kind = ?
					AND book_url_deleted_at IS NULL
					AND book_url_id <> ?
				`,
					ex.BookURLSchoolID, effBookID, effKind, ex.BookURLID,
				).
				Updates(map[string]any{
					"book_url_is_primary": false,
					"book_url_updated_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&model.BookURLModel{}).
			Where("book_url_id = ?", ex.BookURLID).
			Updates(patch).Error; err != nil {

			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_book_urls_primary_per_kind_alive"):
				return fiber.NewError(fiber.StatusConflict, "Hanya boleh satu URL primary per (buku, kind)")
			case strings.Contains(msg, "uq_book_urls_book_href_alive"):
				return fiber.NewError(fiber.StatusConflict, "URL sudah ada untuk buku ini (aktif)")
			case strings.Contains(msg, "unique"):
				return fiber.NewError(fiber.StatusConflict, "Duplikat data URL")
			default:
				return err
			}
		}
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Reload untuk response
	_ = ctl.DB.WithContext(c.Context()).
		First(&ex, "book_url_id = ?", ex.BookURLID).Error

	return helper.JsonUpdated(c, "Berhasil memperbarui URL buku", dto.FromBookURLModel(&ex))
}

/*
=========================================================

	DELETE (soft delete, staff only) + opsi set delete_pending untuk purge

=========================================================
*/
func (ctl *BookURLController) Delete(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var ex model.BookURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("book_url_id = ? AND book_url_deleted_at IS NULL", id).
		Take(&ex).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard staff
	if err := helperAuth.EnsureStaffSchool(c, ex.BookURLSchoolID); err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]any{
		"book_url_deleted_at": &now,
		"book_url_updated_at": now,
	}

	// Jika ada object_key aktif & belum ada delete_pending_until → set default 30 hari
	if ex.BookURLObjectKey != nil && strings.TrimSpace(*ex.BookURLObjectKey) != "" {
		if !ex.BookURLDeletePendingUntil.Valid {
			updates["book_url_delete_pending_until"] = now.Add(30 * 24 * time.Hour)
		}
	}

	if err := ctl.DB.WithContext(c.Context()).
		Model(&model.BookURLModel{}).
		Where("book_url_id = ?", ex.BookURLID).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Berhasil menghapus URL buku", fiber.Map{
		"book_url_id": ex.BookURLID,
	})
}
