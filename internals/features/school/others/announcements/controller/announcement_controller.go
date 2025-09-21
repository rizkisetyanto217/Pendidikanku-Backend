// internals/features/lembaga/announcements/announcement/controller/announcement_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	annDTO "masjidku_backend/internals/features/school/others/announcements/dto"
	annModel "masjidku_backend/internals/features/school/others/announcements/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"
)

type AnnouncementController struct{ DB *gorm.DB }

func NewAnnouncementController(db *gorm.DB) *AnnouncementController {
	return &AnnouncementController{DB: db}
}

var validateAnnouncement = validator.New()

// ===================== Utils =====================

func parseUUIDsCSV(s string) ([]uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid: %q", p)
		}
		out = append(out, id)
	}
	return out, nil
}

// =======================================
// Masjid resolver (izin DKM atau member)
// =======================================

func resolveMasjidIDUsingHelpers(c *fiber.Ctx) (uuid.UUID, string, bool, error) {
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return uuid.Nil, "", false, err
	}
	var masjidID uuid.UUID
	var slug string = mc.Slug

	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if mc.Slug != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, "", false, fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
		return uuid.Nil, "", false, helperAuth.ErrMasjidContextMissing
	}

	// Coba DKM/Admin dulu
	if id, er := helperAuth.EnsureMasjidAccessDKM(c, helperAuth.MasjidContext{ID: masjidID}); er == nil && id != uuid.Nil {
		return id, slug, true, nil
	}
	// Kalau bukan DKM, minimal harus member
	if !helperAuth.UserHasMasjid(c, masjidID) {
		return uuid.Nil, "", false, fiber.NewError(fiber.StatusForbidden, "Anda tidak terdaftar pada masjid ini (membership).")
	}
	return masjidID, slug, false, nil
}

func (h *AnnouncementController) Create(c *fiber.Ctx) error {
	// Sisipkan DB utk resolver slug→id
	c.Locals("DB", h.DB)

	// Resolve masjid & role
	masjidID, _, isAdminOrDKM, err := resolveMasjidIDUsingHelpers(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	isTeacher := func() bool {
		if id, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()

	if !isAdminOrDKM && !isTeacher {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak diizinkan")
	}

	var req annDTO.CreateAnnouncementRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	log.Printf("[ann.create] CT=%q path=%s masjid_id=%s admin_or_dkm=%v teacher=%v",
		ct, c.OriginalURL(), masjidID, isAdminOrDKM, isTeacher)

	// ---------- Parse body ----------
	if strings.HasPrefix(ct, "multipart/form-data") {
		req.AnnouncementTitle = strings.TrimSpace(c.FormValue("announcement_title"))
		req.AnnouncementDate = strings.TrimSpace(c.FormValue("announcement_date"))
		req.AnnouncementContent = strings.TrimSpace(c.FormValue("announcement_content"))

		// NEW: slug dari multipart (opsional)
		if v := strings.TrimSpace(c.FormValue("announcement_slug")); v != "" {
			s := helper.Slugify(v, 160)
			req.AnnouncementSlug = &s
		}

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

		// (1) JSON satu field (opsional)
		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			if err := json.Unmarshal([]byte(uj), &req.URLs); err != nil {
				log.Printf("[ann.create] urls_json unmarshal error: %v, raw=%s", err, uj)
				return helper.JsonError(c, fiber.StatusBadRequest, "urls_json tidak valid: "+err.Error())
			}
		}

		// (2) Bracket/array style
		if len(req.URLs) == 0 {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				ups := helperOSS.ParseURLUpsertsFromMultipart(form, &helperOSS.URLParseOptions{
					BracketPrefix: "urls",
					DefaultKind:   "attachment",
				})
				if len(ups) > 0 {
					for _, u := range ups {
						req.URLs = append(req.URLs, annDTO.AnnouncementURLUpsert{
							AnnouncementURLKind:      u.Kind,
							AnnouncementURLLabel:     u.Label,
							AnnouncementURLHref:      u.Href,
							AnnouncementURLObjectKey: u.ObjectKey,
							AnnouncementURLOrder:     u.Order,
							AnnouncementURLIsPrimary: u.IsPrimary,
						})
					}
				}
			}
		}

	} else {
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		// Normalisasi slug jika datang via JSON
		if req.AnnouncementSlug != nil {
			s := helper.Slugify(*req.AnnouncementSlug, 160)
			req.AnnouncementSlug = &s
		}
	}

	// Normalisasi URL metadata
	for i := range req.URLs {
		req.URLs[i].Normalize()
	}

	// Validasi
	if err := validateAnnouncement.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	// ---------- TX ----------
	tx := h.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ann.create] panic: %v (rollback)", r)
			_ = tx.Rollback().Error
		}
	}()

	// Build & set created_by
	m := req.ToModel(masjidID)
	if isTeacher && !isAdminOrDKM {
		mtID, err := helperAuth.GetMasjidTeacherIDForMasjid(c, masjidID)
		if err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Akun Anda tidak terdaftar sebagai guru di masjid ini")
		}
		m.AnnouncementCreatedByTeacherID = &mtID
	} else {
		m.AnnouncementCreatedByTeacherID = nil
	}

	// ---------- SLUG: normalize + ensure-unique per tenant (alive-only) ----------
	// Jika slug kosong → generate dari title
	baseSlug := ""
	if m.AnnouncementSlug != nil && strings.TrimSpace(*m.AnnouncementSlug) != "" {
		baseSlug = helper.Slugify(*m.AnnouncementSlug, 160)
	} else {
		baseSlug = helper.Slugify(m.AnnouncementTitle, 160)
		// fallback kalau title terlalu pendek setelah slugify
		if baseSlug == "item" || baseSlug == "" {
			baseSlug = helper.Slugify(
				fmt.Sprintf("ann-%s", strings.Split(m.AnnouncementID.String(), "-")[0]),
				160,
			)
		}
	}

	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		tx,
		"announcements",
		"announcement_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			// selaras dengan index uq_announcements_slug_per_tenant_alive
			return q.Where(`
				announcement_masjid_id = ?
				AND announcement_deleted_at IS NULL
			`, masjidID)
		},
		160,
	)
	if err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	// set slug ke model
	m.AnnouncementSlug = &uniqueSlug
	// ---------- END SLUG ----------

	// Insert announcement
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_announcements_slug_per_tenant_alive") ||
			strings.Contains(msg, "announcement_slug") && strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
		}
		log.Printf("[ann.create] insert announcement error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat pengumuman")
	}

	// ---------- Build URL items ----------
	var urlItems []annModel.AnnouncementURLModel

	// (a) dari JSON / bracket/array (sudah diparse di req.URLs)
	for _, it := range req.URLs {
		row := annModel.AnnouncementURLModel{
			AnnouncementURLMasjidId:       masjidID,
			AnnouncementURLAnnouncementId: m.AnnouncementID,
			AnnouncementURLKind:           strings.TrimSpace(it.AnnouncementURLKind),
			AnnouncementURLHref:           it.AnnouncementURLHref,
			AnnouncementURLObjectKey:      it.AnnouncementURLObjectKey,
			AnnouncementURLLabel:          it.AnnouncementURLLabel,
			AnnouncementURLOrder:          it.AnnouncementURLOrder,
			AnnouncementURLIsPrimary:      it.AnnouncementURLIsPrimary,
		}
		if row.AnnouncementURLKind == "" {
			row.AnnouncementURLKind = "attachment"
		}
		urlItems = append(urlItems, row)
	}

	// (b) dari files multipart
	if strings.HasPrefix(ct, "multipart/form-data") {
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			for k, fhs := range form.File {
				log.Printf("[ann.create] form.File key=%q count=%d", k, len(fhs))
			}
			fhs, usedKeys := helperOSS.CollectUploadFiles(form, nil)
			log.Printf("[ann.create] collected files=%d via keys=%v", len(fhs), usedKeys)

			if len(fhs) > 0 {
				oss, oerr := helperOSS.NewOSSServiceFromEnv("")
				if oerr != nil {
					tx.Rollback()
					log.Printf("[ann.create] OSS init error: %v", oerr)
					return helper.JsonError(c, fiber.StatusBadGateway, "OSS tidak siap")
				}
				ctx := context.Background()

				for idx, fh := range fhs {
					log.Printf("[ann.create] uploading file #%d name=%q size=%d", idx+1, fh.Filename, fh.Size)

					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "announcements", fh)
					if uerr != nil {
						tx.Rollback()
						log.Printf("[ann.create] upload error for %q: %v", fh.Filename, uerr)
						return helper.JsonError(c, fiber.StatusBadRequest, uerr.Error())
					}

					// cari slot metadata yang kosong
					var row *annModel.AnnouncementURLModel
					for i := range urlItems {
						if urlItems[i].AnnouncementURLHref == nil && urlItems[i].AnnouncementURLObjectKey == nil {
							row = &urlItems[i]
							break
						}
					}
					if row == nil {
						urlItems = append(urlItems, annModel.AnnouncementURLModel{
							AnnouncementURLMasjidId:       masjidID,
							AnnouncementURLAnnouncementId: m.AnnouncementID,
							AnnouncementURLKind:           "attachment",
							AnnouncementURLOrder:          len(urlItems) + 1,
						})
						row = &urlItems[len(urlItems)-1]
					}

					row.AnnouncementURLHref = &publicURL
					if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						row.AnnouncementURLObjectKey = &key
					}
					if strings.TrimSpace(row.AnnouncementURLKind) == "" {
						row.AnnouncementURLKind = "attachment"
					}

					log.Printf("[ann.create] uploaded -> url=%s kind=%s order=%d primary=%v",
						publicURL, row.AnnouncementURLKind, row.AnnouncementURLOrder, row.AnnouncementURLIsPrimary)
				}
			} else {
				log.Printf("[ann.create] no files found in multipart form")
			}
		}
	}

	// Konsistensi
	for _, it := range urlItems {
		if it.AnnouncementURLAnnouncementId != m.AnnouncementID {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "URL item tidak merujuk ke pengumuman yang sama")
		}
	}

	// Simpan URLs
	if len(urlItems) > 0 {
		if err := tx.Create(&urlItems).Error; err != nil {
			tx.Rollback()
			log.Printf("[ann.create] insert URLs error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan lampiran")
		}
		// jaga supaya hanya 1 primary per kind
		for _, it := range urlItems {
			if it.AnnouncementURLIsPrimary {
				if err := tx.Model(&annModel.AnnouncementURLModel{}).
					Where("announcement_url_masjid_id = ? AND announcement_url_announcement_id = ? AND announcement_url_kind = ? AND announcement_url_id <> ?",
						masjidID, m.AnnouncementID, it.AnnouncementURLKind, it.AnnouncementURLId).
					Update("announcement_url_is_primary", false).Error; err != nil {
					tx.Rollback()
					log.Printf("[ann.create] unset other primary error: %v", err)
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal set primary lampiran")
				}
			}
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		log.Printf("[ann.create] tx commit error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	// isi URLs ringkas utk response
	var rows []annModel.AnnouncementURLModel
	_ = h.DB.Where("announcement_url_announcement_id = ?", m.AnnouncementID).
		Order("announcement_url_order ASC, announcement_url_created_at ASC").
		Find(&rows)
	resp := annDTO.NewAnnouncementResponse(m)
	for _, r := range rows {
		if r.AnnouncementURLHref == nil {
			continue
		}
		resp.Urls = append(resp.Urls, &annDTO.AnnouncementURLLite{
			ID:             r.AnnouncementURLId,
			Label:          r.AnnouncementURLLabel,
			AnnouncementID: r.AnnouncementURLAnnouncementId,
			Href:           *r.AnnouncementURLHref,
		})
	}

	log.Printf("[ann.create] OK announcement_id=%s slug=%s urls=%d", m.AnnouncementID, ptrStr(m.AnnouncementSlug), len(resp.Urls))
	return helper.JsonCreated(c, "Pengumuman & lampiran berhasil dibuat", resp)
}

// util kecil
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// --- tenant guard fetch

// ===================== UPDATE =====================
// PUT /admin/announcements/:id
func (h *AnnouncementController) Update(c *fiber.Ctx) error {
	// Sisipkan DB utk resolver slug→id (dipakai helperAuth.GetMasjidIDBySlug)
	c.Locals("DB", h.DB)

	// Resolve masjid & role
	masjidID, _, isAdminOrDKM, err := resolveMasjidIDUsingHelpers(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	isTeacher := func() bool {
		if id, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	if !isAdminOrDKM && !isTeacher {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak diizinkan")
	}

	// Param ID
	rawID := strings.TrimSpace(c.Params("id"))
	annID, err := uuid.Parse(rawID)
	if err != nil || annID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "announcement_id tidak valid")
	}

	// Ambil record existing (beserta Theme minimal untuk response builder)
	var m annModel.AnnouncementModel
	if err := h.DB.
		Preload("Theme").
		Where("announcement_id = ? AND announcement_masjid_id = ?", annID, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Pengumuman tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ========== Parse body ==========
	var req annDTO.UpdateAnnouncementRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	log.Printf("[ann.update] CT=%q path=%s ann_id=%s masjid_id=%s admin_or_dkm=%v teacher=%v",
		ct, c.OriginalURL(), annID, masjidID, isAdminOrDKM, isTeacher)

	if strings.HasPrefix(ct, "multipart/form-data") {
		// Text fields
		if v := strings.TrimSpace(c.FormValue("announcement_title")); v != "" {
			req.AnnouncementTitle = &v
		}
		if v := strings.TrimSpace(c.FormValue("announcement_date")); v != "" {
			req.AnnouncementDate = &v
		}
		if v := strings.TrimSpace(c.FormValue("announcement_content")); v != "" {
			req.AnnouncementContent = &v
		}
		if v := strings.TrimSpace(c.FormValue("announcement_is_active")); v != "" {
			b := strings.EqualFold(v, "true") || v == "1"
			req.AnnouncementIsActive = &b
		}
		if v := strings.TrimSpace(c.FormValue("announcement_theme_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.AnnouncementThemeID = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("announcement_class_section_id")); v != "" {
			// kosongkan jadi NULL kalau string = "null"
			if strings.EqualFold(v, "null") {
				req.AnnouncementClassSectionID = nil
			} else if id, err := uuid.Parse(v); err == nil {
				req.AnnouncementClassSectionID = &id
			}
		}
		// NEW: slug dari multipart (opsional)
		if v := strings.TrimSpace(c.FormValue("announcement_slug")); v != "" {
			s := helper.Slugify(v, 160)
			req.AnnouncementSlug = &s
		}

		// (1) urls_json (append metadata URL baru)
		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			if err := json.Unmarshal([]byte(uj), &req.URLs); err != nil {
				log.Printf("[ann.update] urls_json unmarshal error: %v, raw=%s", err, uj)
				return helper.JsonError(c, fiber.StatusBadRequest, "urls_json tidak valid: "+err.Error())
			}
		}

		// (2) Bracket/array style → append juga (sama seperti Create), via helper
		if len(req.URLs) == 0 {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				ups := helperOSS.ParseURLUpsertsFromMultipart(form, &helperOSS.URLParseOptions{
					BracketPrefix: "urls",
					DefaultKind:   "attachment",
				})
				if len(ups) > 0 {
					for _, u := range ups {
						req.URLs = append(req.URLs, annDTO.AnnouncementURLUpsert{
							AnnouncementURLKind:      u.Kind,
							AnnouncementURLLabel:     u.Label,
							AnnouncementURLHref:      u.Href,
							AnnouncementURLObjectKey: u.ObjectKey,
							AnnouncementURLOrder:     u.Order,
							AnnouncementURLIsPrimary: u.IsPrimary,
						})
					}
				}
			}
		}

		// (3) Delete IDs: url_delete_ids = "uuid1,uuid2"
		if del := strings.TrimSpace(c.FormValue("url_delete_ids")); del != "" {
			ids, err := parseUUIDsCSV(del)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "url_delete_ids tidak valid")
			}
			req.DeleteURLIDs = ids
		}

		// (4) Set primary per-kind: url_primary_kind[] + url_primary_id[]
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			kinds := form.Value["url_primary_kind[]"]
			ids := form.Value["url_primary_id[]"]
			for i := 0; i < len(kinds) && i < len(ids); i++ {
				kind := strings.TrimSpace(kinds[i])
				if kind == "" {
					continue
				}
				if id, err := uuid.Parse(strings.TrimSpace(ids[i])); err == nil && id != uuid.Nil {
					if req.PrimaryPerKind == nil {
						req.PrimaryPerKind = map[string]uuid.UUID{}
					}
					req.PrimaryPerKind[kind] = id
				}
			}
		}
	} else {
		// JSON
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		// Normalisasi slug jika datang via JSON
		if req.AnnouncementSlug != nil {
			s := helper.Slugify(*req.AnnouncementSlug, 160)
			req.AnnouncementSlug = &s
		}
	}

	// Normalisasi URL metadata (append)
	for i := range req.URLs {
		req.URLs[i].Normalize()
	}

	// Validasi ringan (abaikan required pada field nil)
	if err := validateAnnouncement.StructPartial(req); err != nil {
		// optional: log/ignore sesuai kebutuhan
	}

	// ========== TX ==========
	tx := h.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ann.update] panic: %v (rollback)", r)
			_ = tx.Rollback().Error
		}
	}()

	// Lock row (fresh)
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("announcement_id = ? AND announcement_masjid_id = ?", annID, masjidID).
		First(&m).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Pengumuman tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Simpan nilai lama untuk deteksi perubahan
	oldTitle := m.AnnouncementTitle
	oldSlug := ptrStr(m.AnnouncementSlug)

	// Terapkan perubahan field dasar (kecuali slug unik — akan di-handle khusus di bawah)
	req.ApplyToModel(&m)

	// ===== SLUG handling (normalize & ensure-unique per tenant, exclude diri sendiri) =====
	// Jika user mengirim slug → pakai itu sebagai base.
	// Jika tidak, dan slug di DB kosong → generate dari title baru.
	var baseSlug string
	wantSlugChange := false

	if req.AnnouncementSlug != nil {
		if s := strings.TrimSpace(*req.AnnouncementSlug); s != "" {
			baseSlug = helper.Slugify(s, 160)
			wantSlugChange = true
		} else {
			// "" dianggap ingin mengosongkan → kita generate dari title
			baseSlug = helper.Slugify(m.AnnouncementTitle, 160)
			wantSlugChange = true
		}
	} else if m.AnnouncementSlug == nil || strings.TrimSpace(ptrStr(m.AnnouncementSlug)) == "" {
		// slug kosong di DB → generate dari title
		baseSlug = helper.Slugify(m.AnnouncementTitle, 160)
		wantSlugChange = true
	}

	// Optional: kalau mau regenerate saat title berubah meski slug lama ada, aktifkan blok ini:
	// if !wantSlugChange && oldTitle != m.AnnouncementTitle {
	// 	baseSlug = helper.Slugify(m.AnnouncementTitle, 160)
	// 	wantSlugChange = true
	// }

	if wantSlugChange {
		if baseSlug == "" || baseSlug == "item" {
			baseSlug = helper.Slugify(fmt.Sprintf("ann-%s", strings.Split(m.AnnouncementID.String(), "-")[0]), 160)
		}
		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			"announcements",
			"announcement_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				// Selaras index uq_announcements_slug_per_tenant_alive + exclude diri sendiri
				return q.Where(`
					announcement_masjid_id = ?
					AND announcement_deleted_at IS NULL
					AND announcement_id <> ?
				`, masjidID, m.AnnouncementID)
			},
			160,
		)
		if err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		m.AnnouncementSlug = &uniqueSlug
	}
	// ===== END SLUG =====

	// Simpan perubahan dasar + slug
	if err := tx.Save(&m).Error; err != nil {
		tx.Rollback()
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_announcements_slug_per_tenant_alive") ||
			(strings.Contains(msg, "announcement_slug") && strings.Contains(msg, "unique")) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// ====== Operasi URL: append dari metadata/file, delete by IDs, set primary per-kind ======

	// (a) append dari metadata
	var toInsert []annModel.AnnouncementURLModel
	for _, it := range req.URLs {
		row := annModel.AnnouncementURLModel{
			AnnouncementURLMasjidId:       masjidID,
			AnnouncementURLAnnouncementId: m.AnnouncementID,
			AnnouncementURLKind:           strings.TrimSpace(it.AnnouncementURLKind),
			AnnouncementURLHref:           it.AnnouncementURLHref,
			AnnouncementURLObjectKey:      it.AnnouncementURLObjectKey,
			AnnouncementURLLabel:          it.AnnouncementURLLabel,
			AnnouncementURLOrder:          it.AnnouncementURLOrder,
			AnnouncementURLIsPrimary:      it.AnnouncementURLIsPrimary,
		}
		if row.AnnouncementURLKind == "" {
			row.AnnouncementURLKind = "attachment"
		}
		toInsert = append(toInsert, row)
	}

	// (b) append dari files multipart (pakai helper)
	if strings.HasPrefix(ct, "multipart/form-data") {
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			for k, fhs := range form.File {
				log.Printf("[ann.update] form.File key=%q count=%d", k, len(fhs))
			}
			fhs, usedKeys := helperOSS.CollectUploadFiles(form, nil)
			log.Printf("[ann.update] collected files=%d via keys=%v", len(fhs), usedKeys)

			if len(fhs) > 0 {
				oss, oerr := helperOSS.NewOSSServiceFromEnv("")
				if oerr != nil {
					tx.Rollback()
					log.Printf("[ann.update] OSS init error: %v", oerr)
					return helper.JsonError(c, fiber.StatusBadGateway, "OSS tidak siap")
				}
				ctx := context.Background()

				for idx, fh := range fhs {
					log.Printf("[ann.update] uploading file #%d name=%q size=%d", idx+1, fh.Filename, fh.Size)

					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "announcements", fh)
					if uerr != nil {
						tx.Rollback()
						log.Printf("[ann.update] upload error for %q: %v", fh.Filename, uerr)
						return helper.JsonError(c, fiber.StatusBadRequest, uerr.Error())
					}

					row := annModel.AnnouncementURLModel{
						AnnouncementURLMasjidId:       masjidID,
						AnnouncementURLAnnouncementId: m.AnnouncementID,
						AnnouncementURLKind:           "attachment",
						AnnouncementURLOrder:          0,
					}
					row.AnnouncementURLHref = &publicURL
					if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						row.AnnouncementURLObjectKey = &key
					}
					toInsert = append(toInsert, row)
				}
			}
		}
	}

	// Insert batch (kalau ada)
	if len(toInsert) > 0 {
		if err := tx.Create(&toInsert).Error; err != nil {
			tx.Rollback()
			log.Printf("[ann.update] insert new URLs error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambah lampiran")
		}
	}

	// Delete by IDs (scoped ke masjid & ann)
	if len(req.DeleteURLIDs) > 0 {
		if err := tx.Where("announcement_url_masjid_id = ? AND announcement_url_announcement_id = ? AND announcement_url_id IN ?",
			masjidID, m.AnnouncementID, req.DeleteURLIDs).
			Delete(&annModel.AnnouncementURLModel{}).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus lampiran")
		}
	}

	// Set primary per-kind (jika ada)
	if len(req.PrimaryPerKind) > 0 {
		for kind, id := range req.PrimaryPerKind {
			// Pastikan id milik ann & kind tsb
			var cnt int64
			if err := tx.Model(&annModel.AnnouncementURLModel{}).
				Where("announcement_url_masjid_id = ? AND announcement_url_announcement_id = ? AND announcement_url_id = ? AND announcement_url_kind = ?",
					masjidID, m.AnnouncementID, id, kind).
				Count(&cnt).Error; err != nil || cnt == 0 {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusBadRequest, "Primary lampiran tidak valid untuk kind="+kind)
			}
			// Unset others
			if err := tx.Model(&annModel.AnnouncementURLModel{}).
				Where("announcement_url_masjid_id = ? AND announcement_url_announcement_id = ? AND announcement_url_kind = ?",
					masjidID, m.AnnouncementID, kind).
				Update("announcement_url_is_primary", false).Error; err != nil {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal reset primary "+kind)
			}
			// Set target
			if err := tx.Model(&annModel.AnnouncementURLModel{}).
				Where("announcement_url_id = ?", id).
				Update("announcement_url_is_primary", true).Error; err != nil {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal set primary "+kind)
			}
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		log.Printf("[ann.update] tx commit error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	// Response (lengkap dengan Urls ringkas)
	var rows []annModel.AnnouncementURLModel
	_ = h.DB.Where("announcement_url_announcement_id = ?", m.AnnouncementID).
		Order("announcement_url_order ASC, announcement_url_created_at ASC").
		Find(&rows)

	resp := annDTO.NewAnnouncementResponse(&m)
	for _, r := range rows {
		if r.AnnouncementURLHref == nil {
			continue
		}
		resp.Urls = append(resp.Urls, &annDTO.AnnouncementURLLite{
			ID:             r.AnnouncementURLId,
			Label:          r.AnnouncementURLLabel,
			AnnouncementID: r.AnnouncementURLAnnouncementId,
			Href:           *r.AnnouncementURLHref,
		})
	}

	log.Printf("[ann.update] OK announcement_id=%s old_slug=%q new_slug=%q urls=%d title_changed=%v",
		m.AnnouncementID, oldSlug, ptrStr(m.AnnouncementSlug), len(resp.Urls), oldTitle != m.AnnouncementTitle)
	return helper.JsonOK(c, "Pengumuman berhasil diperbarui", resp)
}



// ===================== DELETE (single attachment) =====================
// DELETE /.../announcements/:id/urls/:url_id
func (h *AnnouncementController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", h.DB)

	masjidID, _, isAdminOrDKM, err := resolveMasjidIDUsingHelpers(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	isTeacher := func() bool {
		if id, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	if !isAdminOrDKM && !isTeacher {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak diizinkan")
	}

	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil || annID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "announcement_id tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("url_id")))
	if err != nil || urlID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "url_id tidak valid")
	}

	// Pastikan ann milik masjid ini
	var m annModel.AnnouncementModel
	if err := h.DB.
		Where("announcement_id = ? AND announcement_masjid_id = ?", annID, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Pengumuman tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil pengumuman")
	}

	// Ambil URL row
	var u annModel.AnnouncementURLModel
	if err := h.DB.
		Where("announcement_url_id = ? AND announcement_url_masjid_id = ? AND announcement_url_announcement_id = ?",
			urlID, masjidID, annID).
		First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Lampiran tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil lampiran")
	}

	// TX (ringan)
	tx := h.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
		}
	}()

	// Pindahkan file ke spam (best-effort)
	if u.AnnouncementURLHref != nil && strings.TrimSpace(*u.AnnouncementURLHref) != "" {
		if dst, err := helperOSS.MoveToSpamByPublicURLENV(*u.AnnouncementURLHref, 0); err != nil {
			log.Printf("[ann.url.delete] move-to-spam gagal: %v (href=%s)", err, *u.AnnouncementURLHref)
		} else {
			log.Printf("[ann.url.delete] moved to spam: %s", dst)
		}
	}

	// Hapus row (hard)
	if err := tx.Where(
		"announcement_url_id = ? AND announcement_url_masjid_id = ? AND announcement_url_announcement_id = ?",
		urlID, masjidID, annID,
	).Delete(&annModel.AnnouncementURLModel{}).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus lampiran")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}
	return helper.JsonOK(c, "Lampiran berhasil dihapus", fiber.Map{
		"announcement_id": annID,
		"url_id":          urlID,
	})
}
