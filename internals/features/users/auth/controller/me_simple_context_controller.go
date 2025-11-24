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

	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"
	userModel "madinahsalam_backend/internals/features/users/users/model"

	helper "madinahsalam_backend/internals/helpers" // JsonOK/JsonError
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* =============== Link models (pastikan ada, atau definisikan ringan di sini) =============== */
// type UserTeacher struct {...}      func (UserTeacher) TableName() string { return "user_teachers" }
// type SchoolTeacher struct {...}    func (SchoolTeacher) TableName() string { return "school_teachers" }
// type UserProfile struct {...}      func (UserProfile) TableName() string { return "user_profiles" }
// type SchoolStudent struct {...}    func (SchoolStudent) TableName() string { return "school_students" }

/* =============== DTO Response (baru & ringkas) =============== */

// type: internals/features/users/auth/dto_context.go (misal)
type SchoolRoleOption struct {
	SchoolID      uuid.UUID `json:"school_id"`
	SchoolName    string    `json:"school_name"`
	SchoolSlug    *string   `json:"school_slug,omitempty"`
	SchoolIconURL *string   `json:"school_icon_url,omitempty"`

	Roles           []string   `json:"roles"`
	SchoolTeacherID *uuid.UUID `json:"school_teacher_id,omitempty"`
	SchoolStudentID *uuid.UUID `json:"school_student_id,omitempty"`
}

type ScopeSelection struct {
	SchoolID *uuid.UUID `json:"school_id,omitempty"`
	Role     *string    `json:"role,omitempty"`
}

type MyScopeResponse struct {
	UserID        uuid.UUID          `json:"user_id"`
	UserName      string             `json:"user_name"`
	UserAvatarURL *string            `json:"user_avatar_url,omitempty"`
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

// strptr mengubah string menjadi *string; kosong -> nil (biar omitempty jalan)
func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
// Di sini kita hanya manfaatkan school_id sebagai kandidat sekolah;
// mapping role dari JWT tidak lagi dipakai sebagai sumber kebenaran role.
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

	// kumpulkan school_ids unik
	seen := map[uuid.UUID]struct{}{}

	for _, s := range cl.SchoolIDs {
		if id, e := uuid.Parse(strings.TrimSpace(s)); e == nil && id != uuid.Nil {
			if _, ok := seen[id]; !ok {
				ids = append(ids, id)
				seen[id] = struct{}{}
			}
		}
	}

	for _, mr := range cl.SchoolRoles {
		if id, e := uuid.Parse(strings.TrimSpace(mr.SchoolID)); e == nil && id != uuid.Nil {
			if _, ok := seen[id]; !ok {
				ids = append(ids, id)
				seen[id] = struct{}{}
			}
			// roleMap[id] sengaja tidak diisi/diterapkan ke response:
			// kebenaran role sekarang didasarkan pada tabel user_roles + roles.
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

	// 2a) Ambil avatar URL terkini
	var avatarRecord struct {
		URL *string `gorm:"column:url"`
	}
	if err := ac.DB.WithContext(c.Context()).
		Model(&UserProfile{}).
		Select("user_profile_avatar_url AS url").
		Where("user_profile_user_id = ?", userUUID).
		Where("user_profile_deleted_at IS NULL").
		Order("COALESCE(user_profile_updated_at, user_profile_created_at) DESC").
		Limit(1).
		Scan(&avatarRecord).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil avatar: "+err.Error())
	}

	// ====== Per-school bucket: roles + optional ID teacher/student ======
	type roleBucket struct {
		roles     map[string]struct{}
		teacherID *uuid.UUID
		studentID *uuid.UUID
	}

	schoolRoles := map[uuid.UUID]*roleBucket{}

	getBucket := func(sid uuid.UUID) *roleBucket {
		if b, ok := schoolRoles[sid]; ok {
			return b
		}
		b := &roleBucket{
			roles: map[string]struct{}{},
		}
		schoolRoles[sid] = b
		return b
	}

	addRole := func(sid uuid.UUID, r string) {
		r = strings.ToLower(strings.TrimSpace(r))
		if r == "" {
			return
		}
		getBucket(sid).roles[r] = struct{}{}
	}

	// 3a) TEACHER — ambil (school_teacher_id, school_id) lalu SET ID saja (tidak menentukan role)
	{
		var mtRows []struct {
			ID       uuid.UUID `gorm:"column:school_teacher_id"`
			SchoolID uuid.UUID `gorm:"column:school_teacher_school_id"`
		}
		if err := ac.DB.WithContext(c.Context()).
			Model(&SchoolTeacher{}).
			Select("school_teacher_id, school_teacher_school_id").
			Joins("JOIN user_teachers ut ON ut.user_teacher_id = school_teachers.school_teacher_user_teacher_id").
			Where("ut.user_teacher_user_id = ?", userUUID).
			Where("school_teacher_deleted_at IS NULL").
			Scan(&mtRows).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil school_teachers: "+err.Error())
		}

		for _, row := range mtRows {
			b := getBucket(row.SchoolID)
			id := row.ID
			b.teacherID = &id
		}
	}

	// 3b) STUDENT — ambil (school_student_id, school_id) via user_profiles aktif (SET ID saja)
	{
		var msRows []struct {
			ID       uuid.UUID `gorm:"column:school_student_id"`
			SchoolID uuid.UUID `gorm:"column:school_student_school_id"`
		}

		// Ambil semua profile aktif user ini
		var profileIDs []uuid.UUID
		if err := ac.DB.WithContext(c.Context()).
			Model(&UserProfile{}).
			Where("user_profile_user_id = ?", userUUID).
			Where("user_profile_deleted_at IS NULL").
			Pluck("user_profile_id", &profileIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil user_profiles: "+err.Error())
		}

		if len(profileIDs) > 0 {
			if err := ac.DB.WithContext(c.Context()).
				Model(&SchoolStudent{}).
				Select("school_student_id, school_student_school_id").
				Where("school_student_user_profile_id IN ?", profileIDs).
				Where("school_student_deleted_at IS NULL").
				Scan(&msRows).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil school_students: "+err.Error())
			}

			for _, row := range msRows {
				b := getBucket(row.SchoolID)
				id := row.ID
				b.studentID = &id
			}
		}
	}

	// 3c) ROLES RESMI — ambil dari user_roles + roles (canonical source of truth)
	{
		type userRoleRow struct {
			SchoolID *uuid.UUID `gorm:"column:school_id"`
			RoleName string     `gorm:"column:role_name"`
		}

		var urRows []userRoleRow
		if err := ac.DB.WithContext(c.Context()).
			Table("user_roles").
			Select("user_roles.school_id, roles.role_name").
			Joins("JOIN roles ON roles.role_id = user_roles.role_id").
			Where("user_roles.user_id = ?", userUUID).
			Where("user_roles.deleted_at IS NULL").
			Scan(&urRows).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil roles user: "+err.Error())
		}

		for _, row := range urRows {
			// Global roles (school_id NULL) bisa di-handle terpisah nanti kalau mau;
			// untuk scope per-sekolah, kita hanya pakai yang punya school_id.
			if row.SchoolID == nil || *row.SchoolID == uuid.Nil {
				continue
			}
			sid := *row.SchoolID
			addRole(sid, row.RoleName)
		}
	}

	// 3d) CLAIMS-ONLY dari JWT (untuk kandidat school_id, bukan sumber role)
	idsFromJWT, _ := parseSchoolInfoFromJWT(c)

	// 3e) Resolusi slug (juga dipakai sebagai FILTER memberships)
	var filterSchoolID *uuid.UUID

	slug := strings.TrimSpace(c.Params("masjid_slug"))
	if slug == "" {
		slug = strings.TrimSpace(c.Params("school_slug"))
	}
	if slug == "" {
		slug = strings.TrimSpace(c.Params("slug"))
	}

	if slug != "" {
		var row struct {
			ID uuid.UUID `gorm:"column:school_id"`
		}
		if err := ac.DB.WithContext(c.Context()).
			Model(&schoolModel.SchoolModel{}).
			Select("school_id").
			Where("school_slug = ?", slug).
			Where("school_deleted_at IS NULL").
			Where("school_is_active = ?", true).
			First(&row).Error; err == nil && row.ID != uuid.Nil {

			// simpan untuk filter akhir
			filterSchoolID = &row.ID

			// pastikan sekolah ini ikut kandidat meskipun tidak ada di roles/JWT
			idsFromJWT = append(idsFromJWT, row.ID)
		}
	}

	// 4) Kumpulan kandidat school_id
	candidate := map[uuid.UUID]struct{}{}
	for id := range schoolRoles {
		candidate[id] = struct{}{}
	}
	for _, id := range idsFromJWT {
		if id != uuid.Nil {
			candidate[id] = struct{}{}
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
		UserAvatarURL: avatarRecord.URL,
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

	for _, s := range schools {
		b := schoolRoles[s.SchoolID]

		// kumpulkan roles (bisa kosong)
		roles := []string{}
		var teacherID *uuid.UUID
		var studentID *uuid.UUID

		if b != nil {
			for r := range b.roles {
				roles = append(roles, r)
			}
			if b.teacherID != nil {
				teacherID = b.teacherID
			}
			if b.studentID != nil {
				studentID = b.studentID
			}
		}

		opt := SchoolRoleOption{
			SchoolID:        s.SchoolID,
			SchoolName:      s.SchoolName,
			SchoolSlug:      strptr(s.SchoolSlug),
			SchoolIconURL:   s.SchoolIconURL,
			Roles:           roles,
			SchoolTeacherID: teacherID,
			SchoolStudentID: studentID,
		}

		resp.Memberships = append(resp.Memberships, opt)
	}

	// === FILTER membership berdasarkan slug (kalau ada) ===
	if filterSchoolID != nil {
		filtered := make([]SchoolRoleOption, 0, len(resp.Memberships))
		for _, m := range resp.Memberships {
			if m.SchoolID == *filterSchoolID {
				filtered = append(filtered, m)
			}
		}
		resp.Memberships = filtered
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