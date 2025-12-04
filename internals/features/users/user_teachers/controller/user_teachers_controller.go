package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	schoolTeacherModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	classsectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	userdto "madinahsalam_backend/internals/features/users/user_teachers/dto"
	"madinahsalam_backend/internals/features/users/user_teachers/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserTeacherController struct {
	DB       *gorm.DB
	Validate *validator.Validate
	OSS      *helperOSS.OSSService
}

// ---- ctor
func NewUserTeacherController(db *gorm.DB, v *validator.Validate, oss *helperOSS.OSSService) *UserTeacherController {
	return &UserTeacherController{DB: db, Validate: v, OSS: oss}
}

const defaultUserTeacherRetention = 30 * 24 * time.Hour // fallback 30 hari

// ========================= helpers kecil =========================

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ganti jadi:
func jsonEqual(a, b datatypes.JSON) bool {
	if a == nil && b == nil {
		return true
	}
	return bytes.Equal(a, b)
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	if idStr == "" {
		return uuid.Nil, errors.New(name + " is required")
	}
	u, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, errors.New(name + " is invalid uuid")
	}
	return u, nil
}

// scope cek berdasarkan key
func withinUserTeacherScope(userTeacherID uuid.UUID, publicURL string) bool {
	key, err := helperOSS.KeyFromPublicURL(publicURL)
	if err != nil {
		return false
	}
	prefix := "user_teachers/" + userTeacherID.String() + "/"
	return strings.HasPrefix(key, prefix)
}

// NOTE: ganti implementasi auth sesuai kebutuhanmu (owner/admin).
func EnsureOwnerOrAdminUserTeacher(c *fiber.Ctx, ut *model.UserTeacherModel) error {
	return nil // untuk sementara, tanpa auth check
}

func reqID(c *fiber.Ctx) string {
	if v := c.Locals("reqid"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "-"
}

// isTeacherProfileCompleted: aturan sederhana kapan profil dianggap "lengkap"
func isTeacherProfileCompleted(m *model.UserTeacherModel) bool {
	trim := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.TrimSpace(*p)
	}

	// Wajib minimal:
	// - Nama snapshot (string langsung di field)
	// - Whatsapp
	// - Gender
	// - Field KEAHLIAN atau ShortBio
	nameOK := strings.TrimSpace(m.UserTeacherUserFullNameCache) != ""
	waOK := trim(m.UserTeacherWhatsappURL) != ""
	genderOK := trim(m.UserTeacherGender) != ""
	fieldOK := trim(m.UserTeacherField) != "" || trim(m.UserTeacherShortBio) != ""

	return nameOK && waOK && genderOK && fieldOK
}

func ctIsMultipart(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	return strings.HasPrefix(ct, "multipart/form-data") || strings.Contains(ct, "multipart/form-data")
}

// cari file avatar di beberapa nama field umum
func pickAvatarFile(c *fiber.Ctx) (fh *multipart.FileHeader, which string) {
	keys := []string{"avatar", "user_teacher_avatar", "user_teacher_avatar_url", "file"}
	for _, k := range keys {
		f, err := c.FormFile(k)
		if err == nil && f != nil && f.Size > 0 {
			return f, k
		}
	}
	return nil, ""
}

// multipart TANPA payload: baca per-field manual (tanpa BodyParser)
// - angka harus murni (tanpa komentar)
// - JSON harus valid
// - field time & slot-avatar lama diabaikan saat create
// multipart TANPA payload: baca per-field manual (tanpa BodyParser)
// - angka harus murni (tanpa komentar)
// - JSON harus valid
// - field time & slot-avatar lama diabaikan saat create
func parseMultipartNoPayload(c *fiber.Ctx, rid string, req *userdto.CreateUserTeacherRequest) error {
	get := func(k string) string { return strings.TrimSpace(c.FormValue(k)) }

	// wajib
	req.UserTeacherUserFullNameCache = get("user_teacher_user_full_name_cache")

	// profil ringkas
	req.UserTeacherField = get("user_teacher_field")
	req.UserTeacherShortBio = get("user_teacher_short_bio")
	req.UserTeacherLongBio = get("user_teacher_long_bio")
	req.UserTeacherGreeting = get("user_teacher_greeting")
	req.UserTeacherEducation = get("user_teacher_education")
	req.UserTeacherActivity = get("user_teacher_activity")

	// angka murni (tanpa koma/komen)
	if s := get("user_teacher_experience_years"); s != "" {
		n64, err := strconv.ParseInt(s, 10, 16)
		if err != nil {
			log.Printf("[user-teacher#create] reqid=%s invalid experience_years=%q: %v", rid, s, err)
			return fmt.Errorf("user_teacher_experience_years harus angka 0..80 (tanpa komentar)")
		}
		n := int16(n64)
		if n < 0 || n > 80 {
			return fmt.Errorf("user_teacher_experience_years harus 0..80")
		}
		req.UserTeacherExperienceYears = &n
	}

	// demografis
	req.UserTeacherGender = get("user_teacher_gender")
	req.UserTeacherLocation = get("user_teacher_location")
	req.UserTeacherCity = get("user_teacher_city")

	// JSONB
	if s := get("user_teacher_specialties"); s != "" {
		if !json.Valid([]byte(s)) {
			log.Printf("[user-teacher#create] reqid=%s specialties invalid json=%q", rid, s)
			return fmt.Errorf("user_teacher_specialties harus JSON valid")
		}
		j := datatypes.JSON([]byte(s))
		req.UserTeacherSpecialties = &j
	}
	if s := get("user_teacher_certificates"); s != "" {
		if !json.Valid([]byte(s)) {
			log.Printf("[user-teacher#create] reqid=%s certificates invalid json=%q", rid, s)
			return fmt.Errorf("user_teacher_certificates harus JSON valid")
		}
		j := datatypes.JSON([]byte(s))
		req.UserTeacherCertificates = &j
	}

	// sosial
	req.UserTeacherInstagramURL = get("user_teacher_instagram_url")
	req.UserTeacherWhatsappURL = get("user_teacher_whatsapp_url")
	req.UserTeacherYoutubeURL = get("user_teacher_youtube_url")
	req.UserTeacherLinkedinURL = get("user_teacher_linkedin_url")
	req.UserTeacherGithubURL = get("user_teacher_github_url")
	req.UserTeacherTelegramUsername = get("user_teacher_telegram_username")

	// title
	req.UserTeacherTitlePrefix = get("user_teacher_title_prefix")
	req.UserTeacherTitleSuffix = get("user_teacher_title_suffix")

	// flags
	if s := get("user_teacher_is_verified"); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			log.Printf("[user-teacher#create] reqid=%s is_verified invalid=%q: %v", rid, s, err)
			return fmt.Errorf("user_teacher_is_verified harus boolean (true/false)")
		}
		req.UserTeacherIsVerified = &b
	}
	if s := get("user_teacher_is_active"); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			log.Printf("[user-teacher#create] reqid=%s is_active invalid=%q: %v", rid, s, err)
			return fmt.Errorf("user_teacher_is_active harus boolean (true/false)")
		}
		req.UserTeacherIsActive = &b
	}

	// ⚠️ avatar_*_old & delete_pending_until sengaja diabaikan saat CREATE
	return nil
}

// ========================= CREATE =========================
// POST /api/user-teachers
// - multipart: file "avatar" + field "payload" (JSON CreateUserTeacherRequest)
// - json: body langsung CreateUserTeacherRequest
// ========================= CREATE =========================
// POST /api/user-teachers
// - multipart: file "avatar" + field "payload" (JSON CreateUserTeacherRequest)  ➜ rekomendasi
// - multipart: per-field text (tanpa "payload"), angka murni & JSON valid        ➜ juga didukung
// - json: body langsung CreateUserTeacherRequest
func (uc *UserTeacherController) Create(c *fiber.Ctx) error {
	rid := reqID(c)
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	log.Printf("[user-teacher#create] reqid=%s start ct=%q", rid, ct)

	var req userdto.CreateUserTeacherRequest

	// --- parsing payload ---
	if ctIsMultipart(ct) {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			log.Printf("[user-teacher#create] reqid=%s multipart with payload (len=%d)", rid, len(s))
			if err := json.Unmarshal([]byte(s), &req); err != nil {
				log.Printf("[user-teacher#create] reqid=%s payload json unmarshal error: %v", rid, err)
				return helper.JsonError(c, fiber.StatusBadRequest, "payload JSON tidak valid")
			}
		} else {
			log.Printf("[user-teacher#create] reqid=%s multipart without payload → parse per-field", rid)
			if err := parseMultipartNoPayload(c, rid, &req); err != nil {
				log.Printf("[user-teacher#create] reqid=%s multipart per-field parse error: %v", rid, err)
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			log.Printf("[user-teacher#create] reqid=%s json body parse error: %v", rid, err)
			return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
		}
	}

	// --- user_id dari token (anti-spoof) ---
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		log.Printf("[user-teacher#create] reqid=%s auth error: %v", rid, err)
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	req.UserTeacherUserID = userID
	log.Printf("[user-teacher#create] reqid=%s user_id=%s", rid, userID)

	// --- validasi payload ---
	if err := uc.Validate.Struct(req); err != nil {
		log.Printf("[user-teacher#create] reqid=%s validation error: %v", rid, err)
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	log.Printf("[user-teacher#create] reqid=%s validation OK name=%q field=%q expYears=%v",
		rid, req.UserTeacherUserFullNameCache, req.UserTeacherField, req.UserTeacherExperienceYears)

	// --- pastikan 1 user = 1 profile ---
	var exist int64
	if err := uc.DB.Model(&model.UserTeacherModel{}).
		Where("user_teacher_user_id = ?", userID).
		Count(&exist).Error; err != nil {
		log.Printf("[user-teacher#create] reqid=%s unique-check error: %v", rid, err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek duplikasi")
	}
	if exist > 0 {
		log.Printf("[user-teacher#create] reqid=%s conflict: profile already exists for user %s", rid, userID)
		return helper.JsonError(c, fiber.StatusConflict, "User sudah memiliki profil pengajar")
	}

	// --- map ke model ---
	m := req.ToModel()
	now := time.Now()
	m.UserTeacherID = uuid.New()
	m.UserTeacherCreatedAt = now
	m.UserTeacherUpdatedAt = now

	// --- upload avatar (opsional, hanya multipart) ---
	// --- upload avatar (opsional, hanya multipart) ---
	if ctIsMultipart(ct) {
		// 1) coba cari file dengan beberapa alias key
		if fh, which := pickAvatarFile(c); fh != nil {
			log.Printf("[user-teacher#create] reqid=%s avatar file found key=%q filename=%q size=%d",
				rid, which, fh.Filename, fh.Size)

			svc := uc.OSS
			if svc == nil {
				tmp, e := helperOSS.NewOSSServiceFromEnv("")
				if e != nil {
					log.Printf("[user-teacher#create] reqid=%s init OSS error: %v", rid, e)
					return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
				}
				svc = tmp
			}

			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, m.UserTeacherID, "avatar", fh)
			if upErr != nil {
				log.Printf("[user-teacher#create] reqid=%s upload avatar error: %v", rid, upErr)
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				log.Printf("[user-teacher#create] reqid=%s extract key error: %v", rid, kerr)
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (avatar)")
			}
			m.UserTeacherAvatarURL = &url
			m.UserTeacherAvatarObjectKey = &key
			log.Printf("[user-teacher#create] reqid=%s avatar uploaded url=%s key=%s", rid, url, key)

		} else {
			// 2) fallback: kalau tidak ada file, cek apakah ada URL tekstual yang valid
			if raw := strings.TrimSpace(c.FormValue("user_teacher_avatar_url")); raw != "" &&
				(strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://")) {

				if key, err := helperOSS.KeyFromPublicURL(raw); err == nil {
					// gunakan URL yang sudah publik dan berasal dari bucket yang sama
					m.UserTeacherAvatarURL = &raw
					m.UserTeacherAvatarObjectKey = &key
					log.Printf("[user-teacher#create] reqid=%s no file, reuse avatar url=%s key=%s", rid, raw, key)
				} else {
					log.Printf("[user-teacher#create] reqid=%s provided avatar_url not recognized: %v", rid, err)
					// tidak fatal — lanjut tanpa avatar
				}
			} else {
				log.Printf("[user-teacher#create] reqid=%s no avatar file or reusable url provided", rid)
			}
		}
	}

	// --- simpan DB ---
	if err := uc.DB.Create(&m).Error; err != nil {
		log.Printf("[user-teacher#create] reqid=%s insert error: %v", rid, err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat user_teacher")
	}
	log.Printf("[user-teacher#create] reqid=%s created id=%s", rid, m.UserTeacherID)

	// --- respon ---
	log.Printf("[user-teacher#create] reqid=%s success", rid)
	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": userdto.ToUserTeacherResponse(m),
	})
}

// ========================= PATCH (CORE) =========================
func (uc *UserTeacherController) applyPatch(c *fiber.Ctx, m *model.UserTeacherModel, _ bool) error {
	before := *m
	var req userdto.UpdateUserTeacherRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))

	now := time.Now()
	retention := helperOSS.GetRetentionDuration()
	if retention == 0 {
		retention = defaultUserTeacherRetention
	}
	deleteAfter := now.Add(retention)

	changedAvatar := false

	// --- parse payload ---
	if strings.HasPrefix(ct, "multipart/form-data") {
		// bodyparser akan mapping langsung semua field dengan tag form/json
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
		}

		// OSS svc
		svc := uc.OSS
		if svc == nil {
			tmp, e := helperOSS.NewOSSServiceFromEnv("")
			if e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
			}
			svc = tmp
		}

		// file avatar
		if fh, err := c.FormFile("avatar"); err == nil && fh != nil {
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, m.UserTeacherID, "avatar", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (avatar)")
			}

			// 2-slot
			if m.UserTeacherAvatarURL != nil && *m.UserTeacherAvatarURL != "" {
				m.UserTeacherAvatarURLOld = m.UserTeacherAvatarURL
				m.UserTeacherAvatarObjectKeyOld = m.UserTeacherAvatarObjectKey
				m.UserTeacherAvatarDeletePendingUntil = &deleteAfter
			}
			m.UserTeacherAvatarURL = &url
			m.UserTeacherAvatarObjectKey = &key
			changedAvatar = true
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
		}
	}

	// --- validasi DTO ---
	if err := uc.Validate.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// --- apply patch ke model ---
	req.ApplyPatch(m)
	m.UserTeacherUpdatedAt = now

	// ================= COMPLETION LOGIC =================
	prevCompleted := before.UserTeacherIsCompleted
	newCompleted := m.UserTeacherIsCompleted

	// Kalau profil ingin/masih completed, pastikan minimal gender + whatsapp terisi
	if newCompleted {
		genderOK := m.UserTeacherGender != nil && strings.TrimSpace(*m.UserTeacherGender) != ""
		waOK := m.UserTeacherWhatsappURL != nil && strings.TrimSpace(*m.UserTeacherWhatsappURL) != ""

		if !genderOK || !waOK {
			return helper.JsonError(
				c,
				fiber.StatusBadRequest,
				"Untuk menandai profil sebagai lengkap, minimal isi jenis kelamin dan nomor WhatsApp.",
			)
		}
	}

	// Atur completed_at:
	// - false -> true : set sekarang (jika belum pernah)
	// - true  -> false : kosongkan (optional, di sini kita reset)
	if !prevCompleted && newCompleted {
		if m.UserTeacherCompletedAt == nil {
			m.UserTeacherCompletedAt = &now
		}
	} else if prevCompleted && !newCompleted {
		m.UserTeacherCompletedAt = nil
	}
	// ===================================================

	// --- delta updates ---
	updates := map[string]any{
		"user_teacher_updated_at": m.UserTeacherUpdatedAt,
	}

	// helper inline
	applyIfChanged := func(col string, beforeVal, afterVal any) {
		if beforeVal != afterVal {
			updates[col] = afterVal
		}
	}
	applyIfChangedStr := func(col string, beforeVal, afterVal *string) {
		if derefStr(beforeVal) != derefStr(afterVal) {
			updates[col] = afterVal
		}
	}

	// ringkas
	applyIfChanged("user_teacher_user_full_name_cache", before.UserTeacherUserFullNameCache, m.UserTeacherUserFullNameCache)
	applyIfChangedStr("user_teacher_field", before.UserTeacherField, m.UserTeacherField)
	applyIfChangedStr("user_teacher_short_bio", before.UserTeacherShortBio, m.UserTeacherShortBio)
	applyIfChangedStr("user_teacher_long_bio", before.UserTeacherLongBio, m.UserTeacherLongBio)
	applyIfChangedStr("user_teacher_greeting", before.UserTeacherGreeting, m.UserTeacherGreeting)
	applyIfChangedStr("user_teacher_education", before.UserTeacherEducation, m.UserTeacherEducation)
	applyIfChangedStr("user_teacher_activity", before.UserTeacherActivity, m.UserTeacherActivity)
	applyIfChanged("user_teacher_experience_years", before.UserTeacherExperienceYears, m.UserTeacherExperienceYears)

	// demografis
	applyIfChangedStr("user_teacher_gender", before.UserTeacherGender, m.UserTeacherGender)
	applyIfChangedStr("user_teacher_location", before.UserTeacherLocation, m.UserTeacherLocation)
	applyIfChangedStr("user_teacher_city", before.UserTeacherCity, m.UserTeacherCity)

	// jsonb
	if !jsonEqual(before.UserTeacherSpecialties, m.UserTeacherSpecialties) {
		updates["user_teacher_specialties"] = m.UserTeacherSpecialties
	}
	if !jsonEqual(before.UserTeacherCertificates, m.UserTeacherCertificates) {
		updates["user_teacher_certificates"] = m.UserTeacherCertificates
	}

	// sosial
	applyIfChangedStr("user_teacher_instagram_url", before.UserTeacherInstagramURL, m.UserTeacherInstagramURL)
	applyIfChangedStr("user_teacher_whatsapp_url", before.UserTeacherWhatsappURL, m.UserTeacherWhatsappURL)
	applyIfChangedStr("user_teacher_youtube_url", before.UserTeacherYoutubeURL, m.UserTeacherYoutubeURL)
	applyIfChangedStr("user_teacher_linkedin_url", before.UserTeacherLinkedinURL, m.UserTeacherLinkedinURL)
	applyIfChangedStr("user_teacher_github_url", before.UserTeacherGithubURL, m.UserTeacherGithubURL)
	applyIfChangedStr("user_teacher_telegram_username", before.UserTeacherTelegramUsername, m.UserTeacherTelegramUsername)

	// title
	applyIfChangedStr("user_teacher_title_prefix", before.UserTeacherTitlePrefix, m.UserTeacherTitlePrefix)
	applyIfChangedStr("user_teacher_title_suffix", before.UserTeacherTitleSuffix, m.UserTeacherTitleSuffix)

	// avatar current
	applyIfChangedStr("user_teacher_avatar_url", before.UserTeacherAvatarURL, m.UserTeacherAvatarURL)
	applyIfChangedStr("user_teacher_avatar_object_key", before.UserTeacherAvatarObjectKey, m.UserTeacherAvatarObjectKey)

	// avatar shadow
	if changedAvatar {
		updates["user_teacher_avatar_url_old"] = m.UserTeacherAvatarURLOld
		updates["user_teacher_avatar_object_key_old"] = m.UserTeacherAvatarObjectKeyOld
		updates["user_teacher_avatar_delete_pending_until"] = m.UserTeacherAvatarDeletePendingUntil
	}

	// flags
	applyIfChanged("user_teacher_is_verified", before.UserTeacherIsVerified, m.UserTeacherIsVerified)
	applyIfChanged("user_teacher_is_active", before.UserTeacherIsActive, m.UserTeacherIsActive)

	// NEW: completion flags
	applyIfChanged("user_teacher_is_completed", before.UserTeacherIsCompleted, m.UserTeacherIsCompleted)
	applyIfChanged("user_teacher_completed_at", before.UserTeacherCompletedAt, m.UserTeacherCompletedAt)

	// tidak ada perubahan nyata
	if len(updates) == 1 {
		return helper.JsonOK(c, "Tidak ada perubahan", fiber.Map{
			"item": userdto.ToUserTeacherResponse(*m),
		})
	}

	// simpan perubahan ke user_teachers
	if err := uc.DB.Model(m).Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// === SNAPSHOT SYNC ke school_teachers ===
	changedSnapshot :=
		before.UserTeacherUserFullNameCache != m.UserTeacherUserFullNameCache ||
			derefStr(before.UserTeacherAvatarURL) != derefStr(m.UserTeacherAvatarURL) ||
			derefStr(before.UserTeacherWhatsappURL) != derefStr(m.UserTeacherWhatsappURL) ||
			derefStr(before.UserTeacherTitlePrefix) != derefStr(m.UserTeacherTitlePrefix) ||
			derefStr(before.UserTeacherTitleSuffix) != derefStr(m.UserTeacherTitleSuffix)

	if changedSnapshot {

		set := map[string]any{
			"school_teacher_user_teacher_full_name_cache":    m.UserTeacherUserFullNameCache,
			"school_teacher_user_teacher_avatar_url_cache":   m.UserTeacherAvatarURL,
			"school_teacher_user_teacher_whatsapp_url_cache": m.UserTeacherWhatsappURL,
			"school_teacher_user_teacher_title_prefix_cache": m.UserTeacherTitlePrefix,
			"school_teacher_user_teacher_title_suffix_cache": m.UserTeacherTitleSuffix,
			"school_teacher_updated_at":                      time.Now(),
		}

		// Guard opsional: kalau migrasinya belum naik, jangan bikin 500.
		if !uc.DB.Migrator().HasColumn(&schoolTeacherModel.SchoolTeacherModel{}, "school_teacher_user_teacher_user_full_name_cache") {
			log.Printf("[user-teacher#patch] snapshot columns not found — skip sync to school_teachers")
		} else {
			if err := uc.DB.Model(&schoolTeacherModel.SchoolTeacherModel{}).
				Where("school_teacher_user_teacher_id = ? AND school_teacher_deleted_at IS NULL", m.UserTeacherID).
				Updates(set).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError,
					"Profil tersimpan, tapi gagal sync snapshot pengajar di school")
			}
		}
	}

	// === SNAPSHOT SYNC ke class_sections (teacher & assistant) ===
	if changedSnapshot {
		// ... (bagian ini sama persis seperti punyamu sebelumnya, tidak perlu diubah)
		// aku biarkan singkat supaya kamu fokus ke logic completed di atas
		// --------------------------------
		hasTeacherSnap := uc.DB.Migrator().HasColumn(&classsectionModel.ClassSectionModel{}, "class_section_teacher_snapshot")
		hasAssistantSnap := uc.DB.Migrator().HasColumn(&classsectionModel.ClassSectionModel{}, "class_section_assistant_teacher_snapshot")


		if !(hasTeacherSnap || hasAssistantSnap) {
			log.Printf("[user-teacher#patch] class_sections snapshot columns not found — skip sync to class_sections")
		} else {
			var mtIDs []uuid.UUID
			if err := uc.DB.
				Model(&schoolTeacherModel.SchoolTeacherModel{}).
				Where("school_teacher_user_teacher_id = ? AND school_teacher_deleted_at IS NULL", m.UserTeacherID).
				Pluck("school_teacher_id", &mtIDs).Error; err != nil {
				log.Printf("[user-teacher#patch] failed pluck school_teacher ids: %v", err)
			}

			if len(mtIDs) > 0 {
				type smallTeacherSnap struct {
					UserTeacherID uuid.UUID `json:"user_teacher_id"`
					Name          string    `json:"name"`
					AvatarURL     *string   `json:"avatar_url,omitempty"`
					WhatsappURL   *string   `json:"whatsapp_url,omitempty"`
					TitlePrefix   *string   `json:"title_prefix,omitempty"`
					TitleSuffix   *string   `json:"title_suffix,omitempty"`
					UpdatedAt     time.Time `json:"updated_at"`
				}
				payload := smallTeacherSnap{
					UserTeacherID: m.UserTeacherID,
					Name:          m.UserTeacherUserFullNameCache,
					AvatarURL:     m.UserTeacherAvatarURL,
					WhatsappURL:   m.UserTeacherWhatsappURL,
					TitlePrefix:   m.UserTeacherTitlePrefix,
					TitleSuffix:   m.UserTeacherTitleSuffix,
					UpdatedAt:     now,
				}
				b, _ := json.Marshal(payload)
				jsonb := datatypes.JSON(b)

				setTeacher := map[string]any{}
				setAssistant := map[string]any{}
				if hasTeacherSnap {
					setTeacher["class_section_teacher_snapshot"] = jsonb
				}
				if hasAssistantSnap {
					setAssistant["class_section_assistant_teacher_snapshot"] = jsonb
				}


				if hasTeacherSnap {
					if err := uc.DB.
						Model(&classsectionModel.ClassSectionModel{}).
						Where("class_section_teacher_id IN ? AND class_section_deleted_at IS NULL", mtIDs).
						Updates(setTeacher).Error; err != nil {
						log.Printf("[user-teacher#patch] failed sync class_section_teacher_snapshot: %v", err)
					}
				}

				if hasAssistantSnap {
					if err := uc.DB.
						Model(&classsectionModel.ClassSectionModel{}).
						Where("class_section_assistant_teacher_id IN ? AND class_section_deleted_at IS NULL", mtIDs).
						Updates(setAssistant).Error; err != nil {
						log.Printf("[user-teacher#patch] failed sync class_section_assistant_teacher_snapshot: %v", err)
					}
				}
			}
		}
	}

	// === SNAPSHOT SYNC ke class_section_subject_teachers (teacher & assistant) ===
	{
		// (bagian ini juga sama seperti sebelumnya, tidak perlu disentuh)
		hasTeacherSnap := uc.DB.Migrator().HasColumn(&csstModel.ClassSectionSubjectTeacherModel{}, "class_section_subject_teacher_teacher_snapshot")
		hasAssistantSnap := uc.DB.Migrator().HasColumn(&csstModel.ClassSectionSubjectTeacherModel{}, "class_section_subject_teacher_assistant_teacher_snapshot")
		hasTeacherID := uc.DB.Migrator().HasColumn(&csstModel.ClassSectionSubjectTeacherModel{}, "class_section_subject_teacher_teacher_id")
		hasAssistantID := uc.DB.Migrator().HasColumn(&csstModel.ClassSectionSubjectTeacherModel{}, "class_section_subject_teacher_assistant_teacher_id")
		hasUpdatedAt := uc.DB.Migrator().HasColumn(&csstModel.ClassSectionSubjectTeacherModel{}, "class_section_subject_teacher_updated_at")

		if !(hasTeacherSnap || hasAssistantSnap) {
			log.Printf("[user-teacher#patch] CSST snapshot columns not found — skip sync to class_section_subject_teachers")
		} else {
			var mtIDs []uuid.UUID
			if err := uc.DB.
				Model(&schoolTeacherModel.SchoolTeacherModel{}).
				Where("school_teacher_user_teacher_id = ? AND school_teacher_deleted_at IS NULL", m.UserTeacherID).
				Pluck("school_teacher_id", &mtIDs).Error; err != nil {
				log.Printf("[user-teacher#patch] failed pluck school_teacher ids for CSST: %v", err)
			}

			if len(mtIDs) > 0 {
				type teacherMiniSnap struct {
					UserTeacherID uuid.UUID `json:"user_teacher_id"`
					Name          string    `json:"name"`
					AvatarURL     *string   `json:"avatar_url,omitempty"`
					WhatsappURL   *string   `json:"whatsapp_url,omitempty"`
					TitlePrefix   *string   `json:"title_prefix,omitempty"`
					TitleSuffix   *string   `json:"title_suffix,omitempty"`
					UpdatedAt     time.Time `json:"updated_at"`
				}
				minSnap := teacherMiniSnap{
					UserTeacherID: m.UserTeacherID,
					Name:          m.UserTeacherUserFullNameCache,
					AvatarURL:     m.UserTeacherAvatarURL,
					WhatsappURL:   m.UserTeacherWhatsappURL,
					TitlePrefix:   m.UserTeacherTitlePrefix,
					TitleSuffix:   m.UserTeacherTitleSuffix,
					UpdatedAt:     now,
				}
				b, _ := json.Marshal(minSnap)
				jsonb := datatypes.JSON(b)

				setTeacher := map[string]any{}
				setAssistant := map[string]any{}
				if hasTeacherSnap {
					setTeacher["class_section_subject_teacher_teacher_snapshot"] = jsonb
				}
				if hasAssistantSnap {
					setAssistant["class_section_subject_teacher_assistant_teacher_snapshot"] = jsonb
				}
				if hasUpdatedAt {
					setTeacher["class_section_subject_teacher_updated_at"] = now
					setAssistant["class_section_subject_teacher_updated_at"] = now
				}

				if hasTeacherSnap && hasTeacherID {
					if err := uc.DB.
						Model(&csstModel.ClassSectionSubjectTeacherModel{}).
						Where("class_section_subject_teacher_teacher_id IN ? AND class_section_subject_teacher_deleted_at IS NULL", mtIDs).
						Updates(setTeacher).Error; err != nil {
						log.Printf("[user-teacher#patch] failed sync CSST teacher_snapshot: %v", err)
					}
				}

				if hasAssistantSnap && hasAssistantID {
					if err := uc.DB.
						Model(&csstModel.ClassSectionSubjectTeacherModel{}).
						Where("class_section_subject_teacher_assistant_teacher_id IN ? AND class_section_subject_teacher_deleted_at IS NULL", mtIDs).
						Updates(setAssistant).Error; err != nil {
						log.Printf("[user-teacher#patch] failed sync CSST assistant_teacher_snapshot: %v", err)
					}
				}
			}
		}
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": userdto.ToUserTeacherResponse(*m),
	})
}

// PATCH /user-teachers/me
// - Selalu update profil pengajar milik user dari token (user_teacher_user_id)
// - Mendukung:
//   - JSON biasa (Content-Type: application/json)
//   - multipart/form-data dengan field "payload" (JSON string) + file image (avatar)
func (uc *UserTeacherController) PatchMe(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Ambil existing row by user_id
	var before model.UserTeacherModel
	if err := uc.DB.First(&before, "user_teacher_user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil pengajar tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, "multipart/form-data")

	var in userdto.UpdateUserTeacherRequest

	// =========== PARSE PAYLOAD ===========
	if isMultipart {
		// Prioritas: payload JSON string (sama pola dengan user profile)
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := c.App().Config().JSONDecoder([]byte(s), &in); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Invalid payload JSON")
			}
		} else {
			// Langsung bind form-data → struct (pakai tag form:"...")
			if err := c.BodyParser(&in); err != nil {
				log.Println("[WARN] multipart BodyParser error:", err)
			}
		}
	} else {
		// JSON biasa
		if err := c.BodyParser(&in); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request format")
		}
	}

	// =========== APPLY PATCH KE MODEL ===========
	after := before
	in.ApplyPatch(&after)

	now := time.Now()
	after.UserTeacherUpdatedAt = now

	// =========== HANDLE AVATAR (FILE) via MULTIPART ===========
	if isMultipart {
		svc, err := uc.ensureOSS()
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
		}

		// Asumsi getImageFormFile akan ambil file dari field standar (misal: "avatar")
		if fh, err := getImageFormFile(c); err == nil && fh != nil {
			ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
			defer cancel()

			// folder/category pakai "teacher-avatar" biar beda dengan user profile
			url, upErr := helperOSS.UploadImageToOSS(ctx, svc, userID, "teacher-avatar", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (avatar)")
			}

			// 2-slot (old → pending delete), pakai nilai sebelum update
			if before.UserTeacherAvatarURL != nil && *before.UserTeacherAvatarURL != "" {
				due := now.Add(helperOSS.GetRetentionDuration())
				after.UserTeacherAvatarURLOld = before.UserTeacherAvatarURL
				after.UserTeacherAvatarObjectKeyOld = before.UserTeacherAvatarObjectKey
				after.UserTeacherAvatarDeletePendingUntil = &due
			}

			// Set avatar baru
			after.UserTeacherAvatarURL = &url
			after.UserTeacherAvatarObjectKey = &key
		}
	}

	// =========== AUTO FLAG: is_completed & completed_at ===========
	wasCompleted := before.UserTeacherIsCompleted
	nowCompleted := isTeacherProfileCompleted(&after) // helper di bawah

	if nowCompleted && !wasCompleted {
		after.UserTeacherIsCompleted = true
		if after.UserTeacherCompletedAt == nil {
			after.UserTeacherCompletedAt = &now
		}
	} else if !nowCompleted && wasCompleted {
		// turun kasta: profil jadi tidak lengkap lagi
		after.UserTeacherIsCompleted = false
		after.UserTeacherCompletedAt = nil
	}
	// kalau sama-sama true atau sama-sama false → biarkan timestamp apa adanya

	// =========== SAVE ===========
	if err := uc.DB.Save(&after).Error; err != nil {
		log.Println("[ERROR] Failed to save user_teacher:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan profil pengajar")
	}

	return helper.JsonUpdated(c, "Profil pengajar diperbarui", userdto.ToUserTeacherResponse(after))
}

func (uc *UserTeacherController) Patch(c *fiber.Ctx) error {
	utID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	var m model.UserTeacherModel
	if err := uc.DB.First(&m, "user_teacher_id = ?", utID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User teacher tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if err := EnsureOwnerOrAdminUserTeacher(c, &m); err != nil {
		return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
	}
	return uc.applyPatch(c, &m, true)
}

// ========================= DELETE FILE =========================
// DELETE /api/user-teachers/:id/files  { "url": "https://..." }
type delFileReq struct {
	URL string `json:"url"`
}

func (uc *UserTeacherController) DeleteFile(c *fiber.Ctx) error {
	utID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body delFileReq
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.URL) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (butuh url)")
	}

	// scope check
	if !withinUserTeacherScope(utID, body.URL) {
		return helper.JsonError(c, fiber.StatusForbidden, "URL di luar scope user_teacher ini")
	}

	// move ke spam (retention/orphan cleaner di luar)
	spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(body.URL, 15*time.Second)
	if mvErr != nil {
		return helper.JsonError(c, fiber.StatusBadGateway, fmt.Sprintf("Gagal memindahkan ke spam: %v", mvErr))
	}

	// bersihkan kolom yang menunjuk URL tsb (best-effort)
	var m model.UserTeacherModel
	if err := uc.DB.First(&m, "user_teacher_id = ?", utID).Error; err == nil {
		changed := false
		now := time.Now()

		if m.UserTeacherAvatarURL != nil && *m.UserTeacherAvatarURL == body.URL {
			m.UserTeacherAvatarURL = nil
			m.UserTeacherAvatarObjectKey = nil
			changed = true
		}
		if m.UserTeacherAvatarURLOld != nil && *m.UserTeacherAvatarURLOld == body.URL {
			m.UserTeacherAvatarURLOld = nil
			m.UserTeacherAvatarObjectKeyOld = nil
			m.UserTeacherAvatarDeletePendingUntil = nil
			changed = true
		}

		if changed {
			m.UserTeacherUpdatedAt = now
			_ = uc.DB.Save(&m).Error // best-effort
		}
	}

	return helper.JsonOK(c, "Dipindahkan ke spam", fiber.Map{
		"from_url": body.URL,
		"spam_url": spamURL,
	})
}

func getImageFormFile(c *fiber.Ctx) (*multipart.FileHeader, error) {
	names := []string{"avatar", "image", "file", "photo", "picture"}
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh, nil
		}
	}
	return nil, errors.New("gambar tidak ditemukan")
}

func (upc *UserTeacherController) ensureOSS() (*helperOSS.OSSService, error) {
	if upc.OSS != nil {
		return upc.OSS, nil
	}
	svc, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		return nil, err
	}
	upc.OSS = svc
	return svc, nil
}
