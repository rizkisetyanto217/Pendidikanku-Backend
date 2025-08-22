// file: internals/constants/roles.go
package constants

import "fmt"

// ==========================
// ✅ Single-source of truth (selaras dengan CHECK di DB)
//   CHECK (role IN ('owner','user','teacher','treasurer','admin','dkm','author','student'))
// ==========================
const (
	RoleOwner     = "owner"
	RoleUser      = "user"
	RoleTeacher   = "teacher"
	RoleTreasurer = "treasurer" // pengganti "accountant"
	RoleAdmin     = "admin"     // admin platform/tenant
	RoleDKM       = "dkm"       // admin DKM (jika dipakai terpisah dari "admin")
	RoleAuthor    = "author"
	RoleStudent   = "student"
)

// (Opsional) Deprecation alias untuk backward compatibility.
// Disarankan ganti semua penggunaan "accountant" → "treasurer".
const (
	// Deprecated: gunakan RoleTreasurer
	RoleAccountantDeprecated = "accountant"
)

// ==========================
// ✅ Allowed Roles (sinkron dengan DB CHECK)
// ==========================
var AllowedRoles = []string{
	RoleOwner,
	RoleUser,
	RoleTeacher,
	RoleTreasurer,
	RoleAdmin,
	RoleDKM,
	RoleAuthor,
	RoleStudent,
}

// ==========================
// ✅ Grouped Role Slices (untuk guard middleware)
// ==========================
var (
	// Semua selain user & student
	NonUserRoles = []string{
		RoleAdmin, RoleDKM, RoleTeacher, RoleTreasurer, RoleAuthor, RoleOwner,
	}

	// Staf operasional (semua role non-user/non-student)
	StaffAndAbove = []string{
		RoleAdmin, RoleDKM, RoleTeacher, RoleTreasurer, RoleAuthor, RoleOwner,
	}

	// Guru ke atas (guru, admin, dkm, owner)
	TeacherAndAbove = []string{
		RoleTeacher, RoleAdmin, RoleDKM, RoleOwner,
	}

	// Admin ke atas (admin, dkm, owner)
	AdminAndAbove = []string{
		RoleAdmin, RoleDKM, RoleOwner,
	}

	// Keuangan (treasurer/bendahara) & atasan
	FinanceAndAbove = []string{
		RoleTreasurer, RoleAdmin, RoleDKM, RoleOwner,
	}

	// Pembuat konten (author/teacher/admin/dkm/owner)
	ContentCreators = []string{
		RoleAuthor, RoleTeacher, RoleAdmin, RoleDKM, RoleOwner,
	}

	OwnerOnly = []string{RoleOwner}
	AdminOnly = []string{RoleAdmin} // kalau ingin admin tanpa DKM, pakai ini
)

// ==========================
// ✅ Error Messages
// ==========================
const (
	ErrOnlyTeachersCanAccess = "❌ Hanya teacher, admin/dkm, atau owner yang boleh mengakses fitur %s."
	ErrOnlyAdminsCanAccess   = "❌ Hanya admin/dkm atau owner yang boleh mengakses fitur %s."
	ErrOnlyNonUserCanAccess  = "❌ Hanya role selain 'user' yang boleh mengakses fitur %s."
	ErrOnlyOwnersCanAccess   = "❌ Hanya owner yang boleh mengakses fitur %s."
)

// Fungsi helper untuk pesan error dinamis
func RoleErrorTeacher(feature string) string  { return fmt.Sprintf(ErrOnlyTeachersCanAccess, feature) }
func RoleErrorAdmin(feature string) string    { return fmt.Sprintf(ErrOnlyAdminsCanAccess, feature) }
func RoleErrorNonUser(feature string) string  { return fmt.Sprintf(ErrOnlyNonUserCanAccess, feature) }
func RoleErrorOwner(feature string) string    { return fmt.Sprintf(ErrOnlyOwnersCanAccess, feature) }

// ==========================
// ✅ Utility
// ==========================
func ContainsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
