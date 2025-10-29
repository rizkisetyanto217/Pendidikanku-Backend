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

	dto "masjidku_backend/internals/features/school/classes/class_sections/dto"
	model "masjidku_backend/internals/features/school/classes/class_sections/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	userProfileSnapshot "masjidku_backend/internals/features/users/users/snapshot"
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

// Snapshot masjid (ringkas)
func getMasjidSnapshot(tx *gorm.DB, masjidID uuid.UUID) (name, slug, logo, icon, bg *string, err error) {
	var row struct {
		Name *string `gorm:"column:masjid_name"`
		Slug *string `gorm:"column:masjid_slug"`
		Logo *string `gorm:"column:masjid_logo_url"`
		Icon *string `gorm:"column:masjid_icon_url"`
		Bg   *string `gorm:"column:masjid_background_url"`
	}
	err = tx.Table("masjids").
		Select("masjid_name, masjid_slug, masjid_logo_url, masjid_icon_url, masjid_background_url").
		Where("masjid_id = ? AND masjid_deleted_at IS NULL", masjidID).
		Take(&row).Error
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return row.Name, row.Slug, row.Logo, row.Icon, row.Bg, nil
}

/* =========================
   Student upsert + snapshots
========================= */

// Get/create masjid_students + isi snapshots (profil & masjid)
func getOrCreateMasjidStudentWithSnapshots(
	ctx context.Context,
	tx *gorm.DB,
	masjidID uuid.UUID,
	userProfileID uuid.UUID,
	profileSnap *userProfileSnapshot.UserProfileSnapshot, // boleh nil
) (uuid.UUID, error) {
	// snapshot profil kalau belum ada
	if profileSnap == nil {
		if ps, e := userProfileSnapshot.BuildUserProfileSnapshotByProfileID(ctx, tx, userProfileID); e == nil {
			profileSnap = ps
		}
	}
	// snapshot masjid
	mName, mSlug, mLogo, mIcon, mBg, _ := getMasjidSnapshot(tx, masjidID)

	// ada existing?
	var cur struct {
		ID uuid.UUID `gorm:"column:masjid_student_id"`
	}
	err := tx.Table("masjid_students").
		Select("masjid_student_id").
		Where("masjid_student_masjid_id = ? AND masjid_student_user_profile_id = ? AND masjid_student_deleted_at IS NULL",
			masjidID, userProfileID).
		Take(&cur).Error

	now := time.Now()

	if err == nil {
		// top-up snapshots (best-effort)
		updates := map[string]any{
			"masjid_student_updated_at": now,
		}
		if profileSnap != nil {
			if name := strings.TrimSpace(profileSnap.Name); name != "" {
				updates["masjid_student_user_profile_name_snapshot"] = name
			}
			if v := nzTrim(profileSnap.AvatarURL); v != nil {
				updates["masjid_student_user_profile_avatar_url_snapshot"] = *v
			}
			if v := nzTrim(profileSnap.WhatsappURL); v != nil {
				updates["masjid_student_user_profile_whatsapp_url_snapshot"] = *v
			}
			if v := nzTrim(profileSnap.ParentName); v != nil {
				updates["masjid_student_user_profile_parent_name_snapshot"] = *v
			}
			if v := nzTrim(profileSnap.ParentWhatsappURL); v != nil {
				updates["masjid_student_user_profile_parent_whatsapp_url_snapshot"] = *v
			}
		}
		if v := nzTrim(mName); v != nil {
			updates["masjid_student_masjid_name_snapshot"] = *v
		}
		if v := nzTrim(mSlug); v != nil {
			updates["masjid_student_masjid_slug_snapshot"] = *v
		}
		if v := nzTrim(mLogo); v != nil {
			updates["masjid_student_masjid_logo_url_snapshot"] = *v
		}
		if v := nzTrim(mIcon); v != nil {
			updates["masjid_student_masjid_icon_url_snapshot"] = *v
		}
		if v := nzTrim(mBg); v != nil {
			updates["masjid_student_masjid_background_url_snapshot"] = *v
		}
		if e := tx.Table("masjid_students").Where("masjid_student_id = ?", cur.ID).Updates(updates).Error; e != nil {
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
		"masjid_student_id":              newID,
		"masjid_student_masjid_id":       masjidID,
		"masjid_student_user_profile_id": userProfileID,
		"masjid_student_slug":            newID.String(),
		"masjid_student_status":          "active",
		"masjid_student_sections":        datatypes.JSON([]byte("[]")),
		"masjid_student_created_at":      now,
		"masjid_student_updated_at":      now,
	}
	if profileSnap != nil {
		if name := strings.TrimSpace(profileSnap.Name); name != "" {
			values["masjid_student_user_profile_name_snapshot"] = name
		}
		if v := nzTrim(profileSnap.AvatarURL); v != nil {
			values["masjid_student_user_profile_avatar_url_snapshot"] = *v
		}
		if v := nzTrim(profileSnap.WhatsappURL); v != nil {
			values["masjid_student_user_profile_whatsapp_url_snapshot"] = *v
		}
		if v := nzTrim(profileSnap.ParentName); v != nil {
			values["masjid_student_user_profile_parent_name_snapshot"] = *v
		}
		if v := nzTrim(profileSnap.ParentWhatsappURL); v != nil {
			values["masjid_student_user_profile_parent_whatsapp_url_snapshot"] = *v
		}
	}
	if v := nzTrim(mName); v != nil {
		values["masjid_student_masjid_name_snapshot"] = *v
	}
	if v := nzTrim(mSlug); v != nil {
		values["masjid_student_masjid_slug_snapshot"] = *v
	}
	if v := nzTrim(mLogo); v != nil {
		values["masjid_student_masjid_logo_url_snapshot"] = *v
	}
	if v := nzTrim(mIcon); v != nil {
		values["masjid_student_masjid_icon_url_snapshot"] = *v
	}
	if v := nzTrim(mBg); v != nil {
		values["masjid_student_masjid_background_url_snapshot"] = *v
	}

	if err := tx.Table("masjid_students").Create(values).Error; err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

/* =========================
   Handler
========================= */

// POST /api/a/:masjid_id/student-class-sections/join     (versi lama: masjid_id di path)
// POST /api/student-class-sections/join                   (versi baru: auto masjid dari code)
// Body: { "student_code": "...." }
func (ctl *StudentClassSectionController) JoinByCodeAutoMasjid(c *fiber.Ctx) error {
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

	// --- 1) Temukan section dari code (tanpa filter masjid dulu) ---
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

	// ✅ masjid_id diturunkan dari section
	masjidID := sec.ClassSectionMasjidID
	if masjidID == uuid.Nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Section tidak memiliki konteks masjid yang valid")
	}

	// --- 2) Validasi section (awal; guard final di UPDATE atomic) ---
	if !sec.ClassSectionIsActive {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Section tidak aktif")
	}
	// Pre-check kapasitas bisa dilewati; final guard ada di UPDATE atomic di bawah.
	// if sec.ClassSectionCapacity != nil && *sec.ClassSectionCapacity > 0 &&
	// 	sec.ClassSectionTotalStudents >= *sec.ClassSectionCapacity {
	// 	_ = tx.Rollback()
	// 	return helper.JsonError(c, fiber.StatusConflict, "Kelas penuh")
	// }

	// --- 3) Pastikan ada masjid_student + isi snapshots ---
	usersProfileID, err := getUsersProfileID(tx, userID)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
	}
	profileSnap, perr := userProfileSnapshot.BuildUserProfileSnapshotByProfileID(c.Context(), tx, usersProfileID)
	if perr != nil && !errors.Is(perr, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot profil")
	}
	masjidStudentID, err := getOrCreateMasjidStudentWithSnapshots(c.Context(), tx, masjidID, usersProfileID, profileSnap)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek/buat status student")
	}

	// --- 4) Cegah double-join ---
	var exists int64
	if err := tx.Table("student_class_sections").
		Where(`
			student_class_section_masjid_id = ?
			AND student_class_section_masjid_student_id = ?
			AND student_class_section_section_id = ?
			AND student_class_section_deleted_at IS NULL
		`, masjidID, masjidStudentID, sec.ClassSectionID).
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
	scs := &model.StudentClassSection{
		StudentClassSectionID:              uuid.New(),
		StudentClassSectionMasjidID:        masjidID,
		StudentClassSectionMasjidStudentID: masjidStudentID,
		StudentClassSectionSectionID:       sec.ClassSectionID,
		StudentClassSectionStatus:          model.StudentClassSectionActive,
		StudentClassSectionAssignedAt:      now,
		StudentClassSectionCreatedAt:       now,
		StudentClassSectionUpdatedAt:       now,
	}
	if profileSnap != nil {
		if name := strings.TrimSpace(profileSnap.Name); name != "" {
			scs.StudentClassSectionUserProfileNameSnapshot = &name
		}
		scs.StudentClassSectionUserProfileAvatarURLSnapshot = nzTrim(profileSnap.AvatarURL)
		scs.StudentClassSectionUserProfileWhatsappURLSnapshot = nzTrim(profileSnap.WhatsappURL)
		scs.StudentClassSectionUserProfileParentNameSnapshot = nzTrim(profileSnap.ParentName)
		scs.StudentClassSectionUserProfileParentWhatsappURLSnapshot = nzTrim(profileSnap.ParentWhatsappURL)
	}

	if err := tx.Create(scs).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan ke section")
	}

	// --- 6) INCREMENT ATOMIC dengan guard kapasitas ---
	res := tx.Exec(`
		UPDATE class_sections
		SET class_section_total_students = class_section_total_students + 1,
		    class_section_updated_at = NOW()
		WHERE class_section_id = ?
		  AND class_section_deleted_at IS NULL
		  AND class_section_is_active = TRUE
		  AND (class_section_capacity IS NULL OR class_section_total_students < class_section_capacity)
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

	log.Printf("[SCS][JOIN] user=%s masjid=%s section=%s", userID, masjidID, sec.ClassSectionID)

	resp := fiber.Map{
		"item": fiber.Map{
			"student_class_section": dto.FromModel(scs),
			"class_section_id":      sec.ClassSectionID.String(),
		},
		"masjid_id": masjidID.String(),
	}
	if profileSnap != nil {
		resp["user_profile_snapshot"] = profileSnap
	}

	return helper.JsonOK(c, "Berhasil bergabung", resp)
}