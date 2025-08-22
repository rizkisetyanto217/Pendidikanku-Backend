// file: internals/constants/roles.go
package constants

import (
	"fmt"
	"strings"
)

/*
Single source of truth (selaras dengan CHECK di DB):
CHECK (role IN ('owner','user','teacher','treasurer','admin','dkm','author','student'))
*/

// ==========================
// ✅ Role Names
// ==========================
const (
	RoleOwner     = "owner"
	RoleUser      = "user"
	RoleTeacher   = "teacher"
	RoleTreasurer = "treasurer" // pengganti "accountant"
	RoleAdmin     = "admin"     // admin platform/tenant
	RoleDKM       = "dkm"       // admin DKM
	RoleAuthor    = "author"
	RoleStudent   = "student"

	// Deprecated: gunakan RoleTreasurer
	RoleAccountantDeprecated = "accountant"
)

// ==========================
// ✅ Allowed & Grouped Roles (PUBLIC VARS — kompatibel lama)
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

// Semua selain user & student
var NonUserRoles = []string{
	RoleAdmin, RoleDKM, RoleTeacher, RoleTreasurer, RoleAuthor, RoleOwner,
}

// Staf operasional (semua role non-user/non-student)
var StaffAndAbove = []string{
	RoleAdmin, RoleDKM, RoleTeacher, RoleTreasurer, RoleAuthor, RoleOwner,
}

// Guru ke atas (guru, admin, dkm, owner)
var TeacherAndAbove = []string{
	RoleTeacher, RoleAdmin, RoleDKM, RoleOwner,
}

// Admin ke atas (admin, dkm, owner)
var AdminAndAbove = []string{
	RoleAdmin, RoleDKM, RoleOwner,
}

// Keuangan (treasurer) & atasan
var FinanceAndAbove = []string{
	RoleTreasurer, RoleAdmin, RoleDKM, RoleOwner,
}

// Pembuat konten
var ContentCreators = []string{
	RoleAuthor, RoleTeacher, RoleAdmin, RoleDKM, RoleOwner,
}

var (
	OwnerOnly = []string{RoleOwner}
	AdminOnly = []string{RoleAdmin}
)

// ==========================
/* ✅ Error Messages & Helpers */
// ==========================
const (
	ErrOnlyTeachersCanAccess = "❌ Hanya teacher, admin/dkm, atau owner yang boleh mengakses fitur %s."
	ErrOnlyAdminsCanAccess   = "❌ Hanya admin/dkm atau owner yang boleh mengakses fitur %s."
	ErrOnlyNonUserCanAccess  = "❌ Hanya role selain 'user' yang boleh mengakses fitur %s."
	ErrOnlyOwnersCanAccess   = "❌ Hanya owner yang boleh mengakses fitur %s."
)

func RoleErrorTeacher(feature string) string { return fmt.Sprintf(ErrOnlyTeachersCanAccess, feature) }
func RoleErrorAdmin(feature string) string   { return fmt.Sprintf(ErrOnlyAdminsCanAccess, feature) }
func RoleErrorNonUser(feature string) string { return fmt.Sprintf(ErrOnlyNonUserCanAccess, feature) }
func RoleErrorOwner(feature string) string   { return fmt.Sprintf(ErrOnlyOwnersCanAccess, feature) }

// ==========================
// ✅ Utilities
// ==========================

func NormalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

// Validate role against AllowedRoles (case-insensitive)
func ValidateRole(role string) bool {
	role = NormalizeRole(role)
	for _, r := range AllowedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// ContainsRole checks membership in a slice group (ke belakang)
func ContainsRole(roles []string, role string) bool {
	role = NormalizeRole(role)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// ==========================
// (Optional) Fast checks (helper O(1) via switch)
// ==========================

func InOwnerOnly(role string) bool       { return ContainsRole(OwnerOnly, NormalizeRole(role)) }
func InAdminOnly(role string) bool       { return ContainsRole(AdminOnly, NormalizeRole(role)) }
func InAdminAndAbove(role string) bool   { return ContainsRole(AdminAndAbove, NormalizeRole(role)) }
func InTeacherAndAbove(role string) bool { return ContainsRole(TeacherAndAbove, NormalizeRole(role)) }
func InNonUserRoles(role string) bool    { return ContainsRole(NonUserRoles, NormalizeRole(role)) }
func InStaffAndAbove(role string) bool   { return ContainsRole(StaffAndAbove, NormalizeRole(role)) }
func InFinanceAndAbove(role string) bool { return ContainsRole(FinanceAndAbove, NormalizeRole(role)) }
func InContentCreators(role string) bool { return ContainsRole(ContentCreators, NormalizeRole(role)) }
