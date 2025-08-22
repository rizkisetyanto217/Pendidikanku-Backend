// internals/features/masjids/masjid_admins_teachers/controller/masjid_admin_controller.go
package controller

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/dto"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	helper "masjidku_backend/internals/helpers"
)

type MasjidAdminController struct {
	DB *gorm.DB
}

func NewMasjidAdminController(db *gorm.DB) *MasjidAdminController {
	return &MasjidAdminController{DB: db}
}

/* =========================================================
   Helpers (local)
   ========================================================= */

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }


/* =========
   CONFIG
   ========= */
// Role global yang dipromosikan saat user dijadikan admin masjid.
// Ubah ke constants.RoleDKM kalau mau “dkm”.
var promoteRoleOnAdd = constants.RoleDKM

/* =========================================================
   Helpers untuk role user (dipakai di Add/Rev)
   ========================================================= */

// Set role user ke targetRole kecuali kalau dia OWNER (bypass) atau rolenya sudah sama.
func setUserRoleIfNeeded(tx *gorm.DB, userID string, targetRole string) error {
	var u struct{ Role string }
	if err := tx.Table("users").Select("role").Where("id = ?", userID).First(&u).Error; err != nil {
		return err
	}
	cur := strings.ToLower(strings.TrimSpace(u.Role))
	if cur == constants.RoleOwner || cur == strings.ToLower(strings.TrimSpace(targetRole)) {
		return nil
	}
	return tx.Table("users").Where("id = ?", userID).Update("role", targetRole).Error
}

// Jika user sudah TIDAK punya admin aktif di masjid manapun, dan rolenya sekarang admin/dkm,
// turunkan ke user. Owner tidak disentuh.
func demoteUserIfNoOtherAdmin(tx *gorm.DB, userID string) error {
	// hitung admin aktif user ini
	var cnt int64
	if err := tx.Model(&model.MasjidAdminModel{}).
		Where("masjid_admins_user_id = ? AND masjid_admins_is_active = TRUE", userID).
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	// cek role saat ini
	var u struct{ Role string }
	if err := tx.Table("users").Select("role").Where("id = ?", userID).First(&u).Error; err != nil {
		return err
	}
	role := strings.ToLower(strings.TrimSpace(u.Role))
	if role == constants.RoleOwner {
		return nil
	}
	if role == constants.RoleAdmin || role == constants.RoleDKM {
		return tx.Table("users").Where("id = ?", userID).Update("role", constants.RoleUser).Error
	}
	return nil
}

func extractMasjidID(c *fiber.Ctx, fallback string) string {
	// 1) locals (dari IsMasjidAdmin)
	if v, ok := c.Locals("masjid_id").(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	// 2) body (dua kunci yang didukung)
	if b := c.Body(); len(b) > 0 {
		type bodyMasjid struct {
			MasjidID              string `json:"masjid_id"`
			MasjidAdminsMasjidID  string `json:"masjid_admins_masjid_id"`
		}
		var bm bodyMasjid
		_ = c.BodyParser(&bm) // best-effort
		if strings.TrimSpace(bm.MasjidID) != "" {
			return strings.TrimSpace(bm.MasjidID)
		}
		if strings.TrimSpace(bm.MasjidAdminsMasjidID) != "" {
			return strings.TrimSpace(bm.MasjidAdminsMasjidID)
		}
	}
	// 3) param / query / header
	if v := strings.TrimSpace(c.Params("masjid_id")); v != "" { return v }
	if v := strings.TrimSpace(c.Query("masjid_id")); v != "" { return v }
	if v := strings.TrimSpace(c.Get("X-Masjid-ID")); v != "" { return v }
	// 4) fallback (mis. dari DTO)
	return strings.TrimSpace(fallback)
}

func mustParseUUID(c *fiber.Ctx, id, field string) (uuid.UUID, bool) {
	u, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		helper.JsonError(c, fiber.StatusBadRequest, field+" bukan UUID yang valid")
		return uuid.Nil, false
	}
	return u, true
}

func (ctrl *MasjidAdminController) userRole(userID string) (string, error) {
	var u struct{ Role string }
	err := ctrl.DB.Table("users").Select("role").Where("id = ?", userID).First(&u).Error
	if err != nil {
		return "", err
	}
	return norm(u.Role), nil
}

/* =========================================================
   POST /api/a/masjid-admins
   Body: { "masjid_admins_masjid_id": "...", "masjid_admins_user_id": "...", "masjid_admins_is_active": true? }
   Behaviour:
     - Tolak jika target adalah owner (owner tak perlu jadi admin).
     - Upsert: jika sudah ada record & nonaktif → aktifkan kembali.
     - Jika sudah aktif → kembalikan 200 (idempotent).
   ========================================================= */
func (ctrl *MasjidAdminController) AddAdmin(c *fiber.Ctx) error {
	var body dto.MasjidAdminRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input")
	}

	// Ambil masjid_id dari berbagai sumber jika kosong di body
	if strings.TrimSpace(body.MasjidAdminsMasjidID.String()) == "" {
		if mid := extractMasjidID(c, ""); mid != "" {
			if u, ok := mustParseUUID(c, mid, "masjid_id"); ok {
				body.MasjidAdminsMasjidID = u
			} else {
				return nil
			}
		}
	}
	if strings.TrimSpace(body.MasjidAdminsUserID.String()) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_admins_user_id wajib diisi")
	}

	// Cegah: target owner tidak perlu dibuat admin
	roleNow, err := ctrl.userRole(body.MasjidAdminsUserID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa role user")
	}
	if roleNow == constants.RoleOwner {
		return helper.JsonError(c, fiber.StatusBadRequest, "Target adalah owner—tidak perlu dibuat admin")
	}

	// Transaksi: upsert admin + promote role
	return ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// Upsert behaviour
		var existing model.MasjidAdminModel
		err := tx.
			Where("masjid_admins_masjid_id = ? AND masjid_admins_user_id = ?",
				body.MasjidAdminsMasjidID, body.MasjidAdminsUserID).
			First(&existing).Error

		switch {
		case err == nil:
			if existing.MasjidAdminsIsActive {
				// pastikan role global “admin/dkm”
				if e := setUserRoleIfNeeded(tx, body.MasjidAdminsUserID.String(), promoteRoleOnAdd); e != nil {
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
				}
				return helper.JsonUpdated(c, "User sudah menjadi admin di masjid ini (idempotent)", dto.ToMasjidAdminResponse(&existing))
			}
			// aktifkan kembali
			if e := tx.Model(&model.MasjidAdminModel{}).
				Where("masjid_admins_id = ?", existing.MasjidAdminsID).
				Update("masjid_admins_is_active", true).Error; e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengaktifkan kembali admin")
			}
			existing.MasjidAdminsIsActive = true
			// promote role
			if e := setUserRoleIfNeeded(tx, body.MasjidAdminsUserID.String(), promoteRoleOnAdd); e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
			}
			return helper.JsonUpdated(c, "Admin diaktifkan kembali", dto.ToMasjidAdminResponse(&existing))

		case errors.Is(err, gorm.ErrRecordNotFound):
			// create baru
			admin := body.ToModelCreate()
			if e := tx.Create(admin).Error; e != nil {
				msg := strings.ToLower(e.Error())
				if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
					// idempotent
					if e2 := setUserRoleIfNeeded(tx, body.MasjidAdminsUserID.String(), promoteRoleOnAdd); e2 != nil {
						return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
					}
					return helper.JsonUpdated(c, "User sudah menjadi admin di masjid ini (idempotent)", dto.ToMasjidAdminResponse(admin))
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan admin")
			}
			// promote role
			if e := setUserRoleIfNeeded(tx, body.MasjidAdminsUserID.String(), promoteRoleOnAdd); e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
			}
			return helper.JsonCreated(c, "Admin berhasil ditambahkan", dto.ToMasjidAdminResponse(admin))

		default:
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa admin existing")
		}
	})
}

/* =========================================================
   POST /api/a/masjid-admins/by-masjid
   Body (opsional): { "masjid_admins_masjid_id": "..." }
   Behaviour:
     - Ambil masjid_id dari body/query/header/locals.
   ========================================================= */
func (ctrl *MasjidAdminController) GetAdminsByMasjid(c *fiber.Ctx) error {
	var body struct {
		MasjidAdminsMasjidID string `json:"masjid_admins_masjid_id"`
	}
	_ = c.BodyParser(&body)

	masjidID := extractMasjidID(c, body.MasjidAdminsMasjidID)
	if masjidID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib dikirim (header/query/body)")
	}
	mid, ok := mustParseUUID(c, masjidID, "masjid_id")
	if !ok {
		return nil
	}

	var admins []model.MasjidAdminModel
	if err := ctrl.DB.
		Preload("User").
		Where("masjid_admins_masjid_id = ? AND masjid_admins_is_active = TRUE", mid).
		Find(&admins).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar admin aktif")
	}

	out := make([]dto.MasjidAdminResponse, 0, len(admins))
	for i := range admins {
		out = append(out, dto.ToMasjidAdminResponse(&admins[i]))
	}
	return helper.JsonOK(c, "Daftar admin aktif berhasil diambil", out)
}

/* =========================================================
   PUT /api/a/masjid-admins/revoke
   Body (fleksibel):
     - {"user_id":"...", "masjid_id":"..."}  // generik
     - {"masjid_admins_user_id":"...", "masjid_admins_masjid_id":"..."} // kompat lama
   Behaviour:
     - Jika target owner → idempotent OK (owner tidak punya record admin).
     - Jika bukan admin / sudah nonaktif → idempotent OK.
     - Jika aktif → nonaktifkan.
   ========================================================= */
type revokeReq struct {
	// field generik
	UserID   string `json:"user_id"`
	MasjidID string `json:"masjid_id"`
	// field spesifik tabel (kompatibel dengan payload lama)
	MasjidAdminsUserID   string `json:"masjid_admins_user_id"`
	MasjidAdminsMasjidID string `json:"masjid_admins_masjid_id"`
}

func (ctrl *MasjidAdminController) RevokeAdmin(c *fiber.Ctx) error {
	var req revokeReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input")
	}

	// Normalisasi input user_id & masjid_id
	userID := strings.TrimSpace(req.MasjidAdminsUserID)
	if userID == "" { userID = strings.TrimSpace(req.UserID) }
	masjidID := strings.TrimSpace(req.MasjidAdminsMasjidID)
	if masjidID == "" { masjidID = strings.TrimSpace(req.MasjidID) }
	if masjidID == "" { masjidID = extractMasjidID(c, "") }

	if userID == "" || masjidID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "user_id/masjid_id wajib dikirim (boleh pakai nama generik atau nama kolom tabel)")
	}
	if _, ok := mustParseUUID(c, userID, "user_id"); !ok { return nil }
	if _, ok := mustParseUUID(c, masjidID, "masjid_id"); !ok { return nil }

	log.Printf("[REVOKE] user_id=%s masjid_id=%s", userID, masjidID)

	// Transaksi: revoke admin + demote kalau perlu
	return ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Target owner? → idempotent OK
		role, err := ctrl.userRole(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "User tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil role user")
		}
		if role == constants.RoleOwner {
			// Owner tidak punya record admin—pastikan tidak demote apa pun
			return helper.JsonUpdated(c, "Target adalah owner—tidak ada admin yang perlu dicabut (idempotent)", nil)
		}

		// 2) Revoke jika masih aktif
		res := tx.Model(&model.MasjidAdminModel{}).
			Where("masjid_admins_user_id = ? AND masjid_admins_masjid_id = ? AND masjid_admins_is_active = TRUE",
				userID, masjidID).
			Update("masjid_admins_is_active", false)
		if res.Error != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menonaktifkan admin")
		}

		// 3) Setelah revoke (atau idempotent), cek apakah user masih admin di masjid lain.
		if err := demoteUserIfNoOtherAdmin(tx, userID); err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
		}

		if res.RowsAffected == 0 {
			// Apakah ada record sama sekali?
			var cnt int64
			if err := tx.Model(&model.MasjidAdminModel{}).
				Where("masjid_admins_user_id = ? AND masjid_admins_masjid_id = ?", userID, masjidID).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek data admin")
			}
			if cnt == 0 {
				return helper.JsonUpdated(c, "Tidak ada admin untuk user & masjid ini (idempotent)", nil)
			}
			return helper.JsonUpdated(c, "Admin sudah nonaktif sebelumnya (idempotent)", nil)
		}

		return helper.JsonUpdated(c, "Admin berhasil dinonaktifkan", nil)
	})
}
