package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	userdto "masjidku_backend/internals/features/users/user_teachers/dto"
	"masjidku_backend/internals/features/users/user_teachers/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	mtModel "masjidku_backend/internals/features/lembaga/teachers_students/model"

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

// ========================= CREATE =========================
// POST /api/user-teachers
// - multipart: file "avatar" + field "payload" (JSON CreateUserTeacherRequest)
// - json: body langsung CreateUserTeacherRequest
func (uc *UserTeacherController) Create(c *fiber.Ctx) error {
	var req userdto.CreateUserTeacherRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))

	// --- parse payload ---
	if strings.HasPrefix(ct, "multipart/form-data") {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := json.Unmarshal([]byte(s), &req); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "payload JSON tidak valid")
			}
		} else {
			if err := c.BodyParser(&req); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
			}
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
		}
	}

	// --- ambil user_id dari token ---
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	req.UserTeacherUserID = userID // override, biar ga bisa spoof

	// --- validasi payload ---
	if err := uc.Validate.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// pastikan unique per user
	var exist int64
	if err := uc.DB.Model(&model.UserTeacherModel{}).
		Where("user_teacher_user_id = ?", userID).
		Count(&exist).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek duplikasi")
	}
	if exist > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "User sudah memiliki profil pengajar")
	}

	// --- map ke model ---
	m := req.ToModel()
	now := time.Now()
	m.UserTeacherID = uuid.New() // generate selalu baru di sini
	m.UserTeacherCreatedAt = now
	m.UserTeacherUpdatedAt = now

	// --- upload avatar kalau multipart ---
	if strings.HasPrefix(ct, "multipart/form-data") {
		if fh, err := c.FormFile("avatar"); err == nil && fh != nil {
			svc := uc.OSS
			if svc == nil {
				tmp, e := helperOSS.NewOSSServiceFromEnv("")
				if e != nil {
					return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
				}
				svc = tmp
			}
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, m.UserTeacherID, "avatar", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (avatar)")
			}
			m.UserTeacherAvatarURL = &url
			m.UserTeacherAvatarObjectKey = &key
		}
	}

	// --- simpan ---
	if err := uc.DB.Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat user_teacher")
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": userdto.ToUserTeacherResponse(m),
	})
}

// GET /api/user-teachers/me
func (uc *UserTeacherController) GetMe(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var m model.UserTeacherModel
	if err := uc.DB.First(&m, "user_teacher_user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil pengajar belum dibuat")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

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
	applyIfChanged("user_teacher_name", before.UserTeacherName, m.UserTeacherName)
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

	// === SNAPSHOT SYNC ke masjid_teachers ===
	changedSnapshot :=
		before.UserTeacherName != m.UserTeacherName ||
			derefStr(before.UserTeacherAvatarURL) != derefStr(m.UserTeacherAvatarURL) ||
			derefStr(before.UserTeacherWhatsappURL) != derefStr(m.UserTeacherWhatsappURL) ||
			derefStr(before.UserTeacherTitlePrefix) != derefStr(m.UserTeacherTitlePrefix) ||
			derefStr(before.UserTeacherTitleSuffix) != derefStr(m.UserTeacherTitleSuffix)

	if changedSnapshot {
		set := map[string]any{
			"masjid_teacher_name_user_snapshot":         m.UserTeacherName,
			"masjid_teacher_avatar_url_user_snapshot":   m.UserTeacherAvatarURL,
			"masjid_teacher_whatsapp_url_user_snapshot": m.UserTeacherWhatsappURL,
			"masjid_teacher_title_prefix_user_snapshot": m.UserTeacherTitlePrefix,
			"masjid_teacher_title_suffix_user_snapshot": m.UserTeacherTitleSuffix,
			"masjid_teacher_updated_at":                 time.Now(),
		}
		if err := uc.DB.Model(&mtModel.MasjidTeacherModel{}).
			Where("masjid_teacher_user_teacher_id = ? AND masjid_teacher_deleted_at IS NULL", m.UserTeacherID).
			Updates(set).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError,
				"Profil tersimpan, tapi gagal sync snapshot pengajar di masjid")
		}
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": userdto.ToUserTeacherResponse(*m),
	})
}

// ========================= PATCH WRAPPERS =========================

func (uc *UserTeacherController) PatchMe(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	var m model.UserTeacherModel
	if err := uc.DB.First(&m, "user_teacher_user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil pengajar tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return uc.applyPatch(c, &m, false)
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
