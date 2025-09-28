// file: internals/features/lembaga/masjid_teachers/controller/masjid_teacher_join_controller.go
package controller

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	yDTO "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/dto"     // DTO response masjid_teachers
	yModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model" // model masjid_teachers
)

/*
POST /api/u/:masjid_id/join-teacher
Body: { "code": "...." }
Syarat: user login & sudah punya user_teacher (profil guru)
*/
func (ctrl *MasjidTeacherController) JoinAsTeacherWithCode(c *fiber.Ctx) error {
	// === resolve masjid ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID := mc.ID
	if masjidID == uuid.Nil && mc.Slug != "" {
		if id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug); er == nil {
			masjidID = id
		}
	}
	if masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
	}

	// === ambil user_id dari token ===
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	// === body ===
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Code wajib diisi")
	}

	// === validasi code hash ===
	if !checkTeacherCodeValid(ctrl.DB, masjidID, body.Code) {
		return fiber.NewError(fiber.StatusUnauthorized, "Kode guru salah atau sudah kadaluarsa")
	}

	// === transaction ===
	var created yModel.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// --- ambil user_teacher_id user ini ---
		var userTeacherIDStr string
		if err := tx.Raw(`
			SELECT user_teacher_id::text
			  FROM user_teachers
			 WHERE user_teacher_user_id = ?
			   AND user_teacher_deleted_at IS NULL
			 LIMIT 1
		`, userID).Scan(&userTeacherIDStr).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca profil guru")
		}
		if strings.TrimSpace(userTeacherIDStr) == "" {
			return fiber.NewError(fiber.StatusConflict, "Profil guru (user_teacher) belum dibuat")
		}
		userTeacherID, perr := uuid.Parse(userTeacherIDStr)
		if perr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "user_teacher_id tidak valid")
		}

		// --- cek duplikat ---
		var dup int64
		if err := tx.Model(&yModel.MasjidTeacherModel{}).
			Where(`
				masjid_teacher_masjid_id = ?
				AND masjid_teacher_user_teacher_id = ?
				AND masjid_teacher_deleted_at IS NULL
			`, masjidID, userTeacherID).
			Count(&dup).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi data pengajar")
		}
		if dup > 0 {
			return fiber.NewError(fiber.StatusConflict, "Anda sudah terdaftar sebagai pengajar di masjid ini")
		}

		// --- ambil SNAPSHOT dari user_teachers ---
		var ut struct {
			Name        string
			AvatarURL   *string
			WhatsappURL *string
			TitlePrefix *string
			TitleSuffix *string
		}
		if err := tx.Raw(`
			SELECT
				user_teacher_name           AS name,
				user_teacher_avatar_url     AS avatar_url,
				user_teacher_whatsapp_url   AS whatsapp_url,
				user_teacher_title_prefix   AS title_prefix,
				user_teacher_title_suffix   AS title_suffix
			FROM user_teachers
			WHERE user_teacher_id = ?
			  AND user_teacher_deleted_at IS NULL
			LIMIT 1
		`, userTeacherID).Scan(&ut).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca snapshot profil guru")
		}
		if strings.TrimSpace(ut.Name) == "" {
			// safeguard: seharusnya NOT NULL di schema
			return fiber.NewError(fiber.StatusInternalServerError, "Profil guru tidak valid (nama kosong)")
		}

		// --- insert record + isi snapshot ---
		rec := &yModel.MasjidTeacherModel{
			MasjidTeacherMasjidID:      masjidID,
			MasjidTeacherUserTeacherID: userTeacherID,

			MasjidTeacherIsActive:  true,
			MasjidTeacherIsPublic:  true, // default visible; adjust if kebijakan lain
			MasjidTeacherCreatedAt: time.Now(),
			MasjidTeacherUpdatedAt: time.Now(),

			// SNAPSHOT (→ dari user_teachers)
			MasjidTeacherNameUserSnapshot:        sptr(ut.Name),
			MasjidTeacherAvatarURLUserSnapshot:   ut.AvatarURL,
			MasjidTeacherWhatsappURLUserSnapshot: ut.WhatsappURL,
			MasjidTeacherTitlePrefixUserSnapshot: ut.TitlePrefix,
			MasjidTeacherTitleSuffixUserSnapshot: ut.TitleSuffix,
		}
		if err := tx.Create(rec).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mendaftarkan sebagai pengajar")
		}
		created = *rec

		// --- update stats (best-effort) ---
		if err := ctrl.Stats.EnsureForMasjid(tx, masjidID); err == nil {
			_ = ctrl.Stats.IncActiveTeachers(tx, masjidID, +1)
		}
		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonCreated(c, "Berhasil bergabung sebagai pengajar", yDTO.NewMasjidTeacherResponse(&created))
}

/* ===================== Helpers ===================== */

// sptr: trim → nil jika kosong (rapikan JSON)
func sptr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// checkTeacherCodeValid → bandingkan hash dari code dengan masjid.masjid_teacher_code_hash
func checkTeacherCodeValid(db *gorm.DB, masjidID uuid.UUID, plain string) bool {
	var hashStr string
	if err := db.Raw(`
		SELECT masjid_teacher_code_hash
		  FROM masjids
		 WHERE masjid_id = ?
	`, masjidID).Scan(&hashStr).Error; err != nil {
		return false
	}
	if strings.TrimSpace(hashStr) == "" {
		return false
	}
	return VerifyCodeHash(plain, []byte(hashStr))
}

// HashCode menghasilkan hash dari kode plain
func HashCode(plain string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
}

// VerifyCodeHash membandingkan plain code dengan hash yang tersimpan
func VerifyCodeHash(plain string, hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	err := bcrypt.CompareHashAndPassword(hash, []byte(plain))
	return err == nil
}
