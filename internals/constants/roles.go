package constants

import "fmt"

// Template pesan error role
const (
	ErrOnlyTeachersCanAccess = "❌ Hanya teacher, admin, atau owner yang boleh mengakses fitur %s."
	ErrOnlyAdminsCanAccess   = "❌ Hanya admin yang boleh mengakses fitur %s."
	ErrOnlyNonUserCanAccess  = "❌ Hanya role selain 'user' yang boleh mengakses fitur %s."
	ErrOnlyOwnersCanAccess   = "❌ Hanya owner yang boleh mengakses fitur %s."
)

// Fungsi helper untuk menghasilkan pesan error dinamis
func RoleErrorTeacher(feature string) string {
	return fmt.Sprintf(ErrOnlyTeachersCanAccess, feature)
}

func RoleErrorAdmin(feature string) string {
	return fmt.Sprintf(ErrOnlyAdminsCanAccess, feature)
}

func RoleErrorNonUser(feature string) string {
	return fmt.Sprintf(ErrOnlyNonUserCanAccess, feature)
}

func RoleErrorOwner(feature string) string {
	return fmt.Sprintf(ErrOnlyOwnersCanAccess, feature)
}

// ==========================
// ✅ Grouped Role Slices
// ==========================
var (
	AllRoles = []string{
		RoleUser,
		RoleAdmin,
		RoleTeacher,
		RoleOwner,
		RoleAccountant,
	}

	NonUserRoles = []string{
		RoleAdmin,
		RoleTeacher,
		RoleOwner,
		RoleAccountant,
	}

	TeacherAndAbove = []string{
		RoleTeacher,
		RoleAdmin,
		RoleOwner,
	}

	OwnerAndAbove = []string{
		RoleOwner,
		RoleAdmin,
	}

	AdminOnly = []string{
		RoleAdmin,
	}

	OwnerOnly = []string{
		RoleOwner,
	}
)
