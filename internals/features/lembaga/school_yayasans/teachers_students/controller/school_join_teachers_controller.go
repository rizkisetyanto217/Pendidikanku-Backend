package controller

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

	teacherDTO "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	teacherModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
)

// ===== ganti struct row-nya biar cocok dgn tabel schools =====
type teacherJoinCodeRow struct {
	SchoolID  uuid.UUID
	CodeHash  string
	SetAt     *time.Time
	IsActive  bool
	DeletedAt *time.Time
}

// ===== kode diambil dari kolom di tabel schools =====
func getSchoolIDFromTeacherCode(ctx context.Context, db *gorm.DB, code string) (uuid.UUID, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Code wajib diisi")
	}

	var rows []teacherJoinCodeRow
	if err := db.WithContext(ctx).Raw(`
		SELECT
			school_id                       AS school_id,
			school_teacher_code_hash        AS code_hash,
			school_teacher_code_set_at      AS set_at,
			school_is_active                AS is_active,
			school_deleted_at               AS deleted_at
		FROM schools
		WHERE school_deleted_at IS NULL
		  AND school_is_active = TRUE
		  AND school_teacher_code_hash IS NOT NULL
		ORDER BY school_teacher_code_set_at DESC NULLS LAST, school_created_at DESC
		LIMIT 2000
	`).Scan(&rows).Error; err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi kode")
	}

	for _, r := range rows {
		if r.DeletedAt != nil || !r.IsActive || strings.TrimSpace(r.CodeHash) == "" {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(strings.TrimSpace(r.CodeHash)), []byte(code)) == nil {
			return r.SchoolID, nil
		}
	}

	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Kode guru salah atau sudah kadaluarsa")
}

/*
POST /api/u/join-teacher
Body: { "code": "...." }

- user_id diambil dari token (wajib login)
- school_id diambil dari kode guru (kolom school_teacher_code_hash di tabel schools)
- user harus sudah punya user_teacher (profil guru)
*/
func (ctrl *SchoolTeacherController) JoinAsTeacherWithCode(c *fiber.Ctx) error {
	// user dari token
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	// body
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Code wajib diisi")
	}
	codePlain := strings.TrimSpace(body.Code)
	if codePlain == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Code wajib diisi")
	}

	log.Printf("[JOIN-TEACHER] incoming request user_id=%s code=%s", userID.String(), codePlain)

	// ✅ school_id dari code (validasi + cek expiry/revoked)
	schoolID, err := getSchoolIDFromTeacherCode(c.Context(), ctrl.DB, codePlain)
	if err != nil {
		// err sudah user-friendly (fiber.Error)
		log.Printf("[JOIN-TEACHER] code validation failed user_id=%s code=%s err=%v", userID.String(), codePlain, err)
		return err
	}
	log.Printf("[JOIN-TEACHER] code valid user_id=%s school_id=%s", userID.String(), schoolID.String())

	var created teacherModel.SchoolTeacherModel
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
			log.Printf("[JOIN-TEACHER] read user_teacher failed user_id=%s err=%v", userID.String(), err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca profil guru")
		}
		if strings.TrimSpace(userTeacherIDStr) == "" {
			log.Printf("[JOIN-TEACHER] user_teacher not found user_id=%s", userID.String())
			return fiber.NewError(fiber.StatusConflict, "Profil guru (user_teacher) belum dibuat")
		}
		userTeacherID, perr := uuid.Parse(userTeacherIDStr)
		if perr != nil {
			log.Printf("[JOIN-TEACHER] invalid user_teacher_id user_teacher_id_str=%s err=%v", userTeacherIDStr, perr)
			return fiber.NewError(fiber.StatusInternalServerError, "user_teacher_id tidak valid")
		}

		// cek duplikat alive
		var dup int64
		if err := tx.Model(&teacherModel.SchoolTeacherModel{}).
			Where(`
				school_teacher_school_id = ?
				AND school_teacher_user_teacher_id = ?
				AND school_teacher_deleted_at IS NULL
			`, schoolID, userTeacherID).
			Count(&dup).Error; err != nil {
			log.Printf("[JOIN-TEACHER] dup check failed user_teacher_id=%s err=%v", userTeacherID.String(), err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi data pengajar")
		}
		if dup > 0 {
			log.Printf("[JOIN-TEACHER] already joined school_id=%s user_teacher_id=%s", schoolID.String(), userTeacherID.String())
			return fiber.NewError(fiber.StatusConflict, "Anda sudah terdaftar sebagai pengajar di school ini")
		}

		// 🆕 Generate TEACHER CODE per sekolah (school_number + tahun + auto increment per school_id)
		plainTeacherCode, _, err := GenerateTeacherCodeForSchool(c.Context(), tx, schoolID)
		if err != nil {
			log.Printf("[JOIN-TEACHER] generate teacher code failed school_id=%s err=%v", schoolID.String(), err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat kode guru")
		}
		log.Printf("[JOIN-TEACHER] generated teacher_code=%s school_id=%s user_teacher_id=%s", plainTeacherCode, schoolID.String(), userTeacherID.String())

		// snapshot dari user_teachers (+ gender dari user_profiles)
		var ut struct {
			Name        string  `gorm:"column:name"`
			AvatarURL   *string `gorm:"column:avatar_url"`
			WhatsappURL *string `gorm:"column:whatsapp_url"`
			TitlePrefix *string `gorm:"column:title_prefix"`
			TitleSuffix *string `gorm:"column:title_suffix"`
			Gender      *string `gorm:"column:gender"`
		}
		if err := tx.Raw(`
			SELECT
				ut.user_teacher_full_name_cache      AS name,
				ut.user_teacher_avatar_url         AS avatar_url,
				ut.user_teacher_whatsapp_url       AS whatsapp_url,
				ut.user_teacher_title_prefix       AS title_prefix,
				ut.user_teacher_title_suffix       AS title_suffix,
				up.user_profile_gender             AS gender
			FROM user_teachers ut
			LEFT JOIN user_profiles up
			  ON up.user_profile_user_id = ut.user_teacher_user_id
			 AND up.user_profile_deleted_at IS NULL
			WHERE ut.user_teacher_id = ?
			  AND ut.user_teacher_deleted_at IS NULL
			LIMIT 1
		`, userTeacherID).Scan(&ut).Error; err != nil {
			log.Printf("[JOIN-TEACHER] read snapshot failed user_teacher_id=%s err=%v", userTeacherID.String(), err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca snapshot profil guru")
		}
		if strings.TrimSpace(ut.Name) == "" {
			log.Printf("[JOIN-TEACHER] invalid snapshot: empty name user_teacher_id=%s", userTeacherID.String())
			return fiber.NewError(fiber.StatusInternalServerError, "Profil guru tidak valid (nama kosong)")
		}

		// 👇 generate base slug & ensure unik per sekolah
		baseSlug := helper.SuggestSlugFromName(ut.Name)
		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			"school_teachers",
			"school_teacher_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("school_teacher_school_id = ?", schoolID)
			},
			100, // max length
		)
		if err != nil {
			log.Printf("[JOIN-TEACHER] ensure unique slug failed school_id=%s user_teacher_id=%s err=%v",
				schoolID.String(), userTeacherID.String(), err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug guru")
		}

		now := time.Now()

		// insert record + isi snapshot + SIMPAN teacher code di kolom school_teacher_code
		rec := &teacherModel.SchoolTeacherModel{
			SchoolTeacherSchoolID:      schoolID,
			SchoolTeacherUserTeacherID: userTeacherID,

			// simpan TEACHER CODE (school_number + tahun + auto increment per sekolah)
			SchoolTeacherCode: sptr(plainTeacherCode),

			// NEW: joined_at + slug
			SchoolTeacherJoinedAt: &now,
			SchoolTeacherSlug:     sptr(uniqueSlug),

			SchoolTeacherIsActive:  true,
			SchoolTeacherIsPublic:  true,
			SchoolTeacherCreatedAt: now,
			SchoolTeacherUpdatedAt: now,

			SchoolTeacherUserTeacherFullNameCache:        sptr(ut.Name),
			SchoolTeacherUserTeacherAvatarURLCache:   ut.AvatarURL,
			SchoolTeacherUserTeacherWhatsappURLCache: ut.WhatsappURL,
			SchoolTeacherUserTeacherTitlePrefixCache: ut.TitlePrefix,
			SchoolTeacherUserTeacherTitleSuffixCache: ut.TitleSuffix,
			SchoolTeacherUserTeacherGenderCache:      ut.Gender,
			// JSONB sections & csst akan ikut default DB ('[]') kalau tidak di-set di sini
		}

		log.Printf("[JOIN-TEACHER] creating school_teacher row=%+v", rec)

		if err := tx.Create(rec).Error; err != nil {
			log.Printf("[JOIN-TEACHER] create school_teacher failed err=%v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mendaftarkan sebagai pengajar")
		}
		created = *rec

		log.Printf(
			"[JOIN-TEACHER] created school_teacher_id=%s school_id=%s user_teacher_id=%s teacher_code=%v slug=%v",
			created.SchoolTeacherID.String(),
			created.SchoolTeacherSchoolID.String(),
			created.SchoolTeacherUserTeacherID.String(),
			created.SchoolTeacherCode,
			created.SchoolTeacherSlug,
		)

		// statistik (best-effort)
		if err := ctrl.Stats.EnsureForSchool(tx, schoolID); err == nil {
			_ = ctrl.Stats.IncActiveTeachers(tx, schoolID, +1)
		} else {
			log.Printf("[JOIN-TEACHER] stats EnsureForSchool failed school_id=%s err=%v", schoolID.String(), err)
		}

		// grant role 'teacher'
		if err := grantTeacherRole(tx, userID, schoolID); err != nil {
			log.Printf("[JOIN-TEACHER] grant teacher role failed user_id=%s school_id=%s err=%v", userID.String(), schoolID.String(), err)
		}

		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonCreated(c, "Berhasil bergabung sebagai pengajar", teacherDTO.NewSchoolTeacherResponse(&created))
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

// GenerateTeacherCodeForSchool
// Format: {school_number}{tahun_masuk}{auto_increment_per_sekolah}
//
// Contoh:
//
//	school_number = 12
//	tahun = 2025
//	guru aktif saat ini = 7
//	→ nextIdx = 8
//	→ code = "001220250008"
//
// Tiap sekolah punya sequence-nya sendiri,
// walaupun semua data di tabel school_teachers yang sama.
func GenerateTeacherCodeForSchool(ctx context.Context, db *gorm.DB, schoolID uuid.UUID) (plainCode string, hash []byte, err error) {
	if schoolID == uuid.Nil {
		return "", nil, fmt.Errorf("school_id tidak valid")
	}

	// 1) Ambil school_number dari tabel schools
	var sc struct {
		Number int64 `gorm:"column:school_number"`
	}
	if err := db.WithContext(ctx).
		Model(&schoolModel.SchoolModel{}).
		Select("school_number").
		Where("school_id = ? AND school_deleted_at IS NULL", schoolID).
		Scan(&sc).Error; err != nil {
		return "", nil, fmt.Errorf("gagal membaca school_number: %w", err)
	}
	if sc.Number < 0 {
		sc.Number = 0
	}

	// 2) Hitung guru AKTIF di sekolah ini (per sekolah)
	var teacherCount int64
	if err := db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM school_teachers
		WHERE school_teacher_school_id = ?
		  AND school_teacher_deleted_at IS NULL
		  AND school_teacher_is_active = TRUE
	`, schoolID).Scan(&teacherCount).Error; err != nil {
		return "", nil, fmt.Errorf("gagal menghitung guru di sekolah: %w", err)
	}

	year := time.Now().Year()
	nextIdx := teacherCount + 1

	// 3) Bentuk kode:
	//    - school_number: 4 digit
	//    - tahun: 4 digit
	//    - auto increment: 4 digit
	//    TANPA strip → contoh: 001220250008
	plainCode = fmt.Sprintf("%04d%04d%04d", sc.Number, year, nextIdx)

	// 4) Hash pakai helper (kalau nanti mau dipakai juga sebagai kode rahasia)
	hash, err = HashCode(plainCode)
	if err != nil {
		return "", nil, fmt.Errorf("gagal hash teacher code: %w", err)
	}

	return plainCode, hash, nil
}
