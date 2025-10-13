// file: internals/features/users/auth/controller/me_context_controller.go
package controller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	masjidModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"
	userModel "masjidku_backend/internals/features/users/users/model"

	helper "masjidku_backend/internals/helpers"        // JsonOK/JsonError
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* =============== Link models (pastikan ada, atau definisikan ringan di sini) =============== */
// type UserTeacher struct {...}      func (UserTeacher) TableName() string { return "user_teachers" }
// type MasjidTeacher struct {...}    func (MasjidTeacher) TableName() string { return "masjid_teachers" }
// type UserProfile struct {...}      func (UserProfile) TableName() string { return "user_profiles" }
// type MasjidStudent struct {...}    func (MasjidStudent) TableName() string { return "masjid_students" }

/* =============== DTO Response (baru & ringkas) =============== */

type MasjidRoleOption struct {
	MasjidID      uuid.UUID `json:"masjid_id"`
	MasjidName    string    `json:"masjid_name"`
	MasjidSlug    string    `json:"masjid_slug"`
	MasjidIconURL *string   `json:"masjid_icon_url,omitempty"`
	Roles         []string  `json:"roles"`
}

type ScopeSelection struct {
	MasjidID *uuid.UUID `json:"masjid_id,omitempty"`
	Role     *string    `json:"role,omitempty"`
}

type MyScopeResponse struct {
	UserID      uuid.UUID          `json:"user_id"`
	UserName    string             `json:"user_name"`
	Memberships []MasjidRoleOption `json:"memberships"`
	Selection   *ScopeSelection    `json:"selection,omitempty"`
}

/* =============== Helper lokal: decode klaim JWT (tanpa verifikasi) =============== */

type jwtMasjidRole struct {
	MasjidID string   `json:"masjid_id"`
	Roles    []string `json:"roles"`
}
type jwtClaimsLite struct {
	MasjidIDs      []string        `json:"masjid_ids"`
	MasjidRoles    []jwtMasjidRole `json:"masjid_roles"`
	ActiveMasjidID string          `json:"active_masjid_id"`
}

// Ambil token dari Authorization: Bearer ... atau cookie access_token
func getAccessTokenFromCtx(c *fiber.Ctx) string {
	auth := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") && len(auth) > 7 {
		return strings.TrimSpace(auth[7:])
	}
	// fallback: cookie (kalau FE simpan di cookie)
	if v := strings.TrimSpace(c.Cookies("access_token")); v != "" {
		return v
	}
	return ""
}

// Decode payload JWT (bagian tengah) tanpa verifikasi untuk baca klaim
func parseMasjidInfoFromJWT(c *fiber.Ctx) (ids []uuid.UUID, roleMap map[uuid.UUID]map[string]struct{}) {
	roleMap = map[uuid.UUID]map[string]struct{}{}

	token := getAccessTokenFromCtx(c)
	if token == "" {
		return
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return
	}
	payloadB64 := parts[1]

	// JWT pakai base64url tanpa padding
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		// coba decoder biasa (jika ada padding)
		if b2, e2 := base64.StdEncoding.DecodeString(payloadB64); e2 == nil {
			payloadBytes = b2
		} else {
			return
		}
	}

	var cl jwtClaimsLite
	if err := json.Unmarshal(payloadBytes, &cl); err != nil {
		return
	}

	// kumpulkan masjid_ids
	seen := map[uuid.UUID]struct{}{}
	for _, s := range cl.MasjidIDs {
		if id, e := uuid.Parse(strings.TrimSpace(s)); e == nil && id != uuid.Nil {
			if _, ok := seen[id]; !ok {
				ids = append(ids, id)
				seen[id] = struct{}{}
			}
		}
	}
	// dari masjid_roles[].masjid_id
	for _, mr := range cl.MasjidRoles {
		if id, e := uuid.Parse(strings.TrimSpace(mr.MasjidID)); e == nil && id != uuid.Nil {
			if _, ok := seen[id]; !ok {
				ids = append(ids, id)
				seen[id] = struct{}{}
			}
			if _, ok := roleMap[id]; !ok {
				roleMap[id] = map[string]struct{}{}
			}
			for _, r := range mr.Roles {
				r = strings.ToLower(strings.TrimSpace(r))
				if r != "" {
					roleMap[id][r] = struct{}{}
				}
			}
		}
	}
	// active_masjid_id (opsional)
	if cl.ActiveMasjidID != "" {
		if id, e := uuid.Parse(strings.TrimSpace(cl.ActiveMasjidID)); e == nil && id != uuid.Nil {
			if _, ok := seen[id]; !ok {
				ids = append(ids, id)
			}
		}
	}

	return
}

/* =============== Controller: GetMyContext (versi scope/role) =============== */
func (ac *AuthController) GetMySimpleContext(c *fiber.Ctx) error {
	// 1) Ambil user_id via helperAuth (diisi middleware)
	userUUID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userUUID == uuid.Nil {
		// Fallback dev: ?user_id=
		if userIDStr := strings.TrimSpace(c.Query("user_id")); userIDStr != "" {
			if parsed, e := uuid.Parse(userIDStr); e == nil {
				userUUID = parsed
			}
		}
		if userUUID == uuid.Nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, "user_id tidak tersedia pada context")
		}
	}

	// 2) Ambil user (PK "id")
	var me userModel.UserModel
	if err := ac.DB.WithContext(c.Context()).
		Select("id, user_name").
		Where("id = ?", userUUID).
		First(&me).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil user: "+err.Error())
	}

	// 3) Kumpulkan masjid_id via TEACHER & STUDENT
	masjidRoles := map[uuid.UUID]map[string]struct{}{} // masjid_id -> set(role)

	// 3a) TEACHER
	{
		var userTeacherIDs []uuid.UUID
		if err := ac.DB.WithContext(c.Context()).
			Model(&UserTeacher{}).
			Where("user_teacher_user_id = ?", userUUID).
			Pluck("user_teacher_id", &userTeacherIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil user_teachers: "+err.Error())
		}
		if len(userTeacherIDs) > 0 {
			var mtMasjidIDs []uuid.UUID
			if err := ac.DB.WithContext(c.Context()).
				Model(&MasjidTeacher{}).
				Where("masjid_teacher_user_teacher_id IN ?", userTeacherIDs).
				Where("masjid_teacher_deleted_at IS NULL").
				Pluck("masjid_teacher_masjid_id", &mtMasjidIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil masjid_teachers: "+err.Error())
			}
			for _, id := range mtMasjidIDs {
				if _, ok := masjidRoles[id]; !ok {
					masjidRoles[id] = map[string]struct{}{}
				}
				masjidRoles[id]["teacher"] = struct{}{}
			}
		}
	}

	// 3b) STUDENT
	{
		var profileIDs []uuid.UUID
		if err := ac.DB.WithContext(c.Context()).
			Model(&UserProfile{}).
			Where("user_profile_user_id = ?", userUUID).
			Where("user_profile_deleted_at IS NULL").
			Pluck("user_profile_id", &profileIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil user_profiles: "+err.Error())
		}
		if len(profileIDs) > 0 {
			var msMasjidIDs []uuid.UUID
			if err := ac.DB.WithContext(c.Context()).
				Model(&MasjidStudent{}).
				Where("masjid_student_user_profile_id IN ?", profileIDs).
				Where("masjid_student_deleted_at IS NULL").
				Pluck("masjid_student_masjid_id", &msMasjidIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil masjid_students: "+err.Error())
			}
			for _, id := range msMasjidIDs {
				if _, ok := masjidRoles[id]; !ok {
					masjidRoles[id] = map[string]struct{}{}
				}
				masjidRoles[id]["student"] = struct{}{}
			}
		}
	}

	// 3c) CLAIMS-ONLY: seed kandidat masjid & role dari JWT (tanpa butuh helperAuth tambahan)
	idsFromJWT, roleMapFromJWT := parseMasjidInfoFromJWT(c)

	// 4) Union kandidat masjid (relasi + klaim)
	candidate := map[uuid.UUID]struct{}{}
	for id := range masjidRoles {
		candidate[id] = struct{}{}
	}
	for _, id := range idsFromJWT {
		candidate[id] = struct{}{}
	}

	// Helper tambah role
	addIf := func(masjidID uuid.UUID, role string) {
		if _, ok := masjidRoles[masjidID]; !ok {
			masjidRoles[masjidID] = map[string]struct{}{}
		}
		masjidRoles[masjidID][role] = struct{}{}
	}

	// 4a) Tambah role yang sudah ada di klaim JWT (jika ada)
	for mid, set := range roleMapFromJWT {
		for r := range set {
			addIf(mid, r)
		}
	}

	// 4b) Tambahkan role via ACL helper (sinkron dengan middleware)
	for mid := range candidate {
		for _, r := range []string{"dkm", "admin", "bendahara"} {
			if helperAuth.HasRoleInMasjid(c, mid, r) {
				addIf(mid, r)
			}
		}
	}

	// 5) Ambil info ringkas masjid untuk semua kandidat
	masjidIDs := make([]uuid.UUID, 0, len(candidate))
	for id := range candidate {
		masjidIDs = append(masjidIDs, id)
	}

	resp := MyScopeResponse{
		UserID:      me.ID,
		UserName:    me.UserName,
		Memberships: []MasjidRoleOption{},
	}

	if len(masjidIDs) == 0 {
		// user belum punya relasi/claim apa pun
		return helper.JsonOK(c, "Context berhasil diambil", resp)
	}

	var masjids []masjidModel.MasjidModel
	if err := ac.DB.WithContext(c.Context()).
		Select("masjid_id, masjid_name, masjid_slug, masjid_icon_url").
		Where("masjid_id IN ?", masjidIDs).
		Where("masjid_deleted_at IS NULL").
		Where("masjid_is_active = ?", true).
		Find(&masjids).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil masjid: "+err.Error())
	}

	for _, m := range masjids {
		set := masjidRoles[m.MasjidID]
		if len(set) == 0 {
			continue
		}
		roles := make([]string, 0, len(set))
		for r := range set {
			roles = append(roles, r)
		}
		resp.Memberships = append(resp.Memberships, MasjidRoleOption{
			MasjidID:      m.MasjidID,
			MasjidName:    m.MasjidName,
			MasjidSlug:    m.MasjidSlug,
			MasjidIconURL: m.MasjidIconURL,
			Roles:         roles,
		})
	}

	// 6) (Opsional) terima seleksi dari query & set cookie
	if selMasjidStr := strings.TrimSpace(c.Query("select_masjid_id")); selMasjidStr != "" {
		if selMasjidID, e := uuid.Parse(selMasjidStr); e == nil {
			if selRole := strings.ToLower(strings.TrimSpace(c.Query("select_role"))); selRole != "" {
				// Validasi role ada di membership untuk masjid itu
				valid := false
				for _, m := range resp.Memberships {
					if m.MasjidID == selMasjidID {
						for _, r := range m.Roles {
							if r == selRole {
								valid = true
								break
							}
						}
						break
					}
				}
				if valid {
					c.Cookie(&fiber.Cookie{
						Name:     "active_masjid_id",
						Value:    selMasjidID.String(),
						Path:     "/",
						HTTPOnly: true,
						SameSite: "Lax",
						Expires:  time.Now().Add(12 * time.Hour),
					})
					c.Cookie(&fiber.Cookie{
						Name:     "active_role",
						Value:    selRole,
						Path:     "/",
						HTTPOnly: true,
						SameSite: "Lax",
						Expires:  time.Now().Add(12 * time.Hour),
					})
					resp.Selection = &ScopeSelection{
						MasjidID: &selMasjidID,
						Role:     &selRole,
					}
				} else {
					return helper.JsonError(c, fiber.StatusBadRequest, "Role/masjid tidak valid untuk user ini")
				}
			}
		}
	}

	return helper.JsonOK(c, "Context berhasil diambil", resp)
}