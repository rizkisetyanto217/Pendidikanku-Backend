// internals/features/masjids/masjid_admins_teachers/controller/masjid_admin_controller.go
package controller

import (
	"errors"
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

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

var promoteRoleOnAdd = constants.RoleDKM

// ======================
// Helpers untuk role user
// ======================
func setUserRoleIfNeeded(tx *gorm.DB, userID string, targetRole string) error {
	var u struct{ Role string }
	if err := tx.Table("users").Select("role").Where("id = ?", userID).First(&u).Error; err != nil {
		return err
	}
	cur := norm(u.Role)
	if cur == constants.RoleOwner || cur == norm(targetRole) {
		return nil
	}
	return tx.Table("users").Where("id = ?", userID).Update("role", targetRole).Error
}

func demoteUserIfNoOtherAdmin(tx *gorm.DB, userID string) error {
	var cnt int64
	if err := tx.Model(&model.MasjidAdminModel{}).
		Where("masjid_admin_user_id = ? AND masjid_admin_is_active = TRUE", userID).
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}

	var u struct{ Role string }
	if err := tx.Table("users").Select("role").Where("id = ?", userID).First(&u).Error; err != nil {
		return err
	}
	role := norm(u.Role)
	if role == constants.RoleOwner {
		return nil
	}
	if role == constants.RoleAdmin || role == constants.RoleDKM {
		return tx.Table("users").Where("id = ?", userID).Update("role", constants.RoleUser).Error
	}
	return nil
}

func extractMasjidID(c *fiber.Ctx, fallback string) string {
	if v, ok := c.Locals("masjid_id").(string); ok && v != "" {
		return strings.TrimSpace(v)
	}
	if b := c.Body(); len(b) > 0 {
		type bodyMasjid struct {
			MasjidID            string `json:"masjid_id"`
			MasjidAdminMasjidID string `json:"masjid_admin_masjid_id"`
		}
		var bm bodyMasjid
		_ = c.BodyParser(&bm)
		if bm.MasjidID != "" {
			return strings.TrimSpace(bm.MasjidID)
		}
		if bm.MasjidAdminMasjidID != "" {
			return strings.TrimSpace(bm.MasjidAdminMasjidID)
		}
	}
	if v := c.Params("masjid_id"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := c.Query("masjid_id"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := c.Get("X-Masjid-ID"); v != "" {
		return strings.TrimSpace(v)
	}
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

// =========================================================
// POST /api/a/masjid-admins
// Body: { "masjid_admin_masjid_id": "...", "masjid_admin_user_id": "...", "masjid_admin_is_active": true? }
// =========================================================
func (ctrl *MasjidAdminController) AddAdmin(c *fiber.Ctx) error {
	var body dto.MasjidAdminRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input")
	}

	if body.MasjidAdminMasjidID == uuid.Nil {
		if mid := extractMasjidID(c, ""); mid != "" {
			if u, ok := mustParseUUID(c, mid, "masjid_id"); ok {
				body.MasjidAdminMasjidID = u
			} else {
				return nil
			}
		}
	}
	if body.MasjidAdminUserID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_admin_user_id wajib diisi")
	}

	roleNow, err := ctrl.userRole(body.MasjidAdminUserID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa role user")
	}
	if roleNow == constants.RoleOwner {
		return helper.JsonError(c, fiber.StatusBadRequest, "Target adalah owner—tidak perlu dibuat admin")
	}

	return ctrl.DB.Transaction(func(tx *gorm.DB) error {
		var existing model.MasjidAdminModel
		err := tx.
			Where("masjid_admin_masjid_id = ? AND masjid_admin_user_id = ?",
				body.MasjidAdminMasjidID, body.MasjidAdminUserID).
			First(&existing).Error

		switch {
		case err == nil:
			if existing.MasjidAdminIsActive {
				if e := setUserRoleIfNeeded(tx, body.MasjidAdminUserID.String(), promoteRoleOnAdd); e != nil {
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
				}
				return helper.JsonUpdated(c, "User sudah menjadi admin di masjid ini (idempotent)", dto.ToMasjidAdminResponse(&existing))
			}
			if e := tx.Model(&model.MasjidAdminModel{}).
				Where("masjid_admin_id = ?", existing.MasjidAdminID).
				Update("masjid_admin_is_active", true).Error; e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengaktifkan kembali admin")
			}
			existing.MasjidAdminIsActive = true
			if e := setUserRoleIfNeeded(tx, body.MasjidAdminUserID.String(), promoteRoleOnAdd); e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
			}
			return helper.JsonUpdated(c, "Admin diaktifkan kembali", dto.ToMasjidAdminResponse(&existing))

		case errors.Is(err, gorm.ErrRecordNotFound):
			admin := body.ToModelCreate()
			if e := tx.Create(admin).Error; e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan admin")
			}
			if e := setUserRoleIfNeeded(tx, body.MasjidAdminUserID.String(), promoteRoleOnAdd); e != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
			}
			return helper.JsonCreated(c, "Admin berhasil ditambahkan", dto.ToMasjidAdminResponse(admin))

		default:
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa admin existing")
		}
	})
}

// =========================================================
// POST /api/a/masjid-admins/by-masjid
// Body (opsional): { "masjid_admin_masjid_id": "..." }
// =========================================================
func (ctrl *MasjidAdminController) GetAdminsByMasjid(c *fiber.Ctx) error {
	var body struct {
		MasjidAdminMasjidID string `json:"masjid_admin_masjid_id"`
	}
	_ = c.BodyParser(&body)

	masjidID := extractMasjidID(c, body.MasjidAdminMasjidID)
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
		Where("masjid_admin_masjid_id = ? AND masjid_admin_is_active = TRUE", mid).
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
   Body:
     { "masjid_admin_user_id":"...", "masjid_admin_masjid_id":"..." }
   Behaviour:
     - Jika target owner → idempotent OK (owner tidak punya record admin).
     - Jika bukan admin / sudah nonaktif → idempotent OK.
     - Jika aktif → nonaktifkan.
   ========================================================= */
type revokeReq struct {
	MasjidAdminUserID   string `json:"masjid_admin_user_id"`
	MasjidAdminMasjidID string `json:"masjid_admin_masjid_id"`
}

func (ctrl *MasjidAdminController) RevokeAdmin(c *fiber.Ctx) error {
	var req revokeReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input")
	}

	userID := strings.TrimSpace(req.MasjidAdminUserID)
	masjidID := strings.TrimSpace(req.MasjidAdminMasjidID)
	if masjidID == "" { // masih boleh fallback dari header/query/locals
		masjidID = extractMasjidID(c, "")
	}

	if userID == "" || masjidID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_admin_user_id dan masjid_admin_masjid_id wajib diisi")
	}
	if _, ok := mustParseUUID(c, userID, "masjid_admin_user_id"); !ok { return nil }
	if _, ok := mustParseUUID(c, masjidID, "masjid_admin_masjid_id"); !ok { return nil }

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
			return helper.JsonUpdated(c, "Target adalah owner—tidak ada admin yang perlu dicabut (idempotent)", nil)
		}

		// 2) Revoke jika masih aktif
		res := tx.Model(&model.MasjidAdminModel{}).
			Where("masjid_admin_user_id = ? AND masjid_admin_masjid_id = ? AND masjid_admin_is_active = TRUE",
				userID, masjidID).
			Update("masjid_admin_is_active", false)
		if res.Error != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menonaktifkan admin")
		}

		// 3) Setelah revoke (atau idempotent), cek apakah user masih admin di masjid lain.
		if err := demoteUserIfNoOtherAdmin(tx, userID); err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui role user")
		}

		if res.RowsAffected == 0 {
			// Apakah ada record sama sekali (meskipun sudah nonaktif)?
			var cnt int64
			if err := tx.Model(&model.MasjidAdminModel{}).
				Where("masjid_admin_user_id = ? AND masjid_admin_masjid_id = ?", userID, masjidID).
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
