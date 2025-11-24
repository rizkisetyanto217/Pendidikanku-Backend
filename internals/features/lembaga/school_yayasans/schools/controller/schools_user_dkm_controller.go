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

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	schoolDto "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/dto"
	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

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
func (mc *SchoolController) ensureOSS() error {
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
		s = "school"
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
// CreateSchoolDKM — multipart only, dengan teacher code "slug-xy"
// =======================================================
func (mc *SchoolController) CreateSchoolDKM(c *fiber.Ctx) error {
	t0 := time.Now()
	rid := uuid.New().String()
	lg := func(msg string, kv ...any) {
		var b strings.Builder
		b.WriteString("[CreateSchoolDKM][rid=")
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
	name := strings.TrimSpace(c.FormValue("school_name"))
	if name == "" {
		lg("validation failed", "reason", "school_name kosong")
		return helper.JsonError(c, fiber.StatusBadRequest, "school_name wajib diisi")
	}
	domain := normalizeDomainPtr(c.FormValue("school_domain"))
	bioShort := ptrStrTrim(c.FormValue("school_bio_short"))
	location := ptrStrTrim(c.FormValue("school_location"))
	city := ptrStrTrim(c.FormValue("school_city"))
	isSchool := parseBool(c.FormValue("school_is_islamic_school"))

	var yayasanID, planID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("school_yayasan_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			yayasanID = &id
		}
	}
	if s := strings.TrimSpace(c.FormValue("school_current_plan_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			planID = &id
		}
	}

	verifStatus := strings.TrimSpace(c.FormValue("school_verification_status"))
	if verifStatus == "" {
		verifStatus = "pending"
	}
	verifNotes := ptrStrTrim(c.FormValue("school_verification_notes"))

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
	var resp schoolDto.SchoolResp
	txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
		base := helper.SuggestSlugFromName(name)
		lg("slug suggest", "base", base)

		slug, err := helper.EnsureUniqueSlugCI(c.Context(), tx, "schools", "school_slug", base, nil, 100)
		if err != nil {
			lg("slug ensure unique failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}
		lg("slug ok", "slug", slug)

		now := time.Now()
		m := schoolModel.SchoolModel{
			SchoolID:            uuid.New(),
			SchoolYayasanID:     yayasanID,
			SchoolCurrentPlanID: planID,

			SchoolName:     name,
			SchoolBioShort: bioShort,
			SchoolLocation: location,
			SchoolCity:     city,

			SchoolDomain: domain,
			SchoolSlug:   slug,

			SchoolIsActive:           true,
			SchoolVerificationStatus: schoolModel.VerificationStatus(strings.ToLower(verifStatus)),
			SchoolVerificationNotes:  verifNotes,
			SchoolIsIslamicSchool:    isSchool,

			SchoolCreatedAt: now,
			SchoolUpdatedAt: now,
		}

		// Levels → JSONB
		if len(levels) > 0 {
			if jb, err := json.Marshal(levels); err == nil {
				m.SchoolLevels = datatypes.JSON(jb)
			}
		}
		syncVerificationFlags(&m)

		// ===== Teacher Code (hash) =====
		if manual := strings.TrimSpace(c.FormValue("school_teacher_code_plain")); manual != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(manual), bcrypt.DefaultCost)
			if err != nil {
				lg("teacher code hash failed (manual)", "err", err.Error())
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat hash teacher code")
			}
			m.SchoolTeacherCodeHash = hash
			m.SchoolTeacherCodeSetAt = &now
			c.Locals("teacher_code", manual) // kembalikan sekali via response
			lg("teacher code set (manual)")
		} else {
			plain, hash, setAt, err := makeTeacherCodeFromSlug(slug) // slug-xy
			if err != nil {
				lg("teacher code generate failed", "err", err.Error())
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat teacher code")
			}
			m.SchoolTeacherCodeHash = hash
			m.SchoolTeacherCodeSetAt = &setAt
			c.Locals("teacher_code", plain) // kembalikan sekali via response
			lg("teacher code set (auto)")
		}
		// ===============================

		if err := tx.Create(&m).Error; err != nil {
			lg("db create school failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan school")
		}
		lg("db create school ok", "school_id", m.SchoolID.String())

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

				publicURL, upErr := helperOSS.UploadAnyToOSS(c.Context(), mc.OSS, m.SchoolID, slot, fh)
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
					if m.SchoolIconURL != nil && strings.TrimSpace(*m.SchoolIconURL) != "" {
						m.SchoolIconURLOld = m.SchoolIconURL
						m.SchoolIconObjectKeyOld = m.SchoolIconObjectKey
						m.SchoolIconDeletePendingUntil = &retUntil
					}
					m.SchoolIconURL = &publicURL
					if key, err := helperOSS.KeyFromPublicURL(publicURL); err == nil {
						m.SchoolIconObjectKey = &key
					}
				case "logo":
					if m.SchoolLogoURL != nil && strings.TrimSpace(*m.SchoolLogoURL) != "" {
						m.SchoolLogoURLOld = m.SchoolLogoURL
						m.SchoolLogoObjectKeyOld = m.SchoolLogoObjectKey
						m.SchoolLogoDeletePendingUntil = &retUntil
					}
					m.SchoolLogoURL = &publicURL
					if key, err := helperOSS.KeyFromPublicURL(publicURL); err == nil {
						m.SchoolLogoObjectKey = &key
					}
				case "background":
					if m.SchoolBackgroundURL != nil && strings.TrimSpace(*m.SchoolBackgroundURL) != "" {
						m.SchoolBackgroundURLOld = m.SchoolBackgroundURL
						m.SchoolBackgroundObjectKeyOld = m.SchoolBackgroundObjectKey
						m.SchoolBackgroundDeletePendingUntil = &retUntil
					}
					m.SchoolBackgroundURL = &publicURL
					if key, err := helperOSS.KeyFromPublicURL(publicURL); err == nil {
						m.SchoolBackgroundObjectKey = &key
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

			m.SchoolUpdatedAt = time.Now()
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
		if err := helperAuth.GrantScopedRoleDKM(tx, userID, m.SchoolID); err != nil {
			lg("grant role failed", "err", err.Error())
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal grant peran DKM")
		}
		lg("grant role ok", "user_id", userID.String(), "school_id", m.SchoolID.String())

		resp = schoolDto.FromModel(&m)
		lg("build response ok", "school_id", resp.SchoolID, "slug", resp.SchoolSlug)
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
	if tc, _ := c.Locals("teacher_code").(string); tc != "" {
		return helper.JsonCreated(c, "School berhasil dibuat", fiber.Map{
			"item":         resp,
			"teacher_code": tc,
		})
	}
	return helper.JsonCreated(c, "School berhasil dibuat", resp)
}

// =======================================================
// GET /api/schools  (list + filter)
// =======================================================

func (mc *SchoolController) GetSchools(c *fiber.Ctx) error {
	tx := mc.DB.Model(&schoolModel.SchoolModel{})

	// q: ILIKE (match trigram index)
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		qq := "%" + q + "%"
		tx = tx.Where("(school_name ILIKE ? OR school_location ILIKE ? OR school_bio_short ILIKE ?)", qq, qq, qq)
	}

	// flags
	if v := strings.TrimSpace(c.Query("verified")); v != "" {
		tx = tx.Where("school_is_verified = ?", v == "true" || v == "1")
	}
	if v := strings.TrimSpace(c.Query("active")); v != "" {
		tx = tx.Where("school_is_active = ?", v == "true" || v == "1")
	}
	if v := strings.TrimSpace(c.Query("is_islamic_school")); v != "" {
		tx = tx.Where("school_is_islamic_school = ?", v == "true" || v == "1")
	}

	// relations
	if s := strings.TrimSpace(c.Query("yayasan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			tx = tx.Where("school_yayasan_id = ?", id)
		}
	}
	if s := strings.TrimSpace(c.Query("plan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			tx = tx.Where("school_current_plan_id = ?", id)
		}
	}

	// levels_any => OR of "school_levels ? ?"
	if s := strings.TrimSpace(c.Query("levels_any")); s != "" {
		parts := splitCSV(s)
		if len(parts) > 0 {
			orSQL := make([]string, 0, len(parts))
			args := make([]interface{}, 0, len(parts))
			for _, p := range parts {
				orSQL = append(orSQL, "school_levels ? ?")
				args = append(args, p)
			}
			tx = tx.Where("("+strings.Join(orSQL, " OR ")+")", args...)
		}
	}
	// levels_all => school_levels @> '["a","b"]'::jsonb
	if s := strings.TrimSpace(c.Query("levels_all")); s != "" {
		parts := splitCSV(s)
		if len(parts) > 0 {
			jb, _ := json.Marshal(parts)
			tx = tx.Where("school_levels @> ?::jsonb", string(jb))
		}
	}

	limit := clampInt(c.Query("limit"), 20, 1, 100)
	offset := clampInt(c.Query("offset"), 0, 0, 100000)
	tx = tx.Limit(limit).Offset(offset).Order("school_created_at DESC")

	var rows []schoolModel.SchoolModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
	}

	out := make([]schoolDto.SchoolResp, 0, len(rows))
	for i := range rows {
		out = append(out, schoolDto.FromModel(&rows[i]))
	}
	return helper.JsonOK(c, "OK", fiber.Map{
		"items":  out,
		"count":  len(out),
		"limit":  limit,
		"offset": offset,
	})
}

// normalize ke enum yang dipakai di package model
func toVerificationStatus(s string) schoolModel.VerificationStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approved":
		return schoolModel.VerificationApproved
	case "rejected":
		return schoolModel.VerificationRejected
	default:
		return schoolModel.VerificationPending
	}
}

// Sinkronkan flag is_verified dan verified_at terhadap status
func syncVerificationFlags(m *schoolModel.SchoolModel) {
	// Pastikan status valid
	m.SchoolVerificationStatus = toVerificationStatus(string(m.SchoolVerificationStatus))

	now := time.Now()
	switch m.SchoolVerificationStatus {
	case schoolModel.VerificationApproved:
		m.SchoolIsVerified = true
		if m.SchoolVerifiedAt == nil {
			m.SchoolVerifiedAt = &now
		}
	case schoolModel.VerificationRejected:
		m.SchoolIsVerified = false
	default: // pending
		m.SchoolIsVerified = false
	}
}

// Ambil levels dari multipart: prioritas JSON tunggal "school_levels", fallback school_levels[]
func parseLevelsFromMultipart(c *fiber.Ctx) []string {
	// 1) coba json array tunggal
	if raw := strings.TrimSpace(c.FormValue("school_levels")); raw != "" {
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
	// 2) kumpulkan repeated fields: school_levels[]
	if mf, _ := c.MultipartForm(); mf != nil {
		if vs, ok := mf.Value["school_levels[]"]; ok {
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
