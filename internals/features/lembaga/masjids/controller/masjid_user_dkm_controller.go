// file: internals/features/masjids/masjids/controller/masjid_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	dto "masjidku_backend/internals/features/lembaga/masjids/dto"
	model "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =======================================================
   Defaults / tunables
======================================================= */

/* =======================================================
   Helpers lokal (logging & parsing)
======================================================= */

func toStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case *string:
		if t == nil {
			return "<nil>"
		}
		return *t
	default:
		return fmt.Sprintf("%v", v)
	}
}

func safeStrPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "ya", "yes", "on":
		return true
	default:
		return false
	}
}

func clampInt(vs string, def, min, max int) int {
	if vs == "" {
		return def
	}
	v, err := strconv.Atoi(vs)
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func ptrStr(s string) *string {
	ss := strings.TrimSpace(s)
	if ss == "" {
		return nil
	}
	return &ss
}
func ptrStrTrim(s string) *string { return ptrStr(s) }

func normalizeDomain(raw string) string {
	d := strings.ToLower(strings.TrimSpace(raw))
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimPrefix(d, "https://")
	return strings.TrimSuffix(d, "/")
}
func normalizeDomainPtr(raw string) *string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	n := normalizeDomain(raw)
	if n == "" {
		return nil
	}
	return &n
}

// lazy init OSS dari ENV jika belum ada
func (mc *MasjidController) ensureOSS() error {
	if mc.OSS != nil && mc.OSS.Bucket != nil && mc.OSS.BucketName != "" {
		return nil
	}
	svc, err := helperOSS.NewOSSServiceFromEnv("") // prefix opsional
	if err != nil {
		// biar konsisten sama log/response kamu
		return fiber.NewError(fiber.StatusFailedDependency, "OSS belum dikonfigurasi")
	}
	mc.OSS = svc
	return nil
}

// splitCSV memecah "a,b, c , , d" -> []string{"a","b","c","d"}
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if t := strings.TrimSpace(r); t != "" {
			out = append(out, t)
		}
	}
	return out
}



/* =======================================================
   DTO request (profile payload)
======================================================= */

type MasjidProfilePayload struct {
	Description string     `json:"description"`
	FoundedYear *int       `json:"founded_year"`

	Address      string `json:"address"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email"`

	GoogleMapsURL string `json:"google_maps_url"`
	InstagramURL  string `json:"instagram_url"`
	WhatsappURL   string `json:"whatsapp_url"`
	YoutubeURL    string `json:"youtube_url"`
	FacebookURL   string `json:"facebook_url"`
	TiktokURL     string `json:"tiktok_url"`
	WhatsappGroupIkhwanURL string `json:"whatsapp_group_ikhwan_url"`
	WhatsappGroupAkhwatURL string `json:"whatsapp_group_akhwat_url"`
	WebsiteURL    string `json:"website_url"`

	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	SchoolNPSN            string     `json:"school_npsn"`
	SchoolNSS             string     `json:"school_nss"`
	SchoolAccreditation   string     `json:"school_accreditation"`
	SchoolPrincipalUserID *uuid.UUID `json:"school_principal_user_id"`
	SchoolPhone           string     `json:"school_phone"`
	SchoolEmail           string     `json:"school_email"`
	SchoolAddress         string     `json:"school_address"`
	SchoolStudentCapacity *int       `json:"school_student_capacity"`
	SchoolIsBoarding      *bool      `json:"school_is_boarding"`
}



/* =======================================================
   Parsers (levels & profile)
======================================================= */
func normalizeStringSlice(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		// optional: lower-case
		t = strings.ToLower(t)
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

/* =======================================================
   CreateMasjidDKM — multipart only, with logs & lazy OSS
======================================================= */

func (mc *MasjidController) CreateMasjidDKM(c *fiber.Ctx) error {
	t0 := time.Now()
	rid := uuid.New().String()
	lg := func(msg string, kv ...any) {
		b := strings.Builder{}
		b.WriteString("[CreateMasjidDKM][rid=")
		b.WriteString(rid)
		b.WriteString("] ")
		b.WriteString(msg)
		for i := 0; i+1 < len(kv); i += 2 {
			b.WriteString(" | ")
			b.WriteString(strings.TrimSpace(toStr(kv[i])))
			b.WriteString("=")
			b.WriteString(toStr(kv[i+1]))
		}
		log.Println(b.String())
	}
	defer func() { lg("DONE", "dur", time.Since(t0).String()) }()

	// Auth
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		lg("auth failed", "err", err.Error())
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	ct := strings.ToLower(c.Get("Content-Type"))
	isMultipart := strings.Contains(ct, "multipart/form-data")
	lg("request begin", "ct", ct, "isMultipart", isMultipart)
	if !isMultipart {
		lg("unsupported media type")
		return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "Gunakan multipart/form-data")
	}

	// Basic fields
	name := strings.TrimSpace(c.FormValue("masjid_name"))
	if name == "" {
		lg("validation failed", "reason", "masjid_name kosong")
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_name wajib diisi")
	}
	domain := normalizeDomainPtr(c.FormValue("masjid_domain"))
	bioShort := ptrStrTrim(c.FormValue("masjid_bio_short"))
	location := ptrStrTrim(c.FormValue("masjid_location"))
	city := ptrStrTrim(c.FormValue("masjid_city"))
	isSchool := parseBool(c.FormValue("masjid_is_islamic_school"))

	var yayasanID, planID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("masjid_yayasan_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			yayasanID = &id
		}
	}
	if s := strings.TrimSpace(c.FormValue("masjid_current_plan_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			planID = &id
		}
	}

	verifStatus := strings.TrimSpace(c.FormValue("masjid_verification_status"))
	if verifStatus == "" {
		verifStatus = "pending"
	}
	verifNotes := ptrStrTrim(c.FormValue("masjid_verification_notes"))

	levels := parseLevelsFromMultipart(c)
	profile := parseProfileFromForm(c)

	// File (opsional)
	fileHeader, _ := c.FormFile("file")
	slot := strings.ToLower(strings.TrimSpace(c.Query("slot")))
	if slot == "" {
		slot = "logo"
	}
	if slot != "logo" && slot != "background" && slot != "misc" {
		lg("validation failed", "reason", "slot invalid", "slot", slot)
		return helper.JsonError(c, fiber.StatusBadRequest, "slot harus salah satu dari: logo, background, misc")
	}

	var fileSize int64
	var fileName, fileCT string
	if fileHeader != nil {
		fileSize = fileHeader.Size
		fileName = fileHeader.Filename
		if fileHeader.Header != nil {
			fileCT = fileHeader.Header.Get("Content-Type")
		}
	}
	lg("parsed form",
		"name", name,
		"domain", safeStrPtr(domain),
		"city", safeStrPtr(city),
		"isSchool", isSchool,
		"levels_len", len(levels),
		"hasProfile", profile != nil,
		"slot", slot,
		"hasFile", fileHeader != nil,
		"fileName", fileName,
		"fileSize", fileSize,
		"fileCT", fileCT,
	)

	// TX
	var resp dto.MasjidResponse
	txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
		base := helper.SuggestSlugFromName(name)
		lg("slug suggest", "base", base)

		slug, err := helper.EnsureUniqueSlugCI(c.Context(), tx, "masjids", "masjid_slug", base, nil, 100)
		if err != nil {
			lg("slug ensure unique failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}
		lg("slug ok", "slug", slug)

		now := time.Now()
		m := model.MasjidModel{
			MasjidID:            uuid.New(),
			MasjidYayasanID:     yayasanID,
			MasjidCurrentPlanID: planID,

			MasjidName:     name,
			MasjidBioShort: bioShort,
			MasjidLocation: location,
			MasjidCity:     city,

			MasjidDomain: domain,
			MasjidSlug:   slug,

			MasjidIsActive:           true,
			MasjidVerificationStatus: model.VerificationStatus(strings.ToLower(verifStatus)),
			MasjidVerificationNotes:  verifNotes,
			MasjidIsIslamicSchool:    isSchool,

			MasjidCreatedAt: now,
			MasjidUpdatedAt: now,
		}
		_ = m.SetLevels(levels)
		// NOTE: fungsi ini diasumsikan sudah ada di package yang sama
		syncVerificationFlags(&m)

		if err := tx.Create(&m).Error; err != nil {
			lg("db create masjid failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan masjid")
		}
		lg("db create masjid ok", "masjid_id", m.MasjidID.String())

		// Profile opsional
		if profile != nil {
			p := model.MasjidProfileModel{
				MasjidProfileMasjidID:               m.MasjidID,
				MasjidProfileDescription:            ptrStr(profile.Description),
				MasjidProfileFoundedYear:            profile.FoundedYear,
				MasjidProfileAddress:                ptrStr(profile.Address),
				MasjidProfileContactPhone:           ptrStr(profile.ContactPhone),
				MasjidProfileContactEmail:           ptrStr(profile.ContactEmail),
				MasjidProfileGoogleMapsURL:          ptrStr(profile.GoogleMapsURL),
				MasjidProfileInstagramURL:           ptrStr(profile.InstagramURL),
				MasjidProfileWhatsappURL:            ptrStr(profile.WhatsappURL),
				MasjidProfileYoutubeURL:             ptrStr(profile.YoutubeURL),
				MasjidProfileFacebookURL:            ptrStr(profile.FacebookURL),
				MasjidProfileTiktokURL:              ptrStr(profile.TiktokURL),
				MasjidProfileWhatsappGroupIkhwanURL: ptrStr(profile.WhatsappGroupIkhwanURL),
				MasjidProfileWhatsappGroupAkhwatURL: ptrStr(profile.WhatsappGroupAkhwatURL),
				MasjidProfileWebsiteURL:             ptrStr(profile.WebsiteURL),
				MasjidProfileLatitude:               profile.Latitude,
				MasjidProfileLongitude:              profile.Longitude,
				MasjidProfileSchoolNPSN:             ptrStr(profile.SchoolNPSN),
				MasjidProfileSchoolNSS:              ptrStr(profile.SchoolNSS),
				MasjidProfileSchoolAccreditation:    ptrStr(profile.SchoolAccreditation),
				MasjidProfileSchoolPrincipalUserID:  profile.SchoolPrincipalUserID,
				MasjidProfileSchoolPhone:            ptrStr(profile.SchoolPhone),
				MasjidProfileSchoolEmail:            ptrStr(profile.SchoolEmail),
				MasjidProfileSchoolAddress:          ptrStr(profile.SchoolAddress),
				MasjidProfileSchoolStudentCapacity:  profile.SchoolStudentCapacity,
			}
			if profile.SchoolIsBoarding != nil {
				p.MasjidProfileSchoolIsBoarding = *profile.SchoolIsBoarding
			}
			if err := tx.Create(&p).Error; err != nil {
				lg("db create profile failed", "err", err.Error())
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan profil masjid")
			}
			lg("db create profile ok")
		}

		// Upload file opsional (LAZY INIT OSS di sini)
		if fileHeader != nil {
			lg("upload begin", "slot", slot, "size", fileHeader.Size, "name", fileHeader.Filename)

			if err := mc.ensureOSS(); err != nil {
				lg("oss not configured", "err", err.Error())
				return fiber.NewError(fiber.StatusFailedDependency, err.Error())
			}
			lg("oss ready", "bucket", mc.OSS.BucketName)

			publicURL, upErr := helperOSS.UploadAnyToOSS(c.Context(), mc.OSS, m.MasjidID, slot, fileHeader)
			if upErr != nil {
				var fe *fiber.Error
				if errors.As(upErr, &fe) {
					lg("upload failed (fiber error)", "code", fe.Code, "msg", fe.Message)
					return fiber.NewError(fe.Code, fe.Message)
				}
				lg("upload failed", "err", upErr.Error())
				return fiber.NewError(fiber.StatusBadGateway, "Gagal upload ke OSS")
			}
			lg("upload ok", "url", publicURL)

			retUntil := now.Add(defaultRetention)
			switch slot {
			case "logo":
				if m.MasjidLogoURL != nil && strings.TrimSpace(*m.MasjidLogoURL) != "" {
					m.MasjidLogoURLOld = m.MasjidLogoURL
					m.MasjidLogoObjectKeyOld = m.MasjidLogoObjectKey
					m.MasjidLogoDeletePendingUntil = &retUntil
				}
				m.MasjidLogoURL = &publicURL
				if key, err := helperOSS.ExtractKeyFromPublicURL(publicURL); err == nil {
					m.MasjidLogoObjectKey = &key
					lg("metadata set (logo)", "key", key)
				} else {
					lg("extract key failed (logo)", "err", err.Error())
				}
			case "background":
				if m.MasjidBackgroundURL != nil && strings.TrimSpace(*m.MasjidBackgroundURL) != "" {
					m.MasjidBackgroundURLOld = m.MasjidBackgroundURL
					m.MasjidBackgroundObjectKeyOld = m.MasjidBackgroundObjectKey
					m.MasjidBackgroundDeletePendingUntil = &retUntil
				}
				m.MasjidBackgroundURL = &publicURL
				if key, err := helperOSS.ExtractKeyFromPublicURL(publicURL); err == nil {
					m.MasjidBackgroundObjectKey = &key
					lg("metadata set (background)", "key", key)
				} else {
					lg("extract key failed (background)", "err", err.Error())
				}
			case "misc":
				lg("metadata untouched for misc")
			}

			m.MasjidUpdatedAt = time.Now()
			if err := tx.Save(&m).Error; err != nil {
				lg("db save metadata failed", "err", err.Error())
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan metadata file")
			}
			lg("db save metadata ok")
		}

		// Roles
		_ = helperAuth.EnsureGlobalRole(tx, userID, "user", &userID)
		if err := helperAuth.GrantScopedRoleDKM(tx, userID, m.MasjidID); err != nil {
			lg("grant role failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal grant peran DKM")
		}
		lg("grant role ok", "user_id", userID.String(), "masjid_id", m.MasjidID.String())

		resp = dto.FromModelMasjid(&m)
		lg("build response ok", "masjid_id", resp.MasjidID, "slug", resp.MasjidSlug)
		return nil
	})
	if txErr != nil {
		if fe, ok := txErr.(*fiber.Error); ok {
			lg("tx failed (fiber error)", "code", fe.Code, "msg", fe.Message)
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		lg("tx failed", "err", txErr.Error())
		return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
	}

	lg("request success")
	return helper.JsonCreated(c, "Masjid berhasil dibuat", resp)
}


// ========== Parser profile (multipart) ==========
// ========== Parser profile (multipart) ==========
func parseProfileFromForm(c *fiber.Ctx) *MasjidProfilePayload {
	log.Println("[parseProfileFromForm] begin")
	hasAny := false
	p := &MasjidProfilePayload{}

	// text fields
	if v := strings.TrimSpace(c.FormValue("profile_description")); v != "" {
		p.Description = v; hasAny = true
	}
	if v := strings.TrimSpace(c.FormValue("profile_founded_year")); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			p.FoundedYear = &i; hasAny = true
		} else {
			log.Println("[parseProfileFromForm] invalid founded_year:", v, "err:", err)
		}
	}

	if v := strings.TrimSpace(c.FormValue("profile_address")); v != "" { p.Address = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_contact_phone")); v != "" { p.ContactPhone = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_contact_email")); v != "" { p.ContactEmail = v; hasAny = true }

	if v := strings.TrimSpace(c.FormValue("profile_google_maps_url")); v != "" { p.GoogleMapsURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_instagram_url")); v != "" { p.InstagramURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_whatsapp_url")); v != "" { p.WhatsappURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_youtube_url")); v != "" { p.YoutubeURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_facebook_url")); v != "" { p.FacebookURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_tiktok_url")); v != "" { p.TiktokURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_whatsapp_group_ikhwan_url")); v != "" { p.WhatsappGroupIkhwanURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_whatsapp_group_akhwat_url")); v != "" { p.WhatsappGroupAkhwatURL = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_website_url")); v != "" { p.WebsiteURL = v; hasAny = true }

	// numeric (lat/long)
	if v := strings.TrimSpace(c.FormValue("profile_latitude")); v != "" {
		if f, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64); err == nil {
			p.Latitude = &f; hasAny = true
		} else {
			log.Println("[parseProfileFromForm] invalid latitude:", v, "err:", err)
		}
	}
	if v := strings.TrimSpace(c.FormValue("profile_longitude")); v != "" {
		if f, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64); err == nil {
			p.Longitude = &f; hasAny = true
		} else {
			log.Println("[parseProfileFromForm] invalid longitude:", v, "err:", err)
		}
	}

	// sekolah
	if v := strings.TrimSpace(c.FormValue("profile_school_npsn")); v != "" { p.SchoolNPSN = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_school_nss")); v != "" { p.SchoolNSS = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_school_accreditation")); v != "" { p.SchoolAccreditation = v; hasAny = true }

	if v := strings.TrimSpace(c.FormValue("profile_school_principal_user_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			p.SchoolPrincipalUserID = &id; hasAny = true
		} else {
			log.Println("[parseProfileFromForm] invalid principal_user_id:", v, "err:", err)
		}
	}

	// ⚠️ CATATAN: jika kolom DB `masjid_profile_school_phone` TIDAK ada, hapus dua baris berikut
	if v := strings.TrimSpace(c.FormValue("profile_school_phone")); v != "" { p.SchoolPhone = v; hasAny = true }

	if v := strings.TrimSpace(c.FormValue("profile_school_email")); v != "" { p.SchoolEmail = v; hasAny = true }
	if v := strings.TrimSpace(c.FormValue("profile_school_address")); v != "" { p.SchoolAddress = v; hasAny = true }

	if v := strings.TrimSpace(c.FormValue("profile_school_student_capacity")); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			p.SchoolStudentCapacity = &i; hasAny = true
		} else {
			log.Println("[parseProfileFromForm] invalid student_capacity:", v, "err:", err)
		}
	}
	if v := strings.TrimSpace(c.FormValue("profile_school_is_boarding")); v != "" {
		b := parseBool(v)
		p.SchoolIsBoarding = &b; hasAny = true
	}

	if !hasAny {
		log.Println("[parseProfileFromForm] no profile fields provided")
		return nil
	}
	log.Println("[parseProfileFromForm] parsed OK")
	return p
}


// =======================================================
// GET /api/masjids  (list + filter)
// =======================================================

func (mc *MasjidController) GetMasjids(c *fiber.Ctx) error {
	tx := mc.DB.Model(&model.MasjidModel{})

	// q: ILIKE (match trigram index)
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		qq := "%" + q + "%"
		tx = tx.Where("(masjid_name ILIKE ? OR masjid_location ILIKE ? OR masjid_bio_short ILIKE ?)", qq, qq, qq)
	}

	// flags
	if v := strings.TrimSpace(c.Query("verified")); v != "" {
		tx = tx.Where("masjid_is_verified = ?", v == "true" || v == "1")
	}
	if v := strings.TrimSpace(c.Query("active")); v != "" {
		tx = tx.Where("masjid_is_active = ?", v == "true" || v == "1")
	}
	if v := strings.TrimSpace(c.Query("is_islamic_school")); v != "" {
		tx = tx.Where("masjid_is_islamic_school = ?", v == "true" || v == "1")
	}

	// relations
	if s := strings.TrimSpace(c.Query("yayasan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			tx = tx.Where("masjid_yayasan_id = ?", id)
		}
	}
	if s := strings.TrimSpace(c.Query("plan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			tx = tx.Where("masjid_current_plan_id = ?", id)
		}
	}

	// levels_any => OR of "masjid_levels ? ?"
	if s := strings.TrimSpace(c.Query("levels_any")); s != "" {
		parts := splitCSV(s)
		if len(parts) > 0 {
			orSQL := make([]string, 0, len(parts))
			args := make([]interface{}, 0, len(parts))
			for _, p := range parts {
				orSQL = append(orSQL, "masjid_levels ? ?")
				args = append(args, p)
			}
			tx = tx.Where("("+strings.Join(orSQL, " OR ")+")", args...)
		}
	}
	// levels_all => masjid_levels @> '["a","b"]'::jsonb
	if s := strings.TrimSpace(c.Query("levels_all")); s != "" {
		parts := splitCSV(s)
		if len(parts) > 0 {
			jb, _ := json.Marshal(parts)
			tx = tx.Where("masjid_levels @> ?::jsonb", string(jb))
		}
	}

	limit := clampInt(c.Query("limit"), 20, 1, 100)
	offset := clampInt(c.Query("offset"), 0, 0, 100000)
	tx = tx.Limit(limit).Offset(offset).Order("masjid_created_at DESC")

	var rows []model.MasjidModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid")
	}

	out := make([]dto.MasjidResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModelMasjid(&rows[i]))
	}
	return helper.JsonOK(c, "OK", fiber.Map{
		"items":  out,
		"count":  len(out),
		"limit":  limit,
		"offset": offset,
	})
}

// =======================================================
// GET /api/masjids/:id_or_slug (detail 1 masjid)
// =======================================================

func (mc *MasjidController) GetMasjid(c *fiber.Ctx) error {
	key := strings.TrimSpace(c.Params("id_or_slug"))
	if key == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter kosong")
	}

	var row model.MasjidModel
	var err error
	if id, parseErr := uuid.Parse(key); parseErr == nil {
		err = mc.DB.First(&row, "masjid_id = ?", id).Error
	} else {
		err = mc.DB.First(&row, "masjid_slug = ?", key).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid")
	}
	return helper.JsonOK(c, "OK", dto.FromModelMasjid(&row))
}

// =======================================================
// GET /api/masjids/:id/profile  (profil + primary files)
// =======================================================

func (mc *MasjidController) GetMasjidProfile(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 1) Ambil profil
	var mp model.MasjidProfileModel
	if err := mc.DB.First(&mp, "masjid_profile_masjid_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil masjid belum tersedia")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil profil")
	}

	// 2) Ambil primary files via VIEW (join sekali jalan)
	type primaryURLRow struct {
		Type               string     `gorm:"column:type" json:"type"`
		FileURL            string     `gorm:"column:file_url" json:"file_url"`
		TrashURL           *string    `gorm:"column:trash_url" json:"trash_url,omitempty"`
		DeletePendingUntil *time.Time `gorm:"column:delete_pending_until" json:"delete_pending_until,omitempty"`
		CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`
	}
	var prim []primaryURLRow
	if err := mc.DB.
		Table("masjids_profiles mp").
		Select("v.type, v.file_url, v.trash_url, v.delete_pending_until, v.created_at, v.updated_at").
		Joins("LEFT JOIN v_masjid_primary_urls v ON v.masjid_id = mp.masjid_profile_masjid_id").
		Where("mp.masjid_profile_masjid_id = ?", id).
		Where("v.type IS NOT NULL").
		Scan(&prim).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil primary files")
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"profile":       dto.FromModelMasjidProfile(&mp),
		"primary_files": prim,
	})
}

// =======================================================
// GET /api/masjids/:id/urls (primary + gallery)
// =======================================================

func (mc *MasjidController) GetMasjidURLs(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Primary files (VIEW)
	type primaryURLRow struct {
		Type               string     `gorm:"column:type" json:"type"`
		FileURL            string     `gorm:"column:file_url" json:"file_url"`
		TrashURL           *string    `gorm:"column:trash_url" json:"trash_url,omitempty"`
		DeletePendingUntil *time.Time `gorm:"column:delete_pending_until" json:"delete_pending_until,omitempty"`
		CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`
	}
	var primary []primaryURLRow
	if err := mc.DB.
		Table("masjids_profiles mp").
		Select("v.type, v.file_url, v.trash_url, v.delete_pending_until, v.created_at, v.updated_at").
		Joins("LEFT JOIN v_masjid_primary_urls v ON v.masjid_id = mp.masjid_profile_masjid_id").
		Where("mp.masjid_profile_masjid_id = ?", id).
		Where("v.type IS NOT NULL").
		Scan(&primary).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil primary files")
	}

	// Gallery (non-singleton) dari tabel masjid_urls
	type galleryURLRow struct {
		MasjidURLID uuid.UUID `gorm:"column:masjid_url_id" json:"masjid_url_id"`
		Type        string    `gorm:"column:masjid_url_type" json:"type"`
		FileURL     string    `gorm:"column:masjid_url_file_url" json:"file_url"`
		CreatedAt   time.Time `gorm:"column:masjid_url_created_at" json:"created_at"`
		UpdatedAt   time.Time `gorm:"column:masjid_url_updated_at" json:"updated_at"`
		IsPrimary   bool      `gorm:"column:masjid_url_is_primary" json:"is_primary"`
	}
	var gallery []galleryURLRow
	if err := mc.DB.
		Table("masjid_urls u").
		Select("u.masjid_url_id, u.masjid_url_type, u.masjid_url_file_url, u.masjid_url_created_at, u.masjid_url_updated_at, u.masjid_url_is_primary").
		Where("u.masjid_url_masjid_id = ? AND u.masjid_url_deleted_at IS NULL AND u.masjid_url_is_active = TRUE", id).
		Where("u.masjid_url_type = ?", "gallery").
		Order("u.masjid_url_created_at DESC").
		Scan(&gallery).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil gallery")
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"primary_files": primary,
		"gallery":       gallery,
	})
}


// letakkan di bawah file yang sama (masjid_controller.go) atau di utils yang kamu pakai untuk model Masjid

// normalize ke enum yang kamu pakai di model
func toVerificationStatus(s string) model.VerificationStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approved":
		return model.VerificationApproved
	case "rejected":
		return model.VerificationRejected
	default:
		return model.VerificationPending
	}
}

// Sinkronkan flag is_verified dan verified_at terhadap status
func syncVerificationFlags(m *model.MasjidModel) {
	// Pastikan status valid
	m.MasjidVerificationStatus = toVerificationStatus(string(m.MasjidVerificationStatus))

	now := time.Now()

	switch m.MasjidVerificationStatus {
	case model.VerificationApproved:
		// set true dan set verified_at jika belum ada
		m.MasjidIsVerified = true
		if m.MasjidVerifiedAt == nil {
			m.MasjidVerifiedAt = &now
		}
	case model.VerificationRejected:
		// tandai tidak terverifikasi; verified_at biarkan (riwayat) atau kosongkan jika kamu mau strict
		m.MasjidIsVerified = false
		// Jika ingin strict hapus tanggal verifikasi saat ditolak, uncomment:
		// m.MasjidVerifiedAt = nil
	default: // pending
		m.MasjidIsVerified = false
		// Untuk pending biasanya biarkan verified_at apa adanya (riwayat) — atau kosongkan jika mau:
		// m.MasjidVerifiedAt = nil
	}
}


// Ambil levels dari multipart: prioritas JSON tunggal "masjid_levels", fallback masjid_levels[]
func parseLevelsFromMultipart(c *fiber.Ctx) []string {
	// 1) coba json array tunggal
	if raw := strings.TrimSpace(c.FormValue("masjid_levels")); raw != "" {
		var arr []string
		if json.Unmarshal([]byte(raw), &arr) == nil {
			out := make([]string, 0, len(arr))
			for _, v := range arr {
				v = strings.TrimSpace(v)
				if v != "" { out = append(out, strings.ToLower(v)) }
			}
			return out
		}
	}
	// 2) kumpulkan repeated fields: masjid_levels[]
	if mf, _ := c.MultipartForm(); mf != nil {
		if vs, ok := mf.Value["masjid_levels[]"]; ok {
			out := make([]string, 0, len(vs))
			for _, v := range vs {
				v = strings.TrimSpace(v)
				if v != "" { out = append(out, strings.ToLower(v)) }
			}
			return out
		}
	}
	return nil
}
