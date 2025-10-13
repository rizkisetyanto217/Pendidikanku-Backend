// file: internals/features/school/classes/class_sections/controller/user_class_section_join_controller.go
package controller

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

// Ambil users_profile_id dari user_id
func getUsersProfileID(tx *gorm.DB, userID uuid.UUID) (uuid.UUID, error) {
	// SESUAIKAN nama tabel & kolom jika berbeda
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

// Ambil/buat masjid_student berdasarkan (masjid_id, users_profile_id)
func getOrCreateMasjidStudentByProfile(tx *gorm.DB, masjidID, usersProfileID uuid.UUID) (uuid.UUID, error) {
	type row struct {
		ID uuid.UUID `gorm:"column:masjid_student_id"`
	}
	var r row

	// lookup
	err := tx.Table("masjid_students").
		Select("masjid_student_id").
		Where("masjid_student_masjid_id = ? AND masjid_student_user_profile_id = ? AND masjid_student_deleted_at IS NULL",
			masjidID, usersProfileID).
		First(&r).Error

	if err == nil {
		return r.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, err
	}

	// create baru (minimal — field lain biarkan default/hook)
	newID := uuid.New()
	values := map[string]any{
		"masjid_student_id":              newID,
		"masjid_student_masjid_id":       masjidID,
		"masjid_student_user_profile_id": usersProfileID,
		// slug sementara pakai uuid; ganti bila ada generator khusus
		"masjid_student_slug":   newID.String(),
		"masjid_student_status": "active",
	}
	if err := tx.Table("masjid_students").Create(values).Error; err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

// forUpdate() kamu sendiri (RowLock). Pakai punyamu.
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

	// -------- 3) Pastikan ada masjid_student --------
	usersProfileID, err := getUsersProfileID(tx, userID)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
	}
	masjidStudentID, err := getOrCreateMasjidStudentByProfile(tx, masjidID, usersProfileID)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek/buat status student")
	}

	// -------- 3a) Snapshot profil user (hanya field yg ada di DTO) --------
	profileSnap, perr := userProfileSnapshot.BuildUserProfileSnapshotByUserID(c.Context(), tx, userID)
	if perr != nil && !errors.Is(perr, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot profil")
	}
	log.Printf("[UCS][JOIN] masjid=%s user=%s profileSnap_ok=%t",
		masjidID.String(), userID.String(), profileSnap != nil)

	// -------- 4) Kapasitas --------
	if sec.ClassSectionCapacity != nil && *sec.ClassSectionCapacity > 0 &&
		sec.ClassSectionTotalStudents >= *sec.ClassSectionCapacity {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Kelas penuh")
	}

	// -------- 5) Cegah double-join --------
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

	// -------- 6) Insert enrollment (+ inject snapshot yg sesuai DTO) --------
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
		nz := func(s *string) *string {
			if s == nil {
				return nil
			}
			v := strings.TrimSpace(*s)
			if v == "" {
				return nil
			}
			return &v
		}
		if name := strings.TrimSpace(profileSnap.Name); name != "" {
			ucs.UserClassSectionUserProfileNameSnapshot = &name
		}
		ucs.UserClassSectionUserProfileAvatarURLSnapshot = nz(profileSnap.AvatarURL)
		ucs.UserClassSectionUserProfileWhatsappURLSnapshot = nz(profileSnap.WhatsappURL)
		ucs.UserClassSectionUserProfileParentNameSnapshot = nz(profileSnap.ParentName)
		ucs.UserClassSectionUserProfileParentWhatsappURLSnapshot = nz(profileSnap.ParentWhatsappURL)
	}

	if err := tx.Create(ucs).Error; err != nil {
		_ = tx.Rollback()
		log.Printf("[UCS][JOIN][ERR] insert: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan ke section")
	}
	log.Printf("[UCS][JOIN] ucs_id=%s snapshot_set=%t",
		ucs.UserClassSectionID.String(), profileSnap != nil)

	// -------- 7) Bump counter --------
	if err := tx.Model(&sec).
		Where("class_section_id = ?", sec.ClassSectionID).
		UpdateColumn("class_section_total_students", gorm.Expr("class_section_total_students + 1")).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update counter")
	}

	// -------- 8) Commit & respond --------
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
		resp["user_profile_snapshot"] = profileSnap // tambahan info di response OK
	}

	return helper.JsonOK(c, "Berhasil bergabung", resp)
}
