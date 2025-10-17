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
   Utils & small helpers
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

// Ambil users_profile_id dari user_id
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

// Snapshot ringan masjid (untuk disimpan di masjid_students)
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

// Membuat/menemukan masjid_student & memastikan snapshots terisi.
func getOrCreateMasjidStudentWithSnapshots(
	ctx context.Context,
	tx *gorm.DB,
	masjidID uuid.UUID,
	userProfileID uuid.UUID,
	profileSnap *userProfileSnapshot.UserProfileSnapshot, // boleh nil
) (uuid.UUID, error) {
	// ambil snapshot profil jika belum ada
	if profileSnap == nil {
		if ps, e := userProfileSnapshot.BuildUserProfileSnapshotByProfileID(ctx, tx, userProfileID); e == nil {
			profileSnap = ps
		}
	}
	// ambil snapshot masjid
	mName, mSlug, mLogo, mIcon, mBg, _ := getMasjidSnapshot(tx, masjidID)

	// cek existing
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
		// top-up snapshot yang kosong (best-effort: set saja nilai terbaru)
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
		if e := tx.Table("masjid_students").
			Where("masjid_student_id = ?", cur.ID).
			Updates(updates).Error; e != nil {
			return uuid.Nil, e
		}
		return cur.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, err
	}

	// create baru (lengkap dgn snapshots)
	newID := uuid.New()
	values := map[string]any{
		"masjid_student_id":              newID,
		"masjid_student_masjid_id":       masjidID,
		"masjid_student_user_profile_id": userProfileID,
		"masjid_student_slug":            newID.String(), // ganti kalau punya generator sendiri
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

// POST /api/a/:masjid_id/user-class-sections/join
func (ctl *UserClassSectionController) JoinByCode(c *fiber.Ctx) error {
	// -------- tenant --------
	rawMasjidID := strings.TrimSpace(c.Params("masjid_id"))
	masjidID, err := uuid.Parse(rawMasjidID)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id path tidak valid")
	}

	// -------- auth --------
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	// -------- body --------
	type studentJoinReq struct {
		StudentCode    string     `json:"student_code" form:"student_code"`
		ClassSectionID *uuid.UUID `json:"class_section_id,omitempty" form:"class_section_id"` // diabaikan
	}
	var req studentJoinReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	code := strings.TrimSpace(req.StudentCode)
	if code == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "student_code wajib diisi")
	}

	// -------- TX --------
	tx := ctl.DB.WithContext(c.Context()).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// -------- 1) Temukan section berdasarkan student_code --------
	var sec model.ClassSectionModel
	found := false

	// (a) plaintext
	if err := tx.
		Clauses(forUpdate()).
		Where(`class_section_masjid_id = ? AND class_section_code = ? AND class_section_deleted_at IS NULL`,
			masjidID, code).
		First(&sec).Error; err == nil {
		found = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari section (by plaintext)")
	}

	// (b) bcrypt scan
	if !found {
		var rows []struct {
			ID   uuid.UUID `gorm:"column:class_section_id"`
			Hash []byte    `gorm:"column:class_section_student_code_hash"`
		}
		if err := tx.Table("class_sections").
			Select("class_section_id, class_section_student_code_hash").
			Where(`
				class_section_masjid_id = ?
				AND class_section_deleted_at IS NULL
				AND class_section_student_code_hash IS NOT NULL
				AND octet_length(class_section_student_code_hash) > 0
			`, masjidID).
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

	// -------- 2) Validasi section --------
	if !sec.ClassSectionIsActive {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Section tidak aktif")
	}
	if sec.ClassSectionMasjidID != masjidID {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusForbidden, "Section bukan milik masjid ini")
	}
	// Kapasitas (jika ada)
	if sec.ClassSectionCapacity != nil && *sec.ClassSectionCapacity > 0 &&
		sec.ClassSectionTotalStudents >= *sec.ClassSectionCapacity {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Kelas penuh")
	}

	// -------- 3) Pastikan ada masjid_student + isi snapshots --------
	usersProfileID, err := getUsersProfileID(tx, userID)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
	}
	// Ambil snapshot profil SEKALI, dipakai untuk masjid_students & user_class_sections
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
	log.Printf("[UCS][JOIN] masjid=%s profile_id=%s student_id=%s profileSnap_ok=%t",
		masjidID.String(), usersProfileID.String(), masjidStudentID.String(), profileSnap != nil)

	// -------- 4) Cegah double-join --------
	var exists int64
	if err := tx.Table("user_class_sections").
		Where(`user_class_section_masjid_id = ?
		       AND user_class_section_masjid_student_id = ?
		       AND user_class_section_section_id = ?
		       AND user_class_section_deleted_at IS NULL`,
			masjidID, masjidStudentID, sec.ClassSectionID).
		Count(&exists).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek keanggotaan")
	}
	if exists > 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Sudah tergabung di section ini")
	}

	// -------- 5) Insert enrollment (+ inject snapshot profil) --------
	now := time.Now()
	ucs := &model.UserClassSection{
		UserClassSectionID:              uuid.New(),
		UserClassSectionMasjidID:        masjidID,
		UserClassSectionMasjidStudentID: masjidStudentID,
		UserClassSectionSectionID:       sec.ClassSectionID,
		UserClassSectionStatus:          model.UserClassSectionActive,
		UserClassSectionAssignedAt:      now,
		UserClassSectionCreatedAt:       now,
		UserClassSectionUpdatedAt:       now,
	}

	if profileSnap != nil {
		if name := strings.TrimSpace(profileSnap.Name); name != "" {
			ucs.UserClassSectionUserProfileNameSnapshot = &name
		}
		ucs.UserClassSectionUserProfileAvatarURLSnapshot = nzTrim(profileSnap.AvatarURL)
		ucs.UserClassSectionUserProfileWhatsappURLSnapshot = nzTrim(profileSnap.WhatsappURL)
		ucs.UserClassSectionUserProfileParentNameSnapshot = nzTrim(profileSnap.ParentName)
		ucs.UserClassSectionUserProfileParentWhatsappURLSnapshot = nzTrim(profileSnap.ParentWhatsappURL)
	}

	if err := tx.Create(ucs).Error; err != nil {
		_ = tx.Rollback()
		log.Printf("[UCS][JOIN][ERR] insert: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan ke section")
	}
	log.Printf("[UCS][JOIN] ucs_id=%s snapshot_set=%t", ucs.UserClassSectionID.String(), profileSnap != nil)

	// -------- 6) Bump counter --------
	if err := tx.Model(&sec).
		Where("class_section_id = ?", sec.ClassSectionID).
		UpdateColumn("class_section_total_students", gorm.Expr("class_section_total_students + 1")).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update counter")
	}

	// -------- 7) Commit & respond --------
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
	}

	resp := fiber.Map{
		"item": dto.ClassSectionJoinResponse{
			UserClassSection: Ptr(dto.FromModel(ucs)),
			ClassSectionID:   sec.ClassSectionID.String(),
		},
	}
	if profileSnap != nil {
		resp["user_profile_snapshot"] = profileSnap
	}

	return helper.JsonOK(c, "Berhasil bergabung", resp)
}
