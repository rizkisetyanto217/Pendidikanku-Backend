// file: internals/features/school/classes/class_sections/controller/user_class_section_join_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	dto "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	model "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	UserProfileCache "madinahsalam_backend/internals/features/users/users/service"
)

/* =========================
   Utils
========================= */

func Ptr[T any](v T) *T            { return &v }
func forUpdate() clause.Expression { return clause.Locking{Strength: "UPDATE"} }

func verifyJoinCode(stored []byte, code string) bool {
	if len(stored) == 0 || strings.TrimSpace(code) == "" {
		return false
	}
	// bcrypt variants
	if strings.HasPrefix(string(stored), "$2a$") ||
		strings.HasPrefix(string(stored), "$2b$") ||
		strings.HasPrefix(string(stored), "$2y$") {
		return bcrypt.CompareHashAndPassword(stored, []byte(code)) == nil
	}
	return false
}

func nzTrim(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

/* =========================
   Lookups
========================= */

// users_profile_id dari user_id
func getUsersProfileID(tx *gorm.DB, userID uuid.UUID) (uuid.UUID, error) {
	type row struct {
		ID uuid.UUID `gorm:"column:user_profile_id"`
	}
	var r row
	err := tx.Table("user_profiles").
		Select("user_profile_id").
		Where("user_profile_user_id = ? AND user_profile_deleted_at IS NULL", userID).
		First(&r).Error
	if err != nil {
		return uuid.Nil, err
	}
	if r.ID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("user_profile_id kosong")
	}
	return r.ID, nil
}

// Cache school (ringkas)
func getSchoolCache(tx *gorm.DB, schoolID uuid.UUID) (name, slug, logo, icon, bg *string, err error) {
	var row struct {
		Name *string `gorm:"column:school_name"`
		Slug *string `gorm:"column:school_slug"`
		Logo *string `gorm:"column:school_logo_url"`
		Icon *string `gorm:"column:school_icon_url"`
		Bg   *string `gorm:"column:school_background_url"`
	}
	err = tx.Table("schools").
		Select("school_name, school_slug, school_logo_url, school_icon_url, school_background_url").
		Where("school_id = ? AND school_deleted_at IS NULL", schoolID).
		Take(&row).Error
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return row.Name, row.Slug, row.Logo, row.Icon, row.Bg, nil
}

/* =========================
   Student upsert + snapshots
========================= */

// Get/create school_students + isi snapshots (profil & school)
func getOrCreateSchoolStudentWithCaches(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	userProfileID uuid.UUID,
	profileSnap *UserProfileCache.UserProfileCache, // boleh nil
) (uuid.UUID, error) {
	// snapshot profil kalau belum ada
	if profileSnap == nil {
		if ps, e := UserProfileCache.BuildUserProfileCacheByProfileID(ctx, tx, userProfileID); e == nil {
			profileSnap = ps
		}
	}
	// snapshot school
	mName, mSlug, mLogo, mIcon, mBg, _ := getSchoolCache(tx, schoolID)

	// ada existing?
	var cur struct {
		ID uuid.UUID `gorm:"column:school_student_id"`
	}
	err := tx.Table("school_students").
		Select("school_student_id").
		Where("school_student_school_id = ? AND school_student_user_profile_id = ? AND school_student_deleted_at IS NULL",
			schoolID, userProfileID).
		Take(&cur).Error

	now := time.Now()

	if err == nil {
		// top-up snapshots (best-effort)
		updates := map[string]any{
			"school_student_updated_at": now,
		}
		if profileSnap != nil {
			if name := strings.TrimSpace(profileSnap.Name); name != "" {
				updates["school_student_user_profile_name_cache"] = name
			}
			if v := nzTrim(profileSnap.AvatarURL); v != nil {
				updates["school_student_user_profile_avatar_url_snapshot"] = *v
			}
			if v := nzTrim(profileSnap.WhatsappURL); v != nil {
				updates["school_student_user_profile_whatsapp_url_snapshot"] = *v
			}
			if v := nzTrim(profileSnap.ParentName); v != nil {
				updates["school_student_user_profile_parent_name_snapshot"] = *v
			}
			if v := nzTrim(profileSnap.ParentWhatsappURL); v != nil {
				updates["school_student_user_profile_parent_whatsapp_url_snapshot"] = *v
			}
			// NEW: gender snapshot di school_students
			if profileSnap.Gender != nil {
				if g := strings.TrimSpace(*profileSnap.Gender); g != "" {
					updates["school_student_user_profile_gender_snapshot"] = g
				}
			}
		}
		if v := nzTrim(mName); v != nil {
			updates["school_student_school_name_snapshot"] = *v
		}
		if v := nzTrim(mSlug); v != nil {
			updates["school_student_school_slug_snapshot"] = *v
		}
		if v := nzTrim(mLogo); v != nil {
			updates["school_student_school_logo_url_snapshot"] = *v
		}
		if v := nzTrim(mIcon); v != nil {
			updates["school_student_school_icon_url_snapshot"] = *v
		}
		if v := nzTrim(mBg); v != nil {
			updates["school_student_school_background_url_snapshot"] = *v
		}
		if e := tx.Table("school_students").Where("school_student_id = ?", cur.ID).Updates(updates).Error; e != nil {
			return uuid.Nil, e
		}
		return cur.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, err
	}

	// create baru (lengkap snapshots)
	newID := uuid.New()
	values := map[string]any{
		"school_student_id":              newID,
		"school_student_school_id":       schoolID,
		"school_student_user_profile_id": userProfileID,
		"school_student_slug":            newID.String(),
		"school_student_status":          "active",
		"school_student_class_sections":  datatypes.JSON([]byte("[]")),
		"school_student_created_at":      now,
		"school_student_updated_at":      now,
	}
	if profileSnap != nil {
		if name := strings.TrimSpace(profileSnap.Name); name != "" {
			values["school_student_user_profile_name_cache"] = name
		}
		if v := nzTrim(profileSnap.AvatarURL); v != nil {
			values["school_student_user_profile_avatar_url_snapshot"] = *v
		}
		if v := nzTrim(profileSnap.WhatsappURL); v != nil {
			values["school_student_user_profile_whatsapp_url_snapshot"] = *v
		}
		if v := nzTrim(profileSnap.ParentName); v != nil {
			values["school_student_user_profile_parent_name_snapshot"] = *v
		}
		if v := nzTrim(profileSnap.ParentWhatsappURL); v != nil {
			values["school_student_user_profile_parent_whatsapp_url_snapshot"] = *v
		}
		// NEW: gender snapshot di create
		if profileSnap.Gender != nil {
			if g := strings.TrimSpace(*profileSnap.Gender); g != "" {
				values["school_student_user_profile_gender_snapshot"] = g
			}
		}
	}
	if v := nzTrim(mName); v != nil {
		values["school_student_school_name_snapshot"] = *v
	}
	if v := nzTrim(mSlug); v != nil {
		values["school_student_school_slug_snapshot"] = *v
	}
	if v := nzTrim(mLogo); v != nil {
		values["school_student_school_logo_url_snapshot"] = *v
	}
	if v := nzTrim(mIcon); v != nil {
		values["school_student_school_icon_url_snapshot"] = *v
	}
	if v := nzTrim(mBg); v != nil {
		values["school_student_school_background_url_snapshot"] = *v
	}

	if err := tx.Table("school_students").Create(values).Error; err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

/* =========================
   Handler
========================= */

// POST /api/a/:school_id/student-class-sections/join     (versi lama: school_id di path)
// POST /api/student-class-sections/join                  (versi baru: auto school dari code)
// Body: { "student_code": "...." }
func (ctl *StudentClassSectionController) JoinByCodeAutoSchool(c *fiber.Ctx) error {
	// --- auth wajib login ---
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	// --- body ---
	var req struct {
		StudentCode string `json:"student_code" form:"student_code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	code := strings.TrimSpace(req.StudentCode)
	if code == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "student_code wajib diisi")
	}

	// --- TX ---
	tx := ctl.DB.WithContext(c.Context()).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// --- 1) Temukan section dari code (tanpa filter school dulu) ---
	var sec model.ClassSectionModel
	found := false

	// (a) plaintext
	if err := tx.
		Clauses(forUpdate()).
		Where(`class_section_deleted_at IS NULL AND class_section_code = ?`, code).
		First(&sec).Error; err == nil {
		found = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari section (by plaintext)")
	}

	// (b) bcrypt hash
	if !found {
		var rows []struct {
			ID   uuid.UUID `gorm:"column:class_section_id"`
			Hash []byte    `gorm:"column:class_section_student_code_hash"`
		}
		if err := tx.Table("class_sections").
			Select("class_section_id, class_section_student_code_hash").
			Where(`
				class_section_deleted_at IS NULL
				AND class_section_student_code_hash IS NOT NULL
				AND octet_length(class_section_student_code_hash) > 0
			`).
			Find(&rows).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari section (by hash)")
		}
		var matchedID uuid.UUID
		for _, r := range rows {
			if verifyJoinCode(r.Hash, code) {
				matchedID = r.ID
				break
			}
		}
		if matchedID == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusUnauthorized, "Kode siswa salah atau sudah tidak berlaku")
		}
		if err := tx.
			Clauses(forUpdate()).
			Where("class_section_id = ? AND class_section_deleted_at IS NULL", matchedID).
			First(&sec).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Section tujuan tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil section")
		}
	}

	// ✅ school_id diturunkan dari section
	schoolID := sec.ClassSectionSchoolID
	if schoolID == uuid.Nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Section tidak memiliki konteks school yang valid")
	}

	// --- 2) Validasi section (awal; guard final di UPDATE atomic) ---
	// Dulu: if !sec.ClassSectionIsActive { ... }
	// Sekarang pakai enum status
	if sec.ClassSectionStatus != model.ClassStatusActive {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Section tidak aktif")
	}

	// --- 3) Pastikan ada school_student + isi snapshots ---
	usersProfileID, err := getUsersProfileID(tx, userID)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
	}
	profileSnap, perr := UserProfileCache.BuildUserProfileCacheByProfileID(c.Context(), tx, usersProfileID)
	if perr != nil && !errors.Is(perr, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot profil")
	}
	schoolStudentID, err := getOrCreateSchoolStudentWithCaches(c.Context(), tx, schoolID, usersProfileID, profileSnap)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek/buat status student")
	}

	// --- 3.5) Ambil kode siswa dari school_students untuk snapshot ---
	var stuRow struct {
		Code *string `gorm:"column:school_student_code"`
	}
	if err := tx.Table("school_students").
		Select("school_student_code").
		Where("school_student_id = ? AND school_student_deleted_at IS NULL", schoolStudentID).
		Take(&stuRow).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil kode siswa")
	}

	// --- 4) Cegah double-join ---
	var exists int64
	if err := tx.Table("student_class_sections").
		Where(`
			student_class_section_school_id = ?
			AND student_class_section_school_student_id = ?
			AND student_class_section_section_id = ?
			AND student_class_section_deleted_at IS NULL
		`, schoolID, schoolStudentID, sec.ClassSectionID).
		Count(&exists).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek keanggotaan")
	}
	if exists > 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Sudah tergabung di section ini")
	}

	// --- 5) Insert enrollment (+ snapshot profil) ---
	now := time.Now()

	// pastikan slug snapshot keisi (NOT NULL di DB)
	slug := strings.TrimSpace(sec.ClassSectionSlug)
	if slug == "" {
		slug = sec.ClassSectionID.String()
	}

	scs := &model.StudentClassSection{
		StudentClassSectionID:               uuid.New(),
		StudentClassSectionSchoolID:         schoolID,
		StudentClassSectionSchoolStudentID:  schoolStudentID,
		StudentClassSectionSectionID:        sec.ClassSectionID,
		StudentClassSectionSectionSlugCache: slug,
		StudentClassSectionStatus:           model.StudentClassSectionActive,
		StudentClassSectionAssignedAt:       now,
		StudentClassSectionCreatedAt:        now,
		StudentClassSectionUpdatedAt:        now,
	}

	if profileSnap != nil {
		if name := strings.TrimSpace(profileSnap.Name); name != "" {
			scs.StudentClassSectionUserProfileNameCache = &name
		}
		scs.StudentClassSectionUserProfileAvatarURLCache = nzTrim(profileSnap.AvatarURL)
		scs.StudentClassSectionUserProfileWhatsappURLCache = nzTrim(profileSnap.WhatsappURL)
		scs.StudentClassSectionUserProfileParentNameCache = nzTrim(profileSnap.ParentName)
		scs.StudentClassSectionUserProfileParentWhatsappURLCache = nzTrim(profileSnap.ParentWhatsappURL)

		// NEW: gender snapshot ke student_class_sections
		if profileSnap.Gender != nil {
			if g := strings.TrimSpace(*profileSnap.Gender); g != "" {
				scs.StudentClassSectionUserProfileGenderCache = &g
			}
		}
	}

	// NEW: snapshot kode siswa (NIS) dari school_students
	if stuRow.Code != nil {
		if v := nzTrim(stuRow.Code); v != nil {
			scs.StudentClassSectionStudentCodeCache = v
		}
	}

	if err := tx.Create(scs).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan ke section")
	}

	// --- 6) INCREMENT ATOMIC dengan guard kapasitas ---
	res := tx.Exec(`
		UPDATE class_sections
		SET class_section_total_students_active = class_section_total_students_active + 1,
		    class_section_updated_at = NOW()
		WHERE class_section_id = ?
		  AND class_section_deleted_at IS NULL
		  AND class_section_status = 'active'
		  AND (class_section_quota_total IS NULL OR class_section_total_students_active < class_section_quota_total)
	`, sec.ClassSectionID)
	if res.Error != nil {
		// kompensasi: hapus enrollment yang baru dibuat
		_ = tx.Where("student_class_section_id = ?", scs.StudentClassSectionID).
			Delete(&model.StudentClassSection{}).Error
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update counter")
	}
	if res.RowsAffected == 0 {
		// kapasitas penuh tepat saat ini → hapus enrollment agar konsisten
		_ = tx.Where("student_class_section_id = ?", scs.StudentClassSectionID).
			Delete(&model.StudentClassSection{}).Error
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Kelas penuh")
	}

	// --- 6.5) INCREMENT class_quota_taken (+1) secara atomic di parent class ---
	resClass := tx.Exec(`
		UPDATE classes
		SET class_quota_taken = class_quota_taken + 1,
		    class_updated_at = NOW()
		WHERE class_id = ?
		  AND class_deleted_at IS NULL
		  AND class_status = 'active'
		  AND (class_quota_total IS NULL OR class_quota_taken < class_quota_total)
	`, sec.ClassSectionClassID)
	if resClass.Error != nil {
		// kompensasi: hapus enrollment & rollback (increment section ikut dibatalkan oleh TX)
		_ = tx.Where("student_class_section_id = ?", scs.StudentClassSectionID).
			Delete(&model.StudentClassSection{}).Error
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update quota kelas")
	}
	if resClass.RowsAffected == 0 {
		// kuota kelas penuh → batalkan join + counter section
		_ = tx.Where("student_class_section_id = ?", scs.StudentClassSectionID).
			Delete(&model.StudentClassSection{}).Error
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Kuota kelas penuh")
	}

	// --- 7) Commit & respond ---
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
	}

	log.Printf("[SCS][JOIN] user=%s school=%s section=%s", userID, schoolID, sec.ClassSectionID)

	// Pakai DTO join terbaru + normalisasi waktu ke school time
	scsResp := dto.FromModelWithSchoolTime(c, scs)
	joinResp := dto.ClassSectionJoinResponse{
		UserClassSection: &scsResp,
		ClassSectionID:   sec.ClassSectionID.String(),
	}

	resp := fiber.Map{
		"item":      joinResp,
		"school_id": schoolID.String(),
	}
	if profileSnap != nil {
		resp["user_profile_snapshot"] = profileSnap
	}

	return helper.JsonOK(c, "Berhasil bergabung", resp)
}
