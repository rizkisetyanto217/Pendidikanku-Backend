// internals/features/lembaga/announcements/announcement/controller/announcement_controller.go
package controller

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	annDTO "masjidku_backend/internals/features/lembaga/announcements/announcement/dto"
	annModel "masjidku_backend/internals/features/lembaga/announcements/announcement/model"
	helper "masjidku_backend/internals/helpers"
)

type AnnouncementController struct{ DB *gorm.DB }

func NewAnnouncementController(db *gorm.DB) *AnnouncementController { return &AnnouncementController{DB: db} }

var validateAnnouncement = validator.New()


// ===================== GET BY ID =====================
// GET /admin/announcements/:id
func (h *AnnouncementController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findWithTenantGuard(annID, masjidID)
	if err != nil {
		return err
	}
	return helper.Success(c, "OK", annDTO.NewAnnouncementResponse(m))
}


// ===================== LIST =====================
// GET /admin/announcements
func (h *AnnouncementController) List(c *fiber.Ctx) error {
	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil { return err }

	var q annDTO.ListAnnouncementQuery
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := h.DB.Model(&annModel.AnnouncementModel{}).
		Where("announcement_masjid_id IN ?", masjidIDs).
		Preload("Theme", func(db *gorm.DB) *gorm.DB {
			// pilih kolom seperlunya biar hemat payload; sesuaikan nama kolom model tema
			return db.
				Select("announcement_themes_id, announcement_themes_masjid_id, announcement_themes_name, announcement_themes_color").
				Where("announcement_themes_deleted_at IS NULL")
		})

	// ===== Filter: Theme
	if q.ThemeID != nil {
		tx = tx.Where("announcement_theme_id = ?", *q.ThemeID)
	}

	// ===== Filter: Section vs Global (NULL) =====
	includeGlobal := true
	if q.IncludeGlobal != nil { includeGlobal = *q.IncludeGlobal }
	onlyGlobal := q.OnlyGlobal != nil && *q.OnlyGlobal

	sectionIDsCSV := strings.TrimSpace(c.Query("section_ids"))
	sectionIDs, parseErr := parseUUIDsCSV(sectionIDsCSV)
	if parseErr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "section_ids berisi UUID tidak valid")
	}

	switch {
	case onlyGlobal:
		tx = tx.Where("announcement_class_section_id IS NULL")

	case len(sectionIDs) > 0:
		if includeGlobal {
			tx = tx.Where("(announcement_class_section_id IN ? OR announcement_class_section_id IS NULL)", sectionIDs)
		} else {
			tx = tx.Where("announcement_class_section_id IN ?", sectionIDs)
		}

	case q.SectionID != nil:
		if includeGlobal {
			tx = tx.Where("(announcement_class_section_id = ? OR announcement_class_section_id IS NULL)", *q.SectionID)
		} else {
			tx = tx.Where("announcement_class_section_id = ?", *q.SectionID)
		}
	}

	// ===== Filter: Attachment
	if q.HasAttachment != nil {
		if *q.HasAttachment {
			tx = tx.Where("announcement_attachment_url IS NOT NULL AND announcement_attachment_url <> ''")
		} else {
			tx = tx.Where("(announcement_attachment_url IS NULL OR announcement_attachment_url = '')")
		}
	}

	// ===== Filter: is_active
	if q.IsActive != nil {
		tx = tx.Where("announcement_is_active = ?", *q.IsActive)
	}

	// ===== Filter: Date range (YYYY-MM-DD)
	parseDate := func(s string) (time.Time, error) { return time.Parse("2006-01-02", strings.TrimSpace(s)) }
	if q.DateFrom != nil && strings.TrimSpace(*q.DateFrom) != "" {
		t, err := parseDate(*q.DateFrom); if err != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		tx = tx.Where("announcement_date >= ?", t)
	}
	if q.DateTo != nil && strings.TrimSpace(*q.DateTo) != "" {
		t, err := parseDate(*q.DateTo); if err != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		tx = tx.Where("announcement_date <= ?", t)
	}

	// ===== Total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ===== Sorting
	sort := "date_desc"
	if q.Sort != nil { sort = strings.ToLower(strings.TrimSpace(*q.Sort)) }
	switch sort {
	case "date_asc":
		tx = tx.Order("announcement_date ASC")
	case "created_at_desc":
		tx = tx.Order("announcement_created_at DESC")
	case "created_at_asc":
		tx = tx.Order("announcement_created_at ASC")
	case "title_asc":
		tx = tx.Order("announcement_title ASC")
	case "title_desc":
		tx = tx.Order("announcement_title DESC")
	default:
		tx = tx.Order("announcement_date DESC")
	}

	// ===== Pagination safety
	if q.Limit <= 0 { q.Limit = 20 }
	if q.Limit > 100 { q.Limit = 100 }
	if q.Offset < 0 { q.Offset = 0 }

	// ===== Fetch
	var rows []annModel.AnnouncementModel
	if err := tx.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== Map ke response
	resp := make([]*annDTO.AnnouncementResponse, 0, len(rows))
	for i := range rows { resp = append(resp, annDTO.NewAnnouncementResponse(&rows[i])) }

	return helper.Success(c, "OK", fiber.Map{
		"limit":  q.Limit,
		"offset": q.Offset,
		"total":  total,
		"count":  len(resp),
		"items":  resp,
	})
}


// // helper lokal (boleh dipindah ke package helper)
func parseUUIDsCSV(csv string) ([]uuid.UUID, error) {
	if csv == "" { return nil, nil }
	parts := strings.Split(csv, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" { continue }
		id, err := uuid.Parse(s)
		if err != nil { return nil, err }
		out = append(out, id)
	}
	return out, nil
}


// internals/features/lembaga/announcements/announcement/controller/announcement_controller.go

// POST /admin/announcements
// POST /admin/announcements
func (h *AnnouncementController) Create(c *fiber.Ctx) error {
    masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
    if err != nil {
        return err
    }
    userID, err := helper.GetUserIDFromToken(c)
    if err != nil {
        return err
    }

    // role detection
    isAdmin := func() bool {
        if id, err := helper.GetMasjidIDFromToken(c); err == nil && id == masjidID {
            return true
        }
        return false
    }()
    isTeacher := func() bool {
        if id, err := helper.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
            return true
        }
        return false
    }()
    if !isAdmin && !isTeacher {
        return fiber.NewError(fiber.StatusForbidden, "Tidak diizinkan")
    }

    var req annDTO.CreateAnnouncementRequest
    ct := c.Get("Content-Type")

    if strings.HasPrefix(ct, "multipart/form-data") {
        // ----- parse text fields -----
        req.AnnouncementTitle = strings.TrimSpace(c.FormValue("announcement_title"))
        req.AnnouncementDate = strings.TrimSpace(c.FormValue("announcement_date"))
        req.AnnouncementContent = strings.TrimSpace(c.FormValue("announcement_content"))

        if v := strings.TrimSpace(c.FormValue("announcement_theme_id")); v != "" {
            if id, err := uuid.Parse(v); err == nil {
                req.AnnouncementThemeID = &id
            }
        }
        if v := strings.TrimSpace(c.FormValue("announcement_class_section_id")); v != "" {
            if id, err := uuid.Parse(v); err == nil {
                req.AnnouncementClassSectionID = &id
            }
        }
        if v := strings.TrimSpace(c.FormValue("announcement_is_active")); v != "" {
            b := strings.EqualFold(v, "true") || v == "1"
            req.AnnouncementIsActive = &b
        }

        // ----- file upload (dua key yang didukung) -----
	var fh *multipart.FileHeader
	if f, err := c.FormFile("attachment"); err == nil && f != nil {
		fh = f
	} else if f2, err2 := c.FormFile("announcement_attachment_url"); err2 == nil && f2 != nil {
		fh = f2
	}

	if fh != nil {
		// Simpan semua file ke folder "announcement" (tanpa subfolder masjid)
		folder := "announcement"

		publicURL, err := helper.UploadFileToSupabase(folder, fh)
		if err != nil {
			return helper.Error(c, fiber.StatusBadRequest, err.Error())
		}
		req.AnnouncementAttachmentURL = &publicURL
	}
    } else {
        // JSON
        if err := c.BodyParser(&req); err != nil {
            return helper.Error(c, fiber.StatusBadRequest, "Payload tidak valid")
        }
        // Catatan: untuk JSON, Attachment dikirim sebagai URL string via announcement_attachment_url
    }

    // Validasi DTO
    if err := validateAnnouncement.Struct(req); err != nil {
        return helper.ValidationError(c, err)
    }

    // aturan role
    if isAdmin && !isTeacher {
        req.AnnouncementClassSectionID = nil // GLOBAL
    }
    if isTeacher {
        if req.AnnouncementClassSectionID == nil || *req.AnnouncementClassSectionID == uuid.Nil {
            return helper.Error(c, fiber.StatusBadRequest, "Teacher wajib memilih section")
        }
        if err := h.ensureSectionBelongsToMasjid(*req.AnnouncementClassSectionID, masjidID); err != nil {
            return err
        }
    }

    // validasi theme bila ada
    if req.AnnouncementThemeID != nil {
        if err := h.ensureThemeBelongsToMasjid(*req.AnnouncementThemeID, masjidID); err != nil {
            return err
        }
    }

    // simpan
    m := req.ToModel(masjidID, userID)
    if err := h.DB.Create(m).Error; err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal membuat pengumuman")
    }
    return helper.Success(c, "Pengumuman berhasil dibuat", annDTO.NewAnnouncementResponse(m))
}


// Pastikan section milik masjid ini
func (h *AnnouncementController) ensureSectionBelongsToMasjid(sectionID, masjidID uuid.UUID) error {
	var cnt int64
	if err := h.DB.
		Table("class_sections").
		Joins("JOIN classes ON classes.class_id = class_sections.class_sections_class_id").
		Where("class_sections.class_sections_id = ? AND classes.class_masjid_id = ?", sectionID, masjidID).
		Count(&cnt).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
	}
	if cnt == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Section bukan milik masjid Anda")
	}
	return nil
}

// Pastikan theme milik masjid & belum soft-deleted
func (h *AnnouncementController) ensureThemeBelongsToMasjid(themeID, masjidID uuid.UUID) error {
	var cnt int64
	if err := h.DB.
		Table("announcement_themes").
		Where("announcement_themes_id = ? AND announcement_themes_masjid_id = ? AND announcement_themes_deleted_at IS NULL",
			themeID, masjidID).
		Count(&cnt).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi tema")
	}
	if cnt == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Tema tidak ditemukan atau bukan milik masjid Anda")
	}
	return nil
}


// --- tenant guard fetch
func (h *AnnouncementController) findWithTenantGuard(id, masjidID uuid.UUID) (*annModel.AnnouncementModel, error) {
	var m annModel.AnnouncementModel
	if err := h.DB.Where("announcement_id = ? AND announcement_masjid_id = ?", id, masjidID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Pengumuman tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil pengumuman")
	}
	return &m, nil
}


// ===================== UPDATE =====================
// PUT /admin/announcements/:id
// PUT /admin/announcements/:id
func (h *AnnouncementController) Update(c *fiber.Ctx) error {
    masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
    if err != nil { return err }
    userID, err := helper.GetUserIDFromToken(c)
    if err != nil { return err }

    annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
    if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

    isAdmin := func() bool {
        if id, err := helper.GetMasjidIDFromToken(c); err == nil && id == masjidID { return true }
        return false
    }()
    isTeacher := func() bool {
        if id, err := helper.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID { return true }
        return false
    }()
    if !isAdmin && !isTeacher {
        return fiber.NewError(fiber.StatusForbidden, "Tidak diizinkan")
    }

    existing, err := h.findWithTenantGuard(annID, masjidID)
    if err != nil { return err }

    var req annDTO.UpdateAnnouncementRequest
    ct := c.Get("Content-Type")
    removeAttachment := false

    if strings.HasPrefix(ct, "multipart/form-data") {
        if v := strings.TrimSpace(c.FormValue("announcement_title")); v != "" { req.AnnouncementTitle = &v }
        if v := strings.TrimSpace(c.FormValue("announcement_date")); v != "" { req.AnnouncementDate = &v }
        if v := strings.TrimSpace(c.FormValue("announcement_content")); v != "" { req.AnnouncementContent = &v }
        if v := strings.TrimSpace(c.FormValue("announcement_theme_id")); v != "" {
            if id, err := uuid.Parse(v); err == nil { req.AnnouncementThemeID = &id }
        }
        if v := strings.TrimSpace(c.FormValue("announcement_class_section_id")); v != "" {
            if id, err := uuid.Parse(v); err == nil { req.AnnouncementClassSectionID = &id }
        }
        if v := strings.TrimSpace(c.FormValue("announcement_is_active")); v != "" {
            b := strings.EqualFold(v, "true") || v == "1"; req.AnnouncementIsActive = &b
        }
        if v := strings.TrimSpace(c.FormValue("remove_attachment")); v != "" {
            removeAttachment = strings.EqualFold(v, "true") || v == "1"
        }

        // ==== upload file baru (opsional) ====
        if fh, ok := pickAnnFile(c); ok && fh != nil {
            // Hapus file lama (best-effort)
            if existing.AnnouncementAttachmentURL != nil && *existing.AnnouncementAttachmentURL != "" {
                if bucket, path, err := helper.ExtractSupabasePath(*existing.AnnouncementAttachmentURL); err == nil {
                    if unesc, errU := url.PathUnescape(path); errU == nil { path = unesc }
                    if errDel := helper.DeleteFromSupabase(bucket, path); errDel != nil {
                        log.Println("[WARN] gagal hapus file lama:", errDel)
                    }
                } else {
                    log.Println("[WARN] extract path gagal:", err)
                }
            }
            // Upload baru ke folder "announcement"
            folder := "announcement"
            publicURL, upErr := helper.UploadFileToSupabase(folder, fh)
            if upErr != nil {
                return helper.Error(c, fiber.StatusBadRequest, upErr.Error())
            }
            req.AnnouncementAttachmentURL = &publicURL
            // ada file baru → abaikan removeAttachment
            removeAttachment = false
        }
    } else {
        if err := c.BodyParser(&req); err != nil {
            return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
        }
    }

    // Validasi DTO
    if err := validateAnnouncement.Struct(req); err != nil {
        return helper.ValidationError(c, err)
    }

    // Admin: paksa global
    if isAdmin && !isTeacher {
        req.AnnouncementClassSectionID = nil
    }

    // Teacher: validasi section bila di-set
    if isTeacher && req.AnnouncementClassSectionID != nil && *req.AnnouncementClassSectionID != uuid.Nil {
        if err := h.ensureSectionBelongsToMasjid(*req.AnnouncementClassSectionID, masjidID); err != nil {
            return err
        }
    }

    // Hapus attachment jika diminta & tidak ada upload baru
    if removeAttachment && (req.AnnouncementAttachmentURL == nil) {
        if existing.AnnouncementAttachmentURL != nil && *existing.AnnouncementAttachmentURL != "" {
            if bucket, path, err := helper.ExtractSupabasePath(*existing.AnnouncementAttachmentURL); err == nil {
                if unesc, errU := url.PathUnescape(path); errU == nil { path = unesc }
                if errDel := helper.DeleteFromSupabase(bucket, path); errDel != nil {
                    log.Println("[WARN] gagal hapus file (remove_attachment):", errDel)
                }
            }
        }
        empty := ""           // ApplyToModel akan set ke NULL
        req.AnnouncementAttachmentURL = &empty
    }

    // Terapkan perubahan
    req.ApplyToModel(existing)

    // Enforce rule per role
    if isTeacher && existing.AnnouncementClassSectionID == nil {
        return fiber.NewError(fiber.StatusBadRequest, "Teacher wajib memilih section (tidak boleh global)")
    }
    if isTeacher && !isAdmin && existing.AnnouncementCreatedByUserID != userID {
        return fiber.NewError(fiber.StatusForbidden, "Hanya pembuat yang boleh mengubah pengumuman ini")
    }

    now := time.Now()
    existing.AnnouncementUpdatedAt = &now

    if err := h.DB.Model(&annModel.AnnouncementModel{}).
        Where("announcement_id = ? AND announcement_masjid_id = ?", existing.AnnouncementID, masjidID).
        Updates(existing).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui pengumuman")
    }

    return helper.JsonUpdated(c, "Pengumuman diperbarui", annDTO.NewAnnouncementResponse(existing))
}

// ambil file dari dua kemungkinan key
func pickAnnFile(c *fiber.Ctx) (*multipart.FileHeader, bool) {
    if f, err := c.FormFile("attachment"); err == nil && f != nil {
        return f, true
    }
    if f, err := c.FormFile("announcement_attachment_url"); err == nil && f != nil {
        return f, true
    }
    return nil, false
}


// ===================== DELETE =====================
// DELETE /admin/announcements/:id (soft delete)
// DELETE /admin/announcements/:id
func (h *AnnouncementController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err
	}
	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// role detection
	isAdmin := func() bool {
		if id, err := helper.GetMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	isTeacher := func() bool {
		if id, err := helper.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	if !isAdmin && !isTeacher {
		return fiber.NewError(fiber.StatusForbidden, "Tidak diizinkan")
	}

	existing, err := h.findWithTenantGuard(annID, masjidID)
	if err != nil {
		return err
	}

	// Rule delete:
	// - Admin: boleh hapus apapun di masjidnya.
	// - Teacher: TIDAK boleh hapus global (section NULL) dan hanya boleh hapus yang dia buat.
	if isTeacher && !isAdmin {
		if existing.AnnouncementClassSectionID == nil {
			return fiber.NewError(fiber.StatusForbidden, "Teacher tidak boleh menghapus pengumuman global")
		}
		if existing.AnnouncementCreatedByUserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Hanya pembuat yang boleh menghapus pengumuman ini")
		}
	}

	// Opsi hard delete
	force := strings.EqualFold(c.Query("force"), "true")

	// Jika hard delete → hapus file di Supabase (best-effort)
	if force && existing.AnnouncementAttachmentURL != nil && *existing.AnnouncementAttachmentURL != "" {
		if bucket, path, err := helper.ExtractSupabasePath(*existing.AnnouncementAttachmentURL); err == nil {
			if err := helper.DeleteFromSupabase(bucket, path); err != nil {
				fmt.Println("[WARN] gagal hapus file di Supabase:", err.Error())
			}
		} else {
			fmt.Println("[WARN] URL Supabase tidak valid, skip delete:", err.Error())
		}
	}

	db := h.DB
	if force {
		db = db.Unscoped()
	}

	if err := db.
		Where("announcement_id = ? AND announcement_masjid_id = ?", existing.AnnouncementID, masjidID).
		Delete(&annModel.AnnouncementModel{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus pengumuman")
	}

	msg := "Pengumuman dihapus"
	if force {
		msg = "Pengumuman dihapus permanen"
	}
	return helper.JsonDeleted(c, msg, fiber.Map{
		"announcement_id": existing.AnnouncementID,
	})
}
