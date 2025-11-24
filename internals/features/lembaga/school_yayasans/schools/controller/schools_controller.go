package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	schoolDto "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/dto"
	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

	classModel "madinahsalam_backend/internals/features/school/classes/classes/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SchoolController struct {
	DB       *gorm.DB
	Validate *validator.Validate
	OSS      *helperOSS.OSSService
}

func NewSchoolController(db *gorm.DB, v *validator.Validate, oss *helperOSS.OSSService) *SchoolController {
	return &SchoolController{DB: db, Validate: v, OSS: oss}
}

// ========== helpers lokal ==========
func parseSchoolID(c *fiber.Ctx) (uuid.UUID, error) {
	s := strings.TrimSpace(c.Params("school_id")) // ⬅️ was: "id"
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id pada path tidak valid")
	}
	return id, nil
}

// Cek scope menggunakan key hasil ekstrak dari public URL
func withinSchoolScope(schoolID uuid.UUID, publicURL string) bool {
	key, err := helperOSS.KeyFromPublicURL(publicURL)
	if err != nil {
		return false
	}
	prefix := "schools/" + schoolID.String() + "/"
	return strings.HasPrefix(key, prefix)
}

// ------- util kecil untuk banding nilai pointer & json -------
func val(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func jsonEqual(a, b datatypes.JSON) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return string(a) == string(b)
}

/* ====== KODE GURU: helper & endpoint ====== */
// POST /api/u/schools/:id/teacher-code/rotate
func (mc *SchoolController) GetTeacherCode(c *fiber.Ctx) error {
	raw := strings.TrimSpace(c.Params("school_id")) // ⬅️ was: "id"
	schoolID, err := uuid.Parse(raw)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id pada path tidak valid")
	}

	gotID, aerr := helperAuth.EnsureSchoolAccessDKM(c, helperAuth.SchoolContext{ID: schoolID})
	if aerr != nil {
		return helper.JsonError(c, aerr.(*fiber.Error).Code, aerr.Error())
	}
	if gotID != schoolID {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	var m schoolModel.SchoolModel
	if err := mc.DB.First(&m, "school_id = ?", schoolID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	plain, hash, setAt, err := makeTeacherCodeFromSlug(m.SchoolSlug)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat teacher code")
	}

	m.SchoolTeacherCodeHash = hash
	m.SchoolTeacherCodeSetAt = &setAt
	m.SchoolUpdatedAt = time.Now()

	if err := mc.DB.Model(&m).Updates(map[string]any{
		"school_teacher_code_hash":   m.SchoolTeacherCodeHash,
		"school_teacher_code_set_at": m.SchoolTeacherCodeSetAt,
		"school_updated_at":          m.SchoolUpdatedAt,
	}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan kode guru")
	}

	return helper.JsonOK(c, "Kode guru didapatkan", fiber.Map{
		"teacher_code": plain,
		"set_at":       setAt,
	})
}

// PATCH /api/u/schools/:id/teacher-code
// Body: { "code": "<plain or 2-char suffix>" }

const teacherCodeMaxLen = 128 // batasi panjang biar aman (boleh ubah)

type patchTeacherCodeJSON struct {
	Code string `json:"code"`
}

// PATCH /api/u/schools/:id/teacher-code
// Body:
//   - JSON:      { "code": "apa saja (bebas simbol)" }
//   - Form:      code=apa%20saja
//   - Multipart: code=apa%20saja
func (mc *SchoolController) PatchTeacherCode(c *fiber.Ctx) error {
	raw := strings.TrimSpace(c.Params("school_id")) // ⬅️ was: "id"
	schoolID, err := uuid.Parse(raw)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id pada path tidak valid")
	}

	gotID, aerr := helperAuth.EnsureSchoolAccessDKM(c, helperAuth.SchoolContext{ID: schoolID})
	if aerr != nil {
		return helper.JsonError(c, aerr.(*fiber.Error).Code, aerr.Error())
	}
	if gotID != schoolID {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	// baca body (json/form/raw) sama seperti sebelumnya…
	var code string
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	switch {
	case strings.HasPrefix(ct, "application/json"):
		var p patchTeacherCodeJSON
		if err := json.Unmarshal(c.Body(), &p); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "JSON tidak valid")
		}
		code = p.Code
	case strings.HasPrefix(ct, "application/x-www-form-urlencoded"),
		strings.HasPrefix(ct, "multipart/form-data"):
		code = c.FormValue("code")
	default:
		code = string(c.Body())
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "code wajib diisi")
	}
	if len(code) > teacherCodeMaxLen {
		return helper.JsonError(c, fiber.StatusBadRequest, "code terlalu panjang")
	}

	var m schoolModel.SchoolModel
	if err := mc.DB.First(&m, "school_id = ?", schoolID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	now := time.Now()
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat hash code")
	}
	m.SchoolTeacherCodeHash = hash
	m.SchoolTeacherCodeSetAt = &now
	m.SchoolUpdatedAt = now

	if err := mc.DB.Model(&m).Updates(map[string]any{
		"school_teacher_code_hash":   m.SchoolTeacherCodeHash,
		"school_teacher_code_set_at": m.SchoolTeacherCodeSetAt,
		"school_updated_at":          m.SchoolUpdatedAt,
	}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan kode guru")
	}

	return helper.JsonOK(c, "Kode guru diperbarui", fiber.Map{
		"teacher_code": code,
		"set_at":       now,
	})
}

/* ====== PATCH (existing) ====== */
// PATCH /api/schools/:id
func (mc *SchoolController) Patch(c *fiber.Ctx) error {
	id, err := parseSchoolID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ===== AUTH (DKM) — pin ke path id =====
	schoolID, aerr := helperAuth.EnsureSchoolAccessDKM(c, helperAuth.SchoolContext{ID: id})
	if aerr != nil {
		return helper.JsonError(c, aerr.(*fiber.Error).Code, aerr.Error())
	}
	if schoolID != id {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: school tidak sesuai")
	}

	// Ambil row existing
	var m schoolModel.SchoolModel
	if err := mc.DB.First(&m, "school_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil school")
	}
	before := m // snapshot untuk deteksi delta

	// --- state ---
	var u schoolDto.SchoolUpdateReq
	now := time.Now()
	changedMedia := false
	retainUntil := now.Add(helperOSS.GetRetentionDuration())

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))

	// [A] multipart/form-data
	if strings.HasPrefix(ct, "multipart/form-data") {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := json.Unmarshal([]byte(s), &u); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "payload JSON tidak valid")
			}
		} else {
			_ = c.BodyParser(&u) // best-effort untuk field sederhana
		}

		// OSS service
		svc := mc.OSS
		if svc == nil {
			tmp, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
			}
			svc = tmp
		}

		// -- icon --
		if fh, err := c.FormFile("icon"); err == nil && fh != nil {
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, id, "icon", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (icon)")
			}
			if m.SchoolIconURL != nil && *m.SchoolIconURL != "" {
				m.SchoolIconURLOld = m.SchoolIconURL
				m.SchoolIconObjectKeyOld = m.SchoolIconObjectKey
				m.SchoolIconDeletePendingUntil = &retainUntil
			}
			m.SchoolIconURL = &url
			m.SchoolIconObjectKey = &key
			changedMedia = true
		}

		// -- logo --
		if fh, err := c.FormFile("logo"); err == nil && fh != nil {
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, id, "logo", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (logo)")
			}
			if m.SchoolLogoURL != nil && *m.SchoolLogoURL != "" {
				m.SchoolLogoURLOld = m.SchoolLogoURL
				m.SchoolLogoObjectKeyOld = m.SchoolLogoObjectKey
				m.SchoolLogoDeletePendingUntil = &retainUntil
			}
			m.SchoolLogoURL = &url
			m.SchoolLogoObjectKey = &key
			changedMedia = true
		}

		// -- background --
		if fh, err := c.FormFile("background"); err == nil && fh != nil {
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, id, "background", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (background)")
			}
			if m.SchoolBackgroundURL != nil && *m.SchoolBackgroundURL != "" {
				m.SchoolBackgroundURLOld = m.SchoolBackgroundURL
				m.SchoolBackgroundObjectKeyOld = m.SchoolBackgroundObjectKey
				m.SchoolBackgroundDeletePendingUntil = &retainUntil
			}
			m.SchoolBackgroundURL = &url
			m.SchoolBackgroundObjectKey = &key
			changedMedia = true
		}

	} else {
		// [B] JSON biasa
		if err := c.BodyParser(&u); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Terapkan patch field non-file (current-only)
	schoolDto.ApplyUpdate(&m, &u)
	m.SchoolUpdatedAt = now

	// === regenerate slug kalau nama berubah ===
	if before.SchoolName != m.SchoolName {
		base := helper.SuggestSlugFromName(m.SchoolName)
		// Hindari count kena row sendiri
		scopeFn := func(q *gorm.DB) *gorm.DB {
			return q.Where("school_id <> ?", id)
		}
		uniq, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			mc.DB,
			"schools",     // table
			"school_slug", // kolom slug
			base,
			scopeFn, // per-tenant? sesuaikan di sini bila perlu
			100,
		)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}
		m.SchoolSlug = uniq
	}

	// Bangun updates map hanya kolom yang berubah
	updates := map[string]any{"school_updated_at": m.SchoolUpdatedAt}

	// inti identitas
	if before.SchoolName != m.SchoolName {
		updates["school_name"] = m.SchoolName
	}
	if val(before.SchoolBioShort) != val(m.SchoolBioShort) {
		updates["school_bio_short"] = m.SchoolBioShort
	}
	if val(before.SchoolLocation) != val(m.SchoolLocation) {
		updates["school_location"] = m.SchoolLocation
	}
	if val(before.SchoolCity) != val(m.SchoolCity) {
		updates["school_city"] = m.SchoolCity
	}
	if val(before.SchoolDomain) != val(m.SchoolDomain) {
		updates["school_domain"] = m.SchoolDomain
	}
	if before.SchoolSlug != m.SchoolSlug {
		updates["school_slug"] = m.SchoolSlug
	}

	// verifikasi & flags
	if before.SchoolIsActive != m.SchoolIsActive {
		updates["school_is_active"] = m.SchoolIsActive
	}
	if string(before.SchoolVerificationStatus) != string(m.SchoolVerificationStatus) {
		updates["school_verification_status"] = m.SchoolVerificationStatus
	}
	if val(before.SchoolVerificationNotes) != val(m.SchoolVerificationNotes) {
		updates["school_verification_notes"] = m.SchoolVerificationNotes
	}
	if val(before.SchoolContactPersonName) != val(m.SchoolContactPersonName) {
		updates["school_contact_person_name"] = m.SchoolContactPersonName
	}
	if val(before.SchoolContactPersonPhone) != val(m.SchoolContactPersonPhone) {
		updates["school_contact_person_phone"] = m.SchoolContactPersonPhone
	}
	if before.SchoolIsIslamicSchool != m.SchoolIsIslamicSchool {
		updates["school_is_islamic_school"] = m.SchoolIsIslamicSchool
	}
	if string(before.SchoolTenantProfile) != string(m.SchoolTenantProfile) {
		updates["school_tenant_profile"] = m.SchoolTenantProfile
	}
	if !jsonEqual(before.SchoolLevels, m.SchoolLevels) {
		updates["school_levels"] = m.SchoolLevels
	}

	// media current (ICON + LOGO + BACKGROUND)
	if val(before.SchoolIconURL) != val(m.SchoolIconURL) {
		updates["school_icon_url"] = m.SchoolIconURL
	}
	if val(before.SchoolIconObjectKey) != val(m.SchoolIconObjectKey) {
		updates["school_icon_object_key"] = m.SchoolIconObjectKey
	}
	if val(before.SchoolLogoURL) != val(m.SchoolLogoURL) {
		updates["school_logo_url"] = m.SchoolLogoURL
	}
	if val(before.SchoolLogoObjectKey) != val(m.SchoolLogoObjectKey) {
		updates["school_logo_object_key"] = m.SchoolLogoObjectKey
	}
	if val(before.SchoolBackgroundURL) != val(m.SchoolBackgroundURL) {
		updates["school_background_url"] = m.SchoolBackgroundURL
	}
	if val(before.SchoolBackgroundObjectKey) != val(m.SchoolBackgroundObjectKey) {
		updates["school_background_object_key"] = m.SchoolBackgroundObjectKey
	}

	// media shadow (2-slot) — hanya untuk yang benar2 berubah
	if changedMedia {
		if val(before.SchoolIconURL) != val(m.SchoolIconURL) {
			updates["school_icon_url_old"] = m.SchoolIconURLOld
			updates["school_icon_object_key_old"] = m.SchoolIconObjectKeyOld
			updates["school_icon_delete_pending_until"] = m.SchoolIconDeletePendingUntil
		}
		if val(before.SchoolLogoURL) != val(m.SchoolLogoURL) {
			updates["school_logo_url_old"] = m.SchoolLogoURLOld
			updates["school_logo_object_key_old"] = m.SchoolLogoObjectKeyOld
			updates["school_logo_delete_pending_until"] = m.SchoolLogoDeletePendingUntil
		}
		if val(before.SchoolBackgroundURL) != val(m.SchoolBackgroundURL) {
			updates["school_background_url_old"] = m.SchoolBackgroundURLOld
			updates["school_background_object_key_old"] = m.SchoolBackgroundObjectKeyOld
			updates["school_background_delete_pending_until"] = m.SchoolBackgroundDeletePendingUntil
		}
	}

	if len(updates) == 1 { // cuma updated_at
		return helper.JsonOK(c, "Tidak ada perubahan", fiber.Map{
			"item": schoolDto.FromModel(&m),
		})
	}

	if err := mc.DB.Model(&m).Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": schoolDto.FromModel(&m),
	})
}

// 🟢 DELETE SCHOOL
func (mc *SchoolController) Delete(c *fiber.Ctx) error {
	id, err := parseSchoolID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 AUTH: minimal DKM di school tsb (atau Owner kalau mau lebih ketat)
	schoolID, aerr := helperAuth.EnsureSchoolAccessDKM(c, helperAuth.SchoolContext{ID: id})
	if aerr != nil {
		if fe, ok := aerr.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, aerr.Error())
	}
	if schoolID != id {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: school tidak sesuai")
	}

	// Pastikan school exist & belum soft-deleted
	var sch schoolModel.SchoolModel
	if err := mc.DB.
		Where("school_id = ? AND school_deleted_at IS NULL", id).
		First(&sch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan / sudah dihapus")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
	}

	// =======================
	// GUARD RELASI
	// =======================

	// pattern: hitung data aktif di tabel2 inti → kalau >0, tolak
	type dep struct {
		name  string
		count int64
	}

	deps := []dep{
		// school_profiles
		func() dep {
			var n int64
			_ = mc.DB.Model(&schoolModel.SchoolProfileModel{}).
				Where("school_profile_school_id = ? AND school_profile_deleted_at IS NULL", id).
				Count(&n).Error
			return dep{name: "profil sekolah", count: n}
		}(),
		// classes
		func() dep {
			var n int64
			_ = mc.DB.Model(&classModel.ClassModel{}).
				Where("class_school_id = ? AND class_deleted_at IS NULL", id).
				Count(&n).Error
			return dep{name: "kelas (classes)", count: n}
		}(),
	}

	for _, d := range deps {
		if d.count > 0 {
			return helper.JsonError(
				c,
				fiber.StatusBadRequest,
				fmt.Sprintf("Tidak dapat menghapus school karena masih ada %s terkait (%d record).", d.name, d.count),
			)
		}
	}

	// =======================
	// Soft delete school
	// =======================

	if err := mc.DB.Delete(&sch).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus school")
	}

	return helper.JsonDeleted(c, "School berhasil dihapus", fiber.Map{
		"school_id": sch.SchoolID,
	})
}
