// file: internals/features/announcements/urls/controller/announcement_url_controller.go
package controller

import (
	"strings"
	"time"

	dto "masjidku_backend/internals/features/school/others/announcements/dto"
	model "masjidku_backend/internals/features/school/others/announcements/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnnouncementURLController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewAnnouncementURLController(db *gorm.DB, v interface{ Struct(any) error }) *AnnouncementURLController {
	return &AnnouncementURLController{DB: db, Validator: v}
}

/*
=========================================================

	CREATE (staff only)

=========================================================
*/
func (ctl *AnnouncementURLController) Create(c *fiber.Ctx) error {
	// 1) Parse payload
	var p dto.CreateAnnouncementURLRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi ringan
	p.AnnouncementURLKind = strings.TrimSpace(p.AnnouncementURLKind)
	if p.AnnouncementURLHref != nil {
		t := strings.TrimSpace(*p.AnnouncementURLHref)
		if t == "" {
			p.AnnouncementURLHref = nil
		} else {
			p.AnnouncementURLHref = &t
		}
	}
	if p.AnnouncementURLObjectKey != nil {
		t := strings.TrimSpace(*p.AnnouncementURLObjectKey)
		if t == "" {
			p.AnnouncementURLObjectKey = nil
		} else {
			p.AnnouncementURLObjectKey = &t
		}
	}
	if p.AnnouncementURLLabel != nil {
		t := strings.TrimSpace(*p.AnnouncementURLLabel)
		if t == "" {
			p.AnnouncementURLLabel = nil
		} else {
			p.AnnouncementURLLabel = &t
		}
	}

	// 2) Validasi DTO
	if err := ctl.Validator.Struct(p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 3) Resolve masjid context + guard staff
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
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

	// 4) Validasi: announcement harus milik masjid & masih hidup
	var ann struct {
		MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
		DeletedAt *time.Time `gorm:"column:deleted_at"`
	}
	if err := ctl.DB.Table("announcements").
		Select("announcement_masjid_id AS masjid_id, announcement_deleted_at AS deleted_at").
		Where("announcement_id = ?", p.AnnouncementURLAnnouncementId).
		Take(&ann).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusBadRequest, "Announcement tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil announcement")
	}
	if ann.MasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Announcement bukan milik masjid Anda")
	}
	if ann.DeletedAt != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Announcement sudah dihapus")
	}

	// 5) Simpan (handle primary unik per (announcement, kind))
	var created model.AnnouncementURLModel
	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		// Jika primary → nonaktifkan yang lain dulu agar tidak kena unique partial index
		if p.AnnouncementURLIsPrimary {
			if err := tx.Model(&model.AnnouncementURLModel{}).
				Where(`
					announcement_url_masjid_id = ?
					AND announcement_url_announcement_id = ?
					AND announcement_url_kind = ?
					AND announcement_url_deleted_at IS NULL
				`, masjidID, p.AnnouncementURLAnnouncementId, p.AnnouncementURLKind).
				Updates(map[string]any{
					"announcement_url_is_primary": false,
					"announcement_url_updated_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}

		m := p.ToModel(masjidID)
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "uq_ann_urls_primary_per_kind_alive") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Hanya boleh satu URL primary per (announcement, kind)")
			}
			return err
		}
		created = m
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 6) Response
	return helper.JsonCreated(c, "Berhasil menambahkan URL pengumuman", dto.FromAnnouncementURLModel(created))
}

/*
=========================================================

	PATCH (staff only) — partial update + handle primary & object_key rotation

=========================================================
*/
func (ctl *AnnouncementURLController) Patch(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	// Ambil existing (alive)
	var ex model.AnnouncementURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_id = ? AND announcement_url_deleted_at IS NULL", id).
		Take(&ex).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard staff pada masjid terkait
	if err := helperAuth.EnsureStaffMasjid(c, ex.AnnouncementURLMasjidId); err != nil {
		return err
	}

	// Parse payload
	var p dto.UpdateAnnouncementURLRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Normalisasi ringan
	if p.AnnouncementURLKind != nil {
		t := strings.TrimSpace(*p.AnnouncementURLKind)
		p.AnnouncementURLKind = &t
	}
	if p.AnnouncementURLHref != nil {
		t := strings.TrimSpace(*p.AnnouncementURLHref)
		if t == "" {
			p.AnnouncementURLHref = nil
		} else {
			p.AnnouncementURLHref = &t
		}
	}
	if p.AnnouncementURLObjectKey != nil {
		t := strings.TrimSpace(*p.AnnouncementURLObjectKey)
		if t == "" {
			p.AnnouncementURLObjectKey = nil
		} else {
			p.AnnouncementURLObjectKey = &t
		}
	}
	if p.AnnouncementURLLabel != nil {
		t := strings.TrimSpace(*p.AnnouncementURLLabel)
		if t == "" {
			p.AnnouncementURLLabel = nil
		} else {
			p.AnnouncementURLLabel = &t
		}
	}
	// Validasi DTO (omitempty -> partial)
	if err := ctl.Validator.Struct(p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Hitung "nilai efektif" (setelah patch) untuk keperluan set primary & scope unik
	effAnnouncementID := ex.AnnouncementURLAnnouncementId
	if p.AnnouncementURLAnnouncementId != nil {
		effAnnouncementID = *p.AnnouncementURLAnnouncementId
	}
	effKind := ex.AnnouncementURLKind
	if p.AnnouncementURLKind != nil {
		effKind = *p.AnnouncementURLKind
	}

	// Jika announcement_id diganti → validasi kepemilikan & alive
	if p.AnnouncementURLAnnouncementId != nil {
		var ann struct {
			MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
			DeletedAt *time.Time `gorm:"column:deleted_at"`
		}
		if err := ctl.DB.Table("announcements").
			Select("announcement_masjid_id AS masjid_id, announcement_deleted_at AS deleted_at").
			Where("announcement_id = ?", *p.AnnouncementURLAnnouncementId).
			Take(&ann).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return helper.JsonError(c, fiber.StatusBadRequest, "Announcement tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil announcement")
		}
		if ann.MasjidID != ex.AnnouncementURLMasjidId {
			return helper.JsonError(c, fiber.StatusForbidden, "Announcement bukan milik masjid Anda")
		}
		if ann.DeletedAt != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Announcement sudah dihapus")
		}
	}

	// Siapkan patch map
	patch := map[string]any{
		"announcement_url_updated_at": time.Now(),
	}

	if p.AnnouncementURLAnnouncementId != nil {
		patch["announcement_url_announcement_id"] = *p.AnnouncementURLAnnouncementId
	}
	if p.AnnouncementURLKind != nil {
		patch["announcement_url_kind"] = *p.AnnouncementURLKind
	}
	if p.AnnouncementURLHref != nil || p.AnnouncementURLHref == nil {
		// Jika eksplisit kirim null, tetap apply null
		if p.AnnouncementURLHref == nil {
			patch["announcement_url_href"] = gorm.Expr("NULL")
		} else {
			patch["announcement_url_href"] = *p.AnnouncementURLHref
		}
	}
	// Object key rotation
	if p.AnnouncementURLObjectKey != nil {
		newKey := strings.TrimSpace(*p.AnnouncementURLObjectKey)
		oldKey := ""
		if ex.AnnouncementURLObjectKey != nil {
			oldKey = strings.TrimSpace(*ex.AnnouncementURLObjectKey)
		}
		if newKey != oldKey {
			patch["announcement_url_object_key"] = func() any {
				if newKey == "" {
					return gorm.Expr("NULL")
				}
				return newKey
			}()
			// Simpan old ke *_old (kalau ada nilai lama)
			if oldKey != "" {
				patch["announcement_url_object_key_old"] = oldKey
				// default retensi 30 hari jika tidak diset eksplisit
				if p.AnnouncementURLDeletePendingUntil != nil {
					patch["announcement_url_delete_pending_until"] = *p.AnnouncementURLDeletePendingUntil
				} else {
					patch["announcement_url_delete_pending_until"] = time.Now().Add(30 * 24 * time.Hour)
				}
			} else {
				patch["announcement_url_object_key_old"] = gorm.Expr("NULL")
				if p.AnnouncementURLDeletePendingUntil != nil {
					patch["announcement_url_delete_pending_until"] = *p.AnnouncementURLDeletePendingUntil
				} else {
					patch["announcement_url_delete_pending_until"] = gorm.Expr("NULL")
				}
			}
		}
	}
	if p.AnnouncementURLLabel != nil || p.AnnouncementURLLabel == nil {
		if p.AnnouncementURLLabel == nil {
			patch["announcement_url_label"] = gorm.Expr("NULL")
		} else {
			patch["announcement_url_label"] = *p.AnnouncementURLLabel
		}
	}
	if p.AnnouncementURLOrder != nil {
		patch["announcement_url_order"] = *p.AnnouncementURLOrder
	}
	if p.AnnouncementURLIsPrimary != nil {
		patch["announcement_url_is_primary"] = *p.AnnouncementURLIsPrimary
	}
	if p.AnnouncementURLDeletePendingUntil != nil && p.AnnouncementURLObjectKey == nil {
		// izinkan override manual tanpa ganti object_key
		patch["announcement_url_delete_pending_until"] = *p.AnnouncementURLDeletePendingUntil
	}

	//  Transaksi: kalau set primary → matikan yang lain dulu (di scope efektif)
	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		if p.AnnouncementURLIsPrimary != nil && *p.AnnouncementURLIsPrimary {
			if err := tx.Model(&model.AnnouncementURLModel{}).
				Where(`
					announcement_url_masjid_id = ?
					AND announcement_url_announcement_id = ?
					AND announcement_url_kind = ?
					AND announcement_url_deleted_at IS NULL
					AND announcement_url_id <> ?
				`,
					ex.AnnouncementURLMasjidId, effAnnouncementID, effKind, ex.AnnouncementURLId,
				).
				Updates(map[string]any{
					"announcement_url_is_primary": false,
					"announcement_url_updated_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&model.AnnouncementURLModel{}).
			Where("announcement_url_id = ?", ex.AnnouncementURLId).
			Updates(patch).Error; err != nil {

			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "uq_ann_urls_primary_per_kind_alive") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Hanya boleh satu URL primary per (announcement, kind)")
			}
			return err
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
		First(&ex, "announcement_url_id = ?", ex.AnnouncementURLId).Error

	return helper.JsonUpdated(c, "Berhasil memperbarui URL pengumuman", dto.FromAnnouncementURLModel(ex))
}

/*
=========================================================

	DELETE (soft delete, staff only) + opsi set delete_pending untuk purge

=========================================================
*/
func (ctl *AnnouncementURLController) Delete(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var ex model.AnnouncementURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_id = ? AND announcement_url_deleted_at IS NULL", id).
		Take(&ex).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard staff
	if err := helperAuth.EnsureStaffMasjid(c, ex.AnnouncementURLMasjidId); err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]any{
		"announcement_url_deleted_at": &now,
		"announcement_url_updated_at": now,
	}

	// Jika ada object_key aktif & belum ada delete_pending_until → set default 30 hari
	if ex.AnnouncementURLObjectKey != nil && strings.TrimSpace(*ex.AnnouncementURLObjectKey) != "" {
		if ex.AnnouncementURLDeletePendingUntil == nil {
			updates["announcement_url_delete_pending_until"] = now.Add(30 * 24 * time.Hour)
		}
	}

	if err := ctl.DB.WithContext(c.Context()).
		Model(&model.AnnouncementURLModel{}).
		Where("announcement_url_id = ?", ex.AnnouncementURLId).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Berhasil menghapus URL pengumuman", fiber.Map{
		"announcement_url_id": ex.AnnouncementURLId,
	})
}
