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

	schoolModel "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
	userModel "schoolku_backend/internals/features/users/users/model"

	helper "schoolku_backend/internals/helpers" // JsonOK/JsonError
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =============== Link models (pastikan ada, atau definisikan ringan di sini) =============== */
// type UserTeacher struct {...}      func (UserTeacher) TableName() string { return "user_teachers" }
// type SchoolTeacher struct {...}    func (SchoolTeacher) TableName() string { return "school_teachers" }
// type UserProfile struct {...}      func (UserProfile) TableName() string { return "user_profiles" }
// type SchoolStudent struct {...}    func (SchoolStudent) TableName() string { return "school_students" }

/* =============== DTO Response (baru & ringkas) =============== */

type SchoolRoleOption struct {
	SchoolID      uuid.UUID `json:"school_id"`
	SchoolName    string    `json:"school_name"`
	SchoolSlug    string    `json:"school_slug"`
	SchoolIconURL *string   `json:"school_icon_url,omitempty"`
	Roles         []string  `json:"roles"`
}

type ScopeSelection struct {
	SchoolID *uuid.UUID `json:"school_id,omitempty"`
	Role     *string    `json:"role,omitempty"`
}

// ====== Tambah/ubah tipe respons (pastikan didefinisikan di file yg sama) ======
type MyScopeResponse struct {
	UserID        uuid.UUID          `json:"user_id"`
	UserName      string             `json:"user_name"`
	UserAvatarURL *string            `json:"user_avatar_url,omitempty"` // ⬅️ baru
	Memberships   []SchoolRoleOption `json:"memberships"`
	Selection     *ScopeSelection    `json:"selection,omitempty"`
}

/* =============== Helper lokal: decode klaim JWT (tanpa verifikasi) =============== */

type jwtSchoolRole struct {
	SchoolID string   `json:"school_id"`
	Roles    []string `json:"roles"`
}
type jwtClaimsLite struct {
	SchoolIDs      []string        `json:"school_ids"`
	SchoolRoles    []jwtSchoolRole `json:"school_roles"`
	ActiveSchoolID string          `json:"active_school_id"`
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
func parseSchoolInfoFromJWT(c *fiber.Ctx) (ids []uuid.UUID, roleMap map[uuid.UUID]map[string]struct{}) {
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

	// kumpulkan school_ids
	seen := map[uuid.UUID]struct{}{}
	for _, s := range cl.SchoolIDs {
		if id, e := uuid.Parse(strings.TrimSpace(s)); e == nil && id != uuid.Nil {
			if _, ok := seen[id]; !ok {
				ids = append(ids, id)
				seen[id] = struct{}{}
			}
		}
	}
	// dari school_roles[].school_id
	for _, mr := range cl.SchoolRoles {
		if id, e := uuid.Parse(strings.TrimSpace(mr.SchoolID)); e == nil && id != uuid.Nil {
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
	// active_school_id (opsional)
	if cl.ActiveSchoolID != "" {
		if id, e := uuid.Parse(strings.TrimSpace(cl.ActiveSchoolID)); e == nil && id != uuid.Nil {
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

	// 2a) ⬅️ Ambil avatar URL terkini dari user_profiles (yang tidak terhapus)
	var avatarRecord struct {
		URL *string `gorm:"column:url"`
	}
	if err := ac.DB.WithContext(c.Context()).
		Model(&UserProfile{}).
		Select("user_profile_avatar_url AS url").
		Where("user_profile_user_id = ?", userUUID).
		Where("user_profile_deleted_at IS NULL").
		// urutkan by updated_at (fallback ke created_at), ambil yang paling baru
		Order("COALESCE(user_profile_updated_at, user_profile_created_at) DESC").
		Limit(1).
		Scan(&avatarRecord).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil avatar: "+err.Error())
	}

	// 3) Kumpulkan school_id via TEACHER & STUDENT
	schoolRoles := map[uuid.UUID]map[string]struct{}{}

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
			var mtSchoolIDs []uuid.UUID
			if err := ac.DB.WithContext(c.Context()).
				Model(&SchoolTeacher{}).
				Where("school_teacher_user_teacher_id IN ?", userTeacherIDs).
				Where("school_teacher_deleted_at IS NULL").
				Pluck("school_teacher_school_id", &mtSchoolIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil school_teachers: "+err.Error())
			}
			for _, id := range mtSchoolIDs {
				if _, ok := schoolRoles[id]; !ok {
					schoolRoles[id] = map[string]struct{}{}
				}
				schoolRoles[id]["teacher"] = struct{}{}
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
			var msSchoolIDs []uuid.UUID
			if err := ac.DB.WithContext(c.Context()).
				Model(&SchoolStudent{}).
				Where("school_student_user_profile_id IN ?", profileIDs).
				Where("school_student_deleted_at IS NULL").
				Pluck("school_student_school_id", &msSchoolIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil school_students: "+err.Error())
			}
			for _, id := range msSchoolIDs {
				if _, ok := schoolRoles[id]; !ok {
					schoolRoles[id] = map[string]struct{}{}
				}
				schoolRoles[id]["student"] = struct{}{}
			}
		}
	}

	// 3c) CLAIMS-ONLY dari JWT
	idsFromJWT, roleMapFromJWT := parseSchoolInfoFromJWT(c)

	// 4) Union kandidat school
	candidate := map[uuid.UUID]struct{}{}
	for id := range schoolRoles {
		candidate[id] = struct{}{}
	}
	for _, id := range idsFromJWT {
		candidate[id] = struct{}{}
	}

	addIf := func(schoolID uuid.UUID, role string) {
		if _, ok := schoolRoles[schoolID]; !ok {
			schoolRoles[schoolID] = map[string]struct{}{}
		}
		schoolRoles[schoolID][role] = struct{}{}
	}

	for mid, set := range roleMapFromJWT {
		for r := range set {
			addIf(mid, r)
		}
	}
	for mid := range candidate {
		for _, r := range []string{"dkm", "admin", "bendahara"} {
			if helperAuth.HasRoleInSchool(c, mid, r) {
				addIf(mid, r)
			}
		}
	}

	// 5) Ambil info ringkas school
	schoolIDs := make([]uuid.UUID, 0, len(candidate))
	for id := range candidate {
		schoolIDs = append(schoolIDs, id)
	}

	resp := MyScopeResponse{
		UserID:        me.ID,
		UserName:      me.UserName,
		UserAvatarURL: avatarRecord.URL, // ⬅️ isi dari query avatar
		Memberships:   []SchoolRoleOption{},
	}

	if len(schoolIDs) == 0 {
		return helper.JsonOK(c, "Context berhasil diambil", resp)
	}

	var schools []schoolModel.SchoolModel
	if err := ac.DB.WithContext(c.Context()).
		Select("school_id, school_name, school_slug, school_icon_url").
		Where("school_id IN ?", schoolIDs).
		Where("school_deleted_at IS NULL").
		Where("school_is_active = ?", true).
		Find(&schools).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil school: "+err.Error())
	}

	for _, m := range schools {
		set := schoolRoles[m.SchoolID]
		if len(set) == 0 {
			continue
		}
		roles := make([]string, 0, len(set))
		for r := range set {
			roles = append(roles, r)
		}
		resp.Memberships = append(resp.Memberships, SchoolRoleOption{
			SchoolID:      m.SchoolID,
			SchoolName:    m.SchoolName,
			SchoolSlug:    m.SchoolSlug,
			SchoolIconURL: m.SchoolIconURL,
			Roles:         roles,
		})
	}

	// 6) (Opsional) handle seleksi dan set cookie
	if selSchoolStr := strings.TrimSpace(c.Query("select_school_id")); selSchoolStr != "" {
		if selSchoolID, e := uuid.Parse(selSchoolStr); e == nil {
			if selRole := strings.ToLower(strings.TrimSpace(c.Query("select_role"))); selRole != "" {
				valid := false
				for _, m := range resp.Memberships {
					if m.SchoolID == selSchoolID {
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
						Name:     "active_school_id",
						Value:    selSchoolID.String(),
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
						SchoolID: &selSchoolID,
						Role:     &selRole,
					}
				} else {
					return helper.JsonError(c, fiber.StatusBadRequest, "Role/school tidak valid untuk user ini")
				}
			}
		}
	}

	return helper.JsonOK(c, "Context berhasil diambil", resp)
}
