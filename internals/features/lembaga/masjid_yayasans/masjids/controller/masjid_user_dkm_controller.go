package controller

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	masjidDto "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/dto"
	masjidModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

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

// ===== Helper: 2-char base36 & generator dari slug =====
func randBase36(n int) (string, error) {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(alphabet[x.Int64()])
	}
	return b.String(), nil
}

func makeTeacherCodeFromSlug(slug string) (plain string, hash []byte, setAt time.Time, err error) {
	s := strings.TrimSpace(slug)
	if s == "" {
		s = "masjid"
	}
	sfx, err := randBase36(2) // hanya 2 huruf/angka
	if err != nil {
		return "", nil, time.Time{}, err
	}
	plain = fmt.Sprintf("%s-%s", s, sfx)
	hash, err = bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	return plain, hash, time.Now(), nil
}

// =======================================================
// CreateMasjidDKM — multipart only, dengan teacher code "slug-xy"
// =======================================================
func (mc *MasjidController) CreateMasjidDKM(c *fiber.Ctx) error {
	t0 := time.Now()
	rid := uuid.New().String()
	lg := func(msg string, kv ...any) {
		var b strings.Builder
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

	// Content-Type must multipart
	ct := strings.ToLower(c.Get("Content-Type"))
	if !strings.Contains(ct, "multipart/form-data") {
		lg("unsupported media type", "ct", ct)
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

	// Files (opsional)
	iconFH, _ := c.FormFile("icon")
	logoFH, _ := c.FormFile("logo")
	bgFH, _ := c.FormFile("background")

	// Kompat lama: "file" + "slot"
	compatFH, _ := c.FormFile("file")
	slot := strings.ToLower(strings.TrimSpace(c.FormValue("slot")))
	if slot == "" {
		slot = "logo"
	}
	if slot != "icon" && slot != "logo" && slot != "background" && slot != "misc" {
		lg("validation failed", "reason", "slot invalid", "slot", slot)
		return helper.JsonError(c, fiber.StatusBadRequest, "slot harus salah satu dari: icon, logo, background, misc")
	}

	lg("request begin",
		"name", name,
		"domain", safeStrPtr(domain),
		"city", safeStrPtr(city),
		"isSchool", isSchool,
		"levels_len", len(levels),
		"hasIcon", iconFH != nil,
		"hasLogo", logoFH != nil,
		"hasBackground", bgFH != nil,
		"compatSlot", slot,
		"hasCompatFile", compatFH != nil,
	)

	// TX
	var resp masjidDto.MasjidResp
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
		m := masjidModel.MasjidModel{
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
			MasjidVerificationStatus: masjidModel.VerificationStatus(strings.ToLower(verifStatus)),
			MasjidVerificationNotes:  verifNotes,
			MasjidIsIslamicSchool:    isSchool,

			MasjidCreatedAt: now,
			MasjidUpdatedAt: now,
		}

		// Levels → JSONB
		if len(levels) > 0 {
			if jb, err := json.Marshal(levels); err == nil {
				m.MasjidLevels = datatypes.JSON(jb)
			}
		}
		syncVerificationFlags(&m)

		// ===== Teacher Code (hash) =====
		if manual := strings.TrimSpace(c.FormValue("masjid_teacher_code_plain")); manual != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(manual), bcrypt.DefaultCost)
			if err != nil {
				lg("teacher code hash failed (manual)", "err", err.Error())
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat hash teacher code")
			}
			m.MasjidTeacherCodeHash = hash
			m.MasjidTeacherCodeSetAt = &now
			c.Locals("teacher_code_plain", manual) // kembalikan sekali via response
			lg("teacher code set (manual)")
		} else {
			plain, hash, setAt, err := makeTeacherCodeFromSlug(slug) // slug-xy
			if err != nil {
				lg("teacher code generate failed", "err", err.Error())
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat teacher code")
			}
			m.MasjidTeacherCodeHash = hash
			m.MasjidTeacherCodeSetAt = &setAt
			c.Locals("teacher_code_plain", plain) // kembalikan sekali via response
			lg("teacher code set (auto)")
		}
		// ===============================

		if err := tx.Create(&m).Error; err != nil {
			lg("db create masjid failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan masjid")
		}
		lg("db create masjid ok", "masjid_id", m.MasjidID.String())

		// Upload file opsional
		if iconFH != nil || logoFH != nil || bgFH != nil || compatFH != nil {
			if err := mc.ensureOSS(); err != nil {
				lg("oss not configured", "err", err.Error())
				return fiber.NewError(fiber.StatusFailedDependency, err.Error())
			}
			lg("oss ready", "bucket", mc.OSS.BucketName)

			uploadOne := func(slot string, fh *multipart.FileHeader) error {
				if fh == nil {
					return nil
				}
				lg("upload begin", "slot", slot, "name", fh.Filename, "size", fh.Size)

				publicURL, upErr := helperOSS.UploadAnyToOSS(c.Context(), mc.OSS, m.MasjidID, slot, fh)
				if upErr != nil {
					var fe *fiber.Error
					if errors.As(upErr, &fe) {
						lg("upload failed (fiber error)", "code", fe.Code, "msg", fe.Message)
						return fiber.NewError(fe.Code, fe.Message)
					}
					lg("upload failed", "err", upErr.Error())
					return fiber.NewError(fiber.StatusBadGateway, "Gagal upload ke OSS")
				}
				lg("upload ok", "slot", slot, "url", publicURL)

				retUntil := time.Now().Add(helperOSS.GetRetentionDuration())
				switch slot {
				case "icon":
					if m.MasjidIconURL != nil && strings.TrimSpace(*m.MasjidIconURL) != "" {
						m.MasjidIconURLOld = m.MasjidIconURL
						m.MasjidIconObjectKeyOld = m.MasjidIconObjectKey
						m.MasjidIconDeletePendingUntil = &retUntil
					}
					m.MasjidIconURL = &publicURL
					if key, err := helperOSS.KeyFromPublicURL(publicURL); err == nil {
						m.MasjidIconObjectKey = &key
					}
				case "logo":
					if m.MasjidLogoURL != nil && strings.TrimSpace(*m.MasjidLogoURL) != "" {
						m.MasjidLogoURLOld = m.MasjidLogoURL
						m.MasjidLogoObjectKeyOld = m.MasjidLogoObjectKey
						m.MasjidLogoDeletePendingUntil = &retUntil
					}
					m.MasjidLogoURL = &publicURL
					if key, err := helperOSS.KeyFromPublicURL(publicURL); err == nil {
						m.MasjidLogoObjectKey = &key
					}
				case "background":
					if m.MasjidBackgroundURL != nil && strings.TrimSpace(*m.MasjidBackgroundURL) != "" {
						m.MasjidBackgroundURLOld = m.MasjidBackgroundURL
						m.MasjidBackgroundObjectKeyOld = m.MasjidBackgroundObjectKey
						m.MasjidBackgroundDeletePendingUntil = &retUntil
					}
					m.MasjidBackgroundURL = &publicURL
					if key, err := helperOSS.KeyFromPublicURL(publicURL); err == nil {
						m.MasjidBackgroundObjectKey = &key
					}
				default:
				}
				return nil
			}

			if err := uploadOne("icon", iconFH); err != nil {
				return err
			}
			if err := uploadOne("logo", logoFH); err != nil {
				return err
			}
			if err := uploadOne("background", bgFH); err != nil {
				return err
			}

			// kompat lama
			if compatFH != nil {
				useCompat := (slot == "icon" && iconFH == nil) ||
					(slot == "logo" && logoFH == nil) ||
					(slot == "background" && bgFH == nil) ||
					(slot == "misc")
				if useCompat {
					if err := uploadOne(slot, compatFH); err != nil {
						return err
					}
				} else {
					lg("skip compat file because explicit field already present", "slot", slot)
				}
			}

			m.MasjidUpdatedAt = time.Now()
			if err := tx.Save(&m).Error; err != nil {
				lg("db save metadata failed", "err", err.Error())
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan metadata file")
			}
			lg("db save metadata ok")
		} else {
			lg("no files attached")
		}

		// Roles
		_ = helperAuth.EnsureGlobalRole(tx, userID, "user", &userID)
		if err := helperAuth.GrantScopedRoleDKM(tx, userID, m.MasjidID); err != nil {
			lg("grant role failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal grant peran DKM")
		}
		lg("grant role ok", "user_id", userID.String(), "masjid_id", m.MasjidID.String())

		resp = masjidDto.FromModel(&m)
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

	// Sertakan plaintext teacher code SEKALI di response
	if tc, _ := c.Locals("teacher_code_plain").(string); tc != "" {
		return helper.JsonCreated(c, "Masjid berhasil dibuat", fiber.Map{
			"item":               resp,
			"teacher_code_plain": tc,
		})
	}
	return helper.JsonCreated(c, "Masjid berhasil dibuat", resp)
}

// =======================================================
// GET /api/masjids  (list + filter)
// =======================================================

func (mc *MasjidController) GetMasjids(c *fiber.Ctx) error {
	tx := mc.DB.Model(&masjidModel.MasjidModel{})

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

	var rows []masjidModel.MasjidModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid")
	}

	out := make([]masjidDto.MasjidResp, 0, len(rows))
	for i := range rows {
		out = append(out, masjidDto.FromModel(&rows[i]))
	}
	return helper.JsonOK(c, "OK", fiber.Map{
		"items":  out,
		"count":  len(out),
		"limit":  limit,
		"offset": offset,
	})
}

// normalize ke enum yang dipakai di package model
func toVerificationStatus(s string) masjidModel.VerificationStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approved":
		return masjidModel.VerificationApproved
	case "rejected":
		return masjidModel.VerificationRejected
	default:
		return masjidModel.VerificationPending
	}
}

// Sinkronkan flag is_verified dan verified_at terhadap status
func syncVerificationFlags(m *masjidModel.MasjidModel) {
	// Pastikan status valid
	m.MasjidVerificationStatus = toVerificationStatus(string(m.MasjidVerificationStatus))

	now := time.Now()
	switch m.MasjidVerificationStatus {
	case masjidModel.VerificationApproved:
		m.MasjidIsVerified = true
		if m.MasjidVerifiedAt == nil {
			m.MasjidVerifiedAt = &now
		}
	case masjidModel.VerificationRejected:
		m.MasjidIsVerified = false
	default: // pending
		m.MasjidIsVerified = false
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
				if v != "" {
					out = append(out, strings.ToLower(v))
				}
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
				if v != "" {
					out = append(out, strings.ToLower(v))
				}
			}
			return out
		}
	}
	return nil
}
