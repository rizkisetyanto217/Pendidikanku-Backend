// file: internals/features/lembaga/masjid_teachers/controller/masjid_teacher_join_controller.go
package controller

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	yDTO "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/dto"
	yModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"
)

// ===== ganti struct row-nya biar cocok dgn tabel masjids =====
type teacherJoinCodeRow struct {
	MasjidID  uuid.UUID
	CodeHash  string
	SetAt     *time.Time
	IsActive  bool
	DeletedAt *time.Time
}

// ===== ganti fungsi lookup kode: dari masjids, bukan masjid_teacher_join_codes =====
func getMasjidIDFromTeacherCode(ctx context.Context, db *gorm.DB, code string) (uuid.UUID, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Code wajib diisi")
	}

	var rows []teacherJoinCodeRow
	// Ambil kandidat yang:
	// - punya hash
	// - aktif & tidak terhapus
	// (opsional: tambahkan policy kadaluarsa kalau ada)
	if err := db.WithContext(ctx).Raw(`
		SELECT
			masjid_id                       AS masjid_id,
			masjid_teacher_code_hash        AS code_hash,
			masjid_teacher_code_set_at      AS set_at,
			masjid_is_active                AS is_active,
			masjid_deleted_at               AS deleted_at
		FROM masjids
		WHERE masjid_deleted_at IS NULL
		  AND masjid_is_active = TRUE
		  AND masjid_teacher_code_hash IS NOT NULL
		ORDER BY masjid_teacher_code_set_at DESC NULLS LAST, masjid_created_at DESC
		LIMIT 2000
	`).Scan(&rows).Error; err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi kode")
	}

	// Bandingkan bcrypt
	for _, r := range rows {
		if r.DeletedAt != nil || !r.IsActive || strings.TrimSpace(r.CodeHash) == "" {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(strings.TrimSpace(r.CodeHash)), []byte(code)) == nil {
			return r.MasjidID, nil
		}
	}

	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Kode guru salah atau sudah kadaluarsa")
}

/*
POST /api/u/:masjid_id/join-teacher
Body: { "code": "...." }
Syarat: user login & sudah punya user_teacher (profil guru)
*/
// Handler: masjid_id diambil dari code
func (ctrl *MasjidTeacherController) JoinAsTeacherWithCode(c *fiber.Ctx) error {
	// user dari token
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	// body
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Code wajib diisi")
	}

	// ✅ masjid_id dari code (validasi + cek expiry/revoked)
	masjidID, err := getMasjidIDFromTeacherCode(c.Context(), ctrl.DB, strings.TrimSpace(body.Code))
	if err != nil {
		// err sudah user-friendly
		return err
	}

	var created yModel.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// ambil user_teacher_id milik user
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

		// cek duplikat alive
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

		// snapshot dari user_teachers
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
			return fiber.NewError(fiber.StatusInternalServerError, "Profil guru tidak valid (nama kosong)")
		}

		// insert record + isi snapshot
		rec := &yModel.MasjidTeacherModel{
			MasjidTeacherMasjidID:      masjidID,
			MasjidTeacherUserTeacherID: userTeacherID,

			MasjidTeacherIsActive:  true,
			MasjidTeacherIsPublic:  true,
			MasjidTeacherCreatedAt: time.Now(),
			MasjidTeacherUpdatedAt: time.Now(),

			MasjidTeacherUserTeacherNameSnapshot:        sptr(ut.Name),
			MasjidTeacherUserTeacherAvatarURLSnapshot:   ut.AvatarURL,
			MasjidTeacherUserTeacherWhatsappURLSnapshot: ut.WhatsappURL,
			MasjidTeacherUserTeacherTitlePrefixSnapshot: ut.TitlePrefix,
			MasjidTeacherUserTeacherTitleSuffixSnapshot: ut.TitleSuffix,
		}
		if err := tx.Create(rec).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mendaftarkan sebagai pengajar")
		}
		created = *rec

		// statistik (best-effort)
		if err := ctrl.Stats.EnsureForMasjid(tx, masjidID); err == nil {
			_ = ctrl.Stats.IncActiveTeachers(tx, masjidID, +1)
		}

		// grant role 'teacher'
		if err := grantTeacherRole(tx, userID, masjidID); err != nil {
			log.Printf("[WARN] grant teacher role failed: %v", err)
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
